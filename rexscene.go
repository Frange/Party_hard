package main

import (
	"image"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The T-Rex of the cinematics IS the T-Rex of the Jurassic Park level: the "jp_trex" sprite (rexart.go,
// props8.go) with its poses (walk, head down with open jaws, bite, swallow, roar, chew). Nothing in a scene
// paints its own T-Rex any more: the cake after the credits, the head in the neighbour's window and the
// sheep that is a T-Rex all come from drawRexArt. rexscene_test.go keeps it that way.
// ---------------------------------------------------------------------------

// rexArtOX, rexArtOY: b.ox, b.oy of the jp_trex sprite (room on the left and above for the bent head).
const rexArtOX, rexArtOY = 22, 6

// propRec, when not nil, collects "name:variant" of every prop a cinematic draws (tests only).
var propRec *[]string

func recProp(name string, v int) {
	if propRec != nil {
		*propRec = append(*propRec, name+":"+strconv.Itoa(v))
	}
}

// rexAnchorOff: a point of the rex art (rexart.go coordinates) as an offset from the sprite's anchor (the point
// preProp puts at x, y), at scale 1 and looking left.
func rexAnchorOff(ax, ay float64) Vec {
	w, h := 160.0, 92.0
	if k := propKinds["jp_trex"]; k != nil {
		w, h = float64(k.W), float64(k.H)
	}
	return Vec{ax + rexArtOX - math.Round((w/2+1)*wk), ay + rexArtOY - math.Round((h+1)*wk)}
}

// rexPoseOf: the head pose of a variant (the front and back views use the walking head).
func rexPoseOf(v int) rexHeadPose {
	ps := rexHeadPoses[((v%rexPoses)+rexPoses)%rexPoses]
	if ps == (rexHeadPose{}) {
		ps = rexHeadPoses[rexPoseWalkA]
	}
	return ps
}

// rexSkull: the centre of the skull of pose v, from the anchor (scale 1, looking left).
func rexSkull(v int) Vec {
	ps := rexPoseOf(v)
	return rexAnchorOff(ps.hx, ps.hy)
}

// rexHatSpot: where the party hat sits on the head of pose v (the top of the skull, a little to the back),
// from the anchor. For the walking pose it is the spot partyRex and preRex use.
// The height comes from the real pixels of the sprite: the brim rests on the highest pixel of the skull under it
// (its bottom row over the outline), so the hat touches the head in every pose, with the head up, level or down.
func rexHatSpot(v int) Vec {
	v = ((v % rexPoses) + rexPoses) % rexPoses
	if rexHatDone[v] {
		return rexHatCache[v]
	}
	ps := rexPoseOf(v)
	s := rexSkull(v)
	ext := math.Hypot(17*math.Sin(ps.down), 12*math.Cos(ps.down)) // the rotated skull, half its height
	h := Vec{math.Round(s.X - 9*math.Cos(ps.down)), s.Y - ext}
	if b, ox, oy := rexSpritePix(v); b != nil {
		hx := int(h.X + ox)
		top := -1
		for x := hx - 4; x <= hx+3; x++ { // the whole brim (preHat: x-4 .. x+3)
			for y := 0; y < b.h; y++ {
				if x >= 0 && x < b.w && b.p[(y*b.w+x)*4+3] >= 128 {
					if top < 0 || y < top {
						top = y
					}
					break
				}
			}
		}
		if top >= 0 {
			h.Y = float64(top) - oy // the bottom row of the brim covers the outline of the skull
		}
	}
	rexHatCache[v], rexHatDone[v] = h, true
	return h
}

var (
	rexHatCache [rexPoses]Vec
	rexHatDone  [rexPoses]bool
	rexPixCache [rexPoses]*pixBuf
	rexBoxCache [rexPoses][4]int // x0, y0, x1, y1 of the opaque pixels of each pose
)

// rexSpritePix: the pixels of pose v of the jp_trex sprite exactly as buildPropImages makes them, and the anchor
// inside them (the point preProp puts at x, y).
func rexSpritePix(v int) (b *pixBuf, ox, oy float64) {
	k := propKinds["jp_trex"]
	if k == nil || k.Draw == nil {
		return nil, 0, 0
	}
	v = ((v % rexPoses) + rexPoses) % rexPoses
	if rexPixCache[v] == nil {
		rexPixCache[v] = stylizeSprite(renderPropSprite("jp_trex", k, v))
	}
	return rexPixCache[v], math.Round((float64(k.W)/2 + 1) * wk), math.Round((float64(k.H) + 1) * wk)
}

// sceneRex draws the level's T-Rex in pose v with its anchor at (x, y) (scale 1, integer pixels), mirrored when
// flip (then it looks right), with the party hat if hat. It returns the centre of its skull.
func (g *Game) sceneRex(c *ebiten.Image, v int, x, y float64, flip, hat bool) (head Vec) {
	x, y = math.Round(x), math.Round(y)
	if im := rexScenePropImg(v); im != nil && g != nil && g.as != nil {
		recProp("jp_trex", v)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-im.ox, -im.oy)
		if flip {
			op.GeoM.Scale(-1, 1)
		}
		op.GeoM.Translate(x, y)
		c.DrawImage(im.img, op)
	} else {
		g.preProp(c, "jp_trex", v, x, y, 1, flip, 1)
	}
	sx := 1.0
	if flip {
		sx = -1
	}
	s := rexSkull(v)
	head = Vec{x + sx*s.X, y + s.Y}
	if hat {
		h := rexHatSpot(v)
		preHat(c, x+sx*h.X, y+h.Y, 0xf0c030)
	}
	return head
}

var rexSceneImgs [rexPoses]*PropImg

// rexSceneOwn: the poses with a scrap in the mouth (rexart.go). In the cake scene the scrap is painted in the tones
// of the mouth, so it does not look like a red blob on the cake; the T-Rex of the Jurassic level keeps its own.
func rexSceneOwn(v int) bool { return v == 2 || v == 3 || v == rexPoseChew }

// rexScenePropImg: the scene's own picture of pose v (same sprite, discreet scrap), or nil for the other poses.
func rexScenePropImg(v int) *PropImg {
	if !rexSceneOwn(v) {
		return nil
	}
	if rexSceneImgs[v] == nil {
		k := propKinds["jp_trex"]
		if k == nil || k.Draw == nil {
			return nil
		}
		sc, sb := rxScrap, rxScrapBit
		rxScrap, rxScrapBit = rxMouth, shade(rxMouth, 1.25)
		b := stylizeSprite(renderPropSprite("jp_trex", k, v))
		rxScrap, rxScrapBit = sc, sb
		rexSceneImgs[v] = &PropImg{img: b.img(), ox: math.Round((float64(k.W)/2 + 1) * wk), oy: math.Round((float64(k.H) + 1) * wk)}
	}
	return rexSceneImgs[v]
}

// ---- the T-Rex that eats the cake (the scene after the credits, drawStinger) -----------------------------

// Where the cake's top tier is, on the 1x canvas (the scene is drawn at 2x): the rex bites it there.
const stingCakeX, stingCakeY = 200.0, 165.0

// stingRexPose: the pose, the anchor (1x canvas) and the facing of the T-Rex of the stinger at time t. It walks in
// from the right, puts its head down with the jaws open, bites the cake, swallows and chews, burps and leaves.
// The timing is the scene's own: arrives 3.2-5.0, eats 5.0-5.9 (the cake is gone at 5.9), burps 6.9, leaves 7.4.
func stingRexPose(t float64) (v int, x, y float64, flip, ok bool) {
	if t <= 3.2 {
		return 0, 0, 0, false, false
	}
	half := -rexAnchorOff(-rexArtOX, 0).X // the sprite reaches this far on each side of its anchor
	// with the head down (pose 1) the open mouth is ~46 px left of the anchor and ~20 px above it: it lands on
	// the top tier of the cake, and the feet stay on the grass behind the table
	stopX := stingCakeX + 50
	y = stingCakeY + 25
	u := clampF((t-3.2)/1.8, 0, 1)
	startX := float64(endCW) + half + 4 // fully out of the canvas on the right
	x = startX - (startX-stopX)*easeOut(u)
	walk := []int{rexPoseWalkA, rexPoseWalkB}[int(t*6)&1]
	switch {
	case t < 5.0:
		v = walk
		y -= math.Round(math.Abs(math.Sin(t * 9))) // the steps
	case t < 5.3:
		v = 1 // head down, jaws open
	case t < 5.9:
		v = []int{2, 1}[int((t-5.3)*7)&1] // ÑAM ÑAM: bite, open, bite
	case t < 6.9:
		v = []int{3, rexPoseChew}[int((t-5.9)*5)&1]
	case t < 7.4:
		v = 4 // BUUURP
	default:
		out := clampF((t-7.4)/1.0, 0, 1)
		v, flip = walk, true
		x = stopX + (float64(endCW)+half+4-stopX)*out*out // turns round and walks off to the right
		y -= math.Round(math.Abs(math.Sin(t * 9)))
	}
	return v, math.Round(x), math.Round(y), flip, true
}

// drawStingRex draws the T-Rex of the stinger on the scene's canvas (1x) and puts it on dst at 2x.
func (g *Game) drawStingRex(dst *ebiten.Image, e *endState) {
	v, x, y, flip, ok := stingRexPose(e.t)
	if !ok {
		return
	}
	if e.canvas == nil {
		e.canvas = ebiten.NewImage(endCW, endCH)
	}
	e.canvas.Clear()
	recLayer("actor:rex")
	g.sceneRex(e.canvas, v, x, y, flip, true)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	dst.DrawImage(e.canvas, op)
}

// rexSpriteRect: the rectangle of the real pixels of the rex in pose v drawn by sceneRex at (x, y) (canvas pixels,
// scale 1), with the party hat when hat; the soft shadow under the feet does not count.
func rexSpriteRect(v int, x, y float64, flip, hat bool) (r Rect, ok bool) {
	b, ox, oy := rexSpritePix(v)
	if b == nil {
		return Rect{}, false
	}
	v = ((v % rexPoses) + rexPoses) % rexPoses
	if rexBoxCache[v] == [4]int{} {
		x0, y0, x1, y1 := b.w, b.h, -1, -1
		for py := 0; py < b.h; py++ {
			for px := 0; px < b.w; px++ {
				if b.p[(py*b.w+px)*4+3] >= 128 {
					x0, y0 = min(x0, px), min(y0, py)
					x1, y1 = max(x1, px+1), max(y1, py+1)
				}
			}
		}
		if x1 < 0 {
			return Rect{}, false
		}
		rexBoxCache[v] = [4]int{x0, y0, x1, y1}
	}
	x0, y0, x1, y1 := rexBoxCache[v][0], rexBoxCache[v][1], rexBoxCache[v][2], rexBoxCache[v][3]
	l, rt := x+float64(x0)-ox, x+float64(x1)-ox
	if flip {
		l, rt = x-(float64(x1)-ox), x-(float64(x0)-ox)
	}
	top, bot := y+float64(y0)-oy, y+float64(y1)-oy
	if hat {
		h := rexHatSpot(v)
		hx := x + h.X
		if flip {
			hx = x - h.X
		}
		hx, hy := math.Round(hx), math.Round(y+h.Y)
		l, rt = math.Min(l, hx-4), math.Max(rt, hx+4)
		top, bot = math.Min(top, hy-10), math.Max(bot, hy+1)
	}
	return Rect{l, top, rt - l, bot - top}, true
}

// stingBang: one sound word of the cake scene: shown from t0 to t1, at its biggest scale, and the pose whose
// mouth it comes out of.
type stingBang struct {
	s      string
	t0, t1 float64
	col    uint32
	size   float64
	mouthV int
}

var stingBangList = []stingBang{
	{"ÑAM ÑAM", 5.2, 6.5, 0xff8ad0, 3, 2},
	{"BUUURP", 6.9, 8.0, 0x8ae070, 4, 4},
}

// stingBangGap: the air between a sound word and the rex, its hat or the cake (screen pixels).
const stingBangGap = 16

// stingBangSpot: where sceneBang writes word i (screen pixels: the centre x and the top y at u = 0). The word goes
// to the left of the rex, of its hat and of the cake during all its time on screen (the rex walks off to the
// right), at the height of the mouth of its pose, and inside the screen margins at its biggest size.
func (g *Game) stingBangSpot(i int) (cx, y float64) {
	b := stingBangList[i]
	left := math.Inf(1)
	for t := b.t0; t <= b.t1+1e-9; t += 0.02 {
		v, x, ry, flip, ok := stingRexPose(t)
		if !ok {
			continue
		}
		if r, ok := rexSpriteRect(v, x, ry, flip, true); ok {
			left = math.Min(left, 2*r.X)
		}
		if t < 5.9 { // the cake (drawStinger)
			left = math.Min(left, 350)
		}
	}
	sc := pxScale(b.size * 1.5)
	w := g.textW(b.s, sc) + pxOff(sc)
	_, h := textMeasure(g, tr(b.s))
	h = h*sc + pxOff(sc)
	cx = math.Max(16+w/2, math.Round(left-stingBangGap-w/2))
	_, _, my, _, _ := stingRexPose(b.t0)
	s := rexSkull(b.mouthV)
	mouth := 2 * (my + s.Y + 10) // the jaws, under the centre of the skull
	y = math.Round(math.Max(16+18, mouth-h/2))
	return cx, y
}

// drawStingBangs: ÑAM ÑAM while it eats, BUUURP when it burps.
func (g *Game) drawStingBangs(dst *ebiten.Image, t float64) {
	for i, b := range stingBangList {
		if t > b.t0 && t < b.t1 {
			cx, y := g.stingBangSpot(i)
			g.sceneBang(dst, b.s, cx, y, (t-b.t0)/(b.t1-b.t0), b.col, b.size)
		}
	}
}

// ---- small appearances ---------------------------------------------------------------------------------

// windowRex: the head of the T-Rex looking into the neighbour's window (wx, wy, ww, wh on the street canvas);
// k 0..1 is how far it has come in from the right.
func (g *Game) windowRex(c *ebiten.Image, wx, wy, ww, wh, k float64) {
	r := image.Rect(int(math.Round(wx)), int(math.Round(wy)), int(math.Round(wx+ww)), int(math.Round(wy+wh)))
	win, ok := c.SubImage(r).(*ebiten.Image)
	if !ok {
		return
	}
	s := rexSkull(rexPoseWalkA)
	hx, hy := wx+ww/2+8+30*(1-k), wy+wh/2-3 // where the skull goes: eye, snout and teeth in the window
	g.sceneRex(win, rexPoseWalkA, hx-s.X, hy-s.Y, false, false)
}

var rexMiniImg *ebiten.Image

// rexMiniHead: the head of the level's T-Rex at half size, looking left (24x18), for the dinosaur sheep. The
// sprite is painted by drawRexArt and shrunk once; the crop keeps only the head.
func rexMiniHead() *ebiten.Image {
	if rexMiniImg == nil {
		k := propKinds["jp_trex"]
		if k == nil || k.Draw == nil {
			return nil
		}
		b := newPix(k.W+2, k.H+2)
		k.Draw(b, rexPoseWalkA)
		full := stylizeSprite(b.scaled(0.5).outlinedSel(0.45)).img()
		x0, y0 := (rexArtOX-2)/2, (rexArtOY+4)/2 // art x -2..46, y 4..40
		rexMiniImg = full.SubImage(image.Rect(x0, y0, x0+24, y0+18)).(*ebiten.Image)
	}
	return rexMiniImg
}

// miniRexSkull: the centre of the skull inside rexMiniHead (art 26, 18).
func miniRexSkull() Vec { return Vec{(26 + 2) / 2, (18 - 4) / 2} }
