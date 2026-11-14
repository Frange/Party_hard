package main

import (
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Props of the last party (the island, see jurassic.go): dinosaurs, the main
// gate, the visitor centre, towers, jeeps and a few silly extras. All drawn by
// code, same mechanism as every other prop (propKinds).
// ---------------------------------------------------------------------------

// capsule draws a thick segment whose radius goes from r0 to r1.
func capsule(b *pixBuf, x0, y0, x1, y1, r0, r1 float64, col color.RGBA) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))*1.5) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		r := r0 + (r1-r0)*t
		b.ellipse(x0+(x1-x0)*t, y0+(y1-y0)*t, r, r, col)
	}
}

// dinoBody paints an ellipse with an underside shadow and a top highlight.
func dinoBody(b *pixBuf, cx, cy, rx, ry float64, col color.RGBA) {
	b.ellipse(cx, cy+1.5, rx, ry, shade(col, 0.62))
	b.ellipse(cx, cy, rx-0.5, ry-1.2, col)
	b.ellipse(cx-rx*0.12, cy-ry*0.45, rx*0.62, ry*0.34, shade(col, 1.28))
}

func dinoSpots(b *pixBuf, seed int64, cx, cy, rx, ry float64, n int, col color.RGBA) {
	rng := rand.New(rand.NewSource(seed))
	for i := 0; i < n; i++ {
		a, d := rng.Float64()*2*math.Pi, math.Sqrt(rng.Float64())*0.8
		x, y := cx+math.Cos(a)*rx*d, cy+math.Sin(a)*ry*d
		b.ellipse(x, y, 1.6, 1.2, col)
	}
}

func dinoEye(b *pixBuf, x, y int) {
	b.set(x, y, c(0xffffff))
	b.set(x+1, y, c(0x14101a))
}

// pixel font used by the gate sign: 5x5 glyphs
var jpGlyphs = map[rune][5]string{
	'J': {"..###", "...#.", "...#.", "#..#.", ".##.."},
	'U': {"#...#", "#...#", "#...#", "#...#", ".###."},
	'R': {"####.", "#...#", "####.", "#..#.", "#...#"},
	'A': {".###.", "#...#", "#####", "#...#", "#...#"},
	'S': {".####", "#....", ".###.", "....#", "####."},
	'I': {".###.", "..#..", "..#..", "..#..", ".###."},
	'C': {".####", "#....", "#....", "#....", ".####"},
	'P': {"####.", "#...#", "####.", "#....", "#...."},
	'K': {"#...#", "#..#.", "###..", "#..#.", "#...#"},
	'Q': {".###.", "#...#", "#.#.#", "#..#.", ".##.#"},
	'E': {"#####", "#....", "####.", "#....", "#####"},
	'F': {"#####", "#....", "####.", "#....", "#...."},
	'T': {"#####", "..#..", "..#..", "..#..", "..#.."},
	'O': {".###.", "#...#", "#...#", "#...#", ".###."},
}

// jpAccent: letters drawn as another glyph plus an acute accent above it (two cells, one cell clear of the letter).
var jpAccent = map[rune]rune{'Á': 'A'}

func jpText(b *pixBuf, s string, x, y, k int, col, edge color.RGBA) {
	for pass := 0; pass < 2; pass++ {
		cx := x
		for _, r := range s {
			base, acc := jpAccent[r]
			if acc {
				r = base
				for _, cell := range [][2]int{{3, -3}, {2, -2}} { // the acute: up and to the right
					gx, gy := cell[0], cell[1]
					if pass == 0 {
						b.rect(cx+gx*k-1, y+gy*k-1, k+2, k+2, edge)
					} else {
						b.rect(cx+gx*k, y+gy*k, k, k, col)
					}
				}
			}
			g, ok := jpGlyphs[r]
			if ok {
				for gy := 0; gy < 5; gy++ {
					for gx := 0; gx < 5; gx++ {
						if g[gy][gx] != '#' {
							continue
						}
						if pass == 0 {
							b.rect(cx+gx*k-1, y+gy*k-1, k+2, k+2, edge)
						} else {
							b.rect(cx+gx*k, y+gy*k, k, k, col)
						}
					}
				}
			}
			cx += 6 * k
		}
	}
}

func flame(b *pixBuf, x, y int, h int) {
	b.ellipse(float64(x), float64(y)+0.5, 4, float64(h)+0.5, fireRamp[5]) // dark red rim (fire_art.go)
	b.ellipse(float64(x), float64(y), 3.4, float64(h), c(0xe84a18))
	b.ellipse(float64(x), float64(y)+1, 2.4, float64(h)*0.75, c(0xff9a20))
	b.ellipse(float64(x), float64(y)+2, 1.3, float64(h)*0.45, c(0xfff0a0))
}

func stoneBlock(b *pixBuf, x, y, w, h int, base color.RGBA, seed int64) {
	b.rect(x, y, w, h, base)
	rng := rand.New(rand.NewSource(seed))
	for yy := y; yy < y+h; yy += 6 {
		b.rect(x, yy, w, 1, shade(base, 0.7))
		off := (yy / 6 % 2) * 7
		for xx := x + off; xx < x+w; xx += 14 {
			b.rect(xx, yy, 1, 6, shade(base, 0.72))
		}
	}
	for i := 0; i < w*h/30; i++ {
		b.set(x+rng.Intn(w), y+rng.Intn(h), shade(base, 0.85+rng.Float64()*0.35))
	}
	b.rect(x, y, 2, h, shade(base, 1.2))
	b.rect(x+w-2, y, 2, h, shade(base, 0.6))
}

func init() {
	pk := propKinds

	// ---- dinosaurs (all face left; decorative, with a small solid foot) -----------------
	pk["jp_brachio"] = &PropKind{W: 150, H: 90, FW: 70, FH: 14, Vars: 4, NoSolid: true, Draw: func(b *pixBuf, v int) {
		col := []color.RGBA{c(0x7a8aa8), c(0x8a9a68)}[(v>>1)%2]
		hy := float64(v&1) * 3 // the head sways
		spot := shade(col, 0.74)
		b.ellipse(78, 87, 52, 3.5, color.RGBA{0, 0, 0, 80})
		capsule(b, 108, 54, 146, 72, 8, 1.6, shade(col, 0.62))
		capsule(b, 108, 52.5, 146, 70.5, 7, 1.1, col)
		for _, lx := range []float64{50, 64, 94, 108} { // legs
			r := 5.0
			if lx > 80 {
				r = 6
			}
			capsule(b, lx, 58, lx, 82, r, r-1, shade(col, 0.7))
			b.ellipse(lx, 85, r+1, 3, shade(col, 0.5))
		}
		dinoBody(b, 78, 52, 36, 18, col)
		dinoSpots(b, 3, 78, 52, 32, 14, 16, spot)
		capsule(b, 50, 47, 34, 27, 9, 6.4, shade(col, 0.62)) // neck
		capsule(b, 34, 27, 20, 11+hy, 6.4, 4.6, shade(col, 0.62))
		capsule(b, 50, 45, 34, 25, 8, 5.6, col)
		capsule(b, 34, 25, 20, 9.5+hy, 5.6, 4, col)
		capsule(b, 48, 41, 33, 22, 3, 2, shade(col, 1.3))
		b.ellipse(14, 8+hy, 7.5, 4.8, shade(col, 0.7))
		b.ellipse(14, 7+hy, 7, 4.2, col)
		dinoEye(b, 13, 5+int(hy))
		b.set(8, 8+int(hy), shade(col, 0.5))
	}}
	pk["jp_trike"] = &PropKind{W: 104, H: 58, FW: 66, FH: 14, Vars: 4, NoSolid: true, Draw: func(b *pixBuf, v int) {
		col := []color.RGBA{c(0x8a7a70), c(0x9a8a5a)}[(v>>1)%2]
		frill := []color.RGBA{c(0xc86a48), c(0x6a9aa8)}[(v>>1)%2]
		hy := float64(v&1) * 2 // the head nods
		b.ellipse(58, 53, 42, 3.4, color.RGBA{0, 0, 0, 80})
		capsule(b, 90, 38, 101, 47, 6, 1.4, shade(col, 0.7))
		b.ellipse(44, 49, 10, 4.5, shade(col, 0.55))
		b.ellipse(76, 49, 10, 4.5, shade(col, 0.55))
		dinoBody(b, 58, 37, 34, 15, col)
		dinoSpots(b, 7, 58, 37, 28, 11, 12, shade(col, 0.76))
		for k := 0; k < 9; k++ { // frill edge
			a := math.Pi*0.5 + float64(k)/8*math.Pi
			b.ellipse(28+math.Cos(a)*17, 30+math.Sin(a)*19, 3.2, 3.2, shade(frill, 1.15))
		}
		b.ellipse(28, 30, 16, 18, shade(frill, 0.8))
		b.ellipse(28, 29, 14, 16, frill)
		b.ellipse(26, 25, 8, 9, shade(frill, 1.15))
		for k := 0; k < 6; k++ {
			a := float64(k) / 5 * math.Pi
			b.set(28+int(math.Cos(a+0.5)*10), 30+int(math.Sin(a+0.5)*11), shade(frill, 0.6))
		}
		b.ellipse(17, 40+hy, 12, 9, shade(col, 0.7))
		b.ellipse(17, 39+hy, 11.5, 8.2, col)
		capsule(b, 14, 30+hy, 4, 20+hy, 1.6, 0.8, c(0xefe8d0)) // brow horns
		capsule(b, 20, 29+hy, 11, 19+hy, 1.6, 0.8, c(0xefe8d0))
		capsule(b, 8, 40+hy, 2, 43+hy, 1.8, 0.7, c(0xf4eed8)) // nose horn
		b.ellipse(7, 44+hy, 4, 3, shade(col, 0.55))           // beak
		dinoEye(b, 16, 36+int(hy))
	}}
	pk["jp_raptor"] = &PropKind{W: 48, H: 36, FW: 24, FH: 8, Vars: 6, NoSolid: true, Draw: func(b *pixBuf, v int) {
		col := []color.RGBA{c(0x5a7a9a), c(0x8a5a3a), c(0x4a8a5a)}[(v>>1)%3]
		tl := float64(v&1) * 3 // the tail flicks
		stripe := shade(col, 0.55)
		b.ellipse(26, 33, 17, 2.4, color.RGBA{0, 0, 0, 80})
		capsule(b, 32, 20, 46, 12-tl, 4.4, 1, shade(col, 0.7)) // tail
		capsule(b, 21, 24, 19, 31, 2.6, 1.8, shade(col, 0.7))
		capsule(b, 28, 24, 29, 31, 3, 1.8, shade(col, 0.8))
		b.rect(16, 31, 6, 1, c(0xe8e0c8))
		b.rect(26, 31, 6, 1, c(0xe8e0c8))
		dinoBody(b, 25, 19, 11, 6.4, col)
		for k := 0; k < 4; k++ {
			capsule(b, 21+float64(k)*3, 14, 21+float64(k)*3, 22, 0.6, 0.6, stripe)
		}
		capsule(b, 16, 17, 10, 9, 3.4, 2.2, shade(col, 0.8)) // neck
		capsule(b, 15, 16, 10, 9, 2.6, 1.6, col)
		b.ellipse(7, 8, 4.4, 2.8, shade(col, 0.7)) // head
		b.ellipse(7, 7.4, 4, 2.2, col)
		capsule(b, 5, 8, 1.6, 9, 1.4, 1, shade(col, 0.8))
		b.set(8, 6, c(0xffe040))
		capsule(b, 17, 22, 13, 24, 1, 0.8, shade(col, 0.6)) // tiny arm
		b.set(12, 24, c(0xe8e0c8))
	}}
	// The rex, in the poses of its meal (see rex.go):
	// 0 walking, 1 head down with the jaws open, 2 head down biting, 3 head up swallowing, 4 head up roaring.
	pk["jp_trex"] = &PropKind{W: 160, H: 92, FW: 40, FH: 12, Vars: rexPoses, NoSolid: true, Draw: func(b *pixBuf, v int) {
		b.ox, b.oy = rexArtOX, rexArtOY // room on the left and above for the head when it bends down
		drawRexArt(b, v)
	}}
	pk["jp_hadro"] = &PropKind{W: 88, H: 52, FW: 40, FH: 10, Vars: 4, NoSolid: true, Draw: func(b *pixBuf, v int) {
		col := []color.RGBA{c(0x5a9a58), c(0xa8782a)}[(v>>1)%2]
		hy := float64(v&1) * 2.5 // the head sways
		b.ellipse(46, 49, 30, 2.8, color.RGBA{0, 0, 0, 80})
		capsule(b, 64, 32, 85, 42, 6, 1.2, shade(col, 0.7))
		for _, lx := range []float64{38, 54} {
			capsule(b, lx, 36, lx, 46, 4.4, 3, shade(col, 0.7))
			b.ellipse(lx, 47, 5, 2.4, shade(col, 0.5))
		}
		dinoBody(b, 48, 31, 24, 11, col)
		dinoSpots(b, 11, 48, 31, 20, 8, 9, shade(col, 0.7))
		capsule(b, 30, 28, 20, 15+hy, 6, 4, shade(col, 0.7))
		capsule(b, 30, 26.6, 20, 14+hy, 5, 3.4, col)
		b.ellipse(15, 12+hy, 6.4, 3.8, shade(col, 0.7))
		b.ellipse(15, 11.4+hy, 6, 3.3, col)
		capsule(b, 14, 9+hy, 4, 5+hy, 1.8, 1, shade(col, 1.2)) // crest
		dinoEye(b, 16, 9+int(hy))
	}}
	pk["jp_skel"] = &PropKind{W: 100, H: 42, FW: 70, FH: 10, Draw: func(b *pixBuf, v int) {
		bone, sh := c(0xece4cc), c(0xb4aa8c)
		b.ellipse(52, 38, 44, 3.6, color.RGBA{0, 0, 0, 70})
		capsule(b, 36, 14, 94, 26, 2.2, 0.8, sh)
		capsule(b, 36, 13, 94, 25, 1.8, 0.5, bone)
		for k := 0; k < 8; k++ { // ribs
			x := 42 + float64(k)*5
			capsule(b, x, 14+float64(k)*0.4, x+2, 31, 1.2, 0.7, sh)
			capsule(b, x, 14+float64(k)*0.4, x+1, 30, 0.9, 0.5, bone)
		}
		capsule(b, 56, 30, 58, 38, 2, 1.4, bone) // legs
		capsule(b, 74, 30, 76, 38, 2, 1.4, bone)
		b.ellipse(58, 38, 4, 1.6, sh)
		b.ellipse(76, 38, 4, 1.6, sh)
		for k := 0; k < 8; k++ { // tail vertebrae
			b.ellipse(80+float64(k)*2.4, 21+float64(k)*0.8, 1.6-float64(k)*0.12, 1.5-float64(k)*0.1, sh)
		}
		b.ellipse(18, 21, 14, 9, sh) // skull
		b.ellipse(18, 20, 13, 8, bone)
		capsule(b, 14, 22, 4, 24, 5, 3, bone)
		b.ellipse(21, 18, 3, 3, c(0x2a2030))
		for k := 0; k < 5; k++ {
			b.rect(5+k*3, 26, 1, 3, bone)
		}
		capsule(b, 30, 18, 38, 14, 3, 2, bone)
	}}

	// ---- the main gate ------------------------------------------------------------------
	pk["jp_gate"] = &PropKind{W: 190, H: 110, FW: 8, FH: 4, NoSolid: true,
		Solids: []Rect{{-91, -26, 40, 26}, {51, -26, 40, 26}},
		Draw: func(b *pixBuf, v int) {
			stone := c(0x655a52)
			wood, woodD := c(0x6a4a2a), c(0x3a2814)
			b.ellipse(95, 106, 90, 4, color.RGBA{0, 0, 0, 80})
			// the open wooden doors, seen at an angle, behind the pillars
			for side := 0; side < 2; side++ {
				x0, dir := 44, 1
				if side == 1 {
					x0, dir = 146, -1
				}
				for k := 0; k < 22; k++ { // a slanted panel: it gets taller towards the pillar
					x := x0 + dir*k
					top := 50 - (22-k)/3
					bot := 102 - (22-k)/6
					col := wood
					if k%5 == 0 {
						col = woodD
					}
					b.rect(x, top, 1, bot-top, col)
				}
				for _, y := range []int{58, 76, 92} {
					for k := 0; k < 22; k++ {
						b.set(x0+dir*k, y-(22-k)/4, c(0x23201e))
					}
				}
			}
			// the stone pillars: tapered, rough blocks, torches
			for _, px := range []int{4, 146} {
				for y := 8; y < 106; y++ {
					inset := (106 - y) / 16
					stoneBlock(b, px+inset, y, 40-2*inset, 1, stone, int64(y+px))
				}
				stoneBlock(b, px+6, 10, 28, 94, shade(stone, 0.96), int64(px+9))
				for y := 16; y < 100; y += 14 { // rough ledges
					b.rect(px+3, y, 34, 1, shade(stone, 1.25))
					b.rect(px+3, y+1, 34, 1, shade(stone, 0.6))
				}
				b.rect(px+2, 4, 36, 7, shade(stone, 1.12)) // cap
				b.rect(px+2, 4, 36, 1, shade(stone, 1.5))
				b.rect(px+2, 10, 36, 1, shade(stone, 0.55))
				ix := px + 36
				if px > 100 {
					ix = px + 2
				}
				for _, ty := range []int{22, 88} { // torches on the inner side (the lower one below the wide sign)
					b.rect(ix-1, ty+4, 4, 12, c(0x3a2a1c))
					b.rect(ix-3, ty+3, 8, 3, c(0x1a1210))
					flame(b, ix+1, ty-3, 6)
				}
				flame(b, px+20, 2, 5) // the brazier on top
				b.rect(px+15, 3, 10, 3, c(0x1a1210))
			}
			// the arch of beams between the pillars and the sign
			for x := 44; x <= 146; x++ {
				k := float64(x-44) / 102
				y := 34 - int(math.Sin(math.Pi*k)*14)
				b.rect(x, y, 1, 4, c(0x2a1c10))
				b.set(x, y, c(0x5a4026))
			}
			// the sign of our own park (PARQUE FIESTÁSICO, the same name in ES and EN): wider than the
			// gap between the pillars, in front of them; party purple, not the colours of any real park
			b.rect(30, 38, 130, 40, c(0x24102e))
			b.rect(32, 40, 126, 36, c(0x6a2a8a))
			b.rect(32, 40, 126, 2, c(0x9a4ab8))
			b.rect(32, 74, 126, 2, c(0x3a1450))
			jpText(b, "PARQUE", 60, 43, 2, c(0xf8d030), c(0x200a30))     // both lines: same size, centred on the sign (x = 95)
			jpText(b, "FIESTÁSICO", 36, 62, 2, c(0xf8d030), c(0x200a30)) // 10 letters = 118 px; the accent goes above the A
			for _, lx := range []int{36, 154} {                          // lamps on the sign
				b.ellipse(float64(lx), 40, 2, 2, c(0xfff0a0))
			}
			b.ellipse(95, 20, 3.4, 3.4, c(0xf8d030)) // emblem at the top of the arch
			b.ellipse(95, 20, 1.6, 1.6, c(0x6a2a8a))
		}}

	// ---- buildings and towers ----------------------------------------------------------
	pk["jp_rotunda"] = &PropKind{W: 204, H: 134, FW: 120, FH: 28, Draw: func(b *pixBuf, v int) {
		wall, wall2 := c(0xcdbb94), c(0xa8987a)
		glass, glass2 := c(0x5aa0a8), c(0x9ad4d4)
		b.ellipse(102, 128, 90, 5, color.RGBA{0, 0, 0, 80})
		// side wings
		b.rect(6, 92, 44, 36, wall)
		b.rect(6, 88, 44, 6, c(0x8a6a4a))
		b.rect(154, 92, 44, 36, wall)
		b.rect(154, 88, 44, 6, c(0x8a6a4a))
		// drum
		b.rect(34, 76, 136, 52, wall)
		b.rect(34, 76, 136, 3, wall2)
		b.rect(34, 124, 136, 4, wall2)
		for x := 40; x < 164; x += 14 {
			b.rect(x, 92, 8, 14, c(0x2a3a4a))
			b.rect(x, 92, 8, 2, c(0x6a8aa0))
		}
		b.rect(86, 98, 32, 30, c(0x2a2018)) // entrance
		b.rect(88, 100, 28, 28, c(0x5a8a98))
		b.rect(101, 100, 2, 28, c(0x2a2018))
		b.rect(80, 92, 44, 6, c(0x8a6a4a))
		// stepped glass roof
		tiers := [][4]int{{34, 52, 136, 28}, {52, 32, 100, 22}, {72, 16, 60, 18}}
		for _, t := range tiers {
			for y := 0; y < t[3]; y++ {
				k := float64(y) / float64(t[3])
				inset := int(k * 8)
				b.rect(t[0]+inset-1, t[1]+y, t[2]-2*inset+2, 1, mixc(glass2, glass, k))
			}
			for x := t[0] + 4; x < t[0]+t[2]-4; x += 9 {
				capsule(b, float64(x), float64(t[1]+t[3]-1), float64(x)+3, float64(t[1]), 0.5, 0.5, c(0x7a5a3a))
			}
			b.rect(t[0], t[1]+t[3]-2, t[2], 3, c(0x6a4a2a))
		}
		b.rect(94, 6, 16, 12, glass2)
		b.rect(100, 0, 4, 8, c(0x6a4a2a))
		b.rect(96, 6, 12, 1, c(0xffffff))
	}}
	pk["jp_lab"] = &PropKind{W: 124, H: 90, FW: 108, FH: 28, Vars: 2, Draw: func(b *pixBuf, v int) {
		wall := []color.RGBA{c(0xd2d6da), c(0xcfc6aa)}[v%2]
		b.ellipse(62, 86, 54, 4, color.RGBA{0, 0, 0, 80})
		b.rect(6, 28, 112, 58, wall)
		b.rect(6, 28, 112, 4, shade(wall, 0.7))
		b.rect(2, 22, 120, 8, shade(wall, 0.82))
		b.rect(2, 22, 120, 1, shade(wall, 1.2))
		b.rect(6, 82, 112, 4, shade(wall, 0.65))
		for x := 14; x < 108; x += 22 {
			b.rect(x, 42, 14, 16, c(0x35506a))
			b.rect(x, 42, 14, 3, c(0x7aa2c0))
		}
		b.rect(52, 62, 20, 24, c(0x2a3440)) // door
		b.rect(53, 63, 8, 22, c(0x4a6076))
		for x := 8; x < 114; x += 10 { // hazard stripe
			b.rect(x, 78, 5, 4, c(0xf0c020))
		}
		b.rect(18, 8, 24, 16, shade(wall, 0.9)) // roof units
		b.rect(18, 8, 24, 3, shade(wall, 1.2))
		b.rect(80, 12, 20, 12, shade(wall, 0.9))
		b.ellipse(90, 12, 6, 2.4, c(0x2a3440))
		b.ellipse(30, 8, 8, 2.4, c(0x2a3440))
		b.ellipse(110, 52, 6, 6, c(0xf0c020)) // warning disc
		b.ellipse(110, 52, 4, 4, c(0x2a2a30))
		b.ellipse(110, 52, 1.6, 1.6, c(0xf0c020))
	}}
	pk["jp_control"] = &PropKind{W: 70, H: 112, FW: 50, FH: 18, Draw: func(b *pixBuf, v int) {
		wall := c(0xc4c8cc)
		b.ellipse(35, 108, 30, 3.4, color.RGBA{0, 0, 0, 80})
		b.rect(10, 40, 50, 68, wall)
		b.rect(10, 40, 50, 3, shade(wall, 1.2))
		b.rect(56, 40, 4, 68, shade(wall, 0.7))
		b.rect(6, 28, 58, 16, shade(wall, 0.85)) // control room
		for x := 10; x < 60; x += 9 {
			b.rect(x, 31, 7, 9, c(0x4a7a98))
			b.rect(x, 31, 7, 2, c(0x9ad0e8))
		}
		b.rect(4, 24, 62, 5, shade(wall, 0.6))
		b.rect(33, 4, 3, 20, c(0x5a5a64)) // mast
		b.rect(26, 12, 17, 2, c(0x5a5a64))
		b.set(34, 3, c(0xff3030))
		b.set(34, 2, c(0xff8080))
		for y := 50; y < 100; y += 12 {
			b.rect(16, y, 12, 6, c(0x35506a))
		}
		b.rect(34, 80, 14, 28, c(0x2a3440))
	}}
	pk["jp_dish"] = &PropKind{W: 64, H: 64, FW: 14, FH: 6, Draw: func(b *pixBuf, v int) {
		b.ellipse(32, 60, 14, 2.6, color.RGBA{0, 0, 0, 80})
		capsule(b, 32, 58, 24, 36, 2, 2, c(0x6a6a74))
		capsule(b, 32, 58, 42, 36, 2, 2, c(0x6a6a74))
		b.ellipse(32, 26, 26, 22, c(0x8a8a96))
		b.ellipse(31, 26, 23, 19, c(0xdadae2))
		b.ellipse(34, 28, 16, 12, c(0xbcbcc8))
		for k := 0; k < 4; k++ {
			b.ellipse(32, 26, 23-float64(k)*6, 19-float64(k)*5, shade(c(0xdadae2), 1-float64(k)*0.05))
		}
		b.ellipse(34, 28, 4, 3, c(0x6a6a74))
		capsule(b, 34, 28, 48, 10, 0.8, 0.8, c(0x4a4a54))
		b.ellipse(48, 9, 2.4, 2.4, c(0xff3030))
	}}
	pk["jp_mast"] = &PropKind{W: 28, H: 104, FW: 12, FH: 6, Draw: func(b *pixBuf, v int) {
		b.ellipse(14, 100, 10, 2.4, color.RGBA{0, 0, 0, 80})
		col := c(0x8a8a94)
		for y := 6; y < 98; y += 10 {
			inset := y / 28
			b.rect(7+inset, y, 14-2*inset, 1, col)
			b.line(7+inset, y, 21-inset, y+10, col)
			b.line(21-inset, y, 7+inset, y+10, shade(col, 0.8))
		}
		b.rect(7, 6, 1, 92, shade(col, 1.2))
		b.rect(20, 6, 1, 92, shade(col, 0.7))
		b.rect(10, 0, 8, 6, c(0x6a6a74))
		b.set(13, 0, c(0xff3030))
		b.set(14, 0, c(0xff8080))
	}}
	pk["jp_tower"] = &PropKind{W: 48, H: 96, FW: 30, FH: 10, Draw: func(b *pixBuf, v int) {
		wood, dark := c(0x8a6a44), c(0x4a3624)
		b.ellipse(24, 92, 20, 3, color.RGBA{0, 0, 0, 80})
		for _, lx := range []int{10, 36} { // stilts
			capsule(b, float64(lx), 40, float64(lx)+(float64(lx)-24)*0.15, 92, 2.4, 3, wood)
		}
		for y := 52; y < 90; y += 16 {
			b.line(10, y, 36, y+16, dark)
			b.line(36, y, 10, y+16, dark)
		}
		b.rect(6, 38, 36, 4, dark)  // platform
		b.rect(8, 22, 32, 18, wood) // hut
		b.rect(8, 22, 32, 2, shade(wood, 1.25))
		b.rect(12, 28, 24, 8, c(0x2a3a4a))
		b.rect(12, 28, 24, 2, c(0x6a9ab8))
		for y := 0; y < 14; y++ { // roof
			b.rect(2+y/2, 8+y, 44-y, 1, mixc(c(0x7a4a2a), c(0x5a3418), float64(y)/14))
		}
		b.rect(22, 2, 4, 6, dark)
		b.rect(22, 0, 10, 4, c(0xd03030))
		b.line(34, 42, 40, 90, dark)
	}}
	pk["jp_lighthouse"] = &PropKind{W: 36, H: 104, FW: 20, FH: 8, Draw: func(b *pixBuf, v int) {
		b.ellipse(18, 100, 15, 3, color.RGBA{0, 0, 0, 80})
		for y := 24; y < 98; y++ {
			k := float64(y-24) / 74
			w := 12 + int(k*5)
			col := c(0xf2f2f2)
			if (y/16)%2 == 1 {
				col = c(0xd83a3a)
			}
			b.rect(18-w/2, y, w, 1, col)
			b.set(18-w/2, y, shade(col, 1.15))
			b.set(18+w/2-1, y, shade(col, 0.65))
		}
		b.rect(8, 20, 20, 5, c(0x3a3a46))
		b.rect(11, 8, 14, 12, c(0xf8e890))
		b.rect(11, 8, 14, 2, c(0xffffff))
		for x := 11; x < 25; x += 5 {
			b.rect(x, 8, 1, 12, c(0x3a3a46))
		}
		for y := 0; y < 8; y++ {
			b.rect(10+y/2, y, 16-y, 1, c(0xb02828))
		}
		b.rect(16, 36, 4, 6, c(0x2a3440))
	}}
	pk["jp_hut"] = &PropKind{W: 56, H: 50, FW: 38, FH: 10, Draw: func(b *pixBuf, v int) {
		b.ellipse(28, 46, 24, 3, color.RGBA{0, 0, 0, 80})
		b.rect(10, 24, 36, 20, c(0x9a7a50))
		for x := 12; x < 44; x += 6 {
			b.rect(x, 24, 1, 20, c(0x6a4a2a))
		}
		b.rect(22, 30, 12, 14, c(0x2a1c12))
		for y := 0; y < 22; y++ {
			w := 10 + y*2
			b.rect(28-w/2, 2+y, w, 1, mixc(c(0xc8a850), c(0x8a6a30), float64(y)/22))
			if y%3 == 0 {
				b.rect(28-w/2, 2+y, w, 1, c(0xa88838))
			}
		}
	}}

	// ---- vehicles: the park jeep (faces right: headlight on the right) ---------
	pk["jp_jeep"] = &PropKind{W: 60, H: 34, FW: 52, FH: 14, Vars: 3, Category: "vehicle", Draw: vaJeepSide} // vehart.go
	pk["jp_boat"] = &PropKind{W: 124, H: 66, FW: 100, FH: 20, NoSolid: true, Draw: func(b *pixBuf, v int) {
		b.ellipse(62, 60, 56, 4, color.RGBA{255, 255, 255, 70})
		genFerry(b) // vessels.go: the same style as every other boat
	}}
	pk["jp_fin"] = &PropKind{W: 40, H: 26, FW: 8, FH: 4, NoSolid: true, Vars: 2, Draw: func(b *pixBuf, v int) {
		if v == 1 { // a rubber duck, somebody lost it
			b.ellipse(20, 20, 9, 4, c(0xf8d030))
			b.ellipse(26, 13, 4, 4, c(0xf8d030))
			b.rect(29, 13, 4, 2, c(0xf08020))
			b.set(27, 12, c(0x14101a))
			b.ellipse(20, 24, 12, 2, color.RGBA{255, 255, 255, 80})
			return
		}
		for y := 0; y < 18; y++ {
			w := 18 - y
			b.rect(12, 4+y, w, 1, mixc(c(0x5a6a7a), c(0x3a4658), float64(y)/18))
		}
		b.rect(12, 4, 2, 18, c(0x7a8a9a))
		b.ellipse(20, 23, 16, 2.4, color.RGBA{255, 255, 255, 110})
		b.ellipse(20, 23, 10, 1.4, color.RGBA{255, 255, 255, 160})
	}}
	pk["jp_goat"] = &PropKind{W: 28, H: 22, FW: 14, FH: 5, Draw: func(b *pixBuf, v int) {
		b.ellipse(14, 20, 9, 1.6, color.RGBA{0, 0, 0, 70})
		b.ellipse(15, 12, 8, 5, c(0xf0ece0))
		b.ellipse(15, 13, 7, 3, c(0xd8d2c0))
		for _, lx := range []int{9, 12, 18, 21} {
			b.rect(lx, 14, 1, 6, c(0xc8c2b0))
		}
		b.ellipse(7, 9, 3.4, 3, c(0xf0ece0))
		b.rect(4, 7, 1, 3, c(0xc8b890))
		capsule(b, 7, 6, 5, 2, 0.8, 0.5, c(0xb8a878))
		b.set(6, 9, c(0x14101a))
		b.set(3, 11, c(0xe8e0c8))
		b.set(4, 12, c(0xe8e0c8))
		b.set(23, 10, c(0xd8d2c0))
		b.rect(2, 13, 4, 1, c(0xc83030)) // tied up and bored
	}}
	// a nest of three eggs. 0 still, 1 and 2 wobbling, 3 cracking with a baby peeping out, 4 hatched
	pk["jp_egg"] = &PropKind{W: 34, H: 28, FW: 22, FH: 6, NoSolid: true, Vars: 5, Draw: func(b *pixBuf, v int) {
		b.ellipse(17, 24, 14, 3.4, c(0x5a4630))
		b.ellipse(17, 23, 12, 2.6, c(0x8a7048))
		for k, x := range []float64{9, 17, 25} {
			col := c(0xe8e4cc)
			if k == 2 {
				col = c(0xcfd8b0)
			}
			dx, tilt := 0.0, 0.0
			if k == 1 && (v == 1 || v == 2) { // the middle egg rocks
				dx = float64(v)*2 - 3
				tilt = dx * 0.5
			}
			if k == 1 && v >= 3 {
				continue // drawn below
			}
			b.ellipse(x+dx, 17, 4.4, 5.6, shade(col, 0.7))
			b.ellipse(x+dx-0.6, 16.4+tilt*0, 3.8, 5, col)
			b.set(int(x+dx)-2, 14, c(0xffffff))
		}
		if v >= 3 { // the shell broken open and a baby inside
			b.ellipse(17, 19, 4.6, 3.4, shade(c(0xe8e4cc), 0.7))
			b.ellipse(17, 18.4, 4, 2.8, c(0xe8e4cc))
			for k := 0; k < 4; k++ { // jagged rim
				b.rect(13+k*2, 14+(k%2), 2, 2, c(0xe8e4cc))
			}
			b.rect(15, 10, 5, 5, c(0x6aa84a)) // the head
			b.rect(15, 10, 5, 1, c(0x8acc60))
			b.set(16, 12, c(0x14101a))
			b.set(18, 12, c(0x14101a))
			if v == 4 {
				b.rect(17, 14, 2, 1, c(0xd04040)) // squeaking
				b.rect(14, 8, 1, 2, c(0x6aa84a))
				b.rect(20, 8, 1, 2, c(0x6aa84a))
			}
		}
	}}
	pk["jp_warn"] = &PropKind{W: 36, H: 46, FW: 8, FH: 4, Draw: func(b *pixBuf, v int) {
		b.ellipse(18, 43, 9, 1.8, color.RGBA{0, 0, 0, 70})
		b.rect(16, 22, 4, 22, c(0x6a4a2a))
		for y := 0; y < 22; y++ {
			w := y + 2
			b.rect(18-w/2-1, 2+y, w+2, 1, c(0x20202a))
		}
		for y := 0; y < 19; y++ {
			w := y
			b.rect(18-w/2, 4+y, w+1, 1, c(0xf8d030))
		}
		b.rect(16, 11, 3, 3, c(0x20202a)) // a tiny dinosaur chasing a stick man
		b.rect(19, 13, 4, 2, c(0x20202a))
		b.rect(13, 15, 1, 3, c(0x20202a))
		b.set(13, 14, c(0x20202a))
		b.rect(21, 14, 1, 1, c(0xf8d030))
		b.rect(15, 18, 6, 1, c(0x20202a))
	}}
	pk["jp_amber"] = &PropKind{W: 30, H: 46, FW: 22, FH: 8, Draw: func(b *pixBuf, v int) {
		b.ellipse(15, 43, 12, 2.4, color.RGBA{0, 0, 0, 70})
		stoneBlock(b, 4, 26, 22, 18, c(0x8a8478), 4)
		b.rect(2, 24, 26, 3, c(0xb0aa9c))
		b.ellipse(15, 14, 11, 12, c(0xa86a10))
		b.ellipse(15, 13, 10, 11, c(0xe8a020))
		b.ellipse(12, 9, 4, 5, c(0xfcd870))
		b.ellipse(16, 14, 2.2, 1.4, c(0x3a2410)) // the famous mosquito
		b.line(12, 12, 9, 9, c(0x6a4a20))
		b.line(18, 12, 22, 9, c(0x6a4a20))
		b.set(14, 14, c(0x3a2410))
	}}

	// everything that is a building is drawn smaller so it gets in the way less
	for name, f := range map[string]float64{"jp_rotunda": 0.6, "jp_lab": 0.6, "jp_control": 0.58, "jp_dish": 0.62, "jp_mast": 0.6,
		"jp_tower": 0.62, "jp_lighthouse": 0.58, "jp_hut": 0.64} {
		k := pk[name]
		ow, oh, old := k.W, k.H, k.Draw
		k.W, k.H = int(float64(ow)*f), int(float64(oh)*f)
		k.FW, k.FH = math.Max(6, k.FW*f), math.Max(4, k.FH*f)
		k.Draw = func(b *pixBuf, v int) {
			tmp := newPix(ow, oh)
			old(tmp, v)
			sm := boxScale(tmp, 0, 0, ow, oh, b.w, b.h)
			copy(b.p, sm.p)
		}
	}
}
