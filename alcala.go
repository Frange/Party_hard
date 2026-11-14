package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// ALCALÁ DE HENARES (v1.4.0, campaign party 7 of the list, index alcalaIdx): the big main square of a Spanish
// university town at dusk (purple-orange sky over the mountains). The back of the square is a painted backdrop that
// nobody walks into (the floor "alcala", one picture made once by code, alcala_backdrop.go): the plateresque
// university facade with its red roof, the gothic church with its rose window and its belfry, the crenellated stone
// tower, the arcaded building of the terraces and the white houses with balconies, with STORKS in their nests on the
// tower, the belfry and a chimney (alcala_props.go: they bob their heads and flap now and then).
//
// The square itself is wide open: café terraces under the arcade (left), the statue on its pedestal in the middle of a
// flower bed with little fences, the octagonal bandstand on the right where the DJ plays (the dance floor is in front
// of its steps), two bronze figures sitting on a bench, the market stalls (striped awning, ceramics and lanterns),
// trees in stone planters, benches, street lamps and bins. No closed building: the facades are only scenery. The
// small street with the cars runs along the bottom (genStreet: cars, police car, ambulance).
// The names of the city and of its places stay as they are (a real Spanish town); no real person is named.
// ---------------------------------------------------------------------------

const (
	alcalaTitle = "ALCALÁ DE HENARES"
	alcBackH    = 150.0 // the painted backdrop band (void: scenery only)
)

// landmarks (used by the tests too)
var (
	alcKiosk    = Vec{724, 254} // the centre of the bandstand's platform
	alcMonument = Vec{470, 318} // the anchor (front of the flower bed) of the statue
)

func genAlcala(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: alcalaTitle, Intro: "Cigüeñas en los tejados y fiesta mayor en la plaza.",
		W: 960, H: 540, Sky: "none", Police: "car", Day: true, Tint: color.RGBA{34, 18, 52, 40},
		Guests: 45 + rng.Intn(8), CellSize: 16,
	}
	frontX := 420 + math.Round(rng.Float64()*120)
	d.Floors = []Floor{
		F(0, alcBackH, 960, 452-alcBackH, "pave"),
		F(0, 0, 960, alcBackH, "alcala"), // the painted facades and the sky (alcala_backdrop.go)
		F(16, 166, 244, 128, "cobble"),   // the terraces under the arcade
	}
	d.Walls = walls([]Wall{{R(0, 0, 960, alcBackH), WallVoid}})
	genStreet(d, frontX)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.avoid = append(pl.avoid, Vec{frontX, 420}, Vec{frontX, 380})

	// the storks in their nests, on the backdrop (scenery: they never block anything)
	for i, p := range alcStorkSpots {
		pl.at("alc_stork", p.X, p.Y, i%2)
	}

	// the bandstand: the platform (under everyone), its roof (behind the people on it), the DJ and two speakers
	kx, ky := alcKiosk.X, alcKiosk.Y
	pl.at("alc_kiosk", kx, ky+alcKioskFront, 0)
	pl.at("alc_kioskroof", kx, ky-alcKioskRY, 0)
	pl.at("dj", kx, ky+6, 0)
	pl.at("speaker", kx-46, ky+6, 0)
	pl.at("speaker", kx+46, ky+6, 0)
	d.Performers = []Perf{{Vec{kx, ky - 8}, "dj", DirDown}}
	pl.avoid = append(pl.avoid, Vec{kx, ky + 40}, Vec{kx, ky + 70}, Vec{kx - 50, ky + 60}, Vec{kx + 50, ky + 60})

	// the statue in the middle of its flower bed
	pl.at("alc_monument", alcMonument.X, alcMonument.Y, 0)
	pl.avoid = append(pl.avoid, Vec{alcMonument.X, alcMonument.Y + 14})

	// the terraces under the arcade: café tables with their umbrellas and the bar of the café
	for i, p := range [][2]float64{{50, 206}, {124, 200}, {200, 208}, {60, 266}, {146, 262}} {
		pl.at("cafetable", p[0], p[1], (i+rng.Intn(3))%3)
	}
	pl.at("bar", 214, 286, rng.Intn(2))

	// the punch in the middle of the square, between the statue and the dance floor
	pl.at("punch", 600, 360, rng.Intn(2))

	// the two bronze figures on their bench, by the houses on the right
	pl.at("alc_bronzebench", 896, 222, 0)

	// the market: the stall with the striped awning and the one of ceramics and lanterns
	pl.at("alc_stall", 74, 404, 0)
	pl.at("alc_stall", 170, 404, 1)

	// trees in stone planters, benches, street lamps and bins along the walks
	for _, p := range [][2]float64{{300, 206}, {620, 200}, {300, 400}, {846, 412}, {942, 300}} {
		pl.at("alc_pottree", p[0], p[1], 0)
	}
	for _, p := range [][2]float64{{360, 196}, {574, 196}, {380, 418}, {540, 418}, {830, 372}} {
		pl.at("bench", p[0], p[1], 0)
	}
	for _, p := range [][2]float64{{272, 176}, {660, 172}, {270, 330}, {396, 334}, {548, 334}, {840, 186}, {250, 420}, {690, 420}} {
		pl.at("lamp", p[0], p[1], 0)
	}
	for _, p := range [][2]float64{{332, 210}, {606, 210}, {340, 330}, {870, 400}} {
		pl.at("alc_bin", p[0], p[1], 0)
	}
	pl.at("plant", 24, 330, 0)
	pl.at("phone", 940, 190, 0)

	d.Props = props
	d.Roam = []Rect{
		R(20, 170, 236, 120),  // the terraces
		R(290, 160, 340, 80),  // in front of the university
		R(290, 340, 360, 70),  // the south of the square
		R(770, 250, 170, 140), // by the houses
		R(20, 300, 230, 80),   // in front of the market
		R(20, 400, 920, 36),   // along the fence
	}
	d.Dance = []Rect{R(kx-74, ky+44, 148, 58)}
	d.Animals = []AnimalSpawn{{"dog", Vec{560, 400}, R(300, 340, 360, 80)}}
	d.Guards = [][]Vec{{{40, 300}, {250, 300}}, {{300, 420}, {920, 420}}, {{300, 236}, {640, 236}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// ---- the painted backdrop (the floor "alcala", draw.go renderBG) ---------------------------------------------------

var (
	alcBackBuf *pixBuf
	alcBackImg *ebiten.Image
)

// drawAlcalaFloor paints the part r of the backdrop picture.
func drawAlcalaFloor(bg *ebiten.Image, r Rect) {
	if alcBackImg == nil {
		alcBackImg = alcalaBackdrop().img()
	}
	x0, y0 := int(math.Max(0, r.X)), int(math.Max(0, r.Y))
	x1, y1 := int(math.Min(960, r.X+r.W)), int(math.Min(alcBackH, r.Y+r.H))
	if x1 <= x0 || y1 <= y0 {
		return
	}
	sub := alcBackImg.SubImage(image.Rect(x0, y0, x1, y1)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x0), float64(y0))
	bg.DrawImage(sub, op)
}
