package spotself

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestFingerprintSimilarity(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 16), B: 80, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	first, err := fingerprint(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	second, err := fingerprint(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if got := similarity(first, second); got != 1 {
		t.Fatalf("similarity = %v, want 1", got)
	}
}
