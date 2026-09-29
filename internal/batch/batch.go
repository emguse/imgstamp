// Package batch coordinates file-level atomic image-to-PDF conversion.
package batch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/emguse/imgstamp/internal/config"
	"github.com/emguse/imgstamp/internal/pages"
	"github.com/emguse/imgstamp/internal/pdfout"
	"github.com/emguse/imgstamp/internal/stamp"
)

// Result counts input files, not pages.
type Result struct{ Success, Skipped, Failed int }

// Event reports page progress or the final outcome of an input file.
type Event struct {
	File          string
	Page          int
	Status, Paper string
	Err           error
}

// Options supplies validated configuration, a prepared overlay, and paths.
type Options struct {
	Input, Output string
	Config        config.Config
	Stamp         *stamp.Prepared
	Shrink        bool
	Report        func(Event)
}

func canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(abs)
	if parent == abs {
		return "", err
	}
	base, err := canonical(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, filepath.Base(abs)), nil
}
func contains(parent, child string) bool {
	if runtime.GOOS == "windows" {
		parent = strings.ToLower(parent)
		child = strings.ToLower(child)
	}
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ancestorMatches also detects aliases on case-insensitive filesystems, where
// the canonical spelling returned by EvalSymlinks can retain the caller's case.
func ancestorMatches(target os.FileInfo, path string) (bool, error) {
	for {
		info, err := os.Stat(path)
		if err == nil && os.SameFile(target, info) {
			return true, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return false, err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false, nil
		}
		path = parent
	}
}
func (o Options) report(e Event) {
	if o.Report != nil {
		o.Report(e)
	}
}

// Run processes direct child images and publishes only complete PDFs.
func Run(ctx context.Context, o Options) (Result, error) {
	var result Result
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if o.Stamp == nil {
		return result, fmt.Errorf("stamp is not prepared")
	}
	input, err := canonical(o.Input)
	if err != nil {
		return result, err
	}
	output, err := canonical(o.Output)
	if err != nil {
		return result, err
	}
	if contains(input, output) || contains(output, input) {
		return result, fmt.Errorf("input and output directories must be separate and must not contain each other")
	}
	info, err := os.Stat(input)
	if err != nil {
		return result, err
	}
	if !info.IsDir() {
		return result, fmt.Errorf("input is not a directory")
	}
	if overlap, err := ancestorMatches(info, output); err != nil {
		return result, err
	} else if overlap {
		return result, fmt.Errorf("output aliases input or a directory inside input")
	}
	if outputInfo, err := os.Stat(output); err == nil {
		if overlap, err := ancestorMatches(outputInfo, input); err != nil {
			return result, err
		} else if overlap {
			return result, fmt.Errorf("output aliases a directory containing input")
		}
	} else if !os.IsNotExist(err) {
		return result, err
	}
	entries, err := os.ReadDir(input)
	if err != nil {
		return result, err
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return result, err
	}
	// Detect unwritable destinations before reporting individual file failures.
	probe, err := os.CreateTemp(output, ".imgstamp-check-*")
	if err != nil {
		return result, err
	}
	probeName := probe.Name()
	closeErr := probe.Close()
	removeErr := os.Remove(probeName)
	if closeErr != nil {
		return result, closeErr
	}
	if removeErr != nil {
		return result, removeErr
	}
	var stampInfo os.FileInfo
	if o.Config.Stamp.Image != "" {
		stampInfo, err = os.Stat(o.Config.Stamp.Image)
		if err != nil {
			return result, err
		}
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		path := filepath.Join(input, entry.Name())
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !pages.Supported(path) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			result.Failed++
			o.report(Event{File: entry.Name(), Status: "failed", Err: err})
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if stampInfo != nil && os.SameFile(info, stampInfo) {
			continue
		}
		target := filepath.Join(output, entry.Name()+".pdf")
		if _, err := os.Lstat(target); err == nil {
			result.Skipped++
			o.report(Event{File: entry.Name(), Status: "skipped"})
			continue
		} else if !os.IsNotExist(err) {
			result.Failed++
			o.report(Event{File: entry.Name(), Status: "failed", Err: err})
			continue
		}
		err = convert(ctx, path, target, o)
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		if errors.Is(err, os.ErrExist) {
			result.Skipped++
			o.report(Event{File: entry.Name(), Status: "skipped"})
			continue
		}
		if err != nil {
			result.Failed++
			o.report(Event{File: entry.Name(), Status: "failed", Err: err})
			continue
		}
		result.Success++
		o.report(Event{File: entry.Name(), Status: "success"})
	}
	return result, nil
}

func convert(ctx context.Context, path, target string, o Options) error {
	source, err := pages.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()
	temp, err := os.CreateTemp(filepath.Dir(target), ".imgstamp-*.pdf")
	if err != nil {
		return err
	}
	defer func() { _ = temp.Close(); _ = os.Remove(temp.Name()) }()
	pdf := pdfout.New(temp)
	count := 0
	for {
		img, err := source.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("page %d: %w", count+1, err)
		}
		count++
		size, err := o.Config.Paper.Match(img.Bounds().Dx(), img.Bounds().Dy())
		if err != nil {
			return fmt.Errorf("page %d: %w", count, err)
		}
		scale := 1.0
		if o.Shrink {
			if reduced, ok := size.Reduced(); ok {
				scale = math.Min(reduced.WidthMM/size.WidthMM, reduced.HeightMM/size.HeightMM)
				size = reduced
			}
		}
		composite, err := o.Stamp.ApplyScaled(ctx, img, size.Name, scale)
		if err != nil {
			return fmt.Errorf("page %d: %w", count, err)
		}
		if err = pdf.Add(ctx, composite, size, o.Config.Output.ColorMode == "grayscale"); err != nil {
			return fmt.Errorf("page %d: %w", count, err)
		}
		o.report(Event{File: filepath.Base(path), Page: count, Status: "processed", Paper: size.Name})
	}
	if err := pdf.Close(); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// A same-directory hard link publishes only the complete file and atomically
	// refuses to replace an existing path. Unsupported filesystems fail explicitly.
	if err := os.Link(temp.Name(), target); err != nil {
		return fmt.Errorf("publish PDF without overwrite: %w", err)
	}
	return nil
}
