package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The sunset over the sea (levels with Sky "sea": MIAMI BEACH, PLAYA and, above its open water, the
// CRUCERO NOCTURNO). The sky, the sun and the horizon are baked into the level background (draw.go
// renderBG); the two things that move are painted here every frame, right after the background:
//   - the orange reflection of the sun on the water: each line slides a few whole pixels left and
//     right, slowly and each with its own phase, as if the swell rocked it;
//   - the little ship on the horizon: it drifts from left to right, bobbing one pixel, and comes
//     back in from the left once it has left the screen on the right (never a visible jump);
//   - now and then (not on the cruise) a pirate ship sails by, right to left, below the horizon.
// With ambientOn=false (pixel-exact tests) both stay baked into the background as before.

const (
	seaSunX     = 780.0 // centre of the sun (draw.go)
	seaReflTop  = 46.0  // first reflection line
	seaReflN    = 20    // number of lines
	seaShipW    = 40.0  // the hull of the ship on the horizon
	seaShipY    = 28.0
	seaShipX0   = 420.0 // where it was painted when it stood still
	seaShipLoop = screenW + 2*seaShipW + 40
)

var (
	seaReflCol = rgb(0xe8884a)
	seaShipCol = rgb(0x14101e)
)

// seaReflLine: the place of reflection line i (as painted when still).
func seaReflLine(i int) (x, y, w float64) {
	return 700 + float64(i%5)*10, seaReflTop + math.Round(float64(i)*2.4), 90 - float64(i)*3
}

// seaReflShift: the swell's sideways shift of line i at time t, in whole pixels (-4..4).
// Slow (periods of 6-10 s), a different phase per line, and lines further down move a bit more.
func seaReflShift(t float64, i int) float64 {
	amp := 2 + float64(i)*0.12
	sp := 0.62 + 0.25*bgH(i, 71)
	return math.Round(amp * math.Sin(t*sp+float64(i)*0.85))
}

// seaShipPos: the top-left corner of the ship's hull at time t (whole pixels). It drifts at 5 px/s.
func seaShipPos(t float64) (x, y float64) {
	x = math.Mod(seaShipX0+seaShipW+20+t*5, seaShipLoop) - seaShipW - 20
	y = seaShipY + math.Round(0.5+0.5*math.Sin(t*0.9))
	return math.Round(x), y
}

// seaShipImgs: the horizon ship (vessels.go genHorizonShip) as painted into a baked background [0] and
// as painted live over a graded one [1].
var seaShipImgs [2]*ebiten.Image

func seaShipImg(graded bool) *ebiten.Image {
	i, col := 0, seaShipCol
	if graded {
		i, col = 1, seaGraded(seaShipCol)
	}
	if seaShipImgs[i] == nil {
		seaShipImgs[i] = genHorizonShip(col).img()
	}
	return seaShipImgs[i]
}

// drawSeaShip paints the horizon ship with its hull's top-left corner at x, y (whole pixels).
func drawSeaShip(dst *ebiten.Image, x, y float64, graded bool) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y-8)
	dst.DrawImage(seaShipImg(graded), op)
}

// The pirate ship (vessels.go genPirateShip) sails now and then across the sea of the beach levels,
// from right to left, below the horizon: one crossing every seaPirateEvery seconds.
const (
	seaPirateEvery = 300.0
	seaPirateFrom  = 8.0  // the first one starts entering 8 s into the party
	seaPirateSpeed = 9.0  // px/s
	seaPirateWL    = 66.0 // its waterline on the screen
	seaPirateRow   = 49.0 // the waterline row of the sprite
)

var seaPirateImgs [2]*ebiten.Image

// seaPiratePos: the top-left corner of the pirate ship at time t (whole pixels), and whether it is at sea.
func seaPiratePos(t float64) (x, y float64, ok bool) {
	u := math.Mod(t-seaPirateFrom, seaPirateEvery)
	if t < seaPirateFrom || u*seaPirateSpeed > screenW+80 {
		return 0, 0, false
	}
	x = math.Round(screenW + 4 - u*seaPirateSpeed)
	y = seaPirateWL - seaPirateRow + math.Round(0.5+0.5*math.Sin(t*1.3))
	return x, y, true
}

// drawSeaPirate paints the pirate ship (bow to the left, the way it sails) with its pennants waving.
func drawSeaPirate(dst *ebiten.Image, x, y, t float64) {
	f := int(t*3) % 2
	if seaPirateImgs[f] == nil {
		seaPirateImgs[f] = genPirateShip(f).outlinedSel(0.42).img()
	}
	im := seaPirateImgs[f]
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(-1, 1)
	op.GeoM.Translate(x+float64(im.Bounds().Dx()), y)
	dst.DrawImage(im, op)
}

// seaSkyStill: set while a background is rendered as a still picture (the flashbacks, the editor's
// export), so the reflection and the ship are baked into it.
var seaSkyStill bool

// seaSkyMoving: does this level paint its reflection and ship every frame (instead of baking them)?
func seaSkyMoving(def *LevelDef) bool {
	return ambientOn && !seaSkyStill && def != nil && def.Sky == "sea" && def.BGImg == nil && def.PaintBG == nil
}

// bakeSeaSky paints the still reflection and ship into the background (draw.go renderBG), only
// when they do not move.
func bakeSeaSky(bg *ebiten.Image, def *LevelDef) {
	if seaSkyMoving(def) {
		return
	}
	for i := 0; i < seaReflN; i++ {
		x, y, w := seaReflLine(i)
		fillRect(bg, x, y, w, 1, seaReflCol)
	}
	drawSeaShip(bg, seaShipX0, seaShipY, false)
}

// seaGraded: a colour as gradeBG leaves it (the same contrast curve, without the grain), so what is
// painted every frame matches the baked background.
func seaGraded(c color.RGBA) color.RGBA {
	g := func(v uint8) uint8 {
		f := (float64(v)/255-0.5)*1.14 + 0.5
		return uint8(clampF(f*255-6, 0, 255))
	}
	return color.RGBA{g(c.R), g(c.G), g(c.B), 255}
}

// seaWaterBottom: the first row covered by the ground under the horizon (the sand), where the
// reflection must stop because it used to be painted underneath it.
func seaWaterBottom(def *LevelDef) float64 {
	bot := 94.0
	for _, f := range def.Floors {
		if f.R.Y >= 40 && f.R.Y < bot && f.R.X < seaSunX+60 && f.R.X+f.R.W > seaSunX-90 {
			bot = f.R.Y
		}
	}
	return bot
}

// drawSeaSky paints the moving reflection and ship (called from drawBackAnim, behind everything).
func (g *Game) drawSeaSky(dst *ebiten.Image, t float64) {
	def := g.lvl.Def
	if !seaSkyMoving(def) {
		return
	}
	ox, oy := g.camI.X, g.camI.Y
	if def.Shape == nil { // on the cruise the open water covers the reflection (ships.go paintSea)
		bot := seaWaterBottom(def)
		rc := seaGraded(seaReflCol)
		for i := 0; i < seaReflN; i++ {
			x, y, w := seaReflLine(i)
			if y >= bot {
				break
			}
			bgRect(dst, x+seaReflShift(t, i)+ox, y+oy, w, 1, rc)
		}
	}
	x, y := seaShipPos(t)
	drawSeaShip(dst, x+ox, y+oy, true)
	if def.Shape == nil { // the beach levels: now and then a pirate ship
		if px, py, ok := seaPiratePos(t); ok {
			drawSeaPirate(dst, px+ox, py+oy, t)
		}
	}
}
