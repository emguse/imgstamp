package pages

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hhrutter/tiff"
)

func TestMultipageCompressions(t *testing.T) {
	for _, compression := range []tiff.CompressionType{tiff.Uncompressed, tiff.Deflate, tiff.LZW} {
		t.Run(string(rune('0'+compression)), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pages.tif")
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			images := []image.Image{image.NewGray(image.Rect(0, 0, 20, 30)), image.NewGray(image.Rect(0, 0, 30, 20))}
			if err := tiff.EncodeAll(f, images, &tiff.Options{Compression: compression}); err != nil {
				t.Fatal(err)
			}
			f.Close()
			source, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer source.Close()
			for _, want := range images {
				got, err := source.Next(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if got.Bounds() != want.Bounds() {
					t.Fatalf("bounds=%v", got.Bounds())
				}
			}
			if _, err := source.Next(context.Background()); !errors.Is(err, io.EOF) {
				t.Fatalf("expected EOF: %v", err)
			}
		})
	}
}
func TestCCITT(t *testing.T) {
	f, err := os.Open("testdata/bw-gopher.png")
	if err != nil {
		t.Fatal(err)
	}
	want, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"testdata/bw-gopher_ccittGroup3.tiff", "testdata/bw-gopher_ccittGroup4.tiff"} {
		t.Run(path, func(t *testing.T) {
			s, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			got, err := s.Next(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if got.Bounds() != want.Bounds() {
				t.Fatal("dimensions differ")
			}
			for y := 0; y < want.Bounds().Dy(); y++ {
				for x := 0; x < want.Bounds().Dx(); x++ {
					if color.GrayModel.Convert(got.At(x, y)) != color.GrayModel.Convert(want.At(x, y)) {
						t.Fatalf("different pixel at %d,%d", x, y)
					}
				}
			}
		})
	}
}
func TestOrientations(t *testing.T) {
	src := image.NewGray(image.Rect(0, 0, 2, 3))
	for i := 0; i < 6; i++ {
		src.Pix[i] = uint8(i + 1)
	}
	wants := [][]byte{{1, 2, 3, 4, 5, 6}, {2, 1, 4, 3, 6, 5}, {6, 5, 4, 3, 2, 1}, {5, 6, 3, 4, 1, 2}, {1, 3, 5, 2, 4, 6}, {5, 3, 1, 6, 4, 2}, {6, 4, 2, 5, 3, 1}, {2, 4, 6, 1, 3, 5}}
	for o, want := range wants {
		got, err := orient(context.Background(), src, o+1)
		if err != nil {
			t.Fatal(err)
		}
		var values []byte
		for y := 0; y < got.Bounds().Dy(); y++ {
			for x := 0; x < got.Bounds().Dx(); x++ {
				values = append(values, color.GrayModel.Convert(got.At(x, y)).(color.Gray).Y)
			}
		}
		if !bytes.Equal(values, want) {
			t.Fatalf("orientation %d: %v != %v", o+1, values, want)
		}
	}
}
func TestTIFFCycleAndBigTIFF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cycle.tif")
	var b bytes.Buffer
	if err := tiff.Encode(&b, image.NewGray(image.Rect(0, 0, 2, 3)), nil); err != nil {
		t.Fatal(err)
	}
	data := b.Bytes()
	order := binary.LittleEndian
	off := order.Uint32(data[4:8])
	n := order.Uint16(data[off : off+2])
	order.PutUint32(data[int(off)+2+int(n)*12:], off)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.Next(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Next(context.Background()); err == nil {
		t.Fatal("cycle accepted")
	}
	path = filepath.Join(t.TempDir(), "big.tif")
	os.WriteFile(path, []byte{'I', 'I', 43, 0, 8, 0, 0, 0}, 0600)
	if _, err := Open(path); err == nil {
		t.Fatal("BigTIFF accepted")
	}
}
func TestJPEGEXIFOrientation(t *testing.T) {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewGray(image.Rect(0, 0, 20, 30)), nil); err != nil {
		t.Fatal(err)
	}
	exif := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	payload := append([]byte("Exif\x00\x00"), exif...)
	data := []byte{0xff, 0xd8, 0xff, 0xe1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}
	data = append(data, payload...)
	data = append(data, b.Bytes()[2:]...)
	path := filepath.Join(t.TempDir(), "rotated.jpg")
	os.WriteFile(path, data, 0600)
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	img, err := s.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 30 || img.Bounds().Dy() != 20 {
		t.Fatal("EXIF not applied")
	}
}

func TestMissingNextIFDIsNotEndOfSequence(t *testing.T) {
	var b bytes.Buffer
	if err := tiff.Encode(&b, image.NewGray(image.Rect(0, 0, 2, 3)), nil); err != nil {
		t.Fatal(err)
	}
	data := b.Bytes()
	order := binary.LittleEndian
	off := order.Uint32(data[4:8])
	n := order.Uint16(data[off : off+2])
	order.PutUint32(data[int(off)+2+int(n)*12:], uint32(len(data)+100))
	path := filepath.Join(t.TempDir(), "truncated.tif")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.Next(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Next(context.Background()); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected truncated page error, got %v", err)
	}
}

func TestCCITTPhotometricInterpretations(t *testing.T) {
	for _, name := range []string{"synthetic-group3-photo0.tiff", "synthetic-group3-photo1.tiff", "synthetic-group4-photo0.tiff", "synthetic-group4-photo1.tiff"} {
		t.Run(name, func(t *testing.T) {
			s, err := Open(filepath.Join("testdata", name))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			im, err := s.Next(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for y := 0; y < 12; y++ {
				for x := 0; x < 16; x++ {
					want := uint8(255)
					if x >= 2 && x <= 8 && y >= 3 && y <= 7 {
						want = 0
					}
					if got := color.GrayModel.Convert(im.At(x, y)).(color.Gray).Y; got != want {
						t.Fatalf("pixel %d,%d=%d, want %d", x, y, got, want)
					}
				}
			}
		})
	}
}

func TestPNGOrientationAfterPixels(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewGray(image.Rect(0, 0, 20, 30))); err != nil {
		t.Fatal(err)
	}
	exif := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	chunk := make([]byte, 12+len(exif))
	binary.BigEndian.PutUint32(chunk[:4], uint32(len(exif)))
	copy(chunk[4:8], "eXIf")
	copy(chunk[8:], exif)
	binary.BigEndian.PutUint32(chunk[len(chunk)-4:], crc32.ChecksumIEEE(chunk[4:len(chunk)-4]))
	original := b.Bytes()
	data := append([]byte{}, original[:len(original)-12]...)
	data = append(data, chunk...)
	data = append(data, original[len(original)-12:]...)
	path := filepath.Join(t.TempDir(), "rotated.png")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	im, err := s.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if im.Bounds().Dx() != 30 || im.Bounds().Dy() != 20 {
		t.Fatal("PNG orientation ignored")
	}
}
