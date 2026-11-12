package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// MIRROR BALL (idea 297). The prop "mirrorball" (props_venues.go) existed only in the DISCO SETENTERA. addMirrorBall hangs
// one over the main dance floor of the other indoor dance parties (mirrorBallIcons) that lack it, in the campaign and in
// their random-mode rolls, AFTER the editor corrections (so an edited level gets it too). It is an overlay (overlap.go:
// it hangs over the floor, never blocks a tile: NoSolid), so walking and the flood fill of the level are untouched.
//
// drawDiscoGlints: the specks of light the ball throws on the floor around it, turning slowly (drawn over the night
// layer, like the other lights). None while a blackout cuts the power. The coloured spots of Def.Disco (drawLighting)
// stay as they were: these are only the small white/coloured glints of the mirrors.
// ---------------------------------------------------------------------------

// mirrorBallIcons: the dance parties under a roof (or a party boat) that get a mirror ball if they have none.
var mirrorBallIcons = map[string]bool{"house": true, "neon": true, "disco": true, "casino": true, "cake": true, "boat": true}

const (
	glintN    = 14   // specks per ball
	glintRMin = 20.0 // their ring around the point under the ball
	glintRMax = 74.0
)

// addMirrorBall: one ball over the centre of the biggest dance floor (once per level).
func addMirrorBall(d *LevelDef, icon string) {
	if d == nil || !mirrorBallIcons[icon] || len(d.Dance) == 0 {
		return
	}
	for _, p := range d.Props {
		if p.Kind == "mirrorball" {
			return
		}
	}
	best := d.Dance[0]
	for _, r := range d.Dance[1:] {
		if r.W*r.H > best.W*best.H {
			best = r
		}
	}
	c := best.Center()
	x, y := math.Round(c.X), math.Round(math.Min(c.Y+10, best.Y+best.H-2)) // the ball hangs about the middle of the floor
	d.Props = append(d.Props, PropInst{Kind: "mirrorball", X: x, Y: y, Var: int(x+y) % 3})
}

var glintCols = []color.RGBA{{0xff, 0xff, 0xff, 255}, {0xff, 0xc8, 0xf0, 255}, {0xb0, 0xe8, 0xff, 255}, {0xff, 0xf0, 0xa8, 255}}

// drawDiscoGlints: the glints of every mirror ball of the level on the floor around it.
func (g *Game) drawDiscoGlints(dst *ebiten.Image) {
	if g.lvl == nil || (g.mode != ModePlay && g.mode != ModeLose) || g.blackoutOn() {
		return
	}
	for _, pr := range g.lvl.Props {
		if pr.Name != "mirrorball" || pr.gone {
			continue
		}
		g.drawBallGlints(dst, pr)
	}
}

func (g *Game) drawBallGlints(dst *ebiten.Image, pr *Prop) {
	cx, cy := pr.X, pr.Y-4
	sx, sy := g.wx(cx), g.wy(cy)
	if sx < -glintRMax || sx > screenW+glintRMax || sy < -glintRMax || sy > screenH+glintRMax {
		return
	}
	spin := g.time * 0.55
	for i := 0; i < glintN; i++ {
		// a fixed scatter per speck (golden angle), turning together like the ball
		a := float64(i)*2.39996 + spin
		rr := glintRMin + (glintRMax-glintRMin)*math.Mod(float64(i)*0.618034, 1)
		p := Vec{cx + math.Cos(a)*rr, cy + math.Sin(a)*rr*0.62}
		if g.lvl.SolidP(p.X, p.Y) {
			continue
		}
		tw := 0.55 + 0.45*math.Sin(g.time*3.1+float64(i)*1.7) // twinkle
		if tw < 0.25 {
			continue
		}
		col := glintCols[(i+pr.Var)%len(glintCols)]
		a8 := uint8(200 * tw)
		x, y := math.Round(g.wx(p.X)), math.Round(g.wy(p.Y))
		fillRect(dst, x, y, 2, 2, premul(col, float64(a8)/255))
		if i%3 == 0 { // a few are little crosses
			fillRect(dst, x-1, y, 4, 1, premul(col, float64(a8)/255*0.5))
			fillRect(dst, x+1, y-1, 1, 4, premul(col, float64(a8)/255*0.5))
		}
	}
}
