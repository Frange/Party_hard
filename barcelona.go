package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// BARCELONA (v1.4.0): the 4th party of the list. It takes the place of the old ranch party (index 3, icon "ranch"):
// the SAME index, the same icon and the same editor key (c04_ranch), so the wins, medals and the frontier of the saves
// stay where they were (a save that had beaten the ranch has beaten BARCELONA; nothing to migrate). The internal icon
// keeps its old name on purpose (Progress.Medals and the album are keyed by it).
//
// A big square of the city on a sunny day. The back of the square is a painted backdrop that nobody walks into (the
// floor "barcelona", one picture made once by code, barcelona_backdrop.go): the sky and the sea with the sailing boats
// and the sail-shaped hotel, the hill of the Tibidabo with its church, its tower and the cable car (animated), the
// modernist house of mosaic, the city hall with its clock tower and the flags of Spain and Catalonia on its balcony,
// the market hall of brick and stained glass, the Sagrada Família with its crane and the Arc de Triomf.
//
// The square itself is wide open: the round mosaic in the middle is the dance floor (the DJ at its back edge between
// the two flagpoles of Spain and Catalonia), a fountain in a flower bed on each side, café terraces on the left and
// by the market, palms, trees, lamps, benches, the long mosaic bench, the blue newspaper kiosk, the pavilion of the
// park with its icing roof, and the punch in front of the mosaic. No closed building: everything to enter is open air.
// The small street with the cars runs along the bottom (genStreet: cars, police car, ambulance).
// The names of the city and of its places stay as they are (a real city); no real person is named.
// ---------------------------------------------------------------------------

const (
	barcelonaTitle = "BARCELONA"
	bcnBackH       = 150.0 // the painted backdrop band (void: scenery only)
)

// landmarks (used by the tests too)
var (
	bcnMosaic    = Vec{480, 312}                 // the centre of the mosaic
	bcnFountains = []Vec{{276, 350}, {684, 350}} // the anchors (front of the flower bed) of the fountains
	bcnFlagpoles = []Vec{{400, 280}, {560, 280}} // the flags of Spain and Catalonia by the DJ
	bcnCable     = Vec{530, 76}                  // the anchor of the cable car on the backdrop
)

func genBarcelona(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: barcelonaTitle, Intro: "Sol, mar y mosaico: la plaza baila a la catalana.",
		W: 960, H: 540, Sky: "none", Police: "car", Day: true, Tint: color.RGBA{10, 20, 40, 18},
		Guests: 46 + rng.Intn(8), CellSize: 16,
	}
	frontX := 400 + math.Round(rng.Float64()*140)
	d.Floors = []Floor{
		F(0, bcnBackH, 960, 452-bcnBackH, "pave"),
		F(0, 0, 960, bcnBackH, "barcelona"), // the painted sky, sea, hill and landmarks (barcelona_backdrop.go)
		F(16, 162, 214, 116, "cobble"),      // the terrace on the left
		F(744, 162, 200, 70, "cobble"),      // the terrace by the market
	}
	d.Walls = walls([]Wall{{R(0, 0, 960, bcnBackH), WallVoid}})
	genStreet(d, frontX)
	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	pl.avoid = append(pl.avoid, Vec{frontX, 420}, Vec{frontX, 380})

	// scenery on the backdrop (never blocks anything): the flags on the city hall's balcony and the cable car
	pl.at("flagmini", bcnTowerFlags[0].X, bcnTowerFlags[0].Y, flagSpain)
	pl.at("flagmini", bcnTowerFlags[1].X, bcnTowerFlags[1].Y, flagCatalonia)
	pl.at("bcn_cablecar", bcnCable.X, bcnCable.Y, 0)

	// the mosaic in the middle (the dance floor), the DJ at its back edge with his speakers, the two flagpoles
	mx, my := bcnMosaic.X, bcnMosaic.Y
	pl.at("bcn_mosaic", mx, my+bcnMosaicH/2, 0)
	pl.at("dj", mx, my-28, 0)
	pl.at("speaker", mx-44, my-28, 0)
	pl.at("speaker", mx+44, my-28, 0)
	d.Performers = []Perf{{Vec{mx, my - 44}, "dj", DirDown}}
	pl.at("flagpole", bcnFlagpoles[0].X, bcnFlagpoles[0].Y, flagSpain)
	pl.at("flagpole", bcnFlagpoles[1].X, bcnFlagpoles[1].Y, flagCatalonia)

	// the two fountains in their flower beds, one on each side of the mosaic
	for _, f := range bcnFountains {
		pl.at("bcn_fountain", f.X, f.Y, 0)
	}

	// the punch, in front of the mosaic
	pl.at("punch", mx, 398, rng.Intn(2))

	// the terraces: café tables with umbrellas on the left and by the market
	for i, p := range [][2]float64{{44, 206}, {112, 200}, {184, 208}, {64, 262}, {150, 258}} {
		pl.at("cafetable", p[0], p[1], (i+rng.Intn(3))%3)
	}
	for i, p := range [][2]float64{{772, 206}, {844, 200}, {916, 208}} {
		pl.at("cafetable", p[0], p[1], (i+1+rng.Intn(3))%3)
	}

	// the park: the pavilion with the icing roof, the long mosaic bench, the blue kiosk
	pl.at("bcn_pavilion", 872, 400, 0)
	pl.at("bcn_bench", 744, 404, 0)
	pl.at("bcn_kiosk", 70, 350, 0)

	// palms, trees, lamps, benches and bins along the walks
	for i, p := range [][2]float64{{248, 228}, {712, 228}, {230, 412}, {120, 412}} {
		pl.at("palm", p[0], p[1], i%4)
	}
	for _, p := range [][2]float64{{330, 190}, {630, 190}, {940, 300}} {
		pl.at("tree", p[0], p[1], rng.Intn(2))
	}
	for _, p := range [][2]float64{{372, 238}, {588, 238}, {196, 316}, {766, 316}, {300, 412}, {664, 412}, {260, 172}, {700, 172}} {
		pl.at("lamp", p[0], p[1], 0)
	}
	for _, p := range [][2]float64{{410, 196}, {550, 196}, {820, 300}, {140, 304}} {
		pl.at("bench", p[0], p[1], 0)
	}
	for _, p := range [][2]float64{{352, 200}, {612, 200}, {870, 330}, {30, 412}} {
		pl.at("alc_bin", p[0], p[1], 0)
	}
	pl.at("phone", 940, 250, 0)

	d.Props = props
	d.Roam = []Rect{
		R(20, 170, 206, 104),  // the terrace on the left
		R(380, 160, 200, 60),  // in front of the market
		R(748, 166, 192, 60),  // the terrace by the market
		R(300, 372, 360, 30),  // in front of the mosaic
		R(780, 250, 150, 110), // by the pavilion
		R(20, 290, 170, 90),   // by the kiosk
		R(20, 426, 920, 12),   // along the fence
	}
	d.Dance = []Rect{R(mx-70, my-6, 140, 44)}
	d.Animals = []AnimalSpawn{{"dog", Vec{560, 400}, R(300, 370, 360, 60)}}
	d.Guards = [][]Vec{{{40, 300}, {200, 300}}, {{300, 432}, {920, 432}}, {{300, 236}, {660, 236}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// ---- the painted backdrop (the floor "barcelona", draw.go renderBG) ------------------------------------------------

var (
	bcnBackBuf *pixBuf
	bcnBackImg *ebiten.Image
)

// drawBarcelonaFloor paints the part r of the backdrop picture.
func drawBarcelonaFloor(bg *ebiten.Image, r Rect) {
	if bcnBackImg == nil {
		bcnBackImg = barcelonaBackdrop().img()
	}
	x0, y0 := int(math.Max(0, r.X)), int(math.Max(0, r.Y))
	x1, y1 := int(math.Min(960, r.X+r.W)), int(math.Min(bcnBackH, r.Y+r.H))
	if x1 <= x0 || y1 <= y0 {
		return
	}
	sub := bcnBackImg.SubImage(image.Rect(x0, y0, x1, y1)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x0), float64(y0))
	bg.DrawImage(sub, op)
}

// staleOverride: an editor correction saved for the old ranch party (its title was "BULL'S EYE") would lay the ranch's
// walls and props over BARCELONA: it is ignored (the file stays on disk, untouched).
func staleOverride(la *LevelAsset, icon string) bool {
	return icon == "ranch" && la != nil && la.Meta.Title != "" && la.Meta.Title != barcelonaTitle
}
