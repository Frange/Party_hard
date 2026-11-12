package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "tablehide" (reactive): when there is danger (a killer hunting, three or more guests running scared: gunfire, a blast,
// a murder) in a party that has cloth tables (pe_tables.go), SOME of the guests close to a table run under it. They are not
// drawn: the cloth is lifted at the front and only their feet show, with a "¡Shh!" over the table. They come out some seconds
// after the danger is over. A guest who is killed, or who is taken by something else, is simply let go.
//
// WHEN: the parties that have tables (tablesIcons), each time a danger starts, with a 65 % chance per episode, at most four
// guests (two places per table). It rolls no dice of the plan (Reactive) and does not count for the maximum of three.
// ---------------------------------------------------------------------------

const (
	hideMax     = 4
	hideNear    = 280.0
	hideOutWait = 3.5  // seconds without danger before they come out
	hideMaxHold = 30.0 // they never stay in longer than this, whatever happens outside
	hideCool    = 90.0 // after coming out, no new hiding in this party for this long
	hideChance  = 0.65 // per episode of danger
)

var hideLines = pevTexts("¡Shh!", "Silencio...", "No hagas ruido", "¡Que no nos vea!", "Aquí no nos ven...")
var hideOutLines = pevTexts("Uf... ¿ya pasó?", "Creo que ya se fue", "Qué susto...")

// hideEpisode: this danger has already been dealt with (one roll per episode, hideRolled); both are reset when the danger is
// over. hideNextTry spaces out the searches while nobody is close enough to a table.
var hideEpisode, hideRolled bool
var hideNextTry float64

// hideCoolLvl/hideCoolTo: the party where they came out last, and the game time before which they do not hide again.
var hideCoolLvl *Level
var hideCoolTo float64

// hideDanger: something to be afraid of right now.
func hideDanger(g *Game) bool {
	if dbgPartyEvent == "tablehide" { // -pevent tablehide: a made-up danger from second 3 to 40
		return g.time >= 3 && g.time < 40
	}
	for _, s := range g.allSlashers() {
		if s.state == slHunt {
			return true
		}
	}
	n := 0
	for _, gu := range g.guests {
		if !gu.out() && (gu.state == gScared || gu.state == gFlee) {
			if n++; n >= 3 {
				return true
			}
		}
	}
	return false
}

type hideOne struct {
	m    *pevMem
	tbl  *Prop
	slot Vec
	in   bool // already under the table
	t0   float64
	next float64
}

type hideState struct {
	h       []*hideOne
	t0      float64
	calmFor float64
}

func (s *hideState) pevHas(gu *Guest) bool {
	for _, h := range s.h {
		if h.m.gu == gu {
			return true
		}
	}
	return false
}

func init() {
	registerPartyEvent(&partyEvent{ID: "tablehide", Name: "Escondidos bajo la mesa", Group: "danger", Reactive: true,
		Prob: 0.10, AtMin: 0, AtMax: 1, Dur: 45, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool {
			if tablesIcons[t.icon] == 0 && !seatHideIcon(t.icon) || g.lvl == nil { // cloth tables, or the long tables of seats.go
				return false
			}
			if !hideDanger(g) {
				hideEpisode, hideRolled = false, false
				return false
			}
			return !hideEpisode
		},
		Start: tableHideStart, Items: tableHideItems,
		Update: pevGuard(tableHideUpdate, func(g *Game, a *partyAct) { tableHideEnd(g, a.D.(*hideState)) })})
}

// hidePathOK: a way to the place that is not much longer than the straight line (they run to the nearest table, not round the party).
func hidePathOK(g *Game, from, to Vec) bool {
	path := g.lvl.PathG(from, to)
	if path == nil {
		return false
	}
	n, prev := 0.0, from
	for _, p := range path {
		n += dist(prev, p)
		prev = p
	}
	return n <= dist(from, to)*1.6+30 && n <= 340
}

func hideCanGo(g *Game, gu *Guest) bool {
	if !lifeFree(gu) || gu.quirk != "" || gu.leader != nil || gu.armed || gu.partner != nil || gu.name != "" || g.pevTaken(gu) ||
		pevFxs[gu] != nil || g.inSelfie(gu) || g.inContest(gu) || starSlugs[guestSlug(gu)] {
		return false
	}
	switch gu.state {
	case gIdle, gGoto, gDance, gLounge, gScared, gFlee:
		return gu.timer < 1000
	}
	return false
}

func tableHideStart(g *Game, a *partyAct) bool {
	if g.time < hideNextTry || hideCoolLvl == g.lvl && g.time < hideCoolTo && dbgPartyEvent != "tablehide" {
		return false
	}
	r := partyDice()
	if !hideRolled {
		hideRolled = true
		if dbgPartyEvent != "tablehide" && r.Float64() >= hideChance {
			hideEpisode = true // not this time
			return false
		}
	}
	type place struct {
		tbl  *Prop
		slot Vec
	}
	var places []place
	for _, pr := range g.lvl.Props {
		if !seatHideKinds[pr.Name] || pr.gone {
			continue
		}
		for _, s := range tableSlots(g.lvl, Vec{pr.X, pr.Y}) {
			if s != (Vec{}) {
				places = append(places, place{pr, s})
			}
		}
	}
	if len(places) == 0 {
		hideEpisode = true
		return false
	}
	r.Shuffle(len(places), func(i, j int) { places[i], places[j] = places[j], places[i] })
	st := &hideState{t0: g.time}
	for _, pl := range places {
		if len(st.h) >= hideMax {
			break
		}
		var best *Guest
		for _, gu := range g.guests {
			if !hideCanGo(g, gu) || (st.pevHas(gu)) {
				continue
			}
			if d := dist(gu.pos, pl.slot); d < hideNear && (best == nil || d < dist(best.pos, pl.slot)) && hidePathOK(g, gu.pos, pl.slot) {
				best = gu
			}
		}
		if best == nil {
			continue
		}
		m := &pevMem{gu: best}
		if !g.pevSendTo(m, pl.slot) {
			continue
		}
		best.icon, best.iconT = "", 0
		st.h = append(st.h, &hideOne{m: m, tbl: pl.tbl, slot: pl.slot, t0: g.time, next: g.time + 2})
	}
	if len(st.h) == 0 {
		hideNextTry = g.time + 1.5 // nobody close enough: look again in a moment
		return false
	}
	hideEpisode = true
	pevFxPurge(g)
	a.D = st
	return true
}

func hideOutOf(h *hideOne) {
	if f := pevFxs[h.m.gu]; f != nil {
		f.hid = nil
	}
	pevFxDrop(h.m.gu)
}

func tableHideUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*hideState)
	r := partyDice()
	live := st.h[:0]
	for _, h := range st.h {
		gu := h.m.gu
		if !h.m.mine() { // killed, scared again by something else...: not ours any more
			hideOutOf(h)
			continue
		}
		if !h.in {
			if gu.state == gIdle && gu.timer > 1000 { // arrived
				h.in = true
				h.m.want = gIdle
				gu.pos = h.slot
				gu.dir = DirUp
				f := pevFxOf(gu)
				f.hid, f.hidX = h.tbl, h.slot.X
				gu.icon, gu.iconT = pevPick(r, hideLines[:1]), 2.2
			} else if g.time-h.t0 > 9 { // never got there: he stays where he was
				pevRelease(h.m, r)
				continue
			}
		} else {
			h.m.keep()
			if g.time >= h.next {
				h.next = g.time + 4 + r.Float64()*3
				if gu.icon == "" {
					gu.icon, gu.iconT = pevPick(r, hideLines), 2
				}
			}
		}
		live = append(live, h)
	}
	st.h = live
	if len(st.h) == 0 {
		return true
	}
	if g.time-st.t0 > hideMaxHold {
		tableHideEnd(g, st)
		return true
	}
	if hideDanger(g) {
		st.calmFor = 0
	} else if st.calmFor += dt; st.calmFor >= hideOutWait {
		tableHideEnd(g, st)
		return true
	}
	return false
}

func tableHideEnd(g *Game, st *hideState) {
	r := partyDice()
	hideCoolLvl, hideCoolTo = g.lvl, g.time+hideCool
	for i, h := range st.h {
		gu := h.m.gu
		wasMine := h.m.mine()
		hideOutOf(h)
		if wasMine {
			pevRelease(h.m, r)
			if h.in && i%2 == 0 {
				gu.icon, gu.iconT = pevPick(r, hideOutLines), 2
			}
		}
	}
	st.h = nil
}

func tableHideItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*hideState)
	plot := g.plotter(dst)
	for _, h := range st.h {
		h := h
		if !h.in || !h.m.mine() {
			continue
		}
		items = append(items, drawItem{h.tbl.Y + 0.5, func() {
			sx, sy := g.pevXY(Vec{h.slot.X, h.tbl.Y})
			// the cloth is lifted at the front: a dark opening with the feet inside it
			plot(sx-7, sy-8, 14, 8, color.RGBA{0x24, 0x1a, 0x2a, 255})
			g.pevDrawFeet(dst, h.m.gu.cs, Vec{h.slot.X, h.tbl.Y - 2}, 6)
		}})
	}
	return items
}
