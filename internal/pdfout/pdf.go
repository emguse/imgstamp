// Package pdfout writes a minimal image-only PDF incrementally. Image streams
// are Flate-compressed directly to disk, so memory does not grow with page count.
package pdfout

import (
	"compress/zlib"
	"context"
	"fmt"
	"image"
	"io"
	"math"
	"strings"

	"github.com/emguse/imgstamp/internal/paper"
)

type counter struct {
	w   io.Writer
	n   int64
	err error
}

func (c *counter) Write(p []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	n, err := c.w.Write(p)
	c.n += int64(n)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	c.err = err
	return n, err
}

// Writer streams opaque page images and retains only PDF object offsets.
type Writer struct {
	out     *counter
	offsets []int64
	pages   []int
	closed  bool
}

// New starts a PDF on out; Add and Close report any write errors.
func New(out io.Writer) *Writer {
	w := &Writer{out: &counter{w: out}, offsets: make([]int64, 3)}
	w.write("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	return w
}
func (w *Writer) write(s string)              { _, _ = io.WriteString(w.out, s) }
func (w *Writer) reserve() int                { n := len(w.offsets); w.offsets = append(w.offsets, 0); return n }
func (w *Writer) begin(id int)                { w.offsets[id] = w.out.n; w.write(fmt.Sprintf("%d 0 obj\n", id)) }
func (w *Writer) object(id int, value string) { w.begin(id); w.write(value + "\nendobj\n") }

// Add losslessly writes a page image centered and fitted onto the given paper.
func (w *Writer) Add(ctx context.Context, img *image.RGBA, size paper.Size, gray bool) error {
	if w.closed {
		return fmt.Errorf("PDF already closed")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	imageID, lengthID, contentID, pageID := w.reserve(), w.reserve(), w.reserve(), w.reserve()
	space, channels := "/DeviceRGB", 3
	if gray {
		space, channels = "/DeviceGray", 1
	}
	w.begin(imageID)
	w.write(fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace %s /BitsPerComponent 8 /Filter /FlateDecode /Length %d 0 R >>\nstream\n", width, height, space, lengthID))
	start := w.out.n
	zip := zlib.NewWriter(w.out)
	row := make([]byte, width*channels)
	for y := 0; y < height; y++ {
		if err := ctx.Err(); err != nil {
			_ = zip.Close()
			return err
		}
		pixels := img.Pix[img.PixOffset(b.Min.X, b.Min.Y+y):]
		for x := 0; x < width; x++ {
			r, g, bl := pixels[x*4], pixels[x*4+1], pixels[x*4+2]
			if gray {
				row[x] = uint8((19595*uint32(r) + 38470*uint32(g) + 7471*uint32(bl) + 32768) >> 16)
			} else {
				row[x*3], row[x*3+1], row[x*3+2] = r, g, bl
			}
		}
		if _, err := zip.Write(row); err != nil {
			_ = zip.Close()
			return err
		}
	}
	if err := zip.Close(); err != nil {
		return err
	}
	length := w.out.n - start
	w.write("\nendstream\nendobj\n")
	w.object(lengthID, fmt.Sprint(length))
	pw, ph := size.WidthMM*72/25.4, size.HeightMM*72/25.4
	scale := math.Min(pw/float64(width), ph/float64(height))
	iw, ih := float64(width)*scale, float64(height)*scale
	content := fmt.Sprintf("q\n%.6f 0 0 %.6f %.6f %.6f cm\n/Im0 Do\nQ\n", iw, ih, (pw-iw)/2, (ph-ih)/2)
	w.object(contentID, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content))
	w.object(pageID, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.6f %.6f] /Resources << /XObject << /Im0 %d 0 R >> >> /Contents %d 0 R >>", pw, ph, imageID, contentID))
	w.pages = append(w.pages, pageID)
	return w.out.err
}

// Close writes the page tree and cross-reference table, but does not close out.
func (w *Writer) Close() error {
	if w.closed {
		return fmt.Errorf("PDF already closed")
	}
	w.closed = true
	if len(w.pages) == 0 {
		return fmt.Errorf("cannot create empty PDF")
	}
	kids := make([]string, len(w.pages))
	for i, id := range w.pages {
		kids[i] = fmt.Sprintf("%d 0 R", id)
	}
	w.object(1, "<< /Type /Catalog /Pages 2 0 R >>")
	w.object(2, fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(w.pages), strings.Join(kids, " ")))
	offset := w.out.n
	w.write(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", len(w.offsets)))
	for _, at := range w.offsets[1:] {
		if at >= 10_000_000_000 {
			return fmt.Errorf("PDF exceeds classic xref size limit")
		}
		w.write(fmt.Sprintf("%010d 00000 n \n", at))
	}
	w.write(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(w.offsets), offset))
	return w.out.err
}
