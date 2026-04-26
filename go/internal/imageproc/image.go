package imageproc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotSupported is returned by operations that have no real implementation.
var ErrNotSupported = errors.New("imageproc: not supported in this build")

// Processor is the image-analysis surface used by tools.
type Processor interface {
	ExtractText(ctx context.Context, imagePath string) (string, error)
	Describe(ctx context.Context, imagePath string) (string, error)
	Resize(ctx context.Context, imagePath string, maxDim int) ([]byte, error)
}

type processor struct{}

// New returns a Processor backed by tesseract for OCR and the standard
// library for resizing.
func New() Processor { return processor{} }

func (processor) ExtractText(ctx context.Context, imagePath string) (string, error) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", errors.New("imageproc: tesseract not found in PATH")
	}
	cmd := exec.CommandContext(ctx, "tesseract", imagePath, "-", "-l", "eng")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("imageproc: tesseract: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func (processor) Describe(ctx context.Context, imagePath string) (string, error) {
	return "image at " + imagePath + " (decoder not configured; use ExtractText for OCR)", nil
}

func (processor) Resize(ctx context.Context, imagePath string, maxDim int) ([]byte, error) {
	if maxDim <= 0 {
		return nil, errors.New("imageproc: maxDim must be > 0")
	}
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("imageproc: open: %w", err)
	}
	defer f.Close()

	src, format, err := image.Decode(f)
	if err != nil {
		// Fall back to format-specific decoders so we can handle PNG/JPEG
		// even when image.Decode hasn't been seeded by an underscore import.
		if _, seekErr := f.Seek(0, 0); seekErr != nil {
			return nil, fmt.Errorf("imageproc: decode: %w", err)
		}
		ext := strings.ToLower(filepath.Ext(imagePath))
		switch ext {
		case ".png":
			src, err = png.Decode(f)
			format = "png"
		case ".jpg", ".jpeg":
			src, err = jpeg.Decode(f)
			format = "jpeg"
		default:
			return nil, fmt.Errorf("imageproc: decode: %w", err)
		}
		if err != nil {
			return nil, fmt.Errorf("imageproc: decode: %w", err)
		}
	}
	_ = format

	b := src.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW == 0 || srcH == 0 {
		return nil, errors.New("imageproc: empty image")
	}
	dstW, dstH := srcW, srcH
	if srcW > maxDim || srcH > maxDim {
		if srcW >= srcH {
			dstW = maxDim
			dstH = srcH * maxDim / srcW
		} else {
			dstH = maxDim
			dstW = srcW * maxDim / srcH
		}
		if dstW < 1 {
			dstW = 1
		}
		if dstH < 1 {
			dstH = 1
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		sy := y * srcH / dstH
		for x := 0; x < dstW; x++ {
			sx := x * srcW / dstW
			r, g, bl, a := src.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
			dst.Set(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(bl >> 8),
				A: uint8(a >> 8),
			})
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, fmt.Errorf("imageproc: encode: %w", err)
	}
	return out.Bytes(), nil
}
