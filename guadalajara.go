package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// GUADALAJARA (v1.4.0, the 14th row of the list, index guadalajaraIdx = 15, behind the retired CINE): the big square of
// the Spanish city at dusk (purple-orange sky). The back of the square is a painted backdrop that nobody walks into
// (the floor "guadalajara", one picture made once by code, guadalajara_backdrop.go): the hill with the black bull
// silhouette on top, the pink house of the "Pastelería Mariam" with its striped awning, the orange house of "Miel de la
// Alcarria", the renaissance palace with its studded facade and red roof, the stone church with its two little towers,
// the pantheon with its grey dome and rose window, and the purple stadium with its floodlights and crest up on the
// right, reached by a diagonal flight of steps.
//
// The square itself is wide open: the octagonal fountain in the middle (its jets and its water move) with its flower
// beds, the octagonal bandstand on the right where the DJ plays (the dance floor is in front of its steps), two
// pergolas with vines and benches, the honey stall of the market, the purple newspaper kiosk, the statue in its fenced
// garden, café tables in front of the cake shop, trees in stone planters, benches, street lamps and bins. No closed
// building: the facades are only scenery. The small street with the cars runs along the bottom (genStreet).
// The names of the city, its places and its shops stay as they are (a real Spanish city); no real person is named.
// ---------------------------------------------------------------------------

const (
	guadalajaraTitle = "GUADALAJARA"
	gdlBackH         = 150.0 // the painted backdrop band (void: scenery only)
)

// landmarks (used by the tests too)
var (
	gdlFountain = Vec{480, 318} // the anchor (front of the basin) of the fountain
	gdlKiosk    = Vec{760, 286} // the centre of the bandstand's platform
	gdlPark     = Vec{140, 412} // the anchor (front fence) of the statue's garden
)

func genGuadalajara(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: guadalajaraTitle, Intro: "Fuente, templete y miel de la Alcarria: fiesta en la plaza.",
		W: 960, H: 540, Sky: "none", Police: "car", Day: true, Tint: color.RGBA{40, 16, 58, 40},
		Guests: 45 + rng.Intn(8), CellSize: 16,
	}
	frontX := 400 + math.Round(rng.Float64()*140)
	d.Floors = []Floor{
		F(0, gdlBackH, 960, 452-gdlBackH, "pave"),
		F(0, 0, 960, gdlBackH, "guadalajara"), // the painted hill, facades and sky (guadalajara_backdrop.go)
		F(16, 160, 172, 112, "cobble"),        // the terrace of the cake shop
	}
	d.Walls = walls([]Wall{{R(0, 0, 960, gdlBackH), WallVoid}})
	genStreet(d, frontX)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.avoid = append(pl.avoid, Vec{frontX, 420}, Vec{frontX, 380})

	// the fountain in the middle of the square
	pl.at("gdl_fountain", gdlFountain.X, gdlFountain.Y, 0)

	// the bandstand: the platform (under everyone), its roof, the DJ and two speakers (the props of ALCALÁ's bandstand)
	kx, ky := gdlKiosk.X, gdlKiosk.Y
	pl.at("alc_kiosk", kx, ky+alcKioskFront, 0)
	pl.at("alc_kioskroof", kx, ky-alcKioskRY, 0)
	pl.at("dj", kx, ky+6, 0)
	pl.at("speaker", kx-46, ky+6, 0)
	pl.at("speaker", kx+46, ky+6, 0)
	d.Performers = []Perf{{Vec{kx, ky - 8}, "dj", DirDown}}

	// the statue in its fenced garden (bottom left)
	pl.at("gdl_park", gdlPark.X, gdlPark.Y, 0)

	// the terrace of the cake shop: café tables, and the honey stall in front of the honey shop
	for i, p := range [][2]float64{{40, 200}, {110, 196}, {70, 252}, {150, 250}} {
		pl.at("cafetable", p[0], p[1], (i+rng.Intn(3))%3)
	}
	pl.at("gdl_honeystall", 214, 214, 0)

	// the two pergolas with vines and their benches
	pl.at("gdl_pergola", 120, 316, 0)
	pl.at("gdl_pergola", 894, 362, 1)

	// the purple newspaper kiosk and the phone, on the right
	pl.at("gdl_newskiosk", 900, 248, 0)
	pl.at("phone", 940, 186, 0)

	// the punch, in the middle of the square between the fountain and the bandstand, close to the dance floor (west of
	// the fountain stays open: the VIP zone fits there without sweeping away any landmark)
	pl.at("punch", 626, 300, rng.Intn(2))

	// trees in stone planters, benches, street lamps and bins along the walks
	for _, p := range [][2]float64{{300, 226}, {640, 226}, {290, 410}, {604, 410}} {
		pl.at("alc_pottree", p[0], p[1], 0)
	}
	for _, p := range [][2]float64{{360, 190}, {600, 190}, {360, 420}, {540, 420}} {
		pl.at("bench", p[0], p[1], 0)
	}
	for _, p := range [][2]float64{{262, 178}, {700, 176}, {400, 384}, {566, 384}, {826, 186}, {222, 432}, {660, 424}} {
		pl.at("lamp", p[0], p[1], 0)
	}
	for _, p := range [][2]float64{{330, 196}, {630, 196}, {320, 416}, {940, 420}} {
		pl.at("alc_bin", p[0], p[1], 0)
	}

	d.Props = props
	d.Roam = []Rect{
		R(20, 166, 220, 110),  // the terrace of the cake shop
		R(290, 160, 380, 60),  // in front of the palace and the churches
		R(250, 340, 420, 60),  // the south of the square
		R(830, 170, 120, 50),  // by the newspaper kiosk
		R(250, 240, 120, 100), // west of the fountain
		R(20, 400, 920, 36),   // along the fence
	}
	d.Dance = []Rect{R(kx-74, ky+44, 148, 56)}
	d.Animals = []AnimalSpawn{{"dog", Vec{540, 380}, R(260, 340, 380, 70)}}
	d.Guards = [][]Vec{{{40, 290}, {240, 290}}, {{300, 432}, {920, 432}}, {{300, 236}, {660, 236}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// ---- the painted backdrop (the floor "guadalajara", draw.go renderBG) ----------------------------------------------

var (
	gdlBackBuf *pixBuf
	gdlBackImg *ebiten.Image
)

// drawGuadalajaraFloor paints the part r of the backdrop picture.
func drawGuadalajaraFloor(bg *ebiten.Image, r Rect) {
	if gdlBackImg == nil {
		gdlBackImg = guadalajaraBackdrop().img()
	}
	x0, y0 := int(math.Max(0, r.X)), int(math.Max(0, r.Y))
	x1, y1 := int(math.Min(960, r.X+r.W)), int(math.Min(gdlBackH, r.Y+r.H))
	if x1 <= x0 || y1 <= y0 {
		return
	}
	sub := gdlBackImg.SubImage(image.Rect(x0, y0, x1, y1)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x0), float64(y0))
	bg.DrawImage(sub, op)
}
