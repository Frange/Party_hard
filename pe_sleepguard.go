package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "guardia" (idea 274, GUARDIA DORMIDO): in some parties (0-1 per party, own dice) a security guard (the game's own
// guard sprite, with his cap) sleeps on a chair beside a door: "Zzz" and a snoring bubble. He is a Guard of g.guards that
// stays asleep (sleepGuards, one line in updateGuard) until something disturbs him:
//   - a player RUNS within sleepWakeRun, or drags a body within sleepWakeBody;
//   - somebody is killed within sleepWakeKill (a new body), or he is hit by a blast (he dies) or a cream/smoke daze.
// Then he jumps up, shouts "¡Eh!" and does what any guard does: if the player is in front of him he chases him (as a guard who
// saw him), if there is a body he goes to it and calls the police (callerGuard, as a guard who found a body). Never harder
// than an ordinary guard. Walking past him (not running) does not wake him: he lets you through.
// While he sleeps he sees nothing: he is no witness (witnessAct skips him) and finds no bodies. He can be killed asleep with
// the knife like any guard (killGuard: the guards' statistics), silently: nobody is alarmed unless somebody sees it.
// ---------------------------------------------------------------------------

const (
	sleepWakeRun  = 60.0  // a player running this close wakes him
	sleepWakeBody = 36.0  // a player carrying a body this close wakes him
	sleepWakeKill = 110.0 // a kill this close wakes him
)

var (
	sleepWakeLines = pevTexts("¡Eh!", "¡Eh! ¿Quién anda ahí?", "¡No estaba durmiendo!")
)

type sleepGuard struct {
	gd     *Guard
	chair  Vec
	lvl    *Level
	bodies int // bodies already seen when he fell asleep (new ones are noise)
	woke   float64
	why    string // "run", "body", "kill" (tests)
}

// sleepGuards: the sleeping guards of the party being played (a new level forgets them).
var sleepGuards = map[*Guard]*sleepGuard{}

// sleepIcons: indoor parties with doors (the guard sleeps beside one).
var sleepIcons = map[string]bool{"house": true, "casino": true, "cinema": true, "disco": true, "train": true, "cake": true, "boat": true}

func init() {
	registerPartyEvent(&partyEvent{ID: "guardia", Name: "Guardia dormido", Group: "guard",
		Prob: 0.14, AtMin: 2, AtMax: 40, Dur: 400,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && sleepIcons[t.icon] },
		Start:    sleepStart,
		Update:   sleepUpdate})
}

// sleepAsleep: this guard is asleep right now (no eyes, no ears for the rest of the game).
func (g *Game) guardAsleep(gd *Guard) bool {
	if len(sleepGuards) == 0 {
		return false
	}
	s := sleepGuards[gd]
	return s != nil && s.lvl == g.lvl && s.woke == 0 && !gd.dead
}

// sleepSpot: a free place beside a door (inside the room, against the wall), far from the start and the players.
func (g *Game) sleepSpot() (Vec, bool) {
	l := g.lvl
	r := partyDice()
	doors := append([]Door(nil), l.Def.Doors...)
	r.Shuffle(len(doors), func(i, j int) { doors[i], doors[j] = doors[j], doors[i] })
	demo := pevDemo("guardia") != ""
	if demo { // capture: the door nearest to the player first
		for i := 1; i < len(doors); i++ {
			for j := i; j > 0 && dist(doors[j].R.Center(), g.p.pos) < dist(doors[j-1].R.Center(), g.p.pos); j-- {
				doors[j], doors[j-1] = doors[j-1], doors[j]
			}
		}
	}
	for pass := 0; pass < 2; pass++ { // first the doors of the rooms (indoors), then any (a garden gate)
		if p, ok := g.sleepSpotPass(doors, pass == 0, demo); ok {
			return p, true
		}
	}
	return Vec{}, false
}

func (g *Game) sleepSpotPass(doors []Door, indoor, demo bool) (Vec, bool) {
	l := g.lvl
	for _, d := range doors {
		c := Vec{d.R.X + d.R.W/2, d.R.Y + d.R.H/2}
		var offs []Vec
		if d.Horiz {
			hw := d.R.W/2 + 12
			offs = []Vec{{-hw, 16}, {hw, 16}, {-hw, -16}, {hw, -16}}
		} else {
			hh := d.R.H/2 + 12
			offs = []Vec{{16, -hh}, {16, hh}, {-16, -hh}, {-16, hh}}
		}
		for _, o := range offs {
			p := Vec{c.X + o.X, c.Y + o.Y}
			if !l.Free(p) || l.SolidP(p.X, p.Y) || l.badGround(p) || l.InPool(p) || d.R.Grow(6).Has(p.X, p.Y) || indoor && l.Outdoor(p) {
				continue
			}
			if !demo && (dist(p, l.Def.Start) < 120 || !g.pevPlayerFar(p, 150)) {
				continue
			}
			if demo && dist(p, g.p.pos) < 60 {
				continue
			}
			ok := true
			for _, gd := range g.guards {
				if !gd.dead && dist(gd.pos, p) < 40 {
					ok = false
				}
			}
			if ok {
				return p, true
			}
		}
	}
	return Vec{}, false
}

func sleepStart(g *Game, a *partyAct) bool {
	if len(g.as.guardCS) == 0 || g.lvl == nil || g.guestRoom() < 1 { // he counts like any guard: only if the party has room (cap.go)
		return false
	}
	for gd, s := range sleepGuards { // another party's
		if s.lvl != g.lvl {
			delete(sleepGuards, gd)
		}
	}
	p, ok := g.sleepSpot()
	if !ok {
		return false
	}
	if pevDemo("guardia") != "" { // capture: the player a little way off
		g.p.pos = g.lvl.NearestFree(Vec{p.X + 70, p.Y + 20})
	}
	gd := &Guard{pos: p, cs: g.as.guardCS[0], route: []Vec{p}, state: gdWait, timer: 1e9, home: p, leash: 170, dir: DirDown}
	g.guards = append(g.guards, gd)
	g.total++ // one more to count (BAJAS x/N), like every guest or guard who comes in later
	sleepGuards[gd] = &sleepGuard{gd: gd, chair: p, lvl: g.lvl, bodies: len(g.bodies)}
	a.D = gd
	pevDemoLog("guardia", p)
	return true
}

// sleepUpdate: the event is only his arrival (he sleeps on through sleepGuards without taking a place of the running events);
// the capture (-pevent guardia:run) keeps it to make the player run past him.
func sleepUpdate(g *Game, a *partyAct, dt float64) bool {
	gd, _ := a.D.(*Guard)
	if gd == nil || pevDemo("guardia") != "run" {
		return true
	}
	if pevDemo("guardia") == "run" && g.guardAsleep(gd) && g.time-a.t0 > 2.5 { // capture: the player runs past him
		g.p.pos = g.lvl.NearestFree(Vec{gd.pos.X + 30, gd.pos.Y + 10})
		g.p.running, g.p.moving = true, true
	}
	return !g.guardAsleep(gd)
}

// sleepGuardTick (updateGuard): true = he is asleep (nothing else of the guard runs). Checks what wakes him.
func (g *Game) sleepGuardTick(gd *Guard, dt float64) bool {
	if !g.guardAsleep(gd) {
		return false
	}
	s := sleepGuards[gd]
	gd.sight, gd.moving, gd.state, gd.path = 0, false, gdWait, nil
	gd.timer, gd.dir = 1e9, DirDown
	gd.callT = 0
	if gd.smogT > 0 { // dazed by smoke or cream: he wakes up coughing
		g.sleepWake(s, "daze", nil)
		return true
	}
	for _, p := range g.players() {
		if p == nil || p.dead || p.hidden {
			continue
		}
		d := dist(p.pos, gd.pos)
		if p.running && p.moving && d < sleepWakeRun {
			g.sleepWake(s, "run", p)
			return false
		}
		if p.carrying != nil && d < sleepWakeBody {
			g.sleepWake(s, "body", p)
			return false
		}
	}
	for i := s.bodies; i < len(g.bodies); i++ {
		b := g.bodies[i]
		if dist(b.pos, gd.pos) < sleepWakeKill && g.time-b.born < 1 {
			g.sleepWake(s, "kill", nil)
			if g.callerAvailable() { // like a guard who found a body
				g.callerGuard = gd
				gd.callT = 3
				g.alertPos = b.pos
			}
			gd.state, gd.path = gdInvestigate, g.lvl.Path(gd.pos, b.pos)
			return false
		}
	}
	s.bodies = len(g.bodies)
	return true
}

// sleepWake: up he jumps. If a player is the reason (or right in front of him) he reacts like a guard who saw him.
func (g *Game) sleepWake(s *sleepGuard, why string, p *Player) {
	gd := s.gd
	s.woke, s.why = g.time, why
	gd.timer = 0.5
	gd.icon, gd.iconT = sleepWakeLines[0], 1.6
	if why == "run" || why == "body" {
		gd.icon = pevPick(partyDice(), sleepWakeLines)
	}
	g.snd.Play("alert")
	if p != nil {
		gd.dir = faceTo(gd.pos, p.pos)
		gd.state, gd.timer = gdReact, 0.4
		gd.lastSeen = p.pos
		gd.chaseT, gd.lost, gd.sight = 0, 0, 1.6
	}
}

// sleepGuardDraw (the guards' draw loop): the sleeping guard on his chair, "Zzz" floating up. true = drawn.
func (g *Game) sleepGuardDraw(dst *ebiten.Image, gd *Guard) bool {
	if !g.guardAsleep(gd) {
		return false
	}
	plot := g.plotter(dst)
	px := g.pevPx()
	k := g.pevK()
	cx, cy := g.pevXY(gd.pos)
	drawChairPix(plot, cx, cy, px, 0)
	g.drawActor(dst, gd.cs, AnimTired, g.time*0.8, Vec{gd.pos.X, gd.pos.Y}, DirDown, 2, false)
	drawChairPix(plot, cx, cy+px, px, 1)
	// the snoring bubble at his nose, growing and bursting
	ph := math.Mod(g.time*0.6, 1)
	rr := 1 + int(ph*3)
	bx, by := cx+k(5), cy-k(26)
	bub := color.RGBA{0xc8, 0xd8, 0xf0, 255}
	plot(bx-rr*px, by-(rr-1)*px, (2*rr+1)*px, (2*rr-1)*px, bub) // a round bubble: a cross of two bars
	plot(bx-(rr-1)*px, by-rr*px, (2*rr-1)*px, (2*rr+1)*px, bub)
	plot(bx-(rr-1)*px, by-(rr-1)*px, px, px, pevWhite)
	drawSleepZs(plot, cx+k(6), cy-k(36), px, g.time)
	return true
}

// drawSleepZs: three "z" that float up and get bigger, one after the other.
func drawSleepZs(plot plotFn, x, y, px int, t float64) {
	col, dark := color.RGBA{0xc8, 0xd8, 0xff, 255}, color.RGBA{0x2a, 0x34, 0x60, 255}
	for i := 0; i < 3; i++ {
		ph := math.Mod(t*0.5+float64(i)/3, 1)
		s := 2 + int(ph*3) // size of the z in px units
		zx := x + int(ph*10)*px + i*px
		zy := y - int(ph*22)*px
		for _, c := range []struct {
			d   int
			col color.RGBA
		}{{px, dark}, {0, col}} {
			plot(zx+c.d, zy+c.d, s*px, px, c.col)
			plot(zx+c.d, zy+c.d+(s-1)*px, s*px, px, c.col)
			for j := 1; j < s-1; j++ {
				plot(zx+c.d+(s-1-j)*px, zy+c.d+j*px, px, px, c.col)
			}
		}
	}
}
