package stamp

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/emguse/imgstamp/internal/config"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

func resolver(context.Context, string) (*sfnt.Font, error) { return sfnt.Parse(goregular.TTF) }
func textSettings() config.Stamp {
	o := 1.0
	return config.Stamp{Position: "top-left", Text: &config.Text{Value: "Test", Font: "Go", Size: 12, Color: "#FF0000", Opacity: &o, Width: 50, Height: 20}}
}
func TestTextValidation(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func(*config.Stamp)
	}{
		{"overflow", func(s *config.Stamp) { s.Text.Width = 1 }},
		{"missing glyph", func(s *config.Stamp) { s.Text.Value = "\U0010ffff" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := textSettings()
			tt.edit(&s)
			if _, err := PrepareWithFont(context.Background(), s, resolver); err == nil {
				t.Fatal("invalid text accepted")
			}
		})
	}
}
func TestOverlayAndSourceUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seal.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	seal := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	seal.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	if err := png.Encode(f, seal); err != nil {
		t.Fatal(err)
	}
	f.Close()
	p, err := Prepare(context.Background(), config.Stamp{Image: path, Position: "top-left", BackgroundOpacity: .5, ImageOpacity: .5})
	if err != nil {
		t.Fatal(err)
	}
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			src.SetRGBA(x, y, color.RGBA{A: 255})
		}
	}
	dst, err := p.Apply(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if src.RGBAAt(0, 0) != (color.RGBA{A: 255}) {
		t.Fatal("modified source")
	}
	if got := dst.RGBAAt(1, 1); got.R < 127 || got.R > 129 || got.G != got.R {
		t.Fatalf("white background alpha: %v", got)
	}
	if got := dst.RGBAAt(0, 0); got.R < 190 || got.R > 193 || got.G < 63 || got.G > 65 {
		t.Fatalf("image opacity composition: %v", got)
	}
	if dst.RGBAAt(3, 3) != (color.RGBA{A: 255}) {
		t.Fatal("changed pixels outside stamp")
	}
}
func TestNinePositions(t *testing.T) {
	cases := map[string]image.Point{"top-left": {1, 2}, "top-center": {4, 2}, "top-right": {7, 2}, "center-left": {1, 4}, "center": {4, 4}, "center-right": {7, 4}, "bottom-left": {1, 6}, "bottom-center": {4, 6}, "bottom-right": {7, 6}}
	for pos, origin := range cases {
		t.Run(pos, func(t *testing.T) {
			overlay := image.NewRGBA(image.Rect(0, 0, 2, 2))
			for y := 0; y < 2; y++ {
				for x := 0; x < 2; x++ {
					overlay.Set(x, y, color.Black)
				}
			}
			p := Prepared{overlay, config.Stamp{Position: pos, MarginX: 1, MarginY: 2}}
			out, err := p.Apply(context.Background(), image.NewRGBA(image.Rect(0, 0, 10, 10)))
			if err != nil {
				t.Fatal(err)
			}
			for y := 0; y < 10; y++ {
				for x := 0; x < 10; x++ {
					want := uint8(255)
					if image.Pt(x, y).In(image.Rectangle{Min: origin, Max: origin.Add(image.Pt(2, 2))}) {
						want = 0
					}
					if out.RGBAAt(x, y).R != want {
						t.Fatalf("incorrect pixel at %d,%d", x, y)
					}
				}
			}
		})
	}
}
func TestTextOnlyAndPageOverflow(t *testing.T) {
	p, err := PrepareWithFont(context.Background(), textSettings(), resolver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Apply(context.Background(), image.NewGray(image.Rect(0, 0, 10, 10))); err == nil {
		t.Fatal("stamp overflow accepted")
	}
	if _, err := p.Apply(context.Background(), image.NewGray(image.Rect(0, 0, 100, 100))); err != nil {
		t.Fatal(err)
	}
}
