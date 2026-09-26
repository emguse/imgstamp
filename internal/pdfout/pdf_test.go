package pdfout

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"image"
	"image/color"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/emguse/imgstamp/internal/paper"
)

func TestPDFImageStreamsAndPageSizes(t *testing.T) {
	var out bytes.Buffer
	w := New(&out)
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	img.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
	if err := w.Add(context.Background(), img, paper.Size{Name: "A4", WidthMM: 210, HeightMM: 297}, true); err != nil {
		t.Fatal(err)
	}
	if err := w.Add(context.Background(), img, paper.Size{Name: "A3", WidthMM: 420, HeightMM: 297}, false); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data := out.String()
	for _, want := range []string{"/Count 2", "/MediaBox [0 0 595.275591 841.889764]", "/MediaBox [0 0 1190.551181 841.889764]", "/DeviceGray", "/DeviceRGB"} {
		if !strings.Contains(data, want) {
			t.Fatalf("missing %s", want)
		}
	}
	streams := regexp.MustCompile(`(?s)/Filter /FlateDecode /Length [0-9]+ 0 R >>\nstream\n(.*?)\nendstream`).FindAllSubmatch(out.Bytes(), -1)
	if len(streams) != 2 {
		t.Fatalf("expected 2 images, got %d", len(streams))
	}
	wants := [][]byte{{76, 150}, {255, 0, 0, 0, 255, 0}}
	for i, s := range streams {
		r, err := zlib.NewReader(bytes.NewReader(s[1]))
		if err != nil {
			t.Fatal(err)
		}
		pixels, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(pixels, wants[i]) {
			t.Fatalf("pixels: %v", pixels)
		}
	}
	// Independently verify xref entries point at object headers.
	xref := strings.LastIndex(data, "\nxref\n")
	if xref < 0 {
		t.Fatal("missing xref")
	}
	rows := strings.Split(data[xref+1:], "\n")
	count, _ := strconv.Atoi(strings.Fields(rows[1])[1])
	for id := 1; id < count; id++ {
		off, err := strconv.Atoi(strings.Fields(rows[id+2])[0])
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(data[off:], strconv.Itoa(id)+" 0 obj\n") {
			t.Fatalf("bad xref for %d", id)
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }
func TestWriteFailureAndCancellation(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	size := paper.Size{WidthMM: 210, HeightMM: 297}
	if err := New(failingWriter{}).Add(context.Background(), img, size, true); err == nil {
		t.Fatal("lost write error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := New(io.Discard).Add(ctx, img, size, true); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := New(io.Discard).Close(); err == nil {
		t.Fatal("accepted empty PDF")
	}
}
