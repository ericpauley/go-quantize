package quantize

import (
	"bufio"
	"image"
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
	colors := buildBucket(i)
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
	q := MedianCutQuantizer{}
	p := q.Quantize(nil, i)
	t.Logf("Created palette with %d colors", len(p))

	f, err := os.Create("test_output.gif")
	if err != nil {
		t.Fatal("Couldn't open output file")
	}

	options := gif.Options{NumColors: 16, Quantizer: q, Drawer: nil}

	w := bufio.NewWriter(f)

	gif.Encode(w, i, &options)
}
