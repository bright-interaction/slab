// Package imaging provides image decode, resize, encode, and blurhash operations.
// Pure functions, no HTTP or DB dependencies. Reusable from handlers and the build pipeline.
package imaging

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"

	_ "golang.org/x/image/webp"
)

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
func Decode(r io.Reader) (image.Image, string, error) {
	img, format, err := image.Decode(r)
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
