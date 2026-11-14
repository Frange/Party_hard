package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// WallVoid is an invisible solid used for everything outside a shaped venue
// (open sea around a ship, vacuum around a station).
const WallVoid = 8

// voidWalls turns every 8px cell outside shape into an invisible solid and
// merges each row's runs into single rectangles.
func voidWalls(shape func(x, y, g float64) bool, w, h float64) []Wall {
	const cs = 8.0
	var out []Wall
	for y := 0.0; y < h; y += cs {
		run := -1.0
		for x := 0.0; x <= w; x += cs {
			outside := x < w && !shape(x+cs/2, y+cs/2, 0)
			if outside && run < 0 {
				run = x
			}
			if !outside && run >= 0 {
				out = append(out, Wall{Rect{run, y, x - run, cs}, WallVoid})
				run = -1
			}
		}
	}
	return out
}

// yachtShape is the cruise ship: a squared, slightly rounded stern, straight
// sides and a pointed bow to the right. g grows the outline (hull band).
func yachtShape(x, y, g float64) bool {
	const x0, x1, y0, y1, bow, cy = 60.0, 902.0, 60.0, 482.0, 760.0, 271.0
	l, r, t, b := x0-g, x1+g, y0-g, y1+g
	if x < l || x > r || y < t || y > b {
		return false
	}
	rad := 24 + g
	if x < l+rad && (y < t+rad || y > b-rad) {
		dx := l + rad - x
		dy := t + rad - y
		if y > b-rad {
			dy = y - (b - rad)
		}
		return dx*dx+dy*dy <= rad*rad
	}
	if x > bow {
		k := (x - bow) / (r - bow)
		half := (b - t) / 2 * math.Pow(1-k, 0.7)
		return math.Abs(y-cy) <= half
	}
	return true
}

var stationRects = []Rect{
	{240, 130, 480, 280}, // main hall
	{60, 170, 180, 200},  // west dock
	{720, 170, 180, 200}, // east dock
	{380, 40, 200, 90},   // north module
	{380, 410, 200, 90},  // south module
}

// stationShape is the union of the station modules.
func stationShape(x, y, g float64) bool {
	for _, r := range stationRects {
		if r.Grow(g).Has(x, y) {
			return true
		}
	}
	return false
}

func genStation(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "ESTACIÓN ESPACIAL", Intro: "El final del viaje: una estación a la deriva, y una fiesta que no debería estar aquí.",
		W: 960, H: 540, Sky: "space", Police: "car", Transit: "space", Tint: color.RGBA{2, 2, 24, 110},
		CarFrom: Vec{1040, 524}, CarTo: Vec{520, 524}, ParkX0: 400, ParkX1: 580,
		// no helipads: everybody (police and medics too) docks in a spaceship (Transit "space", Police "car"). The two
		// Pads are only the invisible centres of the docking bays: they keep tables, booths and seats out of the bays.
		Start: Vec{480, 344}, Pads: []Vec{{150, 270}, {810, 270}}, Guests: 36 + rng.Intn(8), CellSize: 16,
		Hull: true, Shape: stationShape,
	}
	d.Floors = []Floor{
		F(240, 130, 480, 280, "white"), F(60, 170, 180, 200, "concrete"), F(720, 170, 180, 200, "concrete"),
		F(380, 40, 200, 90, "tile"), F(380, 410, 200, 90, "carpet"),
		F(380, 220, 200, 110, "dance"),
	}
	d.Walls = walls(
		vw(236, 170, 370, 8, WallRoof, [2]float64{240, 300}),
		vw(716, 170, 370, 8, WallRoof, [2]float64{240, 300}),
		hw(380, 580, 126, 8, WallRoof, [2]float64{440, 520}),
		hw(380, 580, 406, 8, WallRoof, [2]float64{440, 520}),
	)
	d.Props = []PropInst{
		P("dj", 480, 182), P("speaker", 400, 178), P("speaker", 560, 178),
		PV("bar", 310, 205, 1), P("punch", 650, 205), P("couch", 300, 380), PV("couch", 660, 385, 2),
		P("table", 450, 370), P("table", 540, 372), P("lamp", 262, 150), P("lamp", 700, 150),
		P("plant", 268, 255), P("plant", 690, 255), P("fridge", 690, 330),
		P("stall", 420, 108), P("stall", 460, 108), P("toilet", 510, 104), P("toilet", 540, 104),
		P("extinguisher", 362, 100),
		PV("couch", 440, 480, 0), P("tv", 520, 478), P("phone", 100, 356), P("phone", 856, 356),
		P("closet", 90, 200), P("closet", 220, 200), P("closet", 740, 200), P("bookshelf", 870, 200),
		// space things where the helipads were (space_props.go)
		P("srocket", 80, 304), P("sdome", 196, 358), PV("swindow", 152, 196, 0), PV("swindow", 806, 196, 2),
		P("ssatdish", 882, 292), P("smodule", 762, 352), PV("swindow", 470, 50, 1),
	}
	d.Performers = []Perf{{Vec{480, 164}, "dj", DirDown}}
	d.Roam = []Rect{R(250, 140, 460, 260), R(70, 180, 160, 180), R(730, 180, 160, 180), R(390, 50, 180, 70), R(390, 420, 180, 70)}
	d.Dance = []Rect{R(388, 228, 184, 94)}
	d.Glows = []Glow{{Vec{480, 275}, color.RGBA{60, 90, 220, 140}, 130, true}, {Vec{150, 270}, color.RGBA{80, 160, 255, 70}, 80, false}, {Vec{810, 270}, color.RGBA{80, 160, 255, 70}, 80, false}}
	d.Guards = [][]Vec{{{290, 180}, {290, 360}}, {{670, 180}, {670, 360}}}
	d.Doors = append(d.Doors, doorLog...)
	d.Walls = append(d.Walls, voidWalls(stationShape, d.W, d.H)...)
	return d
}

// paintSea fills everything outside the ship with open water.
func (g *Game) paintSea(dst *ebiten.Image, shape func(x, y, g float64) bool) {
	rng := rand.New(rand.NewSource(77))
	for y := 40.0; y < screenH; y += 2 {
		col := mixc(rgb(0x1e5a9e), rgb(0x123a70), (y-40)/500)
		run := -1.0
		for x := 0.0; x <= screenW; x += 2 {
			out := x < screenW && !shape(x+1, y+1, 0)
			if out && run < 0 {
				run = x
			}
			if !out && run >= 0 {
				fillRect(dst, run, y, x-run, 2, col)
				run = -1
			}
		}
	}
	for i := 0; i < 900; i++ {
		x, y := rng.Float64()*screenW, 44+rng.Float64()*(screenH-44)
		if shape(x, y, 16) {
			continue
		}
		fillRect(dst, math.Round(x), math.Round(y), 4+float64(rng.Intn(9)), 1, rgba(0x6aa6e8, uint8(40+rng.Intn(60))))
	}
	for y := 40.0; y < screenH; y += 2 {
		for x := 0.0; x < screenW; x += 2 {
			if !shape(x+1, y+1, 0) && shape(x+1, y+1, 20) && rng.Intn(5) < 2 {
				fillRect(dst, x, y, 2, 2, rgba(0xe8f4ff, uint8(30+rng.Intn(70))))
			}
		}
	}
}

// drawShapeHull paints the metal hull band just outside the shape, with
// portholes on the straight flanks, so the venue reads as a ship / station.
func (g *Game) drawShapeHull(dst *ebiten.Image, def *LevelDef) {
	station := def.Sky == "space"
	body, hi, lo := rgb(0x0e2440), rgb(0x1c4068), rgb(0x081830)
	if station {
		body, hi, lo = rgb(0x4a5060), rgb(0x8890a0), rgb(0x2a2e3a)
	}
	for y := 0.0; y < screenH; y += 2 {
		for x := 0.0; x < screenW; x += 2 {
			cx, cy := x+1, y+1
			if def.Shape(cx, cy, 0) || !def.Shape(cx, cy, 14) {
				continue
			}
			col := body
			switch {
			case !def.Shape(cx, cy, 12):
				col = hi
			case def.Shape(cx, cy, 2):
				col = lo
			}
			fillRect(dst, x, y, 2, 2, col)
		}
	}
	if station {
		// solar panels on the east and west docks
		for _, p := range []Rect{{4, 190, 50, 160}, {906, 190, 50, 160}} {
			fillRect(dst, p.X, p.Y, p.W, p.H, rgb(0x1a2f6a))
			for yy := p.Y; yy < p.Y+p.H; yy += 10 {
				fillRect(dst, p.X, yy, p.W, 1, rgb(0x4a6ac0))
			}
			for xx := p.X; xx < p.X+p.W; xx += 12 {
				fillRect(dst, xx, p.Y, 1, p.H, rgb(0x4a6ac0))
			}
			fillRect(dst, p.X, p.Y, p.W, 2, rgb(0x8a9ad8))
		}
		return
	}
	pcol := rgb(0xf0d060)
	r := def.HullRect
	for x := r.X + 36; x < 740; x += 36 {
		fillRect(dst, x, r.Y-9, 5, 5, pcol)
		fillRect(dst, x, r.Y+r.H+5, 5, 5, pcol)
	}
	if def.HullName != "" {
		g.drawTextC(dst, def.HullName, 330, r.Y+r.H+17, rgb(0x9ab4d8), 1)
	}
}
