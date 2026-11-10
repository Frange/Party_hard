package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// Two caricature look-alikes of the speedy duo of the party: the blue spiky runner ("sonic") and the
// two-tailed orange pal ("tails"). Their own drawing layers are baked into the cached frames (no cost
// per drawn frame): swept-back quills that flutter, glove and strap details, feet that blur into a red
// wheel when running, ears, a propeller beanie, and two tails that spin like a propeller (they also
// hover a little when dancing). Own proportions (huge head, slim body), own palette and party details
// (sunglasses pushed up on the forehead, propeller beanie): a parody, not a copy.
// ---------------------------------------------------------------------------

const (
	foxNone = iota
	foxBlue
	foxTwin
)

func foxKind(s *CharSpec) int {
	switch s.Name {
	case "sonic":
		return foxBlue
	case "tails":
		return foxTwin
	}
	return foxNone
}

var (
	fxBlue   = hx(0x2f78e6)
	fxBlueD  = hx(0x1c4fb4)
	fxBeige  = hx(0xf2d2a4)
	fxOrange = hx(0xf08a1c)
	fxOrgD   = hx(0xc86a10)
	fxWhite  = hx(0xfff6e6)
	fxRed    = hx(0xd82a2a)
)

func foxMoving(p Pose) bool {
	switch {
	case p.Anim == AnimWalk, p.Anim == AnimRun, p.Anim == AnimGlide, p.Anim == AnimMarch:
		return true
	}
	return p.Anim >= AnimDance0 && p.Anim <= AnimDance11
}

func foxDancing(p Pose) bool { return p.Anim >= AnimDance0 && p.Anim <= AnimDance11 }

// foxHead recolours and decorates the head: quills, muzzle, eyes, ears, beanie, sunglasses.
func foxHead(b *pixBuf, kind, vw int, hxp, hyp, rx, ry float64, p Pose) {
	cI := func(f float64) int { return int(math.Floor(f)) }
	ihx, cy, top := cI(hxp), cI(hyp), cI(hyp-ry)
	rxi := cI(rx)
	inHead := func(x, y int) bool {
		dx, dy := (float64(x)+0.5-hxp)/rx, (float64(y)+0.5-hyp)/ry
		return dx*dx+dy*dy <= 1.08 && b.get(x, y).A != 0
	}
	fl := 0
	if foxMoving(p) {
		fl = p.F % 2
	}
	run := 0
	if p.Anim == AnimRun {
		run = 1
	}
	body, dark := fxBlue, fxBlueD
	if kind == foxTwin {
		body, dark = fxOrange, fxOrgD
	}
	q := func(x0, y0, x1, y1 int) { // a quill two pixels thick
		b.line(x0, y0, x1, y1, body)
		b.line(x0, y0+1, x1, y1+1, body)
	}
	// fur colour on the upper head (and all of it from behind)
	for y := top - 1; y <= cy+cI(ry)+1; y++ {
		for x := ihx - rxi - 2; x <= ihx+rxi+2; x++ {
			if inHead(x, y) && (y <= cy+1 || vw == vBack) {
				b.set(x, y, body)
			}
		}
	}
	// muzzle
	mz := fxBeige
	if kind == foxTwin {
		mz = fxWhite
	}
	switch vw {
	case vFront:
		for y := cy - 1; y <= cy+cI(ry); y++ {
			for x := ihx - 2; x <= ihx+2; x++ {
				if inHead(x, y) {
					b.set(x, y, mz)
				}
			}
		}
		b.rect(ihx-2, cy-1, 2, 3, hx(0xffffff)) // big eyes
		b.rect(ihx+1, cy-1, 2, 3, hx(0xffffff))
		pup := hx(0x2aa84a)
		if kind == foxTwin {
			pup = hx(0x2a6ae0)
		}
		b.rect(ihx-1, cy, 1, 2, pup)
		b.rect(ihx+1, cy, 1, 2, pup)
		b.set(ihx-1, cy+3, hx(0x14141a)) // nose and smile
		b.set(ihx, cy+3, hx(0x14141a))
	case vSide:
		for y := cy; y <= cy+cI(ry); y++ {
			for x := ihx + 1; x <= ihx+rxi+1; x++ {
				if inHead(x, y) {
					b.set(x, y, mz)
				}
			}
		}
		b.rect(ihx+1, cy-1, 2, 3, hx(0xffffff))
		b.set(ihx+2, cy, hx(0x14141a))
		b.set(ihx+rxi+1, cy+1, hx(0x14141a))
	}
	if kind == foxBlue {
		// quills swept back, they flutter while moving
		l := 3 + run + fl
		switch vw {
		case vFront:
			for k := 0; k < 3; k++ {
				y := top + 1 + k*2
				q(ihx-rxi-1, y, ihx-rxi-1-l+k, y-2+k+fl)
				q(ihx+rxi, y, ihx+rxi+l-k, y-2+k+fl)
				b.set(ihx-rxi-1-l+k, y-2+k+fl, dark)
				b.set(ihx+rxi+l-k, y-2+k+fl, dark)
			}
			for k := -1; k <= 1; k++ {
				q(ihx+k*2, top+1, ihx+k*3, top-3-fl+k*k)
			}
		case vBack:
			for k := -2; k <= 2; k++ {
				q(ihx+k, top+1, ihx+k*3, top-3-fl+(k*k)/2)
				b.line(ihx+k, top+4, ihx+k*3, top+5+fl-(k*k)/3, dark)
			}
		default:
			for k := 0; k < 3; k++ {
				q(ihx-rxi+1, top+1+k*2, ihx-rxi-l-1+k, top+k*2+fl+1)
				b.set(ihx-rxi-l-1+k, top+k*2+fl+1, dark)
			}
			b.line(ihx-1, top+1, ihx-3, top-3-fl, body)
		}
		// party detail: yellow sunglasses pushed up on the forehead
		if vw == vFront {
			b.rect(ihx-3, top+1, 3, 1, hx(0xf0d020))
			b.rect(ihx+1, top+1, 3, 1, hx(0xf0d020))
			b.set(ihx, top+1, hx(0x14141a))
		} else if vw == vSide {
			b.rect(ihx, top+1, 3, 1, hx(0xf0d020))
		}
		return
	}
	// two-tailed pal: pointed ears, white cheeks tuft, propeller beanie
	switch vw {
	case vSide:
		b.line(ihx-1, top+1, ihx-3, top-2, body)
		b.set(ihx-2, top-1, fxWhite)
	default:
		b.line(ihx-rxi, top+1, ihx-rxi-1, top-3, body)
		b.line(ihx+rxi, top+1, ihx+rxi+1, top-3, body)
		b.set(ihx-rxi, top-1, fxWhite)
		b.set(ihx+rxi, top-1, fxWhite)
	}
	if vw == vFront {
		b.set(ihx-3, cy+2, fxWhite)
		b.set(ihx+3, cy+2, fxWhite)
	}
	bean := hx(0xe0c020)
	b.rect(ihx-rxi+1, top-1, 2*rxi-1, 3, bean)
	b.rect(ihx-rxi+1, top+1, 2*rxi-1, 1, hx(0xd82a2a))
	b.rect(ihx, top-3, 1, 2, hx(0x6a6a74))
	if p.F%2 == 0 || !foxMoving(p) {
		b.rect(ihx-3, top-3, 7, 1, hx(0xd0d4e0)) // propeller wide
	} else {
		b.rect(ihx-1, top-3, 3, 1, hx(0xd0d4e0)) // propeller edge-on
	}
}

// foxTails draws the two tails behind the body; they spin when he moves (propeller) and hang otherwise.
func foxTails(b *pixBuf, vw int, c0, hipY float64, p Pose) {
	n := float64(animFrameCount(p.Anim))
	spin := foxMoving(p)
	for i := 0; i < 2; i++ {
		var tx, ty float64
		if spin {
			a := 2*math.Pi*float64(p.F)/n*1.0 + float64(i)*math.Pi
			if p.Anim == AnimRun {
				a *= 2
			}
			tx, ty = math.Cos(a)*5, math.Sin(a)*3.2
			if vw == vSide {
				tx = -3 - math.Abs(math.Cos(a))*3
			}
		} else {
			sw := float64(i)*2 - 1
			tx, ty = sw*4.5, 2.5
			if vw == vSide {
				tx = -3.5 - float64(i)*1.5
				ty = 1 + float64(i)*1.5
			}
		}
		tx *= 1.3
		x0, y0 := c0, hipY-3
		x1, y1 := x0+tx, y0+2+ty
		thickLine(b, x0, y0+1, x1, y1+1, 3, fxOrgD)
		thickLine(b, x0, y0, x1, y1, 3, fxOrange)
		b.ellipse(x1, y1, 2, 1.8, fxWhite)
	}
}

// foxBody: belly patch, gloves handled in the arm, strap and spinning feet.
func foxBody(b *pixBuf, kind, vw, x0, x1, y0, y1, mid int) {
	if vw == vBack {
		return
	}
	col := fxBeige
	if kind == foxTwin {
		col = fxWhite
	}
	if vw == vFront {
		b.rect(mid-1, y0+2, 3, y1-y0-3, col)
	} else {
		b.rect(x1-2, y0+2, 2, y1-y0-3, col)
	}
}

// foxFoot paints the white strap / band on a shoe and, running, the blurred red wheel of the runner.
func foxFoot(b *pixBuf, kind int, vw, sxx, syy, legW int, p Pose) {
	b.rect(sxx, syy, legW, 1, hx(0xf4f4f4))
	if kind == foxBlue && p.Anim == AnimRun {
		if p.F%2 == 0 {
			b.rect(sxx-1, syy-1, legW+2, 3, fxRed)
			b.set(sxx, syy, hx(0xf4f4f4))
		} else {
			b.rect(sxx-1, syy, legW+2, 2, fxRed)
			b.set(sxx+legW-1, syy-1, hx(0xf4f4f4))
		}
	}
}

var _ color.RGBA
