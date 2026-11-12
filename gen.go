package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Procedural scenarios: houses with rooms and doors, rooftops with pools and beaches.
// Every result is checked for connectivity: props that block a path are removed and
// layouts that still leave rooms isolated are discarded.
// ---------------------------------------------------------------------------

func generateLevel(seed int64) *LevelDef {
	rng := rand.New(rand.NewSource(seed))
	kind := genKinds[rng.Intn(len(genKinds))]
	for try := 0; try < 24; try++ {
		d := genNamed(kind, seed+int64(try)*104729, "")
		if d != nil {
			d.Title = fmt.Sprintf("%s #%03d", d.Title, seed%1000)
			return d
		}
	}
	return level1()
}

type genPlacer struct {
	rng   *rand.Rand
	used  []Vec
	out   *[]PropInst
	avoid []Vec
}

func (pl *genPlacer) free(x, y, sp float64) bool {
	for _, u := range pl.used {
		if math.Abs(u.X-x) < sp && math.Abs(u.Y-y) < sp*0.7 {
			return false
		}
	}
	for _, u := range pl.avoid {
		if math.Hypot(u.X-x, u.Y-y) < 44 {
			return false
		}
	}
	return true
}

// scatter places up to n props of the given kinds at random spots inside r.
func (pl *genPlacer) scatter(r Rect, n int, sp float64, kinds ...string) {
	for i := 0; i < n; i++ {
		for t := 0; t < 25; t++ {
			x := r.X + 36 + pl.rng.Float64()*math.Max(1, r.W-72)
			y := r.Y + 74 + pl.rng.Float64()*math.Max(1, r.H-104)
			if !pl.free(x, y, sp) {
				continue
			}
			pl.used = append(pl.used, Vec{x, y})
			*pl.out = append(*pl.out, PropInst{Kind: kinds[pl.rng.Intn(len(kinds))], X: math.Round(x), Y: math.Round(y), Var: pl.rng.Intn(4)})
			break
		}
	}
}

func (pl *genPlacer) at(kind string, x, y float64, v int) {
	pl.used = append(pl.used, Vec{x, y})
	*pl.out = append(*pl.out, PropInst{Kind: kind, X: math.Round(x), Y: math.Round(y), Var: v})
}

var genInner = WallPlaster
var genLo, genSpan = 0.38, 0.24

type genRoom struct {
	R    Rect
	Kind string
}

// genDoorGap: a wide, generous doorway. Rooms should feel like you can walk
// around anything in them, not squeeze through a narrow gap in the wall.
const genDoorGap = 120.0

func genSplit(rng *rand.Rand, r Rect, depth int, rooms *[]Rect, ws *[][]Wall) {
	if depth == 0 || (r.W < 360 && r.H < 340) || (depth < 3 && rng.Float64() < 0.15) {
		*rooms = append(*rooms, r)
		return
	}
	vert := r.W > r.H*1.15
	if r.W < 360 {
		vert = false
	}
	if r.H < 340 {
		vert = true
	}
	if vert {
		if r.W < 360 {
			*rooms = append(*rooms, r)
			return
		}
		x := r.X + r.W*(genLo+rng.Float64()*genSpan)
		gy := r.Y + 24 + rng.Float64()*math.Max(1, r.H-genDoorGap-46)
		*ws = append(*ws, vw(x-4, r.Y, r.Y+r.H, 8, genInner, [2]float64{gy, gy + genDoorGap}))
		genSplit(rng, Rect{r.X, r.Y, x - 4 - r.X, r.H}, depth-1, rooms, ws)
		genSplit(rng, Rect{x + 4, r.Y, r.X + r.W - x - 4, r.H}, depth-1, rooms, ws)
		return
	}
	if r.H < 340 {
		*rooms = append(*rooms, r)
		return
	}
	y := r.Y + r.H*(genLo+rng.Float64()*genSpan)
	gx := r.X + 30 + rng.Float64()*math.Max(1, r.W-genDoorGap-60)
	*ws = append(*ws, hw(r.X, r.X+r.W, y-7, 14, genInner, [2]float64{gx, gx + genDoorGap}))
	genSplit(rng, Rect{r.X, r.Y, r.W, y - 7 - r.Y}, depth-1, rooms, ws)
	genSplit(rng, Rect{r.X, y + 7, r.W, r.Y + r.H - y - 7}, depth-1, rooms, ws)
}

func genRoof(rng *rand.Rand) *LevelDef {
	neon := rng.Intn(2) == 0
	d := rooftop("AZOTEA", "Una azotea con vistas, música y gente que no debería estar aquí.", neon)
	d.Guests = 38 + rng.Intn(10)
	if neon {
		d.Disco = true
	}
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.at("lift", 160, 112, 0)
	pl.avoid = append(pl.avoid, Vec{160, 150}, Vec{770, 410}, Vec{150, 440}, Vec{770, 360}, Vec{150, 390})
	for i := 0; i < 3; i++ {
		pl.at("ac", 232+float64(i)*48, 100, 0)
	}
	dx := 400 + rng.Float64()*200
	pl.at("dj", dx, 108, 0)
	pl.at("speaker", dx-62, 104, 0)
	pl.at("speaker", dx+62, 104, 0)
	d.Performers = []Perf{{Vec{dx, 90}, "dj", DirDown}}
	if rng.Float64() < 0.65 {
		d.Swim = true
		pw, ph := 200+rng.Float64()*100, 130+rng.Float64()*70
		px := 300 + rng.Float64()*(560-pw)
		py := 160 + rng.Float64()*(330-ph-40)
		d.Pools = []Rect{R(math.Round(px), math.Round(py), math.Round(pw), math.Round(ph))}
		for i := 0; i < 4; i++ {
			pl.at("sunbed", px+20+float64(i)*(pw-30)/4, py+ph+44, i%3)
		}
		pl.avoid = append(pl.avoid, Vec{px + pw/2, py + ph/2}, Vec{px + 20, py + 20}, Vec{px + pw - 20, py + ph - 20}, Vec{px + pw - 20, py + 20}, Vec{px + 20, py + ph - 20})
		pl.at("slide", px+pw/2, py+ph+44+36, 0)
	} else {
		d.Floors = append(d.Floors, F(320, 150, 300, 240, "dance"))
		d.Dance = []Rect{R(334, 164, 272, 212)}
	}
	pl.at("phone", 880, 400, 0)
	pl.scatter(R(140, 40, 700, 420), 12+rng.Intn(5), 78, "palm", "plant", "umbrella", "sofa", "table", "bush", "lamp", "buffet", "bar", "sunbed", "rope")
	pl.scatter(R(44, 70, 80, 410), 3, 70, "umbrella", "plant")
	d.Props = props
	d.Roam = []Rect{R(140, 110, 690, 370), R(44, 70, 80, 410), R(834, 70, 80, 410)}
	if len(d.Dance) == 0 {
		d.Dance = []Rect{R(620, 190, 150, 90)}
	}
	d.Guards = [][]Vec{{{220, 430}, {620, 440}}, {{90, 200}, {90, 400}}, {{860, 200}, {860, 400}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

func genBeach(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "PLAYA", Intro: "Sol, arena y una fiesta con demasiada gente para tan poca playa.",
		W: 960, H: 540, Sky: "sea", Police: "car", Disco: rng.Intn(2) == 0,
		Tint:    color.RGBA{18, 4, 30, 80},
		Start:   Vec{130, 500},
		CarFrom: Vec{1040, 512}, CarTo: Vec{640, 512},
		Guests: 40 + rng.Intn(10), CellSize: 16,
	}
	d.Floors = []Floor{
		F(0, 90, 960, 340, "sand"),
		F(0, 430, 960, 36, "boardwalk"),
		F(0, 468, 960, 20, "pave"),
		F(0, 488, 960, 52, "asphalt"),
		F(0, 514, 960, 4, "dash"),
	}
	d.Walls = walls(
		hw(0, 960, 464, 8, WallFence, [2]float64{420, 560}),
		vw(0, 90, 464, 8, WallFence),
		vw(952, 90, 464, 8, WallFence),
	)
	addSeaShore(d) // the sea above the sand is solid: nobody walks into the water (water.go)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.avoid = append(pl.avoid, Vec{130, 440}, Vec{490, 450})
	sx := 300 + rng.Float64()*360
	pl.at("stage", sx, 214, 0)
	pl.at("speaker", sx-200, 204, 0)
	pl.at("speaker", sx+210, 204, 0)
	d.Performers = []Perf{{Vec{sx, 186}, "drum", DirDown}, {Vec{sx - 70, 196}, "guitar", DirDown}, {Vec{sx + 70, 196}, "guitar", DirDown}, {Vec{sx, 204}, "sing", DirDown}}
	djx := 860.0
	if sx > 600 {
		djx = 110
	}
	d.Performers = append(d.Performers, Perf{Vec{djx, 252}, "dj", DirDown})
	pl.at("dj", djx, 270, 0)
	pl.at("speaker", djx-52, 266, 0)
	pl.at("speaker", djx+52, 266, 0)
	pl.avoid = append(pl.avoid, Vec{sx, 240}, Vec{sx - 100, 240}, Vec{sx + 100, 240})
	if rng.Float64() < 0.6 {
		d.Swim = true
		px := 40 + rng.Float64()*600
		py := 300 + rng.Float64()*40
		d.Pools = []Rect{R(math.Round(px), math.Round(py), 150+math.Round(rng.Float64()*90), 80+math.Round(rng.Float64()*20))}
		pl.avoid = append(pl.avoid, d.Pools[0].Center(), Vec{d.Pools[0].X + 10, d.Pools[0].Y + 10}, Vec{d.Pools[0].X + d.Pools[0].W - 10, d.Pools[0].Y + d.Pools[0].H - 10})
	}
	pl.at("phone", 900, 420, 0)
	pl.at("lifeguard", 80+rng.Float64()*100, 360, 0)
	for i := 0; i < 3; i++ {
		pl.at("portapotty", 770+float64(i)*32, 160, i%2)
	}
	pl.scatter(R(10, 70, 940, 330), 15+rng.Intn(5), 80, "palm", "umbrella", "sunbed", "surfboard", "crate", "barrel", "booth", "bush", "sign", "bungalow")
	pl.scatter(R(10, 432, 930, 30), 6, 100, "palm", "lamp")
	pl.at("car", 200, 534, 1)
	d.Props = props
	d.Roam = []Rect{R(20, 230, 920, 190), R(20, 100, 920, 120), R(10, 432, 930, 32), R(100, 470, 800, 16)}
	d.Dance = []Rect{R(sx-150, 240, 300, 120)}
	d.Animals = []AnimalSpawn{{"dog", Vec{140, 300}, R(20, 230, 220, 190)}}
	d.Guards = [][]Vec{{{240, 300}, {240, 410}}, {{700, 300}, {700, 410}}, {{60, 220}, {60, 300}}}
	d.Glows = []Glow{{Vec{sx - 180, 120}, color.RGBA{120, 40, 190, 200}, 80, true}, {Vec{sx + 180, 120}, color.RGBA{40, 80, 200, 200}, 80, true}}
	return d
}

// genBad lists the points of interest that cannot be reached from the start.
func genBad(def *LevelDef) []Vec {
	l := buildLevel(def)
	l.extraGrow = 7 // demand wide passages: people must be able to walk around every object
	l.initNav()
	var bad []Vec
	testR := func(p Vec, rad float64) {
		if !l.Free(p) {
			p2 := l.NearestFree(p)
			if dist(p, p2) > rad {
				bad = append(bad, p)
				return
			}
		}
		if l.Path(def.Start, p) == nil {
			bad = append(bad, p)
		}
	}
	test := func(p Vec) { testR(p, 24) }
	for _, ps := range l.POIs {
		for _, p := range ps {
			test(p)
		}
	}
	for _, r := range def.Roam {
		testR(r.Center(), 70)
		testR(Vec{r.X + 4, r.Y + 4}, 30)
	}
	for _, r := range def.Dance {
		testR(r.Center(), 60)
	}
	for _, route := range def.Guards {
		for _, p := range route {
			test(p)
		}
	}
	for i, pr := range l.Props {
		if i >= len(def.Props) {
			break // booths added by buildLevel are placed on checked floor already and cannot be pruned
		}
		if pr.K.Hide || pr.K.Phone || pr.K.Trap != "" || pr.K.Poison {
			test(Vec{pr.X, pr.Y + 14})
		}
	}
	if def.Police == "car" {
		test(Vec{def.CarTo.X, def.CarTo.Y - 24})
	}
	for _, p := range def.Pads {
		test(p)
	}
	return bad
}

// genValidate removes props that block paths and reports whether the level is fully connected.
func genValidate(def *LevelDef) bool {
	for it := 0; it < 60; it++ {
		bad := genBad(def)
		if len(bad) == 0 {
			return true
		}
		b := bad[0]
		best, bd := -1, 70.0
		for i, pr := range def.Props {
			if pr.Kind == "dj" || pr.Kind == "lift" {
				continue
			}
			if d := math.Hypot(pr.X-b.X, pr.Y-b.Y); d < bd {
				best, bd = i, d
			}
		}
		if best < 0 {
			// maybe the point itself is inside a prop footprint: try props anywhere along the way
			return false
		}
		def.Props = append(def.Props[:best], def.Props[best+1:]...)
	}
	return false
}
