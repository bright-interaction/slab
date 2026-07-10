package imaging

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"path/filepath"

	"github.com/bright-interaction/atomicsite/internal/storage"
)

// Variant describes one processed image file (original-format or WebP at a given width).
type Variant struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Format   string `json:"format"`
	Path     string `json:"path"`
	FileSize int64  `json:"file_size"`
}

// ProcessResult is the output of Process: original dimensions, blurhash placeholder,
// and all variants (original + resized JPEG/PNG + WebP at each target width).
type ProcessResult struct {
	OriginalWidth  int
	OriginalHeight int
	Blurhash       string
	Variants       []Variant
}

// Process takes a raw image byte buffer, decodes it, generates a blurhash, resizes
// to each target width, encodes both the source format and WebP, and stores them
// via the Store. Returns a ProcessResult containing paths + dimensions.
//
// Paths are stored as "{siteID}/{mediaID}/{width}.{ext}" (plus an "original.{ext}").
//
// Supports jpeg, png, gif, webp inputs. For gif, variants are encoded as jpeg
// (only the first frame). For webp input, source-format variants are also webp.
func Process(ctx context.Context, srcBytes []byte, store storage.Store, siteID, mediaID, format string, widths []int) (*ProcessResult, error) {
	img, _, err := Decode(bytes.NewReader(srcBytes))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	origW, origH := Dimensions(img)

	bh, err := Blurhash(img)
	if err != nil {
		return nil, fmt.Errorf("blurhash: %w", err)
	}

	result := &ProcessResult{
		OriginalWidth:  origW,
		OriginalHeight: origH,
		Blurhash:       bh,
	}

	// Source format for re-encoded variants. GIF collapses to JPEG (first frame only).
	srcEncodeFormat := format
	if format == "gif" {
		srcEncodeFormat = "jpeg"
	}

	// 1. Write the re-encoded original (EXIF stripped) at "original.{ext}"
	origExt := ExtFromFormat(srcEncodeFormat)
	origKey := filepath.Join(siteID, mediaID, "original."+origExt)
	origSize, err := encodeAndStore(ctx, store, origKey, img, srcEncodeFormat)
	if err != nil {
		return nil, fmt.Errorf("store original: %w", err)
	}
	result.Variants = append(result.Variants, Variant{
		Width:    origW,
		Height:   origH,
		Format:   srcEncodeFormat,
		Path:     origKey,
		FileSize: origSize,
	})

	// 2. Resized variants in source format + WebP
	for _, w := range widths {
		if w >= origW {
			continue // no upscaling
		}
		resized := Resize(img, w)
		rW, rH := Dimensions(resized)

		// Source format variant
		srcKey := filepath.Join(siteID, mediaID, fmt.Sprintf("%d.%s", w, origExt))
		srcSize, err := encodeAndStore(ctx, store, srcKey, resized, srcEncodeFormat)
		if err != nil {
			return nil, fmt.Errorf("store %d.%s: %w", w, origExt, err)
		}
		result.Variants = append(result.Variants, Variant{
			Width:    rW,
			Height:   rH,
			Format:   srcEncodeFormat,
			Path:     srcKey,
			FileSize: srcSize,
		})

		// WebP variant (skip if source is already webp, since we'd encode twice)
		if srcEncodeFormat != "webp" {
			webpKey := filepath.Join(siteID, mediaID, fmt.Sprintf("%d.webp", w))
			webpSize, err := encodeAndStore(ctx, store, webpKey, resized, "webp")
			if err != nil {
				return nil, fmt.Errorf("store %d.webp: %w", w, err)
			}
			result.Variants = append(result.Variants, Variant{
				Width:    rW,
				Height:   rH,
				Format:   "webp",
				Path:     webpKey,
				FileSize: webpSize,
			})
		}
	}

	return result, nil
}

// encodeAndStore encodes img to the given format and stores it via store. Returns file size.
func encodeAndStore(ctx context.Context, store storage.Store, key string, img image.Image, format string) (int64, error) {
	var buf bytes.Buffer
	var err error
	switch format {
	case "jpeg":
		err = EncodeJPEG(&buf, img, DefaultJPEGQuality)
	case "png":
		err = EncodePNG(&buf, img)
	case "webp":
		err = EncodeWebP(&buf, img, DefaultWebPQuality)
	default:
		return 0, fmt.Errorf("unsupported encode format: %s", format)
	}
	if err != nil {
		return 0, err
	}
	size := int64(buf.Len())
	if _, err := store.Put(ctx, key, &buf, mimeForFormat(format)); err != nil {
		return 0, err
	}
	return size, nil
}

func mimeForFormat(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "gif":
		return "image/gif"
	}
	return "application/octet-stream"
}

// ReadAll drains r into memory with a hard size cap. Used before Process to
// bound memory and enforce upload limits.
func ReadAll(r io.Reader, max int64) ([]byte, error) {
	limited := io.LimitReader(r, max+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(buf)) > max {
		return nil, fmt.Errorf("payload exceeds max size of %d bytes", max)
	}
	return buf, nil
}
