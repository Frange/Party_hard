package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Ghosts of the cinematics (the conga of the secret ending, anything dead that comes back).
//
// drawGhostActor draws ANY CharSet as a ghost, from its normal sprite: translucent (the low alpha is
// intentional, only for ghosts), a cold pale blue tint with less colour, a body that fades and tapers
// downwards into a wavy tail (no feet), a slow vertical float (they glide, they do not walk), a faint
// halo, a trail behind, hollow glowing eyes and the odd gag (chains, "¡UUUH!").
// ---------------------------------------------------------------------------

// ghostLook: the numbers of the effect (kept together so a test can check them).
const (
	ghostAlphaMin = 0.45 // the most opaque a ghost gets at the head, with the wave
	ghostAlphaMax = 0.66
	ghostFadeFrom = 0.50 // the body starts to melt at this fraction of the sprite height
)

// layerRec: while it is not nil (only in tests) every layer of a scene says its name when it is drawn:
// "bg:*" (sky, buildings, ground), "actor:*" (people, ghosts, animals), "front:*" (declared foreground).
var layerRec *[]string

func recLayer(name string) {
	if layerRec != nil {
		*layerRec = append(*layerRec, name)
	}
}

// ghostAlphaAt: opacity of a row of the sprite (0 head .. 1 bottom): constant at the head, melting below.
func ghostAlphaAt(frac, base float64) float64 {
	if frac <= ghostFadeFrom {
		return base
	}
	k := (frac - ghostFadeFrom) / (1 - ghostFadeFrom)
	return base * math.Max(0, 1-k*k*1.15)
}

// ghostWave: the sideways ripple of a row (bigger at the tail).
func ghostWave(frac, t, seed float64) float64 {
	return math.Sin(t*3.4+frac*7+seed) * (0.35 + 2.6*frac*frac)
}

var ghostCM ebiten.ColorM

func init() {
	ghostCM.ChangeHSV(0, 0.5, 1)           // colours drained
	ghostCM.Scale(0.88, 0.96, 1.06, 1)     // cold
	ghostCM.Translate(0.07, 0.10, 0.15, 0) // pale
}

// drawGhostImage: the body. left/top place the sprite, flip mirrors it; seed desynchronises the waves.
func drawGhostImage(dst, img *ebiten.Image, left, top float64, flip bool, t, alpha, seed float64) {
	if img == nil {
		return
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	for y := 0; y < h; y++ {
		frac := float64(y) / float64(h)
		a := ghostAlphaAt(frac, alpha)
		if a <= 0.01 {
			continue
		}
		// the lower part narrows like the tail of a sheet
		in := 0
		if frac > 0.62 {
			in = int((frac - 0.62) / 0.38 * float64(w) * 0.30)
		}
		if w-2*in < 2 {
			continue
		}
		row := img.SubImage(image.Rect(b.Min.X+in, b.Min.Y+y, b.Max.X-in, b.Min.Y+y+1)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(in), 0)
		if flip {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(float64(w), 0)
		}
		op.GeoM.Translate(math.Round(left+ghostWave(frac, t, seed)), math.Round(top)+float64(y))
		op.ColorM = ghostCM
		op.ColorScale.ScaleAlpha(float32(a))
		dst.DrawImage(row, op)
	}
}

// drawGhostActor draws a CharSet as a ghost standing on the ground line footY, centred on x.
// trail: +1 the wake is on the right, -1 on the left, 0 none. idx numbers the ghost (gags differ).
func (g *Game) drawGhostActor(dst *ebiten.Image, cs *CharSet, x, footY float64, dir Dir, anim int, t, alpha, trail float64, idx int) {
	if cs == nil {
		return
	}
	img, flip := cs.img(anim, dir, 0) // always the same pose: they glide
	if img == nil {
		return
	}
	recLayer("actor:ghost")
	seed := float64(idx) * 1.7
	bob := math.Sin(t*2.2+seed)*2.6 - 6
	left := x - charCX
	top := footY - charBase + bob
	body := alpha * (0.88 + 0.12*math.Sin(t*2.6+seed)) // the slow breath of the alpha
	// the halo
	if g != nil && g.as != nil && g.as.glow != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-48, -48)
		s := 26.0 / 48
		op.GeoM.Scale(s, s*1.3)
		op.GeoM.Translate(x, top+charH*0.45)
		op.Blend = ebiten.BlendLighter
		op.ColorScale.ScaleWithColor(color.RGBA{70, 110, 190, 22})
		dst.DrawImage(g.as.glow, op)
	}
	// the wake: two fainter echoes behind
	if trail != 0 {
		for k := 2; k >= 1; k-- {
			drawGhostImage(dst, img, left+trail*float64(k)*7, top+math.Sin(t*2.2+seed-float64(k)*0.6)*1.5, flip, t-float64(k)*0.25, body*0.20/float64(k), seed)
		}
	}
	drawGhostImage(dst, img, left, top, flip, t, body, seed)
	// hollow eyes: two dark sockets with a cold spark
	ex := 0.0
	switch dir {
	case DirLeft:
		ex = -2
	case DirRight:
		ex = 2
	}
	ey := math.Round(top + 12)
	for _, d := range []float64{-3, 2} {
		sx := math.Round(x + ex + d)
		fillRect(dst, sx, ey, 2, 3, rgba(0x0a1024, uint8(230*body+20)))
		fillRect(dst, sx, ey+1, 1, 1, rgba(0xb8f0ff, uint8(255*body)))
	}
	// the chains of some of them
	if idx%3 == 1 {
		for i := 0; i < 6; i++ {
			fi := float64(i)
			cx := x - 5 + math.Sin(t*5+seed+fi*0.9)*1.5 + fi*0.6*trail
			cy := top + charH*0.62 + fi*3 + math.Abs(math.Sin(t*3+fi))*1.5
			col := rgba(0x9aa8c8, uint8(200*body))
			fillRect(dst, math.Round(cx), math.Round(cy), 2, 2, col)
			fillRect(dst, math.Round(cx)+1, math.Round(cy)+1, 1, 2, rgba(0x5a6888, uint8(160*body)))
		}
	}
}

// ghostWail: the "¡UUUH!" of the ghost whose turn it is (one at a time, a short moment each). It is queued
// to be written on the screen (crisp), not on the little canvas of the street.
func (g *Game) ghostWail(e *endState, x, topY, t float64, idx, n int) {
	if n <= 0 {
		return
	}
	slot := t / 2.4
	if int(slot)%n != idx {
		return
	}
	u := slot - math.Floor(slot)
	if u > 0.55 {
		return
	}
	a := uint8(255 * math.Sin(u/0.55*math.Pi))
	const wail = "¡UUUH!"
	e.sayAt(wail, ghostWailX(x, g.textW(wail, 1), endCW), math.Round(topY-8-u*10), 0xcfe4ff, a)
}

// ghostWailX: left edge of the wail bubble, centred-ish on the ghost but kept inside [2, cw-w-2].
func ghostWailX(x, w, cw float64) float64 {
	return math.Round(math.Max(2, math.Min(x-14, cw-w-2)))
}
