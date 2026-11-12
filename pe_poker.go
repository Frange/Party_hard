package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "poker" (idea 267, PÓKER CLANDESTINO): sometimes a little green card table appears in a quiet corner and three or
// four guests sit down to play: the dealer deals, they bet chips, one bluffs, the cards are shown, one wins and laughs, and
// after a couple of hands the big loser storms off. It is ONLY an action of guests (the user: "tú no molestas ni haces nada
// ahí aparte de matar"): there is no prompt and no E; the players are plain guests (they count, they can be killed, they are
// witnesses like anybody). If one of them is killed (or frightened off) the game breaks: the others jump up and run.
//
// The table is a prop of its own ("pokertable", solid only on its real footprint) put in the level when the event starts and
// taken away when it ends; it is checked like the cloth tables (pe_tables.go): nothing under it, and no way of the party may be
// cut by it (if it would, it is not placed). Cards are generic (no brand, plain pips). Own dice (partyDice), never g.rng.
// ---------------------------------------------------------------------------

const (
	pkrGatherMax = 30.0 // seconds to sit down
	pkrPlayLen   = 52.0 // seconds of play
	pkrHandLen   = 9.0  // one hand: deal, bets, show, pause
	pkrNear      = 260.0
	pkrWalkMax   = 160.0 // longest walk (along the way, not the straight line) of a player to his seat
)

var (
	pkrDealLines  = pevTexts("Reparto yo", "Hagan sus apuestas", "Nueva mano", "Cortad, que barajo")
	pkrBetLines   = pevTexts("Subo diez", "Lo veo", "Paso", "Voy con todo", "Me retiro", "Cinco más")
	pkrBluffLines = pevTexts("Llevo una mano de escándalo", "Ni pestañeo", "Yo no me tiro faroles")
	pkrCallLines  = pevTexts("¡Va de farol!", "No me lo creo", "Ese tic te delata")
	pkrWinLines   = pevTexts("¡Escalera de color!", "¡Full de reyes!", "¡Póker de ases!", "¡Toma ya!")
	pkrLoseLines  = pevTexts("¡Otra vez!", "Las cartas me odian", "¿Quién ha barajado?", "Qué suerte tienes...")
	pkrQuitLines  = pevTexts("¡Me habéis desplumado!", "¡Me largo, tramposos!", "¡Esto está amañado!")
	pkrEndLines   = pevTexts("Buena partida", "La próxima, revancha", "Me voy con las ganancias")
	pkrBreakLines = pevTexts("¡Se acabó la partida!", "¡Dejad las cartas!", "¡AAAH!")
)

// the four seats around the table (feet of the sitter, from the table anchor), the way each one looks and his chair
var pkrSeats = [4]struct {
	off Vec
	dir Dir
}{
	{Vec{0, -16}, DirDown},   // behind the table: the dealer, facing us
	{Vec{-26, -2}, DirRight}, // left
	{Vec{26, -2}, DirLeft},   // right
	{Vec{0, 12}, DirUp},      // in front, his back to us
}

const (
	pkSit = iota
	pkDeal
	pkBet
	pkShow
	pkPause
	pkDone
)

type pkrSeat struct {
	m      *pevMem
	seat   int
	sat    bool
	chips  int // how many chips he has won or lost (only for who storms off)
	folded bool
}

type pkrState struct {
	tbl     *Prop
	seats   []*pkrSeat
	phase   int
	t0      float64 // start of the current phase
	start   float64 // start of the play
	hand    int
	nextBet float64
	pot     int
	flying  []pevCoin // chips in the air (to the pot or to the winner)
	cards   []pevCoin
	winner  *pkrSeat
	bluffer *pkrSeat
	quitter bool // somebody already stormed off
	broken  bool
}

func (s *pkrState) pevHas(gu *Guest) bool {
	for _, p := range s.seats {
		if p.m.gu == gu {
			return true
		}
	}
	return false
}

var pokerIcons = map[string]bool{"house": true, "roof": true, "ranch": true, "casino": true, "boat": true, "pirate": true,
	"italy": true, "disco70": true, "train": true, "neon": true}

func init() {
	propKinds["pokertable"] = &PropKind{W: 44, H: 26, FW: 32, FH: 9, Category: "furniture", Draw: drawPokerTable}
	registerPartyEvent(&partyEvent{ID: "poker", Name: "Póker clandestino", Group: "table",
		Prob: 0.12, AtMin: 40, AtMax: 230, Dur: 75, Prio: 1,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && pokerIcons[t.icon] },
		Start:    pokerStart, Items: pokerItems,
		Update: pevGuard(pokerUpdate, func(g *Game, a *partyAct) { pokerEnd(g, a.D.(*pkrState), false) })})
}

// drawPokerTable: a small oval table of green felt with a wooden rim on four legs (bottom row = the feet).
func drawPokerTable(b *pixBuf, v int) {
	wood, woodD, felt, feltD := c(0x8a5a2e), c(0x4e3018), c(0x2e8a4a), c(0x1e6034)
	b.rect(6, 14, 2, 12, woodD)
	b.rect(36, 14, 2, 12, woodD)
	b.rect(13, 16, 2, 10, woodD)
	b.rect(29, 16, 2, 10, woodD)
	b.ellipse(22, 10, 21, 9, woodD)
	b.ellipse(22, 9, 20, 8, wood)
	b.ellipse(22, 9, 17, 6.5, felt)
	b.ellipse(22, 10, 12, 3.5, feltD)
	b.rect(4, 13, 36, 2, woodD)
}

// pkrSpotOK: the table fits at p (nothing under it, its four seats are open floor).
func (g *Game) pkrSpotOK(p Vec, k *PropKind) bool {
	l := g.lvl
	if !pevDebugOn("poker") && (dist(p, l.Def.Start) < 80 || !g.pevPlayerFar(p, 90)) {
		return false
	}
	for _, s := range pkrSeats {
		q := Vec{p.X + s.off.X, p.Y + s.off.Y}
		if !l.Free(q) || l.SolidP(q.X, q.Y) || l.badGround(q) || l.InPool(q) {
			return false
		}
	}
	for _, dr := range l.Def.Dance {
		if dr.Grow(20).Has(p.X, p.Y) {
			return false
		}
	}
	if z := l.vip; z != nil {
		for _, rc := range []Rect{z.in, z.trig, z.gate} {
			if rc.Grow(26).Has(p.X, p.Y) {
				return false
			}
		}
	}
	return !l.propClashes("pokertable", k, p, 0)
}

// pkrPlaceTable puts the table in the level; nil (and nothing changed) if no spot keeps every way open.
func (g *Game) pkrPlaceTable(r *rand.Rand) *Prop {
	l := g.lvl
	k := propKinds["pokertable"]
	fitAllFootprints()
	reach0 := l.reachFrom(l.Def.Start)
	for i := 0; i < 40; i++ {
		p, ok := g.pevSpot(r, 30, false)
		if pevDemo("poker") != "" { // capture (-pevent poker:near): next to the player
			p, ok = g.pevNearPlayerSpot(30, false)
		}
		if !ok || !g.pkrSpotOK(p, k) {
			continue
		}
		if pr := g.pevAddProp("pokertable", k, p, 0, reach0, func(l *Level) bool {
			for _, s := range pkrSeats {
				q := Vec{p.X + s.off.X, p.Y + s.off.Y}
				if !l.Free(q) {
					return false
				}
			}
			return true
		}); pr != nil {
			return pr
		}
	}
	return nil
}

// pevAddProp adds a solid prop to the running level and rebuilds the walking map; it is undone (nil) if a cell that could be
// reached before is cut off now, or if ok (when given) says no.
func (g *Game) pevAddProp(name string, k *PropKind, p Vec, v int, reach0 []bool, ok func(*Level) bool) *Prop {
	l := g.lvl
	nProps, nP, nN := len(l.Props), len(l.solidsP), len(l.solidsN)
	pr := l.addTableProp(name, k, p, v)
	l.initNav()
	bad := ok != nil && !ok(l)
	if !bad {
		reach1 := l.reachFrom(l.Def.Start)
		if len(reach0) == len(reach1) {
			for i := range reach0 {
				if reach0[i] && !reach1[i] && !l.blocked[i] {
					bad = true
					break
				}
			}
		}
	}
	if bad {
		l.Props, l.solidsP, l.solidsN = l.Props[:nProps], l.solidsP[:nP], l.solidsN[:nN]
		l.initNav()
		return nil
	}
	return pr
}

// pevPathLen: the length of a walk (nil = no way: very long).
func pevPathLen(path []Vec, from Vec) float64 {
	if path == nil {
		return 1e9
	}
	n, prev := 0.0, from
	for _, p := range path {
		n += dist(prev, p)
		prev = p
	}
	return n
}

// pevRemoveProp takes a prop added by an event out of the level again (its solid too).
func (g *Game) pevRemoveProp(pr *Prop) {
	l := g.lvl
	if l == nil || pr == nil {
		return
	}
	for i, o := range l.Props {
		if o == pr {
			l.Props = append(l.Props[:i], l.Props[i+1:]...)
			break
		}
	}
	drop := func(rs []Rect) []Rect {
		for i, r := range rs {
			if r == pr.Foot {
				return append(rs[:i], rs[i+1:]...)
			}
		}
		return rs
	}
	if !pr.K.NoSolid {
		l.solidsP, l.solidsN = drop(l.solidsP), drop(l.solidsN)
	}
	l.initNav()
}

func pokerStart(g *Game, a *partyAct) bool {
	r := partyDice()
	tbl := g.pkrPlaceTable(r)
	if tbl == nil {
		return false
	}
	st := &pkrState{tbl: tbl, phase: pkSit, t0: g.time}
	tp := Vec{tbl.X, tbl.Y}
	// the guests with the shortest walk to the table (along the way: a wall between makes a straight line useless)
	type cg struct {
		gu *Guest
		d  float64
	}
	var cand []cg
	for _, gu := range g.guests {
		if !g.pevFreeGuest(gu) || gu.name != "" || dist(gu.pos, tp) > pkrNear {
			continue
		}
		if d := pevPathLen(g.lvl.PathG(gu.pos, tp), gu.pos); d <= pkrWalkMax {
			cand = append(cand, cg{gu, d})
		}
	}
	for i := 1; i < len(cand); i++ {
		for j := i; j > 0 && cand[j].d < cand[j-1].d; j-- {
			cand[j], cand[j-1] = cand[j-1], cand[j]
		}
	}
	want := 3 + r.Intn(2)
	seat := 0
	for _, c := range cand {
		gu := c.gu
		if len(st.seats) >= want || seat >= len(pkrSeats) {
			break
		}
		q := Vec{tp.X + pkrSeats[seat].off.X, tp.Y + pkrSeats[seat].off.Y}
		m := &pevMem{gu: gu, pt: q}
		if pevDemo("poker") != "" { // capture: they are already there
			gu.pos = q
		}
		if !g.pevSendTo(m, q) {
			continue
		}
		st.seats = append(st.seats, &pkrSeat{m: m, seat: seat})
		seat++
	}
	if len(st.seats) < 3 {
		for _, s := range st.seats {
			pevLetGo(s.m)
		}
		g.pevRemoveProp(tbl)
		return false
	}
	a.D = st
	pevDemoLog("poker", tp)
	return true
}

func (st *pkrState) seated() []*pkrSeat {
	var out []*pkrSeat
	for _, s := range st.seats {
		if s.sat {
			out = append(out, s)
		}
	}
	return out
}

func (st *pkrState) seatPos(s *pkrSeat) Vec {
	o := pkrSeats[s.seat].off
	return Vec{st.tbl.X + o.X, st.tbl.Y + o.Y}
}

func (st *pkrState) potPos() Vec { return Vec{st.tbl.X, st.tbl.Y - 9} }

func pokerUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*pkrState)
	r := partyDice()
	// a player killed or taken away by anything else breaks the game: the rest jump up
	for _, s := range st.seats {
		if !s.m.mine() && (s.sat || s.m.gu.dead) {
			st.broken = true
		}
	}
	if st.broken {
		pokerEnd(g, st, true)
		return true
	}
	live := st.seats[:0]
	for _, s := range st.seats {
		if s.m.mine() {
			live = append(live, s)
		} else {
			pevForget(s.m)
		}
	}
	st.seats = live
	age := g.time - st.t0
	switch st.phase {
	case pkSit:
		for _, s := range st.seats {
			gu := s.m.gu
			if !s.sat && gu.state == gIdle && gu.timer > 1000 {
				s.sat = true
				gu.pos = st.seatPos(s)
				gu.dir = pkrSeats[s.seat].dir
				s.m.pose(AnimIdle)
			}
		}
		if n := len(st.seated()); n == len(st.seats) && n >= 3 || age > pkrGatherMax && n >= 3 {
			for _, s := range st.seats { // the slow ones are not waited for
				if !s.sat {
					pevLetGo(s.m)
				}
			}
			st.seats = st.seated()
			st.start = g.time
			pkrPhase(g, st, pkDeal, r)
		} else if age > pkrGatherMax {
			pokerEnd(g, st, false)
			return true
		}
	case pkDeal:
		pkrSettle(g, st)
		if age > 1.8 {
			pkrPhase(g, st, pkBet, r)
		}
	case pkBet:
		pkrSettle(g, st)
		if g.time >= st.nextBet {
			st.nextBet = g.time + 0.7 + r.Float64()*0.6
			in := pkrIn(st)
			if len(in) > 0 {
				s := in[r.Intn(len(in))]
				gu := s.m.gu
				switch {
				case st.bluffer == s && gu.icon == "":
					gu.icon, gu.iconT = pevPick(r, pkrBluffLines), 2
					s.m.pose(AnimLaugh)
				case r.Float64() < 0.18 && len(in) > 2:
					s.folded = true
					gu.icon, gu.iconT = pkrBetLines[4], 1.6
					s.m.pose(AnimTired)
				default:
					if gu.icon == "" {
						gu.icon, gu.iconT = pevPick(r, pkrBetLines[:4]), 1.5
					}
					s.m.pose(AnimPoint)
					st.flying = append(st.flying, pevCoin{from: Vec{gu.pos.X, gu.pos.Y - 14}, to: st.potPos(), t0: g.time})
					s.chips--
					g.snd.Play("clac")
				}
			}
		}
		if st.bluffer != nil && age > 2.6 && age-dt <= 2.6 {
			for _, s := range pkrIn(st) {
				if s != st.bluffer && s.m.gu.icon == "" {
					s.m.gu.icon, s.m.gu.iconT = pevPick(r, pkrCallLines), 1.8
					break
				}
			}
		}
		if age > 4.4 {
			pkrPhase(g, st, pkShow, r)
		}
	case pkShow:
		pkrSettle(g, st)
		if age > 2.0 {
			pkrPhase(g, st, pkPause, r)
		}
	case pkPause:
		pkrSettle(g, st)
		if age > 1.4 {
			// the big loser of the night storms off (once), then the next hand or the end
			if !st.quitter && st.hand >= 2 && len(st.seats) > 2 {
				var worst *pkrSeat
				for _, s := range st.seats {
					if s != st.winner && (worst == nil || s.chips < worst.chips) {
						worst = s
					}
				}
				if worst != nil && r.Float64() < 0.7 {
					st.quitter = true
					gu := worst.m.gu
					pevRelease(worst.m, r)
					gu.icon, gu.iconT = pevPick(r, pkrQuitLines), 2.4
					gu.path = g.lvl.PathG(gu.pos, g.farRoamFrom(gu.pos, r))
					if gu.path != nil {
						gu.state, gu.next, gu.dur, gu.hurry = gGoto, gIdle, 1, false
					}
					var keep []*pkrSeat
					for _, s := range st.seats {
						if s != worst {
							keep = append(keep, s)
						}
					}
					st.seats = keep
				}
			}
			if g.time-st.start > pkrPlayLen || len(st.seats) < 2 {
				pokerEnd(g, st, false)
				return true
			}
			pkrPhase(g, st, pkDeal, r)
		}
	}
	if st.phase != pkSit {
		for _, s := range st.seats {
			s.m.keep()
			s.m.gu.dir = pkrSeats[s.seat].dir
			s.m.gu.pos = st.seatPos(s)
		}
	}
	return false
}

// farRoamFrom: a roaming point far from p (own dice).
func (g *Game) farRoamFrom(p Vec, r *rand.Rand) Vec {
	best := g.lvl.RandomRoam(r)
	for k := 0; k < 6; k++ {
		if q := g.lvl.RandomRoam(r); dist(q, p) > dist(best, p) {
			best = q
		}
	}
	return best
}

func pkrIn(st *pkrState) []*pkrSeat {
	var out []*pkrSeat
	for _, s := range st.seats {
		if !s.folded {
			out = append(out, s)
		}
	}
	return out
}

// pkrSettle drops the chips and cards that already landed.
func pkrSettle(g *Game, st *pkrState) {
	var n int
	st.flying, n = pevSettleCoins(st.flying, g.time)
	st.pot += n
	st.cards, _ = pevSettleCoins(st.cards, g.time)
}

func pkrPhase(g *Game, st *pkrState, ph int, r *rand.Rand) {
	st.phase, st.t0 = ph, g.time
	switch ph {
	case pkDeal:
		st.hand++
		st.winner, st.bluffer = nil, nil
		for _, s := range st.seats {
			s.folded = false
			s.m.pose(AnimIdle)
		}
		dealer := st.seats[0]
		dealer.m.pose(AnimWave)
		if dealer.m.gu.icon == "" {
			dealer.m.gu.icon, dealer.m.gu.iconT = pevPick(r, pkrDealLines), 1.6
		}
		from := Vec{dealer.m.gu.pos.X, dealer.m.gu.pos.Y - 12}
		for i, s := range st.seats {
			for k := 0; k < 2; k++ {
				st.cards = append(st.cards, pevCoin{from: from, to: pkrCardAt(st, s, k), t0: g.time + float64(i*2+k)*0.18})
			}
		}
	case pkBet:
		st.nextBet = g.time + 0.3
		if r.Float64() < 0.6 {
			st.bluffer = st.seats[r.Intn(len(st.seats))]
		}
	case pkShow:
		in := pkrIn(st)
		if len(in) == 0 {
			in = st.seats
		}
		w := in[r.Intn(len(in))]
		st.winner = w
		w.chips += st.pot + 2
		w.m.pose(AnimCheer)
		w.m.gu.icon, w.m.gu.iconT = pevPick(r, pkrWinLines), 2
		for _, s := range st.seats {
			if s != w {
				if r.Float64() < 0.5 {
					s.m.pose(AnimTalk)
				} else {
					s.m.pose(AnimLaugh)
				}
				if s.m.gu.icon == "" && r.Float64() < 0.5 {
					s.m.gu.icon, s.m.gu.iconT = pevPick(r, pkrLoseLines), 1.8
				}
			}
		}
		// the pot slides to the winner
		for i := 0; i < minI(st.pot, 6); i++ {
			st.flying = append(st.flying, pevCoin{from: st.potPos(), to: Vec{w.m.gu.pos.X, w.m.gu.pos.Y - 10}, t0: g.time + float64(i)*0.08})
		}
		st.pot = -minI(st.pot, 6) // the flying ones are taken out when they land
		g.snd.Play("win")
	case pkPause:
		st.pot = 0
		for _, s := range st.seats {
			s.m.pose(AnimIdle)
		}
	}
}

// pkrCardAt: where card k of a seat lies on the felt (in front of him).
func pkrCardAt(st *pkrState, s *pkrSeat, k int) Vec {
	o := pkrSeats[s.seat].off
	cx, cy := st.tbl.X+o.X*0.45, st.tbl.Y-9+(o.Y+2)*0.22
	return Vec{cx - 2 + float64(k)*4, cy}
}

// pokerEnd lets everybody go; broken = one of them was killed: the rest jump up and run.
func pokerEnd(g *Game, st *pkrState, broken bool) {
	if st.phase == pkDone {
		return
	}
	st.phase = pkDone
	r := partyDice()
	for i, s := range st.seats {
		gu := s.m.gu
		if !s.m.mine() {
			pevForget(s.m)
			continue
		}
		pevRelease(s.m, r)
		if broken {
			g.alarm(gu, pevPick(r, pkrBreakLines))
		} else if i%2 == 0 && gu.icon == "" {
			gu.icon, gu.iconT = pevPick(r, pkrEndLines), 2
		}
	}
	st.seats = nil
	st.flying, st.cards = nil, nil
	g.pevRemoveProp(st.tbl)
}

// ---- drawing ------------------------------------------------------------------------------------------------

var (
	pkrCardBack = color.RGBA{0x3a, 0x58, 0xb8, 255}
	pkrChipCols = []color.RGBA{{0xe0, 0x30, 0x30, 255}, {0x30, 0x60, 0xe0, 255}, {0xf0, 0xf0, 0xf0, 255}, {0x30, 0xa0, 0x40, 255}}
)

// drawCardPix: a playing card of 3x4 (px 1): its back (blue) or its face (white with a red or black pip).
func pkrCardPix(plot plotFn, x, y, px int, face bool, red bool) {
	if !face {
		plot(x, y, 3*px, 4*px, pkrCardBack)
		plot(x+px, y+px, px, 2*px, color.RGBA{0x8a, 0xa8, 0xf0, 255})
		return
	}
	plot(x, y, 3*px, 4*px, pevWhite)
	pip := pevInk
	if red {
		pip = pevRed
	}
	plot(x+px, y+px, px, 2*px, pip)
}

func drawChipPix(plot plotFn, x, y, px int, col color.RGBA) {
	plot(x, y, 2*px, px, col)
	plot(x, y+px, 2*px, px, shade(col, 0.6))
}

func pokerItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*pkrState)
	if st.phase == pkDone || st.tbl == nil {
		return items
	}
	plot := g.plotter(dst)
	px := g.pevPx()
	// the chairs: the back behind the sitter (in front for the one with his back to us), the seat over his thighs
	for i := range pkrSeats {
		if !pkrSeatUsed(st, i) {
			continue
		}
		q := Vec{st.tbl.X + pkrSeats[i].off.X, st.tbl.Y + pkrSeats[i].off.Y}
		switch i {
		case 0: // behind the table: only the back shows
			items = append(items, drawItem{q.Y - 0.5, func() { x, y := g.pevXY(q); drawChairPix(plot, x, y, px, 0) }})
		case 3: // his back to us: the chair back in front of him
			items = append(items, drawItem{q.Y + 0.5, func() { x, y := g.pevXY(q); drawChairPix(plot, x, y+2*px, px, 0) }})
		default:
			items = append(items,
				drawItem{q.Y - 0.5, func() { x, y := g.pevXY(q); drawChairPix(plot, x, y, px, 0) }},
				drawItem{q.Y + 0.5, func() { x, y := g.pevXY(q); drawChairPix(plot, x, y+px, px, 1) }})
		}
	}
	// on the felt: the pot and the cards (drawn just after the table)
	items = append(items, drawItem{st.tbl.Y + 0.2, func() {
		pp := st.potPos()
		x, y := g.pevXY(pp)
		for i := 0; i < minI(maxI(st.pot, 0), 8); i++ {
			drawChipPix(plot, x-4*px+(i%4)*2*px, y-(i/4)*px, px, pkrChipCols[i%len(pkrChipCols)])
		}
		if st.phase == pkSit {
			return
		}
		for _, s := range st.seats {
			for k := 0; k < 2; k++ {
				cp := pkrCardAt(st, s, k)
				landed := true
				for _, cd := range st.cards {
					if cd.to == cp {
						landed = false
					}
				}
				if !landed || s.folded {
					continue
				}
				cx, cy := g.pevXY(cp)
				face := st.phase == pkShow
				pkrCardPix(plot, cx-px, cy-2*px, px, face, (s.seat+k)%2 == 0)
			}
		}
	}})
	// in the air: cards and chips
	for _, cd := range st.cards {
		cd := cd
		if g.time < cd.t0 {
			continue
		}
		items = append(items, drawItem{cd.to.Y + 30, func() {
			u := math.Min(1, (g.time-cd.t0)/pevCoinDur)
			p := Vec{cd.from.X + (cd.to.X-cd.from.X)*u, cd.from.Y + (cd.to.Y-cd.from.Y)*u}
			x, y := g.pevXY(p)
			pkrCardPix(plot, x-px, y-2*px-int(math.Round(6*math.Sin(math.Pi*u)))*px, px, false, false)
		}})
	}
	for i, ch := range st.flying {
		ch, i := ch, i
		items = append(items, drawItem{ch.to.Y + 30, func() {
			p, h, landed := pevCoinAt(ch, g.time)
			if landed {
				return
			}
			x, y := g.pevXY(p)
			drawChipPix(plot, x, y-int(math.Round(h*0.5))*px, px, pkrChipCols[i%len(pkrChipCols)])
		}})
	}
	return items
}

func pkrSeatUsed(st *pkrState, i int) bool {
	for _, s := range st.seats {
		if s.seat == i {
			return true
		}
	}
	return false
}

// pevDemoLog (captures with -pevent id:stage only): where the event happens, so the picture can be cropped there.
func pevDemoLog(id string, p Vec) {
	if pevDemo(id) != "" {
		fmt.Fprintf(os.Stderr, "PEVDEMO %s %.0f %.0f\n", id, p.X, p.Y)
	}
}
