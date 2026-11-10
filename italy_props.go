package main

import (
	"fmt"
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Props of the ITALIA party (italy.go): the leaning tower, the Roman amphitheatre (a ring cut in segments so that
// every piece sorts with the people at its own depth), the Venetian palaces, gondolas and mooring poles, the gelato
// cart, the pizza oven, vespas and cypresses. Everything is drawn by code; generic, no brands.
// ---------------------------------------------------------------------------

// the amphitheatre: ground ellipse (outer radii), wall thickness and the number of segments of the ring
const (
	colosRX   = 165.0
	colosRY   = 62.0
	colosT    = 14.0 // thickness of the wall on the ground, along x
	colosTY   = 10.0 // and along y
	colosN    = 24   // segments of 15 degrees, segment k centred at k*15 (0 = right, 90 = front, 180 = left)
	colosGate = 6    // the main arch (front centre): open, only the lintel overhead
	palazzoH  = 118  // height of the Venetian palaces
)

// colosRuined: the segments that collapsed down to rubble (an open gap on the right front)
var colosRuined = map[int]bool{2: true, 3: true}

// colosHeight: height of the wall of segment k (the left and the back stand tall, the right and the front are ruined).
var colosHeight = [colosN]float64{26, 22, 4, 4, 18, 36, 46, 36, 24, 30, 46, 62, 76, 76, 76, 76, 76, 76, 76, 70, 62, 52, 42, 32}

// colosPart: one piece of the ring, placed at (centre + D) by the generator.
type colosPart struct {
	Kind string
	D    Vec
}

var colosParts []colosPart

func colosKind(k int) string { return fmt.Sprintf("colos%02d", k) }

func itHash(a, b int) int {
	h := uint32(a*73856093) ^ uint32(b*19349663) ^ 0x9e3779b9
	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15
	return int(h & 0x7fffffff)
}

func degRad(d float64) float64 { return d * math.Pi / 180 }

// colosSegment builds the prop of ring segment k: its sprite, its anchor relative to the ellipse centre and its solids.
func colosSegment(k int) (*PropKind, Vec) {
	a0, a1 := degRad(float64(k)*15-7.5), degRad(float64(k)*15+7.5)
	h := colosHeight[k]
	irx, iry := colosRX-colosT, colosRY-colosTY
	// bounding box of the band on the ground and of the wall above it (ellipse-centre coordinates)
	minX, maxX, minY, maxY := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	gMinX, gMaxX, gMinY, gMaxY := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	faceMaxY := math.Inf(-1)
	for t := a0 - 0.01; t <= a1+0.01; t += 0.002 {
		ox, oy := colosRX*math.Cos(t), colosRY*math.Sin(t)
		ix, iy := irx*math.Cos(t), iry*math.Sin(t)
		for _, p := range [][2]float64{{ox, oy}, {ix, iy}} {
			minX, maxX = math.Min(minX, p[0]), math.Max(maxX, p[0])
			minY, maxY = math.Min(minY, p[1]-h-6), math.Max(maxY, p[1])
		}
		if t >= a0 && t <= a1 {
			for _, p := range [][2]float64{{ox, oy}, {ix, iy}} {
				gMinX, gMaxX = math.Min(gMinX, p[0]), math.Max(gMaxX, p[0])
				gMinY, gMaxY = math.Min(gMinY, p[1]), math.Max(gMaxY, p[1])
			}
			fy := iy
			if math.Sin(t) >= 0 {
				fy = oy
			}
			faceMaxY = math.Max(faceMaxY, fy)
		}
	}
	ax := math.Round((minX + maxX) / 2)
	ay := math.Round(faceMaxY) + 1 // the anchor: the lowest point of the visible face
	halfW := math.Ceil(math.Max(ax-minX, maxX-ax)) + 2
	W := int(halfW * 2)
	H := int(math.Ceil(ay-minY)) + 2
	kind := &PropKind{W: W, H: H, FW: 4, FH: 3, NoSolid: true}
	if k != colosGate && !colosRuined[k] {
		kind.Solids = []Rect{{math.Round(gMinX-ax) + 1, math.Round(gMinY-ay) + 1, math.Max(2, math.Round(gMaxX-gMinX)-2), math.Max(2, math.Round(gMaxY-gMinY)-1)}}
	}
	if k == colosGate { // the two pillars of the main arch are solid, the passage between them is not
		half := colosRX * math.Sin(degRad(7.5))
		open := math.Ceil(colosRX * math.Sin(degRad(7.5*colosGateOpen)))
		pw := math.Floor(half - open)
		y0, hgt := math.Round(gMinY-ay)+1, math.Max(2, math.Round(gMaxY-gMinY)-1)
		kind.Solids = []Rect{{-open - pw - ax, y0, pw, hgt}, {open - ax, y0, pw, hgt}}
	}
	kind.Draw = func(b *pixBuf, v int) { colosDraw(b, k, float64(W)/2-ax, float64(H)-ay) }
	return kind, Vec{ax, ay}
}

// colosGateOpen: the half width of the passage of the main arch, as a fraction of the half width of its segment;
// outside it stand the two solid pillars.
const colosGateOpen = 0.64

// colosDraw paints segment k into b; (ox, oy) is where the ellipse centre falls in the sprite. Warm, weathered
// travertine (kept muted), every storey shaded on its own (the cornice lit, a band of shadow under it), deep arches
// (a dark inside, the vault a little lighter, one lit jamb and one in shadow), half columns between them, broken
// stepped tops with weeds on the ruined side and real holes in the collapsed part.
func colosDraw(b *pixBuf, k int, ox, oy float64) {
	stone := rgb(0xc4b8a4) // travertine: muted warm beige
	topC := rgb(0xdcd4c4)
	dark := rgb(0x261e1a)
	vault := rgb(0x45382f)
	backHole := rgb(0x3a302a)
	moss := rgb(0x5e7a44)
	mossL := rgb(0x7a9456)
	irx, iry := colosRX-colosT, colosRY-colosTY
	a0, a1 := degRad(float64(k)*15-7.5), degRad(float64(k)*15+7.5)
	type col struct {
		x                int
		fy, ey, h, light float64
		s                float64
		front            bool
		t                float64
	}
	var cols []col
	lastX := -9999
	for t := a0 - 0.004; t <= a1+0.004; t += 0.0015 {
		front := math.Sin(t) >= 0
		fx, fy := colosRX*math.Cos(t), colosRY*math.Sin(t)
		ex, ey := irx*math.Cos(t), iry*math.Sin(t)
		light := 0.84 - 0.2*math.Cos(t) // the low sun comes from the left (west)
		if !front {
			fx, fy, ex, ey = ex, ey, fx, fy
			light = 0.8 + 0.18*math.Cos(t)
		}
		x := int(math.Round(fx + ox))
		if x == lastX {
			continue
		}
		lastX = x
		h := colosHeight[k]
		if h > 6 && h < 70 { // a ruined top: broken in blocky steps, not noise
			h += float64(itHash(int(math.Floor((fx+ox)/4)), k)%9) - 4
		}
		cols = append(cols, col{x: x, fy: fy + oy, ey: ey + oy, h: h, light: light, s: t * 118, front: front, t: t})
		_ = ex
	}
	gateMid := degRad(float64(colosGate) * 15)
	for _, c := range cols {
		base := int(math.Round(c.fy))
		hh := int(math.Round(c.h))
		u := math.Mod(c.s, 16)
		if u < 0 {
			u += 16
		}
		if colosRuined[k] { // the collapse: a stub of the piers between real holes, blocks and weeds on the ground
			pier := u < 4 || u > 14.5
			top := 3
			if pier {
				top = 6 + itHash(int(c.s/16), k)%9
			}
			for z := 0; z <= top; z++ {
				if !pier && z > 1 && itHash(c.x, z)%3 == 0 {
					continue // gaps between the fallen blocks
				}
				cc := shade(stone, c.light*(0.78+float64(itHash(z/2, c.x/3)%3)*0.07))
				if z == top {
					cc = shade(topC, c.light*0.95)
				}
				if (z+itHash(c.x/3, 5))%5 == 0 {
					cc = shade(cc, 0.8) // the joints of the blocks
				}
				b.set(c.x, base-z, cc)
			}
			if itHash(c.x, 7)%3 == 0 { // weeds
				b.set(c.x, base-top-1, moss)
				if itHash(c.x, 9)%2 == 0 {
					b.set(c.x, base-top-2, mossL)
				}
			}
			continue
		}
		openFrom := 0 // the gate: the passage under a round arch, the pillars stay solid
		if k == colosGate {
			d := math.Abs(c.t-gateMid) / degRad(7.5) / colosGateOpen // 0 in the middle, 1 at the pillars
			if d < 1 {
				openFrom = int(26 + 12*math.Sqrt(1-d*d))
			}
		}
		pilaster := u >= 0.5 && u < 2.5
		for z := openFrom; z <= hh; z++ {
			tier := z / 22
			r := 22*(tier+1) - z                                 // rows from the top of the tier
			tk := []float64{0.86, 0.94, 1.0, 1.04}[min(tier, 3)] // the lower storeys sit in more shadow
			L := c.light * tk
			cc := shade(stone, L)
			switch {
			case z >= hh-1:
				cc = shade(topC, c.light)
			case r <= 1:
				cc = shade(topC, L*0.98) // cornice
			case r == 2:
				cc = shade(stone, L*0.62) // the shadow under the cornice
			case r == 3:
				cc = shade(stone, L*0.8)
			case tier >= 3:
				if r >= 8 && r <= 12 && u >= 6 && u <= 10 { // the attic: small windows
					cc = dark
				} else if u >= 0 && u < 1.5 {
					cc = shade(stone, L*0.9) // flat pilasters
				}
			case k == colosGate && openFrom > 0:
				cc = shade(stone, L*0.97)
				if z == openFrom || z == openFrom+1 {
					cc = shade(stone, L*0.7) // the arch ring
				}
				if z >= openFrom+2 && z <= openFrom+4 && math.Abs(c.t-gateMid) < degRad(0.6) {
					cc = shade(topC, L) // the keystone
				}
			default:
				in := u >= 4.5 && u <= 14.5 && r >= 5 && r <= 19
				if in && r <= 7 { // round top of the arch
					w := float64(8-r) * 1.3
					in = u >= 4.5+w && u <= 14.5-w
				}
				switch {
				case in && c.front:
					cc = dark
					if r <= 8 {
						cc = vault // the vault inside, a little lit
					}
					if u < 6 {
						cc = shade(dark, 0.8) // the jamb in shadow
					} else if u > 13.2 {
						cc = shade(stone, L*0.72) // the lit jamb: the wall is thick
					}
				case in:
					cc = backHole
					if u > 13.2 {
						cc = shade(stone, L*0.66)
					}
				case pilaster && r > 3:
					cc = shade(stone, L*1.08) // the half column, lit on its left
					if u >= 2 {
						cc = shade(stone, L*0.78)
					}
				case r == 20 || r == 21:
					cc = shade(stone, L*0.9) // the impost band under the arches
				}
			}
			if itHash(c.x*3, z)%17 == 0 { // weathered stone
				cc = shade(cc, 0.86)
			}
			b.set(c.x, base-z, cc)
		}
		if k == colosGate && openFrom > 0 && openFrom < hh {
			b.set(c.x, base-openFrom+1, shade(stone, 0.45)) // the underside of the arch
		}
		// the top of the wall: the thickness seen from above, with weeds on the broken parts
		top0, top1 := int(math.Round(math.Min(c.fy, c.ey)))-hh, int(math.Round(math.Max(c.fy, c.ey)))-hh
		for y := top0; y < top1; y++ {
			cc := shade(topC, c.light*0.9)
			if c.h < 70 && itHash(c.x, y)%6 == 0 {
				cc = moss
			}
			b.set(c.x, y, cc)
		}
		if c.h < 70 && itHash(c.x, 3)%4 == 0 { // weeds sticking up from the broken top
			b.set(c.x, top0-1, moss)
			if itHash(c.x, 4)%2 == 0 {
				b.set(c.x, top0-2, mossL)
			}
		}
		if itHash(c.x, 11)%5 == 0 { // grass at the foot of the wall
			b.set(c.x, base, moss)
		}
	}
}

func init() {
	pk := propKinds

	// ---- the leaning tower ------------------------------------------------------------------------------------
	// the base is drawn left of the centre of the sprite (the top leans right): its footprint follows the base
	pk["pisa"] = &PropKind{W: pisaW, H: pisaH, FW: 44, FH: 12, hasFoot: true, FootDX: pisaBaseX - pisaW/2 - 22, FootDY: -12, Draw: drawPisa}
	// its garden: a flat piece of ground drawn under everything (the floor below is cobbles)
	pk["pisalawn"] = &PropKind{W: pisaLawnW, H: pisaLawnH, FW: 4, FH: 3, NoSolid: true, SortOff: -600, Draw: drawPisaLawn}
	overlayKinds["pisalawn"] = true

	// ---- the amphitheatre: 24 pieces of the ring plus its sand floor ----------------------------------------
	colosParts = nil
	for k := 0; k < colosN; k++ {
		kind, d := colosSegment(k)
		name := colosKind(k)
		pk[name] = kind
		overlayKinds[name] = true
		colosParts = append(colosParts, colosPart{name, d})
	}
	frx, fry := colosRX-colosT+2, colosRY-colosTY+2
	pk["colosfloor"] = &PropKind{W: int(frx*2) + 2, H: int(fry*2) + 2, FW: 4, FH: 3, NoSolid: true, SortOff: -600,
		Draw: func(b *pixBuf, v int) {
			cx, cy := frx+1, fry+1
			sand, sandD := rgb(0xc4b090), rgb(0xa49274)
			b.ellipse(cx, cy, frx, fry, sandD)
			b.ellipse(cx, cy+1, frx-3, fry-3, sand)
			for i := 0; i < 160; i++ { // rake marks
				x := int(cx-frx) + itHash(i, 3)%int(frx*2)
				y := int(cy-fry) + itHash(i, 5)%int(fry*2)
				dx, dy := (float64(x)-cx)/(frx-6), (float64(y)-cy)/(fry-5)
				if dx*dx+dy*dy < 1 {
					b.rect(x, y, 3, 1, shade(sand, 0.9))
				}
			}
			// the underground passages (the hypogeum) seen where the boards are missing, on the left of the arena
			px, py := int(cx-128), int(cy-6)
			b.rect(px, py, 64, 26, rgb(0x4a3a2c))
			for y := py + 2; y < py+26; y += 5 {
				b.rect(px, y, 64, 2, rgb(0xb89a70))
			}
			for _, x := range []int{px, px + 21, px + 42, px + 62} {
				b.rect(x, py, 2, 26, rgb(0xa88a60))
			}
		}}
	overlayKinds["colosfloor"] = true

	// ---- Venice ------------------------------------------------------------------------------------------------
	pk["venpalazzo"] = &PropKind{W: 92, H: palazzoH, FW: 90, FH: 8, Vars: 4, Draw: drawPalazzo}
	pk["gondola"] = &PropKind{W: 104, H: 34, FW: 4, FH: 3, NoSolid: true, Vars: 2, Draw: drawGondola}
	pk["palina"] = &PropKind{W: 6, H: 42, FW: 4, FH: 3, NoSolid: true, Vars: 3, Draw: func(b *pixBuf, v int) {
		stripe := []uint32{0xc83a30, 0x2a5aa8, 0x2a8a5a}[v%3]
		for y := 4; y < 40; y++ {
			cc := rgb(0xf0ece0)
			if (y/5)%2 == 0 {
				cc = rgb(stripe)
			}
			b.rect(1, y, 4, 1, cc)
			b.set(1, y, shade(cc, 1.1))
			b.set(4, y, shade(cc, 0.75))
		}
		b.rect(0, 1, 6, 3, rgb(0xd0a030)) // the gilded cap
		b.rect(1, 0, 4, 1, rgb(0xf0c850))
		b.rect(1, 36, 4, 6, rgb(0x3a4a3a)) // wet, mossy foot
	}}
	for _, n := range []string{"gondola", "palina"} {
		waterOK[n] = true
		overlayKinds[n] = true
	}

	// ---- the piazza --------------------------------------------------------------------------------------------
	pk["cypress"] = &PropKind{W: 18, H: 62, FW: 8, FH: 5, Draw: func(b *pixBuf, v int) {
		b.ellipse(9, 60, 7, 2, color.RGBA{0, 0, 0, 70})
		b.rect(8, 54, 3, 7, rgb(0x5a3a24))
		for y := 2; y < 56; y++ {
			t := float64(y-2) / 54 // a flame: narrow top, widest at two thirds
			w := 8 * math.Sin(math.Pi*math.Pow(t, 0.75)) * (1 - 0.25*t)
			for x := int(math.Round(9 - w)); x <= int(math.Round(9+w)); x++ {
				cc := rgb(0x24452a)
				if float64(x) < 9-w*0.3 {
					cc = rgb(0x36603a)
				}
				if itHash(x, y)%7 == 0 {
					cc = shade(cc, 0.8)
				}
				b.set(x, y, cc)
			}
		}
	}}
	pk["vespa"] = &PropKind{W: 34, H: 26, FW: 28, FH: 7, Vars: 3, Draw: func(b *pixBuf, v int) {
		body := rgb([]uint32{0x8ad0b8, 0xc83a30, 0xf0e2b8}[v%3])
		b.ellipse(17, 24, 15, 2, color.RGBA{0, 0, 0, 70})
		for _, wx := range []float64{7, 27} { // wheels
			b.ellipse(wx, 20, 4, 4, rgb(0x18181c))
			b.ellipse(wx, 20, 1.6, 1.6, rgb(0x9a9aa4))
		}
		b.ellipse(10, 15, 9, 5.5, body) // the round rear
		b.rect(4, 15, 20, 4, body)
		b.rect(22, 6, 5, 13, body) // leg shield
		b.ellipse(26, 18, 4, 3, body)
		b.rect(6, 8, 11, 3, rgb(0x5a3a24)) // seat
		b.rect(6, 8, 11, 1, rgb(0x7a5638))
		b.rect(25, 3, 2, 4, rgb(0x3a3a40)) // stem and handlebar
		b.rect(21, 2, 9, 2, rgb(0x2a2a30))
		b.ellipse(28, 5, 2, 2, rgb(0xfff2b0)) // headlight
		b.rect(9, 12, 6, 1, shade(body, 1.2))
		b.rect(4, 18, 20, 1, shade(body, 0.8))
	}}
	pk["gelato"] = &PropKind{W: 46, H: 54, FW: 38, FH: 10, Vars: 2, UseKind: "drink", Use: []Vec{{-11, 12}, {11, 12}}, Poison: true,
		Draw: func(b *pixBuf, v int) {
			cart := rgb([]uint32{0xf4c8d4, 0xc8e8e0}[v%2])
			b.ellipse(23, 52, 20, 2, color.RGBA{0, 0, 0, 70})
			// striped umbrella on a pole
			b.rect(22, 6, 2, 26, rgb(0xe8e8ec))
			for x := 0; x < 46; x++ {
				d := math.Abs(float64(x)-22.5) / 22.5
				top := int(math.Round(2 + 8*d*d))
				cc := rgb(0xf4f0e8)
				if (x/6)%2 == 0 {
					cc = rgb([]uint32{0xe04a6a, 0x3aa0a0}[v%2])
				}
				b.rect(x, top, 1, 12-top+int(4*d), cc)
			}
			// the cart with the tubs
			b.cube(4, 26, 38, 22, 6, rgb(0xe8eef4), cart)
			flavours := []uint32{0x9ac870, 0xf0a0b0, 0xf4e080, 0x6a3a20, 0xf4ecd0, 0x8a8ae0}
			for i, fc := range flavours {
				x := 6 + i*6
				b.rect(x, 27, 5, 4, rgb(fc))
				b.rect(x, 27, 5, 1, shade(rgb(fc), 1.15))
			}
			b.rect(8, 36, 30, 2, shade(cart, 0.8))
			b.rect(14, 39, 18, 5, rgb(0xf4f0e8)) // a cone painted on the side
			b.rect(20, 41, 6, 6, rgb(0xd8a050))
			b.ellipse(23, 40, 4, 3, rgb(0xf0a0b0))
			b.ellipse(9, 48, 3.5, 3.5, rgb(0x2a2a30))
			b.ellipse(37, 48, 3.5, 3.5, rgb(0x2a2a30))
		}}
	pk["pizzaoven"] = &PropKind{W: 60, H: 56, FW: 54, FH: 12, Trap: "explode", UseKind: "drink", Use: []Vec{{-16, 13}, {16, 13}},
		Draw: func(b *pixBuf, v int) {
			brick, brickD := rgb(0xb0583a), rgb(0x7a3a26)
			b.ellipse(30, 54, 28, 2, color.RGBA{0, 0, 0, 70})
			b.rect(44, 2, 6, 16, brickD) // chimney
			b.rect(43, 1, 8, 2, rgb(0x5a2a1a))
			b.ellipse(30, 24, 22, 16, brickD) // the dome
			b.ellipse(29, 23, 20, 14, brick)
			for y := 12; y < 36; y += 4 {
				for x := 12; x < 48; x += 8 {
					b.rect(x+(y/4%2)*4, y, 1, 3, brickD)
				}
			}
			b.ellipse(30, 30, 9, 7, rgb(0x1a0e0a)) // the mouth and the fire
			b.ellipse(30, 33, 6, 3, rgb(0xf08a20))
			b.ellipse(30, 34, 3, 1.6, rgb(0xffd060))
			// the counter in front with a pizza and boxes
			b.cube(2, 36, 56, 16, 5, rgb(0xeee6d8), rgb(0x3a7a4a))
			b.rect(2, 44, 56, 2, rgb(0xf4f0e8))
			b.rect(2, 47, 56, 2, rgb(0xc83a30))
			b.ellipse(16, 38, 7, 2.6, rgb(0xe8b060))
			b.ellipse(16, 38, 5.6, 1.9, rgb(0xd04030))
			for _, p := range [][2]int{{13, 38}, {18, 37}, {16, 39}} {
				b.set(p[0], p[1], rgb(0xf4ecd0))
			}
			b.cube(38, 31, 14, 6, 2, rgb(0xf4ecd8), rgb(0xd8c8a8))
			b.cube(39, 27, 12, 5, 2, rgb(0xf4ecd8), rgb(0xd8c8a8))
		}}
}

// the leaning tower: sprite size, the base on the ground (sprite coordinates) and the lean in pixels per row
const (
	pisaW, pisaH = 100, 248
	pisaBaseX    = 30.0  // centre of the base, from the left of the sprite
	pisaBaseY    = 244   // the ground row of the plinth
	pisaLean     = 0.105 // the top leans to the right (east)
)

// drawPisa: the leaning tower, base on the bottom row, clearly leaning to the right. Eight storeys: the plinth and
// the blind arcade of the ground floor, six open loggias (columns in front of a dark gallery, a cornice and a shadow
// under every one), the narrower belfry with its big openings and a small lead dome with the flag. White-cream marble
// lit from the left, blue-grey shadows on the right and a dark rim on both edges, so it reads against the cobbles.
func drawPisa(b *pixBuf, v int) {
	marble := rgb(0xf2ece0)
	cool := rgb(0x7c8496) // blue-grey shadow side
	rim := rgb(0x4a5060)
	hole := rgb(0x23242e)  // inside the galleries
	holeL := rgb(0x3e404c) // the inner wall seen through the arches, a little lit
	lead := rgb(0x8a92a2)
	b.ellipse(pisaBaseX+26, pisaBaseY+1, 40, 3, color.RGBA{0, 0, 0, 70}) // the shadow falls to the right
	b.ellipse(pisaBaseX+6, pisaBaseY+1, 30, 3, color.RGBA{0, 0, 0, 60})
	cxAt := func(z int) float64 { return pisaBaseX + float64(z)*pisaLean }
	// lit: the colour of the marble at u (-1 left edge .. +1 right edge) times k
	lit := func(u, k float64) color.RGBA {
		t := math.Pow(clampF((u+0.25)/1.25, 0, 1), 1.3)
		cc := mixc(marble, cool, t*0.85)
		if u < -0.6 {
			cc = shade(cc, 1.04)
		}
		return shade(cc, k)
	}
	type tierT struct {
		z0, z1 int
		r      float64
		kind   int // 0 plinth, 1 ground floor (blind arcade), 2 loggia, 3 belfry, 4 dome
	}
	tiers := []tierT{{0, 3, 27, 0}, {3, 6, 25.5, 0}, {6, 46, 24, 1}}
	for i := 0; i < 6; i++ {
		tiers = append(tiers, tierT{46 + i*25, 71 + i*25, 24, 2})
	}
	tiers = append(tiers, tierT{196, 224, 17, 3}, tierT{224, 236, 12, 4})
	for _, tr := range tiers {
		for z := tr.z0; z < tr.z1; z++ {
			y := pisaBaseY - z
			cx := cxAt(z)
			r := tr.r
			if tr.kind == 4 { // the little dome
				f := float64(z-tr.z0) / float64(tr.z1-tr.z0)
				r = math.Max(1.5, tr.r*math.Sqrt(math.Max(0, 1-f*f)))
			}
			row := tr.z1 - 1 - z // rows from the top of the tier
			x0, x1 := int(math.Floor(cx-r)), int(math.Ceil(cx+r))
			for x := x0; x <= x1; x++ {
				u := (float64(x) + 0.5 - cx) / r
				if u < -1.02 || u > 1.02 {
					continue
				}
				u = clampF(u, -1, 1)
				ang := math.Asin(u) // angle on the cylinder
				cc := lit(u, 1)
				switch tr.kind {
				case 0: // plinth steps
					cc = lit(u, 0.86)
					if z == tr.z1-1 {
						cc = lit(u, 1.02)
					}
				case 1: // ground floor: tall blind arches with a dark outline, lozenges under the arch tops, the door
					f := math.Mod(ang/(2*math.Pi/12)+12.5, 1)
					in := row >= 6 && row <= 36 && f > 0.16 && f < 0.84
					if in && row <= 9 { // round top
						w := float64(10-row) * 0.09
						in = f > 0.16+w && f < 0.84-w
					}
					edge := in && (f < 0.24 || f > 0.76 || row <= 7)
					switch {
					case row <= 1:
						cc = lit(u, 1.06)
					case row == 2:
						cc = lit(u, 0.72)
					case edge:
						cc = lit(u, 0.62)
					case in && row >= 11 && row <= 15 && math.Abs(f-0.5) < 0.12 && (row+int(f*20))%2 == 0:
						cc = shade(cool, 0.75) // the inlaid lozenge
					case in:
						cc = lit(u, 0.86)
					}
					if math.Abs(u) < 0.13 && row >= 16 { // the door
						cc = shade(hole, 1.1)
						if row == 16 {
							cc = lit(u, 0.6)
						}
					}
				case 2: // loggia: columns in front of the dark gallery
					f := math.Mod(ang/(2*math.Pi/16)+16.5, 1)
					open := row >= 5 && row <= 20 && f > 0.3 && f < 0.86
					if open && row <= 7 { // round arch tops
						w := float64(8-row) * 0.08
						open = f > 0.3+w && f < 0.86-w
					}
					switch {
					case row == 0:
						cc = lit(u, 1.1)
					case row == 1:
						cc = lit(u, 1.0)
					case row == 2 || row == 3:
						cc = lit(u, 0.66) // the shadow under the cornice
					case open:
						cc = hole
						if row <= 9 {
							cc = holeL // the vault inside, a little lighter
						}
						if f < 0.36 { // the lit edge of the column on the left of the opening
							cc = mixc(hole, lit(u, 0.9), 0.35)
						}
					case row >= 21:
						cc = lit(u, 0.9) // the balustrade band
						if row == 21 {
							cc = lit(u, 0.74)
						}
					default:
						if f >= 0.86 || f <= 0.3 { // the column: a bright line on its lit side
							if f > 0.9 && f < 0.97 {
								cc = lit(u, 1.06)
							}
						}
					}
				case 3: // belfry: big openings and the bells
					f := math.Mod(ang/(2*math.Pi/10)+10.5, 1)
					open := row >= 5 && row <= 22 && f > 0.25 && f < 0.82
					if open && row <= 8 {
						w := float64(9-row) * 0.07
						open = f > 0.25+w && f < 0.82-w
					}
					switch {
					case row <= 1:
						cc = lit(u, 1.08)
					case row <= 3:
						cc = lit(u, 0.68)
					case open:
						cc = hole
						if row >= 10 && row <= 14 && math.Abs(f-0.535) < 0.12 && math.Abs(u) < 0.5 {
							cc = rgb(0x6a6458) // a dull bronze bell in the middle opening
						}
					}
				case 4:
					t := clampF((u+0.4)/1.4, 0, 1)
					cc = shade(mixc(lead, shade(lead, 0.55), t), 1)
					if u < -0.3 && row < 4 {
						cc = shade(lead, 1.15)
					}
				}
				// a dark rim on both edges: the tower never melts into the cobbles
				if tr.kind != 4 && (x == x0 || x == x1) {
					if x == x1 {
						cc = rim
					} else {
						cc = shade(cc, 0.8)
					}
				}
				b.set(x, y, cc)
			}
			// cornices stick out one pixel at the top of every storey
			if (tr.kind == 1 || tr.kind == 2 || tr.kind == 3) && row <= 1 {
				b.set(x0-1, y, lit(-1, 1.08))
				b.set(x1+1, y, rim)
			}
		}
	}
	// the lantern on the dome and the flag
	tx := int(math.Round(cxAt(236)))
	b.rect(tx-1, pisaBaseY-238, 3, 2, lead)
	b.rect(tx, pisaBaseY-244, 1, 6, rim)
	b.rect(tx+1, pisaBaseY-244, 4, 2, rgb(0xb83a32))
	b.rect(tx+1, pisaBaseY-242, 3, 1, rgb(0x8a2a24))
}

// drawPisaLawn: the garden of the tower, a flat ground piece under everything: an organic lawn (no hard rectangle)
// with a low stone kerb, clipped hedges along part of the edge, gravel paths to the tower and a marble ring around
// its base. Sprite coordinates: the tower stands at (pisaLawnTX, pisaLawnTY).
const (
	pisaLawnW, pisaLawnH   = 208, 214
	pisaLawnTX, pisaLawnTY = 84, 160
)

func drawPisaLawn(b *pixBuf, v int) {
	cx, cy := 104.0, 104.0
	rx, ry := 100.0, 106.0
	// the radius of the lawn at angle a: a wobbly, irregular outline
	rad := func(a float64) float64 {
		return 0.86 + 0.06*math.Sin(3*a+0.7) + 0.04*math.Sin(5*a+2.1) + 0.03*math.Sin(9*a)
	}
	inside := func(x, y float64, k float64) bool {
		dx, dy := (x-cx)/rx, (y-cy)/ry
		a := math.Atan2(dy, dx)
		return math.Hypot(dx, dy) < rad(a)*k
	}
	grass := []color.RGBA{rgb(0x3e6636), rgb(0x46703a), rgb(0x385e32), rgb(0x4e7a40)}
	gravel, gravelD := rgb(0xb4aa96), rgb(0x8e8676)
	kerb, kerbD := rgb(0xc8c2b4), rgb(0x6e6a62)
	hedge, hedgeL, hedgeD := rgb(0x2a4a2c), rgb(0x3e6a3c), rgb(0x1a3020)
	tx, ty := float64(pisaLawnTX), float64(pisaLawnTY)
	onPath := func(x, y float64) bool {
		// the main path from the bottom to the base, one to the upper left and one to the right
		if math.Abs(x-tx-2) < 7 && y > ty {
			return true
		}
		d1 := math.Abs((y-ty)-(x-tx)*0.9) / 1.35 // towards the upper left
		if x < tx && d1 < 4 {
			return true
		}
		d2 := math.Abs((y-ty)+(x-tx)*0.45) / 1.1 // towards the right
		if x > tx && d2 < 4 {
			return true
		}
		// the marble ring around the base
		e := math.Hypot((x-tx)/40, (y-ty)/13)
		return e < 1
	}
	for y := 0; y < pisaLawnH; y++ {
		for x := 0; x < pisaLawnW; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if !inside(fx, fy, 1) {
				continue
			}
			kerbRow := !inside(fx, fy, 0.965)
			var cc color.RGBA
			switch {
			case kerbRow:
				cc = kerb
				if !inside(fx, fy+2, 1) {
					cc = kerbD // the lower face of the kerb
				}
			case onPath(fx, fy):
				cc = gravel
				if itHash(x, y)%5 == 0 {
					cc = gravelD
				}
				if e := math.Hypot((fx-tx)/40, (fy-ty)/13); e < 1 {
					cc = shade(rgb(0xd8d2c4), 0.92) // marble paving
					if e > 0.9 {
						cc = kerbD
					}
				}
			default:
				cc = grass[(itHash(x/2, y/2)+itHash(x, y)%2)%len(grass)]
				if (x+y*3)%11 == 0 && itHash(x, y)%3 == 0 {
					cc = shade(cc, 1.18) // little blades catching the light
				}
			}
			b.set(x, y, cc)
		}
	}
	// path edges: a thin darker line where gravel meets grass
	for y := 1; y < pisaLawnH-1; y++ {
		for x := 1; x < pisaLawnW-1; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if !inside(fx, fy, 0.96) || onPath(fx, fy) {
				continue
			}
			if onPath(fx, fy-1) || onPath(fx-1, fy) || onPath(fx+1, fy) {
				b.set(x, y, shade(grass[2], 0.7))
			}
		}
	}
	// clipped hedges along the back and the sides of the lawn (not on the paths)
	for a := 0.0; a < 2*math.Pi; a += 0.012 {
		if s := math.Sin(a); s > 0.35 { // leave the front open
			continue
		}
		k := rad(a) * 0.9
		x := cx + rx*k*math.Cos(a)
		y := cy + ry*k*math.Sin(a)
		if onPath(x, y) || onPath(x, y+3) || onPath(x, y-3) {
			continue
		}
		ix, iy := int(x), int(y)
		b.rect(ix, iy-3, 1, 4, hedge)
		b.set(ix, iy-4, hedgeL)
		b.set(ix, iy+1, hedgeD)
		if itHash(ix, iy)%4 == 0 {
			b.set(ix, iy-3, hedgeL)
		}
	}
	// a few flowers in the grass (muted)
	for i := 0; i < 26; i++ {
		x, y := itHash(i, 41)%pisaLawnW, itHash(i, 43)%pisaLawnH
		if inside(float64(x), float64(y), 0.85) && !onPath(float64(x), float64(y)) && !onPath(float64(x), float64(y)+2) {
			b.set(x, y, []color.RGBA{rgb(0xd8d0e0), rgb(0xc86a6a), rgb(0xe0dcc8)}[i%3])
		}
	}
}

// drawPalazzo: a Venetian palace seen from the canal: coloured plaster, pointed windows with green shutters, a
// balcony on the noble floor, a water door at the bottom and funnel chimneys on the roof.
func drawPalazzo(b *pixBuf, v int) {
	const B = palazzoH                                                 // the bottom row (the waterline)
	wall := rgb([]uint32{0xc8a060, 0xa45a44, 0xddd3c0, 0xc08a76}[v%4]) // ochre, brick, cream, faded pink
	shutter := rgb(0x3a7a4a)
	stoneW := rgb(0xf0ece0)
	top := 16 + (v%2)*8
	b.rect(0, top, 92, B-top, wall)
	for y := top; y < B; y++ { // plaster: a light left edge and a darker right one
		b.set(0, y, shade(wall, 1.1))
		b.set(91, y, shade(wall, 0.75))
		if itHash(y, v)%5 == 0 {
			b.rect(itHash(y, v+3)%88, y, 3, 1, shade(wall, 0.92))
		}
	}
	b.rect(0, top-3, 92, 3, rgb(0xa04a30)) // roof tiles edge
	b.rect(0, top-1, 92, 1, rgb(0x7a3220))
	b.rect(0, top, 92, 2, shade(stoneW, 0.9)) // cornice
	for _, x := range []int{14, 70} {         // funnel chimneys
		b.rect(x, top-12, 4, 9, rgb(0xb06a4a))
		b.rect(x-2, top-15, 8, 4, rgb(0x8a4a30))
	}
	window := func(x, y, w, h int, balcony bool) {
		b.rect(x-1, y-1, w+2, h+2, shade(stoneW, 0.95))
		b.rect(x, y, w, h, rgb(0x2a2420))
		b.set(x, y, shade(stoneW, 0.95)) // pointed (gothic) top
		b.set(x+w-1, y, shade(stoneW, 0.95))
		b.rect(x-3, y, 2, h, shutter) // shutters
		b.rect(x+w+1, y, 2, h, shutter)
		b.rect(x-3, y, 2, 1, shade(shutter, 1.3))
		if balcony {
			b.rect(x-4, y+h, w+8, 2, stoneW)
			for bx := x - 4; bx < x+w+4; bx += 2 {
				b.set(bx, y+h-2, stoneW)
				b.set(bx, y+h-1, stoneW)
			}
		}
	}
	for i := 0; i < 4; i++ {
		x := 9 + i*21
		window(x, top+9, 6, 11, false)
		window(x, top+31, 6, 13, i == 1 || i == 2) // the noble floor
		window(x, top+56, 6, 11, false)
	}
	b.rect(28, top+47, 36, 2, stoneW) // a long balcony under the noble floor
	// water door and the wet, greenish foot of the wall
	b.rect(38, B-20, 16, 16, rgb(0x2a2420))
	b.ellipse(46, float64(B-20), 8, 4, rgb(0x2a2420))
	b.rect(38, B-20, 1, 16, stoneW)
	b.rect(53, B-20, 1, 16, stoneW)
	b.rect(0, B-6, 92, 6, mixc(wall, rgb(0x3a5a40), 0.6))
	b.rect(0, B-2, 92, 2, rgb(0x2a3a2a))
	for _, x := range []int{8, 74} { // a flower box on the ground floor windows
		b.rect(x, B-18, 8, 7, rgb(0x2a2420))
		b.rect(x-1, B-11, 10, 2, rgb(0x8a4a30))
		b.set(x+1, B-12, rgb(0xe04a5a))
		b.set(x+4, B-12, rgb(0xf0d030))
		b.set(x+6, B-12, rgb(0xe04a5a))
	}
}

// drawGondola: a black gondola seen from the quay; var 1 points the other way and has its gondolier.
func drawGondola(b *pixBuf, v int) {
	const W = 104
	flip := func(x int) int {
		if v%2 == 1 {
			return W - 1 - x
		}
		return x
	}
	set := func(x, y int, cc color.RGBA) { b.set(flip(x), y, cc) }
	b.ellipse(52, 31, 46, 3, color.RGBA{0, 30, 40, 90}) // its shadow on the water
	hull, hullL := rgb(0x07070b), rgb(0x4a4a5c)         // black lacquer
	for x := 0; x < W; x++ {
		d := (float64(x) - 50) / 50 // -1 stern ... +1 prow
		top := 22 - 13*math.Pow(math.Abs(d), 4)
		if d > 0.9 { // the prow rises higher
			top -= (d - 0.9) * 30
		}
		bot := 29 - 3*d*d
		for y := int(math.Round(top)); y <= int(math.Round(bot)); y++ {
			cc := hull
			if y == int(math.Round(top)) {
				cc = hullL
			}
			set(x, y, cc)
		}
	}
	// the deck: dark wood, the red seat with gold trim
	for x := 26; x < 78; x++ {
		set(x, 21, rgb(0x2a1c14))
	}
	for x := 44; x < 58; x++ {
		set(x, 18, rgb(0xb02030))
		set(x, 19, rgb(0xb02030))
		set(x, 20, rgb(0xd0a030))
	}
	set(44, 17, rgb(0xd0a030))
	set(57, 17, rgb(0xd0a030))
	// the silver comb at the prow
	for i := 0; i < 6; i++ {
		set(98, 6+i*2, rgb(0xd8dce4))
		set(99, 6+i*2, rgb(0xd8dce4))
		set(100, 6+i*2, rgb(0xb8bcc4))
	}
	set(99, 4, rgb(0xe8ecf4))
	set(100, 5, rgb(0xe8ecf4))
	if v%2 == 1 { // the gondolier at the stern: striped shirt, straw hat and the long oar
		for y := 3; y < 21; y++ {
			cc := rgb(0xf0ece0)
			if y >= 6 && y <= 13 && (y/2)%2 == 0 {
				cc = rgb(0x1a2a5a)
			}
			if y > 13 {
				cc = rgb(0x1a1a22)
			}
			set(14, y, cc)
			set(15, y, cc)
		}
		set(14, 2, rgb(0xe8c890))
		set(15, 2, rgb(0xe8c890))
		for x := 12; x < 18; x++ {
			set(x, 1, rgb(0xe8d070))
		}
		set(13, 0, rgb(0xc83a30))
		b.line(flip(16), 9, flip(30), 30, rgb(0x8a6a40))
	}
}
