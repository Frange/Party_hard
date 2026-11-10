package main

import "math"

// The Eiffel Tower prop ("eiffel", props3.go): 130 x 224, base on the bottom row, centred on x = 65.
// Dark brown-grey wrought iron with fine X lattice, four legs seen as two joined by a big lower arch
// (the centre of the arch is fully transparent: people walk under it), a first platform with a wide
// railing, a see-through gap between the legs, a smaller second platform, a slender shaft with a small
// knot, a top lookout with lit windows and a thin antenna with a flag. The legs keep the old footprint
// (solids at x = -58..-41 and 41..58 from the centre).

const (
	eifW, eifH   = 130, 224
	eifCX        = 65.0
	eifGround    = 222 // last row of the feet
	eifP1, eifP2 = 174, 134
	eifArchRX    = 41.0
	eifArchRY    = 34.0
	eifGapTop    = 146
	eiffelY      = 314 // where PARIS (gen2.go) stands it: the top at y 90, 11 px under the intro plate
)

var (
	eifIron  = [4]uint32{0x8e7664, 0x564539, 0x342a25, 0x161110} // lit (warm) .. darkest
	eifMesh  = uint32(0x281f1b)                                  // the dark body of the lattice
	eifRust  = uint32(0x7a5030)
	eifLight = uint32(0xf4d878)
)

// eifHalf: half width of the tower at row y (exponential flare between key rows).
func eifHalf(y float64) float64 {
	keys := [][2]float64{{14, 2}, {30, 3.5}, {62, 5}, {100, 9.5}, {eifP2, 19}, {eifP1, 44}, {eifGround, 62}}
	if y <= keys[0][0] {
		return keys[0][1]
	}
	for i := 1; i < len(keys); i++ {
		if y <= keys[i][0] {
			a, b := keys[i-1], keys[i]
			t := (y - a[0]) / (b[0] - a[0])
			return math.Exp(math.Log(a[1]) + t*(math.Log(b[1])-math.Log(a[1])))
		}
	}
	return keys[len(keys)-1][1]
}

func eifClear(b *pixBuf, x, y int) {
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return
	}
	i := (y*b.w + x) * 4
	b.p[i], b.p[i+1], b.p[i+2], b.p[i+3] = 0, 0, 0, 0
}

// eifGapHalf: half width of the sky gap between the legs at row y (0 outside the gap rows).
func eifGapHalf(y int) float64 {
	if y < eifGapTop || y > eifP1-1 {
		return 0
	}
	leg := 13.0 + 6*float64(y-eifGapTop)/float64(eifP1-eifGapTop)
	g := eifHalf(float64(y)) - leg
	t := float64(eifP1-1-y) / float64(eifP1-1-eifGapTop)
	g *= math.Sqrt(math.Max(0, 1-t*t*t))
	return math.Max(0, g)
}

func drawEiffel(b *pixBuf, v int) {
	i0, i1, i2, i3 := c(eifIron[0]), c(eifIron[1]), c(eifIron[2]), c(eifIron[3])
	rust := c(eifRust)
	cx := int(eifCX)
	// 1. the body: posts on both edges, X lattice in panels, horizontal girders
	for y := 14; y <= eifGround; y++ {
		h := eifHalf(float64(y))
		l, r := int(math.Round(eifCX-h)), int(math.Round(eifCX+h))
		if h < 4.5 {
			b.rect(l, y, r-l+1, 1, i1)
			if y >= 44 && r-l >= 7 {
				for x := l + 2; x <= r-2; x++ {
					if (x+y)%4 == 0 {
						b.set(x, y, c(eifMesh))
					}
				}
			}
			b.set(l, y, i0)
			b.set(r, y, i2)
			continue
		}
		// the body: a dense dark lattice (lit diagonals on the left, few small holes), not an empty frame
		for x := l + 3; x <= r-3; x++ {
			d1, d2 := (x+y)%6, (x-y+600)%6
			switch {
			case d1 == 3 && (d2 == 3 || d2 == 2):
				eifClear(b, x, y)
			case d1 == 0 || d2 == 0:
				if x < cx {
					b.set(x, y, i1)
				} else {
					b.set(x, y, i2)
				}
			default:
				b.set(x, y, c(eifMesh))
			}
		}
		b.rect(l, y, 2, 1, i0)
		b.rect(l+2, y, 1, 1, i1)
		b.rect(r-1, y, 2, 1, i2)
		b.rect(r-2, y, 1, 1, i1)
	}
	y := 14
	for y < eifGround {
		h := eifHalf(float64(y))
		if h < 4.5 {
			y += 4
			continue
		}
		ph := int(math.Max(6, math.Min(26, 2*h*0.95)))
		if y+ph > eifGround {
			ph = eifGround - y
		}
		h2 := eifHalf(float64(y + ph))
		xl1, xr1 := eifCX-h+2.5, eifCX+h-2.5
		xl2, xr2 := eifCX-h2+2.5, eifCX+h2-2.5
		b.line(int(xl1), y, int(xr2), y+ph, i2)
		b.line(int(xr1), y, int(xl2), y+ph, i1)
		b.line(int(xl1), y, int(xr1), y, i1) // girder
		if (y/ph)%3 == 1 {
			b.line(int(xl1)+1, y+1, int(xr2)-1, y+ph-1, rust) // a little rust on one diagonal
		}
		y += ph
	}
	// 2. the arch under the first platform: carve the gap between the legs and the big arch
	for yy := eifGapTop; yy < eifP1; yy++ {
		g := eifGapHalf(yy)
		for x := cx - int(g); x <= cx+int(g); x++ {
			eifClear(b, x, yy)
		}
		if g > 0 {
			b.rect(cx-int(g)-1, yy, 2, 1, i1)
			b.rect(cx+int(g), yy, 2, 1, i3)
			b.set(cx-int(g)-2, yy, i0)
		}
	}
	// girder band under the platform: dense short verticals
	for yy := eifP1 + 4; yy <= eifP1+13; yy++ {
		h := eifHalf(float64(yy))
		for x := cx - int(h) + 2; x <= cx+int(h)-2; x++ {
			if yy == eifP1+4 || yy == eifP1+13 {
				b.set(x, yy, i2)
			} else if (x-cx)%4 == 0 {
				b.set(x, yy, i1)
			} else {
				b.set(x, yy, c(0x2a211e))
			}
		}
	}
	for yy := eifP1 + 14; yy <= eifGround; yy++ {
		for x := 0; x < eifW; x++ {
			d := math.Hypot((float64(x)-eifCX+0.5)/eifArchRX, (float64(yy)-float64(eifGround)-0.5)/eifArchRY)
			dy := float64(yy) - float64(eifGround) - 0.5
			if dy < 0 && d < 0.84 {
				eifClear(b, x, yy)
			} else if dy < 0 && d < 1.0 {
				col := i2
				if x < cx {
					col = i1
				}
				if d > 0.95 || d < 0.88 {
					col = i3
				}
				b.set(x, yy, col)
			} else if dy < 0 && d >= 1.0 && d < 1.07 && b.get(x, yy).A != 0 {
				b.set(x, yy, i0)
			}
		}
	}
	// the legs: fine X lattice (cross hatch) between the outer post and the opening
	for yy := eifGapTop; yy <= eifGround-4; yy++ {
		if yy >= eifP1 && yy <= eifP1+13 {
			continue
		}
		h := eifHalf(float64(yy))
		for x := cx - int(h) + 3; x <= cx+int(h)-3; x++ {
			if yy < eifP1 {
				if g := eifGapHalf(yy); g > 0 && math.Abs(float64(x-cx)) <= g+3 {
					continue
				}
			} else {
				d := math.Hypot((float64(x)-eifCX+0.5)/eifArchRX, (float64(yy)-float64(eifGround)-0.5)/eifArchRY)
				if float64(yy) < eifGround+0.5 && d < 1.09 {
					continue
				}
			}
			switch {
			case yy%11 == 0:
				b.set(x, yy, i1) // girder
			case ((x+yy)%5 == 0 || (x-yy+500)%5 == 0):
				b.set(x, yy, i2)
			case (x+yy)%5 == 2 && (x-yy+500)%5 == 2:
				eifClear(b, x, yy) // a few small holes: the legs are massive iron
			default:
				b.set(x, yy, c(eifMesh))
				if x < cx && ((x+yy)%5 == 1 || (x-yy+500)%5 == 4) {
					b.set(x, yy, i1) // the lit side of the lattice
				}
			}
		}
	}
	// legs: the bottom of the arch opening is open to the ground (transparent), feet plates
	for yy := eifGround - 3; yy <= eifGround; yy++ {
		for _, side := range []int{-1, 1} {
			x0 := cx + side*58 - 9
			b.rect(x0, yy, 19, 1, i3)
		}
	}
	for _, side := range []int{-1, 1} {
		b.rect(cx+side*58-9, eifGround-3, 19, 1, i1)
	}
	// 3. platforms with railings
	deck := func(py int, over int, rail int) {
		h := eifHalf(float64(py))
		l, w := int(eifCX-h)-over, int(2*h)+2*over+1
		b.rect(l, py, w, 2, i0)
		b.rect(l, py+2, w, 4, i3)
		for x := l + 1; x < l+w-1; x += 4 {
			b.set(x, py+3, i2) // the rivets of the thick girder
		}
		b.rect(l+1, py+6, w-2, 1, i1)
		b.rect(l, py-rail, w, 1, i1) // top rail
		for x := l; x < l+w; x += 3 {
			b.rect(x, py-rail+1, 1, rail-1, i2)
		}
		b.rect(l, py-1, w, 1, i2)
	}
	deck(eifP1, 5, 5)
	deck(eifP2, 3, 4)
	deck(84, 2, 2) // the small knot on the shaft
	// 4. the top lookout with lit windows and the antenna
	b.rect(cx-6, 24, 13, 2, i0) // balcony
	b.rect(cx-6, 26, 13, 1, i3)
	b.rect(cx-5, 20, 11, 4, i1) // cabin
	b.rect(cx-5, 20, 11, 1, i0)
	for _, wx := range []int{cx - 4, cx - 1, cx + 2} {
		b.rect(wx, 21, 2, 2, c(eifLight))
		b.set(wx, 21, c(0xfff0b0))
	}
	b.rect(cx-4, 17, 9, 3, i2) // roof
	b.rect(cx-3, 15, 7, 2, i1)
	b.rect(cx-1, 6, 3, 9, i2)
	b.set(cx-1, 6, i0)
	b.line(cx, 0, cx, 7, i3)           // antenna
	b.rect(cx+1, 1, 4, 2, c(0xd04040)) // flag
	b.rect(cx+1, 3, 3, 1, c(0xe8e0d0))
	// legs of the lookout: brace
	b.line(cx-6, 26, cx-2, 34, i2)
	b.line(cx+6, 26, cx+2, 34, i1)
	eifOutline(b)
}

// eifOutline: a continuous dark 1 px rim on the silhouette and around the big openings (the sky outside, the gap
// between the legs and the arch), so the iron never melts into the cobbles. The small lattice holes keep no rim.
func eifOutline(b *pixBuf) {
	W, H := b.w, b.h
	outside := make([]bool, W*H)
	var stack []int
	push := func(x, y int) {
		if x < 0 || y < 0 || x >= W || y >= H {
			return
		}
		i := y*W + x
		if outside[i] || b.p[i*4+3] != 0 {
			return
		}
		outside[i] = true
		stack = append(stack, i)
	}
	for x := 0; x < W; x++ {
		push(x, 0)
		push(x, H-1)
	}
	for y := 0; y < H; y++ {
		push(0, y)
		push(W-1, y)
	}
	push(int(eifCX)+b.ox, 160+b.oy) // the gap between the legs is closed by the first platform
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		x, y := i%W, i/W
		push(x+1, y)
		push(x-1, y)
		push(x, y+1)
		push(x, y-1)
	}
	rim := c(eifIron[3])
	var mark []int
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			i := y*W + x
			if b.p[i*4+3] == 0 {
				continue
			}
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx >= 0 && ny >= 0 && nx < W && ny < H && outside[ny*W+nx] {
					mark = append(mark, i)
					break
				}
			}
		}
	}
	for _, i := range mark {
		if p := b.p[i*4:]; p[0] < 0xf0 { // never darken the lit windows or the flag
			if !(p[0] > 0xc0 && p[1] < 0x60) && !(p[0] > 0xe0 && p[1] > 0xc0) {
				p[0], p[1], p[2], p[3] = rim.R, rim.G, rim.B, 255
			}
		}
	}
}
