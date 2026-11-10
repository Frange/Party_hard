package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// GUEST EMOTES (idea 60). A small pixel-art icon (9x9) over the head of a guest tells what he is up to, read from
// the states the game already has (no new AI):
//
//	?  waiting for somebody to talk to        (witnesses: none, see suspect.go)
//	zzz sleeping   heart kissing or dancing in pairs   note dancing   skull poisoned or bitten   anger punching   bubble chatting
//
//   - At most emoteMax (6) at the same time, and at most emoteAmbientMax (3) of the calm ones (note, zzz, bubble...): the urgent
//     ones (! skull anger) come first. Every guest rests emoteRest seconds after showing one, so they rotate.
//   - An icon lives emoteLife seconds, pops in (2x for the first 0.1 s, then 1x: whole steps) and blinks out in the last 0.25 s.
//   - It is not drawn over a speech bubble, over another emote or over a player (his J1 / J2 tag, his "!" alert and his halo), and
//     not for guests that are off screen, inside a toilet, abducted or already out.
//   - emotesOn=false (flag -noemotes) turns it all off.
// ---------------------------------------------------------------------------

const (
	emoteMax        = 6
	emoteAmbientMax = 3
	emoteLife       = 1.3
	emoteRest       = 7.0
	emoteRestUrgent = 3.0
	emoteSize       = 9
)

// emotesOn: the switch (true by default).
var emotesOn = true

type emoteKind int

const (
	emNone emoteKind = iota
	emQuestion
	emExcl
	emZzz
	emHeart
	emNote
	emSkull
	emAnger
	emTalk
	emCount
)

// urgent kinds are the ones that matter for the player; the others are decoration.
func (k emoteKind) urgent() bool {
	switch k {
	case emExcl, emSkull, emAnger:
		return true
	}
	return false
}

// emoteOf: which icon a guest wears now (from his state; emNone = none).
func emoteOf(gu *Guest) emoteKind {
	if gu == nil || gu.out() || gu.py != nil || gu.lift > 0 || gu.inside != nil {
		return emNone
	}
	if gu.poison > 0 || gu.bitten > 0 {
		return emSkull
	}
	switch gu.state {
	case gAlarm, gToPhone, gCall, gReport:
		return emNone // a witness wears NO icon: the player's "!" and the cuffs are the only "they saw you" signals (suspect.go)
	case gFlee, gScared, gVomit:
		return emNone // no drop of sweat over the head of the ones who flee, are scared or sick
	case gPunch:
		return emAnger
	case gSleep:
		return emZzz
	case gKiss, gDanceWith:
		return emHeart
	case gDance:
		return emNote
	case gChat:
		return emTalk
	case gWaitP:
		return emQuestion
	}
	return emNone
}

type emoteSlot struct {
	gu   *Guest
	kind emoteKind
	t    float64
}

type emoteState struct {
	slots [emoteMax]emoteSlot
	rest  map[*Guest]float64 // g.time until which a guest does not get another one
	rot   int
	gap   float64
	// what the last frame drew (tests)
	nDrawn int
	drawn  [emoteMax]struct {
		gu         *Guest
		kind       emoteKind
		x, y, w, h float64
	}
}

func (e *emoteState) reset() {
	*e = emoteState{rest: map[*Guest]float64{}}
}

func (e *emoteState) count() (n, ambient int) {
	for i := range e.slots {
		if e.slots[i].gu != nil {
			n++
			if !e.slots[i].kind.urgent() {
				ambient++
			}
		}
	}
	return
}

// emoteOnScreen: the head of the guest is inside the screen.
func (g *Game) emoteOnScreen(gu *Guest) bool {
	x, y := g.wx(gu.pos.X), g.wy(gu.pos.Y)
	return x > 8 && x < screenW-8 && y > 50 && y < screenH-4
}

// updateEmotes frees the icons that ended and hands new ones out (one per frame at most, urgent first).
func (g *Game) updateEmotes(dt float64) {
	e := &g.emo
	if e.rest == nil {
		e.rest = map[*Guest]float64{}
	}
	if !emotesOn || g.mode != ModePlay {
		for i := range e.slots {
			e.slots[i] = emoteSlot{}
		}
		return
	}
	for i := range e.slots {
		s := &e.slots[i]
		if s.gu == nil {
			continue
		}
		s.t += dt
		if s.t >= emoteLife || emoteOf(s.gu) != s.kind {
			r := emoteRest
			if s.kind.urgent() {
				r = emoteRestUrgent
			}
			e.rest[s.gu] = g.time + r
			*s = emoteSlot{}
		}
	}
	if len(e.rest) > 300 {
		for k, v := range e.rest {
			if v < g.time {
				delete(e.rest, k)
			}
		}
	}
	e.gap -= dt
	n, amb := e.count()
	n0 := len(g.guests)
	if n >= emoteMax || e.gap > 0 || n0 == 0 {
		return
	}
	e.rot = (e.rot + 7) % n0
	for pass := 0; pass < 2; pass++ { // urgent first, then the calm ones
		for k := 0; k < n0; k++ {
			gu := g.guests[(e.rot+k)%n0]
			kind := emoteOf(gu)
			if kind == emNone || kind.urgent() != (pass == 0) {
				continue
			}
			if !kind.urgent() && amb >= emoteAmbientMax {
				continue
			}
			if e.rest[gu] > g.time || (gu.iconT > 0 && gu.icon != "") || !g.emoteOnScreen(gu) {
				continue
			}
			taken := false
			for i := range e.slots {
				if e.slots[i].gu == gu {
					taken = true
				}
			}
			if taken {
				continue
			}
			for i := range e.slots {
				if e.slots[i].gu == nil {
					e.slots[i] = emoteSlot{gu: gu, kind: kind}
					e.gap = 0.15
					return
				}
			}
		}
	}
}

// ---- the pictures ----------------------------------------------------------------------------------------------

// 7x7 fills: X main colour, w light, k dark. The 1 px outline around them is added when the picture is built.
var emoteArt = [emCount][7]string{
	emQuestion: {".XXXXX.", "XX...XX", "XX...XX", "....XX.", "...XX..", ".......", "...XX.."},
	emExcl:     {"..XXX..", "..XXX..", "..XXX..", "..XXX..", "...X...", ".......", "..XXX.."},
	emZzz:      {"XXXXXXX", ".....XX", "....XX.", "...XX..", "..XX...", ".XX....", "XXXXXXX"},
	emHeart:    {".XX.XX.", "XwXXXXX", "XXXXXXX", "XXXXXXX", ".XXXXX.", "..XXX..", "...X..."},
	emNote:     {"..XXXX.", "..XXXXX", "..XX.XX", "..XX...", ".XXX...", "XXXX...", ".XX...."},
	emSkull:    {".XXXXX.", "XXXXXXX", "XkXXXkX", "XkXXXkX", ".XXkXX.", ".XXXXX.", "..X.X.."},
	emAnger:    {"XX...XX", "XXX.XXX", ".XXXXX.", "..XXX..", ".XXXXX.", "XXX.XXX", "XX...XX"},
	emTalk:     {"XXXXXXX", "XXXXXXX", "XkXkXkX", "XXXXXXX", ".XXXXX.", "XX.....", "X......"},
}

var emoteColor = [emCount][3]color.RGBA{ // main, light, dark
	emQuestion: {rgb(0xffe060), rgb(0xffffff), rgb(0x806010)},
	emExcl:     {rgb(0xfff0a0), rgb(0xffffff), rgb(0x806010)},
	emZzz:      {rgb(0xb8c8ff), rgb(0xffffff), rgb(0x4a5a90)},
	emHeart:    {rgb(0xff4a78), rgb(0xffc0d0), rgb(0x801030)},
	emNote:     {rgb(0xc890ff), rgb(0xffffff), rgb(0x502890)},
	emSkull:    {rgb(0xe8e8e0), rgb(0xffffff), rgb(0x3a2a30)},
	emAnger:    {rgb(0xff3030), rgb(0xff9090), rgb(0x700000)},
	emTalk:     {rgb(0xffffff), rgb(0xffffff), rgb(0x6a5a9a)},
}

var (
	emoteImg  [emCount]*ebiten.Image
	emoteOp   ebiten.DrawImageOptions
	emoteDark = rgb(0x1a0c10)
)

// emoteImage builds (once) the 9x9 picture of a kind: the art with its 1 px dark outline.
func emoteImage(k emoteKind) *ebiten.Image {
	if im := emoteImg[k]; im != nil {
		return im
	}
	im := ebiten.NewImage(emoteSize, emoteSize)
	art := emoteArt[k]
	fill := func(x, y int) byte {
		if x < 0 || y < 0 || x >= 7 || y >= 7 {
			return '.'
		}
		return art[y][x]
	}
	for y := -1; y <= 7; y++ {
		for x := -1; x <= 7; x++ {
			c := fill(x, y)
			var col color.RGBA
			switch c {
			case 'X':
				col = emoteColor[k][0]
			case 'w':
				col = emoteColor[k][1]
			case 'k':
				col = emoteColor[k][2]
			default:
				near := false
				for dy := -1; dy <= 1 && !near; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if fill(x+dx, y+dy) != '.' {
							near = true
							break
						}
					}
				}
				if !near {
					continue
				}
				col = emoteDark
			}
			fillRect(im, float64(x+1), float64(y+1), 1, 1, col)
		}
	}
	emoteImg[k] = im
	return im
}

// emoteScale: 2 for the first 0.1 s (the pop), then 1.
func emoteScale(t float64) float64 {
	if t < 0.1 {
		return 2
	}
	return 1
}

// emoteVisible: it blinks out in the last 0.25 s.
func emoteVisible(t float64) bool {
	if left := emoteLife - t; left < 0.25 {
		return int(left*24)%2 == 0
	}
	return true
}

// emoteRect: where the icon of a guest goes on the screen (x, y, w, h), above his head and centred.
func (g *Game) emoteRect(gu *Guest, t float64) (x, y, w, h float64) {
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	s := emoteScale(t)
	w, h = emoteSize*s, emoteSize*s
	x = math.Round(g.wx(gu.pos.X)) - math.Round(w/2)
	y = math.Round(g.wy(gu.pos.Y)) - 40*sc - 4 - h
	return
}

// drawEmotes paints the icons; bubbles are the speech bubbles the manager just picked (an emote never covers one).
func (g *Game) drawEmotes(dst *ebiten.Image, bubbles []bubReq) {
	e := &g.emo
	e.nDrawn = 0
	if !emotesOn || g.mode != ModePlay || g.lvl == nil {
		return
	}
	for i := range e.slots {
		s := &e.slots[i]
		if s.gu == nil || !emoteVisible(s.t) {
			continue
		}
		x, y, w, h := g.emoteRect(s.gu, s.t)
		if x < 0 || y < 0 || x+w > screenW {
			continue
		}
		if g.emoteBlocked(x, y, w, h, bubbles) {
			continue
		}
		op := &emoteOp
		op.GeoM.Reset()
		sc := emoteScale(s.t)
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(x, y)
		dst.DrawImage(emoteImage(s.kind), op)
		if e.nDrawn < emoteMax {
			e.drawn[e.nDrawn].gu, e.drawn[e.nDrawn].kind = s.gu, s.kind
			e.drawn[e.nDrawn].x, e.drawn[e.nDrawn].y, e.drawn[e.nDrawn].w, e.drawn[e.nDrawn].h = x, y, w, h
			e.nDrawn++
		}
	}
}

// emoteBlocked: the rectangle touches a bubble, a player (tag, alert mark, body) or an icon already drawn this frame.
func (g *Game) emoteBlocked(x, y, w, h float64, bubbles []bubReq) bool {
	for _, b := range bubbles {
		bw := b.w
		bx := clampF(b.cx-math.Round(bw/2), 2, screenW-bw-2)
		bh := b.h
		if bh == 0 {
			bh = 13
		}
		by := b.y - b.shift
		if emRectsHit(x, y, w, h, bx, by, bw, bh+6) { // +6: the tail of the bubble
			return true
		}
	}
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	chk := func(p *Player) bool {
		if p == nil || p.dead || p.hidden {
			return false
		}
		cx, fy := g.wx(p.pos.X), g.wy(p.pos.Y)
		// body, J1/J2 tag, speech line and alert mark: a column above the player
		return emRectsHit(x, y, w, h, cx-16, fy-40*sc-44, 32, 40*sc+46)
	}
	if chk(g.first()) || chk(g.p2) {
		return true
	}
	for i := range g.lastMarks { // the arrow over the last guests always wins (lastguests.go)
		if m := &g.lastMarks[i]; !m.arrow && emRectsHit(x, y, w, h, m.rx, m.ry, m.rw, m.rh) {
			return true
		}
	}
	for i := 0; i < g.emo.nDrawn; i++ {
		d := &g.emo.drawn[i]
		if emRectsHit(x, y, w, h, d.x, d.y, d.w, d.h) {
			return true
		}
	}
	return false
}

func emRectsHit(ax, ay, aw, ah, bx, by, bw, bh float64) bool {
	return ax < bx+bw && bx < ax+aw && ay < by+bh && by < ay+ah
}
