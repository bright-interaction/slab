package imaging

import (
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/HugoSmits86/nativewebp"
)

// Default encoding qualities.
const (
	DefaultJPEGQuality = 82
	DefaultWebPQuality = 80
)

// EncodeJPEG writes a JPEG. Strips EXIF by re-encoding.
func EncodeJPEG(w io.Writer, img image.Image, quality int) error {
	if quality <= 0 {
		quality = DefaultJPEGQuality
	}
	return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
}

// EncodePNG writes a PNG with best compression.
func EncodePNG(w io.Writer, img image.Image) error {
	enc := &png.Encoder{CompressionLevel: png.BestCompression}
	return enc.Encode(w, img)
}

// EncodeWebP writes a WebP using the pure-Go nativewebp encoder (no CGO).
// nativewebp v1.2 is lossless-only; the quality param is accepted for API parity
// but not applied.
func EncodeWebP(w io.Writer, img image.Image, quality int) error {
	_ = quality
	return nativewebp.Encode(w, img, &nativewebp.Options{UseExtendedFormat: false})
}
