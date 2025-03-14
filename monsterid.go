package monsterid

import (
	"embed"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"math/rand/v2"
	"path"
)

//go:embed all:parts/*
var parts embed.FS

var (
	legs  = 5
	hair  = 5
	arms  = 5
	body  = 15
	eyes  = 15
	mouth = 10
)

var bodyParts = []string{"legs", "hair", "arms", "body", "eyes", "mouth"}

type MonsterID struct {
	legs  int
	hair  int
	arms  int
	body  int
	eyes  int
	mouth int
}

// Options represents configuration for monster generation
type Options struct {
	Artistic   bool       // use artistic rendering with colors
	Greyscale  bool       // use greyscale for artistic rendering
	Background color.RGBA // background color (transparent if Alpha=0)
}

// DefaultOptions provides common defaults
func DefaultOptions() Options {
	return Options{
		Artistic:   true,
		Greyscale:  false,
		Background: color.RGBA{R: 240, G: 240, B: 240, A: 255}, // light grey
	}
}

// New creates a monsterid image based on the provided hash.
func New(hash []byte, opts ...Options) image.Image {
	if len(opts) == 0 {
		opts = append(opts, DefaultOptions())
	}
	h := fnv.New64a()
	if _, err := h.Write(hash); err != nil {
		panic(err)
	}
	r := rand.New(rand.NewPCG(h.Sum64(), (h.Sum64()>>1)|1))

	// Select monster parts
	mid := &MonsterID{}
	mid.legs = r.IntN(legs) + 1
	mid.hair = r.IntN(hair) + 1
	mid.arms = r.IntN(arms) + 1
	mid.body = r.IntN(body) + 1
	mid.eyes = r.IntN(eyes) + 1
	mid.mouth = r.IntN(mouth) + 1

	// Create base image
	img := image.NewRGBA(image.Rect(0, 0, 120, 120))

	// Draw background
	if opts[0].Background.A > 0 {
		draw.Draw(img, img.Bounds(), &image.Uniform{C: opts[0].Background}, image.Point{}, draw.Src)
	} else {
		// Transparent background
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				img.SetRGBA(x, y, color.RGBA{})
			}
		}
	}

	// Generate hue for body base color (for artistic mode)
	hue := r.Float64()                  // 0.0-1.0
	saturation := 0.5 + r.Float64()*0.5 // 0.5-1.0

	// Draw each body part
	for _, part := range bodyParts {
		partNum := getPartNumber(mid, part)
		fileName := fmt.Sprintf("%s_%d.png", part, partNum)
		partImage, err := loadPart(fileName)
		if err != nil {
			log.Printf("Error loading part %s: %v", fileName, err)
			continue
		}

		// Apply colorization for artistic mode
		if opts[0].Artistic {
			if part == "body" {
				colorizeImage(partImage, hue, saturation, !opts[0].Greyscale)
			} else if part == "arms" || part == "legs" {
				// Give arms and legs random colors with 30% probability
				if r.Float64() < 0.3 {
					colorizeImage(partImage, r.Float64(), saturation, !opts[0].Greyscale)
				}
			} else if opts[0].Greyscale {
				// Apply greyscale to other parts too
				colorizeImage(partImage, 0, 0, false)
			}
		}

		draw.Draw(img, img.Bounds(), partImage, image.Point{}, draw.Over)
	}

	return img
}

// Helper function to colorize an image with HSL values
func colorizeImage(img *image.RGBA, hue, saturation float64, colorize bool) {
	if !colorize {
		// Convert to greyscale instead of just returning
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, a := img.At(x, y).RGBA()

				// Skip transparent pixels
				if a < 100 {
					continue
				}

				// Convert to greyscale using luminance formula
				grey := uint8((0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 256)
				img.Set(x, y, color.RGBA{
					R: grey,
					G: grey,
					B: grey,
					A: uint8(a >> 8),
				})
			}
		}
		return
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			// Skip transparent pixels
			if a < 100 {
				continue
			}

			// Skip white or near-white pixels
			lightness := float64(r+g+b) / (3 * 0xFFFF)
			if lightness > 0.85 {
				continue
			}

			// Convert pixel to HSL, modify hue/saturation, convert back
			_, _, l := rgbToHsl(float64(r)/0xFFFF, float64(g)/0xFFFF, float64(b)/0xFFFF)
			r2, g2, b2 := hslToRgb(hue, saturation, l)

			img.Set(x, y, color.RGBA{
				R: uint8(r2 * 255),
				G: uint8(g2 * 255),
				B: uint8(b2 * 255),
				A: uint8(a >> 8),
			})
		}
	}
}

// Helper to load a part image from embedded resources
func loadPart(fileName string) (*image.RGBA, error) {
	asset, err := parts.Open(path.Join("parts", fileName))
	if err != nil {
		return nil, err
	}
	defer asset.Close()

	assetImg, err := png.Decode(asset)
	if err != nil {
		return nil, err
	}

	// Convert to RGBA if it isn't already
	bounds := assetImg.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, assetImg, bounds.Min, draw.Src)

	return rgba, nil
}

// RGB to HSL conversion
func rgbToHsl(r, g, b float64) (float64, float64, float64) {
	x := math.Max(math.Max(r, g), b)
	n := math.Min(math.Min(r, g), b)
	h, s, l := 0.0, 0.0, (x+n)/2

	if x != n {
		d := x - n

		switch x {
		case r:
			h = (g - b) / d
			if g < b {
				h += 6
			}
		case g:
			h = (b-r)/d + 2
		case b:
			h = (r-g)/d + 4
		}

		h /= 6
	}

	return h, s, l
}

// HSL to RGB conversion
func hslToRgb(h, s, l float64) (float64, float64, float64) {
	var r, g, b float64

	if s == 0 {
		r, g, b = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}

		p := 2*l - q
		r = hueToRgb(p, q, h+1.0/3.0)
		g = hueToRgb(p, q, h)
		b = hueToRgb(p, q, h-1.0/3.0)
	}

	return r, g, b
}

// Helper for HSL to RGB conversion
func hueToRgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}

	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}

	return p
}

func getPartNumber(mid *MonsterID, part string) int {
	switch part {
	case "legs":
		return mid.legs
	case "hair":
		return mid.hair
	case "arms":
		return mid.arms
	case "body":
		return mid.body
	case "eyes":
		return mid.eyes
	case "mouth":
		return mid.mouth
	}
	return 0
}
