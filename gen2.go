package main

import (
	"image/color"
	"math"
	"math/rand"
	"sort"
)

// ---------------------------------------------------------------------------
// Buildings (house, club, pub, cinema, mall, supermarket) and open-air venues
// (bullring, stadium, plazas, festival, funfair). Wide corridors, doors to every room,
// wall furniture only against walls.
// ---------------------------------------------------------------------------

type bldStyle struct {
	title, intro string
	outer, inner int
	main         string
	kinds        []string
	floors       map[string]string
	tint         color.RGBA
	disco        bool
	bx0, bx1     float64
	depth        int
	guests       int
	baseFloor    string
}

var genStyles = map[string]*bldStyle{
	"house": {title: "CASA", intro: "Una casa que nadie ha visto antes y una fiesta que nadie quería.", outer: WallBrick, inner: WallPlaster,
		main: "living", kinds: []string{"kitchen", "bed", "lounge", "bar", "bath", "lounge", "bed", "kitchen"},
		floors: map[string]string{"living": "wood", "kitchen": "tile", "bath": "tile", "bed": "carpet", "lounge": "parquet", "bar": "parquet"},
		tint:   color.RGBA{0, 4, 30, 80}, guests: 34, depth: 3},
	"club": {title: "DISCOTECA", intro: "Luces, humo y bajos que hacen temblar el suelo. Nadie oirá nada.", outer: WallClub, inner: WallClub,
		main: "dance", kinds: []string{"clubbar", "vip", "bath", "vip", "clubbar", "lounge"},
		floors: map[string]string{"dance": "concrete", "clubbar": "concrete", "vip": "carpet", "bath": "tile", "lounge": "carpet"},
		tint:   color.RGBA{10, 0, 40, 110}, disco: true, guests: 42, depth: 3},
	"pub": {title: "PUB", intro: "Un pub de barrio con billar, máquina de música y mucha cerveza.", outer: WallBrick, inner: WallWood,
		main: "pubmain", kinds: []string{"billiards", "kitchen", "lounge", "bath", "billiards", "bar"},
		floors: map[string]string{"pubmain": "parquet", "billiards": "wood", "kitchen": "tile", "bath": "tile", "lounge": "wood", "bar": "parquet"},
		tint:   color.RGBA{20, 8, 0, 80}, guests: 34, depth: 2},
	"cinema": {title: "CINE", intro: "Un cine sin butacas donde la película ya no importa.", outer: WallPlaster, inner: WallPlaster,
		main: "hall", kinds: []string{"lobby", "lobby", "bath", "lobby"},
		floors: map[string]string{"hall": "carpet", "lobby": "marble", "bath": "tile"},
		tint:   color.RGBA{16, 0, 20, 100}, guests: 36, depth: 2},
	"mall": {title: "CENTRO COMERCIAL", intro: "Tiendas, fuentes y una fiesta en horario comercial.", outer: WallPlaster, inner: WallPlaster,
		main: "concourse", kinds: []string{"shop", "shop", "shop", "shop", "food", "shop"},
		floors: map[string]string{"concourse": "mall", "shop": "tile", "food": "mall"},
		tint:   color.RGBA{0, 6, 24, 60}, guests: 46, bx0: 40, bx1: 920, depth: 2},
	"market": {title: "SUPERMERCADO", intro: "Pasillo 7: bebidas. Pasillo 8: problemas.", outer: WallPlaster, inner: WallPlaster,
		main: "aisles", kinds: []string{"storage", "bath"},
		floors: map[string]string{"aisles": "mall", "storage": "concrete", "bath": "tile"},
		tint:   color.RGBA{0, 6, 24, 60}, guests: 44, bx0: 40, bx1: 920, depth: 1},
	"station": {title: "ESTACIÓN ESPACIAL", intro: "El final del viaje: una estación a la deriva, y una fiesta que no debería estar aquí.",
		outer: WallRoof, inner: WallPlaster,
		main: "lounge", kinds: []string{"storage", "bath", "bar", "lounge", "storage"},
		floors: map[string]string{"lounge": "white", "bath": "tile", "bar": "arena", "storage": "concrete"},
		tint:   color.RGBA{2, 2, 24, 120}, guests: 36, depth: 2},
}

func genBuilding(rng *rand.Rand, style string) *LevelDef {
	doorLog = nil
	st := genStyles[style]
	genInner = st.inner
	genLo, genSpan = 0.38, 0.24
	if style == "market" {
		genLo, genSpan = 0.66, 0.1
	}
	d := &LevelDef{
		Title: st.title, Intro: st.intro,
		W: 960, H: 540, Sky: "none", Police: "car", Disco: st.disco,
		Tint:    st.tint,
		CarFrom: Vec{1040, 510}, CarTo: Vec{740, 510},
		Guests: st.guests + rng.Intn(8), CellSize: 16,
	}
	bx0 := 100 + float64(rng.Intn(50))
	bx1 := 800 + float64(rng.Intn(60))
	if st.bx0 > 0 {
		bx0, bx1 = st.bx0, st.bx1
	}
	if bx1-bx0 < 600 {
		bx1 = bx0 + 600
	}
	frontX := bx0 + 140 + rng.Float64()*(bx1-bx0-280)
	d.Floors = []Floor{
		F(0, 0, 960, 452, "grass"),
		F(0, 454, 960, 28, "pave"),
		F(0, 482, 960, 58, "asphalt"),
		F(0, 508, 960, 4, "dash"),
		F(frontX-50, 400, 100, 46, "pave"),
	}
	d.Start = Vec{frontX, 470}
	var rooms []Rect
	var ws [][]Wall
	inner := Rect{bx0 + 8, 46, bx1 - bx0 - 16, 350}
	minRooms := map[string]int{"house": 3, "club": 3, "pub": 3, "cinema": 3, "mall": 3, "market": 2, "casino": 3}[style]
	for try := 0; try < 30; try++ {
		doorLog = nil
		rooms, ws = nil, nil
		genSplit(rng, inner, st.depth, &rooms, &ws)
		if len(rooms) >= minRooms {
			break
		}
	}
	// Buildings usually only had a front door; now a side or rear one shows
	// up sometimes too, reusing the same hw/vw gap mechanism as the front
	// door — no new concept needed.
	var lGap, rGap, rearGap [][2]float64
	if rng.Float64() < 0.35 {
		y0 := 100 + rng.Float64()*180
		lGap = [][2]float64{{y0, y0 + 100}}
	}
	if rng.Float64() < 0.35 {
		y0 := 100 + rng.Float64()*180
		rGap = [][2]float64{{y0, y0 + 100}}
	}
	if rng.Float64() < 0.25 && bx1-bx0 > 280 {
		x0 := bx0 + 90 + rng.Float64()*(bx1-bx0-280)
		rearGap = [][2]float64{{x0, x0 + 100}}
	}
	if lGap == nil && rGap == nil && rearGap == nil {
		// never a building with only the front door (a trap with the police at it, v1.4.0): a side door on the side
		// farther from the front one (no dice: the rest of the party comes out as before)
		if frontX < (bx0+bx1)/2 {
			rGap = [][2]float64{{170, 270}}
		} else {
			lGap = [][2]float64{{170, 270}}
		}
	}
	ws = append(ws,
		hw(bx0, bx1, 30, 16, st.outer, rearGap...),
		vw(bx0, 30, 412, 8, st.outer, lGap...),
		vw(bx1-8, 30, 412, 8, st.outer, rGap...),
		hw(bx0, bx1, 396, 16, st.outer, [2]float64{frontX - 48, frontX + 48}),
		hw(0, 960, 0, 14, WallHedge),
		vw(0, 14, 452, 10, WallFence),
		vw(950, 14, 452, 10, WallFence),
		hw(0, 960, 444, 10, WallFence, [2]float64{frontX - 80, frontX + 90}),
	)
	d.Walls = walls(ws...)
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].W*rooms[i].H > rooms[j].W*rooms[j].H })
	kinds := append([]string{}, st.kinds...)
	rng.Shuffle(len(kinds), func(i, j int) { kinds[i], kinds[j] = kinds[j], kinds[i] })
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	for _, dr := range doorLog {
		cx, cy := dr.R.X+dr.R.W/2, dr.R.Y+dr.R.H/2
		pl.avoid = append(pl.avoid, Vec{cx, cy}, Vec{cx, cy + 34}, Vec{cx, cy - 34}, Vec{cx + 34, cy}, Vec{cx - 34, cy})
	}
	pl.avoid = append(pl.avoid, Vec{frontX, 380}, Vec{frontX, 350})
	phones, hasDJ := 0, false
	for i, r := range rooms {
		kind := st.main
		if i > 0 {
			kind = kinds[(i-1)%len(kinds)]
			if kind == "bath" && r.W*r.H > 30000 {
				kind = st.kinds[0]
			}
		}
		fl := st.floors[kind]
		if fl == "" {
			fl = "tile"
		}
		d.Floors = append(d.Genesis(), F(r.X, r.Y, r.W, r.H, fl))
		_ = fl
		ri := r.Grow(-8)
		d.Roam = append(d.Roam, R(ri.X, ri.Y+30, ri.W, math.Max(20, ri.H-30)))
		if genFurnish(kind, r, rng, pl, d, &phones) {
			hasDJ = true
		}
	}
	if !hasDJ {
		r := rooms[0]
		cx := r.X + r.W/2
		pl.at("dj", cx, r.Y+62, 0)
		d.Performers = append(d.Performers, Perf{Vec{cx, r.Y + 44}, "dj", DirDown})
	}
	if phones == 0 {
		pl.scatter(rooms[0], 1, 40, "phone")
	}
	left := R(14, 40, bx0-24, 360)
	right := R(bx1+10, 40, 938-bx1-10, 360)
	for _, g := range []Rect{left, right} {
		if g.W < 36 {
			continue
		}
		d.Roam = append(d.Roam, R(g.X, g.Y, g.W, g.H))
		switch style {
		case "pub":
			pl.scatter(g, 2+rng.Intn(2), 80, "cafetable", "bench", "tree")
		case "club":
			pl.scatter(g, 2, 80, "bench", "lamp", "bush")
		case "casino":
			pl.scatter(g, 3, 80, "palm", "lamp", "planter", "bench")
		default:
			pl.scatter(g, 3+rng.Intn(3), 76, "tree", "bush", "lamp", "bbq", "table", "dumpster", "plant", "bush")
		}
	}
	pl.scatter(R(14, 400, 930, 30), 4, 140, "bush", "lamp", "tree")
	pl.at("car", 160+rng.Float64()*200, 534, 0)
	pl.at("car", 600+rng.Float64()*200, 534, rng.Intn(4))
	d.Props = props
	d.Roam = append(d.Roam, R(14, 410, 930, 36), R(300, 456, 400, 22))
	if left.W > 36 {
		d.Guards = append(d.Guards, []Vec{{left.X + left.W/2, 180}, {left.X + left.W/2, 360}})
		d.Animals = append(d.Animals, AnimalSpawn{[]string{"dog", "cat", "dog"}[rng.Intn(3)], Vec{left.X + left.W/2, 300}, left})
	}
	if right.W > 36 {
		d.Guards = append(d.Guards, []Vec{{right.X + right.W/2, 180}, {right.X + right.W/2, 370}})
		if rng.Intn(2) == 0 {
			d.Animals = append(d.Animals, AnimalSpawn{"cat", Vec{right.X + right.W/2, 260}, right})
		}
	}
	if len(d.Guards) == 0 {
		d.Guards = [][]Vec{{{frontX - 200, 430}, {frontX + 200, 430}}}
		inside := rooms[0].Grow(-40)
		d.Guards = append(d.Guards, []Vec{{inside.X + 20, inside.Y + inside.H - 20}, {inside.X + inside.W - 20, inside.Y + inside.H - 20}})
	}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// Genesis returns the floors so far (helper kept tiny so the append above reads naturally).
func (d *LevelDef) Genesis() []Floor { return d.Floors }

// genFurnish fills one room according to its purpose. It returns true when it placed a DJ.
func genFurnish(kind string, r Rect, rng *rand.Rand, pl *genPlacer, d *LevelDef, phones *int) bool {
	cx := r.X + r.W/2
	dj := false
	if ok, hasDJ := genFurnishCasino(kind, r, rng, pl, d, phones); ok {
		return hasDJ
	}
	switch kind {
	case "living":
		dw, dh := math.Min(r.W-80, 200), math.Min(r.H-150, 130)
		if dh < 60 {
			dh = 60
		}
		dx, dy := cx-dw/2, r.Y+r.H/2-dh/2+22
		d.Floors = append(d.Floors, F(dx, dy, dw, dh, "dance"))
		d.Dance = append(d.Dance, R(dx+8, dy+8, dw-16, dh-16))
		pl.at("dj", cx, r.Y+62, 0)
		pl.at("speaker", cx-90, r.Y+58, 0)
		pl.at("speaker", cx+90, r.Y+58, 0)
		d.Performers = append(d.Performers, Perf{Vec{cx, r.Y + 44}, "dj", DirDown})
		dj = true
		pl.at("punch", cx+60, math.Min(dy+dh+24, r.Y+r.H-26), 0) // by the dance floor, not in a corner (v1.4.0)
		pl.at("couch", r.X+70, r.Y+r.H-28, 1)
		pl.at("phone", r.X+r.W-28, r.Y+r.H/2, 0)
		*phones++
		pl.scatter(r, 2, 80, "plant", "tv")
	case "dance":
		dw, dh := math.Min(r.W-70, 260), math.Min(r.H-140, 170)
		dx, dy := cx-dw/2, r.Y+r.H/2-dh/2+20
		d.Floors = append(d.Floors, F(dx, dy, dw, dh, "dance"))
		d.Dance = append(d.Dance, R(dx+8, dy+8, dw-16, dh-16))
		pl.at("dj", cx, r.Y+62, 1)
		pl.at("speaker", cx-100, r.Y+58, 0)
		pl.at("speaker", cx+100, r.Y+58, 0)
		pl.at("truss", cx-60, r.Y+36, 0)
		d.Performers = append(d.Performers, Perf{Vec{cx, r.Y + 44}, "dj", DirDown})
		dj = true
		pl.at("cage", r.X+40, r.Y+r.H/2, 0)
		pl.at("cage", r.X+r.W-40, r.Y+r.H/2, 0)
		d.Glows = append(d.Glows, Glow{Vec{cx, r.Y + r.H/2}, color.RGBA{120, 30, 160, 150}, 130, true})
	case "clubbar":
		pl.at("bar", cx, r.Y+58, 1)
		pl.scatter(r, 3, 80, "stool", "table")
		pl.at("phone", r.X+r.W-26, r.Y+r.H-30, 0)
		*phones++
	case "vip":
		pl.at("couch", r.X+60, r.Y+60, 2)
		pl.at("couch", r.X+r.W-60, r.Y+60, 2)
		pl.scatter(r, 2, 90, "table", "plant")
	case "pubmain":
		pl.at("bar", cx, r.Y+58, 0)
		pl.at("jukebox", r.X+34, r.Y+60, 0)
		pl.at("speaker", r.X+r.W-34, r.Y+60, 0)
		pl.at("dj", r.X+r.W-90, r.Y+60, 0)
		d.Performers = append(d.Performers, Perf{Vec{r.X + r.W - 90, r.Y + 42}, "dj", DirDown})
		dj = true
		d.Dance = append(d.Dance, R(cx-60, r.Y+r.H/2, 120, 60))
		pl.scatter(r, 4, 80, "table", "stool", "table")
		pl.at("phone", r.X+30, r.Y+r.H-30, 0)
		*phones++
	case "billiards":
		pl.at("pooltable", cx-50, r.Y+r.H/2+10, 0)
		if r.W > 330 {
			pl.at("pooltable", cx+60, r.Y+r.H/2+10, 0)
		}
		pl.scatter(r, 2, 80, "stool", "plant")
	case "hall":
		pl.at("screen", cx, r.Y+80, rng.Intn(3))
		pl.at("dj", cx+90, r.Y+130, 0)
		pl.at("speaker", cx+30, r.Y+128, 0)
		pl.at("speaker", cx+150, r.Y+128, 0)
		d.Performers = append(d.Performers, Perf{Vec{cx + 90, r.Y + 112}, "dj", DirDown})
		dj = true
		d.Dance = append(d.Dance, R(cx-130, r.Y+r.H/2-10, 200, 90))
		pl.at("popcorn", r.X+36, r.Y+r.H-28, 0)
		pl.at("popcorn", r.X+r.W-36, r.Y+r.H-28, 0)
		pl.at("posters", r.X+r.W/2-80, r.Y+r.H-14, 0)
	case "lobby":
		pl.at("ticketbooth", r.X+44, r.Y+66, 0)
		pl.at("popcorn", r.X+r.W-40, r.Y+62, 0)
		pl.at("posters", cx, r.Y+44, rng.Intn(3))
		pl.scatter(r, 2, 80, "couch", "plant")
		pl.at("phone", r.X+30, r.Y+r.H-30, 0)
		*phones++
	case "concourse":
		pl.at("fountain", cx, r.Y+r.H/2+30, 0)
		pl.at("dj", r.X+70, r.Y+62, 0)
		pl.at("speaker", r.X+20+28, r.Y+60, 0)
		pl.at("speaker", r.X+120, r.Y+60, 0)
		d.Performers = append(d.Performers, Perf{Vec{r.X + 70, r.Y + 44}, "dj", DirDown})
		dj = true
		d.Dance = append(d.Dance, R(r.X+30, r.Y+84, 150, 70))
		for i := 0; i < 3; i++ {
			pl.at("planter", r.X+r.W-70, r.Y+80+float64(i)*(r.H-130)/2, i)
		}
		pl.scatter(r, 3, 90, "bench", "plant", "planter")
		pl.at("phone", r.X+r.W-26, r.Y+r.H-26, 0)
		*phones++
	case "shop":
		for k := 0; k < 2 && r.W > 150; k++ {
			pl.at("shelf", r.X+r.W/2, r.Y+70+float64(k)*80, k)
		}
		pl.at("counter", cx, r.Y+46, 0)
		pl.scatter(r, 2+int(r.W*r.H/14000), 60, "plant", "cart", "shelf", "bench", "stool")
	case "food":
		pl.scatter(r, 4, 60, "table", "stool", "plant")
		pl.at("counter", cx, r.Y+56, 0)
		pl.at("fridge", cx-60, r.Y+56, 0)
	case "aisles":
		// rows of shelves with wide aisles; checkouts and freezers against the walls
		for x := r.X + 70; x < r.X+r.W-70; x += 104 {
			pl.at("freezer", x, r.Y+56, 0)
		}
		rowsY := []float64{r.Y + 150, r.Y + 232}
		for ri, y := range rowsY {
			if y > r.Y+r.H-80 {
				continue
			}
			for x := r.X + 120; x < r.X+r.W-150; x += 134 {
				pl.at("shelf", x, y, (ri+int(x))%4)
			}
		}
		for i := 0; i < 3; i++ {
			pl.at("checkout", r.X+r.W-50-float64(i)*84, r.Y+r.H-24, 0)
		}
		pl.at("cart", r.X+50, r.Y+r.H-30, 0)
		pl.at("cart", r.X+80, r.Y+r.H-30, 0)
		pl.at("dj", r.X+70, r.Y+r.H-60, 0)
		pl.at("speaker", r.X+34, r.Y+r.H-62, 0)
		pl.at("speaker", r.X+110, r.Y+r.H-62, 0)
		d.Performers = append(d.Performers, Perf{Vec{r.X + 70, r.Y + r.H - 78}, "dj", DirDown})
		dj = true
		d.Dance = append(d.Dance, R(r.X+170, r.Y+r.H-100, 220, 60))
		pl.at("phone", r.X+r.W-26, r.Y+r.H/2, 0)
		*phones++
	case "storage":
		pl.scatter(r, 3, 80, "crate", "barrel", "crate")
	case "kitchen":
		pl.at("fridge", r.X+30, r.Y+56, 0)
		pl.at("counter", r.X+80, r.Y+56, 0)
		pl.at("stove", r.X+130, r.Y+56, 0)
		pl.scatter(r, 1, 80, "table")
	case "bath":
		pl.at("bathtub", r.X+r.W-54, r.Y+56, 0)
		pl.at("toilet", r.X+28, r.Y+48, 0)
	case "bed":
		pl.at("bed", r.X+r.W-56, r.Y+r.H-36, 0)
		pl.at("closet", r.X+30, r.Y+56, 0)
		pl.at("closet", r.X+78, r.Y+56, 0)
		pl.scatter(r, 1, 80, "plant", "table")
		if *phones < 2 {
			pl.at("phone", r.X+22, r.Y+r.H-30, 0)
			*phones++
		}
	case "lounge":
		pl.at("fireplace", cx, r.Y+56, 0)
		pl.scatter(r, 2, 90, "couch", "sofa")
		pl.scatter(r, 1, 80, "table", "plant")
	case "bar":
		pl.at("bar", cx, r.Y+58, 0)
		pl.scatter(r, 2, 80, "stool", "table", "plant")
	}
	return dj
}

// ---------------------------------------------------------------------------
// open-air venues
// ---------------------------------------------------------------------------

func genStreet(d *LevelDef, frontX float64) {
	d.Floors = append(d.Floors,
		F(0, 454, 960, 28, "pave"),
		F(0, 482, 960, 58, "asphalt"),
		F(0, 508, 960, 4, "dash"))
	d.Walls = append(d.Walls, hw(0, 960, 444, 10, WallFence, [2]float64{frontX - 80, frontX + 90})...)
	d.Start = Vec{frontX, 470}
	d.CarFrom, d.CarTo = Vec{1040, 510}, Vec{740, 510}
}

func genPlaza(rng *rand.Rand, eiffel bool) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "PLAZA EUROPEA", Intro: "Una plaza de cuento con fuente, terrazas y gente que sobra.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{10, 6, 30, 70},
		Guests: 44 + rng.Intn(8), CellSize: 16,
	}
	if eiffel {
		d.Title, d.Intro = parisTitle, "Bajo la torre Eiffel, la ciudad del amor y del caos."
	}
	frontX := 380 + rng.Float64()*200
	d.Floors = []Floor{F(0, 0, 960, 452, "cobble")}
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	// building facades along the top, with doors that are only decoration
	var ws [][]Wall
	ws = append(ws, hw(0, 960, 20, 78, WallBrick, [2]float64{150, 214}, [2]float64{430, 494}, [2]float64{700, 764}))
	ws = append(ws, vw(0, 98, 452, 10, WallFence), vw(950, 98, 452, 10, WallFence))
	d.Walls = walls(ws...)
	pl.avoid = append(pl.avoid, Vec{frontX, 420})
	cx, cy := 480.0, 280.0
	if eiffel {
		d.Floors = append(d.Floors, F(40, 110, 880, 330, "grass"), F(0, 110, 40, 330, "cobble"), F(420, 110, 120, 330, "cobble"), F(40, 300, 880, 40, "cobble"))
		pl.at("eiffel", cx, eiffelY, 0) // low enough that the top stays clear of the title and the intro plate
		// PARÍS (paris.go): the painted city and the shopfronts replace the brick facade, and the city's props
		doorLog = nil // the doors of the facade above are logged again by the shopfronts
		d.Walls = walls(append(parisDress(d, pl, rng), vw(0, 98, 452, 10, WallFence), vw(950, 98, 452, 10, WallFence))...)
		pl.avoid = append(pl.avoid, Vec{cx, eiffelY - 12}, Vec{cx, eiffelY - 62}, Vec{cx - 40, eiffelY - 4}, Vec{cx + 40, eiffelY - 4})
		for _, x := range []float64{150, 270, 690, 810} {
			pl.at("hedge", x, 170, 0)
			pl.at("hedge", x, 410, 0)
		}
		for _, x := range []float64{110, 850} {
			pl.at("planter", x, 250, rng.Intn(3))
			pl.at("planter", x, 330, rng.Intn(3))
		}
		pl.at("bench", 360, 330, 0)
		pl.at("bench", 600, 330, 0)
	} else {
		d.Floors = append(d.Floors, F(300, 150, 360, 250, "marble"))
		pl.at("fountain", cx, cy, 0)
		pl.avoid = append(pl.avoid, Vec{cx, cy}, Vec{cx - 45, cy}, Vec{cx + 45, cy}, Vec{cx, cy - 20})
		for _, a := range []float64{-1, 1} {
			pl.at("bench", cx+a*130, cy+10, 0)
			pl.at("bench", cx+a*130, cy-70, 0)
		}
		pl.at("statue", 130, 200, rng.Intn(2))
		pl.at("statue", 830, 200, rng.Intn(2))
		pl.at("obelisk", 860, 380, 0)
		pl.scatter(R(40, 110, 220, 330), 2, 90, "planter", "tree")
		pl.scatter(R(700, 110, 220, 330), 2, 90, "planter", "tree")
	}
	// café terraces
	for i := 0; i < 3; i++ {
		pl.at("cafetable", 190+float64(i)*66, 150, i)
	}
	for i := 0; i < 3; i++ {
		pl.at("cafetable", 760-float64(i)*66, 150, (i+1)%3)
	}
	pl.at("dj", 110, 360, 0)
	pl.at("speaker", 62, 356, 0)
	pl.at("speaker", 160, 356, 0)
	d.Performers = []Perf{{Vec{110, 342}, "dj", DirDown}, {Vec{840, 330}, "guitar", DirDown}}
	lamp := "lamp"
	if eiffel {
		lamp = "par_lamp" // the period street lamps of PARÍS (paris_props.go)
	}
	pl.at(lamp, 330, 200, 0)
	pl.at(lamp, 630, 200, 0)
	pl.at(lamp, 330, 400, 0)
	pl.at(lamp, 630, 400, 0)
	pl.at("phone", 920, 300, 0)
	pl.scatter(R(30, 360, 900, 80), 3, 120, "bush", "tree", "planter")
	pl.at("car", 200+rng.Float64()*100, 534, 1)
	genStreet(d, frontX)
	d.Props = props
	d.Roam = []Rect{R(30, 110, 900, 330), R(14, 410, 930, 36), R(300, 456, 400, 22)}
	if eiffel {
		d.Roam = parisRoam(d.Roam) // the lawn kept for the sack race (paris.go)
	}
	d.Dance = []Rect{R(60, 380, 120, 50)}
	d.Animals = []AnimalSpawn{{"dog", Vec{200, 330}, R(30, 250, 250, 180)}, {"cat", Vec{850, 300}, R(700, 250, 230, 180)}}
	if rng.Intn(2) == 0 {
		d.Animals = append(d.Animals, AnimalSpawn{"hen", Vec{700, 380}, R(600, 340, 300, 90)})
	}
	d.Guards = [][]Vec{{{140, 300}, {140, 420}}, {{820, 300}, {820, 420}}, {{frontX - 160, 430}, {frontX + 160, 430}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

func genBullring(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "PLAZA DE TOROS", Intro: "Arena, gradas y un toro que no ha pagado entrada.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{24, 8, 0, 70}, Disco: false,
		Guests: 42 + rng.Intn(8), CellSize: 16,
	}
	frontX := 440 + rng.Float64()*120
	cx, cy, R0 := 480.0, 232.0, 150.0
	d.Floors = []Floor{F(0, 0, 960, 452, "pave")}
	// round sand arena built from horizontal slices
	for y := cy - R0; y < cy+R0; y += 6 {
		dy := y + 3 - cy
		w := math.Sqrt(math.Max(0, R0*R0-dy*dy))
		d.Floors = append(d.Floors, F(cx-w, y, 2*w, 6, "arena"))
	}
	var wl []Wall
	gates := []float64{math.Pi / 2, -math.Pi / 2, 0, math.Pi}
	for a := 0.0; a < 2*math.Pi; a += 2 * math.Pi / 72 {
		skip := false
		for _, g := range gates {
			da := math.Abs(math.Remainder(a-g, 2*math.Pi))
			if da < 0.38 {
				skip = true
			}
		}
		if skip {
			continue
		}
		wl = append(wl, Wall{R(cx+math.Cos(a)*R0-5, cy+math.Sin(a)*R0-5, 11, 11), WallWood})
	}
	d.Walls = wl
	d.Walls = append(d.Walls, hw(0, 960, 0, 14, WallBrick)...)
	d.Walls = append(d.Walls, vw(0, 14, 452, 10, WallFence)...)
	d.Walls = append(d.Walls, vw(950, 14, 452, 10, WallFence)...)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.avoid = append(pl.avoid, Vec{frontX, 420})
	for _, g := range gates {
		pl.avoid = append(pl.avoid, Vec{cx + math.Cos(g)*(R0+30), cy + math.Sin(g)*(R0+30)}, Vec{cx + math.Cos(g)*(R0-30), cy + math.Sin(g)*(R0-30)})
	}
	pl.at("stands", 270, 56, 0)
	pl.at("stands", 690, 56, 1)
	pl.at("stands", 270, 446, 1)
	pl.at("stands", 690, 446, 0)
	pl.at("dj", cx, cy-R0+64, 0)
	pl.at("speaker", cx-60, cy-R0+60, 0)
	pl.at("speaker", cx+60, cy-R0+60, 0)
	d.Performers = []Perf{{Vec{cx, cy - R0 + 46}, "dj", DirDown}}
	pl.at("phone", 60, 300, 0)
	pl.at("phone", 900, 300, 0)
	pl.scatter(R(20, 80, 160, 320), 3, 100, "lamp", "bench", "tree", "balloons")
	pl.scatter(R(780, 80, 160, 320), 3, 100, "lamp", "bench", "tree", "cottoncandy")
	pl.at("car", 180, 534, 2)
	genStreet(d, frontX)
	d.Props = props
	d.Roam = []Rect{R(cx-R0+30, cy-R0+90, 2*R0-60, 2*R0-130), R(20, 70, 160, 360), R(780, 70, 160, 360), R(14, 410, 930, 36), R(300, 456, 400, 22), R(180, 80, 120, 60), R(660, 80, 120, 60)}
	d.Dance = []Rect{R(cx-80, cy-30, 160, 90)}
	d.Animals = []AnimalSpawn{{"bull", Vec{cx + 40, cy + 20}, R(cx-100, cy-60, 200, 140)}}
	if rng.Intn(2) == 0 {
		d.Animals = append(d.Animals, AnimalSpawn{"horse", Vec{cx - 70, cy + 70}, R(cx-110, cy+10, 220, 100)})
	}
	d.Guards = [][]Vec{{{70, 130}, {70, 400}}, {{890, 130}, {890, 400}}, {{frontX - 160, 430}, {frontX + 160, 430}}}
	d.Glows = []Glow{{Vec{cx, cy}, color.RGBA{90, 50, 10, 110}, 140, true}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

func genStadium(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "ESTADIO", Intro: "Un campo de fútbol pequeño con grada, bengalas y cero árbitros.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{0, 10, 24, 60}, Disco: false,
		Guests: 44 + rng.Intn(8), CellSize: 16,
	}
	frontX := 430 + rng.Float64()*140
	d.Floors = []Floor{F(0, 0, 960, 452, "pave"), F(70, 96, 820, 300, "pitch")}
	line := func(x, y, w, h float64) { d.Floors = append(d.Floors, F(x, y, w, h, "chalk")) }
	line(90, 112, 780, 2)
	line(90, 380, 780, 2)
	line(90, 112, 2, 270)
	line(868, 112, 2, 270)
	line(478, 112, 2, 270)
	line(90, 190, 90, 2)
	line(90, 304, 90, 2)
	line(178, 190, 2, 116)
	line(780, 190, 90, 2)
	line(780, 304, 90, 2)
	line(778, 190, 2, 116)
	for _, a := range []float64{0, 0.5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5, 5.5, 6} { // centre circle
		line(479+math.Cos(a)*42-1, 246+math.Sin(a)*42-1, 3, 3)
	}
	// the pitch is open (no fences round it, v1.4.0): only the outer walls of the ground
	d.Walls = walls(hw(0, 960, 0, 14, WallBrick), vw(0, 14, 452, 10, WallFence), vw(950, 14, 452, 10, WallFence))
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.avoid = append(pl.avoid, Vec{frontX, 420}, Vec{frontX, 380})
	pl.at("stands", 200, 66, 0)
	pl.at("stands", 440, 66, 1)
	pl.at("stands", 680, 66, 0)
	// (no stands at the bottom any more: they hid the pitch from the player)
	pl.at("goalv", 74, 284, 0)  // goal line at x=90, net behind it
	pl.at("goalv", 886, 284, 1) // goal line at x=870
	// the DJ plays ON the pitch, at the top of the centre circle, facing the crowd (it was hidden on a side, v1.4.0)
	pl.at("dj", 480, 178, 0)
	pl.at("speaker", 428, 176, 0)
	pl.at("speaker", 532, 176, 0)
	d.Performers = []Perf{{Vec{480, 160}, "dj", DirDown}}
	pl.at("phone", 30, 300, 0)
	pl.at("car", 640, 534, 0)
	genStreet(d, frontX)
	d.Props = props
	d.Roam = []Rect{R(100, 120, 760, 260), R(14, 100, 74, 320), R(876, 100, 70, 320), R(14, 380, 930, 66), R(300, 456, 400, 22)}
	d.Dance = []Rect{R(420, 196, 120, 96)}
	d.Animals = []AnimalSpawn{{"dog", Vec{300, 250}, R(100, 130, 300, 240)}}
	d.Guards = [][]Vec{{{30, 140}, {30, 400}}, {{frontX - 180, 430}, {frontX + 180, 430}}, {{920, 140}, {920, 260}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

func genFestival(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "FESTIVAL", Intro: "Tres días de música, barro y tiendas de campaña. Nadie echará de menos a nadie.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{14, 4, 34, 80}, Disco: true,
		Guests: 50 + rng.Intn(8), CellSize: 16,
	}
	frontX := 400 + rng.Float64()*200
	d.Floors = []Floor{F(0, 0, 960, 452, "grass"), F(100, 120, 760, 290, "dirt")}
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.avoid = append(pl.avoid, Vec{frontX, 420}, Vec{frontX, 380})
	sx := 400 + rng.Float64()*160
	pl.at("stage", sx, 130, 0)
	pl.at("speaker", sx-210, 124, 0)
	pl.at("speaker", sx+220, 124, 0)
	pl.avoid = append(pl.avoid, Vec{sx, 180}, Vec{sx - 110, 170}, Vec{sx + 110, 170})
	d.Performers = []Perf{{Vec{sx, 100}, "drum", DirDown}, {Vec{sx - 60, 110}, "guitar", DirDown}, {Vec{sx + 60, 110}, "guitar", DirDown}, {Vec{sx, 122}, "sing", DirDown}}
	// second stage with the DJ on a side
	djx := 130.0
	if sx < 560 {
		djx = 830
	}
	pl.at("dj", djx, 290, 1)
	pl.at("speaker", djx-52, 286, 0)
	pl.at("speaker", djx+52, 286, 0)
	d.Performers = append(d.Performers, Perf{Vec{djx, 272}, "dj", DirDown})
	for i := 0; i < 2; i++ {
		pl.at("foodtruck", 130+float64(i)*700, 400, rng.Intn(4))
	}
	pl.scatter(R(30, 130, 900, 270), 6+rng.Intn(3), 120, "tent", "tent", "balloons", "cottoncandy", "bench")
	for i := 0; i < 3; i++ {
		pl.at("portapotty", 40+float64(i)*32, 440, i%2)
	}
	pl.scatter(R(30, 380, 900, 60), 3, 120, "lamp", "bush")
	pl.at("phone", 920, 380, 0)
	pl.at("car", 300, 534, 3)
	genStreet(d, frontX)
	d.Walls = append(d.Walls, vw(0, 0, 452, 10, WallFence)...)
	d.Walls = append(d.Walls, vw(950, 0, 452, 10, WallFence)...)
	d.Walls = append(d.Walls, hw(0, 960, 0, 14, WallHedge)...)
	d.Props = props
	d.Roam = []Rect{R(30, 90, 900, 330), R(14, 410, 930, 36), R(300, 456, 400, 22)}
	d.Dance = []Rect{R(sx-120, 180, 240, 90)}
	d.Animals = []AnimalSpawn{{"dog", Vec{200, 330}, R(30, 250, 300, 160)}, {"cat", Vec{760, 340}, R(600, 250, 300, 160)}}
	d.Guards = [][]Vec{{{50, 150}, {50, 400}}, {{910, 150}, {910, 400}}, {{frontX - 180, 430}, {frontX + 180, 430}}}
	d.Glows = []Glow{{Vec{sx - 200, 110}, color.RGBA{120, 40, 190, 200}, 80, true}, {Vec{sx + 200, 110}, color.RGBA{40, 80, 200, 200}, 80, true}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

func genFunfair(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "PARQUE DE ATRACCIONES", Intro: "Atracciones que no se pueden montar y una fiesta que sí se puede estropear.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{18, 4, 30, 70}, Disco: true,
		Guests: 50 + rng.Intn(8), CellSize: 16,
	}
	frontX := 400 + rng.Float64()*200
	d.Floors = []Floor{F(0, 0, 960, 452, "pave"), F(60, 100, 840, 300, "cobble")}
	for _, r := range []Rect{R(30, 60, 90, 120), R(840, 60, 90, 120)} {
		d.Floors = append(d.Floors, F(r.X, r.Y, r.W, r.H, "grass"))
	}
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.avoid = append(pl.avoid, Vec{frontX, 420}, Vec{frontX, 380})
	// big rides: three slots chosen at random, left / centre / right at the back
	slots := [][2]float64{{190, 200}, {480, 200}, {770, 200}}
	rides := []string{"ferris", "carousel", "coaster"}
	rng.Shuffle(len(rides), func(i, j int) { rides[i], rides[j] = rides[j], rides[i] })
	for i, r := range rides {
		x, y := slots[i][0], slots[i][1]
		switch r {
		case "ferris":
			y = 190
		case "carousel":
			y = 190
		case "coaster":
			y = 150
		}
		pl.at(r, x, y, 0)
		pl.avoid = append(pl.avoid, Vec{x, y + 24}, Vec{x - 40, y}, Vec{x + 40, y})
	}
	for i := 0; i < 4; i++ { // bumper cars in a loose group
		pl.at("bumpercar", 330+float64(i%2)*70, 310+float64(i/2)*54, i)
	}
	pl.at("ticketbooth", 120, 300, 0)
	pl.at("cottoncandy", 800, 300, 0)
	pl.at("balloons", 860, 330, 0)
	pl.at("dj", 640, 340, 0)
	pl.at("speaker", 592, 336, 0)
	pl.at("speaker", 690, 336, 0)
	d.Performers = []Perf{{Vec{640, 322}, "dj", DirDown}}
	pl.scatter(R(60, 260, 840, 150), 4, 120, "bench", "lamp", "tree", "planter")
	pl.at("phone", 920, 400, 0)
	pl.at("car", 560, 534, 1)
	genStreet(d, frontX)
	d.Walls = append(d.Walls, vw(0, 0, 452, 10, WallFence)...)
	d.Walls = append(d.Walls, vw(950, 0, 452, 10, WallFence)...)
	d.Walls = append(d.Walls, hw(0, 960, 0, 14, WallHedge)...)
	d.Props = props
	d.Roam = []Rect{R(30, 90, 900, 330), R(14, 410, 930, 36), R(300, 456, 400, 22)}
	d.Dance = []Rect{R(560, 350, 160, 50)}
	d.Animals = []AnimalSpawn{{"cat", Vec{140, 380}, R(40, 330, 250, 90)}}
	d.Guards = [][]Vec{{{50, 130}, {50, 400}}, {{910, 130}, {910, 400}}, {{frontX - 180, 430}, {frontX + 180, 430}}}
	d.Glows = []Glow{{Vec{480, 170}, color.RGBA{200, 60, 140, 150}, 110, true}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// genKinds lists every generated scenario type.
var genKinds = []string{"house", "club", "pub", "cinema", "mall", "market", "plaza", "paris", "bullring", "stadium", "festival", "funfair", "roof", "beach", "skatepark", "circuit", "station", "highway", "trainstation"}

// genExotic adds a few unusual animals to the open-air venues (never more than a couple).
func genExotic(rng *rand.Rand, d *LevelDef, kind string) {
	var pool []string
	chance := 0.0
	switch kind {
	case "paris", "plaza":
		pool, chance = []string{"giraffe", "elephant", "kangaroo", "cat", "cat"}, 0.5
	case "festival":
		pool, chance = []string{"elephant", "kangaroo", "giraffe"}, 0.65
	case "funfair":
		pool, chance = []string{"giraffe", "kangaroo", "elephant"}, 0.65
	case "beach":
		pool, chance = []string{"kangaroo", "elephant", "cat"}, 0.4
	case "stadium":
		pool, chance = []string{"kangaroo", "bull"}, 0.4
	case "bullring":
		pool, chance = []string{"cat", "hen"}, 0.5
	case "house", "pub", "club", "birthday":
		pool, chance = []string{"cat", "cat", "hen"}, 0.4
	default:
		return
	}
	var big []Rect
	for _, r := range d.Roam {
		if r.W > 200 && r.H > 100 {
			big = append(big, r)
		}
	}
	if len(big) == 0 {
		return
	}
	for n := 0; n < 2 && rng.Float64() < chance; n++ {
		r := big[rng.Intn(len(big))]
		a := R(r.X+10, r.Y+10, math.Min(r.W-20, 260), math.Min(r.H-20, 150))
		k := pool[rng.Intn(len(pool))]
		d.Animals = append(d.Animals, AnimalSpawn{k, Vec{a.X + a.W/2, a.Y + a.H/2}, a})
		if k == "hen" {
			d.Animals = append(d.Animals, AnimalSpawn{"chick", Vec{a.X + a.W/2 + 18, a.Y + a.H/2}, a}, AnimalSpawn{"chick", Vec{a.X + a.W/2 - 14, a.Y + a.H/2 + 8}, a})
		}
	}
}

func genByKind(rng *rand.Rand, kind string) *LevelDef {
	d := genByKind0(rng, kind)
	genExotic(rng, d, kind)
	pruneWallProps(d)
	resolveOverlaps(d)
	clearStage(d)
	return d
}

// wallHuggers may stand on a wall on purpose (grandstands against the fence, lifebuoys on the rail).
var wallHuggers = map[string]bool{"stands": true, "lifebuoy": true, "dj": true, "performer": true}

// freeStanding scenery must not touch a wall at all.
var freeStanding = map[string]bool{"tombstone": true, "pine": true, "snowman": true, "pumpkin": true, "deadtree": true, "tree": true,
	"palm": true, "bush": true, "planter": true, "bench": true, "shelf": true, "roulette": true, "slot": true, "table": true, "fountain": true}

// pruneWallProps removes scenery that would be planted on top of a wall (more than a third of
// its footprint inside it); furniture just leaning on a wall stays.
func pruneWallProps(d *LevelDef) {
	out := d.Props[:0]
	for _, in := range d.Props {
		k := propKinds[in.Kind]
		if k == nil || wallHuggers[in.Kind] {
			out = append(out, in)
			continue
		}
		foot := Rect{in.X - k.FW/2, in.Y - k.FH, k.FW, k.FH}
		bad := false
		for _, w := range d.Walls {
			if w.Style == WallVoid || !rectsOverlap(foot, w.R) {
				continue
			}
			ox := math.Min(foot.X+foot.W, w.R.X+w.R.W) - math.Max(foot.X, w.R.X)
			oy := math.Min(foot.Y+foot.H, w.R.Y+w.R.H) - math.Max(foot.Y, w.R.Y)
			lim := 0.33
			if freeStanding[in.Kind] {
				lim = 0.04
			}
			if ox*oy > lim*foot.W*foot.H {
				bad = true
				break
			}
		}
		if !bad {
			out = append(out, in)
		}
	}
	d.Props = out
}

func genByKind0(rng *rand.Rand, kind string) *LevelDef {
	switch kind {
	case "house", "club", "pub", "cinema", "mall", "market", "casino":
		return genBuilding(rng, kind)
	case "station":
		return genStation(rng)
	case "graveyard":
		return genGraveyard(rng)
	case "ski":
		return genSki(rng)
	case "plaza":
		return genPlaza(rng, false)
	case "paris":
		return genPlaza(rng, true)
	case "bullring":
		return genBullring(rng)
	case "stadium":
		return genStadium(rng)
	case "festival":
		return genFestival(rng)
	case "funfair":
		return genFunfair(rng)
	case "roof":
		return genRoof(rng)
	case "yacht":
		return genYacht(rng)
	case "skatepark":
		return genSkatepark(rng)
	case "circuit":
		return genCircuit(rng)
	case "highway":
		return genHighway(rng)
	case "trainstation":
		return genTrainStation(rng)
	case "birthday":
		return genBirthday(rng)
	case "hockey":
		return genHockey(rng)
	case "disco70":
		return genDisco70(rng)
	case "italy":
		return genItaly(rng) // italy.go
	case "pirate":
		return genPirate(rng) // pirate.go
	case "alcala":
		return genAlcala(rng) // alcala.go
	case "guadalajara":
		return genGuadalajara(rng) // guadalajara.go
	case "barcelona":
		return genBarcelona(rng) // barcelona.go
	}
	return genBeach(rng)
}

// genYacht is deliberately broad and uncluttered: it gives the same layered
// deck / lounge / open water feeling as a party boat while preserving generous
// routes for guests, medics and helicopters.
func genYacht(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "CRUCERO NOCTURNO", Intro: "Mar abierto, música alta y ninguna salida elegante.",
		W: 960, H: 540, Sky: "sea", Police: "car", Transit: "sea", Tint: color.RGBA{0, 16, 35, 72},
		CarFrom: Vec{1040, 508}, CarTo: Vec{560, 496}, ParkX0: 220, ParkX1: 780,
		Start: Vec{105, 410}, Pads: []Vec{{835, 271}, {114, 418}}, Guests: 46, CellSize: 16,
		Hull: true, HullRect: Rect{60, 60, 842, 422}, HullName: "SS ROYAL", Shape: yachtShape,
	}
	// Hull, lower lounge and upper deck. The wide central deck is left open.
	d.Floors = []Floor{
		F(70, 70, 820, 400, "deck"), F(110, 112, 430, 210, "white"),
		F(555, 112, 286, 210, "deck"), F(115, 340, 725, 86, "boardwalk"),
		FC(652, 212, 138, 88, "tile", 0x3aaed0),
		FC(800, 236, 70, 70, "helipad", 0xd8e1e8), FC(72, 376, 84, 84, "helipad", 0xd8e1e8),
	}
	d.Pools = []Rect{R(652, 212, 138, 88)}
	d.Swim = true
	d.Walls = walls(
		hw(70, 760, 63, 6, WallRail), hw(70, 760, 473, 6, WallRail),
		vw(63, 70, 470, 6, WallRail),
		hw(110, 540, 102, 10, WallPlaster, [2]float64{265, 375}),
		vw(540, 112, 322, 8, WallPlaster, [2]float64{170, 282}),
	)
	d.Props = []PropInst{
		P("dj", 332, 140), P("speaker", 235, 148), P("speaker", 428, 148), P("bar", 322, 286), P("punch", 470, 196),
		P("couch", 172, 290), PV("couch", 476, 286, 1), P("table", 225, 250), P("table", 410, 238),
		P("sunbed", 592, 168), PV("sunbed", 615, 325, 1), P("umbrella", 600, 205), PV("umbrella", 804, 207, 2),
		P("bar", 742, 178), P("phone", 812, 330), P("closet", 94, 190), P("plant", 512, 162),
		P("lamp", 108, 355), P("lamp", 530, 360), P("lamp", 824, 346), P("plant", 96, 100), P("plant", 854, 100),
		P("lifebuoy", 180, 68), P("lifebuoy", 480, 68), P("lifebuoy", 780, 68),
		P("lifebuoy", 180, 475), P("lifebuoy", 780, 475),
	}
	d.Performers = []Perf{{Vec{332, 126}, "dj", DirDown}}
	d.Roam = []Rect{R(84, 82, 786, 370), R(122, 122, 400, 185), R(566, 125, 265, 206), R(120, 348, 710, 72)}
	d.Dance = []Rect{R(230, 170, 190, 82)}
	d.Doors = append(d.Doors, doorLog...)
	d.Walls = append(d.Walls, voidWalls(yachtShape, d.W, d.H)...)
	return d
}

// genSkatepark: an open park with ramps, rails and a half-pipe, the same
// generous open layout as genPlaza so there is always room to walk around
// every obstacle.
func genSkatepark(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "SKATE PARK", Intro: "Rampas, barandillas y una fiesta sin casco.",
		W: 960, H: 540, Sky: "none", Police: "car", Tint: color.RGBA{6, 10, 24, 70},
		Guests: 40 + rng.Intn(8), CellSize: 16,
	}
	frontX := 380 + rng.Float64()*200
	d.Floors = []Floor{F(0, 0, 960, 452, "grass"), F(60, 90, 840, 320, "concrete")}
	// v1.4.0 (user): graffiti walls: the back wall of the park is brick now and a free-standing brick wall stands
	// between the two rails; their faces take the pieces of the graffiti crew (skategraffiti.go, pe_graffiti.go).
	d.Walls = walls(
		hw(0, 960, 20, 20, WallBrick),
		vw(0, 40, 452, 10, WallFence),
		vw(950, 40, 452, 10, WallFence),
		hw(skateWallX0, skateWallX1, skateWallY, skateWallH, WallBrick),
	)
	// a paved apron in front of the free-standing wall: the painters stand on it (gfSegments wants street floor)
	d.Floors = append(d.Floors, F(skateWallX0, skateWallY+skateWallH, skateWallX1-skateWallX0, 20, "pave"))
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	// three bowls (round, kidney and rounded square) with kids riding them, and two rails
	pl.at("skatebowl", 200, 190, 0)
	pl.at("skatebowl", 480, 180, 1)
	pl.at("skatebowl", 760, 200, 2)
	pl.at("rail", 330, 310, 0)
	pl.at("rail", 640, 310, 0)
	pl.avoid = append(pl.avoid, Vec{200, 190}, Vec{480, 180}, Vec{760, 200}, Vec{330, 310}, Vec{640, 310})
	for x := skateWallX0; x <= skateWallX1; x += 30 { // nothing in front of (or on) the graffiti wall
		pl.avoid = append(pl.avoid, Vec{x, skateWallY + 8})
	}
	pl.at("dj", 110, 360, 0)
	pl.at("speaker", 62, 356, 0)
	pl.at("speaker", 160, 356, 0)
	d.Performers = []Perf{{Vec{110, 342}, "dj", DirDown}}
	pl.scatter(R(60, 90, 840, 320), 6, 90, "bench", "tree", "bush", "lamp")
	pl.at("phone", 920, 300, 0)
	pl.at("car", 200+rng.Float64()*100, 534, 1)
	genStreet(d, frontX)
	d.Props = props
	d.Roam = []Rect{R(60, 90, 840, 320), R(14, 410, 930, 36), R(300, 456, 400, 22)}
	d.Dance = []Rect{R(80, 330, 120, 50)}
	d.Animals = []AnimalSpawn{{"dog", Vec{700, 400}, R(600, 350, 300, 90)}}
	d.Guards = [][]Vec{{{140, 150}, {140, 380}}, {{820, 150}, {820, 380}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// genCircuit: the street runs across the middle of the scenario instead of
// along one edge, with grandstands to the north and a gap in the track fences to
// cross on foot (FastTraffic + racetrack.go: fast race cars, often, with a warning).
func genCircuit(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "CIRCUITO", Intro: "Un circuito de carreras convertido en pista de baile. Los coches no van a parar.",
		W: 960, H: 540, Sky: "none", Police: "car", FastTraffic: true,
		Tint:    color.RGBA{4, 4, 16, 80},
		Start:   Vec{480, 110},
		CarFrom: Vec{1040, 285}, CarTo: Vec{740, 285},
		Guests: 40 + rng.Intn(6), CellSize: 16,
	}
	// The dashed center line sits at CarFrom.Y: the asphalt band straddles
	// it symmetrically so both one-way lanes (see laneOffset) land cleanly
	// on asphalt, one on each side of the line.
	d.Floors = []Floor{
		F(0, 20, 960, 224, "grass"),
		F(0, 326, 960, 180, "grass"),
		F(0, 244, 960, 82, "asphalt"),
		F(0, 283, 960, 4, "dash"),
	}
	d.Walls = walls(
		hw(0, 960, 20, 14, WallFence),
		hw(0, 960, 506, 14, WallFence),
		vw(0, 20, 486, 10, WallFence),
		vw(950, 20, 486, 10, WallFence),
		hw(0, 960, 236, 8, WallFence, [2]float64{410, 530}),
		hw(0, 960, 326, 8, WallFence, [2]float64{410, 530}),
	)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	// v1.4.0 (user): no grey bridge over the middle of the track and no stands at the bottom (they only got in the
	// way); people cross the track on foot through the gap in the fences, and the race cars (racetrack.go) warn first.
	for i := 0; i < 4; i++ {
		pl.at("stands", 70+float64(i)*220, 60, i%2)
	}
	pl.at("dj", 480, 90, 0)
	pl.at("speaker", 420, 86, 0)
	pl.at("speaker", 540, 86, 0)
	d.Performers = []Perf{{Vec{480, 72}, "dj", DirDown}}
	pl.scatter(R(40, 40, 880, 160), 6, 80, "bench", "plant", "lamp")
	pl.scatter(R(40, 350, 880, 140), 6, 80, "bench", "plant", "lamp")
	pl.at("phone", 920, 100, 0)
	d.Props = props
	d.Roam = []Rect{R(40, 40, 880, 170), R(40, 340, 880, 150)}
	d.Dance = []Rect{R(380, 60, 200, 80)}
	d.Guards = [][]Vec{{{140, 80}, {800, 80}}, {{140, 440}, {800, 440}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// genNamed builds a scenario of a given type from a fixed seed, retrying until it is fully walkable.
func genNamed(kind string, seed int64, title string) *LevelDef {
	var fallback *LevelDef
	for try := 0; try < 40; try++ {
		rng := rand.New(rand.NewSource(seed + int64(try)*7919))
		d := genByKind(rng, kind)
		if genValidate(d) {
			if title != "" {
				d.Title = title
			}
			// prefer a layout where nothing stands on anything else (a key object with no room is kept by resolveOverlaps)
			if try < 39 && len(levelOverlaps(buildLevel(d))) > 0 {
				if fallback == nil {
					fallback = d
				}
				continue
			}
			return d
		}
	}
	return fallback
}
