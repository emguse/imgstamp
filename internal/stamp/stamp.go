// Package stamp prepares a reusable overlay and composites it on each page.
package stamp

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/emguse/imgstamp/internal/config"
	"github.com/emguse/imgstamp/internal/fonts"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Prepared is a reusable, preflight-validated stamp overlay.
type Prepared struct {
	overlay  *image.RGBA
	settings config.Stamp
}

// Resolver obtains a font by its full system name.
type Resolver func(context.Context, string) (*sfnt.Font, error)

// Prepare builds an overlay using locally installed fonts.
func Prepare(ctx context.Context, s config.Stamp) (*Prepared, error) {
	return PrepareWithFont(ctx, s, fonts.Resolve)
}

// PrepareWithFont allows deterministic font resolution, including in tests.
func PrepareWithFont(ctx context.Context, s config.Stamp, resolve Resolver) (*Prepared, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var source image.Image
	var width, height int
	if s.Image != "" {
		f, err := os.Open(s.Image)
		if err != nil {
			return nil, fmt.Errorf("stamp image: %w", err)
		}
		defer f.Close()
		dim, err := png.DecodeConfig(f)
		if err != nil {
			return nil, fmt.Errorf("stamp must be PNG: %w", err)
		}
		if !config.StampPixelsOK(dim.Width, dim.Height) {
			return nil, fmt.Errorf("stamp image exceeds pixel limit")
		}
		width, height = dim.Width, dim.Height
		if s.Width != nil {
			width = *s.Width
			height = max(1, int(math.Round(float64(dim.Height)*float64(width)/float64(dim.Width))))
		}
		if !config.StampPixelsOK(width, height) {
			return nil, fmt.Errorf("resized stamp exceeds pixel limit")
		}
		if _, err = f.Seek(0, 0); err != nil {
			return nil, err
		}
		source, err = png.Decode(f)
		if err != nil {
			return nil, err
		}
	} else if s.Text != nil {
		width, height = s.Text.Width, s.Text.Height
	}
	if !config.StampPixelsOK(width, height) {
		return nil, fmt.Errorf("invalid stamp dimensions")
	}
	var face font.Face
	var textOrigin image.Point
	var textSize image.Point
	var glyphSize image.Point
	if t := s.Text; t != nil {
		if t.Rotation != 0 && t.Rotation != 90 && t.Rotation != 180 && t.Rotation != 270 {
			return nil, fmt.Errorf("text rotation_deg must be 0, 90, 180, or 270")
		}
		if t.X > width-t.Width || t.Y > height-t.Height {
			return nil, fmt.Errorf("text box lies outside stamp %dx%d", width, height)
		}
		f, err := resolve(ctx, t.Font)
		if err != nil {
			return nil, err
		}
		for _, r := range t.Value {
			glyph, err := f.GlyphIndex(nil, r)
			if err != nil {
				return nil, err
			}
			if glyph == 0 {
				return nil, fmt.Errorf("font %q lacks character %U", t.Font, r)
			}
		}
		face, err = opentype.NewFace(f, &opentype.FaceOptions{Size: t.Size, DPI: 72, Hinting: font.HintingNone})
		if err != nil {
			return nil, err
		}
		defer face.Close()
		bounds, advance := font.BoundString(face, t.Value)
		left, top := bounds.Min.X.Floor(), bounds.Min.Y.Floor()
		textWidth := max(bounds.Max.X.Ceil(), advance.Ceil()) - min(0, left)
		textHeight := bounds.Max.Y.Ceil() - top
		glyphSize = image.Pt(textWidth, textHeight)
		textSize = glyphSize
		if t.Rotation == 90 || t.Rotation == 270 {
			textSize = image.Pt(textHeight, textWidth)
		}
		if textSize.X > t.Width || textSize.Y > t.Height {
			label := "text"
			if t.Rotation != 0 {
				label = "rotated text"
			}
			return nil, fmt.Errorf("%s needs %dx%d pixels but box is %dx%d", label, textSize.X, textSize.Y, t.Width, t.Height)
		}
		textOrigin = image.Pt(t.X-min(0, left), t.Y+(t.Height-textHeight)/2-top)
	}
	overlay := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(overlay, overlay.Bounds(), image.NewUniform(color.NRGBA{255, 255, 255, alpha(s.BackgroundOpacity)}), image.Point{}, draw.Src)
	if source != nil {
		if source.Bounds().Dx() != width || source.Bounds().Dy() != height {
			scaled := image.NewRGBA(overlay.Bounds())
			xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), source, source.Bounds(), draw.Src, nil)
			source = scaled
		}
		draw.DrawMask(overlay, overlay.Bounds(), source, source.Bounds().Min, image.NewUniform(color.Alpha{alpha(s.ImageOpacity)}), image.Point{}, draw.Over)
	}
	if t := s.Text; t != nil {
		rgb, err := strconv.ParseUint(strings.TrimPrefix(t.Color, "#"), 16, 24)
		if err != nil {
			return nil, err
		}
		ink := color.NRGBA{uint8(rgb >> 16), uint8(rgb >> 8), uint8(rgb), alpha(*t.Opacity)}
		if t.Rotation == 0 {
			d := font.Drawer{Dst: overlay, Src: image.NewUniform(ink), Face: face, Dot: fixed.P(textOrigin.X, textOrigin.Y)}
			d.DrawString(t.Value)
		} else {
			bounds, _ := font.BoundString(face, t.Value)
			left, top := bounds.Min.X.Floor(), bounds.Min.Y.Floor()
			glyph := image.NewRGBA(image.Rect(0, 0, glyphSize.X, glyphSize.Y))
			d := font.Drawer{Dst: glyph, Src: image.NewUniform(ink), Face: face, Dot: fixed.P(-min(0, left), -top)}
			d.DrawString(t.Value)
			rotated := rotateText(glyph, t.Rotation)
			target := image.Pt(t.X+(t.Width-rotated.Bounds().Dx())/2, t.Y+(t.Height-rotated.Bounds().Dy())/2)
			draw.Draw(overlay, rotated.Bounds().Add(target), rotated, image.Point{}, draw.Over)
		}
	}
	return &Prepared{overlay, s}, nil
}

func rotateText(src *image.RGBA, degrees int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	outBounds := image.Rect(0, 0, w, h)
	if degrees == 90 || degrees == 270 {
		outBounds = image.Rect(0, 0, h, w)
	}
	out := image.NewRGBA(outBounds)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var p image.Point
			switch degrees {
			case 90:
				p = image.Pt(h-1-y, x)
			case 180:
				p = image.Pt(w-1-x, h-1-y)
			case 270:
				p = image.Pt(y, w-1-x)
			}
			out.Set(p.X, p.Y, src.At(x, y))
		}
	}
	return out
}

func alpha(v float64) uint8 { return uint8(math.Round(v * 255)) }

// Apply composites onto an opaque white-backed copy without changing src.
func (p *Prepared) Apply(ctx context.Context, src image.Image, paper string) (*image.RGBA, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	sw, sh := p.overlay.Bounds().Dx(), p.overlay.Bounds().Dy()
	s := p.settings
	marginX, marginY := s.MarginX, s.MarginY
	if margin, ok := s.PaperMargins[paper]; ok {
		if margin.X != nil {
			marginX = *margin.X
		}
		if margin.Y != nil {
			marginY = *margin.Y
		}
	}
	x, y := (w-sw)/2, (h-sh)/2
	if strings.HasSuffix(s.Position, "left") {
		x = marginX
	}
	if strings.HasSuffix(s.Position, "right") {
		x = w - sw - marginX
	}
	if strings.HasPrefix(s.Position, "top") {
		y = marginY
	}
	if strings.HasPrefix(s.Position, "bottom") {
		y = h - sh - marginY
	}
	if x < 0 || y < 0 || x+sw > w || y+sh > h {
		return nil, fmt.Errorf("stamp %dx%d at (%d,%d) does not fit page %dx%d", sw, sh, x, y, w, h)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	draw.Draw(dst, image.Rect(x, y, x+sw, y+sh), p.overlay, image.Point{}, draw.Over)
	return dst, ctx.Err()
}
