package cli

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emguse/imgstamp/internal/config"
	"github.com/emguse/imgstamp/internal/paper"
)

func TestHelpVersionAndInvalidArguments(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want int
	}{{[]string{"--help"}, 0}, {[]string{"--version"}, 0}, {nil, 2}, {[]string{"fonts", "extra"}, 2}, {[]string{"--invalid"}, 2}} {
		var out, err bytes.Buffer
		if got := Run(context.Background(), tt.args, &out, &err, "test"); got != tt.want {
			t.Fatalf("%v returned %d: %s", tt.args, got, err.String())
		}
	}
}

func TestTextOverrideValidation(t *testing.T) {
	newConfig := func() config.Config {
		opacity := .8
		return config.Config{
			Output: config.Output{ColorMode: "grayscale"},
			Paper:  paper.Config{Rules: []paper.Rule{{Name: "a4", Short: 100, Long: 200, Paper: "A4"}}},
			Stamp: config.Stamp{
				Position: "bottom-right",
				Text:     &config.Text{Value: "WBS{d08}-", Font: "Go", Size: 12, Color: "#000000", Opacity: &opacity, Width: 200, Height: 30},
			},
		}
	}

	t.Run("same Unicode character count accepted", func(t *testing.T) {
		cfg := newConfig()
		if err := overrideText(&cfg, "東京{d089}-"); err != nil {
			t.Fatal(err)
		}
		if cfg.Stamp.Text.Value != "東京{d089}-" {
			t.Fatalf("text value = %q", cfg.Stamp.Text.Value)
		}
	})
	t.Run("different count rejected without mutation", func(t *testing.T) {
		cfg := newConfig()
		if err := overrideText(&cfg, "WBS-"); err == nil {
			t.Fatal("accepted different character count")
		}
		if cfg.Stamp.Text.Value != "WBS{d08}-" {
			t.Fatal("failed override changed configured text")
		}
	})
	t.Run("control character rejected without mutation", func(t *testing.T) {
		cfg := newConfig()
		if err := overrideText(&cfg, "WBS{d\n8}-"); err == nil {
			t.Fatal("accepted line break")
		}
		if cfg.Stamp.Text.Value != "WBS{d08}-" {
			t.Fatal("failed override changed configured text")
		}
	})
	t.Run("text field required", func(t *testing.T) {
		cfg := newConfig()
		cfg.Stamp.Text = nil
		if err := overrideText(&cfg, "WBS{d08}-"); err == nil {
			t.Fatal("accepted text override without configured text field")
		}
	})
}

func TestCLIConversionExitCodes(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "in")
	output := filepath.Join(dir, "out")
	os.Mkdir(input, 0755)
	write := func(path string, w, h int) {
		t.Helper()
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, image.NewGray(image.Rect(0, 0, w, h))); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	write(filepath.Join(dir, "seal.png"), 2, 2)
	path := filepath.Join(dir, "stamp.toml")
	os.WriteFile(path, []byte(`
[[paper_detection.rules]]
name="small"
short_px=100
long_px=200
paper="A4"
[stamp]
image="seal.png"
`), 0600)
	args := []string{"--config", path, "--input", input, "--output", output}
	var out, err bytes.Buffer
	if code := Run(context.Background(), append(args, "--text", "DO-NOT-ECHO"), &out, &err, "test"); code != 2 || strings.Contains(err.String(), "DO-NOT-ECHO") {
		t.Fatalf("text without configured field: code=%d stderr=%s", code, err.String())
	}
	if code := Run(context.Background(), args, &out, &err, "test"); code != 0 || !strings.Contains(err.String(), "no supported") {
		t.Fatalf("empty: %d %s", code, err.String())
	}
	write(filepath.Join(input, "drawing.png"), 100, 200)
	if code := Run(context.Background(), args, &out, &err, "test"); code != 0 {
		t.Fatalf("success: %d %s", code, err.String())
	}
	shrinkOutput := filepath.Join(dir, "shrunk")
	shrinkArgs := []string{"--config", path, "--input", input, "--output", shrinkOutput, "--shrink"}
	if code := Run(context.Background(), shrinkArgs, &out, &err, "test"); code != 0 {
		t.Fatalf("shrink success: %d %s", code, err.String())
	}
	shrunkPDF, readErr := os.ReadFile(filepath.Join(shrinkOutput, "drawing.png.pdf"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(shrunkPDF), "/MediaBox [0 0 595.275591 841.889764]") {
		t.Fatal("--shrink changed A4 input page size")
	}
	os.WriteFile(filepath.Join(input, "broken.tif"), []byte("bad tiff"), 0600)
	if code := Run(context.Background(), args, &out, &err, "test"); code != 1 {
		t.Fatalf("partial failure: %d %s", code, err.String())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := Run(ctx, args, &out, &err, "test"); code != 130 {
		t.Fatalf("cancel: %d", code)
	}
}
