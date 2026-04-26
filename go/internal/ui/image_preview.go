package ui

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
)

// shadeRamp maps luminance buckets to Unicode block characters from darkest
// to brightest. The empty space sits at index 0; full block at the end.
var shadeRamp = []rune{' ', '░', '▒', '▓', '█'}

// renderImagePreview decodes the file at path as an image (PNG/JPEG/GIF via
// stdlib) and returns a multiline ASCII shade rendering scaled to fit
// maxWidth columns. Each output cell represents one source rectangle whose
// height is twice its width to compensate for terminal cell aspect.
//
// On decode failure the returned string is the source path and the error is
// non-nil so callers can decide whether to surface a fallback view.
func renderImagePreview(path string, maxWidth int) (string, error) {
	if maxWidth <= 0 {
		maxWidth = 64
	}

	f, err := os.Open(path)
	if err != nil {
		return path, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return path, err
	}

	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return path, nil
	}

	cols := maxWidth
	if cols > srcW {
		cols = srcW
	}
	if cols < 1 {
		cols = 1
	}

	// Each character cell maps to ~2x source rows (rows are roughly twice as
	// tall as columns in monospaced terminals).
	cellW := float64(srcW) / float64(cols)
	cellH := cellW * 2.0
	rows := int(float64(srcH) / cellH)
	if rows < 1 {
		rows = 1
	}

	var b strings.Builder
	b.Grow((cols + 1) * rows)

	for r := 0; r < rows; r++ {
		y0 := bounds.Min.Y + int(float64(r)*cellH)
		y1 := bounds.Min.Y + int(float64(r+1)*cellH)
		if y1 > bounds.Max.Y {
			y1 = bounds.Max.Y
		}
		if y0 >= y1 {
			y0 = y1 - 1
			if y0 < bounds.Min.Y {
				y0 = bounds.Min.Y
			}
		}
		for c := 0; c < cols; c++ {
			x0 := bounds.Min.X + int(float64(c)*cellW)
			x1 := bounds.Min.X + int(float64(c+1)*cellW)
			if x1 > bounds.Max.X {
				x1 = bounds.Max.X
			}
			if x0 >= x1 {
				x0 = x1 - 1
				if x0 < bounds.Min.X {
					x0 = bounds.Min.X
				}
			}
			lum := averageLuminance(img, x0, y0, x1, y1)
			b.WriteRune(shadeFor(lum))
		}
		if r < rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

// averageLuminance returns the mean ITU-R BT.601 luminance of the rectangle
// [x0,y0)-(x1,y1) in the 0..1 range.
func averageLuminance(img image.Image, x0, y0, x1, y1 int) float64 {
	var sum, n uint64
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// RGBA returns 16-bit channels (0..65535). Compute luma in that
			// space then normalize to 0..1 once at the end.
			lum := uint64(r)*299 + uint64(g)*587 + uint64(b)*114
			sum += lum / 1000
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return float64(sum/n) / 65535.0
}

func shadeFor(lum float64) rune {
	if lum <= 0 {
		return shadeRamp[0]
	}
	if lum >= 1 {
		return shadeRamp[len(shadeRamp)-1]
	}
	idx := int(lum * float64(len(shadeRamp)))
	if idx >= len(shadeRamp) {
		idx = len(shadeRamp) - 1
	}
	return shadeRamp[idx]
}
