package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Skeleton bodies: whoever dies electrocuted (the electric trap in a pool or on
// a fuse box, Satan's lightning, Thor's hammer) lights up like an X-ray for a
// moment and then falls as a scorched skeleton, with a wisp of smoke.
// Everything else about the death (points, stats, the body count) is the
// normal one: only the picture changes.
// ---------------------------------------------------------------------------

// elecDeathNext is raised by killGuestElec for the body that spawnBody is about to create.
var elecDeathNext bool

// killGuestElec kills a guest by electricity: the body will be a skeleton.
func (g *Game) killGuestElec(gu *Guest, byPlayer bool) {
	elecDeathNext = true
	g.killGuest(gu, byPlayer)
	elecDeathNext = false
}

// makeSkeletonBody turns a freshly spawned body into a burnt skeleton.
func (g *Game) makeSkeletonBody(b *Body) {
	b.skeleton = true
	b.born = g.time
	g.burst(Vec{b.pos.X, b.pos.Y - 18}, 14, 40, rgb(0x70707a), 1.4, 3, -30)
	g.burst(Vec{b.pos.X, b.pos.Y - 18}, 10, 90, rgb(0xfff0a0), 0.3, 2, 0)
	g.decals = append(g.decals, Decal{pos: b.pos, kind: 1, r: 8})
}

type skelSet struct{ up, ly *ebiten.Image }

var skelCache = map[[2]uint32]*skelSet{}

// skeletonSet returns the standing and the lying skeleton of a character: bones, with a few
// scraps of its own shirt and trousers left on them.
func skeletonSet(cs *CharSet) *skelSet {
	var shirt, pants color.RGBA = rgb(0x4a4a56), rgb(0x2a2a34)
	if cs != nil && cs.Spec != nil {
		shirt, pants = cs.Spec.Shirt, cs.Spec.Pants
	}
	key := [2]uint32{packRGBA(shirt), packRGBA(pants)}
	if s, ok := skelCache[key]; ok {
		return s
	}
	b := genSkeleton(shirt, pants)
	s := &skelSet{up: b.outlinedSel(0.4).img(), ly: b.rotCW().outlinedSel(0.4).img()}
	skelCache[key] = s
	return s
}

func packRGBA(c color.RGBA) uint32 { return uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B) }

// genSkeleton paints an upright skeleton in a character frame (feet on the last row of the sprite).
func genSkeleton(shirt, pants color.RGBA) *pixBuf {
	b := newPix(charW, charH)
	bone, shade, dark := rgb(0xdedac6), rgb(0xa8a28a), rgb(0x24201a)
	soot := rgb(0x4a443a)
	cx := charCX
	// skull
	b.ellipse(float64(cx)-0.5, 14, 4.6, 4.4, bone)
	b.rect(cx-3, 17, 6, 2, bone) // jaw
	for x := cx - 3; x <= cx+2; x += 2 {
		b.set(x, 18, dark) // teeth gaps
	}
	b.rect(cx-3, 13, 2, 2, dark) // eye sockets
	b.rect(cx, 13, 2, 2, dark)
	b.set(cx-1, 16, dark) // nose
	b.set(cx-4, 12, shade)
	b.set(cx+3, 15, shade)
	// scorch marks
	b.set(cx+1, 10, soot)
	b.set(cx-2, 11, soot)
	// spine and ribs
	for y := 19; y <= 31; y++ {
		b.set(cx-1, y, bone)
		b.set(cx, y, shade)
	}
	for i, y := range []int{21, 23, 25, 27} {
		w := 4 - i/2
		b.rect(cx-1-w, y, w, 1, bone)
		b.rect(cx+1, y, w, 1, bone)
		b.rect(cx-1-w, y+1, w, 1, dark)
		b.rect(cx+1, y+1, w, 1, dark)
	}
	// scraps of shirt on the shoulders
	b.rect(cx-6, 20, 3, 2, shirt)
	b.rect(cx+3, 20, 3, 2, shirt)
	// arms: upper bone, elbow, forearm, hand
	for i := 0; i < 2; i++ {
		sx, dir := cx-5, -1
		if i == 1 {
			sx, dir = cx+4, 1
		}
		b.rect(sx+dir, 22, 1, 4, bone)
		b.set(sx+dir, 26, shade)
		b.rect(sx+dir*2, 27, 1, 4, bone)
		b.set(sx+dir*2, 31, shade)
	}
	// pelvis with a scrap of trousers
	b.rect(cx-3, 31, 6, 2, bone)
	b.rect(cx-3, 33, 6, 1, pants)
	b.set(cx-1, 32, dark)
	b.set(cx, 32, dark)
	// legs: femur, knee, tibia, foot
	for i := 0; i < 2; i++ {
		lx := cx - 3
		if i == 1 {
			lx = cx + 2
		}
		b.rect(lx, 34, 1, 4, bone)
		b.set(lx, 38, shade)
		b.rect(lx, 39, 1, 3, bone)
		b.rect(lx-1+i*2, 41, 3, 1, shade)
	}
	return b
}

// drawSkeletonBody: the X-ray flash while it falls, then the lying skeleton.
func (g *Game) drawSkeletonBody(dst *ebiten.Image, b *Body) {
	sk := skeletonSet(b.cs)
	if b.fallT > 0 {
		t := 0.45 - b.fallT
		if t < 0.3 {
			// the character and its skeleton take turns for a moment
			if int(t/0.05)%2 == 0 {
				g.drawActor(dst, b.cs, AnimFall, 0, b.pos, b.dir, 0, false)
			} else {
				g.drawSkelStanding(dst, sk.up, b.pos)
			}
			return
		}
	}
	cx, cy := g.wx(b.pos.X), g.wy(b.pos.Y)
	w, h := sk.ly.Bounds().Dx(), sk.ly.Bounds().Dy()
	fillRect(dst, cx-float64(w)/2+3, cy-2, float64(w)-6, 3, color.RGBA{0, 0, 0, 50})
	g.drawImg(dst, sk.ly, cx, cy, float64(w)/2, float64(h)-2)
	// a wisp of smoke for a few seconds
	if age := g.time - b.born; age >= 0 && age < 4 {
		for i := 0; i < 3; i++ {
			ph := math.Mod(g.time*0.9+float64(i)*0.33, 1)
			a := uint8(120 * (1 - ph))
			fillRect(dst, cx-4+float64(i*4)+math.Sin(ph*6+float64(i))*2, cy-6-ph*16, 2, 2, color.RGBA{150, 150, 160, a})
		}
	}
}

// drawSkelStanding draws the upright skeleton where a standing character would be.
func (g *Game) drawSkelStanding(dst *ebiten.Image, im *ebiten.Image, pos Vec) {
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-charCX, -charBase)
	if sc != 1 {
		op.GeoM.Scale(sc, sc)
	}
	op.GeoM.Translate(g.wx(pos.X), g.wy(pos.Y))
	dst.DrawImage(im, op)
}

// debugZap (-zapnow): electrocutes the three guests nearest to the player once the party is running.
var debugZap bool

func (g *Game) debugZapStep() {
	if !debugZap || g.time < 1.5 {
		return
	}
	debugZap = false
	for n := 0; n < 3; n++ {
		if v := g.nearestGuest(g.p.pos); v != nil {
			v.pos = g.lvl.NearestFree(Vec{g.p.pos.X + float64(n-1)*34, g.p.pos.Y - 46})
			g.burst(Vec{v.pos.X, v.pos.Y - 14}, 14, 90, rgb(0xfff0a0), 0.4, 2, 0)
			g.killGuestElec(v, false)
		}
	}
}
