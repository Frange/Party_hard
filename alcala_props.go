package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Props of ALCALÁ DE HENARES (alcala.go). All drawn by code in the muted dusk palette of the game:
//   alc_kiosk       the octagonal bandstand: platform, steps, railing and posts (flat, under everybody)
//   alc_kioskroof   its slate roof (anchored at the front of the eaves: the DJ stays in sight below it, whoever walks
//                   behind the bandstand passes behind the roof)
//   alc_monument    the statue on its stepped pedestal, in a flower bed with little white fences
//   alc_bronzebench two bronze figures sitting on a bench
//   alc_stall       market stalls: 0 striped awning with prints, 1 ceramics and lanterns under a tiled roof
//   alc_pottree     a round tree in a stone planter
//   alc_bin         a green litter bin
//   alc_stork       a stork in its nest (animated: it preens, clatters its beak and flaps now and then)
// ---------------------------------------------------------------------------

const (
	alcKioskRX    = 62.0 // the platform's octagon (half width / half depth)
	alcKioskRY    = 28.0
	alcKioskFront = 44.0 // the anchor of the platform prop: this far below the centre (the foot of the steps)
	alcKioskPost  = 66   // height of the posts: the eaves stay above the DJ's head
	alcKioskW     = 140
	alcKioskH     = 142
)

// alcOct: (dx, dy) inside the octagon of half sizes rx, ry.
func alcOct(dx, dy, rx, ry float64) bool {
	u, v := math.Abs(dx)/rx, math.Abs(dy)/ry
	return u <= 1 && v <= 1 && (u+v)/1.36 <= 1
}

// alcOctVerts: the 8 corners of the octagon (back ones first).
func alcOctVerts(rx, ry float64) [][2]float64 {
	k := 0.36
	return [][2]float64{{-rx * (1 - k), -ry}, {rx * (1 - k), -ry}, {-rx, -ry * (1 - k)}, {rx, -ry * (1 - k)},
		{-rx, ry * (1 - k)}, {rx, ry * (1 - k)}, {-rx * (1 - k), ry}, {rx * (1 - k), ry}}
}

func init() {
	pk := propKinds

	pk["alc_kiosk"] = &PropKind{W: alcKioskW, H: alcKioskH, FW: 4, FH: 3, NoSolid: true, SortOff: -600, Draw: drawAlcKiosk}
	pk["alc_kioskroof"] = &PropKind{W: alcKioskW, H: 78, FW: 4, FH: 3, NoSolid: true, Draw: drawAlcKioskRoof}
	pk["alc_monument"] = &PropKind{W: 104, H: 100, FW: 96, FH: 36, Draw: drawAlcMonument}
	pk["alc_bronzebench"] = &PropKind{W: 52, H: 38, FW: 48, FH: 10, Draw: drawAlcBronzeBench}
	pk["alc_stall"] = &PropKind{W: 68, H: 58, FW: 64, FH: 16, Vars: 2, Hide: true, Draw: drawAlcStall}
	pk["alc_pottree"] = &PropKind{W: 44, H: 70, FW: 30, FH: 14, Draw: drawAlcPotTree}
	pk["alc_bin"] = &PropKind{W: 12, H: 16, FW: 8, FH: 5, Draw: drawAlcBin}
	pk["alc_stork"] = &PropKind{W: 28, H: 26, FW: 4, FH: 3, NoSolid: true, Vars: 2, Draw: func(b *pixBuf, v int) { drawAlcStork(b, v, 0) }}
	for _, n := range []string{"alc_kiosk", "alc_kioskroof", "alc_stork"} {
		overlayKinds[n] = true
	}
	propAnims["alc_stork"] = &propAnim{n: 5, fps: 1, frame: alcStorkFrame,
		draw: func(b *pixBuf, k *PropKind, v, f int) { drawAlcStork(b, v, f) }}
}

// alcStorkFrame: most of the time the stork stands still (frame 0); every ~9 s (its own phase) it bows to preen (1),
// clatters its beak pointing up (2, 0, 2) and opens its wings (3, 4, 4, 3).
func alcStorkFrame(t, ph float64, busy bool) int {
	c := math.Mod(t+ph*0.37, alcStorkCycle)
	if c < 0 {
		c += alcStorkCycle
	}
	switch {
	case c < 6.2:
		return 0
	case c < 7.0: // the beak up and down: clatter
		return []int{2, 0, 2, 0}[int((c-6.2)/0.2)%4]
	case c < 7.6:
		return 1
	case c < 8.6: // a flap
		return []int{3, 4, 4, 3, 4}[int((c-7.6)/0.2)%5]
	}
	return 0
}

// alcStorkCycle: seconds between two bouts of movement of a stork.
const alcStorkCycle = 9.0

// ---- the bandstand ---------------------------------------------------------------------------------------------

func drawAlcKiosk(b *pixBuf, v int) {
	cx, cy := float64(alcKioskW)/2, float64(alcKioskH)-alcKioskFront
	stone, stoneD, stoneL := rgb(0xc8b496), rgb(0x8a7a66), rgb(0xe4d4b8)
	floor, floorD := rgb(0xb89a78), rgb(0x9a7e60)
	iron := rgb(0x2e2c38)
	rx, ry := alcKioskRX, alcKioskRY
	b.ellipse(cx, cy+12, rx+4, ry+6, color.RGBA{0, 0, 0, 60})
	// the stone base (the platform stands 8 px high) and the boards on top
	for y := int(cy - ry - 1); y <= int(cy+ry+9); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			switch {
			case alcOct(dx, dy, rx, ry):
				c := floor
				if (y+int(dx/3))%4 == 0 {
					c = floorD
				}
				if !alcOct(dx, dy, rx-3, ry-2) {
					c = stoneL // the stone rim of the platform
				}
				b.set(x, y, c)
			case alcOct(dx, dy-8, rx, ry):
				c := stone
				if dx > rx*0.3 {
					c = shade(stone, 0.85)
				}
				if (x/7)%2 == 0 && dy-8 > 0 {
					b.set(x, y, shade(c, 0.95))
				} else {
					b.set(x, y, c)
				}
			}
		}
	}
	// the steps at the front
	for i := 0; i < 4; i++ {
		y := int(cy+ry) + i*4
		w := 34 + i*4
		b.rect(int(cx)-w/2, y, w, 2, stoneL)
		b.rect(int(cx)-w/2, y+2, w, 2, stoneD)
	}
	// the railing all round except over the steps, and the posts at the corners up to the eaves
	for x := int(cx - rx); x <= int(cx+rx); x++ {
		dx := float64(x) + 0.5 - cx
		for _, s := range []float64{-1, 1} {
			// the edge of the octagon at this column (search from the centre outwards)
			ey := 0.0
			for dy := 0.0; dy <= ry; dy++ {
				if alcOct(dx, s*dy, rx, ry) {
					ey = s * dy
				}
			}
			if s > 0 && math.Abs(dx) < 20 {
				continue // the opening over the steps
			}
			y := int(cy + ey)
			b.set(x, y-7, iron)
			if x%3 == 0 {
				b.rect(x, y-6, 1, 6, iron)
			}
		}
	}
	for _, p := range alcOctVerts(rx, ry) {
		x, y := int(math.Round(cx+p[0])), int(math.Round(cy+p[1]))
		b.rect(x-1, y-alcKioskPost, 2, alcKioskPost, iron)
		b.set(x-1, y-alcKioskPost, shade(iron, 1.6))
		b.rect(x-2, y-2, 4, 2, iron)
	}
	// flower pots either side of the steps
	for _, s := range []float64{-1, 1} {
		x := int(cx + s*30)
		y := int(cy+ry) + 8
		b.rect(x-4, y, 8, 6, rgb(0xa85a3a))
		b.ellipse(float64(x), float64(y-1), 5, 3, rgb(0x3a6a3a))
		b.set(x-2, y-2, rgb(0xe05a6a))
		b.set(x+1, y-3, rgb(0xf0c040))
	}
}

func drawAlcKioskRoof(b *pixBuf, v int) {
	const W, H = alcKioskW, 78
	cx := float64(W) / 2
	eaveY := float64(H) - 2 - alcKioskRY // the centre of the eaves (their front edge is the anchor)
	rx, ry := alcKioskRX+4, alcKioskRY
	slate, slateD, slateL := rgb(0x4a4a5c), rgb(0x30303e), rgb(0x6a6a80)
	apexY := eaveY - 42
	for i := 0; i <= 80; i++ { // the pyramid: octagons from the apex down to the eaves
		t := float64(i) / 80
		yc := apexY + (eaveY-apexY)*t
		for y := int(yc - ry*t - 1); y <= int(yc+ry*t+1); y++ {
			for x := int(cx - rx*t - 1); x <= int(cx+rx*t+1); x++ {
				dx, dy := float64(x)+0.5-cx, float64(y)+0.5-yc
				if t == 0 || !alcOct(dx, dy, math.Max(rx*t, 0.5), math.Max(ry*t, 0.5)) {
					continue
				}
				c := slate
				if dx < -rx*t*0.35 {
					c = slateL
				} else if dx > rx*t*0.35 {
					c = slateD
				}
				if int(yc)%5 == 0 {
					c = shade(c, 0.9)
				}
				b.set(x, y, c)
			}
		}
	}
	// the ribs from the apex to the front corners and the trimmed eaves with a fringe
	for _, p := range alcOctVerts(rx, ry)[4:] {
		b.line(int(cx), int(apexY), int(math.Round(cx+p[0])), int(math.Round(eaveY+p[1])), slateD)
	}
	for x := int(cx - rx); x <= int(cx+rx); x++ {
		dx := float64(x) + 0.5 - cx
		ey := 0.0
		for dy := 0.0; dy <= ry; dy++ {
			if alcOct(dx, dy, rx, ry) {
				ey = dy
			}
		}
		y := int(eaveY + ey)
		b.rect(x, y-1, 1, 2, rgb(0xd8c8a0))
		if x%4 == 0 {
			b.rect(x, y+1, 1, 2, rgb(0xb8a070))
		}
	}
	// the finial
	b.rect(int(cx)-1, int(apexY)-8, 2, 9, rgb(0x2e2c38))
	b.ellipse(cx, apexY-8, 2, 2, rgb(0xc8a040))
}

// ---- the statue in its flower bed -----------------------------------------------------------------------------

func drawAlcMonument(b *pixBuf, v int) {
	const W, H = 104, 100
	stone, stoneD, stoneL := rgb(0xc8b8a0), rgb(0x8a7c6a), rgb(0xe6d8c0)
	bronze, bronzeL, bronzeD := rgb(0x4a5a50), rgb(0x6e8070), rgb(0x2c3630)
	b.ellipse(W/2, H-3, 52, 6, color.RGBA{0, 0, 0, 60})
	// the flower bed: a stone kerb round a garden
	b.cube(4, 62, W-8, H-62, 32, rgb(0x3e6a3a), stone)
	b.rect(4, 62, W-8, 2, stoneL)
	b.rect(4, 62, 2, 32, stone)
	b.rect(W-6, 62, 2, 32, stoneD)
	for i := 0; i < 120; i++ { // flowers
		x := 8 + itHash(i, 41)%(W-16)
		y := 66 + itHash(i, 43)%26
		c := []uint32{0xe05a6a, 0xf0c040, 0xe890c8, 0xf4f0e8, 0x5a9a4a}[i%5]
		b.set(x, y, rgb(c))
		if i%3 == 0 {
			b.set(x+1, y, shade(rgb(c), 0.8))
		}
	}
	// the little white fences round the bed
	fence := rgb(0xf0ece0)
	for x := 6; x < W-6; x += 5 {
		b.rect(x, 58, 1, 5, fence)
		b.rect(x, 88, 1, 5, fence)
	}
	b.rect(6, 59, W-12, 1, fence)
	b.rect(6, 89, W-12, 1, fence)
	for y := 60; y < 92; y += 5 {
		b.rect(6, y, 1, 4, fence)
		b.rect(W-7, y, 1, 4, fence)
	}
	// the stepped pedestal
	b.cube(W/2-20, 66, 40, 22, 6, stoneL, stone)
	b.cube(W/2-14, 44, 28, 26, 5, stoneL, stone)
	b.rect(W/2-8, 54, 16, 8, stoneD) // the carved plaque
	b.rect(W/2-6, 56, 12, 4, rgb(0xb0a084))
	b.rect(W/2-16, 42, 32, 3, stoneL) // the cornice
	// the bronze figure: a scholar in a long coat with a book
	cx := W / 2
	b.rect(cx-5, 34, 4, 9, bronzeD) // legs
	b.rect(cx+1, 34, 4, 9, bronze)
	b.rect(cx-7, 18, 14, 18, bronze) // the coat
	b.rect(cx-7, 18, 4, 18, bronzeL)
	b.rect(cx+4, 18, 3, 18, bronzeD)
	b.rect(cx-8, 32, 16, 4, bronze)            // its skirt
	b.rect(cx-6, 17, 12, 3, bronzeL)           // the collar
	b.ellipse(float64(cx), 12, 4, 4.5, bronze) // the head
	b.rect(cx-3, 10, 3, 3, bronzeL)
	b.rect(cx+7, 20, 3, 10, bronzeD) // the right arm down with the book
	b.rect(cx+6, 29, 6, 5, bronzeL)
	b.rect(cx-11, 16, 4, 4, bronze) // the left arm raised, a quill in the hand
	b.rect(cx-12, 10, 3, 7, bronze)
	b.line(cx-12, 9, cx-15, 3, bronzeL)
}

// two bronze figures sitting on a bench (on a low stone plinth)
func drawAlcBronzeBench(b *pixBuf, v int) {
	stone, stoneD := rgb(0xb8aa94), rgb(0x847866)
	bronze, bronzeL, bronzeD := rgb(0x4a5a50), rgb(0x6e8070), rgb(0x2c3630)
	b.ellipse(26, 36, 26, 3, color.RGBA{0, 0, 0, 60})
	b.cube(1, 26, 50, 12, 5, rgb(0xd0c4ae), stone)
	b.rect(1, 36, 50, 2, stoneD)
	b.rect(6, 12, 40, 3, bronzeD) // the bench back
	b.rect(6, 20, 40, 3, bronze)  // the seat
	b.rect(8, 23, 2, 4, bronzeD)
	b.rect(42, 23, 2, 4, bronzeD)
	for i, x := range []int{14, 34} {
		b.rect(x-4, 8, 8, 13, bronze) // the body
		b.rect(x-4, 8, 3, 13, bronzeL)
		b.ellipse(float64(x), 5, 3.5, 4, bronze) // the head
		b.set(x-1, 4, bronzeL)
		b.rect(x-4, 20, 8, 3, bronzeL) // the lap
		b.rect(x-4, 23, 3, 5, bronzeD) // the legs down
		b.rect(x+1, 23, 3, 5, bronzeD)
		if i == 0 { // the first one holds a hat, the second an open book
			b.ellipse(float64(x+6), 16, 3, 1.6, bronzeD)
		} else {
			b.rect(x-3, 15, 7, 3, bronzeL)
			b.set(x, 16, bronzeD)
		}
	}
}

// ---- the market stalls -----------------------------------------------------------------------------------------

func drawAlcStall(b *pixBuf, v int) {
	const W, H = 68, 58
	wood, woodD := rgb(0x8a5e3a), rgb(0x5e3e26)
	b.ellipse(W/2, H-2, 33, 3, color.RGBA{0, 0, 0, 60})
	for _, x := range []int{3, W - 5} { // the posts
		b.rect(x, 14, 2, H-18, woodD)
	}
	// the counter
	b.cube(2, 34, W-4, H-36, 8, rgb(0xa8784a), wood)
	b.rect(2, 48, W-4, 1, woodD)
	if v%2 == 0 {
		// the striped awning (red and white) with a checked hem, and framed prints on the counter
		for x := 0; x < W; x++ {
			top := 6 + int(4*math.Abs(float64(x)-W/2)/(W/2))
			c := rgb(0xf0ece0)
			if (x/6)%2 == 0 {
				c = rgb(0xc8403a)
			}
			b.rect(x, top, 1, 18-top, c)
			b.set(x, top, shade(c, 1.1))
			if (x/3+1)%2 == 0 {
				b.rect(x, 18, 1, 2, rgb(0xc8403a))
			} else {
				b.rect(x, 18, 1, 2, rgb(0xf0ece0))
			}
		}
		for i := 0; i < 6; i++ {
			x := 6 + i*10
			c := []uint32{0x5a7ab0, 0xd8a050, 0x6a9a5a, 0xc86a5a, 0xe6dcc0, 0x8a6aa0}[i]
			b.rect(x, 33, 8, 7, rgb(0x3a2a20))
			b.rect(x+1, 34, 6, 5, rgb(c))
		}
		for i := 0; i < 3; i++ { // prints hanging from the awning
			x := 12 + i*18
			b.rect(x, 21, 8, 9, rgb(0x3a2a20))
			b.rect(x+1, 22, 6, 7, rgb([]uint32{0xe6dcc0, 0xa8c0d8, 0xe8b8a0}[i]))
		}
		return
	}
	// ceramics and lanterns under a small tiled roof
	for y := 2; y < 15; y++ {
		hw := (y - 2) * (W / 2) / 12
		c := mixc(alcRoof, alcRoofD, float64(y-2)/13)
		b.rect(W/2-hw, y, 2*hw, 1, c)
		if y%3 == 0 {
			b.rect(W/2-hw, y, 2*hw, 1, shade(c, 0.85))
		}
	}
	b.rect(0, 14, W, 2, alcRoofD)
	for i := 0; i < 4; i++ { // the lanterns, lit
		x := 8 + i*16
		b.rect(x+2, 16, 1, 3, rgb(0x2a2630))
		b.rect(x, 19, 5, 7, rgb(0x6a4a2a))
		b.rect(x+1, 20, 3, 5, rgb(0xf0c060))
	}
	for i := 0; i < 5; i++ { // plates and jars on the counter
		x := 8 + i*12
		if i%2 == 0 {
			b.ellipse(float64(x), 36, 5, 3, rgb(0xf0ece0))
			b.ellipse(float64(x), 36, 3, 1.6, rgb(0x3a5aa8))
		} else {
			b.rect(x-3, 30, 6, 8, rgb(0x3a5aa8))
			b.rect(x-2, 29, 4, 1, rgb(0xf0ece0))
			b.rect(x-3, 33, 6, 1, rgb(0xf0c040))
		}
	}
}

// ---- trees, bins -------------------------------------------------------------------------------------------------

func drawAlcPotTree(b *pixBuf, v int) {
	stone, stoneL := rgb(0xb8aa94), rgb(0xd8ccb4)
	b.ellipse(22, 68, 18, 3, color.RGBA{0, 0, 0, 60})
	b.rect(20, 34, 4, 24, rgb(0x5a3a24)) // the trunk
	b.rect(20, 34, 1, 24, rgb(0x7a5638))
	b.ellipse(22, 24, 20, 19, rgb(0x2e5a34)) // the crown
	b.ellipse(19, 21, 15, 14, rgb(0x3e7440))
	b.ellipse(16, 17, 8, 7, rgb(0x5a9050))
	for i := 0; i < 26; i++ { // leaves and a few oranges
		x, y := 4+itHash(i, 61)%36, 8+itHash(i, 63)%30
		dx, dy := float64(x)-22, float64(y)-24
		if dx*dx/400+dy*dy/361 > 0.9 {
			continue
		}
		if i%4 == 0 {
			b.set(x, y, rgb(0xe8902a))
			continue
		}
		b.set(x, y, shade(rgb(0x2e5a34), 0.8))
	}
	b.cube(7, 52, 30, 18, 6, rgb(0x6a4a2a), stone) // the stone planter with earth and flowers
	b.rect(7, 52, 30, 1, stoneL)
	for x := 9; x < 36; x += 4 {
		b.set(x, 54, rgb([]uint32{0xe05a6a, 0xf0c040, 0xe890c8}[x%3]))
	}
}

func drawAlcBin(b *pixBuf, v int) {
	g, gD, gL := rgb(0x2e5a3a), rgb(0x1e3e28), rgb(0x4a7a52)
	b.ellipse(6, 15, 5, 1.6, color.RGBA{0, 0, 0, 60})
	b.rect(2, 3, 8, 12, g)
	b.rect(2, 3, 2, 12, gL)
	b.rect(8, 3, 2, 12, gD)
	b.rect(1, 2, 10, 2, gD)
	b.rect(3, 6, 6, 1, gD)
}

// ---- the storks ----------------------------------------------------------------------------------------------------

// drawAlcStork: the nest at the bottom and the stork standing in it, frame f (see alcStorkFrame); var 1 faces right.
func drawAlcStork(b *pixBuf, v, f int) {
	const W = 28
	fx := func(x int) int {
		if v%2 == 1 {
			return W - 1 - x
		}
		return x
	}
	set := func(x, y int, c color.RGBA) { b.set(fx(x), y, c) }
	rect := func(x, y, w, h int, c color.RGBA) {
		for yy := y; yy < y+h; yy++ {
			for xx := x; xx < x+w; xx++ {
				set(xx, yy, c)
			}
		}
	}
	white, whiteD, black, red := rgb(0xf2eee6), rgb(0xc8c2ba), rgb(0x18181e), rgb(0xd8502e)
	// the nest: a wide bowl of twigs
	twig, twigD := rgb(0x7a5a36), rgb(0x4e3820)
	for y := 19; y < 26; y++ {
		hw := 11 - (y-19)/2
		rect(14-hw, y, 2*hw, 1, twig)
		for x := 14 - hw; x < 14+hw; x += 3 {
			set(x+(y%2), y, twigD)
		}
	}
	rect(3, 19, 22, 1, shade(twig, 1.2))
	// the body
	body := func(dy int) {
		for y := 12 + dy; y < 20; y++ {
			rect(10, y, 9, 1, white)
		}
		rect(10, 19, 9, 1, whiteD)
		rect(16, 13+dy, 5, 6, black) // the black flight feathers at the back
		rect(19, 15+dy, 3, 3, black)
	}
	switch f {
	case 1: // bowing: head down to the chest, preening
		body(1)
		rect(10, 12, 2, 3, white)
		rect(9, 14, 3, 3, white)
		set(9, 14, black)
		rect(10, 17, 4, 1, red)
	case 2: // the beak up, clattering
		body(0)
		rect(11, 6, 2, 7, white)
		rect(11, 4, 3, 3, white)
		set(12, 5, black)
		rect(12, 0, 1, 4, red)
		rect(13, 1, 1, 3, red)
	case 3, 4: // the wings open (half, then wide)
		body(0)
		rect(9, 5, 2, 8, white)
		rect(7, 4, 4, 3, white)
		set(8, 5, black)
		rect(3, 5, 4, 1, red)
		span := 6
		if f == 4 {
			span = 12
		}
		for i := 0; i < span; i++ {
			rect(14-i/2, 11-i/2, 1, 4, white) // the near wing up and forward
			rect(19+i/2, 11-i/2, 1, 4, black) // the far wing up and back
			set(14-i/2, 14-i/2, whiteD)
		}
	default: // standing still
		body(0)
		rect(10, 5, 2, 8, white) // the neck
		rect(8, 3, 4, 3, white)  // the head
		set(9, 4, black)
		rect(3, 4, 5, 1, red) // the long red beak
		rect(4, 5, 3, 1, shade(red, 0.8))
	}
}
