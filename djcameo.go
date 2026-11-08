package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The DJ cameo of the good ending: a giant booth on the stage of the party.
//
// All of them are PARODIES with invented names (a silhouette and a gag, no real faces, no brands):
//   Gabbo Pontazo      the funny Italian MC: enormous glasses, stripes in the colours of his flag, jumps about, a spoon
//   David Guette     very tall, black tee, both arms in the air
//   Martín Garbanzo  a kid with a cap
//   Avichi           bearded, headphones round the neck
//   Daft Pank        two robots in helmets, one of them is the turntable
//   Ratón Neón        a mouse head full of lights
// Gabbo never leaves the booth; the other three places rotate. Act 3: countdown and DROP (lights and confetti).
// Everything is drawn with rectangles on the small canvas, and costs almost nothing.
// ---------------------------------------------------------------------------

const (
	djDropAt   = 24.0 // the DROP of act 3
	djRotate   = 2.5  // seconds each DJ keeps his place
	djBoothTop = 172.0
)

type djKind int

const (
	djGabbo djKind = iota
	djDavid
	djMartin
	djAvichi
	djDaft
	djMouse
	djCount
)

// the invented names, as written on the nameplate
var djNames = [djCount]string{"Gabbo Pontazo", "David Guette", "Martín Garbanzo", "Avichi", "Daft Pank", "Ratón Neón"}

// the four places of the booth (x on the small canvas); Gabbo owns the second one
var djSlotX = [4]float64{198, 226, 256, 284}

const djGabboSlot = 1

// djAt: who stands in place s at time t.
func djAt(s int, t float64) djKind {
	if s == djGabboSlot {
		return djGabbo
	}
	k := s
	if s > djGabboSlot {
		k = s - 1
	}
	pool := int(t/djRotate) + k // three rotating places over the five guests
	return djKind(1 + pool%5)
}

// djStar: the DJ of the nameplate (the one in place t/djRotate, round the booth).
func djStar(t float64) djKind { return djAt(int(t/djRotate)%4, t) }

// djNameplate: the text above the stage during acts 1 and 3.
func djNameplate(t float64) string {
	if (t >= 0.9 && t < 7.6) || (t >= 21 && t < djDropAt) {
		return "DJ " + djNames[djStar(t)]
	}
	return ""
}

// djDropU: seconds since the DROP, or -1 before it.
func djDropU(t float64) float64 {
	if t < djDropAt {
		return -1
	}
	return t - djDropAt
}

func djArmsUp(t float64) bool { return t >= 22.8 && t < 27.8 } // the build-up and the party after the DROP

// partyBooth draws the truss, the LED screen, the DJs and the table. It replaces the old small deck.
func (g *Game) partyBooth(c *ebiten.Image, t float64) {
	recLayer("deco:djbooth")
	n := partyNight(t)
	// truss with its lights
	fillRect(c, 178, 126, 2, 60, rgb(0x6a6a7a))
	fillRect(c, 302, 126, 2, 60, rgb(0x6a6a7a))
	fillRect(c, 176, 124, 130, 3, rgb(0x8a8a9a))
	for q := 0; q < 7; q++ {
		lc := partyColors[(q+int(t*3))%len(partyColors)]
		x := 186 + float64(q)*18
		fillRect(c, x, 127, 5, 4, rgb(0x2a2a34))
		fillRect(c, x+1, 131, 3, 1, rgb(lc))
		if n > 0.1 {
			g.preGlow(c, x+2, 132, 9, color.RGBA{uint8(lc >> 16), uint8(lc >> 8), uint8(lc), uint8(70 * n)})
		}
	}
	// the LED screen: an equalizer
	fillRect(c, 188, 134, 104, 36, rgb(0x0c0a1e))
	fillRect(c, 188, 134, 104, 1, rgb(0x3a3a5a))
	fast := 1.0
	if djArmsUp(t) {
		fast = 2.2
	}
	for q := 0; q < 17; q++ {
		h := 3 + 24*math.Abs(math.Sin(t*5*fast+float64(q)*0.9)*math.Sin(t*2.3+float64(q)))
		fillRect(c, 190+float64(q)*6, 168-math.Round(h), 4, math.Round(h), rgb(partyColors[(q+int(t*2))%len(partyColors)]))
	}
	// the DJs, behind the table
	for s := 0; s < 4; s++ {
		g.djDraw(c, djAt(s, t), djSlotX[s], 176, t, float64(s))
	}
	// the table: top, front with a light strip and a big "DJ"
	fillRect(c, 184, djBoothTop, 112, 5, rgb(0x4a4a58))
	fillRect(c, 184, djBoothTop, 112, 1, rgb(0x7a7a8c))
	fillRect(c, 184, djBoothTop+5, 112, 11, rgb(0x24242e))
	for q := 0; q < 28; q++ {
		on := (q+int(t*6))%3 == 0
		lc := rgb(partyColors[q%len(partyColors)])
		if !on {
			lc = shade(lc, 0.45)
		}
		fillRect(c, 186+float64(q)*4, djBoothTop+14, 3, 2, lc)
	}
	// decks and mixer on the top
	for _, dx := range []float64{204, 276} {
		filledEllipse(c, dx, djBoothTop+2, 9, 2, rgb(0x15151b))
		filledEllipse(c, dx, djBoothTop+2, 6, 1, rgb(0x9a9aaa))
		fillRect(c, math.Round(dx+4*math.Cos(t*6)), djBoothTop+2, 2, 1, rgb(0xff4a6a))
	}
	fillRect(c, 232, djBoothTop-1, 16, 4, rgb(0x666678))
	for q := 0; q < 4; q++ {
		fillRect(c, 234+float64(q)*4, djBoothTop-2, 1, 2, rgb(0xf0c030))
	}
	g.djLetters(c, 240, djBoothTop+7, t)
}

// djLetters: "DJ" in the front of the table, in blocks of 3 px.
func (g *Game) djLetters(c *ebiten.Image, cx, y, t float64) {
	col := rgb(partyColors[int(t*4)%len(partyColors)])
	for _, r := range [][4]float64{{-8, 0, 2, 7}, {-8, 0, 5, 2}, {-8, 5, 5, 2}, {-4, 1, 2, 5}, // D
		{3, 0, 2, 6}, {-1, 5, 6, 2}, {-1, 4, 2, 3}} { // J
		fillRect(c, cx+r[0], y+r[1], r[2], r[3], col)
	}
}

// djDraw: one DJ with his feet at y=fy (the table hides the legs).
func (g *Game) djDraw(c *ebiten.Image, k djKind, x, fy, t, ph float64) {
	x = math.Round(x)
	up := djArmsUp(t)
	beat := math.Abs(math.Sin(t*4.6 + ph))
	hop := math.Round(2 * beat)
	if k == djGabbo {
		hop = math.Round(6 * math.Abs(math.Sin(t*6)))
	}
	skin := rgb(0xf0c8a0)
	wave := func(side float64, handY float64) { // an arm going up to handY
		ax := x + side*5
		fillRect(c, ax-1, handY, 2, fy-hop-12-handY, skin)
		fillRect(c, ax-1, handY-2, 3, 3, skin)
	}
	hy := func(up bool, off float64) float64 { // hand height: arms in the air, or half way
		if up {
			return fy - hop - 34 - 3*math.Sin(t*9+off)
		}
		return fy - hop - 20 - 2*math.Sin(t*5+off)
	}
	switch k {
	case djGabbo:
		// the funny Italian: flag-coloured stripes, curly hair, a moustache and enormous glasses
		for i, sc := range []uint32{0x2a9a4a, 0xf4f4f0, 0xd83a3a, 0xf4f4f0, 0x2a9a4a, 0xf4f4f0} {
			fillRect(c, x-5, fy-hop-18+float64(i)*3, 10, 3, rgb(sc))
		}
		wave(-1, hy(true, 0))
		// the other arm holds a giant spoon (to lower the bass)
		sx := x + 6
		fillRect(c, sx-1, fy-hop-30-3*math.Sin(t*6), 2, 14, skin)
		fillRect(c, sx, fy-hop-38-3*math.Sin(t*6), 1, 9, rgb(0xb0b0c0))
		filledEllipse(c, sx+0.5, fy-hop-42-3*math.Sin(t*6), 3, 4, rgb(0xd8d8e4))
		fillRect(c, x-4, fy-hop-27, 8, 9, skin)
		fillRect(c, x-5, fy-hop-30, 10, 4, rgb(0x2a1a14)) // curls
		fillRect(c, x-6, fy-hop-28, 2, 3, rgb(0x2a1a14))
		fillRect(c, x+4, fy-hop-28, 2, 3, rgb(0x2a1a14))
		for _, gx := range []float64{-4, 0} { // glasses, too big for his face
			fillRect(c, x+gx-1, fy-hop-27, 6, 6, rgb(0x15151b))
			fillRect(c, x+gx, fy-hop-26, 4, 4, rgb(0xf4f8ff))
			fillRect(c, x+gx+1, fy-hop-25+float64(int(t*3)%2), 2, 2, rgb(0x15151b))
		}
		fillRect(c, x-3, fy-hop-20, 7, 2, rgb(0x2a1a14)) // moustache
	case djDavid:
		// very tall: black tee, dark hair, both arms up
		fillRect(c, x-5, fy-hop-24, 10, 24, rgb(0x1c1c24))
		fillRect(c, x-3, fy-hop-34, 7, 9, skin)
		fillRect(c, x-4, fy-hop-37, 9, 4, rgb(0x2a2018))
		fillRect(c, x-2, fy-hop-31, 1, 1, rgb(0x15151b))
		fillRect(c, x+2, fy-hop-31, 1, 1, rgb(0x15151b))
		wave(-1, fy-hop-46-3*math.Sin(t*7))
		wave(1, fy-hop-46-3*math.Sin(t*7+1.6))
	case djMartin:
		// the kid with the cap on backwards and an orange hoodie
		fillRect(c, x-5, fy-hop-18, 10, 18, rgb(0xe8782a))
		fillRect(c, x-3, fy-hop-26, 7, 8, rgb(0xf4d4b0))
		fillRect(c, x-4, fy-hop-28, 9, 3, rgb(0x2a6ac8))
		fillRect(c, x+2, fy-hop-27, 4, 1, rgb(0x2a6ac8)) // the peak, at the back
		fillRect(c, x-2, fy-hop-23, 1, 1, rgb(0x15151b))
		fillRect(c, x+1, fy-hop-23, 1, 1, rgb(0x15151b))
		wave(-1, hy(up, 0.5))
		wave(1, hy(true, 2))
	case djAvichi:
		// the bearded Swede: headphones round the neck
		fillRect(c, x-5, fy-hop-19, 10, 19, rgb(0xe8e8ec))
		fillRect(c, x-3, fy-hop-27, 7, 8, skin)
		fillRect(c, x-4, fy-hop-29, 9, 3, rgb(0xd8b060))
		fillRect(c, x-3, fy-hop-22, 7, 4, rgb(0xb8682a)) // the beard
		fillRect(c, x-2, fy-hop-25, 1, 1, rgb(0x15151b))
		fillRect(c, x+1, fy-hop-25, 1, 1, rgb(0x15151b))
		fillRect(c, x-6, fy-hop-19, 12, 2, rgb(0x15151b)) // headphones
		fillRect(c, x-6, fy-hop-19, 2, 4, rgb(0x15151b))
		fillRect(c, x+4, fy-hop-19, 2, 4, rgb(0x15151b))
		wave(-1, hy(up, 1))
		wave(1, hy(up, 3))
	case djDaft:
		// two robots in helmets; the second one is the turntable (a vinyl on his belly)
		for i, off := range []float64{-6, 6} {
			rx := x + off
			hc := []uint32{0xc8ccd8, 0xe0b830}[i]
			fillRect(c, rx-3, fy-hop-17, 7, 17, rgb(0x2a2a34))
			fillRect(c, rx-4, fy-hop-26, 9, 9, rgb(hc))
			fillRect(c, rx-3, fy-hop-24, 7, 4, rgb(0x15151b))
			fillRect(c, rx-3+float64(int(t*4+float64(i))%5), fy-hop-23, 2, 2, rgb(partyColors[(int(t*3)+i)%len(partyColors)]))
			fillRect(c, rx-1, fy-hop-28, 3, 2, rgb(0x6a6a7a))
		}
		filledEllipse(c, x+6, fy-hop-8, 4, 4, rgb(0x15151b)) // the vinyl
		fillRect(c, math.Round(x+6+3*math.Cos(t*8)), math.Round(fy-hop-8+3*math.Sin(t*8)), 1, 1, rgb(0xff4a6a))
		fillRect(c, x-12, hy(up, 0)-1, 2, 12, rgb(0x6a6a7a)) // the arm scratching
	case djMouse:
		// the mouse head: round ears, glowing eyes, a smile that changes colour
		fillRect(c, x-5, fy-hop-17, 10, 17, rgb(0x15151b))
		fillRect(c, x-6, fy-hop-31, 12, 12, rgb(0xe8e8f0))
		filledEllipse(c, x-6, fy-hop-33, 4, 4, rgb(0xe8e8f0))
		filledEllipse(c, x+6, fy-hop-33, 4, 4, rgb(0xe8e8f0))
		lc := rgb(partyColors[int(t*5)%len(partyColors)])
		fillRect(c, x-4, fy-hop-28, 3, 3, lc)
		fillRect(c, x+1, fy-hop-28, 3, 3, lc)
		fillRect(c, x-3, fy-hop-23, 7, 1, lc)
		wave(-1, hy(up, 0))
		wave(1, hy(up, 2))
	}
}

// ---- the DROP ---------------------------------------------------------------------------------------

// partyDrop draws the lights and the confetti of the DROP of act 3 (in front of the people, translucent).
func (g *Game) partyDrop(c *ebiten.Image, t float64) {
	u := djDropU(t)
	if u < 0 || u > 4.2 {
		return
	}
	recLayer("front:fx")
	fade := 1 - clampF((u-2.6)/1.6, 0, 1)
	// the beams, fanning out from the booth
	for b := 0; b < 8; b++ {
		ang := -math.Pi/2 + (float64(b)-3.5)*0.26 + 0.35*math.Sin(t*2.2+float64(b))
		col := partyColors[b%len(partyColors)]
		for s := 6.0; s < 150; s += 4 {
			a := uint8(80 * fade * (1 - s/190))
			fillRect(c, math.Round(240+math.Cos(ang)*s)-2, math.Round(150+math.Sin(ang)*s)-2, 5, 5, rgba(col, a))
		}
	}
	// the flash and the ring
	if u < 0.35 {
		fillRect(c, 0, 0, endCW, endCH, rgba(0xffffff, uint8(200*(1-u/0.35))))
	}
	// the confetti cannon: bursts up and out, then falls
	for k := 0; k < 90; k++ {
		ang := -math.Pi/2 + (hashF(k, 3, 51)-0.5)*2.2
		sp := 70 + 90*hashF(k, 4, 51)
		x := 240 + math.Cos(ang)*sp*u
		y := 140 + math.Sin(ang)*sp*u + 95*u*u
		if y > endCH || x < -4 || x > endCW+4 {
			continue
		}
		fillRect(c, math.Round(x), math.Round(y), 2, 3, rgba(partyColors[k%len(partyColors)], uint8(255*fade)))
	}
}

// ---- the lines ---------------------------------------------------------------------------------------

// djLines: what Gabbo says (Italian-flavoured Spanish). They are balloons of the party: see partyLines.
var djLines = []partyLine{
	{"gabbo", 1.5, 3.1, "¡Mamma mia! ¡Che festa!", false},
	{"gabbo", 3.4, 4.6, "¡Lluvia de spaghetti en 3, 2...!", false},
	{"gabbo", 8.3, 10.2, "¡Bajo fuera! ¡Lo bajo con la cuchara!", false},
	{"gabbo", 11.9, 13.4, "¡Spaghetti drop! ¡Bellissimo!", false},
	{"gabbo", 16.7, 18.2, "¡Mi madre baila mejor! ¡Mamma mia!", false},
	{"gabbo", 23.45, 24.4, "¡Tres, dos, uno... DROP!", false},
}

var djLinesEN = map[string]string{
	"¡Mamma mia! ¡Che festa!":               "Mamma mia! Che festa!",
	"¡Lluvia de spaghetti en 3, 2...!":      "Spaghetti rain in 3, 2...!",
	"¡Bajo fuera! ¡Lo bajo con la cuchara!": "Bass out! I lower it with a spoon!",
	"¡Spaghetti drop! ¡Bellissimo!":         "Spaghetti drop! Bellissimo!",
	"¡Mi madre baila mejor! ¡Mamma mia!":    "My mamma dances better! Mamma mia!",
	"¡Tres, dos, uno... DROP!":              "Three, two, one... DROP!",
	"¡DROP!":                                "DROP!",
}

// the names are invented and the same in both languages
func init() {
	for k, v := range djLinesEN {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, n := range djNames {
		for _, k := range []string{"DJ " + n} {
			if _, ok := langDataEN[k]; !ok {
				langDataEN[k] = k
			}
		}
	}
}

// djSayPos: where the balloon of Gabbo goes (screen coordinates): over his head.
func djSayPos(t float64) (x, y float64) {
	hop := math.Round(6 * math.Abs(math.Sin(t*6)))
	return djSlotX[djGabboSlot] * 2, (176 - hop - 44) * 2
}

// djTexts: the nameplate and the DROP word, on the screen (sharp text).
func (g *Game) djTexts(dst *ebiten.Image, t float64) {
	if s := djNameplate(t); s != "" {
		g.drawTextC(dst, s, 250, 48, rgb(partyColors[int(t*3)%len(partyColors)]), 2)
	}
	if u := djDropU(t); u >= 0 && u < 1.2 {
		g.sceneBang(dst, "¡DROP!", 480, 70, u/1.2, 0xffffff, 6)
	}
}
