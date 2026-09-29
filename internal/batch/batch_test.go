package batch

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emguse/imgstamp/internal/config"
	"github.com/emguse/imgstamp/internal/paper"
	"github.com/emguse/imgstamp/internal/stamp"
	"github.com/hhrutter/tiff"
)

func setup(t *testing.T) Options {
	t.Helper()
	root := t.TempDir()
	input := filepath.Join(root, "input")
	os.Mkdir(input, 0755)
	seal := image.NewRGBA(image.Rect(0, 0, 2, 2))
	seal.Set(0, 0, color.Black)
	sealPath := filepath.Join(root, "seal.png")
	f, err := os.Create(sealPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, seal); err != nil {
		t.Fatal(err)
	}
	f.Close()
	cfg := config.Config{Output: config.Output{ColorMode: "grayscale"}, Paper: paper.Config{Rules: []paper.Rule{{Name: "test", Short: 100, Long: 200, Paper: "A4"}}}, Stamp: config.Stamp{Image: sealPath, Position: "top-left", ImageOpacity: 1}}
	overlay, err := stamp.Prepare(context.Background(), cfg.Stamp)
	if err != nil {
		t.Fatal(err)
	}
	return Options{Input: input, Output: filepath.Join(root, "output"), Config: cfg, Stamp: overlay}
}
func writePNG(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = png.Encode(f, image.NewGray(image.Rect(0, 0, 100, 200))); err != nil {
		t.Fatal(err)
	}
}
func checkNoTemps(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".imgstamp-") {
			t.Fatalf("left temp %s", e.Name())
		}
	}
}
func TestSuccessAndSkip(t *testing.T) {
	o := setup(t)
	writePNG(t, filepath.Join(o.Input, "図面 1.png"))
	os.WriteFile(filepath.Join(o.Input, "notes.txt"), []byte("skip"), 0600)
	result, err := Run(context.Background(), o)
	if err != nil || result != (Result{Success: 1}) {
		t.Fatalf("%+v %v", result, err)
	}
	path := filepath.Join(o.Output, "図面 1.png.pdf")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(before), "%PDF-") {
		t.Fatal("not a PDF")
	}
	result, err = Run(context.Background(), o)
	if err != nil || result != (Result{Skipped: 1}) {
		t.Fatalf("%+v %v", result, err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("overwrote PDF")
	}
	checkNoTemps(t, o.Output)
}

func TestShrinkModeReducesRasterAndSetsTargetPaper(t *testing.T) {
	o := setup(t)
	o.Config.Paper.Rules[0].Paper = "A1"
	o.Shrink = true
	writePNG(t, filepath.Join(o.Input, "drawing.png"))
	var reportedPaper string
	o.Report = func(e Event) {
		if e.Status == "processed" {
			reportedPaper = e.Paper
		}
	}
	result, err := Run(context.Background(), o)
	if err != nil || result != (Result{Success: 1}) {
		t.Fatalf("%+v %v", result, err)
	}
	data, err := os.ReadFile(filepath.Join(o.Output, "drawing.png.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	pdf := string(data)
	for _, want := range []string{"/Width 50 /Height 100", "/MediaBox [0 0 841.889764 1190.551181]"} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("shrunk PDF missing %q", want)
		}
	}
	if reportedPaper != "A3" {
		t.Fatalf("reported paper = %q, want A3", reportedPaper)
	}
}

func TestFailedSecondPageLeavesNoPDFAndContinues(t *testing.T) {
	o := setup(t)
	f, err := os.Create(filepath.Join(o.Input, "a.tif"))
	if err != nil {
		t.Fatal(err)
	}
	if err = tiff.EncodeAll(f, []image.Image{image.NewGray(image.Rect(0, 0, 100, 200)), image.NewGray(image.Rect(0, 0, 101, 200))}, nil); err != nil {
		t.Fatal(err)
	}
	f.Close()
	writePNG(t, filepath.Join(o.Input, "b.png"))
	var reported error
	o.Report = func(e Event) {
		if e.Err != nil {
			reported = e.Err
		}
	}
	result, err := Run(context.Background(), o)
	if err != nil || result != (Result{Success: 1, Failed: 1}) {
		t.Fatalf("%+v %v", result, err)
	}
	if reported == nil || !strings.Contains(reported.Error(), "page 2") {
		t.Fatalf("missing page context: %v", reported)
	}
	if _, err := os.Stat(filepath.Join(o.Output, "a.tif.pdf")); !os.IsNotExist(err) {
		t.Fatal("partial PDF exists")
	}
	checkNoTemps(t, o.Output)
}
func TestCancellationCleansPendingPDF(t *testing.T) {
	o := setup(t)
	writePNG(t, filepath.Join(o.Input, "a.png"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o.Report = func(e Event) {
		if e.Status == "processed" {
			cancel()
		}
	}
	_, err := Run(ctx, o)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(o.Output, "a.png.pdf")); !os.IsNotExist(err) {
		t.Fatal("published cancelled PDF")
	}
	checkNoTemps(t, o.Output)
}
func TestPublishRaceDoesNotOverwrite(t *testing.T) {
	o := setup(t)
	writePNG(t, filepath.Join(o.Input, "a.png"))
	target := filepath.Join(o.Output, "a.png.pdf")
	o.Report = func(e Event) {
		if e.Status == "processed" {
			if err := os.WriteFile(target, []byte("other writer"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	result, err := Run(context.Background(), o)
	if err != nil || result.Skipped != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	data, _ := os.ReadFile(target)
	if string(data) != "other writer" {
		t.Fatal("overwrote racing writer")
	}
	checkNoTemps(t, o.Output)
}
func TestContainmentAndSymlinkAliases(t *testing.T) {
	o := setup(t)
	for _, target := range []string{o.Input, filepath.Join(o.Input, "sub"), filepath.Dir(o.Input)} {
		o.Output = target
		if _, err := Run(context.Background(), o); err == nil {
			t.Fatalf("accepted %s", target)
		}
	}
	alias := filepath.Join(filepath.Dir(o.Input), "alias")
	if err := os.Symlink(o.Input, alias); err != nil {
		t.Skip("symlinks unavailable")
	}
	o.Output = filepath.Join(alias, "nested")
	if _, err := Run(context.Background(), o); err == nil {
		t.Fatal("accepted symlink overlap")
	}
}
func TestExcludeDirectoriesSymlinksAndStamp(t *testing.T) {
	o := setup(t)
	os.Mkdir(filepath.Join(o.Input, "sub"), 0755)
	writePNG(t, filepath.Join(o.Input, "sub", "nested.png"))
	if err := os.Link(o.Config.Stamp.Image, filepath.Join(o.Input, "stamp.png")); err != nil {
		t.Skip("hardlinks unavailable")
	}
	if err := os.Symlink(o.Config.Stamp.Image, filepath.Join(o.Input, "link.png")); err != nil {
		t.Skip("symlinks unavailable")
	}
	result, err := Run(context.Background(), o)
	if err != nil || result != (Result{}) {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestCaseInsensitiveDirectoryAlias(t *testing.T) {
	o := setup(t)
	alias := filepath.Join(filepath.Dir(o.Input), "INPUT")
	if _, err := os.Stat(alias); os.IsNotExist(err) {
		t.Skip("case-sensitive filesystem")
	}
	o.Output = filepath.Join(alias, "nested")
	if _, err := Run(context.Background(), o); err == nil {
		t.Fatal("accepted case-insensitive directory alias")
	}
}
