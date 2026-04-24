package imaging

import (
	"image"

	"github.com/buckket/go-blurhash"
)

// Blurhash returns a blurhash string for img. Downscales to max 64px for speed.
// Uses 4 x-components and 3 y-components which is a common tradeoff between
// placeholder fidelity and hash length.
func Blurhash(img image.Image) (string, error) {
	small := ResizeMax(img, 64)
	return blurhash.Encode(4, 3, small)
}
