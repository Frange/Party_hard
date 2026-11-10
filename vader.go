package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Darth Vader: his own drawing layers on top of the generic character (helmet with mask and grille,
// ribbed chest armour with the control panel and the belt lights, knee pads and boots, a wide cape
// with a blue-grey shadow, and a red-orange saber with a glow and a short cut arc when he stabs).
// Everything is baked into the cached frames (genChar), so it costs nothing per drawn frame.
// Animation: the cape trails when he walks, flares when he runs and breathes in idle; two panel
// lights blink (the frame index picks the state). Only the "vader" character gets the body layers;
// the "Casco oscuro" hat of the shop only gets the helmet.
// ---------------------------------------------------------------------------

func isVader(s *CharSpec) bool { return s.Name == "vader" && s.Top == TopArmor }

var (
	vdBlack  = hx(0x0c0c10)
	vdPlate  = hx(0x17181d)
	vdRib    = hx(0x2c303b)
	vdShine  = hx(0x4c5466)
	vdShadow = hx(0x2a3040)
	vdSilver = hx(0xc4c8d2)
)

// vaderLit: is a panel light on in this frame? (idle has 2 frames, the other animations 3-6)
func vaderLit(p Pose, lamp int) bool {
	if p.Anim == AnimIdle {
		return (p.F+lamp)%2 == 0
	}
	return (p.F/(1+lamp%2)+lamp)%3 != 2
}

// vaderTorsoCol colours one torso pixel of the armour.
func vaderTorsoCol(x, y, x0, x1, y0, y1, vw int) color.RGBA {
	col := vdPlate
	if (x*7+y*13)%11 == 0 {
		col = hx(0x22252d)
	}
	if y <= y0+1 && (x <= x0+1 || x >= x1-2) {
		col = vdShine // shoulder bells
	}
	if vw != vBack && y > y0 && y <= y0+4 && (x == x0+1 || x == x1-2) {
		col = vdRib // ribs at both sides of the panel
	}
	if y >= y1-1 {
		col = hx(0x101116)
	}
	return col
}

// vaderChest paints the control panel, the belt and its lights over the torso.
func vaderChest(b *pixBuf, vw, x0, x1, y0, y1, mid int, p Pose) {
	belt := y1 - 2
	b.rect(x0, belt, x1-x0, 1, hx(0x2e3038))
	switch vw {
	case vBack:
		b.rect(mid, y0+2, 1, belt-y0-2, vdRib)
		b.rect(x0+1, y0+3, x1-x0-2, 1, hx(0x20232b))
	case vSide:
		fx := x1 - 2 // the front of the chest
		b.rect(fx-1, y0+3, 2, 3, vdBlack)
		b.set(fx, y0+4, hx(0xe03030))
		if !vaderLit(p, 0) {
			b.set(fx, y0+4, hx(0x70181c))
		}
		b.set(fx-1, y0+5, hx(0xe8e8f0))
		b.set(fx, belt, hx(0x3a78e8))
		b.set(fx-1, belt, vdSilver)
	default:
		for x := x0 + 1; x < x1-1; x++ { // party detail: a flower lei round the neck
			b.set(x, y0+2, []color.RGBA{hx(0xf070b0), hx(0xf0d020), hx(0xff8a30)}[(x-x0)%3])
		}
		b.rect(mid-2, y0+3, 5, 4, vdBlack)
		b.rect(mid-2, y0+3, 5, 1, hx(0x6a6e7a)) // panel frame
		b.set(mid-1, y0+4, hx(0xe070b0))
		b.set(mid, y0+4, hx(0x40d0e8))
		b.set(mid-1, y0+5, hx(0xf0f0f4))
		if vaderLit(p, 1) {
			b.set(mid, y0+5, hx(0xf0f0f4))
		}
		red := hx(0xf02828)
		if !vaderLit(p, 0) {
			red = hx(0x6a1418) // blinking red light
		}
		b.set(mid+1, y0+5, red)
		// belt: coloured lights at both sides and the silver buckle
		b.set(x0, belt, hx(0x3a78e8))
		b.set(x0+1, belt, hx(0x48c868))
		b.set(x1-2, belt, hx(0xe8c838))
		if vaderLit(p, 2) {
			b.set(x1-1, belt, hx(0x3a78e8))
		}
		b.rect(mid-1, belt, 3, 2, vdSilver)
		b.set(mid, belt+1, hx(0x50545e))
	}
}

// vaderHelmet paints the dome, the flared rim, the face plate and the grille.
func vaderHelmet(b *pixBuf, vw int, hxp, hyp, rx, ry float64, hc color.RGBA) {
	cI := func(f float64) int { return int(math.Floor(f)) }
	ihx, cy, top := cI(hxp), cI(hyp), cI(hyp-ry)
	b.ellipse(hxp, hyp, rx+0.9, ry+0.6, hc)
	// flared rim at the bottom, wider than the dome
	by := cI(hyp + ry)
	b.rect(ihx-4, by-1, 9, 1, hc)
	b.rect(ihx-3, by, 7, 1, shade(hc, 0.8))
	// shine of the dome and the ridge
	b.set(ihx-2, top+1, vdShine)
	b.set(ihx-1, top, hx(0x384050))
	b.set(ihx+2, top+2, hx(0x20242e))
	switch vw {
	case vBack:
		b.rect(ihx, top, 1, 4, hx(0x1c1f27))
	case vSide:
		fx := ihx + 3
		b.rect(fx-1, cy-1, 2, 2, hx(0x20222a))
		b.set(fx, cy, hx(0x040406)) // eye lens
		b.set(fx, cy+2, vdSilver)   // grille
		b.set(fx-1, cy+2, vdBlack)
		b.set(fx+1, by-1, hc)
	default:
		b.rect(ihx, top, 1, 3, hx(0x1c1f27)) // ridge
		b.rect(ihx-2, cy-1, 5, 1, hx(0x30343f))
		b.rect(ihx-2, cy, 2, 1, hx(0x040406)) // eye lenses
		b.rect(ihx+1, cy, 2, 1, hx(0x040406))
		b.set(ihx, cy, hx(0x1c1f27))
		b.rect(ihx-1, cy+1, 3, 1, vdBlack) // triangle of the nose
		b.set(ihx-1, cy+2, vdSilver)       // mouth grille
		b.set(ihx+1, cy+2, vdSilver)
		b.set(ihx, cy+2, vdBlack)
	}
}

// vaderCape paints the long cape: it trails when he walks, flares when he runs and breathes in idle.
func vaderCape(b *pixBuf, s *CharSpec, vw int, front bool, x0, x1, y0, y1 int, p Pose) {
	base := s.Cape
	hem := minI(y1+14, charH-3)
	widen := 0
	swing := 0.0
	n := float64(animFrameCount(p.Anim))
	switch p.Anim {
	case AnimRun:
		widen = 2
		swing = 1.6 * math.Sin(2*math.Pi*float64(p.F)/n)
	case AnimWalk, AnimGlide, AnimMarch, AnimLumber:
		widen = 1
		swing = 1.1 * math.Sin(2*math.Pi*float64(p.F)/n)
	case AnimIdle:
		widen = p.F // breathing: the cape opens a little
	default:
		swing = -(p.Sway*0.7 + p.Lean*0.6) // inertia: it lags behind the body
	}
	for y := y0; y <= hem; y++ {
		t := float64(y-y0) / float64(hem-y0)
		grow := (y-y0)*2/5 + widen
		off := int(math.Round(swing * t))
		var xl, xr int
		switch {
		case vw == vSide:
			xl, xr = x0-2-grow-widen-off, x0-1
		case front:
			xl, xr = x0-1-grow+off, x1+grow+off
		default:
			xl, xr = x0-1-grow+off, x1+grow+off
		}
		if xl < 1 {
			xl = 1
		}
		if vw != vSide && xr > charW-2 {
			xr = charW - 2
		}
		yEnd := y
		if y >= hem-1 { // ragged hem
			xl += (y + xl) % 2
		}
		for x := xl; x <= xr; x++ {
			col := shade(base, 1-0.12*t)
			if (x-xl)%4 == 1 && t > 0.2 {
				col = hx(0x15171d) // fold
			}
			if x == xl || x == xr || (t > 0.55 && (x-xl)%5 == 3) {
				col = vdShadow // blue-grey shadow on the edges
			}
			b.set(x, yEnd, col)
		}
	}
}

// vaderSaber paints the red-orange saber with its glow; while he stabs it points forward and leaves a cut arc.
func vaderSaber(b *pixBuf, vw int, hx0, hy0 float64, p Pose) {
	x, y := int(math.Floor(hx0)), int(math.Floor(hy0))
	core, mid, halo := hx(0xffe0b8), hx(0xff5a1c), hx(0x8a1c14)
	if p.Anim == AnimIdle && p.F%2 == 1 {
		halo = hx(0xb02a18) // the glow pulses
	}
	soft := func(px, py int) {
		if b.get(px, py).A == 0 {
			b.set(px, py, halo)
		}
	}
	dx, dy := 0, -1 // blade direction
	if p.Anim == AnimStab && p.F == 1 {
		switch vw {
		case vSide:
			dx, dy = 1, 0
		case vFront:
			dx, dy = 0, 1
		}
	}
	if p.Anim == AnimStab && p.F == 2 && vw == vSide {
		dx, dy = 1, -1
	}
	x = minI(maxI(x, 1), charW-5)
	y = maxI(y, 2)
	blade := 10
	if dx == 0 {
		blade = minI(blade, maxI(2, y-1)) // keeps the tip inside the box
		if dy > 0 {
			blade = minI(blade, charH-3-y)
		}
	} else {
		blade = minI(blade, charW-4-x)
		if dy < 0 {
			blade = minI(blade, maxI(2, y-2))
		}
	}
	if dx == 0 {
		for i := 1; i <= blade; i++ {
			yy := y + dy*i
			soft(x, yy)
			soft(x+3, yy)
			b.set(x+1, yy, mid)
			b.set(x+2, yy, core)
		}
		b.set(x+1, y+dy*(blade+1), halo)
		b.set(x+2, y+dy*(blade+1), mid)
	} else {
		for i := 1; i <= blade; i++ {
			xx, yy := x+1+dx*i, y+1+dy*i/1
			if dy == 0 {
				soft(xx, yy-1)
				soft(xx, yy+2)
				b.set(xx, yy, mid)
				b.set(xx, yy+1, core)
			} else {
				b.set(xx, yy, mid)
				b.set(xx+1, yy, core)
			}
		}
	}
	b.rect(x, y, 3, 2, hx(0x707080)) // hilt
	b.set(x+1, y, vdSilver)
	// short cut arc while attacking (visual only)
	if p.Anim == AnimStab && p.F >= 1 {
		arc := hx(0xffb070)
		if p.F == 2 {
			arc = hx(0xc86030)
		}
		for i := 0; i < 5; i++ {
			a := -0.2 + float64(i)*0.55
			ax := x + 1 + int(math.Round(math.Sin(a)*8))
			ay := y - int(math.Round(math.Cos(a)*8)) + 3
			if ax >= 1 && ay >= 1 && ax <= charW-2 && ay <= charH-2 && b.get(ax, ay).A == 0 {
				b.set(ax, ay, arc)
			}
		}
	}
}

// vaderLeg paints the knee pad and the tall boot of one leg.
func vaderLeg(b *pixBuf, vw int, mx, my float64, sxx, syy, legW int, shoe color.RGBA) {
	kx, ky := int(math.Floor(mx)), int(math.Floor(my))
	if vw == vSide {
		kx--
	}
	b.rect(kx, ky, legW, 2, hx(0x3a3e4a))
	if vw != vBack {
		b.set(kx, ky, vdSilver)
		b.set(kx+legW-1, ky+1, hx(0x9a9eaa))
	}
	b.rect(sxx, syy-2, legW, 2, shoe) // the boot rises over the shin
	if vw == vSide {
		b.rect(sxx-1, syy-2, legW+2, 1, shoe)
	}
}
