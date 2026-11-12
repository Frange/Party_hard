package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// DRAWING EFFECTS on single guests for the "sometimes" events of the crowd (waiters, drunks, philosopher, gossiping
// granny, karaoke, hiding under tables). A guest keeps being a plain guest of the game (same state machine, same
// collisions, same nav): an event only writes a small mark on him in pevFxs and the draw loop of the guests asks
// pevGuestDraw once per guest:
//   - sway: a soft lateral weave while he walks (the drunk's S): only the DRAWN x moves, never gu.pos;
//   - trip: he is drawn lying on the floor for a moment (the stars are an Items piece of the drunks event);
//   - hid: he is hidden under a table: his own sprite is skipped and only his feet are drawn, under the cloth.
//
// The marks are removed by the event that wrote them (a test checks nothing is left); granny stays marked because
// the newspaper must know she was a granny when she is killed (pevGuestKilled, a hook of killGuest).
// ---------------------------------------------------------------------------

type pevFx struct {
	sway   float64 // amplitude (px) of the lateral weave while walking
	trip   float64 // game time until which he lies on the floor
	hid    *Prop   // the table he hides under
	hidX   float64 // his place under it (world x)
	granny bool    // the gossiping granny (a stat is counted if she dies)
}

var pevFxs = map[*Guest]*pevFx{}

func pevFxOf(gu *Guest) *pevFx {
	f := pevFxs[gu]
	if f == nil {
		f = &pevFx{}
		pevFxs[gu] = f
	}
	return f
}

// pevFxDrop removes the temporary marks of a guest (the granny mark is kept).
func pevFxDrop(gu *Guest) {
	if f := pevFxs[gu]; f != nil {
		if f.granny {
			f.sway, f.trip, f.hid = 0, 0, nil
			return
		}
		delete(pevFxs, gu)
	}
}

// pevFxPurge forgets the marks of guests who are not in this party any more.
func pevFxPurge(g *Game) {
	if len(pevFxs) == 0 {
		return
	}
	in := make(map[*Guest]bool, len(g.guests))
	for _, gu := range g.guests {
		in[gu] = true
	}
	for gu := range pevFxs {
		if !in[gu] {
			delete(pevFxs, gu)
		}
	}
}

// pevGuestDraw is called by the guest draw loop with the position he will be drawn at. true = already drawn (or hidden).
func (g *Game) pevGuestDraw(dst *ebiten.Image, gu *Guest, pos *Vec) bool {
	if gu.seat != nil && g.seatDrawn(gu) { // sitting: drawn with his seat (seats.go)
		return true
	}
	for _, a := range g.pev.act { // events that draw their own guests (diving board, cleaners: poolboard.go, cleaners.go)
		if d, ok := a.D.(interface {
			drawGuest(*Game, *ebiten.Image, *Guest, Vec) bool
		}); ok && d.drawGuest(g, dst, gu, *pos) {
			return true
		}
	}
	if gu.quirk == rwQuirk {
		pos.Y += g.worksJitter(gu) // a worker shaking with his jackhammer (roadworks.go)
	}
	if len(pevFxs) == 0 {
		return false
	}
	f := pevFxs[gu]
	if f == nil {
		return false
	}
	switch {
	case f.hid != nil:
		return true // drawn by tableHideItems (his feet, under the cloth)
	case g.time < f.trip:
		g.drawLying(dst, gu.cs, true, *pos)
		return true
	case f.sway > 0 && gu.moving:
		pos.X += pevSwayX(gu, g.time, f.sway)
	}
	return false
}

// pevSwayX: the drawn sideways offset of a weaving walker (whole pixels, so the sprite stays sharp).
func pevSwayX(gu *Guest, t, amp float64) float64 {
	return math.Round(math.Sin(t*2.4+gu.off*6.3)*amp + math.Sin(t*0.9+gu.off*3)*amp*0.4)
}

// pevGuestKilled: a hook of killGuest. The gossiping granny dying is a fact for the newspaper (stat "abuela_dead").
func (g *Game) pevGuestKilled(gu *Guest) {
	if f := pevFxs[gu]; f != nil {
		f.trip, f.hid = 0, nil
		if f.granny {
			g.stat("abuela_dead", 1)
			if g.pev.ran == nil {
				g.pev.ran = map[string]bool{}
			}
			g.pev.ran["abuela_dead"] = true // this party's fact: the newspaper tells it (newspaper_events.go)
			f.granny = false
		}
	}
}

// ---- pixel pieces (through plotFn, so the tests can paint them into an image) ------------------------------

var (
	pevRed    = color.RGBA{0xd0, 0x30, 0x38, 255}
	pevSteel  = color.RGBA{0xc8, 0xca, 0xd2, 255}
	pevSteelD = color.RGBA{0x80, 0x84, 0x90, 255}
	pevBrown  = color.RGBA{0xc8, 0x8a, 0x48, 255}
	pevGreen  = color.RGBA{0x58, 0xb8, 0x58, 255}
)

// drawFingerPix: a raised index finger beside a head (cx, cy = feet of the figure). side +1 right, -1 left.
func drawFingerPix(plot plotFn, cx, cy, px, side int, k func(float64) int) {
	x := cx + side*5*px
	top := cy - k(40)
	plot(x, top, px, 3*px, pevSkinW)           // the finger
	plot(x-px, top+3*px, 3*px, 3*px, pevSkinW) // the fist
	plot(x-px, top+5*px, 3*px, px, color.RGBA{0xc8, 0xa8, 0x90, 255})
}

// drawRoundGlassesPix: two small round lenses on a face whose eyes are at (cx, ey).
func drawRoundGlassesPix(plot plotFn, cx, ey, px int) {
	for _, s := range []int{-1, 1} {
		x := cx + s*2*px
		plot(x-px, ey-px, 3*px, px, pevInk)
		plot(x-px, ey+px, 3*px, px, pevInk)
		plot(x-px, ey, px, px, pevInk)
		plot(x+px, ey, px, px, pevInk)
		plot(x, ey, px, px, color.RGBA{0xd8, 0xee, 0xff, 255})
	}
	plot(cx, ey, px, px, pevInk)
}

// drawLensPix: a magnifying glass held out in front: a ring with a handle, centre (x, y).
func drawLensPix(plot plotFn, x, y, px int) {
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			d := dx*dx + dy*dy
			if d >= 5 && d <= 10 {
				plot(x+dx*px, y+dy*px, px, px, pevGoldDk)
			} else if d < 5 {
				plot(x+dx*px, y+dy*px, px, px, color.RGBA{0xc8, 0xe8, 0xf8, 255})
			}
		}
	}
	plot(x-2*px, y-2*px, px, px, pevWhite)
	for i := 0; i < 4; i++ {
		plot(x+(3+i)*px, y+(3+i)*px, px, px, pevInk)
	}
}

// drawBowTiePix: a red bow tie at the neck (cx, ny).
func drawBowTiePix(plot plotFn, cx, ny, px int) {
	plot(cx-3*px, ny, 2*px, 3*px, pevRed)
	plot(cx+px, ny, 2*px, 3*px, pevRed)
	plot(cx-px, ny+px, 2*px, px, color.RGBA{0x90, 0x18, 0x20, 255})
}

// drawVestPix: a black waistcoat over the chest of a figure whose feet are at (cx, cy).
func drawVestPix(plot plotFn, cx, cy, px int, k func(float64) int) {
	top := cy - k(25)
	plot(cx-3*px, top, 2*px, k(9), pevInk)
	plot(cx+px, top, 2*px, k(9), pevInk)
	plot(cx-px, top+k(3), px, px, pevGold)
	plot(cx-px, top+k(6), px, px, pevGold)
}

const (
	trayEmpty = iota
	trayGlass
	trayWine
	trayCanape
	trayOlive
)

// drawTrayPix: a round silver tray held out at (x, y) (its middle) with up to five things on it.
func drawTrayPix(plot plotFn, x, y, px int, items []int) {
	plot(x-5*px, y, 11*px, px, pevSteel)
	plot(x-4*px, y+px, 9*px, px, pevSteelD)
	for i, it := range items {
		if i >= 5 {
			break
		}
		ix := x - 4*px + i*2*px
		switch it {
		case trayGlass:
			plot(ix, y-4*px, px, 4*px, pevWhite)
			plot(ix, y-4*px, px, px, glassCol)
		case trayWine:
			plot(ix, y-4*px, px, 4*px, pevWhite)
			plot(ix, y-3*px, px, 2*px, wineCol)
		case trayCanape:
			plot(ix, y-2*px, 2*px, 2*px, pevBrown)
			plot(ix, y-3*px, px, px, pevRed)
		case trayOlive:
			plot(ix, y-3*px, px, 3*px, pevWhite)
			plot(ix, y-4*px, px, px, pevGreen)
		}
	}
}

// drawMicPix: a hand microphone held up in front of the mouth of a figure whose feet are at (cx, cy).
func drawMicPix(plot plotFn, cx, cy, px int, k func(float64) int) {
	x, y := cx+3*px, cy-k(27)
	plot(x, y+2*px, px, 4*px, pevSteelD)
	plot(x-px, y, 3*px, 3*px, pevInk)
	plot(x, y, px, px, pevSteel)
}

// drawEarsPix: both hands pressed over the ears of a head whose face centre is at (cx, cy-k(29)).
func drawEarsPix(plot plotFn, cx, cy, px int, k func(float64) int) {
	y := cy - k(32)
	for _, s := range []int{-1, 1} {
		x := cx + s*5*px
		if s < 0 {
			x -= px
		}
		plot(x, y, 2*px, 3*px, pevSkinW)
		plot(x, y+3*px, 2*px, px, color.RGBA{0xc8, 0xa8, 0x90, 255})
	}
}

// drawShardsPix: a few bits of broken glass on the floor around (x, y); n = how many (up to 6).
func drawShardsPix(plot plotFn, x, y, px, n int, seed int) {
	for i := 0; i < n && i < 6; i++ {
		dx := ((seed*7+i*13)%17 - 8) * px
		dy := ((seed*5+i*11)%7 - 3) * px
		c := glassCol
		if i%3 == 1 {
			c = pevWhite
		}
		plot(x+dx, y+dy, 2*px, px, c)
		if i%2 == 0 {
			plot(x+dx+px, y+dy+px, px, px, pevSteel)
		}
	}
}

// pevSubFeet draws only the lowest `rows` pixel rows of a sprite (the feet of someone under a table) at world pos.
func (g *Game) pevDrawFeet(dst *ebiten.Image, cs *CharSet, pos Vec, rows int) {
	if cs == nil || rows <= 0 {
		return
	}
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	img, flip := cs.img(AnimIdle, DirDown, 0)
	sub := img.SubImage(image.Rect(0, charBase-rows, charW, charBase)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	if sc != 1 {
		op.Filter = ebiten.FilterLinear
	}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(charW, 0)
	}
	op.GeoM.Translate(-charCX, float64(-rows))
	if sc != 1 {
		op.GeoM.Scale(sc, sc)
	}
	op.GeoM.Translate(g.wx(pos.X), g.wy(pos.Y))
	dst.DrawImage(sub, op)
}

// pevLetGo: release a guest if he is still ours, else just forget him (see pevForget).
func pevLetGo(m *pevMem) {
	if m.mine() {
		pevRelease(m, partyDice())
		return
	}
	pevForget(m)
}
