package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// The bay of the BARCO PIRATA moves (v1.4.0): wave crests drift and fade on the open water, a dolphin leaps now and
// then and a shark's fin cuts across. Its own table (not in ambientPlan, which keeps 10-14 levels): built once per
// level like bgFor, drawn by drawBackAnim (ambient.go) right after the sea sky; nothing allocated per frame.

var pirSeaCur *bgLevel

// pirSeaCount: what the bay drew in the last frame (tests read it).
var pirSeaCount int

func pirSeaFor(def *LevelDef) *bgLevel {
	if pirSeaCur != nil && pirSeaCur.def == def {
		return pirSeaCur
	}
	b := &bgLevel{def: def, fx: bgCrests | bgJump}
	rng := rand.New(rand.NewSource(titleSeed(def.Title) + 11))
	for try := 0; try < 1400 && len(b.crests) < 150; try++ {
		x, y, w := 6+rng.Float64()*930, pirSea+6+rng.Float64()*(540-pirSea-10), 8+rng.Float64()*10
		if def.Shape(x, y, 18) || def.Shape(x+w, y, 18) || def.Shape(x+w/2, y, 18) {
			continue
		}
		b.crests = append(b.crests, bgCrest{x, y, w, rng.Float64() * 6.28})
	}
	for try := 0; try < 1500 && len(b.spots) < 10; try++ {
		x, y := 30+rng.Float64()*900, pirSea+30+rng.Float64()*440
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
	pirSeaCur = b
	return b
}

// drawPirateSea: the waves, the dolphin and the fin of the pirate bay (nothing on any other level).
func (g *Game) drawPirateSea(dst *ebiten.Image) {
	def := g.lvl.Def
	if !pirateBay(def) || def.Shape == nil {
		return
	}
	pirSeaCount = 0
	n0 := bgCount // like the sea sky (seasky.go) this is not part of the plan's count; pirSeaCount counts it instead
	b := pirSeaFor(def)
	t := g.time
	ox, oy := g.camI.X, g.camI.Y
	for _, c := range b.crests {
		k := 0.5 + 0.5*math.Sin(t*0.7+c.ph)
		x := c.x + math.Round(5*math.Sin(t*0.35+c.ph))
		bgRect(dst, x+ox, c.y+oy, c.w, 1, bgCol(0xd8, 0xd4, 0xf0, 18+60*k))
		bgRect(dst, x+2+ox, c.y+2+oy, c.w-3, 1, bgCol(0x14, 0x1c, 0x38, 12+34*k))
		pirSeaCount++
	}
	g.drawJump(dst, b, t)
	g.drawSharkFin(dst, def, t, ox, oy)
	bgCount = n0
}

// pirateFinLanes: the open water where the shark's fin goes by: in front of the ship (between its hull and the dock
// level) and behind it, under the horizon. x0..x1 at y; the fin crosses one lane every pirateFinEvery seconds.
var pirateFinLanes = [2][3]float64{{20, 940, 100}, {660, 940, 470}}

const pirateFinEvery = 11.0

// drawSharkFin: the dark fin of a shark cutting through the bay of the BARCO PIRATA, with its little wake. A function
// of the time (ambient.go); it is never drawn over the ship, the gangplank or the dock.
func (g *Game) drawSharkFin(dst *ebiten.Image, def *LevelDef, t, ox, oy float64) {
	if !pirateBay(def) || def.Shape == nil {
		return
	}
	k := int(t / pirateFinEvery)
	ph := (t - float64(k)*pirateFinEvery) / 8 // 8 s to cross, 3 s of nothing
	if ph > 1 {
		return
	}
	ln := pirateFinLanes[k%2]
	x := ln[0] + (ln[1]-ln[0])*ph
	dir := 1.0
	if (k/2)%2 == 1 {
		x, dir = ln[1]-(ln[1]-ln[0])*ph, -1
	}
	y := ln[2] + math.Round(2*math.Sin(t*1.3))
	if def.Shape(x, y, 22) || def.Shape(x-12*dir, y, 22) {
		return
	}
	x = math.Round(x)
	fin, finL := rgb(0x2a3440), rgb(0x5a6878)
	for i := 0.0; i < 7; i++ { // the fin: a triangle leaning back
		w := 7 - i
		bgRect(dst, x-dir*(i*0.5)-boolSide(dir, w)+ox, y-i+oy, w, 1, fin)
	}
	bgRect(dst, x+ox, y-6+oy, 1, 2, finL)
	pirSeaCount++
	for i := 0; i < 4; i++ { // the wake
		bgRect(dst, x-dir*float64(6+i*4)-1+ox, y+1+float64(i%2)+oy, 3, 1, bgWhite(110-float64(i)*25))
	}
	bgRect(dst, x-4+ox, y+1+oy, 9, 1, bgWhite(140))
}

func boolSide(dir, w float64) float64 {
	if dir > 0 {
		return w - 1
	}
	return 0
}
