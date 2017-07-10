package quantize

import "image/color"

type colorAxis uint8

// Color axis constants
const (
	red colorAxis = iota
	green
	blue
)

// gtColor returns if color a is greater than color b on the specified color channel
func gt(c uint8, other color.RGBA, span colorAxis) bool {
	switch span {
	case red:
		return c > other.R
	case green:
		return c > other.G
	default:
		return c > other.B
	}
}

type colorPriority struct {
	p uint32
	color.RGBA
}

type colorBucket []colorPriority

func (cb colorBucket) partition() (colorBucket, colorBucket) {
	mean, span := cb.span()
	left, right := 0, len(cb)-1
	for left < right {
		for gt(mean, cb[left].RGBA, span) {
			left++
		}
		for !gt(mean, cb[right].RGBA, span) {
			right--
		}
		cb[left], cb[right] = cb[right], cb[left]
	}
	return cb[:left], cb[left:]
}

func (cb colorBucket) mean() color.RGBA {
	var r, g, b uint64
	var p uint64
	for _, c := range cb {
		p += uint64(c.p)
		r += uint64(c.R) * uint64(c.p)
		g += uint64(c.G) * uint64(c.p)
		b += uint64(c.B) * uint64(c.p)
	}
	return color.RGBA{uint8(r / p), uint8(g / p), uint8(b / p), 255}
}

type constraint struct {
	min  uint8
	max  uint8
	vals [256]uint64
}

func (c *constraint) update(index uint8, p uint32) {
	if index < c.min {
		c.min = index
	}
	if index > c.max {
		c.max = index
	}
	c.vals[index] += uint64(p)
}

func (c *constraint) span() uint8 {
	return c.max - c.min
}

func (cb colorBucket) span() (uint8, colorAxis) {
	var R, G, B constraint
	R.min = 255
	G.min = 255
	B.min = 255
	var p uint64
	for _, c := range cb {
		R.update(c.R, c.p)
		G.update(c.G, c.p)
		B.update(c.B, c.p)
		p += uint64(c.p)
	}
	var toCount *constraint
	var span colorAxis
	if R.span() > G.span() && R.span() > B.span() {
		span = red
		toCount = &R
	} else if G.span() > B.span() {
		span = green
		toCount = &G
	} else {
		span = blue
		toCount = &B
	}
	var counted uint64
	var i int
	var c uint64
	for i, c = range toCount.vals {
		if counted > p/2 || counted+c == p {
			break
		}
		counted += c
	}
	return uint8(i), span
}
