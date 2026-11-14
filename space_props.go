package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// The props of the ESTACIÓN ESPACIAL (ships.go genStation): a rocket on its launch stand, a hydroponic dome,
// a docked capsule module, a satellite dish and windows that look out at a planet. Static pixel art; the life
// of the level is in its background (space_fx.go: twinkling stars, a turning planet, ships passing).
// ---------------------------------------------------------------------------

func init() {
	pk := propKinds
	pk["srocket"] = &PropKind{W: 30, H: 64, FW: 24, FH: 10, Draw: drawSpaceRocket}
	pk["sdome"] = &PropKind{W: 44, H: 36, FW: 40, FH: 12, Draw: drawSpaceDome}
	pk["smodule"] = &PropKind{W: 50, H: 30, FW: 46, FH: 12, Draw: drawSpaceModule}
	pk["ssatdish"] = &PropKind{W: 34, H: 42, FW: 18, FH: 8, Draw: drawSpaceDish}
	pk["swindow"] = &PropKind{W: 36, H: 34, FW: 32, FH: 6, Vars: 3, Draw: drawSpaceWindow}
}

func drawSpaceRocket(b *pixBuf, v int) {
	b.ellipse(15, 62, 13, 2, color.RGBA{0, 0, 0, 80})
	steel, steelD := rgb(0x6a7080), rgb(0x3a4050)
	b.rect(2, 54, 26, 6, steelD) // the launch stand
	b.rect(2, 54, 26, 1, steel)
	b.rect(4, 20, 2, 34, steelD) // the gantry
	for y := 22; y < 54; y += 6 {
		b.rect(4, y, 6, 1, steel)
	}
	body, bodyL, bodyD := rgb(0xe8e8f0), rgb(0xffffff), rgb(0xa8acb8)
	for y := 6; y < 50; y++ { // the body, with a pointed nose
		w := 6
		if y < 16 {
			w = int(math.Round(6 * math.Sqrt(float64(y-5)/10)))
		}
		b.rect(17-w, y, 2*w, 1, body)
		b.set(17-w, y, bodyL)
		b.set(16+w, y, bodyD)
	}
	b.rect(11, 6, 12, 8, color.RGBA{}) // keep the nose pointed
	for y := 6; y < 16; y++ {
		w := int(math.Round(6 * math.Sqrt(float64(y-5)/10)))
		b.rect(17-w, y, 2*w, 1, rgb(0xd03a3a))
	}
	b.rect(11, 30, 12, 3, rgb(0xd03a3a))
	b.ellipse(17, 23, 3, 3, rgb(0x3a4a6a)) // the porthole
	b.ellipse(16, 22, 1.4, 1.4, rgb(0x9ad8ff))
	for _, f := range [][3]int{{7, 42, 4}, {23, 42, 4}} { // the fins
		b.rect(f[0], f[1], f[2], 10, rgb(0xd03a3a))
	}
	b.rect(13, 50, 8, 3, rgb(0x4a4a56)) // the nozzle
}

func drawSpaceDome(b *pixBuf, v int) {
	b.ellipse(22, 34, 21, 2, color.RGBA{0, 0, 0, 80})
	b.rect(1, 26, 42, 8, rgb(0x5a6070)) // the base ring
	b.rect(1, 26, 42, 1, rgb(0x9aa0b0))
	glass := rgba(0x8ad0ff, 150)
	for y := 2; y < 26; y++ {
		w := int(math.Round(20 * math.Sqrt(1-math.Pow(float64(26-y)/24, 2))))
		b.rect(22-w, y, 2*w, 1, glass)
	}
	green := []color.RGBA{rgb(0x3a9a4a), rgb(0x5ac05a), rgb(0x2a7a3a)}
	for i := 0; i < 7; i++ { // the plants inside
		x := 8 + i*4
		h := 6 + phHash(i, 11)%10
		b.rect(x, 26-h, 3, h, green[i%3])
		b.rect(x-1, 26-h, 5, 2, green[(i+1)%3])
	}
	for y := 4; y < 24; y += 5 { // the frame of the panes
		w := int(math.Round(20 * math.Sqrt(1-math.Pow(float64(26-y)/24, 2))))
		b.rect(22-w, y, 2*w, 1, rgba(0xd8f0ff, 180))
	}
	b.rect(21, 2, 2, 24, rgba(0xd8f0ff, 180))
	b.rect(10, 8, 2, 4, rgba(0xffffff, 220)) // the shine
}

func drawSpaceModule(b *pixBuf, v int) {
	b.ellipse(25, 28, 24, 2, color.RGBA{0, 0, 0, 80})
	body, bodyL, bodyD := rgb(0xc8ccd8), rgb(0xe8ecf4), rgb(0x8a8e9a)
	b.rect(4, 6, 42, 20, body) // a lying capsule
	b.rect(4, 6, 42, 3, bodyL)
	b.rect(4, 23, 42, 3, bodyD)
	b.ellipse(4, 16, 4, 10, bodyD)
	b.ellipse(46, 16, 4, 10, body)
	for x := 12; x < 42; x += 10 { // the ribs
		b.rect(x, 6, 1, 20, bodyD)
	}
	b.ellipse(26, 15, 5, 5, rgb(0x4a5060)) // the hatch
	b.ellipse(26, 15, 3.5, 3.5, rgb(0x2a3040))
	b.rect(24, 14, 5, 2, rgb(0xd0a030))
	b.rect(8, 10, 2, 2, rgb(0x40e060)) // status lights
	b.rect(8, 14, 2, 2, rgb(0xe04040))
	b.rect(30, 3, 10, 3, rgb(0x2a3a8a)) // a small solar panel on top
	b.rect(30, 3, 10, 1, rgb(0x6a8ad8))
}

func drawSpaceDish(b *pixBuf, v int) {
	b.ellipse(17, 40, 10, 2, color.RGBA{0, 0, 0, 80})
	b.rect(13, 30, 8, 10, rgb(0x4a5060)) // the pedestal
	b.rect(13, 30, 2, 10, rgb(0x7a8090))
	b.rect(16, 20, 2, 10, rgb(0x6a7080))
	dish, dishD := rgb(0xe0e4ec), rgb(0x9aa0ac)
	for y := 2; y < 22; y++ { // the dish, tilted up
		k := float64(y-12) / 10
		w := int(math.Round(15 * math.Sqrt(math.Max(0, 1-k*k))))
		b.rect(17-w, y, 2*w, 1, dish)
		b.set(16+w, y, dishD)
	}
	b.line(17, 12, 26, 4, rgb(0x5a6070)) // the feed arm
	b.rect(25, 2, 3, 3, rgb(0xd03a3a))
}

func drawSpaceWindow(b *pixBuf, v int) {
	frame, frameL := rgb(0x4a5060), rgb(0x8a90a0)
	b.rect(0, 0, 36, 28, frame)
	b.rect(0, 0, 36, 1, frameL)
	b.rect(3, 3, 30, 22, rgb(0x04061a)) // space behind the glass
	for i := 0; i < 9; i++ {
		b.set(4+phHash(i+v*9, 7)%28, 4+phHash(i+v*9, 13)%20, rgb(0xe0e0f0))
	}
	pl := []uint32{0xc06a40, 0x3a6ab8, 0x8a5ac8}[v%3]
	cx, cy, r := 22.0, 15.0, 7.0
	for y := -r; y < r; y++ {
		w := math.Sqrt(math.Max(0, r*r-y*y))
		b.rect(int(math.Round(cx-w)), int(cy+y), int(math.Round(2*w)), 1, shade(rgb(pl), 1-y/r*0.35))
	}
	if v%3 == 2 { // a ringed planet
		b.rect(12, 15, 21, 1, rgb(0xe0c8a0))
	}
	b.rect(5, 5, 2, 6, rgba(0xffffff, 90)) // reflection on the glass
	b.rect(4, 28, 28, 6, rgb(0x3a3e4a))    // the sill (its foot)
	b.rect(4, 28, 28, 1, frameL)
}
