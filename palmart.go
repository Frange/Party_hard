package main

import "math"

// The palm tree prop ("palm", props.go): 48 x 72, the foot of the trunk at the bottom centre (the
// footprint and the solid area are those of the old palm: 8 x 6 under x = 24). Four variants
// (height, lean, number of fronds): a ringed beige-brown trunk, wide at the base and narrowing
// as it curves up, lit on the left and in shadow on the right; a crown of 6-8 big drooping fronds
// with ragged leaflets (yellow-green light on top, deep blue-green underneath), coconuts, and a
// tuft of grass at the foot. The wind (ambient.go drawSway) still rocks the crown.

type palmShape struct {
	crownY int     // height of the crown's centre
	lean   float64 // how far the top of the trunk leans sideways (px)
	fronds int
	seed   uint32
}

var palmShapes = [4]palmShape{
	{20, 4, 7, 11},
	{25, -4, 6, 23},
	{17, 2, 8, 37},
	{22, -2.5, 7, 51},
}

// palmAngles: the direction of frond i of n, in degrees (180 = left, 270 = up, 360 = right).
func palmAngles(n, i int) float64 {
	switch n {
	case 6:
		return [6]float64{172, 208, 250, 292, 332, 368}[i]
	case 8:
		return [8]float64{170, 194, 222, 254, 286, 316, 346, 370}[i%8]
	}
	return [7]float64{171, 200, 236, 270, 304, 340, 369}[i%7]
}

var (
	palmBark   = [5]uint32{0xd9b67e, 0xb48c58, 0x8e6a3e, 0x664828, 0x4a321c} // light .. ring shadow
	palmLeafF  = [4]uint32{0xc6e85a, 0x6cc044, 0x2f8f3c, 0x1c5e48}           // front fronds: light, mid, dark, blue shadow
	palmLeafB  = [4]uint32{0x6aae46, 0x2e7e3e, 0x1c5e3e, 0x103c3c}           // back fronds, a step darker
	palmGrassC = [3]uint32{0x2a6a2e, 0x46963a, 0x86c44a}
)

func drawPalm(b *pixBuf, v int) {
	ps := palmShapes[((v%4)+4)%4]
	const baseX, baseY = 24.0, 71
	topY := ps.crownY + 3
	span := float64(baseY - topY)
	trunkX := func(y int) float64 { // the centre of the trunk at row y: it bends more towards the top
		u := float64(baseY-y) / span
		return baseX + ps.lean*math.Pow(u, 1.6) + 0.8*math.Sin(u*math.Pi)
	}

	// grass behind the trunk
	palmGrass(b, ps.seed, false)

	// the trunk, row by row from the foot up
	ringOff := int(ps.seed % 3)
	for y := baseY; y >= topY; y-- {
		u := float64(baseY-y) / span
		w := 7.0 - 3.2*u
		if y > baseY-4 { // the foot flares
			w += float64(y-(baseY-4)) * 0.7
		}
		cx := trunkX(y)
		l := int(math.Round(cx - w/2))
		r := int(math.Round(cx+w/2)) - 1
		ring := (y+ringOff)%4 == 0
		for x := l; x <= r; x++ {
			k := 1 // body
			switch {
			case x == l:
				k = 0 // lit edge
			case x >= r-1:
				k = 2 // shaded side
				if x == r {
					k = 3
				}
			}
			if ring {
				k = min(k+2, 4)
				if x == l {
					k = 2
				}
			} else if (y+ringOff)%4 == 1 && x == l+1 {
				k = 0 // the lit lip under each ring
			}
			b.set(x, y, c(palmBark[k]))
		}
	}

	// the crown: the back fronds first, then the front ones
	cx, cy := trunkX(topY), float64(ps.crownY)
	n := ps.fronds
	for pass := 0; pass < 2; pass++ {
		for i := 0; i < n; i++ {
			back := i%2 == 1
			if back != (pass == 0) {
				continue
			}
			// spread the fronds over the upper half circle, from left to right, with a little jitter;
			// the ones pointing up are seen end-on, so they are shorter and droop less
			a := palmAngles(n, i)*math.Pi/180 + (jpHash(i, int(ps.seed), 5)-0.5)*0.14
			up := math.Pow(math.Abs(math.Sin(a)), 3)
			length := (19 + jpHash(i, int(ps.seed), 7)*6) * (1 - 0.45*up)
			if back {
				length -= 2
			}
			if room := (math.Abs(math.Cos(a)) + 0.01); room*length > 23 { // stay inside the sprite
				length = 23 / room
			}
			droop := (0.62 + jpHash(i, int(ps.seed), 9)*0.3) * (1 - 0.25*up)
			pal := palmLeafF
			if back {
				pal = palmLeafB
			}
			palmFrond(b, cx, cy, a, length, droop, pal, i+int(ps.seed))
		}
	}
	// coconuts under the crown, then the heart of the crown
	for k, d := range [3][2]float64{{-2, 2}, {1, 3}, {3, 1}} {
		x, y := int(math.Round(cx+d[0])), int(math.Round(cy+d[1]))
		b.set(x, y, c(0x5a3a1c))
		b.set(x+1, y, c(0x4a2e16))
		b.set(x, y+1, c(0x3a2410))
		b.set(x+1, y+1, c(0x3a2410))
		if k != 1 {
			b.set(x, y, c(0x8a6036))
		}
	}
	b.set(int(math.Round(cx)), int(cy)-1, c(palmLeafF[1]))
	b.set(int(math.Round(cx))-1, int(cy)-1, c(palmLeafF[0]))

	// grass in front of the trunk
	palmGrass(b, ps.seed, true)
}

// palmFrond draws one frond from (cx, cy) at angle a (radians, screen space: up is -pi/2): a spine that
// arcs out and droops, a fringe of leaflets under it (long and short in turn: a ragged, sawtooth edge,
// dark towards the tips) and short lit leaflets on top.
func palmFrond(b *pixBuf, cx, cy, a, length, droop float64, pal [4]uint32, seed int) {
	dx, dy := math.Cos(a), math.Sin(a)
	steps := int(length * 2)
	type pt struct{ x, y, nx, ny, tx, ty, w float64 }
	var pts [64]pt
	np := 0
	for st := 0; st <= steps && np < len(pts); st++ {
		s := float64(st) / 2
		u := s / length
		x := cx + dx*s
		y := cy + dy*s + droop*s*s/length
		tx, ty := dx, dy+2*droop*s/length
		tl := math.Hypot(tx, ty)
		tx, ty = tx/tl, ty/tl
		nx, ny := -ty, tx // the normal on the lower side
		if ny < 0 || ny == 0 && nx*dx < 0 {
			nx, ny = -nx, -ny
		}
		w := 6.2 * math.Pow(math.Sin(math.Pi*(0.08+0.84*u)), 0.7)
		pts[np] = pt{x, y, nx, ny, tx, ty, w}
		np++
	}
	put := func(x, y float64, col uint32) {
		ix, iy := int(math.Round(x)), int(math.Round(y))
		if ix >= 0 && ix <= 47 && iy >= 0 {
			b.set(ix, iy, c(col))
		}
	}
	// the fringe under the spine
	for k, p := range pts[:np] {
		n := (k/2 + seed) % 2
		l := p.w
		if n == 1 {
			l *= 0.55
		}
		if (k/2+seed)%5 == 4 && k > np/4 {
			l = 1 // a notch between leaflets
		}
		ex, ey := p.nx+p.tx*0.35, p.ny+0.55+p.ty*0.35 // leaflets hang down and point a little outwards
		el := math.Hypot(ex, ey)
		ex, ey = ex/el, ey/el
		for d := 0.0; d <= l; d += 0.6 {
			col := pal[1]
			switch {
			case d > l*0.75 && l > 2:
				col = pal[3]
			case d > 1.2:
				col = pal[2]
			}
			put(p.x+ex*d, p.y+ey*d, col)
		}
	}
	// the spine, lit on top, and short leaflets standing up
	for k, p := range pts[:np] {
		put(p.x, p.y, pal[1])
		put(p.x-p.nx, p.y-p.ny, pal[0])
		if p.w > 3 && (k/2+seed)%2 == 0 {
			ux, uy := -p.nx*0.8+p.tx*0.6, -p.ny*0.8+p.ty*0.6
			for d := 1.0; d <= p.w*0.45; d += 0.6 {
				put(p.x+ux*d, p.y+uy*d, pal[0])
			}
		}
	}
}

// palmGrass: a tuft of grass around the foot of the trunk (behind it, or the blades in front).
func palmGrass(b *pixBuf, seed uint32, front bool) {
	for i := 0; i < 14; i++ {
		x := 11 + i*2 + int(jpHash(i, int(seed), 13)*2)
		if (x >= 21 && x <= 27) != front {
			continue // the blades over the trunk's foot go in front of it, the rest behind
		}
		h := 2 + int(jpHash(i, int(seed), 17)*4)
		if x < 15 || x > 33 {
			h -= 2
		}
		if h < 1 {
			h = 1
		}
		for k := 0; k < h; k++ {
			col := palmGrassC[1]
			switch {
			case k == h-1:
				col = palmGrassC[2]
			case k == 0:
				col = palmGrassC[0]
			}
			b.set(x, 71-k, c(col))
		}
		if i%2 == 0 {
			b.set(x+1, 71, c(palmGrassC[0]))
		}
	}
}
