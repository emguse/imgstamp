// Package config loads and validates the user-facing TOML configuration.
package config

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/emguse/imgstamp/internal/paper"
	"github.com/pelletier/go-toml/v2"
)

// MaxPixels bounds decoded page and stamp allocations.
const MaxPixels = 100_000_000

// Config is the complete TOML application configuration.
type Config struct {
	Output Output       `toml:"output"`
	Paper  paper.Config `toml:"paper_detection"`
	Stamp  Stamp        `toml:"stamp"`
}

// Output controls PDF image color mode.
type Output struct {
	ColorMode string `toml:"color_mode"`
}

// Stamp describes the overlay and its page-relative placement.
type Stamp struct {
	Image             string  `toml:"image"`
	Width             *int    `toml:"width_px"`
	Position          string  `toml:"position"`
	MarginX           int     `toml:"margin_x_px"`
	MarginY           int     `toml:"margin_y_px"`
	BackgroundOpacity float64 `toml:"background_opacity"`
	ImageOpacity      float64 `toml:"image_opacity"`
	Text              *Text   `toml:"text"`
}

// Text describes a single line and its stamp-local bounding box.
type Text struct {
	Value   string   `toml:"value"`
	Font    string   `toml:"font"`
	Size    float64  `toml:"size_px"`
	Color   string   `toml:"color"`
	Opacity *float64 `toml:"opacity"`
	X       int      `toml:"box_x_px"`
	Y       int      `toml:"box_y_px"`
	Width   int      `toml:"box_width_px"`
	Height  int      `toml:"box_height_px"`
}

// Load applies defaults, rejects unknown keys, validates, and resolves image paths.
func Load(path string) (Config, error) {
	c := Config{Output: Output{"grayscale"}, Paper: paper.Config{Tolerance: 2}, Stamp: Stamp{Position: "bottom-right", MarginX: 16, MarginY: 16, BackgroundOpacity: .5, ImageOpacity: .8}}
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	if err = toml.NewDecoder(f).DisallowUnknownFields().Decode(&c); err != nil {
		return c, fmt.Errorf("configuration %s: %w", path, err)
	}
	if t := c.Stamp.Text; t != nil {
		if t.Color == "" {
			t.Color = "#000000"
		}
		if t.Opacity == nil {
			v := .8
			t.Opacity = &v
		}
	}
	if err = c.Validate(); err != nil {
		return c, err
	}
	if c.Stamp.Image != "" && !filepath.IsAbs(c.Stamp.Image) {
		c.Stamp.Image = filepath.Join(filepath.Dir(path), c.Stamp.Image)
	}
	return c, nil
}

// PixelsOK checks dimensions without overflowing a width-times-height product.
func PixelsOK(w, h int) bool { return w > 0 && h > 0 && w <= MaxPixels && h <= MaxPixels/w }
func unit(v float64) bool    { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1 }

var positions = map[string]bool{"top-left": true, "top-center": true, "top-right": true, "center-left": true, "center": true, "center-right": true, "bottom-left": true, "bottom-center": true, "bottom-right": true}
var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Validate rejects invalid or incomplete application settings.
func (c Config) Validate() error {
	if c.Output.ColorMode != "grayscale" && c.Output.ColorMode != "color" {
		return fmt.Errorf("output.color_mode must be grayscale or color")
	}
	if err := c.Paper.Validate(); err != nil {
		return err
	}
	s := c.Stamp
	if s.Image == "" && s.Text == nil {
		return fmt.Errorf("stamp.image or stamp.text is required")
	}
	if !positions[s.Position] {
		return fmt.Errorf("unsupported stamp.position %q", s.Position)
	}
	if s.MarginX < 0 || s.MarginY < 0 || s.MarginX > MaxPixels || s.MarginY > MaxPixels {
		return fmt.Errorf("stamp margins must be between 0 and %d", MaxPixels)
	}
	if s.Width != nil && (s.Image == "" || *s.Width <= 0 || *s.Width > MaxPixels) {
		return fmt.Errorf("stamp.width_px requires an image and a positive bounded width")
	}
	if !unit(s.BackgroundOpacity) || !unit(s.ImageOpacity) {
		return fmt.Errorf("stamp opacity must be between 0 and 1")
	}
	if t := s.Text; t != nil {
		if strings.TrimSpace(t.Value) == "" || strings.TrimSpace(t.Font) == "" {
			return fmt.Errorf("text value and font are required")
		}
		if strings.IndexFunc(t.Value, func(r rune) bool { return unicode.IsControl(r) || r == '\u2028' || r == '\u2029' }) >= 0 {
			return fmt.Errorf("text must be a single line without control characters")
		}
		if math.IsNaN(t.Size) || math.IsInf(t.Size, 0) || t.Size <= 0 || t.Size > 1_000_000 {
			return fmt.Errorf("text size_px must be positive and at most 1000000")
		}
		if !hexColor.MatchString(t.Color) {
			return fmt.Errorf("text color must be #RRGGBB")
		}
		if t.Opacity == nil || !unit(*t.Opacity) {
			return fmt.Errorf("text opacity must be between 0 and 1")
		}
		if !PixelsOK(t.Width, t.Height) || t.X < 0 || t.Y < 0 || t.X > MaxPixels || t.Y > MaxPixels {
			return fmt.Errorf("invalid text box or text box exceeds %d pixels", MaxPixels)
		}
		if s.Image == "" && (t.X != 0 || t.Y != 0) {
			return fmt.Errorf("text-only stamp requires box_x_px = box_y_px = 0")
		}
	}
	return nil
}
