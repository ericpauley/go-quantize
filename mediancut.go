package quantize

import (
	"image"
	"image/color"
	"sort"
)

type colorAxis int

// Color axis constants
const (
	RED colorAxis = iota
	GREEN
	BLUE
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
}

// AggregationType specifies the type of aggregation to be done
type AggregationType int

const (
	// MODE - pick the highest priority value
	MODE AggregationType = iota
	// MEAN - weighted average all values
	MEAN
)

// MedianCutQuantizer implements the go draw.Quantizer interface using the Median Cut method
type MedianCutQuantizer struct {
	// The type of aggregation to be used to find final colors
	Aggregation AggregationType
	// The weighting function to use on each pixel
	Weighting func(image.Image, int, int) uint64
}

func colorSpan(colors []colorPriority) (min color.RGBA, max color.RGBA, priority uint64) {
	min = color.RGBA{255, 255, 255, 255}
	max = color.RGBA{0, 0, 0, 255}
	for _, c := range colors {
		priority += c.p
		r, g, b, _ := c.c.RGBA()
		r /= 257
		g /= 257
		b /= 257
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

func bucketize(colors []colorPriority, num int) (buckets []colorBucket) {
	var bucket colorBucket
	bucket.colors = colors
	bucket.min, bucket.max, bucket.priority = colorSpan(bucket.colors)
	buckets = append(buckets, bucket)
	for len(buckets) < num && len(buckets) < len(colors) {
		bucket, buckets = buckets[0], buckets[1:]
		if len(bucket.colors) < 2 {
			buckets = append(buckets, bucket)
			continue
		}
		rspan := bucket.max.R - bucket.min.R
		gspan := bucket.max.G - bucket.min.G
		bspan := bucket.max.B - bucket.min.B
		var span colorAxis

		if rspan > gspan && rspan > bspan {
			span = RED
		} else if gspan > bspan {
			span = GREEN
		} else {
			span = BLUE
		}

		sort.Slice(bucket.colors, func(i, j int) bool {
			r1, g1, b1, _ := bucket.colors[i].c.RGBA()
			r2, g2, b2, _ := bucket.colors[i].c.RGBA()
			switch span {
			case RED:
				return r1 < r2
			case GREEN:
				return g1 < g2
			default:
				return b1 < b2
			}
		})

		bucket1, bucket2 := bucket, bucket

		var p uint64
		var i int

		for ; i < len(bucket.colors)-1 && p < bucket.priority; i++ {
			p += bucket.colors[i].p
		}

		bucket1.priority = p
		bucket2.priority = bucket.priority - p

		r1, g1, b1, _ := bucket.colors[i-1].c.RGBA()
		r2, g2, b2, _ := bucket.colors[i].c.RGBA()

		switch span {
		case RED:
			bucket1.max.R = uint8(r1 / 257)
			bucket2.min.R = uint8(r2 / 257)
		case GREEN:
			bucket1.max.G = uint8(g1 / 257)
			bucket2.min.G = uint8(g2 / 257)
		case BLUE:
			bucket1.max.B = uint8(b1 / 257)
			bucket2.min.B = uint8(b2 / 257)
		}

		bucket1.colors = bucket.colors[:i]
		bucket2.colors = bucket.colors[i:]

		buckets = append(buckets, bucket1, bucket2)
	}
	return
}

func (q MedianCutQuantizer) palettize(p color.Palette, buckets []colorBucket) color.Palette {
	for _, bucket := range buckets {
		switch q.Aggregation {
		case MEAN:
			var r, g, b uint64
			for _, c := range bucket.colors {
				cr, cg, cb, _ := c.c.RGBA()
				r += uint64(cr) * c.p
				g += uint64(cg) * c.p
				b += uint64(cb) * c.p
			}
			r /= uint64(len(bucket.colors)) * bucket.priority
			g /= uint64(len(bucket.colors)) * bucket.priority
			b /= uint64(len(bucket.colors)) * bucket.priority
			p = append(p, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		case MODE:
			var best *colorPriority
			for _, c := range bucket.colors {
				if best == nil || c.p > best.p {
					best = &c
				}
			}
			p = append(p, best.c)
		}
	}
	return p
}

func (q MedianCutQuantizer) quantizeSlice(p color.Palette, colors []colorPriority) color.Palette {
	buckets := bucketize(colors, cap(p)-len(p))
	return q.palettize(p, buckets)
}

func (q MedianCutQuantizer) buildBucket(m image.Image) []colorPriority {
	colors := make(map[color.Color]uint64)
	if q.Weighting != nil {
		for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
			for y := m.Bounds().Min.Y; y < m.Bounds().Max.Y; y++ {
				colors[m.At(x, y)] += q.Weighting(m, x, y)
			}
		}
	} else {
		for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
			for y := m.Bounds().Min.Y; y < m.Bounds().Max.Y; y++ {
				colors[m.At(x, y)]++
			}
		}
	}

	var colorSlice []colorPriority
	for c, priority := range colors {
		if priority == 0 {
			continue
		}
		r, g, b, _ := c.RGBA()
		c := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}
		colorSlice = append(colorSlice, colorPriority{c, priority})
	}
	return colorSlice
}

// Quantize quantizes an image to a palette
func (q MedianCutQuantizer) Quantize(p color.Palette, m image.Image) color.Palette {
	return q.quantizeSlice(p, q.buildBucket(m))
}
