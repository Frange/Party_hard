package main

import (
	"image/color"
	"math/rand"
)

// genHighway: a party held in the middle of a six-lane motorway. The crowd
// spills onto the asphalt, traffic never lets up and the cars run over anyone
// standing in their lane (see runOver), so most of the carnage is free.
func genHighway(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "AUTOPISTA", Intro: "Seis carriles, cero precauciones. Cruzar es opcional; sobrevivir, también.",
		W: 960, H: 540, Sky: "none", Police: "car", FastTraffic: true, Highway: true, Day: false,
		Tint:    color.RGBA{10, 6, 24, 60},
		Start:   Vec{480, 118},
		CarFrom: Vec{1040, 285}, CarTo: Vec{700, 285},
		Guests: 46 + rng.Intn(6), CellSize: 16,
		Road:  R(0, 177, 960, 216),
		Lanes: []float64{195, 231, 267, 303, 339, 375},
	}
	d.Floors = []Floor{
		F(0, 20, 960, 157, "grass"),
		F(0, 393, 960, 113, "grass"),
		F(0, 177, 960, 216, "asphalt"),
		F(0, 213, 960, 3, "dash"), F(0, 249, 960, 3, "dash"),
		F(0, 283, 960, 4, "dash"),
		F(0, 321, 960, 3, "dash"), F(0, 357, 960, 3, "dash"),
		F(0, 177, 960, 4, "pave"), F(0, 389, 960, 4, "pave"),
	}
	gaps := [][2]float64{{90, 270}, {390, 570}, {690, 870}}
	d.Walls = walls(
		hw(0, 960, 20, 14, WallFence),
		hw(0, 960, 506, 14, WallFence),
		vw(0, 20, 486, 10, WallFence),
		vw(950, 20, 486, 10, WallFence),
		hw(0, 960, 160, 8, WallRail, gaps...),
		hw(0, 960, 398, 8, WallRail, gaps...),
	)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.at("bridge", 470, 285, 0)
	pl.avoid = append(pl.avoid, Vec{470, 285})
	// north verge: the stage and the bar
	pl.at("dj", 480, 78, 0)
	pl.at("speaker", 410, 76, 0)
	pl.at("speaker", 550, 76, 0)
	d.Performers = []Perf{{Vec{480, 64}, "dj", DirDown}}
	pl.at("bar", 190, 110, 0)
	pl.at("foodtruck", 800, 108, 0)
	pl.at("portapotty", 70, 100, 0)
	pl.at("portapotty", 880, 140, 0)
	pl.at("phone", 330, 120, 0)
	pl.at("phone", 620, 120, 0)
	pl.scatter(R(40, 40, 880, 120), 5, 80, "lamp", "plant", "bench")
	// south verge: a broken-down car, a barbecue and deck chairs
	pl.at("car", 160, 450, 1)
	pl.at("car", 780, 452, 3)
	pl.at("bbq", 470, 440, 0)
	pl.at("fridge", 420, 436, 0)
	pl.at("sunbed", 590, 450, 0)
	pl.at("portapotty", 880, 448, 0)
	pl.scatter(R(40, 400, 880, 100), 5, 80, "lamp", "plant", "barrel", "crate")
	d.Props = props
	d.Roam = []Rect{R(40, 40, 880, 110), R(40, 410, 880, 80), R(60, 190, 840, 190)}
	d.Dance = []Rect{R(380, 40, 200, 80)}
	d.Guards = [][]Vec{{{120, 80}, {800, 80}}, {{120, 470}, {820, 470}}}
	d.Animals = []AnimalSpawn{{"dog", Vec{300, 450}, R(200, 410, 300, 70)}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}
