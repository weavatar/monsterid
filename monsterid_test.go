package monsterid

import (
	"bytes"
	"image/color"
	"image/png"
	"math"
	"testing"
)

func TestNewCreatesImageWithCorrectDimensions(t *testing.T) {
	hash := []byte("test-hash")
	img := New(hash)

	bounds := img.Bounds()
	if bounds.Dx() != 120 || bounds.Dy() != 120 {
		t.Errorf("Expected image dimensions 120x120, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestDifferentHashesProduceDifferentImages(t *testing.T) {
	hash1 := []byte("test-hash-1")
	hash2 := []byte("test-hash-2")

	img1 := New(hash1)
	img2 := New(hash2)

	buf1 := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)

	if err := png.Encode(buf1, img1); err != nil {
		t.Fatalf("Failed to encode image 1: %v", err)
	}
	if err := png.Encode(buf2, img2); err != nil {
		t.Fatalf("Failed to encode image 2: %v", err)
	}

	if bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Error("Different hashes produced identical images")
	}
}

func TestSameHashProducesSameImage(t *testing.T) {
	hash := []byte("consistent-hash")

	img1 := New(hash)
	img2 := New(hash)

	buf1 := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)

	if err := png.Encode(buf1, img1); err != nil {
		t.Fatalf("Failed to encode image 1: %v", err)
	}
	if err := png.Encode(buf2, img2); err != nil {
		t.Fatalf("Failed to encode image 2: %v", err)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Error("Same hash produced different images")
	}
}

func TestCustomOptionsAreApplied(t *testing.T) {
	hash := []byte("options-test")

	// Custom background color
	customBg := color.RGBA{R: 255, G: 0, B: 0, A: 255} // Bright red
	opts := Options{
		Artistic:   true,
		Greyscale:  false,
		Background: customBg,
	}

	img := New(hash, opts)

	// Check background color of a corner pixel
	cornerColor := img.At(0, 0)
	r, g, b, a := cornerColor.RGBA()

	// Convert from 0-65535 range to 0-255
	r >>= 8
	g >>= 8
	b >>= 8
	a >>= 8

	if r != 255 || g != 0 || b != 0 || a != 255 {
		t.Errorf("Background color not applied. Expected RGBA(255,0,0,255), got RGBA(%d,%d,%d,%d)", r, g, b, a)
	}
}

func TestTransparentBackground(t *testing.T) {
	hash := []byte("transparent-test")

	opts := Options{
		Artistic:   true,
		Greyscale:  false,
		Background: color.RGBA{R: 0, G: 0, B: 0, A: 0}, // Transparent
	}

	img := New(hash, opts)

	// Check corner pixel (should be transparent where no monster parts are)
	cornerColor := img.At(0, 0)
	_, _, _, a := cornerColor.RGBA()

	if a != 0 {
		t.Errorf("Expected transparent background (alpha=0), got alpha=%d", a>>8)
	}
}

func TestRgbToHslConversion(t *testing.T) {
	tests := []struct {
		r, g, b     float64
		h, s, l     float64
		description string
	}{
		{1.0, 0.0, 0.0, 0.0, 1.0, 0.5, "pure red"},
		{0.0, 1.0, 0.0, 1.0 / 3.0, 1.0, 0.5, "pure green"},
		{0.0, 0.0, 1.0, 2.0 / 3.0, 1.0, 0.5, "pure blue"},
		{1.0, 1.0, 1.0, 0.0, 0.0, 1.0, "white"},
		{0.0, 0.0, 0.0, 0.0, 0.0, 0.0, "black"},
		{0.5, 0.5, 0.5, 0.0, 0.0, 0.5, "gray"},
	}

	for _, test := range tests {
		h, s, l := rgbToHsl(test.r, test.g, test.b)

		const epsilon = 0.01
		if math.Abs(h-test.h) > epsilon || math.Abs(s-test.s) > epsilon || math.Abs(l-test.l) > epsilon {
			t.Errorf("For %s: expected HSL(%.2f, %.2f, %.2f), got HSL(%.2f, %.2f, %.2f)",
				test.description, test.h, test.s, test.l, h, s, l)
		}
	}
}

func TestHslToRgbConversion(t *testing.T) {
	tests := []struct {
		h, s, l     float64
		r, g, b     float64
		description string
	}{
		{0.0, 1.0, 0.5, 1.0, 0.0, 0.0, "pure red"},
		{1.0 / 3.0, 1.0, 0.5, 0.0, 1.0, 0.0, "pure green"},
		{2.0 / 3.0, 1.0, 0.5, 0.0, 0.0, 1.0, "pure blue"},
		{0.0, 0.0, 1.0, 1.0, 1.0, 1.0, "white"},
		{0.0, 0.0, 0.0, 0.0, 0.0, 0.0, "black"},
		{0.0, 0.0, 0.5, 0.5, 0.5, 0.5, "gray"},
	}

	for _, test := range tests {
		r, g, b := hslToRgb(test.h, test.s, test.l)

		const epsilon = 0.01
		if math.Abs(r-test.r) > epsilon || math.Abs(g-test.g) > epsilon || math.Abs(b-test.b) > epsilon {
			t.Errorf("For %s: expected RGB(%.2f, %.2f, %.2f), got RGB(%.2f, %.2f, %.2f)",
				test.description, test.r, test.g, test.b, r, g, b)
		}
	}
}

func TestRgbHslRoundtripConversion(t *testing.T) {
	testRGBs := []struct {
		r, g, b float64
	}{
		{0.1, 0.2, 0.3},
		{0.5, 0.6, 0.7},
		{0.9, 0.8, 0.7},
		{0.2, 0.7, 0.9},
	}

	for _, rgb := range testRGBs {
		h, s, l := rgbToHsl(rgb.r, rgb.g, rgb.b)
		r2, g2, b2 := hslToRgb(h, s, l)

		const epsilon = 0.01
		if math.Abs(r2-rgb.r) > epsilon || math.Abs(g2-rgb.g) > epsilon || math.Abs(b2-rgb.b) > epsilon {
			t.Errorf("RGB roundtrip failed: initial RGB(%.2f, %.2f, %.2f), final RGB(%.2f, %.2f, %.2f)",
				rgb.r, rgb.g, rgb.b, r2, g2, b2)
		}
	}
}

func TestGreyscaleOption(t *testing.T) {
	hash := []byte("greyscale-test")

	// Create a greyscale monster
	opts := Options{
		Artistic:   true,
		Greyscale:  true,
		Background: color.RGBA{R: 255, G: 255, B: 255, A: 255},
	}

	img := New(hash, opts)

	// Sample multiple points in the image, looking for non-greyscale pixels
	bounds := img.Bounds()
	foundColoredPixel := false

	// Sample a grid of points across the image
	for y := bounds.Min.Y + 20; y < bounds.Max.Y; y += 20 {
		for x := bounds.Min.X + 20; x < bounds.Max.X; x += 20 {
			r, g, b, a := img.At(x, y).RGBA()
			// Skip transparent pixels
			if a < 100 {
				continue
			}

			// For greyscale images, R=G=B
			if !(r == g && g == b) {
				foundColoredPixel = true
				break
			}
		}
		if foundColoredPixel {
			break
		}
	}

	if foundColoredPixel {
		t.Error("Found non-greyscale pixel in greyscale mode")
	}
}
