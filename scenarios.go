package main

import (
	"image/color"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Three scenarios with a look of their own: a Halloween graveyard, a casino
// (built with the building generator) and a snowy ski resort.
// ---------------------------------------------------------------------------

func init() {
	genKinds = append(genKinds, "graveyard", "casino", "ski")
	genStyles["casino"] = &bldStyle{title: "CASINO", intro: "Luces doradas, fichas por el suelo y una fiesta en la que todos apuestan... su vida.",
		outer: WallClub, inner: WallClub, main: "casinomain", kinds: []string{"gamblers", "vip", "clubbar", "bath", "gamblers"},
		floors: map[string]string{"casinomain": "carpet", "gamblers": "carpet", "vip": "marble", "clubbar": "parquet", "bath": "tile"},
		tint:   color.RGBA{24, 4, 10, 95}, disco: true, guests: 44, depth: 3}
}

// genFurnishCasino fills the casino rooms (called from genFurnish).
func genFurnishCasino(kind string, r Rect, rng *rand.Rand, pl *genPlacer, d *LevelDef, phones *int) (handled, dj bool) {
	cx := r.X + r.W/2
	switch kind {
	case "casinomain":
		dw, dh := min2(r.W-120, 220), min2(r.H-170, 110)
		if dh < 60 {
			dh = 60
		}
		dx, dy := cx-dw/2, r.Y+r.H/2-dh/2+26
		d.Floors = append(d.Floors, F(dx, dy, dw, dh, "dance"))
		d.Dance = append(d.Dance, R(dx+8, dy+8, dw-16, dh-16))
		pl.at("dj", cx, r.Y+62, 0)
		pl.at("speaker", cx-84, r.Y+58, 0)
		pl.at("speaker", cx+84, r.Y+58, 0)
		d.Performers = append(d.Performers, Perf{Vec{cx, r.Y + 44}, "dj", DirDown})
		for x := r.X + 34; x < cx-120; x += 28 {
			pl.at("slot", x, r.Y+54, int(x/28)%2)
		}
		for x := r.X + r.W - 34; x > cx+120; x -= 28 {
			pl.at("slot", x, r.Y+54, int(x/28)%2)
		}
		pl.at("roulette", r.X+84, r.Y+r.H-52, 0)
		pl.at("roulette", r.X+r.W-84, r.Y+r.H-52, 0)
		pl.at("punch", r.X+r.W-60, r.Y+r.H/2, 0)
		pl.at("phone", r.X+30, r.Y+r.H/2, 0)
		*phones++
		d.Glows = append(d.Glows, Glow{Vec{cx, dy + dh/2}, color.RGBA{230, 170, 40, 120}, 120, true})
		return true, true
	case "gamblers":
		pl.at("roulette", cx-50, r.Y+r.H/2+8, 0)
		if r.W > 330 {
			pl.at("roulette", cx+60, r.Y+r.H/2+8, 0)
		}
		for x := r.X + 36; x < r.X+r.W-30; x += 30 {
			pl.at("slot", x, r.Y+54, int(x/30)%2)
		}
		pl.scatter(r, 2, 80, "stool", "plant")
		return true, false
	}
	return false, false
}

func min2(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func genGraveyard(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "CEMENTERIO", Intro: "Noche de brujas en el camposanto: música, niebla y muertos que bailan.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{14, 0, 30, 118},
		Guests: 40 + rng.Intn(8), CellSize: 16,
	}
	frontX := 420 + rng.Float64()*120
	d.Floors = []Floor{
		F(0, 0, 960, 452, "grass"), F(frontX-30, 200, 60, 252, "dirt"), F(70, 318, 820, 40, "dirt"),
		F(330, 84, 300, 80, "cobble"), F(384, 184, 192, 92, "dance"),
	}
	d.Walls = walls(
		hw(0, 960, 0, 14, WallHedge),
		vw(0, 14, 452, 10, WallFence),
		vw(950, 14, 452, 10, WallFence),
		hw(0, 960, 444, 10, WallFence, [2]float64{frontX - 80, frontX + 90}),
	)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.at("crypt", 480, 154, 0)
	pl.at("dj", 480, 198, 0)
	pl.at("speaker", 414, 196, 0)
	pl.at("speaker", 546, 196, 0)
	d.Performers = []Perf{{Vec{480, 182}, "dj", DirDown}}
	pl.avoid = append(pl.avoid, Vec{480, 154}, Vec{480, 220}, Vec{frontX, 300}, Vec{frontX, 400})
	// graves in rows on both sides of the path, a few pumpkins and lanterns
	for _, side := range []Rect{R(50, 60, 280, 250), R(630, 60, 280, 250), R(50, 370, 330, 70), R(580, 370, 330, 70)} {
		for i := 0; i < 8; i++ {
			pl.scatter(side, 1, 44, "tombstone")
		}
	}
	pl.scatter(R(40, 60, 880, 380), 6, 90, "deadtree")
	pl.scatter(R(40, 60, 880, 380), 10, 60, "pumpkin")
	pl.scatter(R(40, 60, 880, 380), 6, 120, "lamp")
	pl.scatter(R(40, 60, 880, 380), 2, 120, "statue", "bench")
	pl.at("phone", 60, 360, 0)
	genStreet(d, frontX)
	d.Props = props
	d.Roam = []Rect{R(40, 60, 880, 380), R(14, 410, 930, 36), R(300, 456, 400, 22)}
	d.Dance = []Rect{R(396, 214, 168, 56)}
	d.Glows = []Glow{
		{Vec{450, 112}, color.RGBA{60, 230, 120, 130}, 80, true}, {Vec{512, 112}, color.RGBA{60, 230, 120, 130}, 80, true},
		{Vec{480, 230}, color.RGBA{150, 60, 220, 110}, 110, true},
	}
	d.Animals = []AnimalSpawn{{"cat", Vec{200, 330}, R(100, 300, 300, 100)}, {"dog", Vec{760, 330}, R(620, 300, 280, 100)}}
	d.Guards = [][]Vec{{{140, 100}, {140, 400}}, {{820, 100}, {820, 400}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

func genSki(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "ESTACIÓN DE ESQUÍ", Intro: "Nieve, un chalet con chimenea y una pista de hielo para hacer el tonto.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{0, 10, 44, 80},
		Guests: 42 + rng.Intn(8), CellSize: 16,
	}
	frontX := 380 + rng.Float64()*200
	d.Floors = []Floor{
		F(0, 0, 960, 452, "snow"), F(560, 236, 320, 140, "white"), F(236, 188, 200, 86, "dance"),
		F(frontX-40, 276, 80, 176, "pave"),
	}
	d.Walls = walls(
		hw(0, 960, 0, 14, WallHedge),
		vw(0, 14, 452, 10, WallFence),
		vw(950, 14, 452, 10, WallFence),
		hw(0, 960, 444, 10, WallFence, [2]float64{frontX - 80, frontX + 90}),
		// the ice rink boards
		hw(560, 880, 232, 6, WallRail, [2]float64{690, 750}),
		hw(560, 880, 374, 6, WallRail, [2]float64{690, 750}),
		vw(556, 232, 380, 6, WallRail),
		vw(878, 232, 380, 6, WallRail),
	)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.at("chalet", 330, 160, 0)
	pl.at("dj", 336, 202, 0)
	pl.at("speaker", 270, 200, 0)
	pl.at("speaker", 402, 200, 0)
	d.Performers = []Perf{{Vec{336, 186}, "dj", DirDown}}
	pl.at("igloo", 120, 180, 0)
	// the rink is a landscape one: the goals stand upright at its two ends (goal line at x=592 and x=846)
	pl.at("goalv", 579, 344, 0)
	pl.at("goalv", 860, 344, 1)
	pl.avoid = append(pl.avoid, Vec{330, 160}, Vec{336, 230}, Vec{120, 180}, Vec{720, 306}, Vec{frontX, 330}, Vec{frontX, 420})
	pl.scatter(R(30, 60, 900, 380), 14, 70, "pine")
	pl.scatter(R(30, 60, 900, 380), 5, 80, "snowman")
	pl.scatter(R(30, 60, 560, 380), 4, 100, "bench", "lamp")
	pl.scatter(R(520, 60, 400, 170), 3, 110, "lamp", "bbq")
	pl.at("phone", 920, 120, 0)
	genStreet(d, frontX)
	d.Props = props
	d.Roam = []Rect{R(40, 60, 880, 380), R(14, 410, 930, 36), R(300, 456, 400, 22)}
	d.Dance = []Rect{R(248, 206, 176, 56)}
	d.Glows = []Glow{
		{Vec{300, 150}, color.RGBA{255, 190, 90, 120}, 90, true}, {Vec{372, 150}, color.RGBA{255, 190, 90, 120}, 90, true},
		{Vec{720, 306}, color.RGBA{120, 190, 255, 70}, 130, false},
	}
	d.Animals = []AnimalSpawn{{"dog", Vec{520, 400}, R(300, 330, 240, 100)}}
	d.Guards = [][]Vec{{{120, 260}, {120, 420}}, {{900, 100}, {900, 200}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}
