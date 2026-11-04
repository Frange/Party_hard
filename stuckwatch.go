package main

import "math"

// ---------------------------------------------------------------------------
// Nobody stays stuck. Every entity that is meant to be walking (guests, guards,
// police, animals, the rex and the player when a direction is held) is watched:
// if it has not advanced for stuckLimit seconds it is moved a little, to the
// nearest free spot, preferably in the direction it was going, and its path is
// dropped so it chooses another.
// ---------------------------------------------------------------------------

const stuckLimit = 5.0

type stuckRec struct {
	pos Vec
	t   float64
}

var stuckRecs = map[any]*stuckRec{}
var stuckLast float64

// relocate is the nearest walkable spot to pos that is not closed by ok, leaning towards goal.
func (l *Level) relocate(pos, goal Vec, ok func(Vec) bool) (Vec, bool) {
	best, bd := pos, math.MaxFloat64
	found := false
	for r := 8.0; r <= 90 && !found; r += 8 {
		for a := 0.0; a < 2*math.Pi; a += math.Pi / 10 {
			p := Vec{pos.X + math.Cos(a)*r, pos.Y + math.Sin(a)*r}
			if p.X < 4 || p.Y < 4 || p.X > l.W-4 || p.Y > l.H-4 || !l.Free(p) || l.SolidP(p.X, p.Y) || (ok != nil && !ok(p)) {
				continue
			}
			// a little weight on the goal: it should come out on the side it was heading for
			if d := dist(p, goal)*0.6 + r; d < bd {
				best, bd, found = p, d, true
			}
		}
	}
	return best, found
}

// stuckCheck reports true when the entity was moved. wants: it should be going somewhere now.
func (g *Game) stuckCheck(key any, pos *Vec, wants bool, goal Vec, dt float64, ok func(Vec) bool) bool {
	rec := stuckRecs[key]
	if rec == nil {
		rec = &stuckRec{pos: *pos}
		stuckRecs[key] = rec
	}
	if !wants || dist(rec.pos, *pos) > 1.5 {
		rec.pos, rec.t = *pos, 0
		return false
	}
	rec.t += dt
	if rec.t < stuckLimit {
		return false
	}
	rec.t = 0
	if np, found := g.lvl.relocate(*pos, goal, ok); found {
		*pos = np
		rec.pos = np
		return true
	}
	return false
}

// watchAllStuck is called every frame of the party.
func (g *Game) watchAllStuck(dt float64) {
	if g.lvl == nil || g.mode != ModePlay {
		return
	}
	if g.time < stuckLast { // a new party
		stuckRecs = map[any]*stuckRec{}
	}
	stuckLast = g.time
	for _, gu := range g.guests {
		if gu.out() {
			continue
		}
		switch gu.state {
		case gGoto, gFlee, gScared, gRun, gToPhone, gToilet, gReport, gChatGo, gAlarm:
			goal := gu.pos
			if n := len(gu.path); n > 0 {
				goal = gu.path[n-1]
			}
			if g.stuckCheck(gu, &gu.pos, !gu.moving, goal, dt, nil) {
				gu.path, gu.state, gu.timer = nil, gIdle, 1
			}
		default:
			delete(stuckRecs, gu)
		}
	}
	for _, gd := range g.guards {
		if gd.dead {
			continue
		}
		walking := gd.state == gdPatrol || gd.state == gdChase || gd.state == gdSearch || gd.state == gdInvestigate || gd.state == gdReturn
		goal := gd.pos
		if n := len(gd.path); n > 0 {
			goal = gd.path[n-1]
		}
		if g.stuckCheck(gd, &gd.pos, walking && !gd.moving, goal, dt, nil) {
			gd.path = nil
		}
	}
	for _, c := range g.cops {
		goal := c.pos
		if n := len(c.path); n > 0 {
			goal = c.path[n-1]
		}
		walking := len(c.path) > 0 && c.state != cArrest
		if g.stuckCheck(c, &c.pos, walking && !c.moving, goal, dt, nil) {
			c.path = nil
		}
	}
	for _, a := range g.animals {
		if a.dead {
			continue
		}
		goal := a.pos
		if n := len(a.path); n > 0 {
			goal = a.path[n-1]
		}
		if g.stuckCheck(a, &a.pos, len(a.path) > 0 && !a.moving, goal, dt, nil) {
			a.path = nil
		}
	}
	if p := g.p; p != nil && !p.dead && p.knockT <= 0 && !p.hidden {
		// a direction is held and nothing happens
		dx, dy := 0.0, 0.0
		if g.walkBot {
			dx, dy = g.botDx, g.botDy
		} else {
			if actLeft.held() {
				dx--
			}
			if actRight.held() {
				dx++
			}
			if actUp.held() {
				dy--
			}
			if actDown.held() {
				dy++
			}
		}
		goal := Vec{p.pos.X + dx*30, p.pos.Y + dy*30}
		g.stuckCheck(p, &p.pos, dx != 0 || dy != 0, goal, dt, nil)
	}
	if r := g.rex; r != nil && r.state != jrSleep {
		goal := r.pos
		if n := len(r.path); n > 0 {
			goal = r.path[n-1]
		} else if r.cur.at != nil {
			goal = r.cur.at()
		}
		wants := (r.state == jrWander || r.state == jrChase) && !r.moving
		if g.stuckCheck(r, &r.pos, wants, goal, dt, func(p Vec) bool { return !g.lvl.InPool(p) }) {
			r.path, r.cur, r.state, r.repath = nil, rexPrey{}, jrWander, 0
		}
	}
}
