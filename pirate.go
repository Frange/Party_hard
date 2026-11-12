package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// BARCO PIRATA (campaign party 25, right before the space station): a pirate party on board a galleon anchored in a
// bay at sunset. The deck is the party: the raised quarterdeck (the stern, left) with the helm, the captain's table
// and the fiddler, the main deck with the DJ, the main mast, cannons along both rails, the rum and the hatch, and the
// forecastle (the bow, right) with the anchor and a treasure. Stairs lead up to both castles. A gangplank goes down to
// a small wooden dock where every boat ties up (guests, police, ambulance...: Transit "sea"). Everything else is open
// sea: solid (Shape + voidWalls), nobody walks on the water. Dusk light: a cool mauve wash over warm accents, never the
// golden / yellow cast the user rejected (Day + Tint, checked by pirate_test.go). The sea, the hull, the planks and
// the dock are painted once by code (pirateScene, drawn through the floors "bay" and "galleon": draw.go renderBG).
// Props: pirate_props.go; texts and tables: pirate_text.go; the icon of the list: pirate_icon.go.
// ---------------------------------------------------------------------------

const (
	pirateTitle = "BARCO PIRATA"

	pirX0   = 104.0 // the stern (left end of the deck)
	pirX1   = 720.0 // where the bow starts to narrow
	pirTip  = 884.0 // the tip of the bow
	pirTop  = 152.0 // the north rail
	pirBot  = 400.0 // the south rail
	pirQD   = 258.0 // the rail of the quarterdeck (the raised stern), 8 px thick
	pirFC   = 700.0 // the rail of the forecastle (the raised bow), 8 px thick
	pirRail = 8.0
	pirSea  = 40.0 // the painted bay starts under the horizon of the sunset sky (draw.go Sky "sea")
)

var (
	pirPlank = Rect{464, 400, 32, 56}  // the gangplank, from the south rail down to the dock
	pirDock  = Rect{328, 456, 320, 48} // the little dock where the boats tie up
	// the stairs: openings in the rails of the two castles (y ranges)
	pirQDStairs = [2][2]float64{{176, 208}, {344, 376}}
	pirFCStairs = [2][2]float64{{184, 216}, {336, 368}}
	pirHatch    = Rect{524, 314, 72, 40} // the grating of the main hatch (walkable)
)

// pirateMid: the centre line of the ship.
const pirateMid = (pirTop + pirBot) / 2

// pirateDeck: is (x,y) on the deck of the galleon grown by g px (g may be negative)?
func pirateDeck(x, y, g float64) bool {
	l, t, b := pirX0-g, pirTop-g, pirBot+g
	if x < l || y < t || y > b {
		return false
	}
	rad := math.Max(1, 20+g) // the rounded corners of the stern
	if x < l+rad && (y < t+rad || y > b-rad) {
		dx := l + rad - x
		dy := t + rad - y
		if y > b-rad {
			dy = y - (b - rad)
		}
		return dx*dx+dy*dy <= rad*rad
	}
	if x <= pirX1 {
		return true
	}
	if x > pirTip+g {
		return false
	}
	return math.Abs(y-pirateMid) <= pirateHalf(x)+g
}

// pirateHalf: half the width of the deck at x (the bow narrows to a point at pirTip).
func pirateHalf(x float64) float64 {
	if x <= pirX1 {
		return (pirBot - pirTop) / 2
	}
	k := (x - pirX1) / (pirTip - pirX1)
	if k >= 1 {
		return 0
	}
	return (pirBot - pirTop) / 2 * math.Pow(1-k, 0.8)
}

// pirateShape: the walkable outline of the party (deck, gangplank and dock); everything outside is the sea.
func pirateShape(x, y, g float64) bool {
	return pirateDeck(x, y, g) || pirPlank.Grow(g).Has(x, y) || pirDock.Grow(g).Has(x, y)
}

// pirateBay: is this level the pirate party (its random-mode roll too)?
func pirateBay(def *LevelDef) bool { return def != nil && def.Title == pirateTitle }

// pirateRails: the invisible solid strips of the two castle rails, open at the stairs (the rails themselves are
// painted in the background). Plain walls: no door is drawn in the openings.
func pirateRails() []Wall {
	var out []Wall
	add := func(x float64, gaps [2][2]float64) {
		y := pirTop
		for _, gp := range gaps {
			out = append(out, Wall{R(x, y, pirRail, gp[0]-y), WallVoid})
			y = gp[1]
		}
		out = append(out, Wall{R(x, y, pirRail, pirBot-y), WallVoid})
	}
	add(pirQD, pirQDStairs)
	add(pirFC, pirFCStairs)
	return out
}

func genPirate(rng *rand.Rand) *LevelDef {
	doorLog = nil
	d := &LevelDef{
		Title: pirateTitle, Intro: "Ron, cañones y un capitán que no sabe nadar.",
		W: 960, H: 540, Sky: "sea", Police: "car", Transit: "sea", Day: true, Tint: color.RGBA{30, 28, 58, 36},
		CarFrom: Vec{1040, 528}, CarTo: Vec{600, 528}, ParkX0: 400, ParkX1: 600, LaneOff: 6,
		Start: Vec{548, 376}, Guests: 44 + rng.Intn(6), CellSize: 16, Shape: pirateShape,
	}
	d.Floors = []Floor{
		F(0, pirSea, 960, 540-pirSea, "bay"),
		{R: Rect{pirX0, pirTop, pirTip - pirX0, pirBot - pirTop}, Kind: "galleon"},
		{R: pirPlank, Kind: "galleon"},
		{R: pirDock, Kind: "galleon"},
	}
	d.Walls = append(pirateRails(), voidWalls(pirateShape, d.W, d.H)...)

	var props []PropInst
	pl := &genPlacer{rng: rng, out: &props}
	jx := func(n int) float64 { return float64(rng.Intn(2*n+1) - n) } // a few pixels of jitter for the loose things

	// the quarterdeck: the helm, the captain's table, the mizzen mast, the fiddler, a parrot and a chest to hide in
	pl.at("phelm", 150, 292, 0)
	pl.at("pcaptable", 196, 220, 0)
	pl.at("pmast", 206, 368, 2) // furled sails
	pl.at("pparrot", 236, 296, 0)
	pl.at("pchest", 132, 248+jx(4), rng.Intn(2))
	pl.at("plantern", 128, 176, 0)
	pl.at("plantern", 128, 388, 0)

	// the main deck: the DJ, the main mast with its sail and the flag, cannons on both rails, rum and barrels
	pl.at("dj", 340, 206, 0)
	pl.at("speaker", 286, 204, 0)
	pl.at("speaker", 394, 204, 0)
	pl.at("pmast", 480, 296, 0) // the main mast: the big sail and the skull flag
	pl.at("pmast", 640, 268, 2)
	for _, x := range []float64{456, 560, 664} {
		pl.at("pcannon", x, 176, 0) // pointing out over the north rail
	}
	for _, x := range []float64{300, 396, 588, 668} {
		pl.at("pcannonr", x, 398, 1) // pointing out over the south rail (run out: no room behind it to light it)
	}
	pl.at("prum", 600, 222, rng.Intn(2))
	pl.at("pbarrel", 290, 312+jx(4), 0)
	pl.at("pbarrel", 430, 352, 1)
	pl.at("pcoil", 404, 304+jx(3), 0)
	pl.at("pcoil", 664, 334, 1)
	pl.at("plantern", 278, 162, 0)
	pl.at("plantern", 690, 162, 0)

	// the forecastle: the anchor, a chest, a lantern and the treasure at the bow
	pl.at("panchor", 756, 268, 0)
	pl.at("pchest", 744, 326, rng.Intn(2))
	pl.at("ptreasure", 826, 290, 0)
	pl.at("plantern", 818, 238, 0)

	// the dock: crates, a barrel, mooring posts and a lantern
	pl.at("crate", 346, 482, 0)
	pl.at("pbarrel", 374, 480, 0)
	pl.at("plantern", 634, 474, 0)

	d.Props = props
	d.Performers = []Perf{{Vec{340, 192}, "dj", DirDown}, {Vec{140, 214}, "guitar", DirRight}}
	d.Roam = []Rect{
		R(280, 228, 400, 160), // the main deck
		R(118, 300, 128, 70),  // the quarterdeck
		R(716, 220, 64, 100),  // the forecastle
		R(392, 464, 220, 32),  // the dock
		R(480, 168, 200, 40),  // along the north rail
	}
	d.Dance = []Rect{R(290, 228, 150, 56)}
	d.Guards = [][]Vec{{{300, 350}, {660, 350}}, {{140, 316}, {240, 316}}}
	d.Doors = append(d.Doors, doorLog...)
	return d
}

// ---------------------------------------------------------------------------
// the painted bay, hull, decks and dock (one picture, drawn by the floors "bay" and "galleon")
// ---------------------------------------------------------------------------

var (
	pirateSceneBuf *pixBuf
	pirateSceneImg *ebiten.Image
)

func phHash(a, b int) int {
	h := uint32(a)*374761393 + uint32(b)*668265263 + 0x9e3779b9
	h = (h ^ (h >> 13)) * 1274126177
	return int((h ^ (h >> 16)) & 0x7fffffff)
}

// pirate palette: dusk light, warm accents kept low (wood is a muted brown, the water mauve-blue)
var (
	pirSeaTop  = rgb(0x6c5a86) // the bay near the horizon, lit by the sky
	pirSeaDeep = rgb(0x1c2a46) // in front, deep water
	pirSun     = rgb(0xd88068) // the reflection of the low sun (rosy, not golden)
	pirPlankA  = rgb(0x6e5446)
	pirPlankB  = rgb(0x66503f)
	pirSeam    = rgb(0x392820)
	pirCastle  = rgb(0x7a5e4c)
	pirCastleB = rgb(0x725846)
	pirHullC   = rgb(0x4a3024)
	pirHullD   = rgb(0x2e1c16)
	pirRailC   = rgb(0x8a6a50)
	pirDockA   = rgb(0x7a6c5e)
	pirDockB   = rgb(0x6e6254)
)

// pirateScene paints the whole bay once (960x540; the rows above pirSea stay empty: the sunset sky of Sky "sea").
func pirateScene() *pixBuf {
	if pirateSceneBuf != nil {
		return pirateSceneBuf
	}
	b := newPix(960, 540)
	for y := int(pirSea); y < 540; y++ {
		for x := 0; x < 960; x++ {
			b.set(x, y, pirateSeaAt(x, y))
		}
	}
	pirateHull(b)
	pirateDecks(b)
	pirateDockArt(b)
	pirateBowsprit(b)
	pirateRowboat(b, 236, 474)
	pirateSceneBuf = b
	return b
}

// pirateSeaAt: the colour of the open water at (x,y): a gradient from the lit horizon to the deep front, the rosy
// column of the sun's reflection under the sun (x 780), ripples, and the darker water in the shadow of the ship.
func pirateSeaAt(x, y int) color.RGBA {
	t := math.Pow(clampF((float64(y)-pirSea)/300, 0, 1), 0.7)
	c := mixc(pirSeaTop, pirSeaDeep, t)
	// the sun's reflection: broken streaks that narrow towards the front
	if y < 150 {
		w := 44 - float64(y-int(pirSea))*0.28
		if dx := math.Abs(float64(x) - seaSunX); dx < w && (y/2)%3 != 0 && phHash(x/6, y/2)%5 < 3 {
			c = mixc(c, pirSun, 0.55*(1-dx/w)*(1-float64(y-int(pirSea))/110))
		}
	}
	// ripples: short light and dark dashes
	if y%4 == 0 {
		h := phHash(x/9, y)
		if h%7 == 0 {
			c = mixc(c, rgb(0xa8a0c8), 0.35*(1-t*0.6))
		} else if h%7 == 1 {
			c = shade(c, 0.86)
		}
	}
	// the shadow of the hull on the water (south and east of the ship)
	fx, fy := float64(x), float64(y)
	if pirateDeck(fx-6, fy-40, 2) && !pirateDeck(fx, fy, 40) {
		c = shade(c, 0.78)
	}
	// foam along the waterline
	if !pirateDeck(fx, fy-38, 1) && pirateDeck(fx, fy-40, 1) && phHash(x, y)%3 != 0 {
		c = mixc(c, rgb(0xd8d4e8), 0.6)
	}
	return c
}

// pirateHull: the bulwarks around the deck and, on the south side and round the bow, the side of the hull with its
// gun ports, the wales and the anchor hanging by the bow.
func pirateHull(b *pixBuf) {
	for y := int(pirTop) - 14; y < int(pirBot)+42; y++ {
		for x := int(pirX0) - 12; x < int(pirTip)+14; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if pirateDeck(fx, fy, 0) {
				continue
			}
			switch {
			case pirateDeck(fx, fy, 3): // the cap rail on top of the bulwark
				b.set(x, y, pirRailC)
			case pirateDeck(fx, fy, 8): // the bulwark
				cc := pirHullC
				if fy < pirateMid { // the north bulwark: we see its inner face
					cc = rgb(0x5e4232)
					if x%12 == 0 {
						cc = pirHullD
					}
				}
				b.set(x, y, cc)
			case fy > pirateMid && pirateDeck(fx, fy-34, 8): // the side of the hull below the south rail
				d := fy - 8
				for !pirateDeck(fx, d, 8) && d > pirateMid {
					d--
				}
				k := fy - d // depth below the rail
				cc := pirHullC
				switch {
				case int(k)%7 == 0:
					cc = pirHullD
				case k > 26:
					cc = rgb(0x24161a) // the dark band on the waterline
				case k > 10 && k < 14:
					cc = rgb(0x6a4a34) // the wale
				}
				b.set(x, y, cc)
			case fy < pirateMid && pirateDeck(fx, fy+6, 8): // a sliver of the north side
				b.set(x, y, pirHullD)
			}
		}
	}
	// the gun ports on the south side, with the muzzles of the cannons below deck
	for x := 292; x < 700; x += 44 {
		if x > 440 && x < 520 {
			continue // the gangplank
		}
		y := int(pirBot) + 16
		b.rect(x, y, 9, 7, rgb(0x8a2a24))
		b.rect(x+1, y+1, 7, 5, rgb(0x120c0e))
		b.ellipse(float64(x)+4.5, float64(y)+3.5, 2.2, 2.2, rgb(0x2a2a30))
		b.rect(x, y-2, 9, 1, rgb(0x8a2a24))
	}
	// the stern windows (the captain's cabin) lit from inside: warm but dim
	for i := 0; i < 3; i++ {
		y := int(pirBot) + 14
		x := int(pirX0) + 10 + i*14
		b.rect(x, y, 8, 8, rgb(0x2a1a14))
		b.rect(x+1, y+1, 6, 6, rgb(0xb07850))
		b.rect(x+1, y+4, 6, 1, rgb(0x2a1a14))
		b.rect(x+4, y+1, 1, 6, rgb(0x2a1a14))
	}
	// the anchor hanging by the bow, on its chain
	ax, ay := 772, int(pirateMid+pirateHalf(772))+10
	for y := ay; y < ay+22; y += 2 {
		b.set(ax, y, rgb(0x6a6a74))
	}
	anchorArt(b, ax-8, ay+18, rgb(0x3a3a44), rgb(0x6a6a78))
}

// anchorArt paints a 17x16 anchor with its top-left at (x,y).
func anchorArt(b *pixBuf, x, y int, body, light color.RGBA) {
	b.rect(x+6, y, 5, 3, body) // the ring
	b.set(x+8, y+1, rgb(0x1a1418))
	b.rect(x+7, y+3, 3, 11, body) // the shank
	b.rect(x+3, y+4, 11, 2, body) // the stock
	b.rect(x+7, y+3, 1, 10, light)
	for i := 0; i < 6; i++ { // the arms
		b.rect(x+7-i, y+13-i/2, 2, 2, body)
		b.rect(x+8+i, y+13-i/2, 2, 2, body)
	}
	b.rect(x, y+9, 3, 3, body) // the flukes
	b.rect(x+14, y+9, 3, 3, body)
}

// pirateDecks: the planks of the main deck, the two raised castles with their rails and stairs, and the hatch.
func pirateDecks(b *pixBuf) {
	for y := int(pirTop); y <= int(pirBot); y++ {
		for x := int(pirX0); x <= int(pirTip); x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if !pirateDeck(fx, fy, 0) {
				continue
			}
			castle := fx < pirQD || fx >= pirFC+pirRail
			row := (y - int(pirTop)) / 6
			a, bb := pirPlankA, pirPlankB
			if castle {
				a, bb = pirCastle, pirCastleB
			}
			cc := a
			if row%2 == 1 {
				cc = bb
			}
			if (y-int(pirTop))%6 == 5 {
				cc = pirSeam // the seam between two planks
			} else if (x+row*37)%86 == 0 {
				cc = pirSeam // the butt joint of a plank
			} else if (x+row*37)%86 == 3 && (y-int(pirTop))%6 == 2 {
				cc = rgb(0x8a7a6a) // the nails
			} else if phHash(x/3, y)%23 == 0 {
				cc = shade(cc, 0.9) // the grain
			}
			// the castles throw a shadow on the main deck
			if !castle && ((fx >= pirQD+pirRail && fx < pirQD+pirRail+5) || (fx < pirFC && fx >= pirFC-3)) {
				cc = shade(cc, 0.78)
			}
			b.set(x, y, cc)
		}
	}
	// the rails of the castles: turned balusters under a cap rail, open at the stairs
	rail := func(x int, gaps [2][2]float64) {
		for y := int(pirTop); y < int(pirBot); y++ {
			open := false
			for _, gp := range gaps {
				if float64(y) >= gp[0] && float64(y) < gp[1] {
					open = true
				}
			}
			if open {
				// the steps: light treads and dark risers going down towards the main deck
				for i := 0; i < int(pirRail); i++ {
					cc := pirCastle
					if i%3 == 2 {
						cc = pirSeam
					}
					b.set(x+i, y, cc)
				}
				continue
			}
			// the cap rail seen from above: a dark edge, the lit top of the beam and its shadow on the main deck
			b.set(x, y, pirHullD)
			b.rect(x+1, y, 5, 1, pirRailC)
			b.set(x+1, y, shade(pirRailC, 1.15))
			b.set(x+6, y, pirHullC)
			b.set(x+7, y, shade(pirPlankB, 0.7))
			if (y-int(pirTop))%24 < 3 {
				b.rect(x, y, 7, 1, shade(pirRailC, 0.85)) // a post under the rail
			}
		}
		for _, gp := range gaps { // the newel posts beside every opening
			for _, yy := range []int{int(gp[0]) - 3, int(gp[1])} {
				b.rect(x-1, yy, int(pirRail)+2, 3, pirRailC)
				b.rect(x-1, yy+2, int(pirRail)+2, 1, pirHullD)
			}
		}
	}
	rail(int(pirQD), pirQDStairs)
	rail(int(pirFC), pirFCStairs)
	// the main hatch: a dark grating with its wooden coaming
	h := pirHatch
	b.rect(int(h.X), int(h.Y), int(h.W), int(h.H), rgb(0x5a4030))
	b.rect(int(h.X)+3, int(h.Y)+3, int(h.W)-6, int(h.H)-6, rgb(0x160e0c))
	for y := int(h.Y) + 4; y < int(h.Y+h.H)-3; y += 5 {
		b.rect(int(h.X)+3, y, int(h.W)-6, 2, rgb(0x6e5442))
	}
	for x := int(h.X) + 5; x < int(h.X+h.W)-3; x += 6 {
		b.rect(x, int(h.Y)+3, 2, int(h.H)-6, rgb(0x6e5442))
	}
	b.rect(int(h.X), int(h.Y), int(h.W), 1, rgb(0x8a6a50))
	// the round partners where the masts go through the deck are part of the masts (pirate_props.go)
}

// pirateDockArt: the gangplank (with its cleats and side ropes) and the little dock on its piles.
func pirateDockArt(b *pixBuf) {
	p := pirPlank
	for y := int(p.Y); y < int(p.Y+p.H); y++ {
		for x := int(p.X); x < int(p.X+p.W); x++ {
			cc := pirDockA
			if (x-int(p.X))%8 == 7 {
				cc = pirSeam
			}
			if (y-int(p.Y))%8 == 0 {
				cc = rgb(0x4a382c) // the cleats
			}
			b.set(x, y, cc)
		}
	}
	for y := int(p.Y) - 4; y < int(p.Y+p.H); y++ { // the hand ropes on both sides
		sag := int(2 * math.Sin(math.Pi*float64(y-int(p.Y)+4)/(p.H+4)))
		b.set(int(p.X)-2-sag, y, rgb(0xb8a080))
		b.set(int(p.X+p.W)+1+sag, y, rgb(0xb8a080))
	}
	d := pirDock
	for y := int(d.Y); y < int(d.Y+d.H); y++ {
		for x := int(d.X); x < int(d.X+d.W); x++ {
			cc := pirDockA
			if ((x-int(d.X))/7)%2 == 1 {
				cc = pirDockB
			}
			if (x-int(d.X))%7 == 6 {
				cc = rgb(0x3a3028) // the gaps between the boards
			} else if phHash(x, y/3)%31 == 0 {
				cc = shade(cc, 0.88)
			}
			b.set(x, y, cc)
		}
	}
	b.rect(int(d.X), int(d.Y+d.H), int(d.W), 3, rgb(0x3a3028)) // the edge of the boards seen from the south
	for x := int(d.X); x <= int(d.X+d.W)-6; x += 52 {          // the piles and their ripples
		b.rect(x, int(d.Y+d.H), 6, 7, rgb(0x4a3a2e))
		b.rect(x, int(d.Y+d.H)+7, 6, 1, rgb(0xb0a8c8))
		b.rect(x-2, int(d.Y+d.H)+9, 10, 1, rgba(0xb0a8c8, 140))
	}
	// a bollard at each end with a mooring rope
	for _, x := range []int{int(d.X) + 4, int(d.X+d.W) - 10} {
		b.rect(x, int(d.Y)+4, 6, 6, rgb(0x2a2a30))
		b.rect(x+1, int(d.Y)+4, 4, 1, rgb(0x6a6a74))
	}
}

// pirateBowsprit: the spar sticking out of the bow, its stays and the furled jib, over the water.
func pirateBowsprit(b *pixBuf) {
	x0, y0 := int(pirTip)-10, int(pirateMid)-2
	for i := 0; i < 64; i++ {
		x, y := x0+i, y0-i/3
		b.rect(x, y, 1, 3, rgb(0x5a3a26))
		b.set(x, y, rgb(0x8a6040))
	}
	for i := 0; i < 56; i += 2 { // the stays down to the bow
		b.set(x0+56-i, y0-18+i*14/56+8, rgb(0x2a1e18))
	}
	for i := 0; i < 26; i++ { // the furled jib along the spar
		b.rect(x0+8+i, y0-4-(8+i)/3, 1, 3, rgb(0xd0c4b0))
	}
}

// pirateRowboat: a small boat moored by the dock (at its centre x,y), oars shipped.
func pirateRowboat(b *pixBuf, x, y int) {
	for i := -18; i <= 18; i++ {
		h := int(6 * math.Sqrt(1-float64(i*i)/(19*19)))
		b.rect(x+i, y-h, 1, 2*h, rgb(0x5a3a26))
		if h > 2 {
			b.rect(x+i, y-h+2, 1, 2*h-4, rgb(0x8a6a4a))
		}
	}
	for _, dx := range []int{-8, 0, 8} {
		b.rect(x+dx, y-4, 2, 8, rgb(0x5a3a26)) // the thwarts
	}
	b.rect(x-14, y-1, 28, 1, rgb(0xc8b090)) // the oars
	b.rect(x-20, y+7, 40, 1, rgba(0xd8d4e8, 120))
	for i := 0; i < 20; i++ { // the painter to the dock
		b.set(x+18+i*4/2, y-i/4, rgb(0xb8a080))
	}
}

// drawPirateFloor paints the part r of the bay picture (the floors "bay" and "galleon", draw.go renderBG).
func drawPirateFloor(bg *ebiten.Image, r Rect) {
	if pirateSceneImg == nil {
		pirateSceneImg = pirateScene().img()
	}
	x0, y0 := int(math.Max(0, r.X)), int(math.Max(0, r.Y))
	x1, y1 := int(math.Min(960, r.X+r.W)), int(math.Min(540, r.Y+r.H))
	if x1 <= x0 || y1 <= y0 {
		return
	}
	sub := pirateSceneImg.SubImage(image.Rect(x0, y0, x1, y1)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x0), float64(y0))
	bg.DrawImage(sub, op)
}
