// Package cli implements the command interface and process exit-code policy.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/emguse/imgstamp/internal/batch"
	"github.com/emguse/imgstamp/internal/config"
	"github.com/emguse/imgstamp/internal/fonts"
	"github.com/emguse/imgstamp/internal/stamp"
)

// Run executes the CLI and returns the documented exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 1 && args[0] == "--version" {
		fmt.Fprintln(stdout, "imgstamp", version)
		return 0
	}
	if len(args) > 0 && args[0] == "fonts" {
		if len(args) != 1 {
			fmt.Fprintln(stderr, "fonts accepts no arguments")
			return 2
		}
		entries, err := fonts.Scan(ctx, fonts.SystemDirs())
		if err != nil {
			fmt.Fprintln(stderr, err)
			return errorCode(ctx)
		}
		if len(entries) == 0 {
			fmt.Fprintln(stderr, "no supported system fonts found")
			return 2
		}
		for _, entry := range entries {
			fmt.Fprintln(stdout, entry.Name)
		}
		return 0
	}
	flags := flag.NewFlagSet("imgstamp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "TOML configuration file")
	input := flags.String("input", "", "input directory (direct children only)")
	output := flags.String("output", "", "separate output directory")
	textOverride := flags.String("text", "", "override stamp text (same character count as configured text)")
	shrink := flags.Bool("shrink", false, "reduce A1/A2 to A3 and A3 to A4")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: imgstamp --config FILE --input DIR --output DIR [--shrink] [--text VALUE]\n       imgstamp fonts\n       imgstamp --version")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *configPath == "" || *input == "" || *output == "" {
		flags.Usage()
		return 2
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	textOverrideSet := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "text" {
			textOverrideSet = true
		}
	})
	if textOverrideSet {
		if err := overrideText(&cfg, *textOverride); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}
	overlay, err := stamp.Prepare(ctx, cfg.Stamp)
	if err != nil {
		fmt.Fprintln(stderr, "prepare stamp:", err)
		return errorCode(ctx)
	}
	result, err := batch.Run(ctx, batch.Options{Input: *input, Output: *output, Config: cfg, Stamp: overlay, Shrink: *shrink, Report: func(e batch.Event) {
		if e.Err != nil {
			fmt.Fprintf(stderr, "failed %q: %v\n", e.File, e.Err)
		} else if e.Page > 0 {
			fmt.Fprintf(stdout, "processed %q page=%d paper=%s (pending PDF completion)\n", e.File, e.Page, e.Paper)
		} else {
			fmt.Fprintf(stdout, "%s %q\n", e.Status, e.File)
		}
	}})
	fmt.Fprintf(stdout, "Files: success=%d skipped=%d failed=%d\n", result.Success, result.Skipped, result.Failed)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return errorCode(ctx)
	}
	if result.Failed > 0 {
		return 1
	}
	if result.Success+result.Skipped == 0 {
		fmt.Fprintln(stderr, "warning: no supported input images")
	}
	return 0
}
func errorCode(ctx context.Context) int {
	if ctx.Err() != nil {
		return 130
	}
	return 2
}

func overrideText(cfg *config.Config, value string) error {
	if cfg.Stamp.Text == nil {
		return fmt.Errorf("--text requires stamp.text in the configuration")
	}
	if utf8.RuneCountInString(value) != utf8.RuneCountInString(cfg.Stamp.Text.Value) {
		return fmt.Errorf("--text must have the same character count as stamp.text.value")
	}
	updated := *cfg
	text := *cfg.Stamp.Text
	text.Value = value
	updated.Stamp.Text = &text
	if err := updated.Validate(); err != nil {
		return fmt.Errorf("--text: %w", err)
	}
	*cfg = updated
	return nil
}
