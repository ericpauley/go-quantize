package quantize

import (
	"image"
	"image/color"
	"sort"
)

type colorPriority struct {
	c color.RGBA
	p uint64
}

type colorBucket struct {
	colors   []colorPriority
	min      color.RGBA
	max      color.RGBA
	priority uint64
	passed   bool
}

// MedianCutQuantizer implements the go draw.Quantizer interface using the Median Cut method
type MedianCutQuantizer struct{}

func colorSpan(colors []colorPriority) (min color.RGBA, max color.RGBA, priority uint64) {
	min = color.RGBA{255, 255, 255, 255}
	max = color.RGBA{0, 0, 0, 255}
	for _, c := range colors {
		priority += c.p
		r, g, b, _ := c.c.RGBA()
		if uint8(r) < min.R {
			min.R = uint8(r)
		}
		if uint8(g) < min.G {
			min.G = uint8(g)
		}
		if uint8(b) < min.B {
			min.B = uint8(b)
		}
		if uint8(r) > max.R {
			max.R = uint8(r)
		}
		if uint8(g) > max.G {
			max.G = uint8(g)
		}
		if uint8(b) > max.B {
			max.B = uint8(b)
		}
	}
	return
}

func (q MedianCutQuantizer) quantizeSlice(p color.Palette, colors []colorPriority) color.Palette {
	var bucket colorBucket
	bucket.colors = colors
	bucket.min, bucket.max, bucket.priority = colorSpan(bucket.colors)
	var buckets []colorBucket
	buckets = append(buckets, bucket)
	for len(buckets) < cap(p) && !buckets[0].passed {
		bucket, buckets = buckets[0], buckets[1:]
		if len(bucket.colors) <= 1 { // Mark a bucket and requeue if it only has one element
			bucket.passed = true
			buckets = append(buckets, bucket)
			continue
		}
		rspan := bucket.max.R - bucket.min.R
		gspan := bucket.max.G - bucket.min.G
		bspan := bucket.max.B - bucket.min.B
		b1, b2 := bucket, bucket
		if rspan > gspan && rspan > bspan { // Red is greatest span
			sort.Slice(bucket.colors, func(i, j int) bool {
				return bucket.colors[i].c.R < bucket.colors[j].c.R
			})
			b1.max.R = bucket.colors[len(bucket.colors)/2-1].c.R
			b2.min.R = bucket.colors[len(bucket.colors)/2].c.R
		} else if gspan > bspan { // Green is greatest span
			sort.Slice(bucket.colors, func(i, j int) bool {
				return bucket.colors[i].c.G < bucket.colors[j].c.G
			})
			b1.max.G = bucket.colors[len(bucket.colors)/2-1].c.G
			b2.min.G = bucket.colors[len(bucket.colors)/2].c.G
		} else { // Blue is greatest span
			sort.Slice(bucket.colors, func(i, j int) bool {
				return bucket.colors[i].c.B < bucket.colors[j].c.B
			})
			b1.max.B = bucket.colors[len(bucket.colors)/2-1].c.B
			b2.min.B = bucket.colors[len(bucket.colors)/2].c.B
		}

		b1.colors = bucket.colors[:len(bucket.colors)/2]
		b2.colors = bucket.colors[len(bucket.colors)/2:]

		buckets = append(buckets, b1, b2)
	}
	for _, bucket := range buckets {
		var best *colorPriority
		for _, c := range bucket.colors {
			if best == nil || c.p > best.p {
				best = &c
			}
		}
		p = append(p, best.c)
	}
	return p
}

func buildBucket(m image.Image) []colorPriority {
	colors := make(map[color.Color]uint64)
	for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
		for y := m.Bounds().Min.Y; y < m.Bounds().Max.Y; y++ {
			colors[m.At(x, y)]++
		}
	}
	var colorSlice []colorPriority
	for c, priority := range colors {
		r, g, b, _ := c.RGBA()
		c := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}
		colorSlice = append(colorSlice, colorPriority{c, priority})
	}
	return colorSlice
}

// Quantize quantizes an image to a palette
func (q MedianCutQuantizer) Quantize(p color.Palette, m image.Image) color.Palette {
	return q.quantizeSlice(p, buildBucket(m))
}
