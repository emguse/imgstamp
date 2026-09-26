// Package paper matches pixel dimensions to configured output paper sizes.
package paper

import (
	"fmt"
	"math"
	"strings"
)

// Rule associates expected pixel dimensions with an output paper size.
type Rule struct {
	Name      string   `toml:"name"`
	Short     int      `toml:"short_px"`
	Long      int      `toml:"long_px"`
	Paper     string   `toml:"paper"`
	Tolerance *float64 `toml:"tolerance_percent"`
}

// Config defines the matching tolerance and candidate rules.
type Config struct {
	Tolerance float64 `toml:"tolerance_percent"`
	Rules     []Rule  `toml:"rules"`
}

// Size describes a matched paper in its display orientation.
type Size struct {
	Name              string
	WidthMM, HeightMM float64
}

var sizes = map[string][2]float64{"A1": {594, 841}, "A2": {420, 594}, "A3": {297, 420}, "A4": {210, 297}}

// ValidTolerance reports whether a percentage is finite and within 0..10.
func ValidTolerance(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 10 }

// Validate checks rule names, dimensions, paper names, and tolerances.
func (c Config) Validate() error {
	if !ValidTolerance(c.Tolerance) {
		return fmt.Errorf("paper_detection.tolerance_percent must be between 0 and 10")
	}
	if len(c.Rules) == 0 {
		return fmt.Errorf("at least one paper_detection rule is required")
	}
	names := map[string]bool{}
	for i, r := range c.Rules {
		if strings.TrimSpace(r.Name) == "" || names[r.Name] {
			return fmt.Errorf("paper rule %d needs a unique nonempty name", i+1)
		}
		names[r.Name] = true
		if r.Short <= 0 || r.Long < r.Short {
			return fmt.Errorf("paper rule %q needs 0 < short_px <= long_px", r.Name)
		}
		if _, ok := sizes[r.Paper]; !ok {
			return fmt.Errorf("paper rule %q: unsupported paper %q", r.Name, r.Paper)
		}
		if r.Tolerance != nil && !ValidTolerance(*r.Tolerance) {
			return fmt.Errorf("paper rule %q: tolerance must be between 0 and 10", r.Name)
		}
	}
	return nil
}

// Match returns the sole matching paper or an explicit no-match/ambiguity error.
func (c Config) Match(width, height int) (Size, error) {
	if width <= 0 || height <= 0 {
		return Size{}, fmt.Errorf("invalid page dimensions %dx%d", width, height)
	}
	short, long := min(width, height), max(width, height)
	matches := []Rule{}
	for _, r := range c.Rules {
		tolerance := c.Tolerance
		if r.Tolerance != nil {
			tolerance = *r.Tolerance
		}
		if math.Abs(float64(short)-float64(r.Short))*100 <= float64(r.Short)*tolerance && math.Abs(float64(long)-float64(r.Long))*100 <= float64(r.Long)*tolerance {
			matches = append(matches, r)
		}
	}
	if len(matches) == 0 {
		return Size{}, fmt.Errorf("no paper rule matches %dx%d pixels", width, height)
	}
	if len(matches) > 1 {
		names := []string{}
		for _, r := range matches {
			names = append(names, r.Name)
		}
		return Size{}, fmt.Errorf("ambiguous paper rules for %dx%d: %s", width, height, strings.Join(names, ", "))
	}
	r := matches[0]
	mm := sizes[r.Paper]
	if width > height {
		mm[0], mm[1] = mm[1], mm[0]
	}
	return Size{r.Paper, mm[0], mm[1]}, nil
}
