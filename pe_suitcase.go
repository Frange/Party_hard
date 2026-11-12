package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "suitcase" (MALETA CON PREMIO): a guest walks to an open spot, leaves his closed suitcase on the floor
// ("vuelvo enseguida") and goes on with the party. The suitcase is a real prop of the level (prop "suitcase",
// not solid: it never blocks the way) and shows the usual "[E] Abrir" of the props (useLabel / useTextProp /
// clampUseTextX). Any player may open it, ONCE: it pops open and gives a prize chosen at random (points, a rare
// item of the game, or a funny surprise: rubber duck that squeaks, confetti, banknotes, a cake, a party hat), with a
// short ES/EN line. If nobody opens it, after a while the owner comes back for it and takes it away (no
// consequence). If the owner dies, it stays until the event ends.
//
// WHEN: parties where somebody may well carry luggage (station, cruise, beach, Paris, Italy, motorway, house,
// festival, casino, ski, fair, birthday), 18 % of them, between 40 and 220 s, once. Group "lost".
// ---------------------------------------------------------------------------

var suitcaseIcons = map[string]bool{"train": true, "boat": true, "beach": true, "eiffel": true, "italy": true, "highway": true,
	"house": true, "festival": true, "casino": true, "snow": true, "funfair": true, "cake": true}

const (
	suitcaseWalkMax = 18.0
	suitcaseBackAt  = 55.0 // seconds after it was left: the owner comes back for it
	suitcasePts     = 50
	suitcaseBills   = 25
	suitcaseCake    = 10
)

const suitcasePrompt = "Abrir maleta" // the "[E] ..." label (7+ letters: tr() also finds it inside "[E] ...")

var suitcaseLines = pevTexts("Vuelvo enseguida...", "¿Y mi maleta?", "¡Aquí estaba!", "¡Mi maleta! ¿Quién la ha abierto?")

// the prizes: what the player is told (g.say) for each one
var suitcaseSay = pevTexts("Maleta: ¡un fajo de puntos!", "Maleta: ¡un pato de goma! Cuac", "Maleta: ¡confeti para todos!",
	"Maleta: ¡billetes por el aire!", "Maleta: ¡una tarta entera!", "Maleta: ¡un gorro de fiesta!", "Maleta: ¡algo útil dentro!",
	"Maleta: solo calcetines sucios...")

var suitcaseCrowd = pevTexts("¡JAJAJA!", "¡Cuac!", "¡Qué bonito!", "¡Billetes!", "¡Tarta!", "¡Hala!")

const (
	scPoints = iota
	scDuck
	scConfetti
	scBills
	scCake
	scHat
	scItem
	scSocks
	scPrizes
)

const (
	scWalk = iota
	scLeft
	scBack
	scOpen
)

type suitcaseState struct {
	owner  *pevMem
	pr     *Prop
	spot   Vec
	phase  int
	t0     float64
	prize  int
	openT  float64
	opener *Player
}

func (s *suitcaseState) pevHas(gu *Guest) bool {
	return s.owner != nil && s.owner.gu == gu && (s.phase == scWalk || s.phase == scBack)
}

func init() {
	propKinds["suitcase"] = &PropKind{W: 20, H: 18, FW: 14, FH: 4, NoSolid: true, Vars: 2, Category: "decor", Draw: drawSuitcase}
	registerPartyEvent(&partyEvent{ID: "suitcase", Name: "Maleta con premio", Group: "lost",
		Prob: 0.13, AtMin: 40, AtMax: 220, Dur: 75,
		Eligible: func(g *Game, t partyTraits) bool { return suitcaseIcons[t.icon] && t.roam && t.guests >= 12 },
		Start:    suitcaseStart, Items: suitcaseItems,
		Update: pevGuard(suitcaseUpdate, func(g *Game, a *partyAct) { suitcaseEnd(g, a.D.(*suitcaseState)) })})
}

// drawSuitcase: v 0 closed (handle, straps, travel stickers), v 1 open (the lid up, the purple lining inside).
func drawSuitcase(b *pixBuf, v int) {
	body, dark := c(0xb8542c), c(0x7a3418)
	if v == 0 {
		b.rect(2, 6, 16, 11, body)
		b.rect(2, 16, 16, 1, dark)
		b.rect(2, 6, 16, 1, c(0xd8784a))
		b.rect(8, 3, 4, 1, c(0x2a2a30)) // handle
		b.rect(7, 4, 1, 2, c(0x2a2a30))
		b.rect(12, 4, 1, 2, c(0x2a2a30))
		b.rect(5, 6, 1, 11, dark) // straps
		b.rect(14, 6, 1, 11, dark)
		b.rect(9, 8, 2, 2, c(0xf0c030))  // the lock
		b.rect(3, 12, 3, 3, c(0x4ab0e0)) // stickers
		b.rect(15, 9, 2, 3, c(0xf0e0a0))
		b.rect(16, 13, 1, 2, c(0xe04a5a))
		return
	}
	b.rect(2, 1, 16, 7, body) // the lid, standing up behind
	b.rect(3, 2, 14, 5, c(0x6a3a8a))
	b.rect(2, 8, 16, 9, body)
	b.rect(3, 9, 14, 3, c(0x8a4ab0)) // lining
	b.rect(2, 16, 16, 1, dark)
	b.rect(9, 12, 2, 2, c(0xf0c030))
	b.rect(3, 13, 3, 3, c(0x4ab0e0))
}

func suitcaseStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var spot Vec
	ok := false
	if pevDebugOn("suitcase") {
		spot, ok = g.pevNearPlayerSpot(14, false)
	} else {
		spot, ok = g.pevSpot(r, 14, false)
	}
	if !ok || !g.suitcaseRoomAt(spot) {
		return false
	}
	var owner *Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && gu.name == "" && dist(gu.pos, spot) < 220 && (owner == nil || dist(gu.pos, spot) < dist(owner.pos, spot)) && g.lvl.PathG(gu.pos, spot) != nil {
			owner = gu
		}
	}
	if owner == nil {
		return false
	}
	m := &pevMem{gu: owner}
	if !g.pevSendTo(m, spot) {
		return false
	}
	a.D = &suitcaseState{owner: m, spot: spot, t0: g.time, prize: r.Intn(scPrizes)}
	if pevDebugOn("suitcase") {
		pevDebugLog(g, "suitcase", spot)
	}
	return true
}

// suitcasePlace: the owner sets it down beside him.
func suitcasePlace(g *Game, s *suitcaseState) {
	gu := s.owner.gu
	at := g.lvl.NearestFree(Vec{gu.pos.X + 12, gu.pos.Y + 2})
	if !g.lvl.Free(at) || !g.suitcaseRoomAt(at) {
		at = s.spot
	}
	s.pr = g.lvl.addTableProp("suitcase", propKinds["suitcase"], at, 0)
	gu.icon, gu.iconT = suitcaseLines[0], 2.4
	pevRelease(s.owner, partyDice())
	if d := pevDemo("suitcase"); d == "near" || d == "open" { // the capture: the first player steps up to it
		g.first().pos = g.lvl.NearestFree(Vec{s.pr.X + 6, s.pr.Y + 20})
	}
	s.phase, s.t0 = scLeft, g.time
}

// suitcaseRoomAt: the suitcase fits at p: on no other prop, and far enough from anything else that shows an "[E] ..."
// (a bush, a phone, a bottle of poison, a trap...) for its label never to sit next to another one.
func (g *Game) suitcaseRoomAt(p Vec) bool {
	return !g.lvl.propClashes("suitcase", propKinds["suitcase"], p, 0) && g.pevClearOfUsables(p, 52)
}

// pevClearOfUsables: no prop that shows an "[E] ..." label is within d of p (so the label of an event never sits
// on the label of a prop, even with two players each near a different thing).
func (g *Game) pevClearOfUsables(p Vec, d float64) bool {
	for _, pr := range g.lvl.Props {
		k := pr.K
		if pr.gone || pr.Name == "suitcase" {
			continue
		}
		usable := k.Item != "" || k.Hide || k.Phone || k.Poison || k.Trap != "" || pr.Name == "closet" || pr.Name == "wardrobe"
		if usable && distRect(p, pr.Foot) < d {
			return false
		}
	}
	return true
}

func suitcaseEnd(g *Game, s *suitcaseState) {
	if s.phase == scWalk || s.phase == scBack {
		pevLetGo(s.owner)
	}
	if s.pr != nil && !s.pr.used && !s.pr.gone { // nobody opened it: it goes away with its owner (or the staff)
		s.pr.gone = true
		pevPuff(g, Vec{s.pr.X, s.pr.Y}, 8, partyDice())
	}
}

func suitcaseUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*suitcaseState)
	gu := s.owner.gu
	age := g.time - s.t0
	switch s.phase {
	case scWalk:
		if !s.owner.mine() {
			pevForget(s.owner)
			return true // something else took him before he set it down: no suitcase this time
		}
		s.owner.keep()
		if gu.state == gIdle && gu.timer > 1000 || dist(gu.pos, s.spot) < 4 {
			suitcasePlace(g, s)
		} else if age > suitcaseWalkMax {
			pevLetGo(s.owner)
			return true
		}
	case scLeft:
		if d := pevDemo("suitcase"); d == "open" && age > 4 {
			g.withPlayer(g.first(), func() { s.open(g, g.p) })
			return false
		}
		if age > suitcaseBackAt && !gu.out() && g.pevFreeGuest(gu) {
			m := &pevMem{gu: gu}
			if g.pevSendTo(m, g.lvl.NearestFree(Vec{s.pr.X - 10, s.pr.Y + 2})) {
				s.owner, s.phase, s.t0 = m, scBack, g.time
				gu.icon, gu.iconT = suitcaseLines[1], 2.4
			}
		}
	case scBack:
		if !s.owner.mine() {
			pevForget(s.owner)
			s.phase, s.t0 = scLeft, g.time-suitcaseBackAt+8 // he will try again a bit later
			return false
		}
		s.owner.keep()
		if gu.state == gIdle && gu.timer > 1000 || dist(gu.pos, Vec{s.pr.X, s.pr.Y}) < 14 {
			s.pr.gone = true
			gu.icon, gu.iconT = suitcaseLines[2], 2.4
			pevRelease(s.owner, partyDice())
			return true
		} else if age > suitcaseWalkMax {
			pevLetGo(s.owner)
			s.phase, s.t0 = scLeft, g.time-suitcaseBackAt+8
		}
	case scOpen:
		if g.time-s.openT > 3.5 {
			return true
		}
	}
	return false
}

// suitcaseFor: the running suitcase event that owns this prop (nil = none).
func (g *Game) suitcaseFor(pr *Prop) *suitcaseState {
	for _, a := range g.pev.act {
		if s, ok := a.D.(*suitcaseState); ok && s.pr == pr {
			return s
		}
	}
	return nil
}

// suitcaseLabel: the "[E] Abrir" of the props (items.go useLabel), only while it is closed and its event runs.
func (g *Game) suitcaseLabel(pr *Prop) (string, color.RGBA) {
	if pr.used || pr.gone || !g.partyActive() || g.suitcaseFor(pr) == nil {
		return "", color.RGBA{}
	}
	return suitcasePrompt, rgb(0xffd860)
}

// pevUse: E on the suitcase (the nearest thing with a label, the same reach as the label).
func (s *suitcaseState) pevUse(g *Game, p *Player) bool {
	if s.pr == nil || s.pr.used || s.pr.gone || (s.phase != scLeft && s.phase != scBack) || g.useTextProp() != s.pr {
		return false
	}
	s.open(g, p)
	return true
}

// open: once. The prize, its line and its effect.
func (s *suitcaseState) open(g *Game, p *Player) {
	if s.pr.used {
		return
	}
	r := partyDice()
	if s.phase == scBack { // he was coming back for it: he lets it be
		pevLetGo(s.owner)
	}
	s.pr.used, s.pr.Var = true, 1
	s.phase, s.openT, s.opener = scOpen, g.time, p
	at := Vec{s.pr.X, s.pr.Y}
	g.treatSfx()
	g.snd.Play("spark")
	prize := s.prize
	if prize == scItem {
		kinds := []string{smokeKind, "disguisebag", "poisonbottle"}
		got := false
		for _, i := range r.Perm(len(kinds)) {
			if k := propKinds[kinds[i]]; k != nil && canTake(p, k.Item) {
				g.withPlayer(p, func() { g.pickItem(&Prop{Name: kinds[i], K: k}) })
				got = true
				break
			}
		}
		if !got {
			prize = scPoints // pockets full: points instead
		}
	}
	s.prize = prize
	g.say(suitcaseSay[prize])
	crowd := func(line string) {
		n := 0
		for _, gu := range g.guests {
			if n < 2 && !gu.out() && gu.icon == "" && dist(gu.pos, at) < 90 && lifeFree(gu) {
				gu.icon, gu.iconT = line, 1.8
				n++
			}
		}
	}
	switch prize {
	case scPoints:
		g.award(at, suitcasePts, "")
	case scDuck:
		g.snd.Play("squeak")
		crowd(suitcaseCrowd[0])
	case scConfetti:
		for i := 0; i < 34; i++ {
			a := -math.Pi/2 + (r.Float64()-0.5)*1.6
			sp := 50 + r.Float64()*60
			g.parts = append(g.parts, Particle{pos: Vec{at.X, at.Y - 10}, vel: Vec{math.Cos(a) * sp, math.Sin(a) * sp}, life: 1.4, max: 1.4,
				col: treatPartyCols[i%len(treatPartyCols)], size: 2, grav: 70})
		}
		crowd(suitcaseCrowd[2])
	case scBills:
		for i := 0; i < 14; i++ {
			a := -math.Pi/2 + (r.Float64()-0.5)*1.8
			sp := 40 + r.Float64()*40
			g.parts = append(g.parts, Particle{pos: Vec{at.X, at.Y - 8}, vel: Vec{math.Cos(a) * sp, math.Sin(a) * sp}, life: 1.8, max: 1.8,
				col: color.RGBA{0x7a, 0xc8, 0x6a, 255}, size: 3, grav: 30})
		}
		g.award(at, suitcaseBills, "")
		crowd(suitcaseCrowd[3])
	case scCake:
		g.award(at, suitcaseCake, "")
		crowd(suitcaseCrowd[4])
	case scHat:
		t := g.pevTreatsFor(p)
		t.hatT, t.col = g.time+treatHatDur, treatPartyCols[r.Intn(len(treatPartyCols))]
	}
	if gu := s.owner.gu; !gu.out() && dist(gu.pos, at) < 120 && gu.icon == "" {
		gu.icon, gu.iconT = suitcaseLines[3], 2.4
	}
}

// drawDuckPix: a yellow rubber duck (8x7) whose bottom-left is at (x, y).
func drawDuckPix(plot plotFn, x, y, px int) {
	yel := color.RGBA{0xff, 0xd8, 0x30, 255}
	plot(x, y-3*px, 7*px, 3*px, yel)      // body
	plot(x+4*px, y-6*px, 3*px, 3*px, yel) // head
	plot(x+7*px, y-5*px, 2*px, px, color.RGBA{0xf0, 0x80, 0x20, 255})
	plot(x+5*px, y-5*px, px, px, pevInk)
	plot(x+px, y-2*px, 3*px, px, color.RGBA{0xe8, 0xb8, 0x20, 255})
}

// drawCakePix: a small cake with a candle, bottom centre (x, y).
func drawCakePix(plot plotFn, x, y, px int, t float64) {
	plot(x-5*px, y-4*px, 11*px, 4*px, color.RGBA{0xf0, 0xc0, 0xd8, 255})
	plot(x-5*px, y-5*px, 11*px, px, pevWhite)
	plot(x-5*px, y-2*px, 11*px, px, color.RGBA{0xc8, 0x6a, 0x9a, 255})
	plot(x, y-8*px, px, 3*px, color.RGBA{0x6a, 0xb8, 0xf0, 255})
	if int(t*6)%2 == 0 {
		plot(x, y-10*px, px, 2*px, color.RGBA{0xff, 0xc8, 0x30, 255})
	} else {
		plot(x, y-9*px, px, px, color.RGBA{0xff, 0x8a, 0x20, 255})
	}
}

// drawSocksPix: a pair of striped socks hanging out (x, y bottom-left).
func drawSocksPix(plot plotFn, x, y, px int) {
	for i := 0; i < 2; i++ {
		sx := x + i*4*px
		plot(sx, y-6*px, 2*px, 6*px, pevWhite)
		plot(sx, y-4*px, 2*px, px, pevRed)
		plot(sx, y-px, 3*px, px, pevWhite)
	}
}

func suitcaseItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*suitcaseState)
	if s.phase != scOpen || s.pr == nil {
		return items
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	t := g.time - s.openT
	rise := int(math.Round(math.Min(1, t/0.4) * 10))
	items = append(items, drawItem{s.pr.Y + 0.3, func() {
		x, y := g.pevXY(Vec{s.pr.X, s.pr.Y})
		top := y - k(12) - rise*px
		switch s.prize {
		case scDuck:
			hop := int(math.Abs(math.Sin(t*7)) * 3)
			drawDuckPix(plot, x-4*px, top-hop*px, px)
		case scCake:
			drawCakePix(plot, x, top, px, g.time)
		case scSocks:
			drawSocksPix(plot, x-4*px, top, px)
		case scHat:
			if t < 1 {
				drawPartyHatPix(plot, x, top, px, treatPartyCols[0])
			}
		case scPoints, scBills:
			drawCoinPilePix(plot, x, y-k(10), px, 6)
		}
	}})
	return items
}
