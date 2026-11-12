package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "vendor" (VENDEDOR AMBULANTE): a street vendor with apron and cap pushes his cart along the pavement of the
// party, stops for a while in the middle shouting absurd offers, and walks away by the other side. He IS a guest
// (counted in g.total and the live cap: he only comes if guestRoom() allows it; he can be killed like anybody, he
// sees and reports like anybody). When he leaves the party he is out of the kill target again (as the cleaners).
// A player next to him gets "[E] Comprar (gratis)": ONE purchase per visit (either player), free so as not to
// complicate the economy, with one of four short treats (pe_treats.go): a churro turbo (+30 % speed for 8 s), a party
// hat, a balloon or a lemonade (full stamina). None of them touches sight, stealth or points. Then he leaves.
//
// WHEN: open-air parties with a street (Paris, Italy, festival, fair, beach, motorway, skate park), 20 % of them,
// between 50 and 230 s, once. Group "street" (never together with the mime, the juggler or the buskers).
// ---------------------------------------------------------------------------

var vendorIcons = map[string]bool{"eiffel": true, "italy": true, "festival": true, "funfair": true, "beach": true, "highway": true, "skate": true}

const (
	vendorStopT   = 16.0
	vendorReach   = 30.0
	vendorShoutT  = 4.2
	vendorLegMax  = 40.0
	vendorPrompt  = "Comprar (gratis)"
	vendorDoneMax = 6.0
)

var vendorShouts = pevTexts("¡Churros con wifi!", "¡Helado de croqueta!", "¡Globos que no vuelan!", "¡Limonada de ayer, como nueva!",
	"¡Palomitas cuadradas!", "¡Pipas sin cáscara y sin pipa!", "¡Bocatas invisibles, muy ligeros!", "¡Gorros para el confeti!")

var vendorLines = pevTexts("¡Gracias! ¡Que aproveche!", "¡Me voy, que me multan!", "¡Vuelva pronto!", vendorPrompt)

// what the buyer gets (g.say)
var vendorBuys = pevTexts("Churro turbo: ¡corres más unos segundos!", "Gorro de fiesta: ¡qué elegancia!",
	"Un globo: ¡que no se te escape!", "Limonada: ¡aguante a tope!")

const (
	vdIn = iota
	vdStop
	vdOut
	vdLost
	vdGone
)

type vendorState struct {
	mem         *pevMem
	cs          *CharSet
	mid, exit   Vec
	phase       int
	t0          float64
	nextShout   float64
	bought      bool
	buyer       *Player
	buy         int
	lostT       float64
	shout       int
	cartFlipped bool
}

func (s *vendorState) pevHas(gu *Guest) bool {
	return s.mem != nil && s.mem.gu == gu && s.phase != vdGone
}

func init() {
	registerPartyEvent(&partyEvent{ID: "vendor", Name: "Vendedor ambulante", Group: "street",
		Prob: 0.14, AtMin: 50, AtMax: 230, Dur: 75,
		Eligible: func(g *Game, t partyTraits) bool { return t.outdoor && vendorIcons[t.icon] && t.roam && t.guests >= 14 },
		Start:    vendorStart, Over: vendorOver,
		Update: pevGuard(vendorUpdate, func(g *Game, a *partyAct) { vendorEnd(g, a.D.(*vendorState)) })})
}

func vendorSpec(r *rand.Rand) CharSpec {
	return CharSpec{
		Skin: hx(pick(r, []uint32{0xe8b890, 0xc89070, 0x8a5a3a, 0xf0c8a0})), Hair: hx(0x2a1a10), Shirt: hx(0xf4f2ee), Shirt2: hx(0x3a7ad0),
		Pants: hx(0x2a3a5a), Shoes: hx(0x3a2a1a), HatCol: hx(0xe04040), Accent: hx(0xf4f2ee), Hat: HatCap, Top: TopTee, Hair2: HairShort,
		Beard: 1, Body: BodyBig, Face: 1, noOverride: true,
	}
}

// vendorFloor: the walkway kinds where he likes to walk (pavement, cobbles, boardwalk).
func vendorFloor(l *Level, p Vec) bool {
	kind := ""
	for _, f := range l.Def.Floors {
		if f.R.Has(p.X, p.Y) {
			kind = f.Kind
		}
	}
	return kind == "pave" || kind == "cobble" || kind == "boardwalk"
}

// vendorRoute: an entry near the edge of the party, a stop in the middle and an exit far from the entry, all walkable,
// preferring the pavement; far from the players.
func (g *Game) vendorRoute(r *rand.Rand) (entry, mid, exit Vec, ok bool) {
	l := g.lvl
	var pts []Vec
	for i := 0; i < 160 && len(pts) < 40; i++ {
		p := l.NearestFree(l.RandomRoam(r))
		if !l.Free(p) || l.SolidP(p.X, p.Y) || l.badGround(p) || l.InPool(p) || !l.Outdoor(p) {
			continue
		}
		if i < 120 && !vendorFloor(l, p) {
			continue
		}
		pts = append(pts, p)
	}
	if len(pts) < 2 {
		return
	}
	W, H := l.Def.W, l.Def.H
	edge := func(p Vec) float64 { return math.Min(math.Min(p.X, W-p.X), math.Min(p.Y, H-p.Y)) }
	bs := math.Inf(1)
	for _, p := range pts {
		if s := edge(p); s < bs && (pevDebugOn("vendor") || g.pevPlayerFar(p, 90)) {
			entry, bs = p, s
		}
	}
	if math.IsInf(bs, 1) {
		return
	}
	var path []Vec
	bd := 0.0
	for _, p := range pts {
		if d := dist(p, entry) - edge(p)*0.5; d > bd && dist(p, entry) > 200 {
			if pt := l.PathG(entry, p); pt != nil {
				exit, path, bd = p, pt, d
			}
		}
	}
	if path == nil {
		return
	}
	mid = path[len(path)/2]
	for k := 0; k < len(path)/2; k++ { // the stop nearest the middle that is away from every "[E] ..." of the props
		if q := path[len(path)/2+k]; g.pevClearOfUsables(q, 60) {
			mid = q
			break
		}
		if q := path[len(path)/2-k]; g.pevClearOfUsables(q, 60) {
			mid = q
			break
		}
	}
	if pevDebugOn("vendor") { // the capture: he stops right next to the first player
		if q, ok2 := g.pevNearPlayerSpot(14, true); ok2 && g.pevClearOfUsables(q, 60) && l.PathG(entry, q) != nil && l.PathG(q, exit) != nil {
			mid = q
		}
	}
	return entry, mid, exit, true
}

func vendorStart(g *Game, a *partyAct) bool {
	r := partyDice()
	if g.guestRoom() < 1 {
		return false // the party is full: he comes another day
	}
	entry, mid, exit, ok := g.vendorRoute(r)
	if !ok {
		return false
	}
	spec := vendorSpec(r)
	gu := &Guest{pos: entry, dir: DirDown, cs: newCharSet(&spec), style: r.Intn(4), off: r.Float64() * 6, state: gIdle, timer: 9999, role: r.Intn(2), keep: true}
	g.guests = append(g.guests, gu)
	g.total++
	s := &vendorState{mem: &pevMem{gu: gu, want: gIdle}, cs: gu.cs, mid: mid, exit: exit, t0: g.time, nextShout: g.time + 1}
	if !g.pevSendTo(s.mem, mid) {
		vendorVanish(g, s)
		return false
	}
	a.D = s
	if pevDebugOn("vendor") {
		pevDebugLog(g, "vendor stop", mid)
	}
	return true
}

// vendorVanish: he has left the party: gone, and out of the kill target (as the cleaners, cleaners.go).
func vendorVanish(g *Game, s *vendorState) {
	gu := s.mem.gu
	if !gu.out() {
		pevPuff(g, gu.pos, 8, partyDice())
		gu.gone, gu.icon = true, ""
		g.total--
		g.checkDone()
	}
	s.phase = vdGone
}

func vendorEnd(g *Game, s *vendorState) {
	if s.phase == vdGone || s.mem.gu.out() {
		return
	}
	if s.mem.mine() { // still walking with his cart: he leaves now
		vendorVanish(g, s)
		return
	}
	pevForget(s.mem) // something took him (a fright...): he stays as one more guest
}

func (s *vendorState) shoutNow(g *Game, r *rand.Rand) {
	gu := s.mem.gu
	for _, pl := range g.players() { // a customer at hand: he waits quietly, so his bubble never covers the "[E] Comprar"
		if s.canSell(g, pl) {
			if gu.icon != "" && gu.iconT > 0.3 {
				gu.iconT = 0.3
			}
			return
		}
	}
	if g.time < s.nextShout || gu.icon != "" {
		return
	}
	s.nextShout = g.time + vendorShoutT*(0.8+0.5*r.Float64())
	s.shout = (s.shout + 1 + r.Intn(len(vendorShouts)-1)) % len(vendorShouts)
	gu.icon, gu.iconT = vendorShouts[s.shout], 2.6
}

func vendorUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*vendorState)
	r := partyDice()
	gu := s.mem.gu
	if s.phase == vdGone {
		return true
	}
	if gu.out() { // killed: his body is a body like any other
		return true
	}
	if s.phase != vdLost && !s.mem.mine() {
		pevForget(s.mem)
		s.phase, s.lostT = vdLost, g.time
	}
	switch s.phase {
	case vdIn:
		s.mem.keep()
		s.shoutNow(g, r)
		if gu.state == gIdle && gu.timer > 1000 || dist(gu.pos, s.mid) < 3 {
			s.mem.hold(gIdle)
			s.phase, s.t0 = vdStop, g.time
			gu.dir = DirDown
			if d := pevDemo("vendor"); d == "near" || d == "buy" { // the capture: the first player steps up to him
				g.first().pos = g.lvl.NearestFree(Vec{gu.pos.X + 14, gu.pos.Y + 6})
			}
		} else if g.time-s.t0 > vendorLegMax {
			vendorVanish(g, s)
			return true
		}
	case vdStop:
		s.mem.keep()
		gu.dir = DirDown
		s.shoutNow(g, r)
		if d := pevDemo("vendor"); d == "buy" && g.time-s.t0 > 3 && !s.bought {
			g.withPlayer(g.first(), func() { s.sell(g, g.p) })
		}
		if g.time-s.t0 > vendorStopT || s.bought && g.time-s.t0 > 0 {
			if pevDemo("vendor") != "" && g.time-s.t0 < vendorStopT { // the capture keeps him a moment to see the treat
				return false
			}
			vendorLeave(g, s)
		}
	case vdOut:
		s.mem.keep()
		if !s.bought {
			s.shoutNow(g, r)
		}
		if gu.state == gIdle && gu.timer > 1000 || dist(gu.pos, s.exit) < 4 || g.time-s.t0 > vendorLegMax {
			vendorVanish(g, s)
			return true
		}
	case vdLost: // a fright or a fight took him: when he is calm again he takes his cart and goes
		if g.time-s.lostT > 1 && lifeFree(gu) && gu.state == gIdle && gu.timer < 1000 {
			m := &pevMem{gu: gu}
			if g.pevSendTo(m, s.exit) {
				s.mem, s.phase, s.t0 = m, vdOut, g.time
			}
		}
	}
	return false
}

func vendorLeave(g *Game, s *vendorState) {
	if !g.pevSendTo(s.mem, s.exit) {
		vendorVanish(g, s)
		return
	}
	s.phase, s.t0 = vdOut, g.time
	if s.mem.gu.icon == "" {
		s.mem.gu.icon, s.mem.gu.iconT = vendorLines[1+partyDice().Intn(2)], 2.2
	}
}

// vendorCanSell: the vendor sells to this player now (alive, ours, nothing bought yet, within reach, no prop label closer).
func (s *vendorState) canSell(g *Game, p *Player) bool {
	gu := s.mem.gu
	if s.bought || gu.out() || p == nil || p.dead || p.hidden || p.carrying != nil || (s.phase != vdIn && s.phase != vdStop) || !s.mem.mine() {
		return false
	}
	if dist(p.pos, gu.pos) > vendorReach {
		return false
	}
	ok := true
	g.withPlayer(p, func() { ok = g.useTextProp() == nil })
	return ok
}

// pevUse: E next to the vendor.
func (s *vendorState) pevUse(g *Game, p *Player) bool {
	if !s.canSell(g, p) {
		return false
	}
	s.sell(g, p)
	return true
}

// sell: one of the four treats, once per visit.
func (s *vendorState) sell(g *Game, p *Player) {
	if s.bought {
		return
	}
	r := partyDice()
	s.bought, s.buyer, s.buy = true, p, r.Intn(len(vendorBuys))
	t := g.pevTreatsFor(p)
	switch s.buy {
	case 0:
		t.speedT = g.time + treatSpeedDur
	case 1:
		t.hatT, t.col = g.time+treatHatDur, treatPartyCols[r.Intn(len(treatPartyCols))]
	case 2:
		t.balloonT, t.col = g.time+treatHatDur, treatPartyCols[r.Intn(len(treatPartyCols))]
	case 3:
		p.stamina = math.Max(p.stamina, g.perkStamina)
	}
	g.say(vendorBuys[s.buy])
	g.snd.Play("spark")
	gu := s.mem.gu
	gu.dir = faceTo(gu.pos, p.pos)
	gu.icon, gu.iconT = vendorLines[0], 2.2
}

// drawCartPix: the cart (a box on two wheels under a striped parasol); (x, y) is where its wheels touch the floor.
func drawCartPix(plot plotFn, x, y, px int, k func(float64) int, t float64) {
	w := k(18)
	box := color.RGBA{0x3a, 0x9a, 0xd0, 255}
	plot(x-w/2, y-k(13), w, k(9), box)
	plot(x-w/2, y-k(13), w, px, color.RGBA{0x8a, 0xd0, 0xf0, 255})
	plot(x-w/2+k(2), y-k(10), w-k(4), k(3), pevWhite) // the sign (no words: a churro and a scoop)
	plot(x-w/2+k(4), y-k(9), k(4), px, pevBrown)
	plot(x+k(2), y-k(10), k(3), k(2), color.RGBA{0xf0, 0x9a, 0xc0, 255})
	plot(x-w/2+k(2), y-k(4), k(3), k(3), pevInk) // wheels
	plot(x+w/2-k(5), y-k(4), k(3), k(3), pevInk)
	plot(x-w/2+k(3), y-k(3), px, px, pevSteel)
	plot(x+w/2-k(4), y-k(3), px, px, pevSteel)
	// the parasol
	plot(x, y-k(30), px, k(17), pevSteelD)
	for i := 0; i < 6; i++ {
		c := pevRed
		if i%2 == 1 {
			c = pevWhite
		}
		plot(x-k(11)+i*k(11)*2/6, y-k(31), k(11)*2/6+px, k(3), c)
	}
	plot(x-k(8), y-k(33), k(16), k(2), pevRed)
	_ = t
}

// drawGuest: the vendor pushing his cart (pe_fx.go asks).
func (s *vendorState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	if gu != s.mem.gu || gu.out() || s.phase == vdLost || s.phase == vdGone {
		return false
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	side := 1
	switch gu.dir {
	case DirLeft:
		side = -1
	case DirRight:
		side = 1
	default:
		if s.cartFlipped {
			side = -1
		}
	}
	if gu.dir == DirLeft || gu.dir == DirRight {
		s.cartFlipped = gu.dir == DirLeft
	}
	cx, cy := g.pevXY(pos)
	anim, phase := g.guestPose(gu)
	if gu.dir == DirUp { // the cart goes in front of him: drawn after
		g.drawActor(dst, gu.cs, anim, phase, pos, gu.dir, 0, false)
		drawCartPix(plot, cx+side*k(14), cy+px, px, k, g.time)
		return true
	}
	drawCartPix(plot, cx+side*k(14), cy-px, px, k, g.time)
	g.drawActor(dst, gu.cs, anim, phase, pos, gu.dir, 0, false)
	drawApronPix(plot, cx, cy, px, k)
	return true
}

// vendorOver: the "[E] Comprar (gratis)" over his head for each player within reach (the same reach as the purchase),
// kept 16 px inside the screen like every use label (clampUseTextX); with two players the second one goes higher.
func vendorOver(g *Game, dst *ebiten.Image, a *partyAct) {
	s := a.D.(*vendorState)
	gu := s.mem.gu
	row := 0
	for _, pl := range g.players() {
		if !s.canSell(g, pl) {
			continue
		}
		txt := "[" + actionHintFor(actInteract, pl.idx) + "] " + vendorPrompt
		w := g.textW(txt, 1)
		x := clampUseTextX(math.Round(g.wx(gu.pos.X)), w+map2P(g.is2P()))
		y := math.Round(g.wy(gu.pos.Y)) - math.Round(58*g.lifeScale()) - float64(row*14)
		if g.is2P() {
			y = g.clearOfTags(x, y, w+10)
		}
		col := rgb(0xffd860)
		g.drawTextC(dst, txt, x, y, col, 1)
		if g.is2P() {
			fillRect(dst, math.Round(x-w/2)-7, y+2, 4, 4, playerColor(pl.idx))
		}
		row++
	}
}
