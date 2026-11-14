package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// FIRE ART (v1.4.0, "fuego más realista"): one pixel-art flame painter shared by every fire of the game: the
// arsonist's and the cook's fires (fire.go), the burning wrecks (killer.go drawFlames), the hearth of the fireplace
// (props2.go + its moving frames in propanim.go) and the flare of the barbecue (pe_bbq.go, rect version).
//
// The flame is drawn in layers like a real one: a pale yellow core low in the middle, yellow and orange around it,
// red and a dark red rim at the edges and the tips; the tongues lean and wave, little detached tongues break off the
// top and rise, sparks climb and die out (yellow -> orange -> red -> gone) and a bed of embers glows at the foot.
// Everything is a periodic function of the frame f of n, so a loop of frames closes without a jump.
//
// CHEAP: the frames are painted ONCE into images (fireSets) and the game only picks one by its clock; the images are
// drawn at scale 1 on whole pixels (sharp), the size of a fire is chosen among three painted sizes.
// ---------------------------------------------------------------------------

// fire palette, hot -> cold
var fireRamp = [...]color.RGBA{
	{0xff, 0xf6, 0xcc, 255}, // core
	{0xff, 0xdc, 0x4c, 255}, // yellow
	{0xfc, 0xa4, 0x2c, 255}, // light orange
	{0xf0, 0x6a, 0x1c, 255}, // orange
	{0xc8, 0x36, 0x1a, 255}, // red
	{0x7c, 0x1c, 0x14, 255}, // dark red rim
}

// fireHash: a small integer hash for the ragged pixels (deterministic).
func fireHash(a, b, c int) int {
	h := uint32(a)*374761393 + uint32(b)*668265263 + uint32(c)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	return int((h ^ (h >> 16)) & 0x7fffffff)
}

func fireFrac(x float64) float64 { return x - math.Floor(x) }

// paintFire paints frame f (of a loop of n) of a flame whose foot is centred at (cx, by), w wide and h tall, into
// b. clip (when not empty) limits the pixels it may touch (the opening of a fireplace). embers adds the glowing bed.
func paintFire(b *pixBuf, cx, by, w, h, f, n, seed int, clip image.Rectangle, embers bool) {
	put := func(x, y int, col color.RGBA) {
		if !clip.Empty() && !(image.Point{x, y}).In(clip) {
			return
		}
		b.set(x, y, col)
	}
	f = ((f % n) + n) % n // the loop closes: frame n is frame 0
	u := float64(f) / float64(n)
	ph := 2 * math.Pi * u
	sp := float64(seed%7) * 0.9
	lean := float64(w) * 0.14
	// the body: row by row from the foot up
	for r := 0; r < h; r++ {
		t := float64(r) / float64(h)
		prof := math.Pow(1-t, 0.85) * (1 + 0.35*math.Sin(math.Pi*t))
		if t < 0.14 {
			prof *= 0.78 + t*1.6 // a rounded foot
		}
		hw := float64(w) / 2 * prof
		hw += 0.9 * t * math.Sin(t*9-2*ph+sp) // the edges ripple upwards
		xo := lean * math.Pow(t, 1.4) * math.Sin(ph+t*4.2+sp)
		if hw < 0.6 {
			continue
		}
		y := by - r
		for x := int(math.Floor(-hw - 1)); x <= int(math.Ceil(hw+1)); x++ {
			d := math.Abs(float64(x)-xo) / hw
			jag := float64(fireHash(x+seed*31, r, f)%100)/100*0.5 - 0.18
			if d > 1+jag*t { // ragged outline, more ragged near the tips
				continue
			}
			heat := (1 - math.Pow(math.Min(d, 1), 1.6)) * (1 - 0.9*math.Pow(t, 1.15))
			heat += float64(fireHash(x, r+seed, f*3)%100)/100*0.16 - 0.08
			put(cx+x, y, fireCol(heat))
		}
	}
	// detached tongues that break off the top and rise while they shrink
	for i := 0; i < 3; i++ {
		v := fireFrac(u*2 + float64(i)/3 + float64(seed%5)*0.11)
		ty := by - int(float64(h)*(0.62+0.45*v))
		tx := cx + int(math.Round(lean*math.Sin(ph+float64(i)*2.3+sp)+float64(i-1)*float64(w)*0.18))
		s := int(math.Round(3.2 * (1 - v)))
		for k := 0; k < s+1; k++ { // a little drop, wide at the bottom, pointed on top
			ww := s - k
			col := fireRamp[3]
			if v > 0.45 || k == s {
				col = fireRamp[4]
			}
			if v > 0.8 {
				col = fireRamp[5]
			}
			for x := -ww / 2; x <= ww/2; x++ {
				put(tx+x, ty-k, col)
			}
		}
	}
	// sparks climbing and dying out
	for i := 0; i < 5; i++ {
		v := fireFrac(u + float64(i)*0.37 + float64(seed%3)*0.13)
		if v > 0.9 {
			continue
		}
		sx := cx + int(math.Round(math.Sin(float64(i)*2.1+v*5+sp)*float64(w)*0.45))
		sy := by - int(float64(h)*(0.45+0.9*v))
		col := fireRamp[1]
		switch {
		case v > 0.65:
			col = fireRamp[5]
		case v > 0.4:
			col = fireRamp[4]
		case v > 0.18:
			col = fireRamp[2]
		}
		put(sx, sy, col)
	}
	if !embers {
		return
	}
	// the bed of embers
	for x := -w/2 - 1; x <= w/2+1; x++ {
		for r := 0; r < 2; r++ {
			k := fireHash(x+seed, r, f/2) % 10
			col := color.RGBA{0x4a, 0x16, 0x10, 255}
			switch {
			case k > 7:
				col = fireRamp[2]
			case k > 4:
				col = fireRamp[4]
			case k > 2:
				col = fireRamp[5]
			}
			put(cx+x, by+1+r, col)
		}
	}
}

// fireCol: the colour of a pixel of flame by its heat (1 core .. 0 rim).
func fireCol(heat float64) color.RGBA {
	switch {
	case heat > 0.66:
		return fireRamp[0]
	case heat > 0.5:
		return fireRamp[1]
	case heat > 0.36:
		return fireRamp[2]
	case heat > 0.22:
		return fireRamp[3]
	case heat > 0.1:
		return fireRamp[4]
	}
	return fireRamp[5]
}

// ---- the painted frames of the loose fires ----------------------------------------------------------------------

const fireFrameN = 8

// fireSizes: the painted sizes (flame w, h); the canvas leaves room around for the tongues and the sparks.
var fireSizes = [3][2]int{{8, 13}, {12, 19}, {16, 25}}

type fireFrame struct {
	img    *ebiten.Image
	ox, oy float64 // the foot of the flame inside the image
}

var fireSets [len(fireSizes)][fireFrameN]fireFrame

// buildFireSets paints the loose flames once.
func buildFireSets() {
	if fireSets[0][0].img != nil {
		return
	}
	for s, sz := range fireSizes {
		w, h := sz[0], sz[1]
		cw, ch := w+10, h*2+4
		for f := 0; f < fireFrameN; f++ {
			b := newPix(cw, ch)
			ox, oy := cw/2, ch-3
			paintFire(b, ox, oy, w, h, f, fireFrameN, s*3+1, image.Rectangle{}, false)
			fireSets[s][f] = fireFrame{img: b.img(), ox: float64(ox), oy: float64(oy)}
		}
	}
}

// fireSizeFor: the painted size for a flame of relative size sc (1 = the middle one).
func fireSizeFor(sc float64) int {
	switch {
	case sc < 0.6:
		return 0
	case sc < 0.95:
		return 1
	}
	return 2
}

// drawFireFrame draws one painted flame with its foot at screen point (x, y), on whole pixels.
func drawFireFrame(dst *ebiten.Image, size, f int, x, y float64) {
	buildFireSets()
	fr := &fireSets[size][((f%fireFrameN)+fireFrameN)%fireFrameN]
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(x)-fr.ox, math.Round(y)-fr.oy)
	dst.DrawImage(fr.img, op)
}

// ---- the hearth of the fireplace ---------------------------------------------------------------------------------

const hearthFrames = 8

// drawHearth paints the inside of the fireplace (the opening at 12,20 36x28 of the built-in drawing): soot, two
// logs, the embers and the flame of frame f (of hearthFrames). The still sprite is frame 0.
func drawHearth(b *pixBuf, f int) {
	b.rect(12, 20, 36, 28, c(0x1a1210))
	b.rect(13, 21, 34, 3, c(0x120c0a)) // soot at the top of the opening
	clip := image.Rect(12, 20, 48, 48)
	paintFire(b, 30, 42, 20, 20, f, hearthFrames, 2, clip, false)
	// the logs over the embers, in front of the foot of the flame
	b.rect(16, 43, 28, 3, c(0x3a2418))
	b.rect(18, 42, 11, 2, c(0x5a3420))
	b.rect(31, 42, 11, 2, c(0x5a3420))
	b.set(18, 42, c(0x7a4a2a))
	b.set(41, 42, c(0x7a4a2a))
	for x := 15; x < 46; x++ { // embers under and between the logs
		k := fireHash(x, 7, f/2) % 10
		col := c(0x4a1610)
		switch {
		case k > 7:
			col = fireRamp[2]
		case k > 4:
			col = fireRamp[4]
		case k > 2:
			col = fireRamp[5]
		}
		b.set(x, 46, col)
	}
}
