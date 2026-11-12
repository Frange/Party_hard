package main

import (
	"image/color"
	"math"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The last party: the island of the first film. It is not one of the 21
// campaign levels and never shows up in random mode: you only get here after
// beating the campaign with every trophy (see ending.go), or from the DEBUG
// menu. The engine shows one fixed 960x540 screen, so this is the biggest the
// island can be; the whole background is painted by code (paintJurassic) and
// the dinosaurs, buildings and jeeps are props (props8.go).
// ---------------------------------------------------------------------------

// specialJurassic is the level index of the island (it lives outside levelDefs).
const specialJurassic = 1000

const (
	islandCX, islandCY = 478.0, 272.0
	islandRX, islandRY = 398.0, 232.0
)

var (
	jpPier = Rect{118, 452, 98, 34}
	jpDock = Rect{444, 486, 68, 28} // in front of the main gate: the boats of the police and the rest moor here
)

// islandSD is a rough signed distance to the shore in pixels: negative on land.
func islandSD(x, y float64) float64 {
	u, v := (x-islandCX)/islandRX, (y-islandCY)/islandRY
	rho := math.Hypot(u, v)
	th := math.Atan2(v, u)
	r := 1 + 0.055*math.Sin(3*th+0.8) + 0.045*math.Sin(5*th+2.1) + 0.03*math.Sin(9*th+0.3) + 0.03*math.Sin(2*th+1.0)
	sd := (rho - r) * 310
	if jpPier.Has(x, y) || jpDock.Has(x, y) { // the piers stick out into the sea
		sd = math.Min(sd, -2)
	}
	return sd
}

func jurassicShape(x, y, g float64) bool { return islandSD(x, y) <= g }

// ---- layout ---------------------------------------------------------------------------

var (
	jpPenA     = Rect{205, 112, 150, 84}  // brachiosaurus paddock
	jpPenB     = Rect{125, 224, 150, 86}  // triceratops paddock
	jpPenR     = Rect{602, 196, 128, 100} // raptor pen
	jpLab      = Rect{150, 334, 164, 100} // laboratory compound
	jpLake     = Rect{362, 122, 200, 86}
	jpPond     = Rect{690, 372, 96, 44}
	jpPlaza    = Rect{372, 296, 216, 48}
	jpDance    = Rect{290, 388, 84, 52}
	jpGateZone = Rect{376, 372, 206, 140} // the gate, its sign, its pillars and its way: nothing else may stand here
	jpPads     = []Vec{{640, 128}}        // a helipad that is only scenery: the police come by boat
	jpCliff    = Rect{330, 46, 232, 76}
	jpTunnel   = Rect{700, 140, 166, 112}
	jpCave     = Rect{728, 198, 84, 52} // opening of the tunnel
	jpRoadV    = Rect{460, 340, 36, 170}
	jpRoadH    = Rect{320, 340, 470, 36}
	jpRoadE    = Rect{752, 252, 36, 124}
	jpFenceCol = rgb(0x8a6a44)
)

// jpClearings are the open grounds painted over the jungle; the rest is the dense forest.
var jpClearings = []struct {
	r    Rect
	kind string
}{
	{jpPenA.Grow(10), "grass"}, {jpPenB.Grow(10), "grass"}, {jpPenR.Grow(10), "grass"},
	{jpLab.Grow(6), "concrete"}, {jpPlaza, "stone"}, {Rect{372, 340, 216, 40}, "grass"},
	{Rect{590, 70, 100, 100}, "pave"},
	{Rect{300, 380, 130, 90}, "grass"}, {Rect{526, 380, 130, 90}, "grass"},
	{Rect{600, 300, 150, 40}, "grass"}, {Rect{800, 252, 90, 80}, "grass"},
	{Rect{610, 376, 80, 90}, "grass"}, {Rect{562, 112, 24, 36}, "grass"},
}

func inRects(x, y float64, rs ...Rect) bool {
	for _, r := range rs {
		if r.Has(x, y) {
			return true
		}
	}
	return false
}

func jpOnRoad(x, y float64) bool {
	return jpRoadV.Has(x, y) || jpRoadH.Has(x, y) || jpRoadE.Has(x, y)
}

func jpInClearing(x, y float64) bool {
	for _, c := range jpClearings {
		if c.r.Has(x, y) {
			return true
		}
	}
	return jpOnRoad(x, y) || jpLake.Grow(8).Has(x, y) || jpPond.Grow(8).Has(x, y) || jpPier.Has(x, y) ||
		jpCliff.Has(x, y) || jpTunnel.Grow(4).Has(x, y)
}

// ---- noise -----------------------------------------------------------------------------

func jpHash(x, y int, seed uint32) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + seed*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffff) / 65535
}

func jpNoise(x, y, scale float64, seed uint32) float64 {
	fx, fy := x/scale, y/scale
	ix, iy := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(ix), fy-float64(iy)
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	a := jpHash(ix, iy, seed)
	b := jpHash(ix+1, iy, seed)
	c := jpHash(ix, iy+1, seed)
	d := jpHash(ix+1, iy+1, seed)
	return (a*(1-tx)+b*tx)*(1-ty) + (c*(1-tx)+d*tx)*ty
}

// ---- painting ----------------------------------------------------------------------------

type jpTuft struct {
	x, y, r float64
	col     int
}

var jpGreens = []uint32{0x2f7a30, 0x3a8a36, 0x286a2c, 0x449a3c}

func jpTufts() map[[2]int][]jpTuft {
	rng := rand.New(rand.NewSource(1993))
	const cell = 9
	out := map[[2]int][]jpTuft{}
	for cy := -1; cy < 540/cell+2; cy++ {
		for cx := -1; cx < 960/cell+2; cx++ {
			t := jpTuft{x: float64(cx*cell) + rng.Float64()*cell, y: float64(cy*cell) + rng.Float64()*cell, r: 6 + rng.Float64()*3.2, col: rng.Intn(len(jpGreens))}
			out[[2]int{cx, cy}] = []jpTuft{t}
		}
	}
	return out
}

func jungleAt(tufts map[[2]int][]jpTuft, x, y int) color.RGBA {
	const cell = 9
	cx, cy := x/cell, y/cell
	best, bd := -1, 99.0
	var bt jpTuft
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			for _, t := range tufts[[2]int{cx + dx, cy + dy}] {
				d := math.Hypot(float64(x)+0.5-t.x, float64(y)+0.5-t.y) / t.r
				// later (lower) tufts overlap earlier ones
				d -= (t.y - float64(y)) * 0.0
				if d < 1 && d < bd {
					best, bd, bt = 1, d, t
				}
			}
		}
	}
	if best < 0 {
		return rgb(0x173a1e)
	}
	base := rgb(jpGreens[bt.col])
	ldx, ldy := (float64(x)+0.5-bt.x)/bt.r, (float64(y)+0.5-bt.y)/bt.r
	l := ldx*-0.55 + ldy*-0.8
	switch {
	case l > 0.42:
		return shade(base, 1.45)
	case l > 0.1:
		return shade(base, 1.18)
	case bd > 0.78 || l < -0.5:
		return shade(base, 0.62)
	}
	return base
}

// jpPalmTop draws a palm seen from above (a star of fronds).
func jpPalmTop(b *pixBuf, cx, cy int, rng *rand.Rand, big float64) {
	n := 7
	for i := 0; i < n; i++ {
		a := float64(i)/float64(n)*2*math.Pi + rng.Float64()*0.4
		ln := (10 + rng.Float64()*3) * big
		for t := 0.0; t < ln; t += 0.7 {
			x := float64(cx) + math.Cos(a)*t
			y := float64(cy) + math.Sin(a)*t*0.75 + t*t*0.012
			col := mixc(rgb(0x2a8a34), rgb(0x1a5a28), t/ln)
			if math.Sin(a) < -0.2 {
				col = shade(col, 1.3)
			}
			b.set(int(x), int(y), col)
			b.set(int(x), int(y)+1, shade(col, 0.7))
		}
	}
	b.ellipse(float64(cx), float64(cy), 1.8, 1.4, rgb(0x6a4a2a))
}

func jpFence(b *pixBuf, r Rect) {
	col := jpFenceCol
	if r.W >= r.H { // horizontal
		for x := r.X; x < r.X+r.W; x++ {
			b.set(int(x), int(r.Y)+1, shade(col, 1.2))
			b.set(int(x), int(r.Y)+3, col)
			if int(x)%12 < 2 {
				for y := 0; y < 6; y++ {
					b.set(int(x), int(r.Y)+y-1, shade(col, 0.75+0.12*float64(y%3)))
				}
				b.set(int(x), int(r.Y)-1, shade(col, 1.35))
			}
			b.set(int(x), int(r.Y)+5, rgba(0x000000, 70))
		}
		return
	}
	for y := r.Y; y < r.Y+r.H; y++ {
		b.set(int(r.X)+1, int(y), shade(col, 1.2))
		b.set(int(r.X)+3, int(y), col)
		if int(y)%12 < 2 {
			for x := 0; x < 6; x++ {
				b.set(int(r.X)+x-1, int(y), shade(col, 0.75+0.12*float64(x%3)))
			}
		}
		b.set(int(r.X)+5, int(y), rgba(0x000000, 70))
	}
}

// jpFenceRects are the paddock fences (also the solid walls of the level).
func jpFenceWalls() []Wall {
	gate := func(a, b float64) [2]float64 { return [2]float64{a, b} }
	return walls(
		// brachiosaurus paddock: gate on the south side
		hw(jpPenA.X, jpPenA.X+jpPenA.W, jpPenA.Y, 4, WallFence),
		hw(jpPenA.X, jpPenA.X+jpPenA.W, jpPenA.Y+jpPenA.H, 4, WallFence, gate(300, 346)),
		vw(jpPenA.X, jpPenA.Y, jpPenA.Y+jpPenA.H+4, 4, WallFence),
		vw(jpPenA.X+jpPenA.W, jpPenA.Y, jpPenA.Y+jpPenA.H+4, 4, WallFence),
		// triceratops paddock: gate on the east side
		hw(jpPenB.X, jpPenB.X+jpPenB.W, jpPenB.Y, 4, WallFence),
		hw(jpPenB.X, jpPenB.X+jpPenB.W, jpPenB.Y+jpPenB.H, 4, WallFence),
		vw(jpPenB.X, jpPenB.Y, jpPenB.Y+jpPenB.H+4, 4, WallFence),
		vw(jpPenB.X+jpPenB.W, jpPenB.Y, jpPenB.Y+jpPenB.H+4, 4, WallFence, gate(248, 292)),
		// velociraptor pen: gate on the south side
		hw(jpPenR.X, jpPenR.X+jpPenR.W, jpPenR.Y, 4, WallFence),
		hw(jpPenR.X, jpPenR.X+jpPenR.W, jpPenR.Y+jpPenR.H, 4, WallFence, gate(646, 690)),
		vw(jpPenR.X, jpPenR.Y, jpPenR.Y+jpPenR.H+4, 4, WallFence),
		vw(jpPenR.X+jpPenR.W, jpPenR.Y, jpPenR.Y+jpPenR.H+4, 4, WallFence),
		// laboratory compound: gate on the east side
		hw(jpLab.X, jpLab.X+jpLab.W, jpLab.Y, 4, WallFence),
		hw(jpLab.X, jpLab.X+jpLab.W, jpLab.Y+jpLab.H, 4, WallFence),
		vw(jpLab.X, jpLab.Y, jpLab.Y+jpLab.H+4, 4, WallFence),
		vw(jpLab.X+jpLab.W, jpLab.Y, jpLab.Y+jpLab.H+4, 4, WallFence, gate(368, 414)),
	)
}

// ---------------------------------------------------------------------------

func paintJurassic() *pixBuf {
	b := newPix(screenW, screenH)
	tufts := jpTufts()
	rngSea := rand.New(rand.NewSource(7))
	_ = rngSea
	// ---- the ground: sea, beach, cliffs, forest
	for y := 0; y < screenH; y++ {
		for x := 0; x < screenW; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			sd := islandSD(fx, fy)
			var col color.RGBA
			switch {
			case sd > 0:
				col = jpSea(x, y, sd)
			case sd > -15:
				col = jpBeach(x, y, sd)
			default:
				col = jungleAt(tufts, x, y)
			}
			b.set(x, y, col)
		}
	}
	// rocky shore on the north and the west side
	for y := 0; y < screenH; y++ {
		for x := 0; x < screenW; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			sd := islandSD(fx, fy)
			if sd > 0 || sd < -26 || jpPier.Has(fx, fy) {
				continue
			}
			ang := math.Atan2(fy-islandCY, fx-islandCX)
			north := ang < -0.25 && ang > -2.9
			west := math.Abs(ang) > 2.4
			if (north || west) && jpNoise(fx, fy, 18, 5) > 0.42 && sd > -22 {
				n := jpNoise(fx, fy, 5, 9)
				col := mixc(rgb(0x5a4c40), rgb(0x8a7a68), n)
				if int(fx)%7 < 2 {
					col = shade(col, 0.78)
				}
				if sd < -16 {
					col = shade(col, 1.25)
				}
				b.set(x, y, col)
			}
		}
	}
	// ---- clearings
	for _, cl := range jpClearings {
		jpClearing(b, cl.r, cl.kind)
	}
	// ---- roads
	for _, r := range []Rect{jpRoadV, jpRoadH, jpRoadE} {
		jpRoad(b, r)
	}
	// ---- cliff, waterfall, lake
	jpCliffPaint(b)
	jpWater(b, jpLake, 1)
	jpWater(b, jpPond, 2)
	// ---- the cave
	jpTunnelPaint(b)
	// ---- helipads
	for _, p := range jpPads {
		jpHelipad(b, p, 34)
	}
	jpDockPaint(b)
	if k := propKinds["jp_boat"]; k != nil { // the ferry moored at the pier
		// An exported/edited jp_boat (assets/sprites/objetos/jp_boat) replaces the
		// built-in kind with Native frames and no Draw func: paint that frame
		// (already outlined, with a 1 px margin) instead of calling a nil Draw.
		var bt *pixBuf
		ox, oy := 26, 448
		if k.Draw != nil {
			bt = newPix(k.W, k.H)
			k.Draw(bt, 0)
			bt = bt.outlinedSel(0.4)
		} else if len(k.Native) > 0 {
			bt, ox, oy = k.Native[0], 25, 447
		}
		for y := 0; bt != nil && y < bt.h; y++ {
			for x := 0; x < bt.w; x++ {
				if cc := bt.get(x, y); cc.A > 0 {
					b.set(ox+x, oy+y, cc)
				}
			}
		}
	}
	// ---- pier
	jpPierPaint(b)
	// ---- fences
	for _, w := range jpFenceWalls() {
		jpFence(b, w.R)
	}
	jpFence(b, Rect{jpLab.X + jpLab.W, jpLab.Y, 4, 38})
	// ---- scattered palms in the forest
	rngJ := rand.New(rand.NewSource(33))
	for i := 0; i < 130; i++ {
		x, y := 90+rngJ.Float64()*790, 40+rngJ.Float64()*450
		if islandSD(x, y) > -22 || jpInClearing(x, y) || jpInClearing(x-12, y) || jpInClearing(x+12, y) || jpInClearing(x, y+12) || jpInClearing(x, y-12) {
			continue
		}
		jpPalmTop(b, int(x), int(y), rngJ, 0.9+rngJ.Float64()*0.5)
	}
	// ---- soft shadows under the cliffs and along the shore
	for y := 0; y < screenH; y++ {
		for x := 0; x < screenW; x++ {
			sd := islandSD(float64(x)+0.5, float64(y)+0.5)
			if sd > 0 && sd < 7 {
				i := (y*b.w + x) * 4
				k := 1 - sd/7*0.12
				b.p[i], b.p[i+1], b.p[i+2] = uint8(float64(b.p[i])*k), uint8(float64(b.p[i+1])*k), uint8(float64(b.p[i+2])*k)
			}
		}
	}
	return b
}

func jpSea(x, y int, sd float64) color.RGBA {
	deep, mid, shallow := rgb(0x16468c), rgb(0x1e68b4), rgb(0x3ab8c8)
	var col color.RGBA
	switch {
	case sd < 12:
		col = mixc(rgb(0xb8f0f0), shallow, sd/12)
	case sd < 36:
		col = mixc(shallow, mid, (sd-12)/24)
	default:
		col = mixc(mid, deep, math.Min(1, (sd-36)/160))
	}
	fx, fy := float64(x), float64(y)
	w := jpNoise(fx+fy*0.3, fy*2.4, 9, 3)
	if w > 0.72 {
		col = shade(col, 1.18)
	} else if w < 0.22 {
		col = shade(col, 0.88)
	}
	// white crests
	if jpHash(x/3, y/2, 17) > 0.992 && sd > 8 {
		col = rgb(0xdff4ff)
	}
	// foam on the shore
	f := jpNoise(fx, fy, 3, 21)
	if sd < 3.5+f*2.5 {
		col = mixc(rgb(0xf4fcff), col, math.Max(0, sd-1)/5)
	} else if sd > 8 && sd < 10.5 && f > 0.55 {
		col = mixc(col, rgb(0xcff4f4), 0.55)
	}
	return col
}

func jpBeach(x, y int, sd float64) color.RGBA {
	n := jpNoise(float64(x), float64(y), 4, 41)
	col := mixc(rgb(0xe8d49c), rgb(0xd6bd80), n)
	if sd > -4 { // wet sand
		col = mixc(col, rgb(0xb89c64), (sd+4)/4*0.7)
	}
	if jpHash(x, y, 55) > 0.985 {
		col = rgb(0xf8eccc)
	}
	return col
}

func jpClearing(b *pixBuf, r Rect, kind string) {
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if islandSD(float64(x)+0.5, float64(y)+0.5) > -14 {
				continue
			}
			// soft, ragged edge for the grass; hard edges for the paved ground
			e := math.Min(math.Min(float64(x)-r.X, r.X+r.W-1-float64(x)), math.Min(float64(y)-r.Y, r.Y+r.H-1-float64(y)))
			var col color.RGBA
			switch kind {
			case "grass":
				if e < 2+jpNoise(float64(x), float64(y), 6, 61)*5 {
					continue
				}
				n := jpNoise(float64(x), float64(y), 7, 63)
				col = mixc(rgb(0x4f9a38), rgb(0x6ab446), n)
				if jpHash(x, y, 65) > 0.93 {
					col = shade(col, 1.25)
				}
				if jpHash(x, y, 67) > 0.992 {
					col = rgb([]uint32{0xf4e050, 0xf08ab0, 0xffffff}[int(jpHash(x, y, 69)*3)%3])
				}
			case "stone":
				col = mixc(rgb(0xb8aa92), rgb(0xcbbfa6), jpNoise(float64(x), float64(y), 5, 71))
				if (x/12+y/12)%2 == 0 {
					col = shade(col, 0.94)
				}
				if x%12 == 0 || y%12 == 0 {
					col = shade(col, 0.82)
				}
			case "concrete":
				col = mixc(rgb(0x9a9ea2), rgb(0xb0b4b6), jpNoise(float64(x), float64(y), 6, 73))
				if x%24 == 0 || y%24 == 0 {
					col = shade(col, 0.82)
				}
			case "pave":
				col = mixc(rgb(0x8a8c90), rgb(0x9a9ca0), jpNoise(float64(x), float64(y), 6, 75))
				if e < 2 {
					col = shade(col, 0.7)
				}
			}
			b.set(x, y, col)
		}
	}
}

func jpRoad(b *pixBuf, r Rect) {
	horiz := r.W >= r.H
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if islandSD(float64(x)+0.5, float64(y)+0.5) > -4 {
				continue
			}
			col := mixc(rgb(0x3a3a44), rgb(0x46464f), jpNoise(float64(x), float64(y), 4, 81))
			if jpHash(x, y, 83) > 0.97 {
				col = shade(col, 1.3)
			}
			b.set(x, y, col)
		}
	}
	edge := rgb(0x8a8a98)
	dash := rgb(0xe8d870)
	if horiz {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			b.set(x, int(r.Y), edge)
			b.set(x, int(r.Y+r.H)-1, edge)
			if (x/12)%2 == 0 {
				b.set(x, int(r.Y+r.H/2), dash)
			}
		}
	} else {
		for y := int(r.Y); y < int(r.Y+r.H); y++ {
			b.set(int(r.X), y, edge)
			b.set(int(r.X+r.W)-1, y, edge)
			if (y/12)%2 == 0 {
				b.set(int(r.X+r.W/2), y, dash)
			}
		}
	}
}

func jpHelipad(b *pixBuf, p Vec, rad float64) {
	b.ellipse(p.X, p.Y, rad+4, rad+4, rgb(0x3a3d46))
	b.ellipse(p.X, p.Y, rad+2, rad+2, rgb(0xe8c820))
	b.ellipse(p.X, p.Y, rad, rad, rgb(0x2a2e38))
	for a := 0.0; a < 2*math.Pi; a += 0.05 {
		b.set(int(p.X+math.Cos(a)*(rad-5)), int(p.Y+math.Sin(a)*(rad-5)), rgb(0xf0f0f0))
		b.set(int(p.X+math.Cos(a)*(rad-6)), int(p.Y+math.Sin(a)*(rad-6)), rgb(0xf0f0f0))
	}
	s := int(rad * 0.4)
	cx, cy := int(p.X), int(p.Y)
	b.rect(cx-s, cy-s, 4, 2*s, rgb(0xf0f0f0))
	b.rect(cx+s-4, cy-s, 4, 2*s, rgb(0xf0f0f0))
	b.rect(cx-s, cy-2, 2*s, 4, rgb(0xf0f0f0))
}

func jpCliffPaint(b *pixBuf) {
	r := jpCliff
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			// ragged outline
			top := r.Y + 6 + jpNoise(fx, 0, 14, 91)*22
			side := math.Min(fx-r.X, r.X+r.W-fx)
			if fy < top || side < 4+jpNoise(0, fy, 9, 93)*20 {
				continue
			}
			if islandSD(fx, fy) > -2 {
				continue
			}
			n := jpNoise(fx*3, fy, 9, 95)
			col := mixc(rgb(0x5a4c40), rgb(0x8a7a68), n)
			if int(fx)%6 < 1 {
				col = shade(col, 0.74)
			}
			if fy < top+5 {
				col = mixc(rgb(0x3a8a38), col, (fy-top)/5)
			}
			if jpHash(x, y, 97) > 0.94 {
				col = shade(col, 1.3)
			}
			b.set(x, y, col)
		}
	}
	// the waterfall
	wx0, wx1 := 456, 492
	for y := 66; y < 122; y++ {
		for x := wx0; x < wx1; x++ {
			k := math.Abs(float64(x)-float64(wx0+wx1)/2) / 18
			if k > 1-jpNoise(0, float64(y), 6, 99)*0.2 {
				continue
			}
			n := jpNoise(float64(x)*4, float64(y)*0.6, 3, 101)
			col := mixc(rgb(0x9ad4f4), rgb(0xf4fcff), n)
			if (x+y/3)%5 == 0 {
				col = rgb(0xffffff)
			}
			b.set(x, y, col)
		}
	}
	// mist at the foot
	for y := 114; y < 128; y++ {
		for x := wx0 - 12; x < wx1+12; x++ {
			d := math.Hypot(float64(x-(wx0+wx1)/2)/30, float64(y-121)/7)
			if d < 1 && jpHash(x, y, 103) > 0.35 {
				b.set(x, y, rgb(0xeaf8ff))
			}
		}
	}
}

func jpWater(b *pixBuf, r Rect, seed uint32) {
	// a blobby shore around the rectangle
	for y := int(r.Y) - 7; y < int(r.Y+r.H)+7; y++ {
		for x := int(r.X) - 7; x < int(r.X+r.W)+7; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if islandSD(fx, fy) > -6 {
				continue
			}
			inx := math.Min(fx-r.X, r.X+r.W-fx)
			iny := math.Min(fy-r.Y, r.Y+r.H-fy)
			e := math.Min(inx, iny) + (jpNoise(fx, fy, 7, 111+seed)-0.5)*5
			if e < -5 {
				continue
			}
			var col color.RGBA
			switch {
			case e < -2:
				col = rgb(0x8a7a58) // mud and stones
			case e < 0:
				col = rgb(0xcbb888) // sand edge
			default:
				d := math.Min(1, e/14)
				col = mixc(rgb(0x4ad0d8), rgb(0x1e78b8), d)
				if jpNoise(fx*1.6, fy, 6, 113+seed) > 0.7 {
					col = shade(col, 1.2)
				}
				if jpHash(x/2, y, 115) > 0.99 {
					col = rgb(0xdff4ff)
				}
			}
			b.set(x, y, col)
		}
	}
}

func jpTunnelPaint(b *pixBuf) {
	r := jpTunnel
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			top := r.Y + 4 + jpNoise(fx, 0, 12, 121)*18
			side := math.Min(fx-r.X, r.X+r.W-fx)
			if fy < top || side < 3+jpNoise(0, fy, 8, 123)*8 {
				continue
			}
			n := jpNoise(fx*0.7, fy*1.8, 7, 125)
			col := mixc(rgb(0x6a5c4e), rgb(0xa09078), n)
			if int(fy+jpNoise(fx, 0, 20, 129)*9)%7 < 1 {
				col = shade(col, 0.7)
			}
			if jpNoise(fx, fy, 5, 133) > 0.78 {
				col = mixc(col, rgb(0x5a7a3a), 0.5)
			}
			if jpHash(x, y, 127) > 0.95 {
				col = shade(col, 1.3)
			}
			if fy < top+4 {
				col = mixc(rgb(0x3a8a38), col, (fy-top)/4)
			}
			b.set(x, y, col)
		}
	}
	// the dark mouth, an arch
	c := jpCave
	for y := int(c.Y) - 14; y < int(c.Y+c.H); y++ {
		for x := int(c.X) - 4; x < int(c.X+c.W)+4; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			cx := c.X + c.W/2
			rad := c.W/2 + 4
			cyArch := c.Y + 6
			in := false
			if fy >= cyArch {
				in = math.Abs(fx-cx) <= rad
			} else {
				in = math.Hypot((fx-cx)/rad, (fy-cyArch)/(rad*0.9)) <= 1
			}
			if !in {
				continue
			}
			col := mixc(rgb(0x1a1612), rgb(0x0a0806), (fy-c.Y)/c.H)
			edge := math.Abs(fx-cx) > rad-3 || fy < cyArch-rad*0.9+3
			if edge {
				col = rgb(0x4a3e34)
			}
			b.set(x, y, col)
		}
	}
	// rocks scattered in front
	rng := rand.New(rand.NewSource(131))
	for i := 0; i < 7; i++ {
		x, y := 706+rng.Float64()*130, 244+rng.Float64()*10
		b.ellipse(x, y, 4+rng.Float64()*3, 3, rgb(0x6a5c4e))
		b.ellipse(x-1, y-1, 3, 2, rgb(0x9a8a74))
	}
}

func jpDockPaint(b *pixBuf) {
	r := jpDock
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			col := mixc(rgb(0x9a7a4c), rgb(0xb08a58), jpHash(x, y/2, 151))
			if (x-int(r.X))%5 == 0 {
				col = rgb(0x5a4026)
			}
			if y == int(r.Y+r.H)-1 {
				col = rgb(0x4a3420)
			}
			b.set(x, y, col)
		}
	}
	for y := int(r.Y) + 2; y < int(r.Y+r.H); y += 12 { // posts
		b.rect(int(r.X)-2, y, 3, 4, rgb(0x3a2a1a))
		b.rect(int(r.X+r.W)-1, y, 3, 4, rgb(0x3a2a1a))
	}
	// mooring rings
	b.rect(int(r.X)+4, int(r.Y+r.H)-4, 2, 2, rgb(0x8a8a98))
	b.rect(int(r.X+r.W)-6, int(r.Y+r.H)-4, 2, 2, rgb(0x8a8a98))
}

func jpPierPaint(b *pixBuf) {
	r := jpPier
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			col := mixc(rgb(0x9a7a4c), rgb(0xb08a58), jpHash(x/2, y, 141))
			if (y-int(r.Y))%5 == 0 {
				col = rgb(0x5a4026)
			}
			if x == int(r.X) || x == int(r.X+r.W)-1 {
				col = rgb(0x4a3420)
			}
			b.set(x, y, col)
		}
	}
	for x := int(r.X); x < int(r.X+r.W); x += 14 { // posts
		b.rect(x, int(r.Y)-2, 3, 4, rgb(0x3a2a1a))
		b.rect(x, int(r.Y+r.H)-2, 3, 4, rgb(0x3a2a1a))
	}
}

// ---- the level --------------------------------------------------------------------------

var jpBGCache *ebiten.Image

func jurassicBG() *ebiten.Image {
	if jpBGCache == nil {
		jpBGCache = paintJurassic().img()
	}
	return jpBGCache
}

// jurassicLevel builds the island. It is a fresh LevelDef on every call.
func jurassicLevel() *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "ISLA NUBLADA", Intro: "La última fiesta: una isla, un parque temático y unos invitados con muchos dientes. Hay vallas eléctricas. Estaban.",
		W: 960, H: 540, Sky: "none", Police: "car", Transit: "sea", Day: true, Tint: color.RGBA{70, 36, 8, 34}, CharScale: 0.70, PropScale: 0.65,
		Start:   Vec{560, 324},
		CarFrom: Vec{1040, 520}, CarTo: Vec{478, 520}, ParkX0: 270, ParkX1: 478,
		Guests: 46, CellSize: 14,
		PaintBG: jurassicBG,
	}
	// the roads are painted in the background; as floors they keep the engine from dropping things on them
	d.Floors = []Floor{{R: jpDance, Kind: "dance"}, {R: jpGateZone, Kind: "asphalt"}, {R: Rect{404, 208, 132, 96}, Kind: "asphalt"}, // nothing is dropped behind the rotunda
		{R: jpRoadV, Kind: "asphalt"}, {R: jpRoadH, Kind: "asphalt"}, {R: jpRoadE, Kind: "asphalt"}}
	d.Pools = []Rect{
		R(372, 130, 180, 72), // the lake: the only place to swim
		jpPond,
	}
	d.Swim = true
	d.Walls = jpFenceWalls()
	// the cliff and the mouth of the tunnel are solid rock
	d.Walls = append(d.Walls,
		Wall{R(jpCliff.X+8, jpCliff.Y+4, jpCliff.W-16, 66), WallVoid},
		Wall{R(jpTunnel.X+6, jpTunnel.Y+10, jpCave.X-jpTunnel.X-6, jpTunnel.H-14), WallVoid},
		Wall{R(jpCave.X+jpCave.W, jpTunnel.Y+10, jpTunnel.X+jpTunnel.W-jpCave.X-jpCave.W-6, jpTunnel.H-14), WallVoid},
		Wall{R(jpCave.X, jpTunnel.Y+10, jpCave.W, jpCave.Y-jpTunnel.Y-10), WallVoid},
	)
	// no traffic on the island: the roads are only painted and the jeeps are parked scenery
	d.Props = jurassicProps()
	jpScaleProps(d.Props)
	d.Performers = []Perf{{Vec{330, 448}, "dj", DirUp}}
	d.Roam = []Rect{
		R(380, 304, 200, 34), R(300, 384, 130, 76), R(528, 384, 130, 76),
		R(176, 360, 124, 24),
		R(610, 92, 70, 60),
		R(602, 300, 36, 34), R(800, 256, 84, 70), R(560, 384, 90, 70),
		R(306, 300, 66, 34), R(540, 176, 44, 100), R(748, 300, 36, 30), R(704, 316, 40, 20),
	}
	d.Dance = []Rect{jpDance.Grow(-2)}
	d.Guards = [][]Vec{
		{{300, 380}, {420, 380}}, {{540, 400}, {560, 450}},
		{{150, 250}, {240, 250}}, {{630, 316}, {740, 316}},
	}
	d.Animals = []AnimalSpawn{
		{"dog", Vec{330, 410}, R(300, 384, 130, 70)},
		{"hen", Vec{580, 420}, R(528, 384, 120, 70)},
	}
	d.Slow = jurassicSlow()
	d.Shape = jurassicShape
	d.Walls = append(d.Walls, voidWalls(jurassicShape, d.W, d.H)...)
	d.Walls = append(d.Walls, jpDeckEdges()...)
	resolveOverlaps(d)
	jpPrune(d)
	return d
}

// jpDeckEdges: the solid cells of the sea follow the outline in 8 px steps, which left a strip of water along the
// sides of the pier and the dock where the feet could stand off the planks. A thin invisible edge on every side
// that faces the sea keeps the feet on the deck (the side that touches the beach stays open).
func jpDeckEdges() []Wall {
	const t, step = 6.0, 2.0
	var out []Wall
	run := func(rs []Rect) { // merge the consecutive pieces of one side
		for i := 0; i < len(rs); {
			r := rs[i]
			j := i + 1
			for ; j < len(rs); j++ {
				n := rs[j]
				if r.H == n.H && n.Y == r.Y && n.X == r.X+r.W {
					r.W += n.W
				} else if r.W == n.W && n.X == r.X && n.Y == r.Y+r.H {
					r.H += n.H
				} else {
					break
				}
			}
			out = append(out, Wall{r, WallVoid})
			i = j
		}
	}
	sea := func(x, y float64) bool { return islandSD(x, y) > 0 }
	for _, r := range []Rect{jpPier, jpDock} {
		var top, bot, left, right []Rect
		for x := r.X; x < r.X+r.W; x += step {
			if sea(x+1, r.Y-4) {
				top = append(top, Rect{x, r.Y - t, step, t + 1})
			}
			if sea(x+1, r.Y+r.H+4) {
				bot = append(bot, Rect{x, r.Y + r.H - 1, step, t + 1})
			}
		}
		for y := r.Y; y < r.Y+r.H; y += step {
			if sea(r.X-4, y+1) {
				left = append(left, Rect{r.X - t, y, t + 1, step})
			}
			if sea(r.X+r.W+4, y+1) {
				right = append(right, Rect{r.X + r.W - 1, y, t + 1, step})
			}
		}
		run(top)
		run(bot)
		run(left)
		run(right)
	}
	return out
}

// jpPrune drops small scenery that would cut a path (the big landmarks stay).
func jpPrune(d *LevelDef) {
	keep := map[string]bool{"jp_gate": true, "jp_rotunda": true, "jp_brachio": true, "jp_trike": true, "jp_trex": true, "jp_raptor": true,
		"jp_lab": true, "jp_control": true, "jp_tower": true, "jp_lighthouse": true, "jp_hadro": true, "jp_skel": true, "dj": true}
	for it := 0; it < 40; it++ {
		bad := genBad(d)
		if len(bad) == 0 {
			return
		}
		b := bad[0]
		best, bd := -1, 70.0
		for i, pr := range d.Props {
			if keep[pr.Kind] {
				continue
			}
			if dd := math.Hypot(pr.X-b.X, pr.Y-b.Y); dd < bd {
				best, bd = i, dd
			}
		}
		if best < 0 {
			return
		}
		d.Props = append(d.Props[:best], d.Props[best+1:]...)
	}
}

// jurassicSlow: the thick forest is walkable but slow.
func jurassicSlow() []Rect {
	const cs = 12.0
	var out []Rect
	for y := 20.0; y < 520; y += cs {
		run := -1.0
		for x := 0.0; x <= 960; x += cs {
			jungle := false
			if x < 960 {
				cx, cy := x+cs/2, y+cs/2
				jungle = islandSD(cx, cy) < -16 && !jpInClearing(cx, cy) && !jpInClearing(cx-6, cy) && !jpInClearing(cx+6, cy) &&
					!jpInClearing(cx, cy-6) && !jpInClearing(cx, cy+6)
			}
			if jungle && run < 0 {
				run = x
			}
			if !jungle && run >= 0 {
				out = append(out, R(run, y, x-run, cs))
				run = -1
			}
		}
	}
	return out
}

func jurassicProps() []PropInst {
	P, PV := P, PV
	props := []PropInst{
		// the main gate and its jeeps
		P("jp_gate", 478, 492), PV("jp_jeep", 262, 452, 0), PV("jp_jeep", 736, 460, 1), PV("jp_jeep", 652, 470, 0),
		// visitor centre and its square
		P("jp_rotunda", 470, 298), P("jp_amber", 598, 326), P("jp_warn", 346, 330),
		PV("bench", 640, 442, 0), P("fountain", 558, 338),
		P("dj", 330, 462), P("speaker", 292, 446), P("speaker", 362, 446),
		P("phone", 296, 236), P("phone", 622, 394), P("techphone", 806, 286), P("portapotty", 338, 270), P("portapotty", 360, 270),
		// paddocks
		P("jp_tower", 740, 224), P("jp_tower", 584, 218),
		P("jp_goat", 716, 296),
		// the tunnel and the Rex
		P("jp_warn", 722, 320),
		// laboratory
		PV("jp_lab", 244, 430, 0), P("jp_control", 176, 432), P("jp_dish", 184, 376),
		P("fridge", 268, 350), P("phone", 226, 352),
		// the helipad and the lighthouses
		P("jp_control", 700, 152), P("jp_lighthouse", 872, 300), P("jp_lighthouse", 826, 190),
		// south east: pond, skeleton and a tower
		P("jp_skel", 668, 452), P("jp_tower", 836, 340), P("jp_warn", 674, 404),
		// the pier and the sea
		P("jp_hut", 150, 120), P("jp_hut", 880, 340),
		// forest scenery (only in the forest: the lawns stay open for people and things to use)
		P("palm", 120, 214), P("palm", 340, 104), P("palm", 590, 154), P("palm", 292, 108), P("palm", 850, 252),
		P("palm", 250, 480), P("palm", 760, 480), P("palm", 880, 380),
		P("tree", 392, 222), P("bush", 190, 330), P("bush", 530, 300),
	}
	return props
}

// jpScaleProps shrinks the ordinary objects of the island: with the people at 0.7 and the rex
// enormous, nothing (furniture, booths, speakers, vehicles...) may be nearly as big as the rex.
func jpScaleProps(ps []PropInst) {
	for i := range ps {
		k := propKinds[ps[i].Kind]
		if k == nil {
			continue
		}
		switch ps[i].Kind {
		case "jp_gate":
			ps[i].Scale = 0.9
		case "jp_jeep":
			ps[i].Scale = 0.6
		case "sign", "posters", "jp_warn", "screen": // signs are small: smaller than a person
			ps[i].Scale = 0.45
		case "palm", "tree":
			ps[i].Scale = math.Min(0.7, 76/float64(k.H))
		default:
			if !strings.HasPrefix(ps[i].Kind, "jp_") { // the landmarks are already drawn at their size
				ps[i].Scale = 0.65
			}
		}
	}
}
