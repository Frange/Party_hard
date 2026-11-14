package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The living sky of the ESTACIÓN ESPACIAL (drawn by drawBackAnim, ambient.go, after the extra twinkling stars):
// many more stars that twinkle, the two planets of the backdrop turning (their spots and clouds slide across the
// disc), two satellites drifting on their orbits with a blinking light, small ships crossing above and below the
// station every few seconds and now and then a shooting star. Like the rest of ambient.go: a closed-form function
// of g.time and of a table built once per level; nothing is allocated per frame and nothing is drawn over the station.
// ---------------------------------------------------------------------------

// spacePlanets: the planets painted in the backdrop (draw.go, Sky "space"): x, y, radius, colour.
var spacePlanets = [2]struct {
	x, y, r float64
	spot    uint32
}{{110, 96, 58, 0x8a4a2a}, {850, 450, 66, 0xe8f0ff}}

// spaceShipLanes: the y of the lanes the ships fly along (above the north module and below the south one).
var spaceShipLanes = [2]float64{14, 522}

const spaceShipEvery = 7.0 // a ship crosses every 7 s (lanes and directions alternate)

type spaceTable struct {
	def   *LevelDef
	stars []bgStar
}

var spaceCur *spaceTable

// spaceFxCount: how many space effects were drawn in the last frame (tests read it).
var spaceFxCount int

func spaceFor(def *LevelDef) *spaceTable {
	if spaceCur != nil && spaceCur.def == def {
		return spaceCur
	}
	s := &spaceTable{def: def}
	rng := rand.New(rand.NewSource(titleSeed(def.Title) + 5))
	for try := 0; try < 1200 && len(s.stars) < 110; try++ {
		x, y := math.Round(rng.Float64()*956)+1, math.Round(rng.Float64()*536)+1
		if def.Shape != nil && def.Shape(x, y, 14) {
			continue
		}
		inPlanet := false
		for _, p := range spacePlanets {
			if math.Hypot(x-p.x, y-p.y) < p.r+2 {
				inPlanet = true
			}
		}
		if inPlanet {
			continue
		}
		s.stars = append(s.stars, bgStar{x, y, rng.Float64() * 6.28, 0.5 + rng.Float64()*2.2})
	}
	spaceCur = s
	return s
}

// spaceClear: nothing of the sky may be drawn over the station (its hull band included).
func spaceClear(def *LevelDef, x, y, w, h float64) bool {
	if def.Shape == nil {
		return true
	}
	return !def.Shape(x, y, 14) && !def.Shape(x+w, y, 14) && !def.Shape(x, y+h, 14) && !def.Shape(x+w, y+h, 14)
}

func (g *Game) drawSpaceFx(dst *ebiten.Image, t, ox, oy float64) {
	def := g.lvl.Def
	if def.Sky != "space" {
		return
	}
	spaceFxCount = 0
	s := spaceFor(def)
	for _, st := range s.stars {
		v := math.Max(0, math.Sin(t*st.sp+st.ph))
		a := 20 + 200*v*v
		bgRect(dst, st.x+ox, st.y+oy, 1, 1, bgCol(0xe8, 0xe8, 0xff, a))
		spaceFxCount++
	}
	// the planets turn: spots (and clouds on the blue one) slide across the disc and vanish round the back
	for pi, p := range spacePlanets {
		speed := []float64{0.18, 0.12}[pi]
		for i := 0; i < 16; i++ {
			lat := (bgH(i+pi*40, 40)*2 - 1) * 0.8
			yy := p.y + lat*p.r
			half := math.Sqrt(math.Max(0, p.r*p.r-(yy-p.y)*(yy-p.y)))
			lon := bgH(i+pi*40, 41)*6.28 + t*speed
			if math.Cos(lon) <= 0.1 {
				continue // behind the planet
			}
			w := math.Round((5 + bgH(i+pi*40, 42)*9) * math.Cos(lon))
			x := p.x + math.Sin(lon)*(half-w/2-1)
			if w < 1 {
				continue
			}
			a := 120 * math.Cos(lon)
			c := bgCol(uint8(p.spot>>16), uint8(p.spot>>8), uint8(p.spot), a)
			bgRect(dst, math.Round(x-w/2)+ox, math.Round(yy)+oy, w, 2, c)
			spaceFxCount++
		}
	}
	// two satellites on slow orbits, with a light that blinks
	for i, o := range [2][4]float64{{190, 462, 120, 36}, {770, 82, 140, 40}} {
		a := t*[]float64{0.09, -0.07}[i] + float64(i)*2
		x, y := math.Round(o[0]+math.Cos(a)*o[2]), math.Round(o[1]+math.Sin(a)*o[3])
		if !spaceClear(def, x-8, y-3, 16, 6) {
			continue
		}
		bgRect(dst, x-8+ox, y-1+oy, 5, 3, rgb(0x2a4a9a)) // panels
		bgRect(dst, x+4+ox, y-1+oy, 5, 3, rgb(0x2a4a9a))
		bgRect(dst, x-2+ox, y-2+oy, 5, 5, rgb(0xb8bcc8)) // body
		if int(t*2+float64(i))%2 == 0 {
			bgRect(dst, x+ox, y-3+oy, 1, 1, rgb(0xff4040))
		}
		spaceFxCount++
	}
	// a ship crosses every spaceShipEvery seconds, alternating lane and direction
	k := int(t / spaceShipEvery)
	ph := t - float64(k)*spaceShipEvery
	sx := -30 + ph*(1020/5.5)
	if ph < 5.5 {
		lane := spaceShipLanes[k%2]
		dir := 1.0
		if (k/2)%2 == 1 {
			dir = -1
			sx = 990 - sx
		}
		if spaceClear(def, sx-10, lane-4, 22, 9) {
			g.drawSpaceShip(dst, sx+ox, lane+oy, dir, t)
		}
	}
	// a shooting star now and then over the free corner at the top right
	if m := math.Mod(t, 9.5); m < 0.6 {
		f := m / 0.6
		x, y := 620+f*260, 20+f*90
		for j := 0.0; j < 6; j++ {
			xx, yy := math.Round(x-j*4), math.Round(y-j*1.4)
			if spaceClear(def, xx, yy, 2, 1) {
				bgRect(dst, xx+ox, yy+oy, 2, 1, bgWhite(220-j*34))
			}
		}
		spaceFxCount++
	}
}

// drawSpaceShip: a small ship flying along a lane (nose in the direction of travel), with a flickering flame.
func (g *Game) drawSpaceShip(dst *ebiten.Image, x, y, dir, t float64) {
	x = math.Round(x)
	body, dark := rgb(0xc8ccd8), rgb(0x6a7080)
	bgRect(dst, x-7, y-2, 14, 5, body)
	bgRect(dst, x-7, y+2, 14, 1, dark)
	bgRect(dst, x+dir*7-b2f(dir < 0)*3, y-1, 3, 3, body) // the nose
	bgRect(dst, x+dir*2-1, y-1, 3, 2, rgb(0x6ad8ff))     // the cockpit
	bgRect(dst, x-dir*3-1, y-4, 3, 2, dark)              // the fin
	fl := 3 + float64(int(t*12)%3)
	if dir > 0 {
		bgRect(dst, x-7-fl, y-1, fl, 3, rgb(0xffa030))
	} else {
		bgRect(dst, x+7, y-1, fl, 3, rgb(0xffa030))
	}
	spaceFxCount++
}
