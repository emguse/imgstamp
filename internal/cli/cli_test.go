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
	if code := Run(context.Background(), args, &out, &err, "test"); code != 0 || !strings.Contains(err.String(), "no supported") {
		t.Fatalf("empty: %d %s", code, err.String())
	}
	write(filepath.Join(input, "drawing.png"), 100, 200)
	if code := Run(context.Background(), args, &out, &err, "test"); code != 0 {
		t.Fatalf("success: %d %s", code, err.String())
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
