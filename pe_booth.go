package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "booth": a PHOTO BOOTH stands in a corner of the party (drawn by the event: sign with blinking bulbs, a
// curtain that closes). A guest walks to it, steps in (only head and shoulders show over the curtain), four FLASHES go
// off and he pulls silly faces; he steps out and a STRIP of four funny photos drops out of the slot with a little
// sign (a speech bubble with a joke). Sometimes a second guest follows. When it is over the booth is gone.
// The newspaper of the party may print one of the strips (newspaper_events.go: events["booth"]).
//
// Nothing is solid: the booth takes a clear spot against the edge of the walkable area, no prop is touched.
// WHEN: parties made for it (house, club, neon, disco, birthday, festival, fair, casino, beach, cruise, hockey), 20 %,
// between 40 and 200 s, once. Group "booth".
// ---------------------------------------------------------------------------

var boothIcons = map[string]bool{"house": true, "neon": true, "club": true, "disco": true, "disco70": true, "cake": true, "festival": true,
	"funfair": true, "casino": true, "beach": true, "boat": true, "hockey": true}

const (
	boothFlashes  = 4
	boothFlashLen = 1.15
	boothStep     = 0.8
)

var boothCaps = pevTexts("Cuatro fotos y ni una buena", "Salió hasta la sombra del sombrero", "La tercera foto tiene lengua",
	"Esta se la enseñaré a mis nietos", "¿Quién es el de la esquina? ¡Soy yo!", "Cuatro caras, cuatro mundos",
	"Me veo mejor en persona", "¡Hay una mano en todas!")
var boothLines = pevTexts("¡Sonrían!", "¡Otra más, otra más!", "¡Esa salió con ojos de pez!", "¡Ya salen, ya salen!")

const (
	bhWalk = iota
	bhEnter
	bhFlash
	bhExit
	bhStrip
	bhDone
)

type boothState struct {
	spot, front Vec
	mem         *pevMem
	phase       int
	t0          float64
	flashes     int
	users       int
	maxUsers    int
	stripT      float64 // game time when the last strip dropped (0 = none showing)
	kinds       [4]int  // the faces of the strip
	cap         string
	stripCount  int
}

func (b *boothState) pevHas(gu *Guest) bool { return b.mem != nil && b.mem.gu == gu }

func init() {
	registerPartyEvent(&partyEvent{ID: "booth", Name: "Cabina de fotos", Group: "booth",
		Prob: 0.20, AtMin: 40, AtMax: 200, Dur: 38,
		Eligible: func(g *Game, t partyTraits) bool { return boothIcons[t.icon] && t.roam && t.guests >= 20 },
		Start:    boothStart, Items: boothItems,
		Update: pevGuard(boothUpdate, func(g *Game, a *partyAct) { boothLetGo(a.D.(*boothState)) })})
}

// boothSpot: a clear spot at the edge of the walkable area (the farthest of 30 from the middle of the roaming zone).
func boothSpot(g *Game) (Vec, bool) {
	r := partyDice()
	var c Vec
	for _, z := range g.lvl.Def.Roam {
		c.X += z.X + z.W/2
		c.Y += z.Y + z.H/2
	}
	c = Vec{c.X / float64(len(g.lvl.Def.Roam)), c.Y / float64(len(g.lvl.Def.Roam))}
	var best Vec
	found := false
	for i := 0; i < 50; i++ {
		p, ok := g.pevSpot(r, 17, false)
		if !ok {
			continue
		}
		if f := g.lvl.NearestFree(Vec{p.X, p.Y + 18}); !g.lvl.Free(f) || dist(f, p) > 24 {
			continue
		}
		if !g.boothFits(p) {
			continue
		}
		near := false // somebody free must be able to come (else the spot is no use)
		for _, gu := range g.guests {
			if dist(gu.pos, p) < 240 && g.pevFreeGuest(gu) {
				near = true
				break
			}
		}
		if !near {
			continue
		}
		if !found || dist(p, c) > dist(best, c) {
			best, found = p, true
		}
	}
	return best, found
}

func boothStart(g *Game, a *partyAct) bool {
	spot, ok := boothSpot(g)
	if !ok {
		return false
	}
	r := partyDice()
	st := &boothState{spot: spot, front: g.lvl.NearestFree(Vec{spot.X, spot.Y + 18}), maxUsers: 1 + r.Intn(2), t0: g.time}
	if !st.nextUser(g) {
		return false
	}
	a.D = st
	return true
}

// nextUser: the nearest free guest with a way to the front of the booth walks there.
func (b *boothState) nextUser(g *Game) bool {
	var best *Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && dist(gu.pos, b.spot) < 300 && g.lvl.PathG(gu.pos, b.front) != nil &&
			(best == nil || dist(gu.pos, b.spot) < dist(best.pos, b.spot)) {
			best = gu
		}
	}
	if best == nil {
		return false
	}
	m := &pevMem{gu: best}
	if !g.pevSendTo(m, b.front) {
		return false
	}
	b.mem, b.phase, b.t0, b.flashes = m, bhWalk, g.time, 0
	b.users++
	return true
}

func boothLetGo(b *boothState) {
	if b.mem != nil {
		pevRelease(b.mem, partyDice())
		b.mem = nil
	}
}

func boothUpdate(g *Game, a *partyAct, dt float64) bool {
	b := a.D.(*boothState)
	r := partyDice()
	age := g.time - b.t0
	if b.phase != bhStrip && b.phase != bhDone && (b.mem == nil || !b.mem.mine()) { // somebody else took him: the visit is over
		b.mem = nil
		b.phase, b.t0 = bhStrip, g.time
		b.stripT = 0
		return b.users >= b.maxUsers
	}
	switch b.phase {
	case bhWalk:
		gu := b.mem.gu
		if gu.state == gIdle && gu.timer > 1000 {
			b.phase, b.t0 = bhEnter, g.time
			b.mem.want = gIdle
		} else if age > 22 {
			boothLetGo(b)
			return true
		}
	case bhEnter:
		b.mem.keep()
		gu := b.mem.gu
		f := math.Min(1, age/boothStep)
		gu.pos = Vec{b.front.X + (b.spot.X-b.front.X)*f, b.front.Y + (b.spot.Y-b.front.Y)*f}
		gu.dir = DirUp
		if f >= 1 {
			gu.dir = DirDown
			b.phase, b.t0, b.flashes = bhFlash, g.time, 0
			for i := range b.kinds {
				b.kinds[i] = r.Intn(4)
			}
		}
	case bhFlash:
		b.mem.keep()
		gu := b.mem.gu
		gu.pos, gu.dir = b.spot, DirDown
		n := int(age / boothFlashLen)
		if n > b.flashes && b.flashes < boothFlashes {
			b.flashes = n
			b.mem.pose([]int{AnimLaugh, AnimWave, AnimKiss, AnimPanic}[n%4])
			g.snd.Play("spark")
		}
		if age >= boothFlashLen*(boothFlashes+0.6) {
			b.phase, b.t0 = bhExit, g.time
			b.mem.hold(gIdle)
		}
	case bhExit:
		b.mem.keep()
		gu := b.mem.gu
		f := math.Min(1, age/boothStep)
		gu.pos = Vec{b.spot.X + (b.front.X-b.spot.X)*f, b.spot.Y + (b.front.Y-b.spot.Y)*f}
		gu.dir = DirDown
		if f >= 1 {
			b.phase, b.t0, b.stripT = bhStrip, g.time, g.time
			b.cap = pevPick(r, boothCaps)
			b.stripCount++
			gu.icon, gu.iconT = pevPick(r, boothLines), 1.6
			boothLetGo(b)
		}
	case bhStrip:
		if age > 6.5 {
			if b.users < b.maxUsers && r.Float64() < 0.7 && b.nextUser(g) {
				b.stripT = 0
				return false
			}
			b.phase = bhDone
			return true
		}
		if age > 0.9 && age < 6 {
			g.bubble(b, bubGuest, Vec{b.spot.X + 12, b.spot.Y + 6}, b.cap, false, 0)
		}
	}
	return false
}

// ---- pixels ------------------------------------------------------------------------------------------------------

var (
	boothBody = color.RGBA{0x5a, 0x2a, 0x7a, 255}
	boothDark = color.RGBA{0x2a, 0x14, 0x3a, 255}
	boothCur  = color.RGBA{0xc8, 0x28, 0x48, 255}
	boothCurD = color.RGBA{0x90, 0x1c, 0x34, 255}
	boothBulb = color.RGBA{0xff, 0xe0, 0x60, 255}
)

// drawBoothBackPix: the back wall, the sign and the roof of the booth, drawn BEFORE the guest inside. (cx, cy) = feet.
func drawBoothBackPix(plot plotFn, cx, cy, px int, k func(float64) int, t float64, flash bool) {
	w, h := k(13), k(46)
	plot(cx-w-px, cy-h-px, 2*w+2*px, h+3*px, pevInk)
	plot(cx-w, cy-h, 2*w, h+px, boothBody)
	plot(cx-w+px, cy-k(33), 2*w-2*px, k(32), boothDark) // the inside
	plot(cx-w-px, cy-h-k(4), 2*w+2*px, k(5), boothDark) // sign
	for i := 0; i < 5; i++ {                            // blinking bulbs
		on := (int(t*3)+i)%2 == 0
		c := boothBulb
		if !on {
			c = color.RGBA{0x80, 0x70, 0x30, 255}
		}
		plot(cx-w+k(3)+i*k(5), cy-h-k(2), 2*px, 2*px, c)
	}
	if flash {
		plot(cx-w+px, cy-k(33), 2*w-2*px, k(32), pevWhite)
	}
}

// drawBoothFrontPix: the posts and the curtain, drawn AFTER the guest: closed (hides his body, leaves head and
// shoulders) or open (pulled to the side).
func drawBoothFrontPix(plot plotFn, cx, cy, px int, k func(float64) int, closed bool) {
	w := k(13)
	plot(cx-w, cy-k(34), 2*px, k(35), boothBody)
	plot(cx+w-2*px, cy-k(34), 2*px, k(35), boothBody)
	if closed {
		plot(cx-w+2*px, cy-k(23), 2*w-4*px, k(24), boothCur)
		for x := cx - w + 2*px; x < cx+w-2*px; x += 3 * px {
			plot(x, cy-k(23), px, k(24), boothCurD)
		}
		plot(cx-w+2*px, cy-k(23), 2*w-4*px, px, boothCurD)
	} else {
		plot(cx+w-k(6), cy-k(33), k(5), k(34), boothCur)
		for x := cx + w - k(6); x < cx+w-px; x += 2 * px {
			plot(x, cy-k(33), px, k(34), boothCurD)
		}
	}
	plot(cx-w, cy+px, 2*w, 2*px, pevInk) // threshold
}

// drawFlashSpillPix: the light of a flash spilling on the floor in front of the booth.
func drawFlashSpillPix(plot plotFn, cx, cy, px int, k func(float64) int, u float64) {
	a := uint8(170 * math.Max(0, 1-u*2.2))
	if a == 0 {
		return
	}
	c := color.RGBA{a, a, a, a}
	plot(cx-k(9), cy+k(3), k(18), k(3), c)
	plot(cx-k(6), cy+k(6), k(12), k(2), c)
}

// drawPhotoStripPix: a strip of four photos hanging from the slot at (x, y) (its top-left corner): paper, four frames,
// silly faces made of a few pixels (kinds 0..3: smile, tongue, glasses, cross-eyed).
func drawPhotoStripPix(plot plotFn, x, y, px int, kinds [4]int, drop float64) {
	h := int(math.Round(26 * math.Min(1, drop)))
	if h < 3 {
		return
	}
	plot(x-px, y-px, 10*px, h*px+2*px, pevInk)
	plot(x, y, 8*px, h*px, pevWhite)
	for i := 0; i < 4; i++ {
		fy := y + (1+i*6)*px
		if fy+5*px > y+h*px {
			break
		}
		plot(x+px, fy, 6*px, 5*px, color.RGBA{0x7a, 0x6a, 0x58, 255})
		plot(x+2*px, fy+px, 4*px, 3*px, color.RGBA{0xf0, 0xd0, 0xa8, 255})
		switch kinds[i] {
		case 0:
			plot(x+3*px, fy+px, px, px, pevInk)
			plot(x+5*px, fy+px, px, px, pevInk)
			plot(x+3*px, fy+3*px, 3*px, px, pevInk)
		case 1:
			plot(x+3*px, fy+px, px, px, pevInk)
			plot(x+5*px, fy+px, px, px, pevInk)
			plot(x+4*px, fy+3*px, 2*px, 2*px, color.RGBA{0xe0, 0x50, 0x60, 255})
		case 2:
			plot(x+2*px, fy+px, 4*px, px, pevInk)
			plot(x+3*px, fy+3*px, 2*px, px, pevInk)
		default:
			plot(x+4*px, fy+px, px, px, pevInk)
			plot(x+5*px, fy+px, px, px, pevInk)
			plot(x+3*px, fy+3*px, 3*px, px, pevInk)
		}
	}
}

func boothItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	b := a.D.(*boothState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	closed := b.phase == bhEnter && g.time-b.t0 > 0.35 || b.phase == bhFlash || b.phase == bhExit && g.time-b.t0 < 0.4
	flash := false
	var u float64
	if b.phase == bhFlash {
		u = math.Mod(g.time-b.t0, boothFlashLen)
		flash = u < 0.18 && int((g.time-b.t0)/boothFlashLen) < boothFlashes
	}
	items = append(items, drawItem{b.spot.Y - 0.3, func() {
		cx, cy := g.pevXY(b.spot)
		drawBoothBackPix(plot, cx, cy, px, k, g.time, flash)
	}})
	items = append(items, drawItem{b.spot.Y + 0.3, func() {
		cx, cy := g.pevXY(b.spot)
		drawBoothFrontPix(plot, cx, cy, px, k, closed)
		if b.phase == bhFlash && u < 0.5 {
			drawFlashSpillPix(plot, cx, cy, px, k, u)
		}
		if b.phase == bhStrip && b.stripT > 0 {
			drawPhotoStripPix(plot, cx+k(15), cy-k(14), px, b.kinds, (g.time-b.stripT)/0.9)
		}
	}})
	return items
}

// boothFits: the booth (about 30 px wide, 50 tall, feet at p) leaves the map and every prop alone.
func (g *Game) boothFits(p Vec) bool {
	if p.Y < 58 || p.X < 24 {
		return false
	}
	for _, pr := range g.lvl.Props {
		if pr.gone || pr.K == nil {
			continue
		}
		w, h := float64(pr.K.W)*pr.sc()/2+18, float64(pr.K.H)*pr.sc()
		if math.Abs(pr.X-p.X) < w && pr.Y-h < p.Y+6 && pr.Y > p.Y-52 {
			return false
		}
	}
	return true
}
