package main

import (
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// The three parties that come before the space station (campaign.go): BIRTHDAY, HOCKEY RINK and 70s DISCO.
// Same pattern as the other generators: floors, walls, props (genPlacer), roam / dance zones, guards. Everything is
// checked by genNamed (genValidate: start, points of interest, roam, dance and guard points are reachable).
// ---------------------------------------------------------------------------

// genBirthday: a living room (open plan) with the giant cake, and a garden with a patio.
func genBirthday(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "FIESTA DE CUMPLEAÑOS", Intro: "Tarta gigante, globos y un invitado de más: ¡a soplar las velas!",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{40, 14, 36, 46},
		Guests: 44 + rng.Intn(8), CellSize: 16,
	}
	frontX := 250 + math.Round(rng.Float64()*220)
	d.Floors = []Floor{
		F(0, 0, 960, 452, "grass"),
		F(690, 54, 240, 340, "pave"),
		F(70, 46, 590, 342, "parquet"),
		F(frontX-50, 388, 100, 58, "pave"),
	}
	d.Walls = walls(
		hw(0, 960, 0, 14, WallHedge),
		vw(0, 14, 452, 10, WallFence),
		vw(950, 14, 452, 10, WallFence),
		hw(62, 668, 30, 16, WallBrick),
		vw(62, 30, 396, 8, WallBrick),
		vw(660, 30, 396, 8, WallBrick, [2]float64{180, 290}), // the door to the garden
		hw(62, 668, 388, 16, WallBrick, [2]float64{frontX - 48, frontX + 48}),
	)
	genStreet(d, frontX)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	for _, dr := range doorLog {
		cx, cy := dr.R.X+dr.R.W/2, dr.R.Y+dr.R.H/2
		pl.avoid = append(pl.avoid, Vec{cx, cy}, Vec{cx, cy + 34}, Vec{cx, cy - 34}, Vec{cx + 34, cy}, Vec{cx - 34, cy})
	}
	pl.avoid = append(pl.avoid, Vec{frontX, 380}, Vec{frontX, 350})
	// the cake is the centre of the party; two entertainers (decoration: the magic comes in another update)
	pl.at("bdaycake", 400, 226, rng.Intn(3))
	pl.at("entertainer", 480, 306, 1)
	pl.at("entertainer", 300, 330, 0)
	pl.avoid = append(pl.avoid, Vec{400, 200}, Vec{400, 226}, Vec{480, 290}, Vec{300, 314})
	pl.at("giftable", 560, 122, 0)
	pl.at("giftable", 508, 122, 1)
	pl.at("longtable", 190, 106, 0)
	pl.at("punch", 296, 106, rng.Intn(2))
	for i, x := range []float64{190, 370, 550} { // garlands across the room
		pl.at("bunting", x, 72, i%2)
	}
	for _, p := range [][2]float64{{98, 372}, {628, 372}, {96, 150}, {630, 160}, {250, 372}} {
		pl.at("balloons", p[0], p[1], 0)
	}
	pl.scatter(R(70, 46, 590, 342), 6+rng.Intn(3), 84, "couch", "armchair", "plant", "lamp", "bookshelf")
	// garden: cafe tables, a barbecue, trees, a DJ with speakers and the phone
	for i, p := range [][2]float64{{740, 190}, {850, 190}, {796, 330}} {
		pl.at("cafetable", p[0], p[1], i)
	}
	pl.at("bbq", 890, 290, 0)
	pl.at("tree", 712, 384, 0)
	pl.at("tree", 912, 380, 1)
	pl.at("bench", 724, 270, 0)
	pl.at("dj", 800, 118, 0)
	pl.at("speaker", 746, 114, 0)
	pl.at("speaker", 856, 114, 0)
	d.Performers = []Perf{{Vec{800, 100}, "dj", DirDown}}
	pl.at("phone", 924, 372, 0)
	pl.at("balloons", 706, 120, 0)
	pl.at("balloons", 920, 150, 0)
	d.Props = props
	d.Roam = []Rect{R(80, 70, 570, 300), R(680, 40, 250, 360), R(14, 408, 930, 34)}
	d.Dance = []Rect{R(100, 170, 170, 110), R(740, 226, 130, 60)}
	d.Animals = []AnimalSpawn{{"dog", Vec{800, 360}, R(690, 250, 240, 140)}}
	d.Guards = [][]Vec{{{110, 350}, {620, 350}}, {{706, 380}, {900, 360}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// genHockey: an ice rink with boards and two gates on each side, stands, scoreboard, benches and a resurfacer.
// The ice ("rink" floor, slippery: ice.go) is the middle; the concourse around it is ordinary ground.
func genHockey(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "PISTA DE HOCKEY", Intro: "Hielo, palos y una grada que pide pelea. Hoy el disco eres tú.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{8, 16, 40, 70},
		Guests: 44 + rng.Intn(8), CellSize: 16,
	}
	frontX := 440 + math.Round(rng.Float64()*160)
	d.Floors = []Floor{
		F(0, 0, 960, 452, "concrete"),
		F(150, 102, 660, 268, "rink"),
	}
	d.Walls = walls(
		hw(0, 960, 0, 30, WallBrick),
		vw(0, 30, 452, 10, WallFence),
		vw(950, 30, 452, 10, WallFence),
		hw(142, 818, 94, 8, WallBoards),
		vw(142, 94, 378, 8, WallBoards, [2]float64{198, 262}),
		vw(810, 94, 378, 8, WallBoards, [2]float64{198, 262}),
		hw(142, 818, 370, 8, WallBoards, [2]float64{268, 340}, [2]float64{620, 692}),
	)
	genStreet(d, frontX)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	// on the ice: the two nets
	pl.at("hgoal", 172, 249, 0)
	pl.at("hgoal", 788, 249, 1)
	pl.avoid = append(pl.avoid, Vec{172, 236}, Vec{788, 236}, Vec{480, 236}, Vec{440, 236}, Vec{520, 236})
	// top: stands and the scoreboard
	for _, x := range []float64{150, 360, 600, 810} {
		pl.at("stands", x, 72, int(x/10)%2)
	}
	pl.at("scoreboard", 480, 90, rng.Intn(3))
	// left concourse: players' benches; right: food, the resurfacer and the phone
	pl.at("rinkbench", 78, 170, 0)
	pl.at("rinkbench", 78, 320, 0)
	pl.at("stall", 880, 150, 0)
	pl.at("popcorn", 920, 270, 0)
	pl.at("icemachine", 884, 350, rng.Intn(2))
	pl.at("phone", 924, 112, 0)
	// bottom: DJ with speakers, benches and lamps (the middle stays open for the way in)
	pl.at("dj", 220, 432, 0)
	pl.at("speaker", 160, 428, 0)
	pl.at("speaker", 282, 428, 0)
	d.Performers = []Perf{{Vec{220, 414}, "dj", DirDown}}
	pl.at("bench", 400, 432, 0)
	pl.at("bench", 780, 432, 0)
	pl.at("lamp", 36, 400, 0)
	pl.at("lamp", 924, 400, 0)
	pl.avoid = append(pl.avoid, Vec{frontX, 400}, Vec{304, 386}, Vec{656, 386}, Vec{170, 230}, Vec{790, 230})
	d.Props = props
	d.Roam = []Rect{R(158, 110, 644, 252), R(22, 100, 112, 270), R(826, 100, 112, 270), R(20, 384, 920, 54)}
	d.Dance = []Rect{R(400, 190, 160, 90)}
	d.Guards = [][]Vec{{{70, 110}, {70, 360}}, {{190, 410}, {760, 408}}, {{300, 150}, {660, 330}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// discoFloor: the lit tiles of the 70s disco (disco70.go draws them every frame). 36 x 24 tiles of 8 px.
var discoFloor = Rect{336, 128, 288, 192}

// genDisco70: wood-panelled hall with a lit tile dance floor, mirror ball, lava lamps, bead curtains and pod chairs.
func genDisco70(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "DISCO SETENTERA", Intro: "Pantalones de campana, patillas y una pista que se ilumina sola.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{30, 8, 28, 70},
		Guests: 44 + rng.Intn(8), CellSize: 16,
		Glows: []Glow{
			{Vec{150, 150}, color.RGBA{230, 120, 30, 200}, 90, true}, {Vec{810, 150}, color.RGBA{170, 60, 200, 200}, 90, true},
			{Vec{150, 330}, color.RGBA{170, 60, 200, 200}, 90, true}, {Vec{810, 330}, color.RGBA{230, 120, 30, 200}, 90, true},
		},
	}
	frontX := 360 + math.Round(rng.Float64()*240)
	d.Floors = []Floor{
		F(0, 0, 960, 452, "grass"),
		F(78, 46, 804, 350, "parquet"),
		F(frontX-50, 396, 100, 50, "pave"),
		{R: discoFloor, Kind: "disco70"},
	}
	d.Walls = walls(
		hw(0, 960, 0, 14, WallHedge),
		vw(0, 14, 452, 10, WallFence),
		vw(950, 14, 452, 10, WallFence),
		hw(70, 890, 30, 16, WallWood),
		vw(70, 30, 382, 8, WallWood),
		vw(882, 30, 382, 8, WallWood),
		hw(70, 890, 396, 16, WallWood, [2]float64{frontX - 48, frontX + 48}),
	)
	genStreet(d, frontX)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.avoid = append(pl.avoid, Vec{frontX, 380}, Vec{frontX, 350}, Vec{frontX, 420})
	pl.at("dj", 480, 114, 0)
	pl.at("speaker", 408, 110, 0)
	pl.at("speaker", 552, 110, 0)
	d.Performers = []Perf{{Vec{480, 96}, "dj", DirDown}}
	pl.at("mirrorball", 480, 236, rng.Intn(3))
	for i, x := range []float64{150, 280, 680, 810} {
		pl.at("retropanel", x, 90, (i+rng.Intn(3))%3)
	}
	pl.at("beadcurtain", frontX, 412, rng.Intn(3))
	for i, p := range [][2]float64{{112, 118}, {848, 118}, {112, 366}, {848, 366}} {
		pl.at("lavalamp", p[0], p[1], (i+rng.Intn(3))%3)
	}
	for i, p := range [][2]float64{{132, 196}, {132, 276}, {828, 196}, {828, 276}} {
		pl.at("eggchair", p[0], p[1], (i+rng.Intn(3))%3)
	}
	pl.at("bar", 215, 128, rng.Intn(2))
	pl.at("jukebox", 860, 200, 0)
	pl.at("couch", 215, 372, 0)
	pl.at("couch", 745, 372, 0)
	pl.at("punch", 740, 128, rng.Intn(2))
	pl.at("phone", 864, 372, 0)
	pl.scatter(R(78, 46, 804, 350), 5+rng.Intn(3), 90, "plant", "table", "lamp", "speaker")
	d.Props = props
	d.Roam = []Rect{R(90, 70, 240, 310), R(640, 70, 240, 310), R(336, 330, 288, 50), R(336, 128, 288, 192), R(14, 408, 930, 34)}
	d.Dance = []Rect{discoFloor}
	d.Guards = [][]Vec{{{110, 160}, {110, 340}}, {{850, 160}, {850, 340}}, {{360, 355}, {600, 355}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}
