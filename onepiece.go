package main

import (
	"image/color"
	"math"
)

// ---------------------------------------------------------------------------
// The straw-hat crew, our own parody: big-headed caricatures with a party detail each and a cheap
// animation of their own on the stab frames. Colours only echo the originals (they are not the
// same). Luffy: stretchy arm + flower lei; Sanji: spinning kick + rose and hearts; Zoro: three
// quick sword cuts + shades, three swords and a snore; Nami: clima-tact with a spark + hibiscus;
// Chopper: tiny, bounces and hops + bow tie and antlers. genChar calls opPose / opHead / opBody /
// opHeld (chars.go); everything is drawn into the 24x44 box (pixBuf.set clips) with no allocation.
// ---------------------------------------------------------------------------

func opKind(s *CharSpec) string {
	switch s.Name {
	case "luffy", "sanji", "zoro", "nami", "chopper":
		return s.Name
	}
	return ""
}

// opPose: Chopper hops (small jump and rebound) instead of lunging when he stabs.
func opPose(s *CharSpec, p Pose) Pose {
	if p.Anim == AnimStab && s.Name == "chopper" {
		hp := gaitPoseFor(AnimHop, []int{0, 2, 3}[minI(maxI(p.F, 0), 2)])
		hp.Hold, hp.Anim, hp.F = p.Hold, p.Anim, p.F
		return hp
	}
	return p
}

// opTrail paints a thick line of n pixels from (x,y) along (dx,dy); the lower edge gets a shadow.
func opTrail(b *pixBuf, x, y, dx, dy float64, n int, col, shadow color.RGBA, w int) {
	l := math.Hypot(dx, dy)
	dx, dy = dx/l, dy/l
	for i := 0; i <= n; i++ {
		px, py := int(math.Floor(x+dx*float64(i))), int(math.Floor(y+dy*float64(i)))
		if shadow.A != 0 {
			b.set(px, py+w, shadow)
		}
		for k := 0; k < w; k++ {
			b.set(px, py+k, col)
		}
	}
}

// opHead: party details on the head (called at the end of head() with the head geometry).
func opHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	kind := opKind(s)
	if kind == "" {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	rxi := int(math.Floor(rx))
	ex := int(math.Floor(hxp + rx))
	dark := hx(0x16161e)
	talky := p.Anim == AnimIdle && p.F%2 == 1
	dancing := p.Anim >= AnimDance0 && p.Anim <= AnimDance11
	kicking := p.Anim == AnimStab
	switch kind {
	case "luffy":
		// the straw hat gets a red band, the neck a flower lei
		b.rect(ihx-rxi, top+1, 2*rxi+1, 1, hx(0xd23a2a))
		y := int(math.Floor(hyp+ry)) + 1
		cols := []color.RGBA{hx(0xf0509a), hx(0xf8e040), hx(0xf8f8f8)}
		w := 3
		if vw == vSide {
			w = 2
		}
		for i := -w; i <= w; i++ {
			b.set(ihx+i, y, cols[(i+3)%3])
		}
	case "sanji":
		// a red rose clenched in the smile, floating hearts when he gets excited
		if vw == vFront {
			b.rect(ihx+1, ey+4, 3, 1, hx(0x3a8a3a))
			b.rect(ihx+3, ey+3, 2, 2, hx(0xe02a3a))
		} else if vw == vSide {
			b.rect(ex-1, ey+4, 3, 1, hx(0x3a8a3a))
			b.rect(ex+1, ey+3, 2, 2, hx(0xe02a3a))
		}
		if talky || dancing || kicking {
			x, y := ihx+rxi+2, top-4
			if vw == vSide {
				x = ihx + rxi + 1
			}
			pk := hx(0xf0509a)
			b.set(x+1, y, pk)
			b.set(x+3, y, pk)
			b.rect(x, y+1, 5, 1, pk)
			b.rect(x+1, y+2, 3, 1, pk)
			b.set(x+2, y+3, pk)
		}
	case "zoro":
		// party shades; he snores while he stands
		if vw == vFront {
			b.rect(ihx-3, ey, 7, 2, dark)
			b.set(ihx, ey, dark)
			b.set(ihx-2, ey, hx(0x7aa0d8))
			b.set(ihx+1, ey, hx(0x7aa0d8))
		} else if vw == vSide {
			b.rect(ex-3, ey, 4, 2, dark)
		}
		if p.Anim == AnimIdle && p.F >= 1 {
			x, y := ihx+rxi+2, top-3
			zc := hx(0xe8f0ff)
			b.rect(x, y, 3, 1, zc)
			b.set(x+1, y+1, zc)
			b.rect(x, y+2, 3, 1, zc)
		}
	case "nami":
		// a pink hibiscus tucked in the hair
		x, y := ihx-rxi-1, top+2
		if vw == vSide {
			x = ihx - rxi
		}
		{
			pk := hx(0xff5a8a)
			b.rect(x, y, 2, 2, pk)
			b.set(x-1, y+1, pk)
			b.set(x+1, y-1, pk)
			b.set(x, y+1, hx(0xf8e040))
		}
	case "chopper":
		// pink hat with a white cross and two small antlers
		hc := s.HatCol
		b.rect(ihx-rxi, top-1, 2*rxi+1, 3, hc)
		b.rect(ihx-rxi, top+1, 2*rxi+1, 1, shade(hc, 0.8))
		if vw != vBack {
			wh := hx(0xfdf4f4)
			b.rect(ihx, top-1, 1, 3, wh)
			b.rect(ihx-1, top, 3, 1, wh)
		}
		br := hx(0x7a4a28)
		if vw == vSide {
			b.line(ihx-1, top-1, ihx-2, top-4, br)
			b.set(ihx-3, top-3, br)
			b.set(ihx-1, top-3, br)
		} else {
			b.line(ihx-rxi, top, ihx-rxi-1, top-3, br)
			b.line(ihx+rxi, top, ihx+rxi+1, top-3, br)
			b.set(ihx-rxi-2, top-2, br)
			b.set(ihx+rxi+2, top-2, br)
		}
		// a red bow tie under the chin
		y := int(math.Floor(hyp+ry)) + 1
		if vw != vSide {
			rd := hx(0xe03a3a)
			b.rect(ihx-2, y, 2, 2, rd)
			b.rect(ihx+1, y, 2, 2, rd)
			b.set(ihx, y, hx(0xf8d040))
		} else {
			b.rect(ihx-1, y, 3, 2, hx(0xe03a3a))
		}
	}
}

// opBody: body overlays (after every limb): stretchy arm, spinning kick, swords at the hip.
func opBody(b *pixBuf, s *CharSpec, vw int, p Pose, cx, hipY, shY float64, tw int, sway float64) {
	kind := opKind(s)
	if kind == "" {
		return
	}
	b.cur = 4
	c := int(math.Floor(cx))
	hy := int(math.Floor(hipY))
	sy := float64(int(math.Floor(shY)))
	stab := p.Anim == AnimStab
	switch kind {
	case "luffy":
		if !stab || p.F == 0 {
			return
		}
		n := []int{0, 10, 4}[p.F]
		var sx, dx, dy float64
		switch vw {
		case vSide:
			sx, dx, dy = cx+sway, 1, 0.05
		case vBack:
			sx, dx, dy = cx+float64((tw-1)/2+2)+sway, 0.45, -0.9
		default:
			sx, dx, dy = cx+float64((tw-1)/2+2)+sway, 0.5, 0.9
		}
		sk := s.Skin
		opTrail(b, sx, sy+1, dx, dy, n, sk, shade(sk, 0.72), 2)
		fx, fy := int(math.Floor(sx+dx*float64(n+1))), int(math.Floor(sy+1+dy*float64(n+1)))
		b.rect(fx-1, fy-1, 3, 3, sk)
		b.rect(fx-1, fy+2, 3, 1, shade(sk, 0.7))
		if p.F == 1 { // speed lines
			wh := hx(0xf4f4ff)
			b.rect(fx-5, fy-3, 3, 1, wh)
			b.rect(fx-7, fy+4, 3, 1, wh)
		}
	case "sanji":
		if stab && p.F >= 1 {
			var dx, dy float64
			n := 9
			switch {
			case vw == vSide && p.F == 1:
				dx, dy, n = 1, -0.2, 11
			case vw == vSide:
				dx, dy = 0.8, -0.75
			case vw == vBack:
				dx, dy = 0.7, -0.6
			default:
				dx, dy = 0.75, -0.45
			}
			x0, y0 := cx+1, float64(hy)
			opTrail(b, x0, y0, dx, dy, n, s.Pants, shade(s.Pants, 0.7), 2)
			ex, ey := int(math.Floor(x0+dx*float64(n+1))), int(math.Floor(y0+dy*float64(n+1)))
			b.rect(ex-1, ey-1, 3, 3, hx(0xd8c0a0)) // shoe
			// whirl: a pale arc around the hip, which moves a quarter turn on every frame
			wh := hx(0xe8ecff)
			for i := 0; i < 6; i++ {
				a := float64(p.F)*1.4 + float64(i)*0.45
				b.set(int(math.Floor(cx+math.Cos(a)*8)), hy+int(math.Floor(math.Sin(a)*3.5)), wh)
			}
		}
	case "zoro":
		if stab {
			return // the three cuts are drawn with the held item
		}
		// three swords at the waist: white, gold and dark hilts
		x := c + (tw-1)/2 + 2
		if vw == vSide {
			x = c - 2
		}
		y := hy - 2
		b.rect(x, y, 1, 3, hx(0xe8e8f0))
		b.rect(x+1, y+1, 1, 3, hx(0xe8c040))
		b.rect(x+2, y+2, 1, 3, hx(0x2a2a34))
		b.rect(x-1, y+3, 1, 2, hx(0x9aa0b0))
	case "chopper":
		// a pair of small pink cheeks; nothing else (the hop is his own pose)
		return
	}
}

// opHeld: the weapon of the stab frames. Returns true when it drew the weapon (the knife is not painted).
func opHeld(b *pixBuf, s *CharSpec, kind int, vw int, hx0, hy0 float64, p Pose) bool {
	k := opKind(s)
	if k == "" || kind != HoldKnife {
		return false
	}
	x, y := int(math.Floor(hx0)), int(math.Floor(hy0))
	switch k {
	case "zoro":
		blade, trail := hx(0xdfe6f0), hx(0x9ad8f0)
		// three quick cuts: up-left diagonal, flat, down-left diagonal; each leaves its arc
		dirs := [3][2]float64{{1, -0.8}, {1, 0}, {1, 0.8}}
		d := dirs[minI(maxI(p.F, 0), 2)]
		if vw == vFront {
			d = [2]float64{0.5 * d[0], 0.5*d[1] + 0.9}
		} else if vw == vBack {
			d = [2]float64{0.5 * d[0], 0.5*d[1] - 0.9}
		}
		opTrail(b, float64(x), float64(y), d[0], d[1], 8, blade, color.RGBA{}, 1)
		opTrail(b, float64(x), float64(y-2*int(math.Copysign(1, d[1]+0.001))), d[0], d[1]*0.6, 6, trail, color.RGBA{}, 1)
		b.rect(x-1, y, 2, 1, hx(0xe8c040)) // guard
		return true
	case "nami":
		// clima-tact: a stick with a ball that crackles
		sx, sy := 0.6, -1.0
		if vw == vFront {
			sx, sy = 0.5, 0.5
		}
		opTrail(b, float64(x), float64(y), sx, sy, 7, hx(0x9a6a38), color.RGBA{}, 1)
		bx, by := int(math.Floor(float64(x)+sx/math.Hypot(sx, sy)*8)), int(math.Floor(float64(y)+sy/math.Hypot(sx, sy)*8))
		b.rect(bx-1, by-1, 3, 3, hx(0x5ab8f0))
		b.set(bx, by-1, hx(0xd8f4ff))
		if p.F >= 1 {
			ye := hx(0xfff060)
			b.set(bx-2, by, ye)
			b.set(bx+2, by, ye)
			b.set(bx, by-2, ye)
			b.set(bx, by+2, ye)
			if p.F == 2 {
				b.set(bx-2, by-2, ye)
				b.set(bx+2, by+2, ye)
			}
		}
		return true
	case "luffy", "sanji", "chopper":
		return true // fists, legs and hooves do the fighting
	}
	return false
}
