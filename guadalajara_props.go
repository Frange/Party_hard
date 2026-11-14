package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Props of GUADALAJARA (guadalajara.go). All drawn by code in the muted dusk palette of the game:
//   gdl_fountain    the octagonal fountain: stone basin, flower beds at its corners, the two-bowl column in the middle
//                   and its jets (animated: the drops run, the jets of the rim pulse, the water ripples)
//   gdl_park        the statue on its pedestal in a garden closed by a black iron fence
//   gdl_pergola     a wooden pergola covered with vines and grapes, a bench under it (guests sit in front)
//   gdl_honeystall  the honey stall of the market: red and white awning, jars of honey, "MIEL" on the counter
//   gdl_newskiosk   the purple newspaper kiosk: magazines, posters and the "PRENSA" sign
// The bandstand, the trees in planters and the bins are the ones of ALCALÁ DE HENARES (alcala_props.go).
// ---------------------------------------------------------------------------

const (
	gdlFountW, gdlFountH = 140, 108
	gdlFountFrames       = 4
)

func init() {
	pk := propKinds
	pk["gdl_fountain"] = &PropKind{W: gdlFountW, H: gdlFountH, FW: 120, FH: 50, Draw: func(b *pixBuf, v int) { drawGdlFountain(b, 0) }}
	pk["gdl_park"] = &PropKind{W: 150, H: 98, FW: 140, FH: 50, Draw: drawGdlPark}
	pk["gdl_pergola"] = &PropKind{W: 112, H: 76, FW: 104, FH: 14, Vars: 2, UseKind: "lounge",
		Use: []Vec{{-30, 10}, {0, 10}, {30, 10}}, Draw: drawGdlPergola}
	pk["gdl_honeystall"] = &PropKind{W: 70, H: 62, FW: 64, FH: 16, Hide: true, UseKind: "drink", Use: []Vec{{0, 12}},
		Draw: drawGdlHoneyStall}
	pk["gdl_newskiosk"] = &PropKind{W: 46, H: 62, FW: 40, FH: 22, Hide: true, Draw: drawGdlNewsKiosk}
	propAnims["gdl_fountain"] = &propAnim{n: gdlFountFrames, fps: 6, draw: func(b *pixBuf, k *PropKind, v, f int) { drawGdlFountain(b, f) }}
}

// ---- a tiny pixel font for the signs (capitals, 5 rows; drawn into the sprites, the names are never translated) ----

var gdlGlyphs = map[rune][]string{
	'A': {".#.", "#.#", "###", "#.#", "#.#"},
	'C': {".##", "#..", "#..", "#..", ".##"},
	'D': {"##.", "#.#", "#.#", "#.#", "##."},
	'E': {"###", "#..", "##.", "#..", "###"},
	'I': {"#", "#", "#", "#", "#"},
	'L': {"#..", "#..", "#..", "#..", "###"},
	'M': {"#...#", "##.##", "#.#.#", "#...#", "#...#"},
	'N': {"#..#", "##.#", "#.##", "#..#", "#..#"},
	'P': {"##.", "#.#", "##.", "#..", "#.."},
	'R': {"##.", "#.#", "##.", "#.#", "#.#"},
	'S': {".##", "#..", ".#.", "..#", "##."},
	'T': {"###", ".#.", ".#.", ".#.", ".#."},
	' ': {"..", "..", "..", "..", ".."},
}

// gdlTextW: the width in pixels of s in the sign font (1 px between letters).
func gdlTextW(s string) int {
	w := 0
	for _, r := range s {
		if r == 'Í' {
			r = 'I'
		}
		w += len(gdlGlyphs[r][0]) + 1
	}
	return maxI(0, w-1)
}

// gdlText draws s with its left edge at x and its top at y ('Í' gets its accent one pixel above the letter).
func gdlText(b *pixBuf, s string, x, y int, col color.RGBA) {
	for _, r := range s {
		if r == 'Í' {
			b.set(x+1, y-2, col)
			r = 'I'
		}
		g := gdlGlyphs[r]
		for gy, row := range g {
			for gx := 0; gx < len(row); gx++ {
				if row[gx] == '#' {
					b.set(x+gx, y+gy, col)
				}
			}
		}
		if len(g) > 0 {
			x += len(g[0]) + 1
		}
	}
}

// gdlTextC: gdlText centred on cx.
func gdlTextC(b *pixBuf, s string, cx, y int, col color.RGBA) {
	gdlText(b, s, cx-gdlTextW(s)/2, y, col)
}

// ---- the fountain --------------------------------------------------------------------------------------------------

// drawGdlFountain: the fountain at frame f (0..gdlFountFrames-1). The basin is centred 28 px over the anchor.
func drawGdlFountain(b *pixBuf, f int) {
	const W, H = gdlFountW, gdlFountH
	cx, cy := float64(W)/2, float64(H)-28
	stone, stoneD, stoneL := rgb(0xd0c4b0), rgb(0x8e8270), rgb(0xeee4d0)
	water, waterD, waterL, foam := rgb(0x3e7ab8), rgb(0x2c5a90), rgb(0x6aa8dc), rgb(0xe4f4ff)
	rx, ry := 50.0, 20.0
	b.ellipse(cx, cy+16, rx+16, ry+8, color.RGBA{0, 0, 0, 60})
	// the flower beds at the four diagonal corners, low and outside the basin
	for i, p := range [][2]float64{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		bx, by := cx+p[0]*(rx+2), cy+p[1]*(ry+2)
		b.ellipse(bx, by+2, 13, 6, stoneD)
		b.ellipse(bx, by+1, 12, 5, rgb(0x3a6a3a))
		for j := 0; j < 14; j++ {
			fx := int(bx) - 10 + itHash(j, 70+i)%21
			fy := int(by) - 3 + itHash(j, 80+i)%8
			b.set(fx, fy, rgb([]uint32{0xe05a6a, 0xf0c040, 0xb070d0, 0xf4f0e8}[(j+i)%4]))
		}
	}
	// the basin: the stone wall (5 px high at the front) and the rim, then the water inside
	for y := int(cy - ry - 1); y <= int(cy+ry+6); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			switch {
			case alcOct(dx, dy, rx, ry):
				c := stoneL
				if alcOct(dx, dy, rx-4, ry-3) {
					c = water
					if dy > ry*0.25 {
						c = waterD
					}
					if (int(dx)+int(dy)*3+f*2)%11 == 0 { // ripples that move with the frame
						c = waterL
					}
				} else if dx > rx*0.4 {
					c = stone
				}
				b.set(x, y, c)
			case alcOct(dx, dy-5, rx, ry) && dy > 0:
				c := stone
				if dx > rx*0.35 {
					c = shade(stone, 0.85)
				}
				if (x/8)%2 == 0 {
					c = shade(c, 0.95)
				}
				b.set(x, y, c)
			}
		}
	}
	// the jets of the rim: eight little arcs pointing to the middle, they rise and fall with the frame
	for i := 0; i < 8; i++ {
		a := float64(i)*math.Pi/4 + math.Pi/8
		sx, sy := cx+math.Cos(a)*(rx-6), cy+math.Sin(a)*(ry-4)
		h := 6.0 + float64((f+i)%gdlFountFrames)*2
		for s := 0.0; s <= 1; s += 0.1 {
			x := sx + (cx-sx)*0.4*s
			y := sy + (cy-sy)*0.4*s - h*4*s*(1-s)
			b.set(int(math.Round(x)), int(math.Round(y)), waterL)
		}
		b.set(int(math.Round(sx+(cx-sx)*0.4)), int(math.Round(sy+(cy-sy)*0.4)), foam)
	}
	// the column: a base in the water, the lower bowl, the shaft, the upper bowl and the finial
	b.rect(int(cx)-6, int(cy)-14, 12, 14, stone)
	b.rect(int(cx)-6, int(cy)-14, 3, 14, stoneL)
	b.rect(int(cx)+4, int(cy)-14, 2, 14, stoneD)
	b.ellipse(cx, cy-16, 22, 6, stoneD)
	b.ellipse(cx, cy-17, 21, 5, stone)
	b.ellipse(cx, cy-18, 18, 3.6, water)
	b.rect(int(cx)-3, int(cy)-46, 6, 30, stone)
	b.rect(int(cx)-3, int(cy)-46, 2, 30, stoneL)
	b.ellipse(cx, cy-47, 12, 4, stoneD)
	b.ellipse(cx, cy-48, 11, 3, stone)
	b.ellipse(cx, cy-49, 9, 2, waterL)
	b.rect(int(cx)-1, int(cy)-60, 3, 12, stoneL)
	b.ellipse(cx, cy-61, 3, 2, stoneL)
	// the water: the top spout, the curtains falling from both bowls and drops running down them
	b.rect(int(cx), int(cy)-72, 1, 11, foam)
	b.set(int(cx), int(cy)-72+f*3, waterL)
	for _, o := range []float64{-1, 1} {
		for s := 0; s < 12; s++ { // from the upper bowl down to the lower one
			x := cx + o*(10+float64(s)*0.9)
			y := cy - 48 + float64(s)*2.6
			b.set(int(math.Round(x)), int(math.Round(y)), waterL)
		}
		for s := 0; s < 10; s++ { // from the lower bowl down to the basin
			x := cx + o*(21+float64(s)*0.8)
			y := cy - 16 + float64(s)*1.6
			b.set(int(math.Round(x)), int(math.Round(y)), waterL)
		}
		for j := 0; j < 2; j++ { // the drops (the frame moves them down)
			t := float64((f+j*2)%gdlFountFrames) / gdlFountFrames
			b.set(int(math.Round(cx+o*(10+t*11*0.9))), int(math.Round(cy-48+t*11*2.6)), foam)
			b.set(int(math.Round(cx+o*(21+t*9*0.8))), int(math.Round(cy-16+t*9*1.6)), foam)
		}
	}
	for i := 0; i < 6; i++ { // foam where the curtains fall into the basin
		a := float64(i)*1.05 + float64(f)*0.5
		b.set(int(cx+math.Cos(a)*30), int(cy+math.Sin(a)*8), foam)
	}
}

// ---- the statue in its fenced garden -------------------------------------------------------------------------------

func drawGdlPark(b *pixBuf, v int) {
	const W, H = 150, 98
	stone, stoneD, stoneL := rgb(0xc8bca6), rgb(0x8a7e6c), rgb(0xe6dcc6)
	iron := rgb(0x24222c)
	b.ellipse(W/2, H-3, 74, 6, color.RGBA{0, 0, 0, 60})
	// the garden: a stone kerb round a lawn with flowers and two round bushes
	b.cube(3, 52, W-6, H-54, 34, rgb(0x3e6a3a), stone)
	b.rect(3, 52, W-6, 2, stoneL)
	b.rect(W-5, 52, 2, 34, stoneD)
	b.rect(W/2-8, 54, 16, 32, rgb(0xb8aa92)) // the path to the pedestal
	for i := 0; i < 110; i++ {
		x := 8 + itHash(i, 91)%(W-16)
		y := 56 + itHash(i, 93)%28
		if x > W/2-10 && x < W/2+10 {
			continue
		}
		b.set(x, y, rgb([]uint32{0xe05a6a, 0xf0c040, 0xb070d0, 0xf4f0e8, 0x5a9a4a}[i%5]))
	}
	for _, x := range []float64{24, W - 24} {
		b.ellipse(x, 62, 11, 9, rgb(0x2e5a34))
		b.ellipse(x-2, 59, 7, 6, rgb(0x4a8048))
	}
	// the pedestal and the statue (a figure in a long coat, one arm raised): pale stone
	cx := W / 2
	b.cube(cx-16, 58, 32, 22, 6, stoneL, stone)
	b.cube(cx-10, 34, 20, 28, 5, stoneL, stone)
	b.rect(cx-6, 44, 12, 8, stoneD) // the plaque
	b.rect(cx-4, 46, 8, 4, rgb(0xb8a070))
	st, stL, stD := rgb(0xb4b0a6), rgb(0xd6d2c8), rgb(0x7e7a72)
	b.rect(cx-4, 24, 3, 9, stD) // legs
	b.rect(cx+1, 24, 3, 9, st)
	b.rect(cx-6, 9, 12, 17, st) // the coat
	b.rect(cx-6, 9, 4, 17, stL)
	b.rect(cx+4, 9, 2, 17, stD)
	b.ellipse(float64(cx), 4, 3.5, 4, st) // the head
	b.set(cx-1, 3, stL)
	b.rect(cx+6, 2, 3, 9, st) // the raised arm
	b.rect(cx-9, 10, 3, 10, stD)
	// the black iron fence all round, with spear tips and a gate in the middle of the front
	for x := 4; x < W-4; x += 4 {
		front := !(x > W/2-12 && x < W/2+12)
		b.rect(x, 44, 1, 8, iron)
		b.set(x, 43, shade(iron, 1.8))
		if front {
			b.rect(x, 82, 1, 11, iron)
			b.set(x, 81, shade(iron, 1.8))
		}
	}
	b.rect(4, 45, W-8, 1, iron)
	b.rect(4, 49, W-8, 1, iron)
	for _, seg := range [][2]int{{4, W/2 - 12}, {W/2 + 12, W - 4}} {
		b.rect(seg[0], 84, seg[1]-seg[0], 1, iron)
		b.rect(seg[0], 89, seg[1]-seg[0], 1, iron)
	}
	for y := 44; y < 92; y += 3 { // the sides
		b.rect(4, y, 1, 3, iron)
		b.rect(W-5, y, 1, 3, iron)
	}
	for _, x := range []int{4, W/2 - 13, W/2 + 12, W - 5} { // the posts with their knobs
		b.rect(x-1, 80, 3, 13, iron)
		b.rect(x-1, 79, 3, 1, rgb(0xc8a040))
	}
}

// ---- the pergola with vines ----------------------------------------------------------------------------------------

func drawGdlPergola(b *pixBuf, v int) {
	const W, H = 112, 76
	wood, woodD, woodL := rgb(0x8a5e3a), rgb(0x5e3e26), rgb(0xb07e52)
	leaf, leafD, leafL := rgb(0x3e7440), rgb(0x2a5230), rgb(0x6a9a50)
	grape := rgb(0x6a3a7a)
	if v%2 == 1 {
		grape = rgb(0x9ab048) // white grapes on the other one
	}
	b.ellipse(W/2, H-3, 54, 4, color.RGBA{0, 0, 0, 60})
	// the bench under the pergola
	b.rect(10, H-18, W-20, 3, woodL)
	b.rect(10, H-15, W-20, 2, woodD)
	b.rect(10, H-26, W-20, 3, wood) // its back
	for _, x := range []int{14, W/2 - 1, W - 16} {
		b.rect(x, H-13, 2, 9, woodD)
		b.rect(x, H-26, 2, 8, woodD)
	}
	// the posts (front and back rows) up to the beams
	for _, x := range []int{4, 38, 72, W - 6} {
		b.rect(x, 22, 3, H-24, wood)
		b.rect(x, 22, 1, H-24, woodL)
		b.rect(x-1, H-4, 5, 2, rgb(0x9a9080)) // the stone feet
	}
	for _, y := range []int{16, 22} { // the beams and the crossbars
		b.rect(0, y, W, 3, wood)
		b.rect(0, y, W, 1, woodL)
	}
	for x := 2; x < W; x += 9 {
		b.rect(x, 14, 3, 10, woodD)
	}
	// the vines on top: a thick roof of leaves with bunches of grapes hanging
	for i := 0; i < 34; i++ {
		x := 4 + itHash(i, 101+v)%(W-8)
		y := 6 + itHash(i, 103)%14
		b.ellipse(float64(x), float64(y), 7, 5, leafD)
		b.ellipse(float64(x)-1, float64(y)-1, 5, 3.5, leaf)
	}
	for i := 0; i < 26; i++ {
		x, y := 2+itHash(i, 107)%(W-4), 3+itHash(i, 109)%16
		b.set(x, y, leafL)
	}
	for i := 0; i < 9; i++ { // vines hanging down the posts and bunches of grapes
		x := 6 + i*12 + itHash(i, 111)%5
		l := 4 + itHash(i, 113)%9
		b.rect(x, 22, 1, l, leafD)
		b.set(x+1, 22+l/2, leaf)
		gx, gy := x+2, 24+itHash(i, 115)%4
		for _, d := range [][2]int{{0, 0}, {2, 0}, {1, 1}, {0, 2}, {2, 2}, {1, 3}} {
			b.set(gx+d[0], gy+d[1], grape)
		}
		b.set(gx, gy, shade(grape, 1.3))
	}
	// pots of flowers at both ends
	for _, x := range []int{2, W - 10} {
		b.rect(x, H-10, 8, 7, rgb(0xa85a3a))
		b.ellipse(float64(x+4), float64(H-11), 5, 3, leaf)
		b.set(x+2, H-13, rgb(0xe05a6a))
		b.set(x+5, H-14, rgb(0xf0c040))
	}
}

// ---- the honey stall -----------------------------------------------------------------------------------------------

func drawGdlHoneyStall(b *pixBuf, v int) {
	const W, H = 70, 62
	wood, woodD := rgb(0x8a5e3a), rgb(0x5e3e26)
	red, white := rgb(0xc8403a), rgb(0xf0ece0)
	honey, honeyL, honeyD := rgb(0xd8901e), rgb(0xf0c050), rgb(0x9a5a12)
	b.ellipse(W/2, H-2, 34, 3, color.RGBA{0, 0, 0, 60})
	for _, x := range []int{3, W - 5} { // the posts
		b.rect(x, 14, 2, H-18, woodD)
	}
	// the striped awning with a scalloped hem
	for x := 0; x < W; x++ {
		top := 5 + int(4*math.Abs(float64(x)-W/2)/(W/2))
		c := white
		if (x/6)%2 == 0 {
			c = red
		}
		b.rect(x, top, 1, 17-top, c)
		b.set(x, top, shade(c, 1.1))
		if x%6 < 4 {
			b.rect(x, 17, 1, 2, c)
		}
	}
	// shelves of jars at the back and the counter in front with more jars and a honey dipper
	b.rect(6, 22, W-12, 2, woodD)
	for i := 0; i < 9; i++ {
		x := 8 + i*6
		b.rect(x, 25, 4, 6, honey)
		b.rect(x, 25, 1, 6, honeyL)
		b.rect(x, 24, 4, 1, white) // the lids
	}
	b.cube(2, 36, W-4, H-38, 8, rgb(0xa8784a), wood)
	b.rect(2, 50, W-4, 1, woodD)
	for i := 0; i < 7; i++ {
		x := 6 + i*9
		b.rect(x, 30, 6, 8, honey)
		b.rect(x, 30, 2, 8, honeyL)
		b.rect(x+5, 31, 1, 7, honeyD)
		b.rect(x, 29, 6, 2, []color.RGBA{red, white}[i%2]) // the cloth tops
	}
	// the sign on the front of the counter
	b.rect(W/2-14, 46, 28, 9, rgb(0x3a2418))
	gdlTextC(b, "MIEL", W/2, 48, honeyL)
}

// ---- the newspaper kiosk -------------------------------------------------------------------------------------------

func drawGdlNewsKiosk(b *pixBuf, v int) {
	const W, H = 46, 62
	purple, purpleD, purpleL := rgb(0x6a3a9a), rgb(0x46246a), rgb(0x9a6ac8)
	b.ellipse(W/2, H-2, 22, 3, color.RGBA{0, 0, 0, 60})
	// the body (the side in shade) and the counter
	b.cube(3, 26, W-6, H-28, 10, purpleD, purple)
	b.rect(W-8, 26, 5, H-28, purpleD)
	// magazines and newspapers hanging on the front, in rows
	for r := 0; r < 2; r++ {
		for i := 0; i < 5; i++ {
			x, y := 6+i*7, 38+r*10
			c := rgb([]uint32{0xe8d8b8, 0xd85a5a, 0x5a8ad8, 0xf0c040, 0x6aba6a}[(i+r*2)%5])
			b.rect(x, y, 6, 8, c)
			b.rect(x+1, y+1, 4, 2, shade(c, 0.6))
			b.rect(x+1, y+4, 3, 1, shade(c, 0.75))
		}
	}
	// the roof: a purple hood with a light edge, the sign board on top
	for y := 12; y < 24; y++ {
		hw := 18 + (y-12)/2
		b.rect(W/2-hw, y, 2*hw, 1, mixc(purpleL, purple, float64(y-12)/12))
	}
	b.rect(0, 24, W, 2, purpleD)
	b.rect(5, 2, W-10, 10, rgb(0x241830))
	b.rect(5, 2, W-10, 1, purpleL)
	gdlTextC(b, "PRENSA", W/2, 5, rgb(0xf4f0e8))
	b.rect(W/2-1, 0, 2, 2, purpleL)
}
