// Package imaging provides image decode, resize, encode, and blurhash operations.
// Pure functions, no HTTP or DB dependencies. Reusable from handlers and the build pipeline.
package imaging

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"

	_ "golang.org/x/image/webp"
)

// maxImagePixels caps the decoded pixel count. A small compressed file can
// declare enormous dimensions (a decompression bomb: a few-KB PNG that decodes
// to 50000x50000 = gigabytes of RAM). 40MP comfortably covers pro-camera photos
// (~24MP) while refusing to allocate for a bomb. image.DecodeConfig reads only
// the header, so the check is cheap and happens before any pixel allocation.
const maxImagePixels = 40_000_000

// maxImageBytes bounds the in-memory buffer Decode reads. Sits above the
// default 20MB upload cap so it only ever fires on pathological standalone use.
const maxImageBytes = 64 << 20

// Sniff detects the MIME type from the first 512 bytes using http.DetectContentType.
// Trust this over client-supplied Content-Type headers.
func Sniff(buf []byte) string {
	if len(buf) > 512 {
		buf = buf[:512]
	}
	return http.DetectContentType(buf)
}

// FormatFromMime maps a MIME type to an image format string ("jpeg", "png", "gif", "webp").
// Returns empty string for non-image or unsupported formats.
func FormatFromMime(mime string) string {
	switch mime {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	}
	return ""
}

// ExtFromFormat returns the canonical file extension for a format.
func ExtFromFormat(format string) string {
	switch format {
	case "jpeg":
		return "jpg"
	case "png":
		return "png"
	case "gif":
		return "gif"
	case "webp":
		return "webp"
	}
	return format
}

// Decode reads an image from r. Returns the decoded image and the format string.
// Supports jpeg, png, gif, webp (decode only; encode separately).
//
// Decompression-bomb guard: the input is buffered (byte-capped), the header is
// read with DecodeConfig to learn the declared dimensions, and decode is
// refused when width*height exceeds maxImagePixels, BEFORE image.Decode
// allocates the pixel buffer.
func Decode(r io.Reader) (image.Image, string, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxImageBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("read image: %w", err)
	}
	if int64(len(data)) > maxImageBytes {
		return nil, "", fmt.Errorf("image exceeds %d byte cap", maxImageBytes)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image config: %w", err)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return nil, "", fmt.Errorf("image too large: %dx%d exceeds %d-pixel cap", cfg.Width, cfg.Height, maxImagePixels)
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}
	return img, format, nil
}

// Dimensions returns the width and height of an image.
func Dimensions(img image.Image) (int, int) {
	b := img.Bounds()
	return b.Dx(), b.Dy()
}
