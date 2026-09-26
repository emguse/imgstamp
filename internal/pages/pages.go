// Package pages decodes images and walks TIFF IFDs without retaining all pages.
package pages

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/emguse/imgstamp/internal/config"
	"github.com/hhrutter/tiff"
)

// Source holds an input open while advancing through its pages.
type Source struct {
	f      *os.File
	isTIFF bool
	order  binary.ByteOrder
	next   uint32
	seen   map[uint32]bool
	done   bool
}

// Supported reports whether a filename has a supported image extension.
func Supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tif", ".tiff", ".png", ".jpg", ".jpeg":
		return true
	}
	return false
}

// Open verifies the input signature and initializes page traversal.
func Open(path string) (*Source, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	s := &Source{f: f, seen: map[uint32]bool{}}
	var h [8]byte
	if _, err = f.ReadAt(h[:], 0); err != nil {
		f.Close()
		return nil, err
	}
	if string(h[:2]) == "II" || string(h[:2]) == "MM" {
		s.isTIFF = true
		s.order = binary.LittleEndian
		if string(h[:2]) == "MM" {
			s.order = binary.BigEndian
		}
		if s.order.Uint16(h[2:4]) != 42 {
			f.Close()
			return nil, fmt.Errorf("unsupported TIFF header (BigTIFF is not supported)")
		}
		s.next = s.order.Uint32(h[4:8])
		if s.next == 0 {
			f.Close()
			return nil, fmt.Errorf("TIFF has no pages")
		}
	} else if !bytes.Equal(h[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) && !(h[0] == 0xff && h[1] == 0xd8) {
		f.Close()
		return nil, fmt.Errorf("unsupported image signature")
	}
	return s, nil
}

// Close releases the underlying input file.
func (s *Source) Close() error { return s.f.Close() }

// Next decodes and orients one page; io.EOF denotes only normal sequence end.
func (s *Source) Next(ctx context.Context) (img image.Image, err error) {
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if s.done || (s.isTIFF && s.next == 0) {
		return nil, io.EOF
	}
	// Third-party decoders must not prevent cleaning up a partially written PDF.
	defer func() {
		if r := recover(); r != nil {
			img = nil
			err = fmt.Errorf("image decoder panic: %v", r)
		}
		// EOF is reserved for the normal end of the page sequence. A decoder
		// hitting EOF inside a page must invalidate the whole output file.
		if errors.Is(err, io.EOF) {
			err = fmt.Errorf("truncated image page: %w", io.ErrUnexpectedEOF)
		}
	}()
	orientation := 1
	if s.isTIFF {
		offset := s.next
		if s.seen[offset] {
			return nil, fmt.Errorf("cyclic TIFF page directory")
		}
		s.seen[offset] = true
		meta, next, err := readIFD(s.f, s.order, offset)
		if err != nil {
			return nil, err
		}
		orientation, s.next = meta.orientation, next
		dim, err := tiff.DecodeConfigAt(s.f, int64(offset))
		if err != nil {
			return nil, err
		}
		if !config.PixelsOK(dim.Width, dim.Height) {
			return nil, fmt.Errorf("page dimensions %dx%d exceed %d-pixel limit", dim.Width, dim.Height, config.MaxPixels)
		}
		img, err = tiff.DecodeAt(s.f, int64(offset))
		if err != nil {
			return nil, err
		}
		// hhrutter/tiff v1.0.6 decodes CCITT run colors independently of
		// PhotometricInterpretation. Restore BlackIsZero semantics here;
		// fixtures from an independent encoder cover both tag values.
		if (meta.compression == 3 || meta.compression == 4) && meta.photometric == 1 {
			gray, ok := img.(*image.Gray)
			if !ok {
				return nil, fmt.Errorf("unexpected CCITT pixel format")
			}
			for y := 0; y < gray.Bounds().Dy(); y++ {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				row := gray.Pix[y*gray.Stride : y*gray.Stride+gray.Bounds().Dx()]
				for x := range row {
					row[x] = 255 - row[x]
				}
			}
		}
	} else {
		s.done = true
		var err error
		orientation, err = metadataOrientation(s.f)
		if err != nil {
			return nil, err
		}
		if _, err = s.f.Seek(0, 0); err != nil {
			return nil, err
		}
		dim, _, err := image.DecodeConfig(s.f)
		if err != nil {
			return nil, err
		}
		if !config.PixelsOK(dim.Width, dim.Height) {
			return nil, fmt.Errorf("page dimensions %dx%d exceed %d-pixel limit", dim.Width, dim.Height, config.MaxPixels)
		}
		if _, err = s.f.Seek(0, 0); err != nil {
			return nil, err
		}
		img, _, err = image.Decode(s.f)
		if err != nil {
			return nil, err
		}
	}
	return orient(ctx, img, orientation)
}

type directory struct{ orientation, compression, photometric int }

// readIFD extracts scalar metadata and the next main IFD offset without pixels.
func readIFD(r io.ReaderAt, order binary.ByteOrder, offset uint32) (directory, uint32, error) {
	meta := directory{orientation: 1}
	var count [2]byte
	if _, err := r.ReadAt(count[:], int64(offset)); err != nil {
		return meta, 0, err
	}
	n := int(order.Uint16(count[:]))
	entries := make([]byte, n*12+4)
	if _, err := r.ReadAt(entries, int64(offset)+2); err != nil {
		return meta, 0, err
	}
	for i := 0; i < n; i++ {
		e := entries[i*12 : (i+1)*12]
		tag := order.Uint16(e[:2])
		if tag != 274 && tag != 259 && tag != 262 {
			continue
		}
		if order.Uint16(e[2:4]) != 3 || order.Uint32(e[4:8]) != 1 {
			return meta, 0, fmt.Errorf("invalid scalar TIFF tag %d", tag)
		}
		value := int(order.Uint16(e[8:10]))
		switch tag {
		case 274:
			if value < 1 || value > 8 {
				return meta, 0, fmt.Errorf("invalid orientation %d", value)
			}
			meta.orientation = value
		case 259:
			meta.compression = value
		case 262:
			meta.photometric = value
		}
	}
	return meta, order.Uint32(entries[n*12:]), nil
}

func exifOrientation(data []byte) (int, error) {
	if bytes.HasPrefix(data, []byte("Exif\x00\x00")) {
		data = data[6:]
	}
	if len(data) < 8 {
		return 0, fmt.Errorf("truncated EXIF")
	}
	var order binary.ByteOrder
	switch string(data[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0, fmt.Errorf("invalid EXIF byte order")
	}
	if order.Uint16(data[2:4]) != 42 {
		return 0, fmt.Errorf("invalid EXIF header")
	}
	o, _, err := readIFD(bytes.NewReader(data), order, order.Uint32(data[4:8]))
	return o.orientation, err
}

func metadataOrientation(f *os.File) (int, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return 0, err
	}
	var h [8]byte
	if _, err := io.ReadFull(f, h[:]); err != nil {
		return 0, err
	}
	if h[0] == 0xff && h[1] == 0xd8 {
		if _, err := f.Seek(2, 0); err != nil {
			return 0, err
		}
		for {
			var marker [2]byte
			if _, err := io.ReadFull(f, marker[:]); err != nil {
				return 0, err
			}
			if marker[0] != 0xff {
				return 0, fmt.Errorf("invalid JPEG marker")
			}
			for marker[1] == 0xff {
				if _, err := io.ReadFull(f, marker[1:]); err != nil {
					return 0, err
				}
			}
			if marker[1] == 0xda || marker[1] == 0xd9 {
				return 1, nil
			}
			if marker[1] == 0x01 || marker[1] >= 0xd0 && marker[1] <= 0xd7 {
				continue
			}
			var l [2]byte
			if _, err := io.ReadFull(f, l[:]); err != nil {
				return 0, err
			}
			n := int(binary.BigEndian.Uint16(l[:])) - 2
			if n < 0 {
				return 0, fmt.Errorf("invalid JPEG segment length")
			}
			if marker[1] == 0xe1 {
				data := make([]byte, n)
				if _, err := io.ReadFull(f, data); err != nil {
					return 0, err
				}
				if bytes.HasPrefix(data, []byte("Exif\x00\x00")) {
					return exifOrientation(data)
				}
			} else {
				if _, err := f.Seek(int64(n), io.SeekCurrent); err != nil {
					return 0, err
				}
			}
		}
	}
	// Locate PNG eXIf metadata by seeking over pixel chunks, without decoding them.
	for {
		var chunk [8]byte
		if _, err := io.ReadFull(f, chunk[:]); err != nil {
			return 0, err
		}
		n := binary.BigEndian.Uint32(chunk[:4])
		kind := string(chunk[4:])
		if kind == "IEND" {
			return 1, nil
		}
		if kind == "eXIf" {
			if n > 1<<20 {
				return 0, fmt.Errorf("EXIF metadata exceeds 1 MiB")
			}
			data := make([]byte, int(n))
			if _, err := io.ReadFull(f, data); err != nil {
				return 0, err
			}
			return exifOrientation(data)
		}
		if _, err := f.Seek(int64(n)+4, io.SeekCurrent); err != nil {
			return 0, err
		}
	}
}

func orient(ctx context.Context, src image.Image, o int) (image.Image, error) {
	if o == 1 {
		return src, ctx.Err()
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for x := 0; x < w; x++ {
			dx, dy := x, y
			switch o {
			case 2:
				dx = w - 1 - x
			case 3:
				dx = w - 1 - x
				dy = h - 1 - y
			case 4:
				dy = h - 1 - y
			case 5:
				dx = y
				dy = x
			case 6:
				dx = h - 1 - y
				dy = x
			case 7:
				dx = h - 1 - y
				dy = w - 1 - x
			case 8:
				dx = y
				dy = w - 1 - x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst, nil
}
