package quantize

import (
	"bufio"
	"image"
	"image/color"
	"image/gif"
	"os"
	"testing"

	_ "image/jpeg"
)

func TestBuildBucket(t *testing.T) {
	file, err := os.Open("test_image.jpg")
	if err != nil {
		t.Fatal("Couldn't open test file")
	}
	i, _, err := image.Decode(file)
	if err != nil {
		t.Fatal("Couldn't decode test file")
	}

	q := MedianCutQuantizer{MODE, nil}

	colors := q.buildBucket(i)
	t.Logf("Naive color map contains %d elements", len(colors))

	q = MedianCutQuantizer{MODE, func(i image.Image, x int, y int) uint64 {
		if x < 2 || y < 2 || x > i.Bounds().Max.X-2 || y > i.Bounds().Max.X-2 {
			return 1
		}
		return 0
	}}

	colors = q.buildBucket(i)
	t.Logf("Color map contains %d elements", len(colors))
}

func TestQuantize(t *testing.T) {
	file, err := os.Open("test_image.jpg")
	if err != nil {
		t.Fatal("Couldn't open test file")
	}
	i, _, err := image.Decode(file)
	if err != nil {
		t.Fatal("Couldn't decode test file")
	}
	q := MedianCutQuantizer{MEAN, nil}
	p := q.Quantize(make([]color.Color, 0, 256), i)
	t.Logf("Created palette with %d colors", len(p))

	q = MedianCutQuantizer{MODE, nil}
	p = q.Quantize(make([]color.Color, 0, 256), i)
	t.Logf("Created palette with %d colors", len(p))
}

// TestOverQuantize ensures that the quantizer can properly handle an image with more space than needed in the palette
func TestOverQuantize(t *testing.T) {
	file, err := os.Open("test_image2.gif")
	if err != nil {
		t.Fatal("Couldn't open test file")
	}
	i, _, err := image.Decode(file)
	if err != nil {
		t.Fatal("Couldn't decode test file")
	}
	q := MedianCutQuantizer{MEAN, nil}
	p := q.Quantize(make([]color.Color, 0, 256), i)
	t.Logf("Created palette with %d colors", len(p))
}

func TestGif(t *testing.T) {
	file, err := os.Open("test_image.jpg")
	if err != nil {
		t.Fatal("Couldn't open test file")
	}
	i, _, err := image.Decode(file)
	if err != nil {
		t.Fatal("Couldn't decode test file")
	}

	q := MedianCutQuantizer{MODE, nil}
	f, err := os.Create("test_output.gif")
	if err != nil {
		t.Fatal("Couldn't open output file")
	}

	options := gif.Options{NumColors: 128, Quantizer: q, Drawer: nil}

	w := bufio.NewWriter(f)

	gif.Encode(w, i, &options)
}
