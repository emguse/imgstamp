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

func TestReducedPaperMappings(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		sourceName            string
		width, height         float64
		wantName              string
		wantWidth, wantHeight float64
		wantReduced           bool
	}{
		{"A1 portrait", "A1", 594, 841, "A3", 297, 420, true},
		{"A1 landscape", "A1", 841, 594, "A3", 420, 297, true},
		{"A2 portrait", "A2", 420, 594, "A3", 297, 420, true},
		{"A3 portrait", "A3", 297, 420, "A4", 210, 297, true},
		{"A3 landscape", "A3", 420, 297, "A4", 297, 210, true},
		{"A4 unchanged", "A4", 210, 297, "A4", 210, 297, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, reduced := (Size{Name: tc.sourceName, WidthMM: tc.width, HeightMM: tc.height}).Reduced()
			if got.Name != tc.wantName || got.WidthMM != tc.wantWidth || got.HeightMM != tc.wantHeight || reduced != tc.wantReduced {
				t.Fatalf("Reduced() = %+v, %t; want %s %.0fx%.0f mm, %t", got, reduced, tc.wantName, tc.wantWidth, tc.wantHeight, tc.wantReduced)
			}
		})
	}
}
