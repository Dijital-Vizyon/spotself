package spotself

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math/bits"
)

func fingerprint(r io.Reader) (uint64, error) {
	return fingerprintLimited(r, 0)
}

func fingerprintLimited(r io.Reader, maxPixels int) (uint64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, fmt.Errorf("read image: %w", err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("decode image config: %w", err)
	}
	if maxPixels > 0 && cfg.Width*cfg.Height > maxPixels {
		return 0, fmt.Errorf("image is too large: %dx%d exceeds %d pixels", cfg.Width, cfg.Height, maxPixels)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	if bounds.Empty() {
		return 0, fmt.Errorf("decode image: empty image")
	}

	var samples [64]uint8
	var total uint32
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			px := bounds.Min.X + (x*bounds.Dx()+bounds.Dx()/2)/8
			py := bounds.Min.Y + (y*bounds.Dy()+bounds.Dy()/2)/8
			gray := luminance(img.At(px, py))
			samples[y*8+x] = gray
			total += uint32(gray)
		}
	}

	avg := uint8(total / 64)
	var hash uint64
	for i, gray := range samples {
		if gray >= avg {
			hash |= 1 << uint(i)
		}
	}
	return hash, nil
}

func similarity(a, b uint64) float64 {
	distance := bits.OnesCount64(a ^ b)
	return 1 - float64(distance)/64
}

func luminance(c color.Color) uint8 {
	r, g, b, _ := c.RGBA()
	y := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
	return uint8(y)
}
