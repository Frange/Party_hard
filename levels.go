package main

import (
	"image/color"
	"sort"
)

// hw builds a horizontal wall from x1 to x2 at y with thickness t, leaving gaps [a,b] (x ranges).
func hw(x1, x2, y, t float64, style int, gaps ...[2]float64) []Wall {
	sort.Slice(gaps, func(i, j int) bool { return gaps[i][0] < gaps[j][0] })
	var out []Wall
	cur := x1
	for _, g := range gaps {
		if g[0] > cur {
			out = append(out, Wall{Rect{cur, y, g[0] - cur, t}, style})
		}
		doorLog = append(doorLog, Door{R: Rect{g[0], y, g[1] - g[0], t}, Horiz: true, Style: style})
		cur = g[1]
	}
	if x2 > cur {
		out = append(out, Wall{Rect{cur, y, x2 - cur, t}, style})
	}
	return out
}

// vw builds a vertical wall from y1 to y2 at x with thickness t, leaving gaps [a,b] (y ranges).
func vw(x, y1, y2, t float64, style int, gaps ...[2]float64) []Wall {
	sort.Slice(gaps, func(i, j int) bool { return gaps[i][0] < gaps[j][0] })
	var out []Wall
	cur := y1
	for _, g := range gaps {
		if g[0] > cur {
			out = append(out, Wall{Rect{x, cur, t, g[0] - cur}, style})
		}
		doorLog = append(doorLog, Door{R: Rect{x, g[0], t, g[1] - g[0]}, Horiz: false, Style: style})
		cur = g[1]
	}
	if y2 > cur {
		out = append(out, Wall{Rect{x, cur, t, y2 - cur}, style})
	}
	return out
}

func walls(parts ...[]Wall) []Wall {
	var out []Wall
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func P(kind string, x, y float64) PropInst         { return PropInst{Kind: kind, X: x, Y: y} }
func PV(kind string, x, y float64, v int) PropInst { return PropInst{Kind: kind, X: x, Y: y, Var: v} }
func R(x, y, w, h float64) Rect                    { return Rect{x, y, w, h} }
func F(x, y, w, h float64, kind string) Floor      { return Floor{R: R(x, y, w, h), Kind: kind} }
func FC(x, y, w, h float64, kind string, col uint32) Floor {
	return Floor{R: R(x, y, w, h), Kind: kind, Color: col}
}

var levelDefs []*LevelDef

// doorLog collects the gaps left by hw/vw so that doors can be drawn in them.
var doorLog []Door

func classicLevels() []*LevelDef {
	return []*LevelDef{level1(), level2(), level3(), level4(), level5()}
}

// extraLevels are generated once from fixed seeds so the menu always offers the same venues.
var extraLevels = []struct {
	kind  string
	seed  int64
	title string
}{
	{"bullring", 11, "PLAZA DE TOROS"},
	{"club", 23, "DISCOTECA"},
	{"pub", 35, "PUB"},
	{"cinema", 47, "CINE"},
	{"stadium", 59, "ESTADIO"},
	{"plaza", 71, "PLAZA EUROPEA"},
	{"paris", 83, "PARIS"},
	{"festival", 95, "FESTIVAL"},
	{"funfair", 107, "PARQUE DE ATRACCIONES"},
	{"market", 119, "SUPERMERCADO"},
	{"mall", 131, "CENTRO COMERCIAL"},
}

func builtinLevels() []*LevelDef {
	out := classicLevels()
	for _, e := range extraLevels {
		if d := genNamed(e.kind, e.seed, e.title); d != nil {
			out = append(out, d)
		}
	}
	return out
}

func init() {
	levelDefs = classicLevels() // the generated ones need the prop catalogue, which is built later
}

// ---------------------------------------------------------------------------
// 1. House party
// ---------------------------------------------------------------------------
func level1() *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "HOUSE PARTY",
		Intro: "Los vecinos no te dejan dormir. Haz que la fiesta termine.",
		W:     960, H: 540, Sky: "none", Police: "car",
		Tint:    color.RGBA{0, 4, 30, 80},
		Start:   Vec{500, 468},
		CarFrom: Vec{1040, 510}, CarTo: Vec{740, 510},
		Guests: 30, CellSize: 16,
	}
	d.Floors = []Floor{
		F(0, 0, 960, 452, "grass"),
		F(128, 46, 164, 96, "tile"),
		F(128, 158, 164, 238, "tile"),
		F(300, 46, 280, 350, "wood"),
		F(588, 46, 244, 174, "wood"),
		F(588, 236, 244, 160, "carpet"),
		FC(330, 110, 200, 180, "rug", 0x6a1e24),
		FC(640, 80, 150, 100, "rug", 0x2a5a30),
		F(440, 412, 100, 34, "pave"),
		F(0, 454, 960, 28, "pave"),
		F(0, 482, 960, 58, "asphalt"),
		F(0, 508, 960, 4, "dash"),
		F(360, 130, 140, 130, "dance"),
	}
	d.Walls = walls(
		hw(120, 840, 30, 16, WallBrick),
		vw(120, 30, 412, 8, WallBrick),
		vw(832, 30, 412, 8, WallBrick),
		hw(120, 840, 396, 16, WallBrick, [2]float64{450, 530}),
		vw(292, 46, 396, 8, WallPlaster, [2]float64{76, 120}, [2]float64{250, 336}),
		hw(128, 300, 142, 16, WallPlaster, [2]float64{170, 230}),
		vw(580, 46, 396, 8, WallPlaster, [2]float64{96, 170}, [2]float64{270, 350}),
		hw(588, 832, 220, 16, WallPlaster, [2]float64{680, 744}),
		hw(0, 960, 0, 14, WallHedge),
		vw(0, 14, 452, 10, WallFence),
		vw(950, 14, 452, 10, WallFence),
		hw(0, 960, 444, 10, WallFence, [2]float64{420, 570}),
	)
	d.Props = []PropInst{
		// bathroom
		P("bathtub", 176, 100), P("toilet", 258, 128), P("plant", 142, 130),
		// kitchen
		PV("fridge", 150, 214, 0), P("counter", 198, 214), P("stove", 252, 214),
		P("table", 190, 310), P("table", 246, 350), P("bar", 206, 380), P("plant", 142, 380),
		// living room
		P("dj", 440, 108), P("speaker", 352, 104), P("speaker", 530, 104),
		PV("couch", 342, 372, 1), PV("couch", 520, 372, 0), P("punch", 430, 376),
		P("phone", 566, 340), P("plant", 314, 150), P("plant", 560, 150), P("tv", 330, 270),
		// lounge
		P("fireplace", 700, 100), PV("couch", 630, 196, 2), PV("couch", 790, 196, 0), P("plant", 600, 120), P("table", 710, 170),
		// bedroom
		P("bed", 790, 384), P("closet", 624, 300), P("closet", 810, 300), P("table", 690, 380), P("plant", 600, 380),
		// gardens
		P("tree", 56, 150), P("bush", 70, 250), P("bbq", 60, 340), P("table", 70, 400), P("lamp", 100, 220),
		P("tree", 900, 130), P("bush", 900, 260), P("dumpster", 890, 340), P("bush", 890, 410), P("lamp", 860, 200),
		P("bush", 160, 430), P("tree", 80, 430), P("lamp", 400, 430), P("lamp", 600, 430), P("bush", 700, 430),
		PV("car", 200, 534, 0), PV("car", 400, 534, 3),
	}
	d.Performers = []Perf{{Vec{440, 90}, "dj", DirDown}}
	d.Animals = []AnimalSpawn{{"dog", Vec{70, 300}, R(14, 40, 100, 400)}, {"dog", Vec{900, 220}, R(842, 40, 100, 400)}, {"cat", Vec{880, 380}, R(842, 40, 100, 400)}}
	d.Roam = []Rect{R(300, 150, 270, 230), R(310, 60, 260, 70), R(136, 170, 150, 215), R(136, 56, 150, 76),
		R(596, 60, 228, 150), R(596, 246, 228, 140), R(14, 40, 100, 400), R(842, 40, 100, 400), R(14, 410, 930, 36), R(300, 456, 400, 22)}
	d.Dance = []Rect{R(370, 142, 120, 100)}
	d.Guards = [][]Vec{{{60, 180}, {60, 360}}, {{900, 180}, {900, 370}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// rooftop template shared by levels 2 and 3
func rooftop(title, intro string, neon bool) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: title, Intro: intro,
		W: 960, H: 540, Sky: "city", Police: "heli",
		Start:  Vec{150, 150},
		Pads:   []Vec{{770, 410}, {150, 440}},
		Guests: 36, CellSize: 28,
	}
	base := "rooftop"
	if neon {
		base = "rooftop2"
		d.Tint = color.RGBA{0, 6, 40, 100}
		d.Guests = 40
		d.Strobe = []uint32{0x30e0f0, 0xc060f0, 0x50f0a0, 0x30e0f0}
	} else {
		d.Tint = color.RGBA{4, 6, 30, 70}
	}
	d.Floors = []Floor{
		F(40, 64, 880, 426, base),
		F(40, 64, 90, 426, "carpet"),
		F(830, 64, 90, 426, "deck"),
		FC(770-52, 410-52, 104, 104, "helipad", 0xe8e8f0),
		FC(150-36, 440-36, 72, 72, "helipad", 0xe8e8f0),
	}
	if neon {
		d.Floors[1] = F(40, 64, 90, 426, "rooftop2")
		d.Floors[2] = F(830, 64, 90, 426, "rooftop2")
	}
	d.Walls = walls(
		hw(30, 930, 50, 14, WallRoof),
		vw(30, 50, 504, 10, WallRoof),
		vw(920, 50, 504, 10, WallRoof),
		hw(30, 930, 490, 14, WallRoof),
		vw(130, 200, 330, 6, WallRoof),
	)
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// ---------------------------------------------------------------------------
// 2. Rooftop party (pool)
// ---------------------------------------------------------------------------
func level2() *LevelDef {
	doorLog = nil
	d := rooftop("ROOFTOP PARTY", "Una fiesta en la azotea, con piscina y vigilantes. Cuidado con los guardias.", false)
	d.Swim = true
	d.Disco = false
	d.Pools = []Rect{R(318, 128, 290, 200)}
	d.Props = []PropInst{
		P("lift", 160, 112), P("ac", 232, 100), P("ac", 280, 100), P("ac", 328, 100), P("liberty", 372, 110),
		P("dj", 480, 108), P("speaker", 420, 104), P("speaker", 540, 104),
		P("bungalow", 790, 130), P("palm", 676, 130), P("plant", 400, 150),
		PV("sofa", 880, 160, 1), PV("sofa", 880, 280, 1), P("table", 770, 240), P("plant", 700, 200),
		PV("buffet", 240, 340, 0), P("plant", 180, 300), P("plant", 300, 390),
		PV("umbrella", 72, 190, 1), PV("umbrella", 72, 270, 2), PV("umbrella", 72, 350, 0),
		P("fusebox", 640, 372), P("palm", 650, 330), P("palm", 300, 150),
		PV("sunbed", 360, 380, 0), PV("sunbed", 430, 380, 1), PV("sunbed", 500, 380, 2), PV("sunbed", 570, 380, 0),
		P("bush", 880, 440), P("bush", 90, 460), P("dumpster", 880, 360), P("lamp", 130, 80),
		P("lamp", 650, 90), P("lamp", 890, 100), P("lamp", 600, 460), P("lamp", 300, 460),
		P("slide", 360, 450),
	}
	d.Performers = []Perf{{Vec{480, 90}, "dj", DirDown}}
	d.Roam = []Rect{R(140, 110, 170, 360), R(620, 110, 200, 160), R(310, 336, 310, 128), R(300, 70, 330, 50), R(620, 280, 200, 120), R(44, 70, 80, 410), R(834, 70, 80, 410)}
	d.Dance = []Rect{R(660, 190, 150, 90)}
	d.Guards = [][]Vec{{{220, 430}, {620, 440}}, {{90, 200}, {90, 400}}, {{860, 200}, {860, 400}}}
	d.Glows = []Glow{{Vec{480, 110}, color.RGBA{60, 30, 90, 160}, 70, true}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// ---------------------------------------------------------------------------
// 3. Glow Neon
// ---------------------------------------------------------------------------
func level3() *LevelDef {
	doorLog = nil
	d := rooftop("GLOW NEON", "Luces de neón y guardias de seguridad. Que no se apague la fiesta... para ellos.", true)
	d.Disco = true
	for i := range d.Floors {
		if d.Floors[i].Kind == "helipad" {
			d.Floors[i].Color = 0x40e0ff
		}
	}
	d.Floors = append(d.Floors, F(330, 140, 280, 252, "dance"))
	d.Props = []PropInst{
		P("lift", 160, 112), P("ac", 232, 100), P("ac", 280, 100), P("ac", 328, 100), P("liberty", 372, 110),
		P("dj", 480, 108), P("speaker", 420, 104), P("speaker", 540, 104),
		PV("bar", 820, 120, 1), PV("stool", 790, 138, 0), PV("stool", 850, 138, 0),
		PV("sofa", 880, 200, 0), PV("sofa", 880, 300, 0), P("table", 780, 250), P("plant", 700, 200),
		PV("buffet", 240, 340, 0), P("plant", 180, 300), P("palm", 300, 400), P("palm", 640, 400),
		PV("umbrella", 72, 190, 1), PV("umbrella", 72, 270, 2),
		P("rope", 360, 148), P("rope", 360, 410), P("rope", 600, 148),
		P("bush", 880, 450), P("bush", 90, 460), P("palm", 90, 350),
		P("lamp", 130, 80), P("lamp", 890, 100),
	}
	d.Performers = []Perf{{Vec{480, 90}, "dj", DirDown}}
	d.Roam = []Rect{R(140, 110, 180, 360), R(630, 110, 190, 360), R(334, 144, 270, 244), R(300, 400, 330, 70), R(44, 70, 80, 410), R(834, 70, 80, 410)}
	d.Dance = []Rect{R(346, 156, 246, 216)}
	d.Guards = [][]Vec{{{250, 440}, {640, 450}}, {{90, 200}, {90, 400}}, {{860, 220}, {860, 420}}}
	d.Glows = []Glow{{Vec{470, 270}, color.RGBA{20, 80, 90, 160}, 130, true}, {Vec{770, 410}, color.RGBA{20, 70, 110, 160}, 70, false}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// ---------------------------------------------------------------------------
// 4. Miami beach concert
// ---------------------------------------------------------------------------
func level4() *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "MIAMI BEACH",
		Intro: "Concierto en la playa. Mucha gente, mucho ruido, muchas oportunidades.",
		W:     960, H: 540, Sky: "sea", Police: "car", Swim: false, Disco: true,
		Tint:    color.RGBA{18, 4, 30, 80},
		Start:   Vec{130, 500},
		CarFrom: Vec{1040, 512}, CarTo: Vec{640, 512},
		Guests: 44, CellSize: 16,
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
	d.Props = []PropInst{
		P("stage", 480, 214), P("speaker", 280, 204), P("speaker", 690, 204),
		P("bungalow", 160, 180), P("palm", 280, 130), P("palm", 80, 300),
		PV("portapotty", 770, 160, 0), PV("portapotty", 802, 160, 1), PV("portapotty", 834, 160, 0), PV("portapotty", 866, 160, 1),
		PV("booth", 790, 300, 0), PV("booth", 790, 420, 1),
		P("lifeguard", 110, 360), PV("surfboard", 200, 340, 0), PV("surfboard", 218, 340, 1), PV("surfboard", 236, 340, 2),
		P("crate", 260, 280), P("crate", 700, 280), P("barrel", 150, 260),
		P("palm", 60, 450), P("palm", 200, 452), P("palm", 340, 452), P("palm", 620, 452), P("palm", 740, 452), P("palm", 900, 452),
		P("lamp", 400, 450), P("lamp", 580, 450), P("sign", 520, 300), P("bush", 440, 330),
		PV("car", 200, 534, 1), PV("motorbike", 880, 530, 0),
	}
	d.Performers = []Perf{
		{Vec{480, 186}, "drum", DirDown}, {Vec{410, 196}, "guitar", DirDown}, {Vec{550, 196}, "guitar", DirDown}, {Vec{480, 204}, "sing", DirDown},
		{Vec{800, 232}, "dj", DirDown},
	}
	d.Props = append(d.Props, P("dj", 800, 250), P("speaker", 746, 246), P("speaker", 854, 246))
	d.Roam = []Rect{R(250, 230, 450, 190), R(20, 230, 220, 190), R(720, 230, 50, 190), R(20, 100, 240, 90), R(300, 100, 100, 50), R(10, 432, 930, 32), R(100, 470, 800, 16)}
	d.Dance = []Rect{R(300, 240, 340, 130)}
	d.Animals = []AnimalSpawn{{"dog", Vec{140, 300}, R(20, 230, 220, 190)}, {"dog", Vec{700, 380}, R(250, 300, 450, 120)}, {"cat", Vec{100, 380}, R(20, 230, 220, 190)}}
	d.Guards = [][]Vec{{{240, 300}, {240, 410}}, {{700, 300}, {700, 410}}, {{60, 220}, {60, 300}}}
	d.Glows = []Glow{
		{Vec{300, 120}, color.RGBA{120, 40, 190, 200}, 80, true}, {Vec{660, 120}, color.RGBA{40, 80, 200, 200}, 80, true},
		{Vec{480, 220}, color.RGBA{90, 30, 110, 130}, 110, true},
	}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// ---------------------------------------------------------------------------
// 5. Bull's Eye ranch
// ---------------------------------------------------------------------------
func level5() *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "BULL'S EYE",
		Intro: "Un rancho perdido y una fiesta con botas. Nadie oirá los gritos.",
		W:     960, H: 540, Sky: "none", Police: "car",
		Tint:    color.RGBA{4, 8, 20, 80},
		Start:   Vec{200, 500},
		CarFrom: Vec{1040, 512}, CarTo: Vec{700, 512},
		Guests: 38, CellSize: 16,
	}
	d.Floors = []Floor{
		F(0, 0, 960, 466, "grass"),
		F(236, 60, 258, 262, "wood"),
		F(508, 60, 214, 130, "wood"),
		F(508, 198, 214, 124, "parquet"),
		FC(280, 120, 140, 110, "rug", 0x30302e),
		F(32, 60, 168, 152, "concrete"),
		F(762, 60, 168, 152, "dirt"),
		F(400, 322, 130, 150, "dirt"),
		F(300, 410, 360, 56, "dirt"),
		F(0, 468, 960, 20, "pave"),
		F(0, 488, 960, 52, "asphalt"),
		F(0, 514, 960, 4, "dash"),
		F(260, 130, 120, 90, "dance"),
	}
	d.Walls = walls(
		hw(230, 730, 60, 16, WallWood),
		vw(230, 60, 330, 8, WallWood),
		vw(722, 60, 330, 8, WallWood),
		hw(230, 730, 322, 16, WallWood, [2]float64{420, 520}),
		vw(500, 76, 322, 8, WallWood, [2]float64{110, 170}, [2]float64{236, 296}),
		hw(508, 722, 190, 14, WallWood, [2]float64{590, 650}),
		hw(24, 208, 60, 14, WallWood),
		vw(24, 60, 220, 8, WallWood),
		vw(200, 60, 220, 8, WallWood),
		hw(24, 208, 212, 14, WallWood, [2]float64{90, 150}),
		hw(754, 938, 60, 14, WallWood),
		vw(754, 60, 220, 8, WallWood),
		vw(930, 60, 220, 8, WallWood),
		hw(754, 938, 212, 14, WallWood, [2]float64{820, 880}),
		hw(0, 960, 462, 8, WallFence, [2]float64{360, 560}),
		vw(0, 0, 462, 8, WallFence),
		vw(952, 0, 462, 8, WallFence),
	)
	d.Props = []PropInst{
		// saloon
		P("bar", 360, 108), P("barrel", 280, 110), P("barrel", 308, 110), PV("bookshelf", 262, 130, 0), P("table", 330, 270), P("table", 410, 290), P("table", 260, 300), P("stool", 330, 288),
		PV("stool", 410, 306, 0), P("speaker", 470, 100),
		P("fireplace", 600, 108), PV("couch", 540, 170, 3), P("table", 660, 160), P("plant", 700, 110),
		PV("bed", 650, 292, 0), P("closet", 540, 260), P("closet", 700, 260), P("plant", 520, 300),
		P("phone", 480, 310),
		// garage
		P("crate", 60, 120), P("crate", 90, 130), P("barrel", 170, 110), PV("car", 110, 190, 3), P("phone", 40, 80),
		// barn
		P("haystack", 800, 120), P("haystack", 890, 120), P("crate", 910, 190),
		// yard
		P("tree", 120, 330), P("bush", 60, 400), P("bush", 160, 420), P("tree", 90, 270),
		P("bush", 840, 260), P("bush", 700, 400), P("bush", 900, 420), P("barrel", 770, 270),
		P("fruitstand", 560, 440), PV("motorbike", 640, 420, 1), P("arch", 460, 452), P("sign", 700, 450),
		P("lamp", 210, 240), P("lamp", 740, 240), P("lamp", 360, 340), P("lamp", 560, 340),
		P("dumpster", 220, 410), P("tree", 640, 340),
	}
	d.Performers = []Perf{{Vec{420, 90}, "guitar", DirDown}, {Vec{300, 96}, "sing", DirDown}, {Vec{445, 152}, "dj", DirDown}}
	d.Props = append(d.Props, P("dj", 445, 170))
	yard := R(730, 240, 200, 190)
	d.Animals = []AnimalSpawn{
		{"horse", Vec{800, 330}, yard}, {"horse", Vec{880, 290}, yard}, {"horse", Vec{840, 160}, R(770, 90, 150, 110)},
		{"hen", Vec{760, 380}, yard}, {"hen", Vec{820, 400}, yard}, {"hen", Vec{900, 370}, yard}, {"hen", Vec{860, 260}, yard},
		{"hen", Vec{60, 300}, R(14, 230, 210, 200)}, {"hen", Vec{120, 380}, R(14, 230, 210, 200)}, {"hen", Vec{180, 300}, R(14, 230, 210, 200)},
		{"chick", Vec{90, 320}, R(14, 230, 210, 200)}, {"chick", Vec{140, 340}, R(14, 230, 210, 200)}, {"chick", Vec{160, 390}, R(14, 230, 210, 200)}, {"cat", Vec{200, 420}, R(14, 230, 210, 200)}, {"bull", Vec{780, 190}, R(770, 90, 150, 110)},
		{"dog", Vec{200, 340}, R(14, 230, 210, 200)},
	}
	d.Roam = []Rect{R(240, 130, 250, 180), R(512, 70, 206, 110), R(512, 202, 206, 112), R(28, 70, 166, 130), R(766, 70, 160, 130),
		R(14, 230, 210, 200), R(236, 335, 460, 120), R(720, 230, 220, 200), R(100, 470, 800, 16)}
	d.Dance = []Rect{R(266, 140, 108, 70)}
	d.Guards = [][]Vec{{{150, 300}, {150, 400}}, {{860, 250}, {860, 400}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}
