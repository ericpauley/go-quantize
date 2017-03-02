package quantize

import (
	"image"
	"image/color"
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

func (c colorPriority) RGBA() (r, g, b, a uint32) {
	return c.c.RGBA()
}

type colorBucket []colorPriority

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

func colorSpan(colors []colorPriority) (mean color.RGBA, span colorAxis, priority uint64) {
	var r, g, b uint64
	var r2, g2, b2 uint64
	for _, c := range colors {
		priority += c.p
		cr, cg, cb, _ := c.c.RGBA()
		c64r, c64g, c64b := uint64(cr/257), uint64(cg/257), uint64(cb/257)
		r += uint64(c64r * c.p)
		g += uint64(c64g * c.p)
		b += uint64(c64b * c.p)
		r2 += uint64(c64r * c64r * c.p)
		g2 += uint64(c64g * c64g * c.p)
		b2 += uint64(c64b * c64g * c.p)
	}
	mr := r / priority
	mg := g / priority
	mb := b / priority
	mean = color.RGBA{uint8(mr), uint8(mg), uint8(mb), 255}
	sr := r2/priority - mr*mr
	sg := g2/priority - mg*mg
	sb := b2/priority - mb*mb
	if sr > sg && sr > sb {
		span = RED
	} else if sg > sb {
		span = GREEN
	} else {
		span = BLUE
	}
	return
}

func compareColors(a color.Color, b color.Color, span colorAxis) int {
	ra, ga, ba, _ := a.RGBA()
	rb, gb, bb, _ := b.RGBA()
	switch span {
	case RED:
		if ra > rb {
			return 1
		} else if ra < rb {
			return -1
		}
	case GREEN:
		if ga > gb {
			return 1
		} else if ga < gb {
			return -1
		}
	case BLUE:
		if ba > bb {
			return 1
		} else if ba < bb {
			return -1
		}
	}
	return 0
}

func bucketize(colors []colorPriority, num int) (buckets []colorBucket) {
	bucket := colors
	buckets = []colorBucket{bucket}
	for len(buckets) < num && len(buckets) < len(colors) {
		bucket, buckets = buckets[0], buckets[1:]
		if len(bucket) < 2 {
			buckets = append(buckets, bucket)
			continue
		}
		mean, span, _ := colorSpan(bucket)

		left, right := 0, len(bucket)-1
		for {
			for compareColors(bucket[left], mean, span) < 0 && left < len(bucket) {
				left++
			}
			for compareColors(bucket[right], mean, span) >= 0 && right > 0 {
				right--
			}
			if left >= right {
				for compareColors(bucket[right], mean, span) < 0 {
					right++ // Try to get to the mean
				}
				break
			}
			bucket[left], bucket[right] = bucket[right], bucket[left]
		}

		if right == 0 {
			right = 1
		} else if right == len(bucket)-1 {
			right = len(bucket) - 2
		}

		buckets = append(buckets, bucket[:right], bucket[right:])
	}
	return
}

func (q MedianCutQuantizer) palettize(p color.Palette, buckets []colorBucket) color.Palette {
	for _, bucket := range buckets {
		switch q.Aggregation {
		case MEAN:
			mean, _, _ := colorSpan(bucket)
			p = append(p, mean)
		case MODE:
			var best *colorPriority
			for _, c := range bucket {
				if best == nil || c.p > best.p {
					best = &c
				}
			}
			if best != nil {
				p = append(p, best.c)
			}
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
