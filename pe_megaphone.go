package main

import (
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "megafono" (INVITADO CON MEGÁFONO, "¡Sí, por favor!", idea 284): a guest gets hold of a megaphone, walks to an
// open spot and shouts 3-5 absurd announcements in big yellow bubbles with bold letters (bubbles.go `loud`, drawn by
// drawLoudBubble). The guests around turn to him and come a little closer; each one reacts in his own way to each
// announcement: claps, boos, covers his ears or cheers. A short synthetic squeal at the first announcement.
// He is a plain guest (he can be killed, he sees and reports): if he dies or runs off the megaphone falls to the floor
// with a squeal, the audience runs away and it is over. The bubbles go through the bubble manager, so they keep off the
// HUD and the "[E] ..." prompts (hudclear.go) like every other bubble.
//
// WHEN: loud open venues (festival, fair, beach, rooftop, skate park, birthday, boat, train station, Italy,
// motorway, Paris); 10 % of those parties, between 40 and 260 s; a second time at most, after a long cooldown.
// Capture: -pevent megafono (near the player), megafono:show (no alarm breaks it), megafono:kill (the speaker is stabbed after two announcements).
// ---------------------------------------------------------------------------

const (
	megaChance  = 0.10
	megaWalkMax = 18.0
	megaShoutT  = 3.6
	megaDropT   = 5.0
)

var megaIcons = map[string]bool{"festival": true, "funfair": true, "beach": true, "roof": true, "skate": true,
	"cake": true, "boat": true, "train": true, "italy": true, "highway": true, "eiffel": true}

var (
	megaLines = pevTexts("¡ATENCIÓN: SE ACABA EL HIELO!", "¡EL DJ TIENE PROHIBIDO SER PESADO!",
		"¡QUIEN HAYA DEJADO UN PATO EN LA PISCINA...!", "¡RECUERDEN: BEBAN AGUA!", "¡SE HA PERDIDO UN ZAPATO! ¡SOLO UNO!",
		"¡PROHIBIDO BAILAR PEOR QUE YO!", "¡LA CONGA SALE EN CINCO MINUTOS!", "¡EL BAÑO DE LA IZQUIERDA NO, POR FAVOR!",
		"¡HAY CROQUETAS! ¡REPITO: HAY CROQUETAS!", "¡QUE NADIE TOQUE EL TERMOSTATO!", "¡SE BUSCA PAREJA DE BAILE PARA MI TÍO!",
		"¡EL KARAOKE CIERRA POR SU BIEN!", "¡SÍ, POR FAVOR! ¡MÁS FIESTA!", "¡NO DEN DE COMER AL DJ!",
		"¡COCHE ROJO: TIENE LAS LUCES PUESTAS!", "¡ÚLTIMA LLAMADA PARA EL ÚLTIMO BAILE!", "¡APLAUSOS PARA MÍ, POR FAVOR!",
		"¡PROBANDO, PROBANDO! ¡SE OYE!", "¡NADIE SE VA HASTA QUE YO LO DIGA!")
	megaBoo   = pevTexts("¡Buuu!", "¡Bájate de ahí!", "¡Que alguien le quite eso!")
	megaYay   = pevTexts("¡Sí, por favor!", "¡Bien dicho!", "¡Eso, eso!")
	megaEars  = pevTexts("¡Mis oídos!", "¡No hace falta gritar!")
	megaFirst = pevTexts("¿Se me oye? ¿Sí?")
)

const (
	mgWalk = iota
	mgTalk
	mgDrop
	mgOver
)

// reactions of the audience to one announcement
const (
	mrClap = iota
	mrBoo
	mrEars
	mrYay
)

type megaState struct {
	m        *pevMem
	crowd    pevCrowd
	spot     Vec
	phase    int
	t0       float64
	n, total int
	nextT    float64
	shoutEnd float64
	said     string // the announcement now in the air
	side     Dir
	order    []int
	react    map[*Guest]int
	dropPos  Vec
	dropDir  Dir
	bodies   int
	police   bool
	stature  float64
	demoK    bool
}

func (s *megaState) pevHas(gu *Guest) bool { return s.m != nil && s.m.gu == gu || s.crowd.pevHas(gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "megafono", Name: "Invitado con megáfono", Group: "speech",
		Prob: megaChance, AtMin: 40, AtMax: 260, Dur: 30, Cooldown: 240, MaxRuns: 2,
		Eligible: func(g *Game, t partyTraits) bool { return megaIcons[t.icon] && t.roam && t.guests >= 20 },
		Start:    megaStart, Items: megaItems,
		Update: pevGuard(megaUpdate, func(g *Game, a *partyAct) { megaEnd(g, a.D.(*megaState)) })})
}

// megaSfx: the squeal of the megaphone, built the first time it is needed (not heard by a person yet).
func (g *Game) megaSfx() {
	if g.snd == nil || g.snd.sfx == nil {
		return
	}
	if _, ok := g.snd.sfx["megasqueal"]; ok {
		return
	}
	g.snd.sfx["megasqueal"] = mix(tone(2400, 0.18, 0.14, 6, 0), shift(tone(3150, 0.12, 0.10, 9, 0), 0.05))
}

func megaStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var spot Vec
	var ok bool
	if pevDebugOn("megafono") {
		spot, ok = g.pevNearPlayerSpot(30, false)
	} else {
		spot, ok = g.pevSpot(r, 30, false)
	}
	if !ok {
		return false
	}
	debug := pevDebugOn("megafono") // capture: whoever is free comes to the spot near the player
	var cand []*Guest
	near := 0
	for _, gu := range g.guests {
		if !peleaCanBe(g, gu) || gu.cs == nil || gu.cs.fat {
			continue
		}
		d := dist(gu.pos, spot)
		if d < 240 || debug {
			near++
		}
		if d < 260 || debug {
			cand = append(cand, gu)
		}
	}
	if len(cand) == 0 || near < 4 { // nobody to listen
		return false
	}
	sort.SliceStable(cand, func(i, j int) bool { return dist(cand[i].pos, spot) < dist(cand[j].pos, spot) })
	var m *pevMem
	for _, gu := range cand[:minI(len(cand), 6)] {
		if mm := (&pevMem{gu: gu, pt: spot}); g.pevSendTo(mm, spot) {
			m = mm
			break
		}
	}
	if m == nil {
		return false
	}
	best := m.gu
	s := &megaState{m: m, spot: spot, t0: g.time, total: 3 + r.Intn(3), order: r.Perm(len(megaLines)), side: DirRight,
		react: map[*Guest]int{}, bodies: len(g.bodies), police: g.policeCalled}
	if sp := best.cs.Spec; sp != nil {
		s.stature = float64(sp.Stature)
	}
	if r.Intn(2) == 0 {
		s.side = DirLeft
	}
	pevDemoLog("megafono", spot)
	a.D = s
	return true
}

func (s *megaState) alarm(g *Game) bool {
	if pevDemo("megafono") == "show" { // capture: the random killers of the party do not break it up
		return false
	}
	if g.policeCalled && !s.police {
		return true
	}
	for i := s.bodies; i < len(g.bodies); i++ {
		if dist(g.bodies[i].pos, s.spot) < 120 {
			return true
		}
	}
	return false
}

func megaUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*megaState)
	r := partyDice()
	age := g.time - s.t0
	switch s.phase {
	case mgOver:
		return true
	case mgDrop:
		return age > megaDropT
	}
	// capture: the speaker is stabbed after two announcements
	if pevDemo("megafono") == "kill" && !s.demoK && s.phase == mgTalk && s.n >= 2 {
		s.demoK = true
		g.killGuest(s.m.gu, true)
	}
	if !s.m.mine() || s.alarm(g) { // he died, ran off or something frightened the party: the megaphone falls
		megaDrop(g, s, r)
		return false
	}
	s.crowd.update(g, r, nil, 0, nil)
	switch s.phase {
	case mgWalk:
		gu := s.m.gu
		if gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, s.spot) < 6 {
			s.phase, s.t0 = mgTalk, g.time
			s.m.want = gIdle
			s.nextT = g.time + 1.2
			s.crowd.gather(g, r, gu.pos, 4+r.Intn(3), 30, 46, func(o *Guest) bool { return o == gu })
		} else if age > megaWalkMax {
			megaEnd(g, s)
			return true
		}
	case mgTalk:
		gu := s.m.gu
		s.m.keep()
		gu.dir = s.side
		if g.time >= s.nextT {
			if s.n >= s.total {
				megaEnd(g, s)
				return true
			}
			s.announce(g, r)
		}
		// the audience reacts while the announcement is in the air
		if g.time < s.shoutEnd {
			for _, m := range s.crowd.mem {
				if m.gu.icon != "" || r.Float64() > dt*0.5 {
					continue
				}
				switch s.react[m.gu] {
				case mrBoo:
					m.gu.icon, m.gu.iconT = pevPick(r, megaBoo), 1.4
				case mrYay:
					m.gu.icon, m.gu.iconT = pevPick(r, megaYay), 1.4
				case mrEars:
					if r.Float64() < 0.4 {
						m.gu.icon, m.gu.iconT = pevPick(r, megaEars), 1.4
					}
				}
			}
		}
	}
	return false
}

// announce: the next absurd announcement, turned to one side of the audience; every listener picks a reaction.
func (s *megaState) announce(g *Game, r *rand.Rand) {
	gu := s.m.gu
	if s.n > 0 && r.Float64() < 0.7 {
		if s.side == DirLeft {
			s.side = DirRight
		} else {
			s.side = DirLeft
		}
	}
	gu.dir = s.side
	s.said = megaLines[s.order[s.n%len(s.order)]]
	if s.n == 0 && r.Float64() < 0.35 {
		s.said = megaFirst[0]
	}
	gu.icon, gu.iconT = s.said, megaShoutT
	s.shoutEnd = g.time + megaShoutT
	s.nextT = g.time + megaShoutT + 0.9 + r.Float64()*1.2
	if s.n == 0 {
		g.megaSfx()
		g.snd.Play("megasqueal")
	}
	s.n++
	for _, m := range s.crowd.mem {
		s.react[m.gu] = r.Intn(4)
	}
}

// megaDrop: the megaphone falls where he stood (with a squeal) and the audience runs off.
func megaDrop(g *Game, s *megaState, r *rand.Rand) {
	gu := s.m.gu
	s.dropPos, s.dropDir = Vec{gu.pos.X + 4, gu.pos.Y + 1}, s.side
	if s.phase == mgWalk {
		s.dropPos = gu.pos
	}
	pevForget(s.m)
	for _, m := range s.crowd.mem {
		if m.mine() {
			pevScare(g, m.gu, r)
		} else {
			pevForget(m)
		}
	}
	s.crowd.mem = nil
	s.phase, s.t0, s.said = mgDrop, g.time, ""
	g.megaSfx()
	g.snd.Play("megasqueal")
}

func megaEnd(g *Game, s *megaState) {
	if s.phase == mgOver {
		return
	}
	r := partyDice()
	if s.m.gu.icon == s.said && s.said != "" {
		s.m.gu.icon = ""
	}
	pevLetGo(s.m)
	s.crowd.release(g, r)
	s.phase, s.said = mgOver, ""
}

// bubLoud: the bubble of this key is shouted through the megaphone right now (bubbles.go draws it yellow and bold).
func (g *Game) bubLoud(key any) bool {
	gu, ok := key.(*Guest)
	if !ok || gu == nil {
		return false
	}
	for _, a := range g.pev.act {
		if s, ok := a.D.(*megaState); ok && s.phase == mgTalk && s.m.gu == gu && s.said != "" && gu.icon == s.said {
			return true
		}
	}
	return false
}

// drawLoudBubble: a yellow bubble with a dark frame and bold dark letters (x, y: top-left of the box; h: its height).
func (g *Game) drawLoudBubble(dst *ebiten.Image, r bubReq, x, y, h float64) {
	w := r.w
	ink, fill := rgb(0x1a1620), rgb(0xffd82a)
	fillRect(dst, x+1, y, w-2, h, ink)
	fillRect(dst, x, y+1, w, h-2, ink)
	fillRect(dst, x+2, y+1, w-4, h-2, fill)
	fillRect(dst, x+1, y+2, w-2, h-4, fill)
	fillRect(dst, x+3, y+2, w-6, 1, rgb(0xfff4a8))
	if r.shift == 0 { // the tail
		fillRect(dst, r.cx-3, y+h, 6, 1, ink)
		fillRect(dst, r.cx-2, y+h, 4, 1, fill)
		fillRect(dst, r.cx-2, y+h+1, 4, 1, ink)
		fillRect(dst, r.cx-1, y+h+2, 2, 1, ink)
	}
	for i, l := range r.lines {
		lw := math.Round(g.textW(l, 1)) + 1
		tx := x + math.Round((w-lw)/2)
		ty := y + 2 + float64(i)*bubLineH
		g.drawText(dst, l, tx, ty, ink, 1) // bold: the same letters one pixel to the right
		g.drawText(dst, l, tx+1, ty, ink, 1)
	}
}

// ---- drawing ----------------------------------------------------------------------------------------

var (
	megaBody = color.RGBA{0xf4, 0xf2, 0xec, 255}
	megaRed  = color.RGBA{0xd8, 0x2a, 0x2a, 255}
	megaDark = color.RGBA{0x2a, 0x26, 0x30, 255}
)

// drawMegaphonePix: a megaphone in profile, its mouthpiece at (mx, my), the bell towards side (1 right, -1 left).
func drawMegaphonePix(plot plotFn, mx, my, px, side int, skin color.RGBA) {
	at := func(dx int) int { // the x of column dx (0 = the mouthpiece), mirrored to the left
		if side < 0 {
			return mx - dx*px - px
		}
		return mx + dx*px
	}
	plot(at(0), my-px/2, px, px, megaDark)
	plot(at(1), my-px/2, px, px, megaDark)
	for i := 0; i < 7; i++ {
		hh := 1 + i/2 // half height of the cone at this column
		col := megaBody
		if i == 3 {
			col = megaRed
		}
		plot(at(2+i), my-hh*px, px, 2*hh*px, col)
		plot(at(2+i), my-hh*px, px, px, shade(col, 1.08))
		plot(at(2+i), my+(hh-1)*px, px, px, shade(col, 0.8))
	}
	plot(at(9), my-4*px, px, 8*px, megaRed)  // the rim of the bell
	plot(at(4), my+2*px, px, 3*px, megaDark) // the handle and the hand on it
	plot(at(3), my+3*px, 2*px, 2*px, skin)
}

// drawSoundWavesPix: three arcs that leave the bell (x, y) towards side.
func drawSoundWavesPix(plot plotFn, x, y, px, side int, t float64) {
	for i := 0; i < 3; i++ {
		if (int(t*8)+i)%4 == 3 {
			continue
		}
		d := (3 + i*3) * px
		hh := (2 + i) * px
		xx := x + side*d
		col := color.RGBA{0xff, 0xf0, 0x90, 255}
		plot(xx, y-hh, px, 2*hh, col)
		plot(xx-side*px, y-hh-px, px, px, col)
		plot(xx-side*px, y+hh, px, px, col)
	}
}

func megaSkin(gu *Guest) color.RGBA {
	if gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Skin.A != 0 {
		return gu.cs.Spec.Skin
	}
	return color.RGBA{0xf0, 0xc0, 0x98, 255}
}

func megaItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*megaState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	switch s.phase {
	case mgDrop: // on the floor
		items = append(items, drawItem{s.dropPos.Y, func() {
			x, y := g.pevXY(s.dropPos)
			side := 1
			if s.dropDir == DirLeft {
				side = -1
			}
			drawMegaphonePix(plot, x-side*5*px, y-4*px, px, side, megaSkin(s.m.gu))
			if g.time-s.t0 < 1.2 {
				drawSoundWavesPix(plot, x+side*5*px, y-4*px, px, side, g.time)
			}
		}})
		return items
	case mgOver:
		return items
	}
	gu := s.m.gu
	if !s.m.mine() {
		return items
	}
	items = append(items, drawItem{gu.pos.Y + 0.3, func() {
		cx, cy := g.pevXY(gu.pos)
		hy := cy - k(28-s.stature)
		side := 1
		if gu.dir == DirLeft {
			side = -1
		}
		if gu.moving || gu.dir == DirUp || gu.dir == DirDown { // carried at the side while he walks
			drawMegaphonePix(plot, cx+side*3*px, cy-k(13), px, side, megaSkin(gu))
			return
		}
		shout := g.time < s.shoutEnd && s.phase == mgTalk
		my := hy + px
		if !shout {
			my = hy + k(8) // lowered
		}
		drawMegaphonePix(plot, cx+side*3*px, my, px, side, megaSkin(gu))
		if shout {
			drawSoundWavesPix(plot, cx+side*13*px, my, px, side, g.time)
		}
	}})
	if s.phase != mgTalk || g.time >= s.shoutEnd {
		return items
	}
	for _, m := range s.crowd.mem {
		o := m.gu
		if !m.mine() || o.state != gIdle || o.moving {
			continue
		}
		re := s.react[o]
		items = append(items, drawItem{o.pos.Y + 0.3, func() {
			cx, cy := g.pevXY(o.pos)
			switch re {
			case mrClap, mrYay:
				drawClapPix(plot, cx, cy-k(36), px, g.time, int(math.Abs(o.pos.X)))
			case mrEars: // hands over the ears
				hy := cy - k(28)
				skin := megaSkin(o)
				plot(cx-5*px, hy-px, 2*px, 3*px, skin)
				plot(cx+4*px, hy-px, 2*px, 3*px, skin)
			case mrBoo: // thumbs down over the head
				plot(cx-px, cy-k(38), 3*px, 2*px, megaSkin(o))
				plot(cx, cy-k(38)+2*px, px, 2*px, megaSkin(o))
			}
		}})
	}
	return items
}
