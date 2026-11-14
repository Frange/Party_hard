package main

import (
	"image"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// PARÍS (v1.4.0; it was called TORRE EIFFEL: same index, same icon "eiffel", same editor key and medals). The Eiffel
// Tower still stands in the middle of its gardens (genPlaza, gen2.go) and every event of the party is the same; the
// city came around it:
//   - the back of the square is a painted backdrop nobody walks into (the floor "paris", paris_backdrop.go): the night
//     sky with its moon, Montmartre with the white domes of the Sacré-Cœur, the Arc de Triomphe at the end of an
//     avenue, and the row of Haussmann buildings with their zinc mansard roofs, dormers, chimneys and iron balconies,
//     a French flag waving on a balcony (flagmini, flags.go);
//   - under it the ground floor of the buildings is a real wall (the brick shopfronts, with their three shop
//     doors): the graffiti painter keeps his wall;
//   - on the square: bistro tables with wicker chairs, the art nouveau metro entrance, the green newspaper kiosk, the
//     bouquinistes' green boxes of books, the bakery stall with its baguettes, period street lamps, a little carousel
//     that turns, the accordion player and the mime (both animated), and the French flag on a pole.
// The names of the city and of its places stay as they are; no real person is named.
// ---------------------------------------------------------------------------

const (
	parisTitle = "PARÍS"
	parBackH   = 74.0 // the painted backdrop band (void: scenery only); the shopfront wall runs below it to y 98
)

// parisDress: what PARÍS adds to the plaza of genPlaza (eiffel): the backdrop, the shopfront wall and the city's props.
// It returns the walls of the top of the square (they replace the brick facade of the PLAZA EUROPEA).
func parisDress(d *LevelDef, pl *genPlacer, rng *rand.Rand) [][]Wall {
	d.Floors = append(d.Floors, F(0, 0, 960, parBackH, "paris"))
	ws := [][]Wall{
		{{R(0, 0, 960, parBackH), WallVoid}},
		// the shopfronts: a real brick wall (three shop doors), so the painter of graffiti still has his wall
		hw(0, 960, parBackH, 98-parBackH, WallBrick, [2]float64{150, 214}, [2]float64{430, 494}, [2]float64{700, 764}),
	}
	// scenery on the backdrop: the French flag on a balcony
	pl.at("flagmini", parFlagSpot.X, parFlagSpot.Y, flagFrance)
	// the little carousel, the metro entrance and the French flag on its pole
	pl.at("carousel", 744, 372, 0)  // (the top right garden stays clear: the sack race runs there)
	pl.at("par_metro", 252, 336, 0) // on the walk: the gardens stay clear for the sack race
	pl.at("flagpole", 380, 296, flagFrance)
	// bistro tables with wicker chairs and the bouquinistes, along the shopfronts at both ends (the middle of the top
	// stays clear: the sack race runs there)
	pl.at("par_bouquiniste", 64, 136, 0)
	pl.at("par_bistro", 120, 138, 0)
	pl.at("par_bistro", 840, 138, 1)
	pl.at("par_bouquiniste", 912, 124, 1)
	// the green kiosk, the bakery stall, the accordion player and the mime
	pl.at("par_kiosk", 900, 236, 0)
	pl.at("par_bakery", 586, 384, rng.Intn(2))
	pl.at("par_busker", 690, 318, 0)
	pl.at("par_busker", 334, 418, 1)
	return ws
}

// parLawn: the middle of the top right garden. The guests' wandering (and so the café sets and the items that are
// laid where people roam) leaves it out, so the sack race always finds its course there; people still walk across it.
var parLawn = R(548, 204, 280, 92)

// parisRoam: the square's wandering area without the lawn of the sack race (the first rect is the square).
func parisRoam(roam []Rect) []Rect {
	sq := roam[0]
	out := []Rect{
		R(sq.X, sq.Y, sq.W, parLawn.Y-sq.Y),                                         // the top, along the shopfronts
		R(sq.X, parLawn.Y, parLawn.X-sq.X, parLawn.H),                               // left of the lawn
		R(parLawn.X+parLawn.W, parLawn.Y, sq.X+sq.W-parLawn.X-parLawn.W, parLawn.H), // right of the lawn
		R(sq.X, parLawn.Y+parLawn.H, sq.W, sq.Y+sq.H-parLawn.Y-parLawn.H),           // the bottom half
	}
	return append(out, roam[1:]...)
}

// parFlagSpot: the anchor of the French flag on the backdrop (a balcony of the building on the right of the avenue).
var parFlagSpot = Vec{744, 44}

// ---- the painted backdrop (the floor "paris", draw.go renderBG) -------------------------------------------------

var (
	parBackBuf *pixBuf
	parBackImg *ebiten.Image
)

// drawParisFloor paints the part r of the backdrop picture.
func drawParisFloor(bg *ebiten.Image, r Rect) {
	if parBackImg == nil {
		parBackImg = parisBackdrop().img()
	}
	x0, y0 := int(math.Max(0, r.X)), int(math.Max(0, r.Y))
	x1, y1 := int(math.Min(960, r.X+r.W)), int(math.Min(parBackH, r.Y+r.H))
	if x1 <= x0 || y1 <= y0 {
		return
	}
	sub := parBackImg.SubImage(image.Rect(x0, y0, x1, y1)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x0), float64(y0))
	bg.DrawImage(sub, op)
}
