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

func TestSameLengthReplacementMustStillFitTextBox(t *testing.T) {
	s := textSettings()
	s.Text.Value = "WWWW"
	s.Text.Size = 20
	s.Text.Width = 30
	if _, err := PrepareWithFont(context.Background(), s, resolver); err == nil {
		t.Fatal("accepted same-length text that exceeds its configured box")
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
	dst, err := p.Apply(context.Background(), src, "")
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
			out, err := p.Apply(context.Background(), image.NewRGBA(image.Rect(0, 0, 10, 10)), "")
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
	if _, err := p.Apply(context.Background(), image.NewGray(image.Rect(0, 0, 10, 10)), ""); err == nil {
		t.Fatal("stamp overflow accepted")
	}
	if _, err := p.Apply(context.Background(), image.NewGray(image.Rect(0, 0, 100, 100)), ""); err != nil {
		t.Fatal(err)
	}
}

func TestApplyScaledShrinksPageAndKeepsStampPixels(t *testing.T) {
	overlay := image.NewRGBA(image.Rect(0, 0, 6, 4))
	for y := 0; y < overlay.Bounds().Dy(); y++ {
		for x := 0; x < overlay.Bounds().Dx(); x++ {
			overlay.SetRGBA(x, y, color.RGBA{R: 0, A: 255})
		}
	}
	p := Prepared{overlay: overlay, settings: config.Stamp{Position: "top-left"}}
	src := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < src.Bounds().Dy(); y++ {
		for x := 0; x < src.Bounds().Dx(); x++ {
			src.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	got, err := p.ApplyScaled(context.Background(), src, "A4", .5)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds() != image.Rect(0, 0, 20, 10) {
		t.Fatalf("scaled bounds = %v, want 20x10", got.Bounds())
	}
	if got.RGBAAt(5, 3).R != 0 {
		t.Fatal("stamp was scaled down with the page")
	}
	if got.RGBAAt(6, 3).R != 255 {
		t.Fatal("stamp extends past its configured pixel width")
	}
}

func TestApplyScaledRejectsInvalidScale(t *testing.T) {
	p := Prepared{overlay: image.NewRGBA(image.Rect(0, 0, 1, 1)), settings: config.Stamp{Position: "top-left"}}
	for _, scale := range []float64{0, -1, 1.1} {
		if _, err := p.ApplyScaled(context.Background(), image.NewRGBA(image.Rect(0, 0, 2, 2)), "A4", scale); err == nil {
			t.Fatalf("accepted scale %v", scale)
		}
	}
}

func TestPaperSpecificMargins(t *testing.T) {
	positions := map[string]image.Point{"A1": {3, 4}, "A2": {1, 2}}
	x, y := 3, 4
	overlay := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for py := 0; py < 2; py++ {
		for px := 0; px < 2; px++ {
			overlay.Set(px, py, color.Black)
		}
	}
	p := Prepared{overlay, config.Stamp{Position: "top-left", MarginX: 1, MarginY: 2, PaperMargins: map[string]config.PaperMargin{"A1": {X: &x, Y: &y}}}}
	for paper, origin := range positions {
		out, err := p.Apply(context.Background(), image.NewRGBA(image.Rect(0, 0, 10, 10)), paper)
		if err != nil {
			t.Fatal(err)
		}
		if got := out.RGBAAt(origin.X, origin.Y); got.R != 0 || got.G != 0 || got.B != 0 {
			t.Fatalf("%s stamp did not start at %v: %v", paper, origin, got)
		}
	}
}

func TestTextRotation(t *testing.T) {
	boxes := map[int]image.Rectangle{}
	for _, degrees := range []int{0, 90, 180, 270} {
		s := textSettings()
		s.Text.Value = "Test"
		s.Text.Size = 16
		s.Text.Width, s.Text.Height = 80, 50
		s.Text.Rotation = degrees
		p, err := PrepareWithFont(context.Background(), s, resolver)
		if err != nil {
			t.Fatalf("rotation %d: %v", degrees, err)
		}
		boxes[degrees] = nontransparentBounds(p.overlay)
	}
	if boxes[90].Dx() != boxes[0].Dy() || boxes[90].Dy() != boxes[0].Dx() {
		t.Fatalf("90 degree bounds %v do not swap 0 degree bounds %v", boxes[90], boxes[0])
	}
	if boxes[270].Dx() != boxes[0].Dy() || boxes[270].Dy() != boxes[0].Dx() {
		t.Fatalf("270 degree bounds %v do not swap 0 degree bounds %v", boxes[270], boxes[0])
	}
	if boxes[180].Dx() != boxes[0].Dx() || boxes[180].Dy() != boxes[0].Dy() {
		t.Fatalf("180 degree bounds %v differ from 0 degree bounds %v", boxes[180], boxes[0])
	}
	for _, degrees := range []int{90, 180, 270} {
		b := boxes[degrees]
		if b.Min.X < 0 || b.Min.Y < 0 || b.Max.X > 80 || b.Max.Y > 50 {
			t.Fatalf("rotation %d was not centered in the text box: %v", degrees, b)
		}
	}
}

func nontransparentBounds(img *image.RGBA) image.Rectangle {
	var bounds image.Rectangle
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if img.RGBAAt(x, y).A == 0 {
				continue
			}
			if bounds.Empty() {
				bounds = image.Rect(x, y, x+1, y+1)
			} else {
				bounds = bounds.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return bounds
}

func TestRotatedTextMustFitBox(t *testing.T) {
	s := textSettings()
	s.Text.Width, s.Text.Height = 10, 10
	s.Text.Rotation = 90
	if _, err := PrepareWithFont(context.Background(), s, resolver); err == nil {
		t.Fatal("rotated text exceeding its box was accepted")
	}
}

func TestRotateTextDirections(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 2; x++ {
			src.SetRGBA(x, y, color.RGBA{R: uint8(y*2 + x + 1), A: 255})
		}
	}
	for _, tc := range []struct {
		degrees int
		want    []uint8
	}{
		{90, []uint8{5, 3, 1, 6, 4, 2}},
		{180, []uint8{6, 5, 4, 3, 2, 1}},
		{270, []uint8{2, 4, 6, 1, 3, 5}},
	} {
		got := rotateText(src, tc.degrees)
		var values []uint8
		for y := 0; y < got.Bounds().Dy(); y++ {
			for x := 0; x < got.Bounds().Dx(); x++ {
				values = append(values, got.RGBAAt(x, y).R)
			}
		}
		if len(values) != len(tc.want) {
			t.Fatalf("rotation %d dimensions: got %v, want %v", tc.degrees, values, tc.want)
		}
		for i := range values {
			if values[i] != tc.want[i] {
				t.Fatalf("rotation %d pixels: got %v, want %v", tc.degrees, values, tc.want)
			}
		}
	}
}

func TestPrepareRejectsInvalidRotation(t *testing.T) {
	s := textSettings()
	s.Text.Rotation = 45
	if _, err := PrepareWithFont(context.Background(), s, resolver); err == nil {
		t.Fatal("unsupported text rotation was accepted")
	}
}
