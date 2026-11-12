package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// METAL DETECTOR (idea 315 of the user): in SOME parties of the casino, the club and the cinema a doorway between two
// rooms has a row of walk-through metal detector arches (pixel art drawn here, no props). The guests walk through it
// without a sound; a PLAYER (either of the two with 2 players) always carries his knife, so crossing it sets the alarm
// off: a short beep, the lights of the arches blink red, and the nearest free guard starts the ORDINARY guard chase
// (gdReact -> gdChase: the same leash, endurance and losing the target as any guard, so it can be shaken off). The
// guard stands a moment before he runs (mdReact: time to react, never an unfair end). The marked player wears the
// handcuffs mark (drawCuffs, the same symbol of suspect.go/twojail.go) while the warning lasts (the guard is after him,
// at most mdMarkMax s). With no guard free, the nearest officer on scene comes to look at the door (cSearch).
// After an alarm the arches stay quiet for mdCooldown s: no alarm in a loop.
//
// It is NOT a party event (the mean of events per party does not go up): it is an element of the level, rolled with its
// OWN dice (mdDice, never g.rng) the first frame the level is played (mdActive), one party in 2.5 of those venues.
// Nothing solid is added: the arches are drawn in the gap of the door, so the walking map does not change.
//
// Debug: -metaldet on (forces the arches where they fit) or -metaldet trip (also walks the player through at 0.5 s,
// for captures). The progress is frozen like with every debug flag.
// ---------------------------------------------------------------------------

const (
	mdChance   = 0.4
	mdCooldown = 20.0 // seconds of silence after an alarm
	mdAlarmT   = 4.0  // seconds the arches blink red
	mdReact    = 1.2  // the guard stands this long before he runs
	mdMarkMax  = 12.0 // the cuffs mark lasts at most this long
	mdCopMark  = 6.0  // ...or this long when an officer comes to look
	mdArchH    = 34.0 // height of an arch (world px)
	mdArchW    = 40.0 // width of one arch: a door gets as many as fit
	mdBand     = 26.0 // the crossing box reaches this far on each side of the door
	mdGuardR   = 560.0
	mdMinW     = 56.0
	mdMaxW     = 130.0
)

// mdVenues: the parties where the arches may come (by Def.Icon).
var mdVenues = map[string]bool{"casino": true, "disco": true, "cinema": true}

var (
	mdRng     *rand.Rand
	mdSeed    int64 // tests: non-zero = deterministic dice
	mdOff     bool  // tests: never
	mdForce   bool  // tests: for sure where they fit
	mdInTests bool  // tests: roll the real dice under `go test`
	mdFlag    = flag.String("metaldet", "", "debug: metal detector in the level given: on | trip (walks the player through)")
)

var (
	mdGuardLines = pevTexts("¡Alto! ¿Qué llevas ahí?", "¡Eh, tú! ¡Eso ha pitado!", "¡Control! ¡Quieto ahí!")
	mdCopLines   = pevTexts("¿Quién ha hecho pitar eso?")
	mdSayLine    = pevTexts("¡Ha pitado el detector de metales!")
)

func mdDice() *rand.Rand {
	if mdRng == nil {
		sd := mdSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x3e7a1d
		}
		mdRng = rand.New(rand.NewSource(sd))
	}
	return mdRng
}

func mdDebug() bool { return mdFlag != nil && *mdFlag != "" }

// mdFits: a venue of the detector with the usual scale.
func mdFits(d *LevelDef) bool {
	return d != nil && d.Title != "" && mdVenues[d.Icon] && (d.CharScale == 0 || d.CharScale == 1)
}

// mdRun: the arches of the level being played.
type mdRun struct {
	lvl        *Level
	door       Rect // the gap of the door
	n          int  // arches side by side
	coolUntil  float64
	alarmUntil float64
	greenUntil float64
	side       map[*Player]float64 // the side of the door each player was on while in the box (0 = out of it)
	chaser     *Guard
	cop        *Cop
	target     *Player
	markUntil  float64
	trips      int
	staged     int
}

var (
	md       *mdRun
	mdNoneAt *Level // the last level looked at that has no arches
)

// mdActive: the arches of the level being played (nil = none). Decided once per level with the own dice.
func (g *Game) mdActive() *mdRun {
	if g.lvl == nil || g.lvl == mdNoneAt {
		return nil
	}
	if md != nil && md.lvl == g.lvl {
		return md
	}
	md = nil
	roll := mdDice().Float64()
	ok := !mdOff && mdFits(g.lvl.Def) && !g.stealth && g.levelIdx != specialJurassic
	if ok && !mdDebug() {
		switch {
		case g.dc != nil:
			ok = false
		case mdForce:
		case testing.Testing() && !mdInTests:
			ok = false
		default:
			ok = roll < mdChance
		}
	}
	if ok {
		if dr, found := g.lvl.mdFindDoor(mdDice()); found {
			md = &mdRun{lvl: g.lvl, door: dr, n: mdArches(dr.W), side: map[*Player]float64{}}
			if mdDebug() && !testing.Testing() {
				fmt.Printf("metaldet: door %.0f,%.0f %.0fx%.0f\n", dr.X, dr.Y, dr.W, dr.H)
			}
			return md
		}
	}
	mdNoneAt = g.lvl
	return nil
}

func mdArches(w float64) int { return int(math.Max(1, math.Floor(w/mdArchW))) }

// mdBox: where a crossing is watched (the gap and a strip on each side).
func (r *mdRun) box() Rect {
	return Rect{r.door.X - 2, r.door.Y - mdBand, r.door.W + 4, r.door.H + 2*mdBand}
}

// mdArchRect: what the arches cover when drawn (world).
func mdArchRect(dr Rect) Rect {
	b := dr.Y + dr.H
	return Rect{dr.X, b - mdArchH - 2, dr.W, mdArchH + 4}
}

// mdFindDoor: a doorway of a horizontal wall, between two reachable floors, away from the start, the dance floor,
// the performers and the VIP zone, with nothing drawn on it. false if none.
func (l *Level) mdFindDoor(r *rand.Rand) (Rect, bool) {
	d := l.Def
	if l.cw == 0 {
		return Rect{}, false
	}
	reach := l.reachFrom(l.NearestFree(d.Start))
	var cands []Rect
	for _, dr := range d.Doors {
		gap := dr.R
		if !dr.Horiz || gap.W < mdMinW || gap.W > mdMaxW {
			continue
		}
		if l.mdDoorOK(gap, reach) {
			cands = append(cands, gap)
		}
	}
	if len(cands) == 0 {
		return Rect{}, false
	}
	return cands[r.Intn(len(cands))], true
}

func (l *Level) mdDoorOK(gap Rect, reach []bool) bool {
	d := l.Def
	cx := gap.X + gap.W/2
	if dist(Vec{cx, gap.Y + gap.H/2}, d.Start) < 140 {
		return false
	}
	open := func(q Vec) bool {
		return l.Free(q) && !l.SolidP(q.X, q.Y) && !l.badGround(q) && !l.InPool(q) && reach[l.navIdx(q)]
	}
	// the middle of the door is open on both sides (the edges may have a prop nearby)
	if !open(Vec{cx, gap.Y - mdBand + 6}) || !open(Vec{cx, gap.Y + gap.H + mdBand - 6}) {
		return false
	}
	box := Rect{gap.X - 2, gap.Y - mdBand, gap.W + 4, gap.H + 2*mdBand}
	near := Rect{gap.X, gap.Y - 10, gap.W, gap.H + 20} // nothing stands in the gap itself
	arch := mdArchRect(gap)
	for _, o := range l.Props {
		if o.gone {
			continue
		}
		if pbOverlap(o.Foot.Grow(1), near) || !overlayKinds[o.Name] && pbOverlap(o.base().Grow(1), arch) {
			return false
		}
	}
	for _, pf := range d.Performers {
		if box.Grow(10).Has(pf.Pos.X, pf.Pos.Y) {
			return false
		}
	}
	for _, dz := range d.Dance {
		if dz.Grow(12).Hits(box) {
			return false
		}
	}
	if z := l.vip; z != nil {
		for _, rc := range []Rect{z.in, z.trig, z.gate} {
			if rc.Grow(20).Hits(box) {
				return false
			}
		}
	}
	return true
}

// ---- play ----------------------------------------------------------------------------------------

// updateMetalDet: the crossings, the alarm and the end of the warning (game.go, right after the roadworks).
func (g *Game) updateMetalDet(dt float64) {
	if g.mode != ModePlay {
		return
	}
	run := g.mdActive()
	if run == nil {
		return
	}
	g.mdStage(run)
	box := run.box()
	mid := run.door.Y + run.door.H/2
	for _, p := range g.players() {
		if p == nil || p.dead || p.jail.held || p.hidden || !box.Has(p.pos.X, p.pos.Y) {
			delete(run.side, p)
			continue
		}
		s := 1.0
		if p.pos.Y < mid {
			s = -1
		}
		prev := run.side[p]
		run.side[p] = s
		if prev != 0 && prev != s {
			g.mdTrip(run, p)
		}
	}
	// a guest in the gap: the arches only blink green (they carry nothing)
	for _, gu := range g.guests {
		if !gu.out() && run.door.Has(gu.pos.X, gu.pos.Y) {
			run.greenUntil = g.time + 0.35
			break
		}
	}
	// the warning lasts while the guard is after him
	if run.chaser != nil {
		gd := run.chaser
		if gd.dead || gd.state != gdReact && gd.state != gdChase && gd.state != gdBeat {
			run.chaser = nil
			run.markUntil = math.Min(run.markUntil, g.time)
		}
	}
	if run.target != nil && (run.target.dead || run.target.jail.held) {
		run.markUntil = g.time
	}
}

// mdTrip: a player walked through the arches.
func (g *Game) mdTrip(run *mdRun, p *Player) {
	if g.time < run.coolUntil {
		return
	}
	run.trips++
	run.coolUntil, run.alarmUntil = g.time+mdCooldown, g.time+mdAlarmT
	run.target, run.chaser, run.cop = p, nil, nil
	g.mdSfx()
	if g.snd != nil && g.snd.sfx != nil {
		g.snd.Play("metaldet")
	}
	g.say(mdSayLine[0])
	r := mdDice()
	if gd := g.mdNearestGuard(p.pos); gd != nil {
		gd.state, gd.timer = gdReact, mdReact
		if dist(gd.pos, p.pos) < 70 {
			gd.timer += 0.6 // right beside him: a little longer to react
		}
		gd.icon, gd.iconT = pevPick(r, mdGuardLines), 2
		gd.lastSeen, gd.path = p.pos, nil
		gd.chaseT, gd.lost, gd.sight = 0, 0, 1.6
		run.chaser = gd
		run.markUntil = g.time + mdMarkMax
		return
	}
	if c := g.mdNearestCop(p.pos); c != nil {
		c.state, c.timer = cSearch, 4
		c.path = g.lvl.Path(c.pos, run.door.Center())
		c.icon, c.iconT = mdCopLines[0], 2
		run.cop = c
		run.markUntil = g.time + mdCopMark
		return
	}
	run.markUntil = g.time + mdAlarmT
}

// mdNearestGuard: the nearest guard who can come (not VIP staff, not asleep, not already chasing).
func (g *Game) mdNearestGuard(at Vec) *Guard {
	var best *Guard
	bd := mdGuardR
	for _, gd := range g.guards {
		if gd.dead || gd.vip != 0 || gd.smogT > 0 || g.guardAsleep(gd) || len(gd.route) == 0 {
			continue
		}
		switch gd.state {
		case gdPatrol, gdWait, gdReturn, gdSearch, gdInvestigate:
		default:
			continue
		}
		if d := dist(gd.pos, at); d < bd {
			best, bd = gd, d
		}
	}
	return best
}

// mdNearestCop: the nearest officer on scene who is free to come and look.
func (g *Game) mdNearestCop(at Vec) *Cop {
	var best *Cop
	bd := 1e18
	for _, c := range g.cops {
		if c.medic || c.smogT > 0 || c.climb > 0 || c.state != cSweep && c.state != cSearch && c.state != cReturn {
			continue
		}
		if d := dist(c.pos, at); d < bd {
			best, bd = c, d
		}
	}
	return best
}

// mdFocus: the guard the alarm sent goes after the player who crossed, not the nearest one (2 players).
func (g *Game) mdFocus(gd *Guard) func() {
	if run := md; run != nil && run.lvl == g.lvl && run.chaser == gd && g.is2P() {
		if p := run.target; p != nil && !p.dead && !p.jail.held {
			old := g.p
			g.p = p
			return func() { g.p = old }
		}
	}
	return g.focus(gd.pos)
}

// mdMarked: the player wears the cuffs mark of the detector now.
func (g *Game) mdMarked(p *Player) bool {
	run := md
	return run != nil && run.lvl == g.lvl && run.target == p && g.time < run.markUntil
}

// mdSfx adds the beep of the arches the first time it is needed.
func (g *Game) mdSfx() {
	if g.snd == nil || g.snd.sfx == nil || g.snd.sfx["metaldet"] != nil {
		return
	}
	g.snd.sfx["metaldet"] = mix(tone(1480, 0.09, 0.4, 9, 1), shift(tone(1480, 0.09, 0.4, 9, 1), 0.15), shift(tone(1480, 0.14, 0.4, 7, 1), 0.30))
}

// mdStage: -metaldet trip walks the player through the arches at 0.5 s (captures).
func (g *Game) mdStage(run *mdRun) {
	if mdFlag == nil || *mdFlag != "trip" || g.p == nil || run.staged >= 2 {
		return
	}
	c := run.door.Center()
	switch {
	case run.staged == 0 && g.time >= 0.5:
		g.p.pos, run.staged = Vec{c.X, run.door.Y - mdBand + 6}, 1
	case run.staged == 1 && g.time >= 0.6:
		g.p.pos, run.staged = Vec{c.X, run.door.Y + run.door.H + 10}, 2
	}
}

// ---- drawing ---------------------------------------------------------------------------------------

// mdItems: the arches, sorted with the people by the foot of the door (draw.go).
func (g *Game) mdItems(dst *ebiten.Image, items []drawItem) []drawItem {
	run := g.mdActive()
	if run == nil {
		return items
	}
	return append(items, drawItem{run.door.Y + run.door.H, func() { g.mdDraw(dst, run) }})
}

func (g *Game) mdDraw(dst *ebiten.Image, run *mdRun) {
	k := func(v float64) float64 { return math.Round(v * wk) }
	dr := run.door
	alarm := g.time < run.alarmUntil
	blink := alarm && int(g.time*6)%2 == 0
	base := dr.Y + dr.H
	if alarm { // the red glow on the floor
		filledEllipse(dst, g.wx(dr.X+dr.W/2), g.wy(base+2), k(dr.W/2+10), k(9), rgba(0xff2020, 70))
	}
	aw := math.Floor(dr.W / float64(run.n))
	x0 := dr.X + math.Floor((dr.W-aw*float64(run.n))/2)
	frame, frameL, frameD, panel := rgb(0x9aa2ae), rgb(0xd4dae2), rgb(0x4a505c), rgb(0x2a2e38)
	for i := 0; i < run.n; i++ {
		ax := x0 + float64(i)*aw + 1
		w := aw - 2
		for _, px := range []float64{ax, ax + w - 5} {
			X, Y := g.wx(px), g.wy(base-mdArchH)
			fillRect(dst, X, Y, k(5), k(mdArchH), frame)
			fillRect(dst, X, Y, k(1), k(mdArchH), frameL)
			fillRect(dst, X+k(4), Y, k(1), k(mdArchH), frameD)
			fillRect(dst, X+k(1), Y+k(8), k(3), k(18), panel)
			// the LED strip of the post
			for j := 0.0; j < 4; j++ {
				col := rgb(0x2e6e3a)
				if blink {
					col = rgb(0xff3030)
				} else if alarm {
					col = rgb(0x7a1a1a)
				}
				fillRect(dst, X+k(2), Y+k(10+j*4), k(1), k(2), col)
			}
			fillRect(dst, X-k(1), g.wy(base-1), k(7), k(2), frameD) // the foot
		}
		// the top bar with its light
		X, Y := g.wx(ax), g.wy(base-mdArchH-5)
		fillRect(dst, X, Y, k(w), k(6), frame)
		fillRect(dst, X, Y, k(w), k(1), frameL)
		fillRect(dst, X, Y+k(5), k(w), k(1), frameD)
		fillRect(dst, X+k(4), Y+k(2), k(w-8), k(2), panel)
		lc := rgb(0x2a8a3a)
		switch {
		case blink:
			lc = rgb(0xff2a2a)
		case alarm:
			lc = rgb(0x8a1a1a)
		case g.time < run.greenUntil:
			lc = rgb(0x6cff7a)
		}
		fillRect(dst, X+k(w/2-3), Y+k(2), k(6), k(2), lc)
	}
}

// mdDrawMarks: the cuffs over the player the detector marked, while the warning lasts (draw.go, with the suspects').
func (g *Game) mdDrawMarks(dst *ebiten.Image) {
	if g.mode != ModePlay {
		return
	}
	for _, p := range g.players() {
		if g.mdMarked(p) && !p.hidden && !p.dead && !g.isSuspectP(p) {
			g.drawCuffs(dst, p.pos)
		}
	}
}
