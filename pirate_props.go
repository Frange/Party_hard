package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// The props of the BARCO PIRATA party (pirate.go): masts with their sails and the skull flag (the flag waves: two
// frames per style, see pirateAnim), cannons on their carriages (the ones of the north rail are a trap: the cannon fires), the rum rack (drinks,
// can be poisoned), barrels and chests to hide in, the treasure (picked up for points: items.go), the helm, the
// captain's table, lanterns, coiled ropes, the anchor and the parrot on its perch. Pixel art in the style of the
// rest of the game: flat shaded bands, a light edge on the left, a dark one on the right.
// ---------------------------------------------------------------------------

// pirateTreasurePts: what the treasure of the bow is worth.
const pirateTreasurePts = 100

// pirateAnim: props whose variants come in pairs (style*2 + frame) and swap the frame with the clock (draw.go drawProp).
var pirateAnim = map[string]float64{"pmast": 2.5, "pparrot": 1.5}

// pirateAnimVar: the variant to draw at time t (the frame of the pair picked by the clock and the place).
func pirateAnimVar(name string, v int, t, x float64) int {
	fps, ok := pirateAnim[name]
	if !ok {
		return v
	}
	return v&^1 | (int(t*fps)+int(x))%2
}

const (
	pmastW, pmastH = 104, 176
)

func init() {
	pk := propKinds
	pk["pmast"] = &PropKind{W: pmastW, H: pmastH, FW: 12, FH: 7, Vars: 4, Draw: drawPirateMast}
	pk["pcannon"] = &PropKind{W: 26, H: 24, FW: 22, FH: 9, Vars: 2, Trap: "explode", Draw: drawPirateCannon}
	// the cannons of the south rail are run out against it: the same drawing, but nobody can stand behind them to
	// light the fuse (their front is over the water), so they are not a trap
	pk["pcannonr"] = &PropKind{W: 26, H: 24, FW: 22, FH: 9, Vars: 2, Draw: drawPirateCannon}
	pk["prum"] = &PropKind{W: 42, H: 36, FW: 38, FH: 10, Vars: 2, UseKind: "drink", Use: []Vec{{-12, 12}, {12, 12}}, Poison: true,
		Draw: drawPirateRum}
	pk["pbarrel"] = &PropKind{W: 22, H: 28, FW: 18, FH: 8, Vars: 2, Hide: true, Draw: func(b *pixBuf, v int) {
		wood := []uint32{0x7a5034, 0x6a4a36}[v%2]
		pirateBarrel(b, 2, 3, 18, 24, rgb(wood))
	}}
	pk["pchest"] = &PropKind{W: 30, H: 24, FW: 26, FH: 9, Vars: 2, Hide: true, Draw: func(b *pixBuf, v int) {
		wood := []uint32{0x6a3e24, 0x4a3a52}[v%2]
		b.ellipse(15, 22, 14, 2, color.RGBA{0, 0, 0, 70})
		b.cube(1, 9, 28, 13, 3, shade(rgb(wood), 1.15), rgb(wood))
		for y := 3; y < 10; y++ { // the rounded lid
			w := int(math.Round(13 * math.Sqrt(1-math.Pow(float64(10-y)/8, 2))))
			b.rect(15-w-1, y, 2*w+2, 1, shade(rgb(wood), 1.1-0.03*float64(9-y)))
		}
		gold := rgb(0xc8a040)
		for _, x := range []int{4, 24} { // iron bands and the lock
			b.rect(x, 3, 2, 19, rgb(0x3a3a42))
		}
		b.rect(1, 9, 28, 1, rgb(0x3a3a42))
		b.rect(13, 10, 4, 5, gold)
		b.set(15, 12, rgb(0x2a1a10))
	}}
	pk["ptreasure"] = &PropKind{W: 26, H: 22, FW: 18, FH: 6, NoSolid: true, Item: "treasure", Draw: func(b *pixBuf, v int) {
		wood := rgb(0x6a3e24)
		b.ellipse(13, 20, 12, 2, color.RGBA{0, 0, 0, 70})
		b.cube(2, 10, 22, 10, 2, shade(wood, 1.15), wood)
		b.rect(2, 2, 22, 6, shade(wood, 0.8)) // the open lid behind
		b.rect(3, 3, 20, 4, rgb(0x2a1a10))
		gold, goldL, goldD := rgb(0xd8b048), rgb(0xf4e090), rgb(0x9a7428)
		for i := 0; i < 26; i++ { // the coins heaped up
			x := 3 + phHash(i, 3)%20
			y := 7 + phHash(i, 5)%5
			b.rect(x, y, 2, 1, []color.RGBA{gold, goldL, goldD}[i%3])
		}
		b.rect(9, 6, 2, 2, rgb(0xd04060)) // a ruby and a pearl necklace
		b.rect(15, 7, 1, 1, rgb(0xf4f0f4))
		b.rect(16, 6, 1, 1, rgb(0xf4f0f4))
		b.rect(17, 7, 1, 1, rgb(0xf4f0f4))
		b.rect(2, 10, 22, 1, rgb(0x3a3a42))
		b.rect(11, 12, 4, 4, gold)
	}}
	pk["phelm"] = &PropKind{W: 30, H: 36, FW: 12, FH: 6, Draw: func(b *pixBuf, v int) {
		wood, woodD, woodL := rgb(0x7a5034), rgb(0x4a2e1e), rgb(0xa07450)
		b.ellipse(15, 34, 8, 2, color.RGBA{0, 0, 0, 70})
		b.rect(12, 20, 6, 15, woodD) // the pedestal
		b.rect(12, 20, 2, 15, wood)
		cx, cy := 15.0, 14.0
		for a := 0.0; a < 2*math.Pi; a += math.Pi / 4 { // the spokes and their handles
			for r := 0.0; r <= 13; r++ {
				b.set(int(math.Round(cx+math.Cos(a)*r)), int(math.Round(cy+math.Sin(a)*r)), wood)
			}
			b.ellipse(cx+math.Cos(a)*13, cy+math.Sin(a)*13, 1.6, 1.6, woodL)
		}
		for a := 0.0; a < 2*math.Pi; a += 0.05 { // the rim
			b.rect(int(math.Round(cx+math.Cos(a)*9)), int(math.Round(cy+math.Sin(a)*9)), 2, 1, woodD)
		}
		b.ellipse(cx, cy, 3, 3, rgb(0xc8a040)) // the brass hub
		b.set(14, 13, rgb(0xf0d880))
	}}
	pk["pcaptable"] = &PropKind{W: 58, H: 32, FW: 54, FH: 12, UseKind: "drink", Use: []Vec{{-16, 12}, {16, 12}}, Draw: func(b *pixBuf, v int) {
		top, side := rgb(0x7a5034), rgb(0x4a2e1e)
		b.ellipse(29, 30, 27, 2, color.RGBA{0, 0, 0, 70})
		for _, x := range []int{4, 50} {
			b.rect(x, 18, 4, 12, side)
		}
		b.cube(1, 8, 56, 12, 8, top, side)
		b.rect(10, 9, 22, 7, rgb(0xd8c8a0)) // the map
		b.rect(10, 9, 22, 1, rgb(0xb8a880))
		for i := 0; i < 9; i++ { // the dotted route and the X
			b.set(12+i*2, 13-i%3/2, rgb(0x8a3a2a))
		}
		b.set(29, 11, rgb(0xc03020))
		b.set(30, 12, rgb(0xc03020))
		b.set(30, 11, rgb(0xc03020))
		b.set(29, 12, rgb(0xc03020))
		b.rect(38, 2, 3, 9, rgb(0xe8e0d0)) // a candle
		b.rect(39, 0, 1, 2, rgb(0xf0a050))
		b.rect(44, 4, 4, 7, rgb(0x2a4a2a)) // a bottle of rum
		b.rect(45, 1, 2, 3, rgb(0x2a4a2a))
		b.rect(45, 6, 2, 3, rgb(0xd8c8a0))
		b.rect(6, 6, 3, 4, rgb(0xa8a8b0)) // goblets
		b.rect(6, 10, 3, 1, rgb(0x6a6a74))
		b.rect(51, 7, 3, 3, rgb(0xa8a8b0))
	}}
	pk["plantern"] = &PropKind{W: 12, H: 42, FW: 6, FH: 4, Light: true, LightR: 64, LightC: color.RGBA{150, 96, 70, 100}, Draw: func(b *pixBuf, v int) {
		b.ellipse(6, 41, 4, 1.5, color.RGBA{0, 0, 0, 70})
		b.rect(5, 12, 2, 29, rgb(0x4a2e1e)) // the post
		b.rect(5, 12, 1, 29, rgb(0x7a5034))
		b.rect(2, 3, 8, 10, rgb(0x2a2a30)) // the lantern
		b.rect(3, 5, 6, 6, rgb(0xe8a060))
		b.rect(4, 6, 4, 4, rgb(0xf8d0a0))
		b.rect(2, 2, 8, 1, rgb(0x4a4a54))
		b.rect(5, 0, 2, 2, rgb(0x4a4a54))
	}}
	pk["pcoil"] = &PropKind{W: 20, H: 12, FW: 14, FH: 5, NoSolid: true, Vars: 2, Draw: func(b *pixBuf, v int) {
		rope, ropeD := rgb(0xb8a080), rgb(0x7a6a50)
		for r := 8.0; r >= 2; r -= 2 {
			b.ellipse(10, 7, r, r*0.55, ropeD)
			b.ellipse(10, 7, r-1, (r-1)*0.55, rope)
		}
		b.ellipse(10, 7, 1.2, 0.8, ropeD)
		if v%2 == 1 { // a loose end
			b.rect(16, 9, 4, 1, rope)
		}
	}}
	pk["panchor"] = &PropKind{W: 36, H: 26, FW: 28, FH: 8, Draw: func(b *pixBuf, v int) {
		b.ellipse(18, 24, 16, 2, color.RGBA{0, 0, 0, 70})
		anchorBig(b)
		for i := 0; i < 8; i++ { // a bit of chain
			b.rect(1+i*2, 20+(i%2), 2, 1, rgb(0x5a5a64))
		}
	}}
	pk["pparrot"] = &PropKind{W: 18, H: 38, FW: 6, FH: 4, Vars: 2, Draw: drawPirateParrot}
	overlayKinds["pcoil"] = true
}

// pirateBarrel: an upright barrel of w x h with its top-left at (x,y).
func pirateBarrel(b *pixBuf, x, y, w, h int, wood color.RGBA) {
	cx := float64(x) + float64(w)/2
	b.ellipse(cx, float64(y+h), float64(w)/2, 2, color.RGBA{0, 0, 0, 70})
	for yy := y + 3; yy < y+h; yy++ {
		k := float64(yy-y-3) / float64(h-3)
		bulge := int(math.Round(1.5 * math.Sin(math.Pi*k)))
		x0, x1 := x-bulge, x+w+bulge
		for xx := x0; xx < x1; xx++ {
			cc := wood
			switch {
			case xx < x0+3:
				cc = shade(wood, 1.18)
			case xx >= x1-3:
				cc = shade(wood, 0.72)
			case (xx-x0)%5 == 4:
				cc = shade(wood, 0.85) // the staves
			}
			b.set(xx, yy, cc)
		}
	}
	for _, yy := range []int{y + 6, y + h - 5} { // the iron hoops
		b.rect(x-1, yy, w+2, 2, rgb(0x3a3a42))
		b.rect(x-1, yy, w+2, 1, rgb(0x6a6a74))
	}
	b.ellipse(cx, float64(y+3), float64(w)/2, 3, shade(wood, 1.25)) // the head
	b.ellipse(cx, float64(y+3), float64(w)/2-2, 1.8, shade(wood, 1.05))
}

// anchorBig: the anchor lying on the forecastle (36x22).
func anchorBig(b *pixBuf) {
	iron, light := rgb(0x3a3a44), rgb(0x7a7a88)
	b.rect(15, 1, 6, 5, iron) // the ring
	b.rect(17, 2, 2, 3, rgb(0x14101a))
	b.rect(16, 6, 4, 14, iron) // the shank
	b.rect(16, 6, 1, 14, light)
	b.rect(8, 7, 20, 3, rgb(0x5a3a26)) // the wooden stock
	b.rect(8, 7, 20, 1, rgb(0x8a6040))
	for i := 0; i < 10; i++ { // the arms curving up
		y := 19 - i*i/12
		b.rect(17-i-2, y, 3, 2, iron)
		b.rect(17+i, y, 3, 2, iron)
	}
	b.rect(3, 10, 4, 4, iron) // the flukes
	b.rect(29, 10, 4, 4, iron)
	b.rect(3, 10, 4, 1, light)
	b.rect(29, 10, 4, 1, light)
}

// drawPirateMast: v/2 is the style (0: the main mast with the big sail set and the skull flag; 1: sails furled on
// their yards and a red pennant), v%2 the frame of the waving flag. The foot of the mast is on the bottom row.
func drawPirateMast(b *pixBuf, v int) {
	style, frame := v/2, v%2
	pole, poleL, poleD := rgb(0x5a3a26), rgb(0x8a6040), rgb(0x34221a)
	cx := pmastW / 2
	foot := pmastH - 1
	top := 14
	if style == 1 {
		top = 30
	}
	rope := rgb(0x3a2c22)
	// the shrouds from the top of the mast to the deck
	for _, ex := range []int{6, pmastW - 7} {
		x0, y0, x1, y1 := cx, top+10, ex, foot-4
		n := y1 - y0
		for i := 0; i <= n; i += 2 {
			b.set(x0+(x1-x0)*i/n, y0+i, rope)
		}
	}
	// the partner (the ring round the foot) and the mast itself, slightly tapered
	b.ellipse(float64(cx), float64(foot-2), 9, 3, poleD)
	b.ellipse(float64(cx), float64(foot-3), 7, 2, rgb(0x6a4a34))
	for y := top; y < foot-2; y++ {
		w := 4
		if y > top+50 {
			w = 5
		}
		x0 := cx - w/2
		b.rect(x0, y, w, 1, pole)
		b.set(x0, y, poleL)
		b.set(x0+w-1, y, poleD)
	}
	b.ellipse(float64(cx), float64(top-1), 2, 2, rgb(0xc8a040)) // the truck on top
	sail, sailD := rgb(0xd4c8b0), rgb(0xaea296)
	yard := func(y, half int) {
		b.rect(cx-half, y, 2*half, 2, pole)
		b.rect(cx-half, y, 2*half, 1, poleL)
	}
	furled := func(y, half int) { // the sail rolled up on top of its yard
		for x := cx - half + 3; x < cx+half-3; x++ {
			cc := sail
			if (x/5)%2 == 0 {
				cc = sailD
			}
			b.rect(x, y-3, 1, 3, cc)
		}
		for x := cx - half + 8; x < cx+half-6; x += 12 { // the gaskets
			b.rect(x, y-3, 1, 3, rope)
		}
	}
	if style == 1 {
		yard(56, 30)
		furled(56, 30)
		yard(100, 40)
		furled(100, 40)
		// a red pennant on the top
		for i := 0; i < 14; i++ {
			dy := int(math.Round(1.2 * math.Sin(float64(i)*0.6+float64(frame)*math.Pi)))
			h := 4 - i/4
			b.rect(cx+2+i, top+dy, 1, h, rgb(0xb03030))
		}
		return
	}
	// the main mast: the topsail furled, the crow's nest, the big sail set and the black flag
	yard(46, 32)
	furled(46, 32)
	b.rect(cx-10, 68, 20, 8, rgb(0x5a3a26)) // the crow's nest
	b.rect(cx-10, 68, 20, 2, rgb(0x8a6040))
	b.rect(cx-10, 74, 20, 2, poleD)
	yard(88, 44)
	for y := 90; y < 134; y++ { // the sail, swelling in the wind
		k := float64(y-90) / 43
		bulge := int(math.Round(3 * math.Sin(math.Pi*k)))
		xl, xr := cx-42-bulge, cx+42+bulge
		for x := xl; x < xr; x++ {
			cc := sail
			switch {
			case x > xr-10:
				cc = sailD
			case x < xl+3:
				cc = shade(sail, 1.06)
			case (y-90)%9 == 8:
				cc = shade(sail, 0.92) // the seams of the cloths
			}
			b.set(x, y, cc)
		}
	}
	b.rect(cx-30, 100, 8, 6, rgb(0xb8a888)) // a patch
	b.rect(cx-30, 100, 8, 1, shade(rgb(0xb8a888), 0.85))
	// the skull and crossbones painted on the sail
	ink := rgb(0x2a2228)
	b.ellipse(float64(cx), 108, 7, 6, ink)
	b.rect(cx-4, 112, 8, 4, ink)
	b.rect(cx-4, 106, 3, 3, sail) // the eyes
	b.rect(cx+1, 106, 3, 3, sail)
	b.set(cx, 110, sail)
	for i := 0; i < 9; i++ { // the bones
		b.rect(cx-12+i*3, 118+i/2, 3, 2, ink)
		b.rect(cx+9-i*3, 118+i/2, 3, 2, ink)
	}
	// the sheets from the corners of the sail down to the rails
	for i := 0; i < 36; i += 2 {
		b.set(cx-42-i/3, 134+i, rope)
		b.set(cx+41+i/3, 134+i, rope)
	}
	// the black flag with the skull, waving from the top of the mast
	for i := 0; i < 24; i++ {
		dy := int(math.Round(1.5 * math.Sin(float64(i)*0.45+float64(frame)*math.Pi)))
		b.rect(cx+2+i, 1+dy, 1, 13, rgb(0x16141a))
		if i == 23 {
			b.rect(cx+2+i, 1+dy, 1, 13, rgb(0x2a2630))
		}
	}
	sk := rgb(0xece8e0)
	dy := int(math.Round(1.5 * math.Sin(10*0.45+float64(frame)*math.Pi)))
	b.rect(cx+11, 3+dy, 6, 5, sk) // the skull of the flag
	b.rect(cx+12, 8+dy, 4, 1, sk)
	b.set(cx+12, 5+dy, rgb(0x16141a))
	b.set(cx+15, 5+dy, rgb(0x16141a))
	for i := 0; i < 4; i++ {
		b.set(cx+9+i*3/1, 10+dy+i/2, sk)
		b.set(cx+18-i*3/1, 10+dy+i/2, sk)
	}
}

// drawPirateCannon: v 0 points north (over the rail behind it), v 1 points south, at the camera.
func drawPirateCannon(b *pixBuf, v int) {
	iron, ironL, ironD := rgb(0x2e2e36), rgb(0x5a5a66), rgb(0x16161c)
	wood, woodD := rgb(0x6a4430), rgb(0x3e281c)
	b.ellipse(13, 22, 12, 2, color.RGBA{0, 0, 0, 70})
	b.cube(3, 12, 20, 10, 3, wood, woodD) // the carriage
	for _, x := range []float64{5, 21} {  // the wheels
		b.ellipse(x, 20, 3, 3, woodD)
		b.ellipse(x, 20, 1.2, 1.2, iron)
	}
	if v%2 == 0 { // the barrel going back over the rail, foreshortened
		for y := 1; y < 14; y++ {
			w := 4 + y/4
			b.rect(13-w/2, y, w, 1, iron)
			b.set(13-w/2, y, ironL)
			b.set(13-w/2+w-1, y, ironD)
		}
		b.ellipse(13, 2, 2.5, 1.5, ironD) // the muzzle
		b.rect(11, 6, 5, 1, ironL)
		return
	}
	// pointing at us: the breech on the carriage and the round muzzle in front
	b.rect(9, 8, 8, 6, iron)
	b.rect(9, 8, 8, 1, ironL)
	b.ellipse(13, 15, 5, 4.5, iron)
	b.ellipse(13, 15, 3.4, 3, ironD)
	b.ellipse(13, 15, 2, 1.8, rgb(0x050506))
	b.set(10, 13, ironL)
	b.rect(12, 5, 2, 3, rgb(0xb8a080)) // the fuse
}

// drawPirateRum: a rack of rum barrels with a tap and a couple of tankards (v: two woods).
func drawPirateRum(b *pixBuf, v int) {
	wood := rgb([]uint32{0x7a5034, 0x6a4a36}[v%2])
	b.ellipse(21, 34, 20, 2, color.RGBA{0, 0, 0, 70})
	b.cube(1, 22, 40, 12, 3, rgb(0x5a3a26), rgb(0x3e281c)) // the rack
	for i, x := range []int{2, 22} {                       // two barrels lying on the rack, heads to the front
		cx := float64(x) + 9
		b.ellipse(cx, 18, 9, 8, shade(wood, 0.8))
		b.ellipse(cx, 18, 7.5, 6.5, wood)
		b.ellipse(cx, 18, 5, 4, shade(wood, 1.15))
		b.rect(int(cx)-8, 17, 16, 1, rgb(0x3a3a42))
		b.rect(int(cx)-1, 20, 2, 3, rgb(0xc8a040)) // the tap
		_ = i
	}
	pirateBarrel(b, 13, 0, 14, 12, wood) // a small cask on top
	b.rect(13, 3, 14, 2, rgb(0xe8dcc0))  // its label: XXX
	for _, x := range []int{15, 19, 23} {
		b.set(x, 3, rgb(0x2a1a10))
		b.set(x+1, 4, rgb(0x2a1a10))
	}
	for _, x := range []int{3, 34} { // tankards
		b.rect(x, 26, 5, 6, rgb(0x8a8a94))
		b.rect(x, 26, 5, 1, rgb(0xe8dcc0))
		b.rect(x+5, 28, 1, 3, rgb(0x6a6a74))
	}
}

// drawPirateParrot: a red macaw on its T perch; the two frames lift the wings.
func drawPirateParrot(b *pixBuf, v int) {
	frame := v % 2
	b.ellipse(9, 36, 5, 1.5, color.RGBA{0, 0, 0, 70})
	b.rect(8, 14, 2, 22, rgb(0x5a3a26)) // the stand
	b.rect(4, 35, 10, 2, rgb(0x3e281c))
	b.rect(2, 13, 14, 2, rgb(0x7a5034)) // the bar
	red, blue, yellow := rgb(0xc83028), rgb(0x2a5ab0), rgb(0xe8c040)
	b.rect(9, 1, 5, 5, red) // the head
	b.rect(13, 3, 2, 3, rgb(0xe8e0d0))
	b.set(14, 4, rgb(0x2a2a30))
	b.set(13, 2, rgb(0xf4f4f4))
	b.set(12, 2, rgb(0x14101a)) // the eye
	b.rect(8, 6, 6, 7, red)     // the body
	b.rect(8, 9, 2, 3, yellow)
	if frame == 0 {
		b.rect(6, 7, 3, 6, blue) // the folded wing
		b.rect(6, 11, 3, 1, yellow)
	} else {
		b.rect(3, 4, 5, 3, blue) // the wing up
		b.rect(2, 3, 3, 2, blue)
		b.rect(4, 6, 3, 1, yellow)
	}
	b.rect(9, 13, 3, 1, rgb(0x6a6a74)) // the feet
	b.rect(8, 13, 2, 8, blue)          // the long tail
	b.rect(8, 18, 2, 3, red)
}
