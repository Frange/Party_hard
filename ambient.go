package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Background animation (ambient.go): subtle, cheap life behind the actors in a
// few levels -- surf and sun glitter on the beach, waves and jumping dolphins
// at sea, snow, fog, drifting leaves/dust, twinkling stars, swaying trees.
//
// Rules: drawn right after the floors (behind props, bodies and actors), low
// contrast, pure decoration (no hitboxes, no sort order, nothing in Update).
// Everything is a closed-form function of g.time and of a per-level table that
// is built once, so it is deterministic and allocates nothing per frame.
// ambientOn=false turns all of it off (pixel-exact tests).
// ---------------------------------------------------------------------------

// ambientOn: the background animation (PH_AMBIENT=0 turns it off, for comparing frame times). Tests that compare pixels switch it off.
var ambientOn = os.Getenv("PH_AMBIENT") != "0"

type bgFx uint16

const (
	bgShore   bgFx = 1 << iota // foam lines that creep up the beach and slide back
	bgGlitter                  // sun sparkles on the sea
	bgCrests                   // little light wave crests on open water
	bgJump                     // a dolphin leaps now and then (every 20-40 s)
	bgSway                     // trees and bushes lean in the wind (1 px)
	bgSnow                     // soft falling flakes
	bgFog                      // low fog drifting over the ground
	bgDust                     // dust motes carried by the wind
	bgLeaves                   // a few leaves carried by the wind
	bgStars                    // stars that twinkle
	bgPool                     // sparkles on the water of the pools
)

// ambientEntry: the plan of one level (by its title; levels made by the user have none).
type ambientEntry struct {
	Title string
	Fx    bgFx
	Sound string // ambient sound loop (ambient_sound.go); only heard if ambientSound is on
}

// ambientPlan: 13 levels with an effect, every other level has none.
var ambientPlan = []ambientEntry{
	{"MIAMI BEACH", bgShore | bgGlitter | bgCrests | bgSway, "waves"},
	{"PLAYA", bgShore | bgGlitter | bgCrests | bgSway, "waves"},
	{"CRUCERO NOCTURNO", bgCrests | bgJump | bgPool, "waves"},
	{"CEMENTERIO", bgFog, "crickets"},
	{"ESTACIÓN DE ESQUÍ", bgSnow | bgSway, "wind"},
	{"ESTACIÓN ESPACIAL", bgStars, "hum"},
	{"PLAZA EUROPEA", bgLeaves | bgSway, "crowd"},
	{"TORRE EIFFEL", bgLeaves | bgSway, "crowd"},
	{"BULL'S EYE", bgDust | bgSway, "wind"},
	{"SKATE PARK", bgLeaves | bgSway, "crowd"},
	{"FESTIVAL", bgDust | bgSway, "crowd"},
	{"HOUSE PARTY", bgPool | bgSway, ""},
	{"ESTACIÓN DE TREN", bgDust, "train"},
}

func ambientFor(title string) (bgFx, string) {
	for _, e := range ambientPlan {
		if e.Title == title {
			return e.Fx, e.Sound
		}
	}
	return 0, ""
}

type bgCrest struct{ x, y, w, ph float64 }
type bgSpot struct{ x0, y0, x1 float64 }
type bgStar struct{ x, y, ph, sp float64 }

// bgLevel: the precomputed table of a level.
type bgLevel struct {
	def     *LevelDef
	fx      bgFx
	shoreY  float64 // top of the sand (where the sea ends)
	seaTop  float64
	crests  []bgCrest
	spots   []bgSpot
	stars   []bgStar
	ok      []bool // 8 px cells where snow / leaves / dust may be seen (open ground)
	cw, ch  int
	dustCol color.RGBA
}

var bgCur *bgLevel

func titleSeed(s string) int64 {
	h := int64(7)
	for _, r := range s {
		h = h*131 + int64(r)
	}
	return h
}

// bgFor returns the table of the level, building it the first time.
func bgFor(def *LevelDef) *bgLevel {
	if bgCur != nil && bgCur.def == def {
		return bgCur
	}
	b := &bgLevel{def: def}
	for k := range swayCache {
		delete(swayCache, k)
	}
	b.fx, _ = ambientFor(def.Title)
	bgCur = b
	if b.fx == 0 {
		return b
	}
	rng := rand.New(rand.NewSource(titleSeed(def.Title)))
	b.shoreY = 90
	for _, f := range def.Floors {
		if f.Kind == "sand" && f.R.Y < b.shoreY+200 {
			b.shoreY = f.R.Y
			break
		}
	}
	b.seaTop = 46
	seaBand := def.Sky == "sea" && def.Shape == nil
	if b.fx&bgCrests != 0 {
		if seaBand {
			for i := 0; i < 44; i++ {
				b.crests = append(b.crests, bgCrest{rng.Float64() * 940, b.seaTop + rng.Float64()*(b.shoreY-b.seaTop-16), 8 + rng.Float64()*10, rng.Float64() * 6.28})
			}
		} else if def.Shape != nil {
			for try := 0; try < 900 && len(b.crests) < 110; try++ {
				x, y, w := 6+rng.Float64()*930, 46+rng.Float64()*480, 8+rng.Float64()*10
				if def.Shape(x, y, 18) || def.Shape(x+w, y, 18) || def.Shape(x+w/2, y, 18) {
					continue
				}
				b.crests = append(b.crests, bgCrest{x, y, w, rng.Float64() * 6.28})
			}
		}
	}
	if b.fx&bgJump != 0 && def.Shape != nil {
		for try := 0; try < 1500 && len(b.spots) < 10; try++ {
			x, y := 30+rng.Float64()*900, 70+rng.Float64()*450
			dx := 46.0
			if rng.Intn(2) == 0 {
				dx = -dx
			}
			clear := true
			for k := 0.0; k <= 1.0; k += 0.125 {
				if def.Shape(x+dx*k, y, 26) || def.Shape(x+dx*k, y-18*math.Sin(math.Pi*k), 26) {
					clear = false
					break
				}
			}
			if clear && x+dx > 20 && x+dx < 940 {
				b.spots = append(b.spots, bgSpot{x, y, x + dx})
			}
		}
	}
	if b.fx&bgStars != 0 && def.Shape != nil {
		for try := 0; try < 400 && len(b.stars) < 42; try++ {
			x, y := math.Round(rng.Float64()*956)+1, math.Round(rng.Float64()*536)+1
			if def.Shape(x, y, 12) {
				continue
			}
			b.stars = append(b.stars, bgStar{x, y, rng.Float64() * 6.28, 0.8 + rng.Float64()*1.6})
		}
	}
	if b.fx&(bgSnow|bgLeaves|bgDust) != 0 {
		b.buildMask(def)
	}
	b.dustCol = rgba(0xd8c090, 120)
	return b
}

// buildMask marks the 8 px cells that are open ground: an outdoor floor (not a road or a room) with no wall or pool on it.
func (b *bgLevel) buildMask(def *LevelDef) {
	b.cw, b.ch = int(def.W/8)+1, int(def.H/8)+1
	b.ok = make([]bool, b.cw*b.ch)
	for cy := 0; cy < b.ch; cy++ {
		for cx := 0; cx < b.cw; cx++ {
			x, y := float64(cx*8+4), float64(cy*8+4)
			kind := ""
			for i := len(def.Floors) - 1; i >= 0; i-- {
				if def.Floors[i].R.Has(x, y) {
					kind = def.Floors[i].Kind
					break
				}
			}
			good := kind != "" && !floorIndoor(kind) && kind != "asphalt" && kind != "dash" && kind != "helipad"
			if good {
				for _, w := range def.Walls {
					if w.R.Has(x, y) {
						good = false
						break
					}
				}
			}
			if good {
				for _, p := range def.Pools {
					if p.Has(x, y) {
						good = false
						break
					}
				}
			}
			b.ok[cy*b.cw+cx] = good
		}
	}
}

func (b *bgLevel) open(x, y float64) bool {
	cx, cy := int(x/8), int(y/8)
	if x < 0 || y < 0 || cx >= b.cw || cy >= b.ch {
		return false
	}
	return b.ok[cy*b.cw+cx]
}

// bgOp is reused by every effect: nothing is allocated per frame.
var bgOp ebiten.DrawImageOptions

// bgCount: how many rectangles the effects have drawn (tests read it).
var bgCount int

func bgRect(dst *ebiten.Image, x, y, w, h float64, c color.RGBA) {
	if c.A == 0 {
		return
	}
	bgCount++
	bgOp.GeoM.Reset()
	bgOp.GeoM.Scale(w, h)
	bgOp.GeoM.Translate(x, y)
	bgOp.ColorScale.Reset()
	bgOp.ColorScale.ScaleWithColor(c)
	dst.DrawImage(px, &bgOp)
}

func bgWhite(a float64) color.RGBA {
	if a <= 0 {
		return color.RGBA{}
	}
	if a > 255 {
		a = 255
	}
	v := uint8(a)
	return color.RGBA{v, v, v, v}
}

func bgCol(r, g, b uint8, a float64) color.RGBA {
	if a <= 0 {
		return color.RGBA{}
	}
	if a > 255 {
		a = 255
	}
	k := a / 255
	return color.RGBA{uint8(float64(r) * k), uint8(float64(g) * k), uint8(float64(b) * k), uint8(a)}
}

func bgH(i int, seed uint32) float64 { return jpHash(i, 17, seed) }

// bgWind: the gust at a place and a moment, about -1..1 (slow: a period of several seconds).
func bgWind(t, x float64) float64 {
	return 0.7*math.Sin(t*0.55+x*0.011) + 0.3*math.Sin(t*1.5+x*0.03+1.3)
}

// drawBackAnim paints the background animation of the level (see the top of the file).
func (g *Game) drawBackAnim(dst *ebiten.Image) {
	if !ambientOn || g.lvl == nil || g.lvl.Def == nil || px == nil {
		return
	}
	g.drawSeaSky(dst, g.time) // the sun's reflection rocking on the swell, the ship on the horizon (seasky.go)
	b := bgFor(g.lvl.Def)
	if b.fx == 0 {
		return
	}
	t := g.time
	ox, oy := g.camI.X, g.camI.Y
	fx := b.fx
	if fx&bgCrests != 0 {
		for _, c := range b.crests {
			// a slow swell: the crest drifts a few pixels and fades in and out smoothly (no flicker);
			// a darker line right under it gives the wave its shape
			k := 0.5 + 0.5*math.Sin(t*0.6+c.ph)
			x := c.x + math.Round(4*math.Sin(t*0.3+c.ph))
			bgRect(dst, x+ox, c.y+oy, c.w, 1, bgCol(0xb8, 0xd8, 0xff, 14+44*k))
			bgRect(dst, x+2+ox, c.y+2+oy, c.w-3, 1, bgCol(0x10, 0x30, 0x78, 10+30*k))
		}
	}
	if fx&bgGlitter != 0 && b.def.Sky == "sea" {
		span := b.shoreY - 8 - b.seaTop
		for i := 0; i < 16; i++ {
			yy := b.seaTop + bgH(i, 1)*span
			hw := 10 + (yy-b.seaTop)*0.9
			x := 748 + (bgH(i, 2)*2-1)*hw + math.Round(3*math.Sin(t*0.9+float64(i)))
			s := math.Sin(t*2.2 + bgH(i, 3)*6.28)
			a := 220 * s * s * s
			if s > 0 {
				bgRect(dst, x+ox, math.Round(yy)+oy, 2+math.Floor(bgH(i, 4)*3), 1, bgCol(0xff, 0xd8, 0xa0, a))
			}
		}
	}
	if fx&bgShore != 0 {
		for i := 0; i*14 < int(b.def.W); i++ {
			x := float64(i * 14)
			ph := bgH(i, 5) * 6.28
			s := math.Sin(t*0.8 + x*0.012 + ph*0.3)
			y := math.Round(b.shoreY - 8 + 5*(0.5+0.5*s))
			if bgH(i, 6) > 0.2 {
				bgRect(dst, x+ox, y+oy, 12, 1, bgWhite(120))
			}
			s2 := math.Sin(t*0.8 + x*0.012 + ph*0.3 - 0.7)
			bgRect(dst, x+2+ox, math.Round(b.shoreY-12+5*(0.5+0.5*s2))+oy, 8, 1, bgWhite(46))
		}
	}
	if fx&bgJump != 0 {
		g.drawJump(dst, b, t)
	}
	if fx&bgStars != 0 {
		for _, s := range b.stars {
			v := math.Max(0, math.Sin(t*s.sp+s.ph))
			a := 30 + 210*v*v
			bgRect(dst, s.x+ox, s.y+oy, 1, 1, bgWhite(a))
			if a > 170 {
				bgRect(dst, s.x-1+ox, s.y+oy, 3, 1, bgWhite(a*0.35))
				bgRect(dst, s.x+ox, s.y-1+oy, 1, 3, bgWhite(a*0.35))
			}
		}
	}
	if fx&bgPool != 0 {
		for pi, p := range g.lvl.Pools {
			n := int(p.W*p.H/500) + 2 // slow ripples: a light line with a darker one under it, drifting inside the pool
			if n > 10 {
				n = 10
			}
			for i := 0; i < n; i++ {
				ph := bgH(pi*16+i, 7) * 6.28
				w := 6 + math.Floor(bgH(pi*16+i, 10)*6)
				if w > p.W-6 {
					w = p.W - 6
				}
				if w < 2 {
					continue
				}
				x := p.X + 3 + (p.W-6-w)*(0.5+0.5*math.Sin(t*0.25+ph))
				y := p.Y + 3 + bgH(pi*16+i, 9)*(p.H-7)
				k := 0.5 + 0.5*math.Sin(t*0.7+ph)
				bgRect(dst, math.Round(x)+ox, math.Round(y)+oy, w, 1, bgWhite(30+90*k))
				bgRect(dst, math.Round(x)+1+ox, math.Round(y)+1+oy, w-1, 1, bgCol(0x10, 0x40, 0x90, 14+26*k))
			}
			for i := 0; i < 4; i++ { // the old sparkles
				s := math.Sin(t*1.8 + bgH(pi*8+i, 7)*6.28)
				if s <= 0.3 {
					continue
				}
				x := p.X + 3 + bgH(pi*8+i, 8)*(p.W-8) + math.Round(2*math.Sin(t*0.7+float64(i)))
				y := p.Y + 3 + bgH(pi*8+i, 9)*(p.H-6)
				bgRect(dst, math.Round(x)+ox, math.Round(y)+oy, 2, 1, bgWhite(150*s*s))
			}
		}
	}
	if fx&bgSnow != 0 {
		W, H := b.def.W, b.def.H
		for i := 0; i < 90; i++ {
			sp := 12 + float64(i%5)*5
			y := math.Mod(bgH(i, 10)*H+t*sp, H+20) - 10
			x := bgH(i, 11)*W + 10*math.Sin(t*0.5+bgH(i, 12)*6.28)
			if !b.open(x, y) {
				continue
			}
			sz := 1.0
			if i%4 == 0 {
				sz = 2
			}
			bgRect(dst, math.Round(x)+ox, math.Round(y)+oy, sz, sz, bgWhite(150))
		}
	}
	if fx&(bgLeaves|bgDust) != 0 {
		W, H := b.def.W, b.def.H
		n, sp := 14, 26.0
		if fx&bgDust != 0 {
			n, sp = 22, 38
		}
		for i := 0; i < n; i++ {
			x := math.Mod(bgH(i, 13)*(W+40)+t*sp*(0.7+bgH(i, 14)*0.6), W+40) - 20
			y := bgH(i, 15)*H + 7*math.Sin(t*0.9+bgH(i, 16)*6.28) + (x/W)*20
			if !b.open(x, y) {
				continue
			}
			if fx&bgDust != 0 {
				bgRect(dst, math.Round(x)+ox, math.Round(y)+oy, 1, 1, b.dustCol)
				continue
			}
			c := [3]color.RGBA{rgba(0xa8742c, 200), rgba(0x7a9a3a, 200), rgba(0xc8a040, 200)}[i%3]
			bgRect(dst, math.Round(x)+ox, math.Round(y)+oy, 2, 1, c)
			if int(t*3+float64(i))%4 == 0 { // tumbling
				bgRect(dst, math.Round(x)+ox, math.Round(y)+1+oy, 1, 1, c)
			}
		}
	}
	if fx&bgFog != 0 && g.as != nil && g.as.glow != nil {
		for i := 0; i < 8; i++ {
			span := b.def.W + 260
			cx := math.Mod(bgH(i, 20)*span+t*(3+bgH(i, 21)*5), span) - 130
			cy := b.def.H*0.3 + bgH(i, 22)*b.def.H*0.6 + 3*math.Sin(t*0.2+float64(i))
			bgOp.GeoM.Reset()
			bgOp.GeoM.Translate(-48, -48)
			bgOp.GeoM.Scale(1.9+bgH(i, 23), 0.5)
			bgOp.GeoM.Translate(cx+ox, cy+oy)
			bgOp.ColorScale.Reset()
			bgOp.ColorScale.ScaleWithColor(bgCol(0xb8, 0xc4, 0xd8, 34))
			bgCount++
			dst.DrawImage(g.as.glow, &bgOp)
		}
	}
}

// Dolphin: one leap every 20-40 s (the start of leap k is 12 + 30k +/- 5 s), over a spot of open water.
const (
	bgJumpDur  = 1.7
	bgJumpTail = 1.4
)

func bgJumpStart(k int) float64 { return 12 + float64(k)*30 + (bgH(k, 30)-0.5)*10 }

func (g *Game) drawJump(dst *ebiten.Image, b *bgLevel, t float64) {
	if len(b.spots) == 0 {
		return
	}
	kk := int(t / 30)
	for k := kk - 1; k <= kk+1; k++ {
		if k < 0 {
			continue
		}
		st := bgJumpStart(k)
		if t < st || t > st+bgJumpDur+bgJumpTail {
			continue
		}
		sp := b.spots[int(bgH(k, 31)*float64(len(b.spots)))%len(b.spots)]
		ox, oy := g.camI.X, g.camI.Y
		u := (t - st) / bgJumpDur
		dir := 1.0
		if sp.x1 < sp.x0 {
			dir = -1
		}
		path := func(u float64) (float64, float64) {
			return sp.x0 + (sp.x1-sp.x0)*u, sp.y0 - 18*math.Sin(math.Pi*u)
		}
		ring := func(x, y, age float64) { // ripple of the splash
			if age < 0 || age > 1.3 {
				return
			}
			r := 3 + age*11
			a := 120 * (1 - age/1.3)
			bgRect(dst, x-r+ox, y+oy, 2*r, 1, bgWhite(a))
			bgRect(dst, x-r*0.6+ox, y+1+oy, 1.2*r, 1, bgWhite(a*0.6))
		}
		ring(sp.x0, sp.y0, t-st-0.05)
		ring(sp.x1, sp.y0, t-st-bgJumpDur)
		if u > 1 {
			continue
		}
		for j := 0; j < 7; j++ {
			uj := u - float64(j)*0.03
			if uj < 0 || uj > 1 {
				continue
			}
			x, y := path(uj)
			w, h := 4.0, 3.0
			if j == 0 {
				h = 3
			}
			if j >= 5 {
				h, w = 2, 3
			}
			bgRect(dst, math.Round(x-w/2)+ox, math.Round(y-h/2)+oy, w, h, rgba(0x7a98b8, 235))
			bgRect(dst, math.Round(x-w/2)+ox, math.Round(y+h/2)-1+oy, w, 1, rgba(0xdbe6ee, 235))
			if j == 0 { // snout
				bgRect(dst, math.Round(x+dir*2)+ox, math.Round(y)+oy, 2, 1, rgba(0x7a98b8, 235))
			}
			if j == 3 { // dorsal fin
				bgRect(dst, math.Round(x)+ox, math.Round(y-h/2)-1+oy, 1, 1, rgba(0x5a789a, 235))
			}
		}
		x7, y7 := path(math.Max(0, math.Min(1, u-0.21)))
		if u-0.21 > 0 { // tail fluke
			bgRect(dst, math.Round(x7-dir)+ox, math.Round(y7)-1+oy, 2, 1, rgba(0x5a789a, 235))
			bgRect(dst, math.Round(x7-dir)+ox, math.Round(y7)+1+oy, 2, 1, rgba(0x5a789a, 235))
		}
	}
}

// --- trees in the wind -----------------------------------------------------------------

var swayKinds = map[string]bool{"tree": true, "palm": true, "pine": true, "bush": true}

// swayCache: each swaying sprite cut once into 3 horizontal bands (crown, middle, trunk).
var swayCache = map[*ebiten.Image]*[3]*ebiten.Image{}

// drawSway draws prop sprite img with op like DrawImage would, but when the level has wind the
// crown is shifted by 1 px now and then (and the middle half of it): slow, subtle, no new images.
// It returns false if the prop does not sway (the caller then draws it as usual).
func (g *Game) drawSway(dst *ebiten.Image, pr *Prop, img *ebiten.Image, op *ebiten.DrawImageOptions) bool {
	if !ambientOn || g.lvl == nil || !swayKinds[pr.Name] || pr.scale > 0 && pr.scale != 1 || pr.burn > 0 || pr.wreck {
		return false
	}
	if bgFor(g.lvl.Def).fx&bgSway == 0 {
		return false
	}
	w := bgWind(g.time, pr.X) * (0.8 + 0.4*bgH(int(pr.X)+int(pr.Y)*7, 40))
	top, mid := math.Round(1.5*w), math.Round(0.7*w)
	if top == 0 {
		return false
	}
	h := img.Bounds().Dy()
	if h < 12 {
		return false
	}
	bands := swayCache[img]
	if bands == nil {
		bd := img.Bounds()
		c1, c2 := bd.Min.Y+h*40/100, bd.Min.Y+h*65/100
		bands = &[3]*ebiten.Image{
			img.SubImage(image.Rect(bd.Min.X, bd.Min.Y, bd.Max.X, c1)).(*ebiten.Image),
			img.SubImage(image.Rect(bd.Min.X, c1, bd.Max.X, c2)).(*ebiten.Image),
			img.SubImage(image.Rect(bd.Min.X, c2, bd.Max.X, bd.Max.Y)).(*ebiten.Image),
		}
		swayCache[img] = bands
	}
	y1, y2 := float64(h*40/100), float64(h*65/100)
	o := *op
	shifts := [3]float64{top, mid, 0}
	offs := [3]float64{0, y1, y2}
	for i := 0; i < 3; i++ {
		o.GeoM = op.GeoM
		o.GeoM.Translate(shifts[i], offs[i])
		dst.DrawImage(bands[i], &o)
	}
	return true
}
