package main

import "github.com/hajimehoshi/ebiten/v2"

// ---------------------------------------------------------------------------
// THE FAT LOOK of the eating contest (pe_eatcontest.go): a guest who sits at the contest table is drawn with a big belly,
// wide hips and chubby cheeks while the contest lasts. It works for every kind of guest (the procedural ones and the ones
// drawn in the editor, which only have pixels): each frame of his CharSet is widened row by row with nearest-neighbour
// columns (whole pixels, so the sprite stays sharp), more at the belly, a little at the face and the legs. In profile the
// belly sticks out in front. His collision, nav and colours do not change: only the drawing (a copy of the CharSet with
// `fat`, its own frame cache). The contest puts his own CharSet back when it ends or when he dies.
// ---------------------------------------------------------------------------

// fatCharSet: the same character, fat (shares the rig and its drawings; its own frames).
func fatCharSet(base *CharSet) *CharSet {
	if base == nil || base.fat {
		return base
	}
	cs := *base
	cs.cache = map[int]*ebiten.Image{}
	cs.fat = true
	return &cs
}

// fattenFrame widens one frame. Lying and sleeping frames are turned upright, widened and turned back.
func fattenFrame(b *pixBuf, anim, vw int) *pixBuf {
	if b == nil {
		return b
	}
	switch anim {
	case AnimLying:
		return fattenUp(b.rotCCW(), vFront).rotCW()
	case AnimSleep:
		return fattenUp(b.rotCW(), vFront).rotCCW()
	}
	return fattenUp(b, vw)
}

// fatProfile: how many pixels a row grows on the back (left in profile) and on the front, by its height in the body
// (r = 0 the top of the head, 1 the soles).
func fatProfile(r float64, vw int) (l, rt int) {
	e := 0
	switch {
	case r < 0.07:
		e = 0
	case r < 0.22: // cheeks
		e = 1
	case r < 0.30: // neck and shoulders
		e = 2
	case r < 0.40:
		e = 3
	case r < 0.64: // the belly
		e = 5
	case r < 0.74: // hips
		e = 4
	case r < 0.86:
		e = 2
	default:
		e = 1
	}
	if vw != vSide {
		return e, e
	}
	// in profile (the sprite looks right): the belly goes forward, the back only a little
	switch {
	case r >= 0.36 && r < 0.66:
		return (e + 1) / 2, e + 2
	case r >= 0.22:
		return (e + 1) / 2, e
	}
	return e, e
}

// fattenUp widens an upright frame; returns a new buffer of the same size.
func fattenUp(b *pixBuf, vw int) *pixBuf {
	top, bot := -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] != 0 {
				if top < 0 {
					top = y
				}
				bot = y
				break
			}
		}
	}
	out := newPix(b.w, b.h)
	if top < 0 || bot <= top {
		copy(out.p, b.p)
		return out
	}
	h := float64(bot - top)
	for y := 0; y < b.h; y++ {
		x0, x1 := -1, -1
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] != 0 {
				if x0 < 0 {
					x0 = x
				}
				x1 = x
			}
		}
		if x0 < 0 {
			continue
		}
		el, er := fatProfile(float64(y-top)/h, vw)
		n0, n1 := maxI(0, x0-el), minI(b.w-1, x1+er)
		src, dst := x1-x0+1, n1-n0+1
		for nx := n0; nx <= n1; nx++ {
			sx := x0 + (nx-n0)*src/dst
			si, di := (y*b.w+sx)*4, (y*b.w+nx)*4
			copy(out.p[di:di+4], b.p[si:si+4])
		}
	}
	return out
}

// fatRoom: the guest's idle frame (front) is narrow enough at the belly for the fat look to show (somebody holding a wide
// thing already fills the frame).
func fatRoom(cs *CharSet) bool {
	if cs == nil || cs.Spec == nil {
		return false
	}
	var b *pixBuf
	if cs.rig != nil {
		b = cs.rigFrame(AnimIdle, vFront, 0)
	} else {
		b = genChar(cs.Spec, vFront, poseFor(AnimIdle, 0))
	}
	if b == nil {
		return false
	}
	top, bot := -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] != 0 {
				if top < 0 {
					top = y
				}
				bot = y
				break
			}
		}
	}
	if top < 0 {
		return false
	}
	y := top + (bot-top)*52/100
	x0, x1 := -1, -1
	for x := 0; x < b.w; x++ {
		if b.p[(y*b.w+x)*4+3] != 0 {
			if x0 < 0 {
				x0 = x
			}
			x1 = x
		}
	}
	return x0 >= 0 && x1-x0+1 <= 13
}
