package imaging

import (
	"image"

	"golang.org/x/image/draw"
)

// Resize scales img to targetWidth, preserving aspect ratio. Returns src unchanged
// if its width is already <= targetWidth (no upscaling). Uses Catmull-Rom for
// high-quality downscaling.
func Resize(src image.Image, targetWidth int) image.Image {
	srcW, srcH := Dimensions(src)
	if srcW <= targetWidth {
		return src
	}
	targetH := int(float64(srcH) * float64(targetWidth) / float64(srcW))
	if targetH < 1 {
		targetH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// ResizeMax scales down so that the longest side is at most maxSide. Used for
// blurhash generation where a tiny version is sufficient.
func ResizeMax(src image.Image, maxSide int) image.Image {
	srcW, srcH := Dimensions(src)
	if srcW <= maxSide && srcH <= maxSide {
		return src
	}
	var targetW, targetH int
	if srcW >= srcH {
		targetW = maxSide
		targetH = int(float64(srcH) * float64(maxSide) / float64(srcW))
	} else {
		targetH = maxSide
		targetW = int(float64(srcW) * float64(maxSide) / float64(srcH))
	}
	if targetH < 1 {
		targetH = 1
	}
	if targetW < 1 {
		targetW = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}
