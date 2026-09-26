package fonts

import (
	"context"
	"golang.org/x/image/font/gofont/goregular"
	"os"
	"path/filepath"
	"testing"
)

func TestScanAndLoad(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.ttf"), goregular.TTF, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.ttf"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	entries, err := Scan(context.Background(), []string{dir, dir, filepath.Join(dir, "missing")})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %+v", entries)
	}
	face, err := Load(entries[0])
	if err != nil {
		t.Fatal(err)
	}
	g, err := face.GlyphIndex(nil, 'A')
	if err != nil || g == 0 {
		t.Fatal("font cannot render A")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Scan(ctx, []string{dir}); err == nil {
		t.Fatal("ignored cancellation")
	}
}
