package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// ITALIA (campaign party 20, right after the AUTOPISTA): an Italian piazza in broad daylight (Day: no night layer,
// only a faint cool wash: soft neutral sunlight, never the golden / yellow cast the user rejected; no lamps lit). The
// leaning tower stands on its lawn, a ruined Roman amphitheatre (a walk-in ring: the DJ plays in the arena, you get in through the big arch or
// the collapsed gap) fills the right, and a stretch of Venice on the left: a canal with moored gondolas, mooring poles
// and coloured palaces (the water is not walkable; a small wooden landing juts into it). Gelato cart, pizza oven,
// tables with umbrellas, cypresses, a fountain and vespas; the small street with the cars runs along the bottom.
// Props: italy_props.go; texts and tables: italy_text.go.
// ---------------------------------------------------------------------------

// the amphitheatre's centre and the canal (used by the tests too)
var (
	italyColos = Vec{742, 150}
	italyCanal = R(0, 150, 372, 56)
	italyPier  = R(252, 150, 34, 56)
	italyTower = Vec{446, 332} // the centre of the base of the leaning tower
)

func genItaly(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: "ITALIA", Intro: "Pizza, góndolas y una torre que ya se cae sola.",
		W: 960, H: 540, Sky: "none", Police: "car", Day: true, Tint: color.RGBA{10, 13, 20, 36},
		Guests: 44 + rng.Intn(8), CellSize: 16,
	}
	frontX := 470 + math.Round(rng.Float64()*120)
	cx, cy := italyColos.X, italyColos.Y
	d.Floors = []Floor{
		F(0, 0, 960, 452, "cobble"),
		F(0, 0, 372, 150, "pave"), // the palaces stand on the canal side
		F(0, 206, 372, 8, "pave"), // the stone quay
		Floor{R: italyCanal, Kind: "water"},
		Floor{R: italyPier, Kind: "boardwalk"},
	}
	ws := [][]Wall{
		{{R(0, 0, 960, 14), WallVoid}},
		{{R(italyCanal.X, italyCanal.Y, italyPier.X-italyCanal.X, italyCanal.H), WallVoid}}, // the canal, both sides of the landing
		{{R(italyPier.X+italyPier.W, italyCanal.Y, italyCanal.X+italyCanal.W-italyPier.X-italyPier.W, italyCanal.H), WallVoid}},
		{{R(0, 14, 372, 128), WallVoid}},  // behind the palaces
		{{R(560, 14, 380, 76), WallVoid}}, // behind the amphitheatre
	}
	// the pockets between the back of the ring and the band above it: nobody may hide behind the tall walls
	for x0 := 560.0; x0 < 930; x0 += 20 {
		ymin := math.Inf(1)
		for x := x0; x <= x0+20; x++ {
			dx := (x - cx) / colosRX
			if math.Abs(dx) >= 1 {
				continue
			}
			ymin = math.Min(ymin, cy-colosRY*math.Sqrt(1-dx*dx))
		}
		if ymin < math.Inf(1) && ymin-2 > 90 {
			ws = append(ws, []Wall{{R(x0, 90, 20, ymin-2-90), WallVoid}})
		}
	}
	d.Walls = walls(ws...)
	genStreet(d, frontX)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.avoid = append(pl.avoid, Vec{frontX, 420}, Vec{frontX, 380})

	// the amphitheatre: the sand, the ring and the DJ in the arena
	fl := propKinds["colosfloor"]
	pl.at("colosfloor", cx, cy+float64(fl.H)/2, 0)
	for _, p := range colosParts {
		pl.at(p.Kind, cx+p.D.X, cy+p.D.Y, 0)
	}
	pl.at("dj", cx+20, cy-8, 0)
	pl.at("speaker", cx-34, cy-10, 0)
	pl.at("speaker", cx+74, cy-10, 0)
	d.Performers = []Perf{{Vec{cx + 20, cy - 22}, "dj", DirDown}, {Vec{750, 288}, "guitar", DirDown}}

	// the tower in its garden (an organic lawn drawn as ground, italy_props.go), with cypresses; the tower stands low
	// enough that its flag stays clear of the party title and the intro plate at the start
	pl.at("pisalawn", italyTower.X+pisaLawnW/2-pisaLawnTX+pisaBaseX-pisaW/2, italyTower.Y+pisaLawnH-pisaLawnTY+pisaH-pisaBaseY, 0)
	pl.at("pisa", italyTower.X-pisaBaseX+pisaW/2, italyTower.Y+pisaH-pisaBaseY, 0)
	for _, p := range [][2]float64{{394, 300}, {546, 292}, {412, 214}, {526, 206}} {
		pl.at("cypress", p[0], p[1], 0)
	}

	// Venice: palaces on the far side of the canal, gondolas, mooring poles and the red umbrella on the quay
	for i := 0; i < 4; i++ {
		pl.at("venpalazzo", 46+float64(i)*92, italyCanal.Y, (i+rng.Intn(2)*2)%4)
	}
	pl.at("gondola", 196, 202, 0)
	pl.at("gondola", 84, 180, 1)
	for i, p := range [][2]float64{{20, 204}, {32, 204}, {300, 202}, {312, 202}, {356, 204}} {
		pl.at("palina", p[0], p[1], i%3)
	}
	pl.at("umbrella", 332, 252, 0)

	// the piazza: café tables, gelato and pizza, the fountain, benches, potted plants (daytime: no street lamps lit)
	for i, p := range [][2]float64{{50, 272}, {130, 266}, {50, 344}, {200, 334}} {
		pl.at("cafetable", p[0], p[1], i%3)
	}
	pl.at("gelato", 250, 264, rng.Intn(2))
	pl.at("pizzaoven", 120, 404, 0)
	pl.at("fountain", 616, 352, 0)
	pl.avoid = append(pl.avoid, Vec{616, 332}, Vec{570, 342}, Vec{660, 342})
	pl.at("bench", 330, 330, 0)
	pl.at("bench", 560, 404, 0)
	pl.at("bar", 840, 300, rng.Intn(2))
	pl.at("plant", 360, 260, 0)
	pl.at("plant", 584, 262, 0)
	pl.at("plant", 290, 420, 0)
	for _, p := range [][2]float64{{570, 222}, {924, 214}, {930, 386}, {20, 410}} {
		pl.at("cypress", p[0], p[1], 0)
	}
	for i, x := range []float64{700, 740, 780} { // vespas parked by the street
		pl.at("vespa", x, 432, (i+rng.Intn(3))%3)
	}
	pl.at("phone", 930, 120, 0)

	d.Props = props
	d.Roam = []Rect{
		R(20, 222, 330, 176),  // the café side
		R(380, 262, 520, 160), // the piazza
		R(680, 120, 130, 70),  // the arena
		R(20, 400, 920, 36),   // along the fence
		R(392, 150, 150, 150), // the garden of the tower
	}
	d.Dance = []Rect{R(680, 156, 116, 36), R(680, 300, 140, 70)}
	d.Animals = []AnimalSpawn{{"dog", Vec{300, 380}, R(200, 300, 300, 120)}}
	d.Guards = [][]Vec{{{40, 232}, {350, 232}}, {{400, 420}, {900, 420}}, {{620, 250}, {900, 250}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// drawCanal paints the "water" floor of the canal: green Venetian water with ripples catching the pale blue of the
// daytime sky, darker under the palaces and a stone quay edge along the bottom and the right.
func drawCanal(bg *ebiten.Image, r Rect) {
	deep, mid, light := rgb(0x1e5a5e), rgb(0x2f7f80), rgb(0x6ab8b0)
	for y := r.Y; y < r.Y+r.H; y += 2 {
		t := (y - r.Y) / r.H
		fillRect(bg, r.X, y, r.W, 2, mixc(deep, mid, t))
	}
	for i := 0; i < int(r.W*r.H/90); i++ { // ripples
		x := r.X + float64(itHash(i, 11)%int(r.W))
		y := r.Y + 6 + float64(itHash(i, 17)%int(math.Max(1, r.H-10)))
		w := 4 + float64(itHash(i, 23)%8)
		fillRect(bg, x, y, math.Min(w, r.X+r.W-x), 1, rgba([]uint32{0x6ab8b0, 0x8ec8dc}[i%2], 150)) // ripples and sky
	}
	for i := 0; i < int(r.W/14); i++ { // sun glints
		x := r.X + float64(itHash(i, 29)%int(r.W))
		y := r.Y + 10 + float64(itHash(i, 31)%int(math.Max(1, r.H-16)))
		fillRect(bg, x, y, 2, 1, light)
	}
	fillRect(bg, r.X, r.Y, r.W, 3, rgba(0x0a2a2c, 200)) // the shadow of the palaces
	fillRect(bg, r.X, r.Y+r.H-3, r.W, 3, rgb(0xd8ccb0)) // the quay edge
	fillRect(bg, r.X, r.Y+r.H-1, r.W, 1, rgb(0x8a8068))
	fillRect(bg, r.X+r.W-3, r.Y, 3, r.H, rgb(0xd8ccb0))
}
