// Package quantize offers an implementation of the draw.Quantize interface using an optimized Median Cut method,
// including advanced functionality for fine-grained control of color priority
package quantize

import (
	"image"
	"image/color"
)

type colorAxis int

// Color axis constants
const (
	red colorAxis = iota
	green
	blue
)

type colorPriority struct {
	c color.Color
	p uint64
}

func (c colorPriority) RGBA() (r, g, b, a uint32) {
	return c.c.RGBA()
}

type colorBucket []colorPriority

// AggregationType specifies the type of aggregation to be done
type AggregationType int

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
	Weighting func(image.Image, int, int) uint64
	// Whether to create a transparent entry
	AddTransparent bool
}

// colorSpan performs linear color bucket statistics
func colorSpan(colors []colorPriority) (mean color.Color, span colorAxis, priority uint64) {
	var r, g, b uint64    // Sum of channels
	var r2, g2, b2 uint64 // Sum of square of channels

	for _, c := range colors { // Calculate priority-weighted sums
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

	sr := r2/priority - mr*mr // Calculate the variance to find which span is the broadest
	sg := g2/priority - mg*mg
	sb := b2/priority - mb*mb
	if sr > sg && sr > sb {
		span = red
	} else if sg > sb {
		span = green
	} else {
		span = blue
	}
	return
}

// gtColor returns if color a is greater than color b on the specified color channel
func gtColor(a color.Color, b color.Color, span colorAxis) bool {
	ra, ga, ba, _ := a.RGBA()
	rb, gb, bb, _ := b.RGBA()
	switch span {
	case red:
		return ra > rb
	case green:
		return ga > gb
	default:
		return ba > bb
	}
}

//bucketize takes a bucket and performs median cut on it to obtain the target number of grouped buckets
func bucketize(colors []colorPriority, num int) (buckets []colorBucket) {
	bucket := colors
	buckets = []colorBucket{bucket}

	for len(buckets) < num && len(buckets) < len(colors) { // Limit to palette capacity or number of colors
		bucket, buckets = buckets[0], buckets[1:]
		if len(bucket) < 2 {
			buckets = append(buckets, bucket)
			continue
		}
		mean, span, _ := colorSpan(bucket)

		// Janky quicksort partition, needs some odd edge cases supported
		left, right := 0, len(bucket)-1
		for {
			for gtColor(mean, bucket[left], span) && left < len(bucket) {
				left++
			}
			for !gtColor(mean, bucket[right], span) && right > 0 {
				right--
			}
			if left >= right {
				for gtColor(mean, bucket[right], span) || right == 0 {
					right++ // Ensure pivot is in the right place
				}
				break
			}
			bucket[left], bucket[right] = bucket[right], bucket[left]
		}

		buckets = append(buckets, bucket[:right], bucket[right:])
	}
	return
}

// palettize finds a single color to represent a set of color buckets
func (q MedianCutQuantizer) palettize(p color.Palette, buckets []colorBucket) color.Palette {
	for _, bucket := range buckets {
		switch q.Aggregation {
		case Mean:
			mean, _, _ := colorSpan(bucket)
			p = append(p, mean)
		case Mode:
			var best *colorPriority
			for _, c := range bucket {
				if best == nil || c.p > best.p {
					best = &c
				}
			}
			p = append(p, best.c)
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

// buildBucket creates a prioritized color slice with all the colors in the image
func (q MedianCutQuantizer) buildBucket(m image.Image) (bucket []colorPriority) {
	colors := make(map[color.Color]uint64)
	for x := m.Bounds().Min.X; x < m.Bounds().Max.X; x++ {
		for y := m.Bounds().Min.Y; y < m.Bounds().Max.Y; y++ {
			if q.Weighting != nil {
				priority := q.Weighting(m, x, y)
				if priority != 0 {
					colors[m.At(x, y)] += priority
				}
			} else {
				colors[m.At(x, y)]++
			}
		}
	}

	for c, priority := range colors {
		bucket = append(bucket, colorPriority{c, priority})
	}
	return
}

// Quantize quantizes an image to a palette and returns the palette
func (q MedianCutQuantizer) Quantize(p color.Palette, m image.Image) color.Palette {
	return q.quantizeSlice(p, q.buildBucket(m))
}
