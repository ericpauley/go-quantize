// Package quantize offers an implementation of the draw.Quantize interface using an optimized Median Cut method,
// including advanced functionality for fine-grained control of color priority
package quantize

import (
	"image"
	"image/color"
	"math"
	"sync"
)

type bucketPool struct {
	sync.Pool
	maxCap int
	m      sync.Mutex
}

func (p *bucketPool) getBucket(c int) colorBucket {
	p.m.Lock()
	if p.maxCap > c {
		p.maxCap = p.maxCap * 99 / 100
	}
	if p.maxCap < c {
		p.maxCap = c
	}
	maxCap := p.maxCap
	p.m.Unlock()
	val := p.Pool.Get()
	if val == nil || cap(val.(colorBucket)) < c {
		return make(colorBucket, maxCap)[0:c]
	}
	slice := val.(colorBucket)
	slice = slice[0:c]
	for i := range slice {
		slice[i] = colorPriority{}
	}
	return slice
}

var bpool bucketPool

// AggregationType specifies the type of aggregation to be done
type AggregationType uint8

const (
	// Mode - pick the highest priority value
	Mode AggregationType = iota
	// Mean - weighted average all values
	Mean
)

// MedianCutQuantizer implements the go draw.Quantizer interface using the Median Cut method
type MedianCutQuantizer struct {
	// The type of aggregation to be used to find final colors
	Aggregation AggregationType
	// The weighting function to use on each pixel
	Weighting func(image.Image, int, int) uint32
	// Whether to create a transparent entry
	AddTransparent bool
}

//bucketize takes a bucket and performs median cut on it to obtain the target number of grouped buckets
func bucketize(colors colorBucket, num int) (buckets []colorBucket) {
	if len(colors) == 0 || num == 0 {
		return nil
	}
	bucket := colors
	buckets = make([]colorBucket, 1, num*2)
	buckets[0] = bucket

	for len(buckets) < num && len(buckets) < len(colors) { // Limit to palette capacity or number of colors
		bucket, buckets = buckets[0], buckets[1:]
		if len(bucket) < 2 {
			buckets = append(buckets, bucket)
			continue
		} else if len(bucket) == 2 {
			buckets = append(buckets, bucket[:1], bucket[1:])
			continue
		}

		left, right := bucket.partition()
		buckets = append(buckets, left, right)
	}
	return
}

// palettize finds a single color to represent a set of color buckets
func (q MedianCutQuantizer) palettize(p color.Palette, buckets []colorBucket) color.Palette {
	for _, bucket := range buckets {
		switch q.Aggregation {
		case Mean:
			mean := bucket.mean()
			p = append(p, mean)
		case Mode:
			var best colorPriority
			for _, c := range bucket {
				if c.p > best.p {
					best = c
				}
			}
			p = append(p, best.RGBA)
		}
	}
	return p
}

// quantizeSlice expands the provided bucket and then palettizes the result
func (q MedianCutQuantizer) quantizeSlice(p color.Palette, colors []colorPriority) color.Palette {
	numColors := cap(p) - len(p)
	addTransparent := q.AddTransparent
	if addTransparent {
		for _, c := range p {
			if _, _, _, a := c.RGBA(); a == 0 {
				addTransparent = false
			}
		}
		if addTransparent {
			numColors--
		}
	}
	buckets := bucketize(colors, numColors)
	p = q.palettize(p, buckets)
	if addTransparent {
		p = append(p, color.RGBA{0, 0, 0, 0})
	}
	return p
}

func colorAt(m image.Image, x int, y int) color.RGBA {
	switch i := m.(type) {
	case *image.YCbCr:
		yi := i.YOffset(x, y)
		ci := i.COffset(x, y)
		c := color.YCbCr{
			i.Y[yi],
			i.Cb[ci],
			i.Cr[ci],
		}
		return color.RGBA{c.Y, c.Cb, c.Cr, 255}
	case *image.RGBA:
		ci := i.PixOffset(x, y)
		return color.RGBA{i.Pix[ci+0], i.Pix[ci+1], i.Pix[ci+2], i.Pix[ci+3]}
	default:
		return color.RGBAModel.Convert(i.At(x, y)).(color.RGBA)
	}
}

// buildBucketMultiple creates a prioritized color slice with all the colors in
// the images.
func (q MedianCutQuantizer) buildBucketMultiple(ms []image.Image) (bucket colorBucket) {
	if len(ms) < 1 {
		return colorBucket{}
	}

	// If all images are not the same size, and if the first image is not the
	// largest on both X and Y dimensions, this function will eventually trigger
	// a panic unless we've configured the bounds to be based on the greatest x
	// and y of all images in the gif, which we do here:
	leastX, greatestX, leastY, greatestY := math.MaxInt32, 0, math.MaxInt32, 0
	for _, palettedImage := range ms {
		if palettedImage.Bounds().Min.X < leastX {
			leastX = palettedImage.Bounds().Min.X
		}
		if palettedImage.Bounds().Max.X > greatestX {
			greatestX = palettedImage.Bounds().Max.X
		}

		if palettedImage.Bounds().Min.Y < leastY {
			leastY = palettedImage.Bounds().Min.Y
		}
		if palettedImage.Bounds().Max.Y > greatestY {
			greatestY = palettedImage.Bounds().Max.Y
		}
	}

	size := (greatestX - leastX) * (greatestY - leastY) * 2
	sparseBucket := bpool.getBucket(size)

	for _, m := range ms {
		// Since images may have variable size, don't go beyond each specific
		// image's X and Y bounds while we iterate, rather than using the global
		// min and max x and y
		for y := m.Bounds().Min.Y; y < m.Bounds().Max.Y; y++ {
			for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
				priority := uint32(1)
				if q.Weighting != nil {
					priority = q.Weighting(m, x, y)
				}
				if priority != 0 {
					c := colorAt(m, x, y)
					index := int(c.R)<<16 | int(c.G)<<8 | int(c.B)
					for i := 1; ; i++ {
						p := &sparseBucket[index%size]
						if p.p == 0 || p.RGBA == c {
							*p = colorPriority{p.p + priority, c}
							break
						}
						index += 1 + i
					}
				}
			}
		}
	}

	bucket = sparseBucket[:0]
	switch ms[0].(type) {
	case *image.YCbCr:
		for _, p := range sparseBucket {
			if p.p != 0 {
				r, g, b := color.YCbCrToRGB(p.R, p.G, p.B)
				bucket = append(bucket, colorPriority{p.p, color.RGBA{r, g, b, p.A}})
			}
		}
	default:
		for _, p := range sparseBucket {
			if p.p != 0 {
				bucket = append(bucket, p)
			}
		}
	}
	return
}

// Quantize quantizes an image to a palette and returns the palette
func (q MedianCutQuantizer) Quantize(p color.Palette, m image.Image) color.Palette {
	bucket := q.buildBucketMultiple([]image.Image{m})
	defer bpool.Put(bucket)
	return q.quantizeSlice(p, bucket)
}

// QuantizeMultiple quantizes several images at once to a palette and returns
// the palette
func (q MedianCutQuantizer) QuantizeMultiple(p color.Palette, m []image.Image) color.Palette {
	bucket := q.buildBucketMultiple(m)
	defer bpool.Put(bucket)
	return q.quantizeSlice(p, bucket)
}
