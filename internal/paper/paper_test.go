package paper

import "testing"

func TestMatchToleranceAndOrientation(t *testing.T) {
	c := Config{Tolerance: 2, Rules: []Rule{{Name: "a", Short: 1000, Long: 2000, Paper: "A3"}}}
	for _, tt := range []struct {
		name          string
		w, h          int
		ok, landscape bool
	}{
		{"exact", 1000, 2000, true, false}, {"landscape", 2000, 1000, true, true},
		{"lower boundary", 980, 1960, true, false}, {"upper boundary", 1020, 2040, true, false},
		{"short outside", 979, 2000, false, false}, {"long outside", 1000, 2041, false, false},
		{"invalid", 0, 2000, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			size, err := c.Match(tt.w, tt.h)
			if (err == nil) != tt.ok {
				t.Fatalf("size=%+v err=%v", size, err)
			}
			if tt.ok && (size.WidthMM > size.HeightMM) != tt.landscape {
				t.Fatal("wrong orientation")
			}
		})
	}
}
func TestOverridesAndAmbiguity(t *testing.T) {
	zero := 0.0
	c := Config{Tolerance: 2, Rules: []Rule{{Name: "exact", Short: 100, Long: 200, Paper: "A4", Tolerance: &zero}}}
	if _, err := c.Match(101, 200); err == nil {
		t.Fatal("rule override ignored")
	}
	c.Rules = append(c.Rules, Rule{Name: "same-paper", Short: 100, Long: 200, Paper: "A4"})
	if _, err := c.Match(100, 200); err == nil {
		t.Fatal("same-paper ambiguity accepted")
	}
	c.Rules[1].Short = 150
	c.Rules[1].Long = 300
	if _, err := c.Match(150, 300); err != nil {
		t.Fatal(err)
	}
}
