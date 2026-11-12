package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "statue" (ESTATUA HUMANA): in street, square and open-air parties a guest painted silver or gold stands
// still in a statue pose, on a small pedestal or on the floor. Rule of the owner: "que esté inmóvil pero sea un
// invitado igual": he IS a plain guest of the party (in g.guests, counted for the caps and the kill target, he sees
// and reports like anybody, he can be killed and leaves a body with his painted colours). He only stays still: he does
// not walk anywhere while he is a statue. If a player passes very close, sometimes he jumps into a new pose and a guest
// nearby gets a fright (a hop and a cry), with no consequence. If anything else takes him (a fright, a fight...) he
// simply stops being a statue (still painted). At the end he steps down and goes on with the party, still painted.
//
// WHEN: street / square / open-air parties (Paris, fair, festival, Italy, beach, skate park, rooftop, motorway),
// 20 % of them, between 30 and 160 s, once. Group "statue".
// ---------------------------------------------------------------------------

var statueIcons = map[string]bool{"eiffel": true, "funfair": true, "festival": true, "italy": true, "beach": true, "skate": true,
	"roof": true, "highway": true}

const (
	statueWalkMax = 20.0
	statueNear    = 22.0 // a player this close may wake him up
	statueGap     = 5.0  // seconds between two moves
	statueStepH   = 5    // pedestal height (sprite pixels)
)

var statueLines = pevTexts("¡AAAH! ¡Se ha movido!", "¡Qué susto!", "¡Está vivo!", "¡Mamá!", "¡Una estatua viva!", "¡Qué quieto está!")

// the poses (animation, fixed phase): pointing, waving, a dance step, singing, guitar, the thinker
var statuePoses = []struct {
	anim  int
	phase float64
}{{AnimPoint, 0.35}, {AnimWave, 0.25}, {AnimDance3, 0.5}, {AnimSing, 0.3}, {AnimGuitar, 0.2}, {AnimLove, 0.4}, {AnimPirouette, 0.15}}

type statueState struct {
	mem      *pevMem
	spot     Vec
	posed    bool
	t0       float64
	pose     int
	dir      Dir
	pedestal bool
	gold     bool
	nextMove float64
	moves    int
	fright   *Guest
	frightT  float64
	painted  *CharSet
}

func (s *statueState) pevHas(gu *Guest) bool { return s.mem != nil && s.mem.gu == gu }

func init() {
	registerPartyEvent(&partyEvent{ID: "statue", Name: "Estatua humana", Group: "statue",
		Prob: 0.14, AtMin: 30, AtMax: 160, Dur: 110,
		Eligible: func(g *Game, t partyTraits) bool { return t.outdoor && statueIcons[t.icon] && t.roam && t.guests >= 14 },
		Start:    statueStart, Items: statueItems,
		Update: pevGuard(statueUpdate, func(g *Game, a *partyAct) { statueEnd(g, a.D.(*statueState)) })})
}

// statueSpec: the guest's own look painted in silver (or gold) from head to toe.
func statueSpec(base *CharSpec, gold bool) CharSpec {
	var sp CharSpec
	if base != nil && base.Skin.A != 0 {
		sp = *base
	} else {
		sp = genericSpec(partyDice())
	}
	main, light, dark := hx(0xb8bcc8), hx(0xdcdee6), hx(0x868a96)
	if gold {
		main, light, dark = hx(0xd8b040), hx(0xf0d470), hx(0xa07c20)
	}
	sp.Skin, sp.Hair, sp.Shirt, sp.Shirt2, sp.Pants, sp.Shoes = main, dark, light, main, dark, dark
	sp.HatCol, sp.Accent, sp.CarryCol = light, light, light
	if sp.Susp.A != 0 { // only what he already wears is repainted (a zero colour means "none": a cape would appear)
		sp.Susp = dark
	}
	if sp.Cape.A != 0 {
		sp.Cape = main
	}
	if sp.Mask.A != 0 {
		sp.Mask = main
	}
	sp.Makeup, sp.Censor, sp.Carry, sp.Weapon, sp.Ride = false, false, HoldNone, 0, 0
	sp.Name, sp.DisplayName, sp.noOverride = "", "", true
	return sp
}

func statueStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var spot Vec
	ok := false
	if pevDebugOn("statue") {
		spot, ok = g.pevNearPlayerSpot(16, true)
	} else {
		spot, ok = g.pevSpot(r, 16, true)
	}
	if !ok {
		return false
	}
	var who *Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && gu.name == "" && gu.cs != nil && dist(gu.pos, spot) < 240 &&
			(who == nil || dist(gu.pos, spot) < dist(who.pos, spot)) && g.lvl.PathG(gu.pos, spot) != nil {
			who = gu
		}
	}
	if who == nil {
		return false
	}
	m := &pevMem{gu: who}
	if !g.pevSendTo(m, spot) {
		return false
	}
	a.D = &statueState{mem: m, spot: spot, t0: g.time, pose: r.Intn(len(statuePoses)), dir: DirDown, pedestal: r.Float64() < 0.6, gold: r.Float64() < 0.4}
	if pevDebugOn("statue") {
		pevDebugLog(g, "statue", spot)
	}
	return true
}

// statuePose: he arrives and freezes, painted.
func statuePose(g *Game, s *statueState) {
	gu := s.mem.gu
	var base *CharSpec
	if gu.cs != nil {
		base = gu.cs.Spec
	}
	sp := statueSpec(base, s.gold)
	s.painted = newCharSet(&sp)
	gu.cs = s.painted // his body, if he dies, keeps the paint
	s.mem.hold(gIdle)
	s.posed, s.t0, s.nextMove = true, g.time, g.time+2
	pevPuff(g, gu.pos, 6, partyDice())
}

func statueEnd(g *Game, s *statueState) {
	pevLetGo(s.mem)
	s.fright = nil
}

func statueUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*statueState)
	r := partyDice()
	gu := s.mem.gu
	if !s.mem.mine() {
		pevForget(s.mem) // a fright, a fight, a murder: he is just a guest again (still painted)
		return true
	}
	s.mem.keep()
	if !s.posed {
		if gu.state == gIdle && gu.timer > 1000 || dist(gu.pos, s.spot) < 3 {
			statuePose(g, s)
		} else if g.time-s.t0 > statueWalkMax {
			pevLetGo(s.mem)
			return true
		}
		return false
	}
	gu.dir, gu.moving, gu.path = s.dir, false, nil
	if s.fright != nil && g.time > s.frightT+0.6 {
		s.fright = nil
	}
	demo := pevDemo("statue") == "move" && g.time-s.t0 > 3
	if g.time < s.nextMove {
		return false
	}
	near := demo
	for _, pl := range g.players() {
		if pl != nil && !pl.dead && !pl.hidden && dist(pl.pos, gu.pos) < statueNear {
			near = true
		}
	}
	if !near {
		return false
	}
	s.nextMove = g.time + statueGap
	if !demo && r.Float64() < 0.4 { // only sometimes
		return false
	}
	// he jumps into a new pose and somebody nearby gets a fright
	s.pose = (s.pose + 1 + r.Intn(len(statuePoses)-1)) % len(statuePoses)
	s.dir = []Dir{DirDown, DirLeft, DirRight, DirDown}[r.Intn(4)]
	s.moves++
	g.snd.Play("boing")
	var vic *Guest
	for _, o := range g.guests {
		if o != gu && !o.out() && lifeFree(o) && dist(o.pos, gu.pos) < 70 && (vic == nil || dist(o.pos, gu.pos) < dist(vic.pos, gu.pos)) {
			vic = o
		}
	}
	if vic != nil {
		s.fright, s.frightT = vic, g.time
		vic.icon, vic.iconT = pevPick(r, statueLines[:4]), 2
	}
	return false
}

// drawGuest: the statue (frozen frame, raised on his pedestal) and the hop of the frightened one (pe_fx.go asks).
func (s *statueState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	if gu.out() {
		return false
	}
	if gu == s.fright {
		u := (g.time - s.frightT) / 0.6
		h := math.Round(math.Sin(math.Pi*math.Min(1, u)) * 7)
		g.drawActor(dst, gu.cs, AnimPanic, g.time*6, pos, gu.dir, -h, false)
		return true
	}
	if gu != s.mem.gu || !s.posed || !s.mem.mine() {
		return false
	}
	p := statuePoses[s.pose]
	lift := 0.0
	if s.pedestal {
		lift = statueStepH // sprite pixels (drawActor scales them)
	}
	g.drawActor(dst, gu.cs, p.anim, p.phase, pos, s.dir, -lift, false)
	return true
}

// drawPedestalPix: a small stone block (16x5 sprite px) under the feet (cx, cy).
func drawPedestalPix(plot plotFn, cx, cy, px int, k func(float64) int, gold bool) {
	top, side := color.RGBA{0xd8, 0xd4, 0xcc, 255}, color.RGBA{0xa0, 0x9a, 0x90, 255}
	w, h := k(16), k(statueStepH)
	plot(cx-w/2, cy-h, w, h, side)
	plot(cx-w/2, cy-h, w, px, top)
	plot(cx-w/2, cy-px, w, px, color.RGBA{0x70, 0x6a, 0x62, 255})
	c := pevSteel
	if gold {
		c = pevGold
	}
	plot(cx-k(3), cy-h+2*px, k(6), px, c) // a little plaque
}

func statueItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*statueState)
	if !s.posed || !s.mem.mine() {
		return items
	}
	gu := s.mem.gu
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	items = append(items, drawItem{gu.pos.Y - 0.2, func() {
		cx, cy := g.pevXY(gu.pos)
		if s.pedestal {
			drawPedestalPix(plot, cx, cy+px, px, k, s.gold)
		} else { // a hat on the floor for the coins
			plot(cx+k(10), cy-px, k(7), 2*px, pevInk)
			drawCoinPix(plot, cx+k(11), cy-2*px, px)
		}
	}})
	return items
}
