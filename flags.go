package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// FLAGS (v1.4.0, BARCELONA and PARÍS): a cloth that waves in the wind, drawn by code and animated (propAnims: the
// cloth ripples along its length, more at the free end than at the pole; the light and the shade move with the folds).
//   flagpole  a free-standing pole on a stone base (solid only at its base) with the flag at the top
//   flagmini  a short pole fixed to a balcony or a cornice (scenery of the painted backdrops: never blocks anything)
// Variants (flagSpain..flagFrance): 0 Spain (red, yellow, red: 1:2:1), 1 Catalonia (the senyera: four red stripes on
// yellow), 2 France (blue, white, red, upright).
// ---------------------------------------------------------------------------

const (
	flagSpain = iota
	flagCatalonia
	flagFrance
	flagVars
)

const (
	flagFrames = 6
	flagClothW = 24 // the cloth of the free-standing pole
	flagMiniW  = 16 // the cloth on a balcony
)

var (
	flagRed    = rgb(0xc8202a)
	flagYellow = rgb(0xf4c418)
	flagBlue   = rgb(0x1e3c9a)
	flagWhite  = rgb(0xf4f2ee)
)

// flagClothH: the height of the cloth (the senyera needs 9 even stripes: 2 px each on the big one).
func flagClothH(v, w int) int {
	if w <= flagMiniW {
		if v == flagCatalonia {
			return 9
		}
		return 10
	}
	if v == flagCatalonia {
		return 18
	}
	return 16
}

// flagColourAt: the colour of the flat cloth at (x, y) of a w x h cloth.
func flagColourAt(v, x, y, w, h int) color.RGBA {
	switch v {
	case flagSpain: // 1:2:1
		if y*4 < h || y*4 >= 3*h {
			return flagRed
		}
		return flagYellow
	case flagCatalonia: // nine stripes: yellow, red, yellow ... yellow
		if (y*9/h)%2 == 1 {
			return flagRed
		}
		return flagYellow
	}
	switch x * 3 / w { // France, from the pole outwards
	case 0:
		return flagBlue
	case 1:
		return flagWhite
	}
	return flagRed
}

// flagWave: how far column x of the cloth drops at frame f (0 at the pole), and how lit that fold is (-1..1).
func flagWave(x, w, f int) (dy int, lit float64) {
	ph := 2*math.Pi*float64(x)/12 - 2*math.Pi*float64(f)/flagFrames
	amp := 0.4 + 1.6*float64(x)/float64(w)
	dy = int(math.Round(amp * math.Sin(ph)))
	lit = math.Cos(ph) * float64(x) / float64(w)
	return
}

// drawFlagCloth paints the waving cloth with its top-left corner at the pole (px, py).
func drawFlagCloth(b *pixBuf, v, px, py, w, f int) {
	h := flagClothH(v, w)
	for x := 0; x < w; x++ {
		dy, lit := flagWave(x, w, f)
		for y := 0; y < h; y++ {
			c := flagColourAt(v, x, y, w, h)
			switch {
			case lit > 0.35:
				c = shade(c, 1.12)
			case lit < -0.35:
				c = shade(c, 0.78)
			}
			b.set(px+x, py+y+dy, c)
		}
	}
}

func init() {
	pk := propKinds
	pk["flagpole"] = &PropKind{W: 48, H: 70, FW: 8, FH: 5, Vars: flagVars, Draw: func(b *pixBuf, v int) { drawFlagpole(b, v, 0) }}
	pk["flagmini"] = &PropKind{W: 22, H: 18, FW: 4, FH: 3, NoSolid: true, Vars: flagVars, Draw: func(b *pixBuf, v int) { drawFlagMini(b, v, 0) }}
	overlayKinds["flagmini"] = true
	propAnims["flagpole"] = &propAnim{n: flagFrames, fps: 6, draw: func(b *pixBuf, k *PropKind, v, f int) { drawFlagpole(b, v, f) }}
	propAnims["flagmini"] = &propAnim{n: flagFrames, fps: 6, draw: func(b *pixBuf, k *PropKind, v, f int) { drawFlagMini(b, v, f) }}
}

// drawFlagpole: the base at the bottom centre (the anchor), the pole on the left of the sprite's middle, the flag at
// the top flying to the right.
func drawFlagpole(b *pixBuf, v, f int) {
	cx := 24 // the anchor column (W/2): the pole stands on it
	base, baseD := rgb(0xb8b0a0), rgb(0x7e7668)
	pole, poleL := rgb(0x6a6e78), rgb(0xc8ccd4)
	b.ellipse(float64(cx), 67, 8, 2.5, color.RGBA{0, 0, 0, 60})
	b.rect(cx-5, 62, 10, 6, base)
	b.rect(cx-5, 66, 10, 2, baseD)
	b.rect(cx-1, 4, 2, 60, pole)
	b.rect(cx-1, 4, 1, 60, poleL)
	b.rect(cx-2, 2, 4, 3, flagYellow) // the gilt knob
	b.rect(cx-1, 2, 1, 1, rgb(0xfff0a0))
	drawFlagCloth(b, v, cx+1, 6, flagClothW-1, f)
}

// drawFlagMini: a short pole sloping out of a wall (its foot is 9 px left of the anchor) and a small flag.
func drawFlagMini(b *pixBuf, v, f int) {
	pole := rgb(0x4a4e58)
	for i := 0; i < 6; i++ {
		b.set(2+i/2, 17-i*2, pole)
		b.set(2+i/2, 16-i*2, pole)
	}
	b.rect(4, 2, 1, 4, pole)
	b.set(4, 1, flagYellow)
	drawFlagCloth(b, v, 5, 2, flagMiniW, f)
}
