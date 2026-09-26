package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const basic = `
[[paper_detection.rules]]
name = "test"
short_px = 100
long_px = 200
paper = "A4"
[stamp]
image = "seal.png"
`

func loadText(t *testing.T, s string) (Config, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stamp.toml")
	if err := os.WriteFile(path, []byte(s), 0600); err != nil {
		t.Fatal(err)
	}
	return Load(path)
}
func TestDefaultsAndExplicitZero(t *testing.T) {
	c, err := loadText(t, basic)
	if err != nil {
		t.Fatal(err)
	}
	if c.Stamp.BackgroundOpacity != .5 || c.Stamp.MarginX != 16 || c.Paper.Tolerance != 2 || c.Output.ColorMode != "grayscale" {
		t.Fatalf("bad defaults: %+v", c)
	}
	if !filepath.IsAbs(c.Stamp.Image) {
		t.Fatal("image path was not resolved relative to config")
	}
	c, err = loadText(t, "[paper_detection]\ntolerance_percent=0\n"+basic+"background_opacity=0\nmargin_x_px=0\nimage_opacity=0\n")
	if err != nil {
		t.Fatal(err)
	}
	if c.Paper.Tolerance != 0 || c.Stamp.BackgroundOpacity != 0 || c.Stamp.MarginX != 0 || c.Stamp.ImageOpacity != 0 {
		t.Fatal("explicit zero replaced by default")
	}
}
func TestInvalidSettings(t *testing.T) {
	cases := map[string]string{
		"unknown":        basic + "unknown=1\n",
		"nan":            basic + "background_opacity=nan\n",
		"negative":       basic + "margin_x_px=-1\n",
		"width":          basic + "width_px=0\n",
		"paper":          strings.Replace(basic, `paper = "A4"`, `paper = "Letter"`, 1),
		"bad tolerance":  "[paper_detection]\ntolerance_percent=11\n" + basic,
		"ambiguous name": basic + "\n[[paper_detection.rules]]\nname='test'\nshort_px=50\nlong_px=60\npaper='A3'\n",
		"multiline":      basic + "\n[stamp.text]\nvalue='''a\nb'''\nfont='Go'\nsize_px=10\nbox_width_px=20\nbox_height_px=20\n",
		"unknown text":   basic + "\n[stamp.text]\nvalue='ok'\nfont='Go'\nsize_px=10\nbox_width_px=20\nbox_height_px=20\ntypo=1\n",
	}
	for name, s := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := loadText(t, s); err == nil {
				t.Fatal("accepted invalid settings")
			}
		})
	}
}
func TestExampleLoads(t *testing.T) {
	c, err := Load("../../examples/stamp.toml")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Paper.Rules) != 4 {
		t.Fatal("expected four paper rules")
	}
}
func TestTextDefaults(t *testing.T) {
	c, err := loadText(t, basic+"\n[stamp.text]\nvalue='ok'\nfont='Go'\nsize_px=10\nbox_width_px=50\nbox_height_px=20\n")
	if err != nil {
		t.Fatal(err)
	}
	if c.Stamp.Text.Color != "#000000" || *c.Stamp.Text.Opacity != .8 {
		t.Fatal("text defaults missing")
	}
}
