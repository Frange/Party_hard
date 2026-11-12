package main

import (
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Themed costumes (idea 213): in SOME parties with a clear theme, SOME of the ordinary guests wear a
// costume or accessory of the party, painted as a small pixel-art layer over their own drawing (rig or
// built-in), in every view, animation and while lying dead.
//
//	party (Def.Icon / title)        costumes (look ids)
//	beach    MIAMI BEACH            straw hat, sunglasses + flower lei, swim ring
//	funfair  PARQUE DE ATRACCIONES  clown (nose, wig tufts, ruff), bunny ears, propeller cap
//	grave    CEMENTERIO             pumpkin head, vampire collar and cape, witch hat, mummy bandages
//	snow     ESTACIÓN DE ESQUÍ      bobble beanie, striped scarf, ski goggles
//	casino   CASINO                 tuxedo bib + bow tie, eye mask, top hat
//	eiffel   TORRE EIFFEL           beret, moustache + neckerchief (women: beret + neckerchief), striped sailor top
//
// Not here on purpose: the birthday, hockey, 70s disco and ITALIA crowds already dress up (venuecrowd.go),
// the pirate ship has its own pirate guests, and the space station and the island stay as they are.
//
// How many: each party rolls its own share between costumeMin and costumeMax (20 % to 30 % of the ordinary
// guests that start the party; late arrivals get the same chance). The dice are our own (costumeDice: never
// g.rng nor the shared rand of builtinChars), one seed per party, so the same seed always dresses the same
// guests the same way. Never dressed: named guests and famous faces, kids, the band and performers, the
// special groups (keep), quirky famous behaviours, the arsonist, guards, VIP staff, police and the players
// (they never go through here). A guest who already wears a hat or a big hairdo (head part 10+ px tall)
// never gets a hat nor a face piece (no hat on a hat): only the neck/body looks (lei is a face look here
// because of the sunglasses; swim ring, vampire, scarf, tuxedo, sailor top, clown ruff). The costumed CharSet keeps the guest's own rig, so the dead body, the
// newspaper photo and the bodies carried around keep the costume. Sets are cached per (look, base, colour)
// and every frame is cached by the CharSet like any other: nothing is allocated per frame.
// ---------------------------------------------------------------------------

const (
	costumeMin = 0.20
	costumeMax = 0.30
)

// dressKey: the costume a CharSet wears (look 0 = none) and its colour variant.
type dressKey struct{ look, v uint8 }

// dressGeom: where the head and the torso are in one upright frame.
type dressGeom struct {
	vw             int
	hx, hy, hw, hh int // head box (hair included)
	tx, ty, tw, th int // torso box; ty is the neck row
	female         bool
	f              int // animation frame (the propeller turns)
}

func (d dressGeom) cx() int   { return d.hx + d.hw/2 }
func (d dressGeom) l() int    { return d.hx }
func (d dressGeom) r() int    { return d.hx + d.hw - 1 }
func (d dressGeom) eyeY() int { return d.hy + d.hh - 3 }

// hatBase: the row the hat brim sits on, leaving `need` rows above it inside the frame.
func (d dressGeom) hatBase(drop, need int) int {
	return maxI(d.hy+drop, need)
}

// dressPen paints a costume on a frame and counts the pixels that would fall outside it (tests).
type dressPen struct {
	b       *pixBuf
	oob     int
	measure bool // only record the extent (x0..x1, y0..y1), paint nothing
	x0, y0  int
	x1, y1  int
}

func (p *dressPen) set(x, y int, c uint32) {
	if p.measure {
		p.x0, p.y0, p.x1, p.y1 = minI(p.x0, x), minI(p.y0, y), maxI(p.x1, x), maxI(p.y1, y)
		return
	}
	if x < 0 || y < 0 || x >= p.b.w || y >= p.b.h {
		p.oob++
		return
	}
	p.b.set(x, y, hx(c))
}

func (p *dressPen) rect(x, y, w, h int, c uint32) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			p.set(x+i, y+j, c)
		}
	}
}

// solid: is there already a pixel of the character here?
func (p *dressPen) solid(x, y int) bool {
	if x < 0 || y < 0 || x >= p.b.w || y >= p.b.h {
		return false
	}
	return p.b.p[(y*p.b.w+x)*4+3] != 0
}

type costumeLook struct {
	id    string
	hat   bool // sits on the head: not for someone already wearing a hat or a big hairdo
	face  bool // needs to find the eyes: not for those either (their face is not where a plain head has it)
	paint func(p *dressPen, d dressGeom, v int)
}

func pickV(v int, l []uint32) uint32 { return l[v%len(l)] }

// costumeLooks: index 0 is "no costume".
var costumeLooks = []costumeLook{
	{id: ""},
	// ---- beach ----
	{id: "paja", hat: true, paint: func(p *dressPen, d dressGeom, v int) {
		by := d.hatBase(1, 2)
		straw := pickV(v, []uint32{0xead27a, 0xf2e0a8, 0xe2bc62, 0xead27a})
		band := pickV(v, []uint32{0xc83838, 0x2a6ac8, 0x2a8a4a, 0x1c1c24})
		p.rect(d.l(), by-2, d.hw, 1, straw)
		p.rect(d.l(), by-1, d.hw, 1, band)
		if d.vw == vSide {
			p.rect(d.l()-1, by, d.hw+3, 1, shadeHex(straw, 0.85))
		} else {
			p.rect(d.l()-2, by, d.hw+4, 1, shadeHex(straw, 0.85))
		}
	}},
	{id: "gafas", face: true, paint: func(p *dressPen, d dressGeom, v int) {
		ey := d.eyeY()
		lens := pickV(v, []uint32{0x141418, 0x2a1a40, 0x1a2a3a, 0x3a1418})
		shine := pickV(v, []uint32{0x6a8ac8, 0xd060c0, 0x60d0e0, 0xf0a040})
		switch d.vw {
		case vFront:
			p.rect(d.l(), ey, d.hw, 1, lens)
			p.set(d.l(), ey, shine)
		case vSide:
			p.rect(d.r()-1, ey, 2, 1, lens)
			p.rect(d.l()+1, ey, maxI(0, d.hw-3), 1, lens)
		default:
			p.set(d.l(), ey, lens)
			p.set(d.r(), ey, lens)
		}
		flowers := []uint32{0xf060a0, 0xf8d838, 0xf8f8f0, 0xf08030}
		c := d.cx()
		for i := -3; i <= 3; i++ {
			p.set(c+i, d.ty, flowers[(i+3+v)%len(flowers)])
		}
		if d.vw != vBack {
			p.set(c-1, d.ty+1, flowers[v%4])
			p.set(c, d.ty+1, flowers[(v+1)%4])
			p.set(c+1, d.ty+1, flowers[(v+2)%4])
		}
	}},
	{id: "flotador", paint: func(p *dressPen, d dressGeom, v int) {
		a := pickV(v, []uint32{0xe83838, 0xf8d030, 0x3a8ae8, 0xf060b0})
		b := pickV(v, []uint32{0xf8f8f8, 0xf08020, 0xf8f8f8, 0xf8f8f8})
		wy := d.ty + maxI(4, d.th*2/3)
		x0, x1 := d.tx-1, d.tx+d.tw
		for x := x0; x <= x1; x++ {
			c := a
			if ((x-x0)/2)%2 == 1 {
				c = b
			}
			p.set(x, wy, c)
			p.set(x, wy+1, c)
			p.set(x, wy+2, shadeHex(c, 0.7))
		}
	}},
	// ---- funfair ----
	{id: "payaso", paint: func(p *dressPen, d dressGeom, v int) {
		wig := pickV(v, []uint32{0xf07818, 0x38c048, 0x3888f0, 0xf058b0})
		ey := d.eyeY()
		if d.hh >= 10 { // a hat or a big hairdo: only the clown's ruff (the face is not where a plain head has it)
			for x := d.l() - 2; x <= d.r()+2; x++ {
				c := uint32(0xf8f8f8)
				if (x+v)%2 == 0 {
					c = wig
				}
				p.set(x, d.ty, c)
			}
			return
		}
		switch d.vw {
		case vFront:
			p.set(d.cx(), ey+1, 0xf02020)
			p.rect(d.l()-2, d.hy+1, 2, 3, wig)
			p.rect(d.r()+1, d.hy+1, 2, 3, wig)
		case vSide:
			p.set(d.r()+1, ey+1, 0xf02020)
			p.rect(d.l()-2, d.hy+1, 2, 3, wig)
		default:
			p.rect(d.l()-2, d.hy+1, 2, 3, wig)
			p.rect(d.r()+1, d.hy+1, 2, 3, wig)
		}
		for x := d.l() - 2; x <= d.r()+2; x++ {
			c := uint32(0xf8f8f8)
			if (x+v)%2 == 0 {
				c = wig
			}
			p.set(x, d.ty, c)
		}
	}},
	{id: "conejo", hat: true, paint: func(p *dressPen, d dressGeom, v int) {
		by := d.hatBase(0, 4)
		fur := pickV(v, []uint32{0xf8f8f8, 0x1c1c24, 0xf8f8f8, 0xd8b088})
		in := uint32(0xf080b0)
		ear := func(x int) {
			p.rect(x, by-4, 2, 4, fur)
			p.rect(x+1, by-3, 1, 2, in)
		}
		if d.vw == vSide {
			ear(d.cx() - 1)
		} else {
			ear(d.l())
			ear(d.r() - 1)
		}
		p.rect(d.l(), by, d.hw, 1, in)
	}},
	{id: "helice", hat: true, paint: func(p *dressPen, d dressGeom, v int) {
		by := d.hatBase(1, 3)
		cols := []uint32{0xe83838, 0xf8d030, 0x3a8ae8, 0x38c048}
		for x := d.l(); x <= d.r(); x++ {
			p.set(x, by, cols[(x-d.l()+v)%4])
			p.set(x, by-1, cols[(x-d.l()+v+2)%4])
		}
		p.set(d.cx(), by-2, 0x9a9aa8)
		if d.f%2 == 0 { // the propeller turns with the animation frames
			p.rect(d.cx()-2, by-3, 5, 1, 0xf8d030)
		} else {
			p.rect(d.cx()-1, by-3, 3, 1, 0xf8d030)
		}
	}},
	// ---- graveyard ----
	{id: "calabaza", hat: true, face: true, paint: func(p *dressPen, d dressGeom, v int) {
		orange := pickV(v, []uint32{0xe8761a, 0xf08a28, 0xd86a14, 0xe8761a})
		rib := shadeHex(orange, 0.78)
		x0, x1 := d.l()-1, d.r()+1
		y0, y1 := d.hy, d.hy+d.hh-2
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if (y == y0 || y == y1) && (x == x0 || x == x1) {
					continue // round corners
				}
				c := orange
				if d.vw != vSide && (x == d.cx()-2 || x == d.cx()+2) {
					c = rib
				}
				p.set(x, y, c)
			}
		}
		p.set(d.cx(), y0-1, 0x3a7a28)
		ey := d.eyeY() - 1
		switch d.vw {
		case vFront:
			p.set(d.cx()-1, ey, 0xf8d040)
			p.set(d.cx()+1, ey, 0xf8d040)
			p.rect(d.cx()-1, ey+2, 3, 1, 0x2a1408)
			p.set(d.cx(), ey+2, 0xf8d040)
		case vSide:
			p.set(x1, ey, 0xf8d040)
			p.rect(x1-1, ey+2, 2, 1, 0x2a1408)
		}
	}},
	{id: "vampiro", paint: func(p *dressPen, d dressGeom, v int) {
		cape := pickV(v, []uint32{0x141018, 0x1c1030, 0x141018, 0x2a0810})
		lining := pickV(v, []uint32{0xc01828, 0xd02040, 0x9a1830, 0xc01828})
		ty := d.ty
		if d.vw == vSide {
			p.rect(d.l()-1, ty-2, 1, 3, lining)
			p.rect(d.l()-2, ty-2, 1, 4, cape)
			p.rect(d.l()-1, ty, d.hw+1, 1, cape)
			return
		}
		p.rect(d.l()-2, ty-3, 1, 4, cape)
		p.rect(d.l()-1, ty-2, 1, 3, lining)
		p.rect(d.r()+2, ty-3, 1, 4, cape)
		p.rect(d.r()+1, ty-2, 1, 3, lining)
		p.rect(d.l()-2, ty, d.hw+4, 1, cape)
		if d.vw == vFront {
			p.set(d.cx(), ty+1, lining) // the brooch
			p.set(d.cx()-1, d.eyeY()+2, 0xf4f4f8)
			p.set(d.cx()+1, d.eyeY()+2, 0xf4f4f8)
		}
	}},
	{id: "bruja", hat: true, paint: func(p *dressPen, d dressGeom, v int) {
		by := d.hatBase(1, 4)
		felt := pickV(v, []uint32{0x4a2a6a, 0x1c1824, 0x2a5a3a, 0x5a1a3a})
		band := pickV(v, []uint32{0xf08020, 0xf08020, 0xc070f0, 0x60d040})
		p.rect(d.l()-2, by, d.hw+4, 1, felt)
		p.rect(d.l(), by-1, d.hw, 1, band)
		p.rect(d.l()+1, by-2, maxI(1, d.hw-2), 1, felt)
		p.rect(d.cx(), by-3, 1, 1, felt)
		p.set(d.cx()+1, by-4, felt) // the bent tip
		p.set(d.l()-2, by, shadeHex(felt, 1.6))
	}},
	{id: "momia", face: true, paint: func(p *dressPen, d dressGeom, v int) {
		light := pickV(v, []uint32{0xece4cc, 0xf0ecd8, 0xe4d8b8, 0xece4cc})
		dark := uint32(0xc0b490)
		wrap := func(x0, y0, x1, y1 int) {
			for y := y0; y <= y1; y++ {
				for x := x0; x <= x1; x++ {
					if !p.solid(x, y) {
						continue
					}
					c := light
					if (x+2*y+v)%3 == 0 {
						c = dark
					}
					p.set(x, y, c)
				}
			}
		}
		wrap(d.l(), d.hy, d.r(), d.hy+d.hh-1)
		wrap(d.tx, d.ty, d.tx+d.tw-1, d.ty+d.th/2)
		ey := d.eyeY()
		switch d.vw {
		case vFront:
			p.set(d.cx()-1, ey, 0x201810)
			p.set(d.cx()+1, ey, 0x201810)
		case vSide:
			p.set(d.r()-1, ey, 0x201810)
		}
	}},
	// ---- ski resort ----
	{id: "gorro", hat: true, paint: func(p *dressPen, d dressGeom, v int) {
		by := d.hatBase(1, 4)
		wool := pickV(v, []uint32{0xd83030, 0x3070d8, 0x30a050, 0xf0c020})
		p.rect(d.l(), by-1, d.hw, 1, wool)
		p.rect(d.l()-1, by, d.hw+2, 1, shadeHex(wool, 1.25))
		p.rect(d.l(), by-2, d.hw, 1, wool)
		p.rect(d.cx()-1, by-4, 2, 2, 0xf8f8f8)
	}},
	{id: "bufanda", paint: func(p *dressPen, d dressGeom, v int) {
		a := pickV(v, []uint32{0xd83030, 0x3070d8, 0x30a050, 0x8a3ab0})
		b := pickV(v, []uint32{0xf8f8f8, 0xf0c020, 0xf8f8f8, 0xf0c020})
		for x := d.l() - 1; x <= d.r()+1; x++ {
			c := a
			if (x+v)%3 == 0 {
				c = b
			}
			p.set(x, d.ty-1, c)
			p.set(x, d.ty, shadeHex(c, 0.85))
		}
		tail := d.cx() + 1
		switch d.vw {
		case vSide:
			tail = d.l() - 1
		case vBack:
			tail = d.l()
		}
		for k := 1; k <= 4; k++ {
			c := a
			if k%2 == 0 {
				c = b
			}
			p.set(tail, d.ty+k, c)
		}
	}},
	{id: "gafasqui", face: true, paint: func(p *dressPen, d dressGeom, v int) {
		ey := d.eyeY()
		lens := pickV(v, []uint32{0xf0a020, 0x40a8f0, 0xb050e0, 0x40d080})
		strap := uint32(0x1c1c24)
		switch d.vw {
		case vFront:
			p.rect(d.l()-1, ey-1, d.hw+2, 1, strap)
			p.rect(d.l(), ey-1, d.hw, 2, lens)
			p.set(d.l(), ey-1, 0xf8f8f8)
		case vSide:
			p.rect(d.l()-1, ey-1, d.hw, 1, strap)
			p.rect(d.r()-1, ey-1, 3, 2, lens)
		default:
			p.rect(d.l()-1, ey-1, d.hw+2, 1, strap)
		}
	}},
	// ---- casino ----
	{id: "esmoquin", paint: func(p *dressPen, d dressGeom, v int) {
		bow := pickV(v, []uint32{0x101014, 0xd01828, 0x2a1a5a, 0xe8c040})
		c := d.cx()
		switch d.vw {
		case vFront:
			p.rect(c-1, d.ty+1, 3, minI(5, d.th-2), 0xf4f4f8)
			for k := 1; k <= minI(5, d.th-2); k++ {
				p.set(c-2, d.ty+k, 0x18181e)
				p.set(c+2, d.ty+k, 0x18181e)
			}
			p.set(c, d.ty+3, 0x18181e)
			p.rect(c-2, d.ty, 2, 2, bow)
			p.rect(c+1, d.ty, 2, 2, bow)
			p.set(c, d.ty, shadeHex(bow, 0.7))
		case vSide:
			p.set(d.r(), d.ty, bow)
			p.set(d.r(), d.ty+1, 0xf4f4f8)
		}
	}},
	{id: "antifaz", face: true, paint: func(p *dressPen, d dressGeom, v int) {
		ey := d.eyeY()
		mask := pickV(v, []uint32{0xe8c040, 0xc040d8, 0xe02838, 0xd0d0e0})
		gem := pickV(v, []uint32{0x18141e, 0xf8f8f8, 0xe8c040, 0x18141e})
		switch d.vw {
		case vFront:
			p.rect(d.l()-1, ey-1, d.hw+2, 1, mask)
			for x := d.l() - 1; x <= d.r()+1; x++ {
				if x == d.cx()-1 || x == d.cx()+1 {
					continue // the eye holes
				}
				p.set(x, ey, shadeHex(mask, 0.45)) // the lower edge in shadow
			}
			p.set(d.l()-1, ey-1, gem)
		case vSide:
			p.rect(d.r()-1, ey-1, 3, 1, mask)
			p.set(d.r()+1, ey, mask)
			p.set(d.r()-1, ey, mask)
			p.rect(d.l(), ey-1, maxI(0, d.hw-2), 1, shadeHex(mask, 0.8))
		default:
			p.rect(d.l(), ey-1, d.hw, 1, mask)
			p.set(d.cx(), ey, gem)
			p.set(d.cx()+1, ey+1, gem)
		}
	}},
	{id: "chistera", hat: true, paint: func(p *dressPen, d dressGeom, v int) {
		by := d.hatBase(1, 5)
		silk := uint32(0x1c1c24)
		band := pickV(v, []uint32{0xe8c040, 0xf4f4f8, 0xf05060, 0xe8c040})
		p.rect(d.l()-1, by, d.hw+2, 1, silk)
		p.rect(d.l(), by-4, d.hw, 3, silk)
		p.rect(d.l(), by-1, d.hw, 1, band)
		p.rect(d.l(), by-4, 1, 3, 0x5a5a6a) // the sheen of the silk
	}},
	// ---- Paris ----
	{id: "boina", hat: true, paint: func(p *dressPen, d dressGeom, v int) {
		by := d.hatBase(1, 2)
		felt := pickV(v, []uint32{0xd83040, 0x4a6ad8, 0xd83040, 0xf0e8d0})
		p.rect(d.l(), by-1, d.hw+1, 1, felt)
		p.rect(d.l()-1, by, d.hw+2, 1, shadeHex(felt, 0.45)) // the dark rim
		p.set(d.cx(), by-2, felt)
	}},
	{id: "bigote", face: true, paint: func(p *dressPen, d dressGeom, v int) {
		scarf := pickV(v, []uint32{0xd02030, 0x2a50c0, 0xd02030, 0xf0c020})
		ey := d.eyeY()
		for x := d.l(); x <= d.r(); x++ {
			c := scarf
			if (x+v)%2 == 0 {
				c = 0xf8f8f8 // polka dots
			}
			p.set(x, d.ty, c)
		}
		if d.vw == vFront {
			p.set(d.cx()+1, d.ty+1, scarf)
			p.set(d.cx()+1, d.ty+2, scarf)
		}
		if d.female {
			return
		}
		switch d.vw {
		case vFront:
			p.rect(d.cx()-1, ey+2, 3, 1, 0x2a1810)
			p.set(d.cx()-2, ey+1, 0x2a1810)
			p.set(d.cx()+2, ey+1, 0x2a1810)
		case vSide:
			p.rect(d.r()-1, ey+2, 2, 1, 0x2a1810)
			p.set(d.r()+1, ey+1, 0x2a1810)
		}
	}},
	{id: "marinero", paint: func(p *dressPen, d dressGeom, v int) {
		navy := pickV(v, []uint32{0x1e2a6a, 0xc02030, 0x1e2a6a, 0x1e2a6a})
		for y := d.ty + 1; y <= d.ty+d.th-3; y++ {
			c := uint32(0xf4f4f0)
			if (y-d.ty)%2 == 0 {
				c = navy
			}
			for x := d.tx; x < d.tx+d.tw; x++ {
				if p.solid(x, y) {
					p.set(x, y, c)
				}
			}
		}
	}},
}

// costumeLevels: which looks each themed party uses.
var costumeLevels = map[string][]string{
	"beach":   {"paja", "gafas", "flotador"},
	"funfair": {"payaso", "conejo", "helice"},
	"grave":   {"calabaza", "vampiro", "bruja", "momia"},
	"snow":    {"gorro", "bufanda", "gafasqui"},
	"casino":  {"esmoquin", "antifaz", "chistera"},
	"eiffel":  {"boina", "bigote", "marinero"},
}

// costumeTitles: the random-mode rolls of those parties carry no icon: they are found by their title.
var costumeTitles = map[string]string{
	"MIAMI BEACH": "beach", "PARQUE DE ATRACCIONES": "funfair", "CEMENTERIO": "grave",
	"ESTACIÓN DE ESQUÍ": "snow", "CASINO": "casino", "TORRE EIFFEL": "eiffel",
}

func costumeLookIdx(id string) int {
	for i, l := range costumeLooks {
		if i > 0 && l.id == id {
			return i
		}
	}
	return 0
}

// costumeKindOf: the costume theme of a party ("" = nobody dresses up).
func costumeKindOf(def *LevelDef) string {
	if def == nil {
		return ""
	}
	if _, ok := costumeLevels[def.Icon]; ok {
		return def.Icon
	}
	if def.Icon == "" {
		return costumeTitles[def.Title]
	}
	return ""
}

// ---- the party's own dice ----------------------------------------------------------------------

var (
	costumeRng  *rand.Rand
	costumeSeed int64 // tests: non-zero = deterministic
)

func costumeDice() *rand.Rand {
	if costumeRng == nil {
		sd := costumeSeed
		if sd == 0 {
			sd = time.Now().UnixNano()
		}
		costumeRng = rand.New(rand.NewSource(sd))
	}
	return costumeRng
}

type costumeSetKey struct {
	base *CharSet
	d    dressKey
}

// costumeParty: the costume plan of the party being played.
type costumeParty struct {
	kind    string
	seed    int64
	frac    float64
	r       *rand.Rand
	sets    map[costumeSetKey]*CharSet
	dressed int
}

func (cp *costumeParty) start(kind string, seed int64) {
	*cp = costumeParty{kind: kind, seed: seed}
	if kind == "" {
		return
	}
	cp.r = rand.New(rand.NewSource(seed))
	cp.frac = costumeMin + (costumeMax-costumeMin)*cp.r.Float64()
	cp.sets = map[costumeSetKey]*CharSet{}
}

// costumeEligible: an ordinary, unnamed guest that may wear a costume.
func (g *Game) costumeEligible(gu *Guest) bool {
	if gu == nil || gu.cs == nil || gu.cs.Spec == nil || gu.cs.dress.look != 0 {
		return false
	}
	if gu.out() || gu.name != "" || gu.perf != "" || gu.keep || gu.quirk != "" || gu.py != nil || gu.famousOK {
		return false
	}
	n := gu.cs.Spec.Name
	return !isKidSlug(n) && !g.isFamous(n) && len(costumeFits(g.costume.kind, gu.cs)) > 0
}

// tallHead: the character already wears a hat or a big hairdo (the head part is taller than a plain head).
func tallHead(cs *CharSet) bool {
	if cs.rig != nil {
		if v := cs.rig.V[0]; v != nil && v.L[PartHead].ok {
			return v.L[PartHead].h >= 10
		}
		return true
	}
	return cs.Spec == nil || cs.Spec.Hat != HatNone
}

// costumeFits: the looks of the party that suit this character.
func costumeFits(kind string, cs *CharSet) []int {
	tall := tallHead(cs)
	var out []int
	for _, id := range costumeLevels[kind] {
		li := costumeLookIdx(id)
		if li == 0 || (tall && (costumeLooks[li].hat || costumeLooks[li].face)) {
			continue
		}
		out = append(out, li)
	}
	return out
}

// dressCostumes: called once the starting crowd is complete. Rolls this party's costume plan and dresses
// its share of the ordinary guests.
func (g *Game) dressCostumes() {
	var def *LevelDef
	if g.lvl != nil {
		def = g.lvl.Def
	}
	g.dressCostumesSeed(costumeKindOf(def), costumeDice().Int63())
}

func (g *Game) dressCostumesSeed(kind string, seed int64) {
	cp := &g.costume
	cp.start(kind, seed)
	if kind == "" {
		return
	}
	var pool []*Guest
	for _, gu := range g.guests {
		if g.costumeEligible(gu) {
			pool = append(pool, gu)
		}
	}
	k := int(math.Round(cp.frac * float64(len(pool))))
	perm := cp.r.Perm(len(pool))
	for i := 0; i < k; i++ {
		g.dressGuest(pool[perm[i]])
	}
}

// costumeArrival: a late guest of a themed party gets the same chance.
func (g *Game) costumeArrival(gu *Guest) {
	cp := &g.costume
	if cp.kind == "" || cp.r == nil || !g.costumeEligible(gu) {
		return
	}
	if g.lvl != nil && costumeKindOf(g.lvl.Def) != cp.kind {
		return // a plan left over from another party (a level loaded some other way)
	}
	if cp.r.Float64() < cp.frac {
		g.dressGuest(gu)
	}
}

func (g *Game) dressGuest(gu *Guest) {
	cp := &g.costume
	looks := costumeFits(cp.kind, gu.cs)
	if len(looks) == 0 {
		return
	}
	li := looks[cp.r.Intn(len(looks))]
	v := cp.r.Intn(4)
	gu.cs = cp.dressedSet(gu.cs, dressKey{uint8(li), uint8(v)})
	cp.dressed++
}

// dressedSet: the same character wearing the costume, shared by every guest with the same base and costume.
func (cp *costumeParty) dressedSet(base *CharSet, dk dressKey) *CharSet {
	key := costumeSetKey{base, dk}
	if cs := cp.sets[key]; cs != nil {
		return cs
	}
	cs := dressedCharSet(base, dk)
	if cp.sets != nil {
		cp.sets[key] = cs
	}
	return cs
}

func dressedCharSet(base *CharSet, dk dressKey) *CharSet {
	cs := *base
	cs.cache = map[int]*ebiten.Image{} // its own frames: the base keeps its plain ones
	cs.dress = dk
	return &cs
}

// ---- painting --------------------------------------------------------------------------------------

// dressFrame paints the costume on a copy of a frame. Lying and sleeping frames are turned upright, dressed
// and turned back, so the dead keep their costume.
func (cs *CharSet) dressFrame(b *pixBuf, anim, vw, f int) *pixBuf {
	if int(cs.dress.look) >= len(costumeLooks) || costumeLooks[cs.dress.look].paint == nil || b == nil {
		return b
	}
	switch anim {
	case AnimLying:
		up := b.rotCCW()
		cs.paintDress(up, anim, vFront, 0)
		return up.rotCW()
	case AnimSleep:
		up := b.rotCW()
		cs.paintDress(up, anim, vFront, 0)
		return up.rotCCW()
	}
	o := &pixBuf{w: b.w, h: b.h, p: append([]byte(nil), b.p...)}
	cs.paintDress(o, anim, vw, f)
	return o
}

// paintDress paints the costume on an upright frame; returns the pixels that fell outside (tests).
func (cs *CharSet) paintDress(b *pixBuf, anim, vw, f int) int {
	d, ok := cs.dressGeomFor(b, anim, vw, f)
	if !ok {
		return 0
	}
	lk, v := costumeLooks[cs.dress.look], int(cs.dress.v)
	// a head near the edge of the box (a deep lean, a dance): slide the costume back inside it
	m := &dressPen{b: b, measure: true, x0: b.w, y0: b.h, x1: -1, y1: -1}
	lk.paint(m, d, v)
	dx, dy := 0, 0
	if m.x1 >= m.x0 {
		if m.x0 < 0 {
			dx = -m.x0
		} else if m.x1 >= b.w {
			dx = b.w - 1 - m.x1
		}
		if m.y0 < 0 {
			dy = -m.y0
		} else if m.y1 >= b.h {
			dy = b.h - 1 - m.y1
		}
	}
	d.hx, d.tx, d.hy, d.ty = d.hx+dx, d.tx+dx, d.hy+dy, d.ty+dy
	p := &dressPen{b: b}
	lk.paint(p, d, v)
	return p.oob
}

// rigPose: the pose rigFrame renders for an animation frame (and the view it uses).
func rigPose(anim, vw, f int, female bool) (Pose, int) {
	switch anim {
	case AnimLying:
		return poseFor(AnimIdle, 0), vFront
	case AnimSleep:
		return poseFor(AnimIdle, 1), vFront
	case AnimPee:
		return peePose(f, female), vw
	}
	return poseFor(anim, f), vw
}

func (cs *CharSet) dressGeomFor(b *pixBuf, anim, vw, f int) (dressGeom, bool) {
	female := cs.Spec != nil && cs.Spec.Female
	if cs.rig != nil && (cs.over == nil || cs.over[overrideKey(anim, vw, f)] == nil) {
		p, rvw := rigPose(anim, vw, f, female)
		if d, ok := cs.rig.dressGeom(rvw, p); ok {
			d.female, d.f = female, f
			return d, true
		}
	}
	d, ok := detectDressGeom(b, vw)
	d.female, d.f = female, f
	return d, ok
}

// dressGeom: the head and torso boxes of a rig frame, moved exactly as render moves them.
func (r *Rig) dressGeom(vw int, p Pose) (dressGeom, bool) {
	v := r.V[assetView(vw)]
	if v == nil {
		return dressGeom{}, false
	}
	h, t := &v.L[PartHead], &v.L[PartTorso]
	if !h.ok {
		return dressGeom{}, false
	}
	bob := math.Round(p.Bob)
	if p.Sit {
		bob = math.Max(bob, 3)
	}
	shiftX, dip := 0.0, 0.0
	if vw == vSide {
		shiftX = p.Lean * 0.7
	} else {
		dip = math.Abs(p.Lean) * 0.3
	}
	sway := p.Sway * 0.7
	hdx, hdy := sway+shiftX, bob+dip+math.Abs(p.HeadZ)*0.25
	if vw == vSide {
		hdx += p.HeadZ * 0.7
	}
	d := dressGeom{vw: vw}
	d.hx, d.hy, d.hw, d.hh = h.x0+int(math.Round(hdx)), h.y0+int(math.Round(hdy)), h.w, h.h
	if t.ok {
		d.tx, d.ty, d.tw, d.th = t.x0+int(math.Round(sway+shiftX*0.5)), t.y0+int(math.Round(bob+dip*0.5)), t.w, t.h
	} else {
		d.tx, d.ty, d.tw, d.th = d.hx, d.hy+d.hh, d.hw, 10
	}
	return d, true
}

// detectDressGeom finds the head and torso of a frame drawn by hand or by the built-in generator.
func detectDressGeom(b *pixBuf, vw int) (dressGeom, bool) {
	cx := b.w / 2
	lo, hi := maxI(0, cx-6), minI(b.w-1, cx+6)
	ext := func(y int) (int, int) {
		x0, x1 := -1, -1
		for x := lo; x <= hi; x++ {
			if b.p[(y*b.w+x)*4+3] != 0 {
				if x0 < 0 {
					x0 = x
				}
				x1 = x
			}
		}
		return x0, x1
	}
	ty := -1
	for y := 0; y < b.h && ty < 0; y++ {
		if a, _ := ext(y); a >= 0 {
			ty = y
		}
	}
	if ty < 0 {
		return dressGeom{}, false
	}
	hx0, hx1 := b.w, -1
	for y := ty; y < minI(b.h, ty+4); y++ {
		if a, c := ext(y); a >= 0 {
			hx0, hx1 = minI(hx0, a), maxI(hx1, c)
		}
	}
	hw := hx1 - hx0 + 1
	neck := ty + 7
	for y := ty + 4; y <= minI(b.h-1, ty+10); y++ {
		if a, c := ext(y); a >= 0 && c-a+1 >= hw+3 {
			neck = y
			break
		}
	}
	hh := minI(9, maxI(5, neck-ty))
	d := dressGeom{vw: vw, hx: hx0, hy: ty, hw: hw, hh: hh}
	tx0, tx1 := b.w, -1
	for y := ty + hh + 1; y < minI(b.h, ty+hh+4); y++ {
		if a, c := ext(y); a >= 0 {
			tx0, tx1 = minI(tx0, a), maxI(tx1, c)
		}
	}
	if tx1 < 0 {
		tx0, tx1 = hx0, hx1
	}
	d.tx, d.ty, d.tw, d.th = tx0, ty+hh, tx1-tx0+1, 10
	return d, true
}
