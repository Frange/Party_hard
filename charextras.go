package main

import (
	"image/color"
	"math"
)

// Head overlays for a few caricature characters (Fry and the four South Park kids).
// genChar calls charExtras at the end of head(); it paints party details and cheap
// animations (hair and pompom shake, blink, grimace, surprise on a stab) on top of
// the generic head. The designs are parodies, not copies of any reference.

// noCharExtras lets tests render the stock head to compare.
var noCharExtras bool

func extrasKind(s *CharSpec) string {
	switch s.Name {
	case "fry", "stan", "kyle", "cartman", "kenny":
		return s.Name
	}
	return ""
}

func charExtras(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	kind := extrasKind(s)
	if kind == "" {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	rxi := int(math.Floor(rx))
	moving := p.Anim == AnimRun || p.Anim == AnimWalk || p.Anim == AnimHop
	wob := 0
	if moving {
		wob = p.F % 2
	}
	angry := p.Anim == AnimStab
	blink := (p.Anim == AnimIdle && p.F == 1) || (p.Anim == AnimHop && p.F == 3)
	dancing := p.Open
	dark := hx(0x2a1a14)
	pink, yel, wht := hx(0xf0509a), hx(0xf8e040), hx(0xf8f8f8)
	lei := func() {
		y := int(math.Floor(hyp+ry)) + 1
		if vw == vSide {
			for i := -2; i <= 2; i++ {
				b.set(ihx+i, y, []color.RGBA{pink, yel, wht}[(i+2)%3])
			}
			return
		}
		for i := -3; i <= 3; i++ {
			b.set(ihx+i, y, []color.RGBA{pink, yel, wht}[(i+3)%3])
		}
	}
	blinkEyes := func() {
		if vw == vFront && blink {
			b.rect(ihx-2, ey, 2, 2, s.Skin)
			b.rect(ihx+1, ey, 2, 2, s.Skin)
			b.rect(ihx-2, ey+1, 2, 1, dark)
			b.rect(ihx+1, ey+1, 2, 1, dark)
		}
	}
	brows := func() {
		if vw == vFront && angry {
			b.set(ihx-2, ey-1, dark)
			b.set(ihx-1, ey, dark)
			b.set(ihx+2, ey-1, dark)
			b.set(ihx+1, ey, dark)
		}
	}
	switch kind {
	case "fry":
		h := s.Hair
		if vw != vSide {
			// a tall crooked lock that whips about when he moves, plus wild side spikes
			b.rect(ihx+1+wob, top-5, 2, 4, h)
			b.rect(ihx+2+wob, top-6, 1, 2, shade(h, 1.15))
			b.set(ihx-rxi-1-wob, top+1, h)
			b.set(ihx+rxi+1+wob, top+2, h)
		} else {
			b.rect(ihx-1-wob, top-5, 2, 4, h)
			b.rect(ihx-2-wob, top-6, 1, 2, shade(h, 1.15))
			b.rect(ihx-rxi-1-wob, top+2, 2, 3, h)
		}
		sk := shade(s.Skin, 0.86)
		switch vw {
		case vFront:
			blinkEyes()
			// long pointed nose
			b.rect(ihx, ey+2, 2, 3, sk)
			b.set(ihx+1, ey+4, shade(s.Skin, 0.7))
			// party glasses: pink double frame
			b.rect(ihx-3, ey-1, 7, 1, pink)
			b.set(ihx-3, ey, pink)
			b.set(ihx, ey, pink)
			b.set(ihx+3, ey, pink)
			b.set(ihx-3, ey+1, pink)
			b.set(ihx+3, ey+1, pink)
			// crooked grimace, a gasp when stabbing, a wide wobble when dancing
			my := ey + 4
			switch {
			case angry:
				b.rect(ihx-2, my, 2, 2, hx(0x5a1a1a))
			case dancing:
				b.rect(ihx-2, my, 4, 1, hx(0x5a1a1a))
				b.set(ihx-2, my-1, hx(0x5a1a1a))
				b.rect(ihx-1, my+1, 2, 1, wht)
			default:
				b.rect(ihx-2, my, 2, 1, hx(0x7a2a2a))
				b.set(ihx-3, my-1, hx(0x7a2a2a))
			}
		case vSide:
			ex := int(math.Floor(hxp + rx))
			b.rect(ex, ey+1, 3, 2, sk) // the nose juts out
			b.set(ex+3, ey+2, shade(s.Skin, 0.7))
			b.rect(ex-3, ey-1, 3, 1, pink)
			b.set(ex-3, ey, pink)
		}
		lei()
	case "stan":
		b.rect(ihx+wob-1, top-3, 3, 2, hx(0xe03030)) // pompom
		if vw == vFront {
			blinkEyes()
			b.rect(ihx-3, ey, 7, 2, hx(0x14141a)) // shades
			b.set(ihx, ey, hx(0x14141a))
			b.set(ihx-2, ey, hx(0x6a88c8))
			b.set(ihx+1, ey, hx(0x6a88c8))
			brows()
		} else if vw == vSide {
			ex := int(math.Floor(hxp + rx*0.45))
			b.rect(ex-1, ey, 3, 2, hx(0x14141a))
		}
	case "kyle":
		if moving {
			b.rect(ihx-rxi-1, int(math.Floor(hyp))+4, 2, 1, s.HatCol)
			b.rect(ihx+rxi, int(math.Floor(hyp))+4+wob, 2, 1, s.HatCol)
		}
		if vw == vFront {
			blinkEyes()
			brows()
		}
		if vw != vSide {
			b.rect(ihx-2, int(math.Floor(hyp+ry))+1, 5, 1, hx(0x2a9a3a)) // green collar
		}
		lei()
	case "cartman":
		b.rect(ihx+wob-1, top-3, 3, 2, hx(0xf8d838)) // yellow pompom
		if vw == vFront {
			blinkEyes()
			brows()
			// party blower between the lips
			b.rect(ihx+1, ey+3, 3+boolI(dancing)*2, 1, hx(0xf06030))
			b.set(ihx+4+boolI(dancing)*2, ey+3, pink)
		} else if vw == vSide {
			ex := int(math.Floor(hxp + rx))
			b.rect(ex-1, ey+3, 2+boolI(dancing), 1, hx(0xf06030))
		}
	case "kenny":
		if vw == vFront {
			// blue eyes inside the closed hood; round with surprise when stabbing
			b.rect(ihx-2, ey, 2, 2, wht)
			b.rect(ihx+1, ey, 2, 2, wht)
			b.set(ihx-1, ey+1, hx(0x3a78e8))
			b.set(ihx+1, ey+1, hx(0x3a78e8))
			if blink {
				b.rect(ihx-2, ey, 2, 2, s.Mask)
				b.rect(ihx+1, ey, 2, 2, s.Mask)
			}
			if angry {
				b.rect(ihx-2, ey-1, 2, 1, wht)
				b.rect(ihx+1, ey-1, 2, 1, wht)
			}
			// pink heart shades perched on the hood
			b.set(ihx-3, ey-1, pink)
			b.set(ihx-1, ey-1, pink)
			b.set(ihx+1, ey-1, pink)
			b.set(ihx+3, ey-1, pink)
		}
		b.set(ihx+wob, top-1, hx(0xf8a840)) // hood tip flicks about
	}
}
