package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

// makeTestImage builds an in-memory image filled with a solid color. Tests use
// concrete pixel buffers so the decode/encode round-trips actually run real
// image bytes rather than stub interfaces.
func makeTestImage(t *testing.T, w, h int, c color.RGBA) image.Image {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
	return buf.Bytes()
}

func TestFormatFromMime(t *testing.T) {
	cases := []struct {
		mime string
		want string
	}{
		{"image/jpeg", "jpeg"},
		{"image/png", "png"},
		{"image/gif", "gif"},
		{"image/webp", "webp"},
		{"image/svg+xml", ""},
		{"application/octet-stream", ""},
		{"text/html", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := FormatFromMime(c.mime); got != c.want {
			t.Errorf("FormatFromMime(%q) = %q, want %q", c.mime, got, c.want)
		}
	}
}

func TestExtFromFormat(t *testing.T) {
	cases := []struct {
		format, want string
	}{
		{"jpeg", "jpg"},
		{"png", "png"},
		{"gif", "gif"},
		{"webp", "webp"},
		{"unknown", "unknown"}, // pass-through for non-canonical names
		{"", ""},
	}
	for _, c := range cases {
		if got := ExtFromFormat(c.format); got != c.want {
			t.Errorf("ExtFromFormat(%q) = %q, want %q", c.format, got, c.want)
		}
	}
}

func TestSniffDetectsPNG(t *testing.T) {
	img := makeTestImage(t, 4, 4, color.RGBA{R: 255})
	data := encodePNG(t, img)
	mime := Sniff(data)
	if mime != "image/png" {
		t.Errorf("PNG sniff: got %q, want image/png", mime)
	}
}

func TestSniffHandlesTrunctedBuffer(t *testing.T) {
	// Anything under 512 bytes still works; Sniff just shouldn't panic.
	short := []byte{0x89, 0x50, 0x4E, 0x47}
	got := Sniff(short)
	if got == "" {
		t.Error("Sniff on tiny buffer should still return a content-type guess")
	}
}

func TestDecodeRoundTripsPNG(t *testing.T) {
	original := makeTestImage(t, 6, 8, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	data := encodePNG(t, original)
	img, format, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if format != "png" {
		t.Errorf("format: got %q, want png", format)
	}
	if w, h := Dimensions(img); w != 6 || h != 8 {
		t.Errorf("dimensions after decode: got %dx%d, want 6x8", w, h)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	_, _, err := Decode(bytes.NewReader([]byte("not an image")))
	if err == nil {
		t.Error("decoding garbage should return an error, not panic")
	}
}

func TestDimensions(t *testing.T) {
	img := makeTestImage(t, 100, 50, color.RGBA{})
	w, h := Dimensions(img)
	if w != 100 || h != 50 {
		t.Errorf("got %dx%d, want 100x50", w, h)
	}
}

func TestResize_DownscalesProportionally(t *testing.T) {
	img := makeTestImage(t, 1600, 900, color.RGBA{A: 255})
	resized := Resize(img, 800)
	w, h := Dimensions(resized)
	if w != 800 {
		t.Errorf("width: got %d, want 800", w)
	}
	if h != 450 {
		t.Errorf("height: got %d, want 450 (aspect-preserving)", h)
	}
}

func TestResize_NoUpscale(t *testing.T) {
	img := makeTestImage(t, 400, 300, color.RGBA{})
	out := Resize(img, 800)
	w, h := Dimensions(out)
	if w != 400 || h != 300 {
		t.Errorf("Resize should not upscale; got %dx%d, want 400x300", w, h)
	}
}

func TestResize_ClampsHeightTo1(t *testing.T) {
	// Extreme aspect ratio: 10000x1 image -> resize to width=1 must not
	// produce height=0 (would crash later stages).
	img := makeTestImage(t, 10000, 1, color.RGBA{})
	out := Resize(img, 1)
	w, h := Dimensions(out)
	if w != 1 || h < 1 {
		t.Errorf("extreme aspect: got %dx%d; height must be >= 1", w, h)
	}
}

func TestResizeMax_LongestSideLimit(t *testing.T) {
	// Wide image: longest side = width
	img := makeTestImage(t, 2000, 1000, color.RGBA{})
	out := ResizeMax(img, 500)
	w, h := Dimensions(out)
	if w != 500 || h != 250 {
		t.Errorf("wide: got %dx%d, want 500x250", w, h)
	}

	// Tall image: longest side = height
	tall := makeTestImage(t, 600, 1200, color.RGBA{})
	out2 := ResizeMax(tall, 400)
	w2, h2 := Dimensions(out2)
	if h2 != 400 || w2 != 200 {
		t.Errorf("tall: got %dx%d, want 200x400", w2, h2)
	}
}

func TestResizeMax_NoUpscale(t *testing.T) {
	img := makeTestImage(t, 50, 30, color.RGBA{})
	out := ResizeMax(img, 200)
	w, h := Dimensions(out)
	if w != 50 || h != 30 {
		t.Errorf("ResizeMax should not upscale; got %dx%d, want 50x30", w, h)
	}
}

func TestBlurhash_Roundtrip(t *testing.T) {
	img := makeTestImage(t, 32, 32, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	hash, err := Blurhash(img)
	if err != nil {
		t.Fatalf("blurhash: %v", err)
	}
	// Standard blurhash strings start with "L" or similar single-char
	// component prefix; they're short, base83-encoded, and never empty.
	if len(hash) < 6 {
		t.Errorf("blurhash unexpectedly short: %q", hash)
	}
}

func TestBlurhash_DifferentColorsProduceDifferentHashes(t *testing.T) {
	red := makeTestImage(t, 16, 16, color.RGBA{R: 255, A: 255})
	blue := makeTestImage(t, 16, 16, color.RGBA{B: 255, A: 255})
	hr, _ := Blurhash(red)
	hb, _ := Blurhash(blue)
	if hr == hb {
		t.Errorf("red and blue should produce different blurhashes; both got %q", hr)
	}
}

func TestEncodeJPEG_ValidOutput(t *testing.T) {
	img := makeTestImage(t, 32, 32, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	var buf bytes.Buffer
	if err := EncodeJPEG(&buf, img, 0); err != nil {
		// Quality=0 should use the default per the contract.
		t.Fatalf("encode: %v", err)
	}
	if !strings.HasPrefix(string(buf.Bytes()[:3]), "\xff\xd8\xff") {
		t.Error("encoded JPEG missing SOI magic")
	}
}

func TestEncodePNG_ValidOutput(t *testing.T) {
	img := makeTestImage(t, 16, 16, color.RGBA{A: 255})
	var buf bytes.Buffer
	if err := EncodePNG(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte{0x89, 0x50, 0x4E, 0x47}) {
		t.Error("encoded PNG missing magic header")
	}
}

func TestEncodeWebP_ValidOutput(t *testing.T) {
	img := makeTestImage(t, 16, 16, color.RGBA{R: 50, G: 100, B: 150, A: 255})
	var buf bytes.Buffer
	if err := EncodeWebP(&buf, img, 0); err != nil {
		t.Fatalf("encode: %v", err)
	}
	// WebP files start with "RIFF" at offset 0, "WEBP" at offset 8.
	if !bytes.HasPrefix(buf.Bytes(), []byte("RIFF")) {
		t.Error("encoded WebP missing RIFF header")
	}
	if len(buf.Bytes()) < 12 || string(buf.Bytes()[8:12]) != "WEBP" {
		t.Error("encoded WebP missing WEBP signature at offset 8")
	}
}

func TestReadAll_RespectsMaxCap(t *testing.T) {
	// Within limit
	r := bytes.NewReader([]byte("abc"))
	out, err := ReadAll(r, 100)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(out) != "abc" {
		t.Errorf("got %q", out)
	}

	// Exactly at limit
	r = bytes.NewReader([]byte("abc"))
	out, err = ReadAll(r, 3)
	if err != nil {
		t.Fatalf("at-limit should pass; got %v", err)
	}
	if string(out) != "abc" {
		t.Errorf("got %q", out)
	}

	// One over limit must error
	r = bytes.NewReader([]byte("abcd"))
	_, err = ReadAll(r, 3)
	if err == nil {
		t.Error("over-limit should error")
	}
}
