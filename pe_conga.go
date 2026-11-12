package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "conga": a line of guests dancing the conga crosses the party. The leader follows a route through the
// roaming area; the others walk his exact steps (pevTrail) one behind the other, dancing, hands on the waist of
// the one in front. Passing guests join (up to 10); when the route ends the line breaks up and everybody goes back
// to his business.
//
// WHEN: every party with a dance floor and room to walk (congaIcons: 25 of the 26 of the campaign, all but the space
// station; 2026-11-11, owner: "nunca la he visto"); 35 % of those, between 70 and 240 s, up to twice (cooldown 150 s).
// Where the 520 px route does not fit (a small or crowded floor) a SHORT conga of 4 to 6 dancers on a route of at least
// congaShortRoute px runs instead of nothing. Group "dance" and Big: it never runs together with the dance contest, the musical
// chairs or the hen/stag party.
// ---------------------------------------------------------------------------

// congaIcons: the parties where a conga fits. Since 2026-11-11 every campaign party whose dance floor and roaming area
// hold the route (measured: a forced conga starts in all of them, TestPevCongaEveryVenue), except the space station
// (zero gravity and corridors). The pirate ship adds itself (pirate_text.go), ITALIA too (italy_text.go).
var congaIcons = map[string]bool{"house": true, "neon": true, "disco": true, "disco70": true, "club": true, "beach": true,
	"cake": true, "festival": true, "boat": true, "ranch": true, "bull": true, "eiffel": true, "funfair": true,
	"stadium": true, "skate": true, "grave": true, "cinema": true, "circuit": true, "snow": true, "roof": true,
	"casino": true, "highway": true, "train": true, "hockey": true}

const (
	congaMax   = 10
	congaGap   = 13.0 // pixels between two dancers along the trail
	congaSpeed = 24.0
	congaJoinR = 26.0 // a guest this close to the line may join
	congaPull  = 46.0 // speed of the one who hurries to his place
	congaRoute = 520.0
	// the short conga: where the long route does not fit, a route of at least this length and 4-6 dancers
	congaShortRoute = 200.0
	congaShortMax   = 6
	congaSmallArea  = 60000.0 // px² of roaming area: under this, the short conga (no campaign party is that small today)
)

var congaLines = pevTexts("¡UNO, DOS, TRES... CONGA!", "¡Conga, conga, conga!", "¡Que no pare la conga!", "¡Me apunto!", "¡Hacedme sitio!", "¡Conga!",
	"¡Más rápido!", "¡Uf, qué sed!", "¡Mis pies!", "¡Otra vuelta!")

type congaState struct {
	line  []*pevMem // line[0] is the leader
	trail pevTrail
	route []Vec
	end   float64 // game time when it breaks up (0 = the route is not over)
	go_   float64 // the leader waits until then for the first dancers to come
	max   int     // dancers at most: congaMax, or congaShortMax on a short route
	nextJ float64
	said  float64
}

func (c *congaState) pevHas(gu *Guest) bool { return pevHasIn(c.line, gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "conga", Name: "Conga", Group: "dance", Big: true,
		Prob: 0.35, AtMin: 70, AtMax: 240, Dur: 28, Cooldown: 150, MaxRuns: 2, Prio: 1,
		Eligible: func(g *Game, t partyTraits) bool { return t.dance && congaIcons[t.icon] && t.roam && t.guests >= 24 },
		Start:    congaStart, Items: congaItems,
		Update: pevGuard(congaUpdate, func(g *Game, a *partyAct) { pevReleaseAll(a.D.(*congaState).line, partyDice()) })})
}

// congaSmall: the roaming area of the party is too small for the long route (congaSmallArea px² in all).
func congaSmall(d *LevelDef) bool {
	a := 0.0
	for _, r := range d.Roam {
		a += r.W * r.H
	}
	return a < congaSmallArea
}

func congaFloor(d *LevelDef) Vec {
	fl := d.Dance[0]
	for _, f := range d.Dance {
		if f.W*f.H > fl.W*fl.H {
			fl = f
		}
	}
	return Vec{fl.X + fl.W/2, fl.Y + fl.H/2}
}

func congaCanJoin(g *Game, gu *Guest) bool {
	if !g.pevFreeGuest(gu) {
		return false
	}
	return gu.state == gIdle || gu.state == gGoto || gu.state == gDance || gu.state == gLounge
}

func congaStart(g *Game, a *partyAct) bool {
	r := partyDice()
	center := congaFloor(g.lvl.Def)
	var cand []*Guest
	for _, gu := range g.guests {
		if congaCanJoin(g, gu) && dist(gu.pos, center) < 150 {
			cand = append(cand, gu)
		}
	}
	if len(cand) < 5 {
		return false
	}
	// the leader: the free guest nearest to the middle of the floor
	lead := cand[0]
	for _, gu := range cand {
		if dist(gu.pos, center) < dist(lead.pos, center) {
			lead = gu
		}
	}
	// the route: a few points of the roaming area, far enough from each other, until it is long enough
	var route []Vec
	at, total := lead.pos, 0.0
	want, lim := congaRoute, congaMax
	if congaSmall(g.lvl.Def) { // a small roaming area: the short conga from the start
		want, lim = congaShortRoute*1.5, congaShortMax
	}
	for k := 0; k < 8 && total < want; k++ {
		best := Vec{}
		bd := 0.0
		for i := 0; i < 6; i++ {
			p := g.lvl.RandomRoam(r)
			if d := dist(p, at); d > bd && d > 70 {
				best, bd = p, d
			}
		}
		if bd == 0 {
			continue
		}
		leg := g.lvl.PathG(at, best)
		if len(leg) == 0 {
			continue
		}
		for _, p := range leg {
			total += dist(at, p)
			at = p
		}
		route = append(route, leg...)
	}
	if total < want*0.6 { // no room for the long one: the short conga, or nothing
		if total < congaShortRoute || len(route) == 0 {
			return false
		}
		lim = congaShortMax
	}
	st := &congaState{route: route, nextJ: g.time + 1.5, go_: g.time + 2.5, max: lim}
	lm := &pevMem{gu: lead}
	lm.hold(gDance)
	st.line = []*pevMem{lm}
	st.trail.add(lead.pos)
	st.trail.prefix(lead.pos, faceVec(lead.pos, route[0]), func(p Vec) bool { return g.lvl.Free(p) && !g.lvl.SolidP(p.X, p.Y) })
	// the first dancers: the nearest ones to the leader who can walk straight to him
	for pass := 0; pass < 3 && len(st.line) < 4; pass++ {
		var best *Guest
		for _, gu := range cand {
			if gu == lead || pevHasIn(st.line, gu) || !g.lvl.clearLine(gu.pos, lead.pos) || dist(gu.pos, lead.pos) > 90 {
				continue
			}
			if best == nil || dist(gu.pos, lead.pos) < dist(best.pos, lead.pos) {
				best = gu
			}
		}
		if best == nil {
			break
		}
		m := &pevMem{gu: best, next: 1}
		m.hold(gDance)
		st.line = append(st.line, m)
	}
	if len(st.line) < 3 {
		lm.want = gDance
		pevRelease(lm, r)
		for _, m := range st.line[1:] {
			pevRelease(m, r)
		}
		return false
	}
	lead.icon, lead.iconT = congaLines[0], 3
	a.D = st
	return true
}

func congaUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*congaState)
	r := partyDice()
	// the leader must be ours; the others who are not are simply dropped from the line
	if !st.line[0].mine() {
		pevReleaseAll(st.line[1:], r)
		return true
	}
	lead := st.line[0].gu
	keep := st.line[:1]
	for _, m := range st.line[1:] {
		if m.mine() {
			keep = append(keep, m)
		}
	}
	st.line = keep
	if len(st.line) < 2 {
		pevRelease(st.line[0], r)
		return true
	}
	// the leader walks the route
	if st.end == 0 && g.time >= st.go_ {
		if _, arr := follow(&lead.pos, &lead.dir, &st.route, congaSpeed, dt); arr {
			st.end = g.time
		}
		st.trail.add(lead.pos)
		if g.time-a.t0 > a.ev.Dur+6 {
			st.end = g.time
		}
	}
	if st.end > 0 {
		congaBreakUp(st)
		return true
	}
	// the others walk his steps (the new one hurries to his place)
	for i, m := range st.line {
		m.keep()
		if i == 0 {
			continue
		}
		slot := st.trail.at(float64(i) * congaGap)
		gu := m.gu
		if m.next > 0 && dist(gu.pos, slot) > 3 { // joining: he runs to his place dancing
			v := congaPull * dt
			d := dist(gu.pos, slot)
			if d <= v {
				gu.pos, m.next = slot, 0
			} else {
				gu.pos = Vec{gu.pos.X + (slot.X-gu.pos.X)/d*v, gu.pos.Y + (slot.Y-gu.pos.Y)/d*v}
				gu.dir = faceTo(gu.pos, slot)
			}
			continue
		}
		m.next = 0
		prev := st.line[i-1].gu.pos
		if dist(gu.pos, slot) > 0.01 {
			gu.dir = faceTo(gu.pos, slot)
		} else {
			gu.dir = faceTo(gu.pos, prev)
		}
		gu.pos = slot
	}
	// more guests join as the line passes
	if g.time >= st.nextJ && len(st.line) < st.max && st.trail.length() > float64(len(st.line)+1)*congaGap {
		st.nextJ = g.time + 0.7
		for _, gu := range g.guests {
			if len(st.line) >= st.max {
				break
			}
			if !congaCanJoin(g, gu) || pevHasIn(st.line, gu) {
				continue
			}
			near := false
			for _, m := range st.line {
				if dist(m.gu.pos, gu.pos) < congaJoinR {
					near = true
					break
				}
			}
			if !near || r.Float64() > 0.4 || !g.lvl.clearLine(gu.pos, st.trail.at(float64(len(st.line))*congaGap)) {
				continue
			}
			m := &pevMem{gu: gu, next: 1}
			m.hold(gDance)
			st.line = append(st.line, m)
			gu.icon, gu.iconT = pevPick(r, congaLines[3:6]), 1.6
		}
	}
	// the leader shouts now and then
	if g.time-st.said > 5 && lead.icon == "" {
		st.said = g.time
		lead.icon, lead.iconT = pevPick(r, []string{congaLines[1], congaLines[2], congaLines[6], congaLines[9]}), 2
	}
	return false
}

func congaBreakUp(st *congaState) {
	rr := partyDice()
	k := 0
	for _, m := range st.line {
		if m.mine() && k < 2 && rr.Float64() < 0.5 {
			m.gu.icon, m.gu.iconT = pevPick(rr, []string{congaLines[7], congaLines[8]}), 2.4
			k++
		}
	}
	pevReleaseAll(st.line, rr)
}

// drawCongaHandsPix: the hands of one dancer on the waist of the one in front (screen points: his hip and hers).
func drawCongaHandsPix(plot plotFn, fx, fy, tx, ty, px int, skin color.RGBA) {
	drawLinePix(plot, fx, fy, tx, ty, px, skin)
	plot(fx-px, fy-px, 2*px, 2*px, skin)
}

func congaItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*congaState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for i := 1; i < len(st.line); i++ {
		m, pm := st.line[i], st.line[i-1]
		if !m.mine() || !pm.mine() {
			continue
		}
		items = append(items, drawItem{m.gu.pos.Y + 0.1, func() {
			fx, fy := g.pevXY(m.gu.pos)
			tx, ty := g.pevXY(pm.gu.pos)
			if absI(fx-tx)+absI(fy-ty) > k(30) {
				return
			}
			drawCongaHandsPix(plot, fx, fy-k(20), tx, ty-k(18), px, m.gu.cs.Spec.Skin)
		}})
	}
	if lead := st.line[0].gu; st.end == 0 && lead != nil {
		items = append(items, drawItem{lead.pos.Y + 0.2, func() {
			cx, cy := g.pevXY(lead.pos)
			drawNotesPix(plot, cx, cy-k(46), px, g.time, 3)
		}})
	}
	return items
}
