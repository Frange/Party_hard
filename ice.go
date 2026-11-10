package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// ICE (hockey rink). Floor kinds "ice" and "rink" are slippery: whoever stands on them keeps their speed (inertia),
// turns slowly and glides to a stop. Off the ice everything is exactly as before (velocity is zeroed), so no other
// party changes. It works with a velocity per walker (Player/Guest/Guard/Cop .vel) that follows the wish speed with
// a slow acceleration, like Level.Slow is a coefficient of the ground (bands.go), but for inertia.
//
//   - the player (updatePlayer): iceMovePlayer, keys give the wish speed, not the speed
//   - guests, guards and police: followIce replaces follow() in their walk helpers (same arguments + the velocity)
//   - REACHABILITY: genValidate / the nav grid are geometric (a path exists or not); the ice only changes HOW a walker
//     follows the path. followIce captures waypoints in a radius, skips the ones it flew past and snaps onto the last
//     one, so everybody still arrives (TestIceWalkersArrive simulates it from the start to every point of interest).
// ---------------------------------------------------------------------------

const (
	iceAccel    = 2.2  // 1/s: how fast the speed follows the wish (instant on normal ground); turns take ~0.5 s
	iceFriction = 0.85 // 1/s: how fast a skater who stops pushing slows down
	iceGrip     = 0.0  // share of the speed kept when hitting a wall (0 = the component is lost)
)

// iceLevel is the level in play (set by loadLevel): followIce has no Game at hand.
var iceLevel *Level

func iceKind(kind string) bool { return kind == "ice" || kind == "rink" }

// hasIce: does the level have any slippery floor? (looked at once)
func (l *Level) hasIce() bool {
	if l == nil || l.Def == nil {
		return false
	}
	if l.iceOn == 0 {
		l.iceOn = -1
		for _, f := range l.Def.Floors {
			if iceKind(f.Kind) {
				l.iceOn = 1
				break
			}
		}
	}
	return l.iceOn > 0
}

// OnIce: is the ground under p slippery? The topmost floor that contains p decides.
func (l *Level) OnIce(p Vec) bool {
	if !l.hasIce() {
		return false
	}
	kind := ""
	for _, f := range l.Def.Floors {
		if f.R.Has(p.X, p.Y) {
			kind = f.Kind
		}
	}
	return iceKind(kind)
}

// iceStep moves vel one step toward the wish velocity (zero wish: friction only).
func iceStep(vel *Vec, wish Vec, dt float64) {
	if wish.X == 0 && wish.Y == 0 {
		k := math.Exp(-iceFriction * dt)
		vel.X *= k
		vel.Y *= k
		if math.Hypot(vel.X, vel.Y) < 1.5 {
			*vel = Vec{}
		}
		return
	}
	k := 1 - math.Exp(-iceAccel*dt)
	vel.X += (wish.X - vel.X) * k
	vel.Y += (wish.Y - vel.Y) * k
}

// solidNPC: the test walkers use (the same solids as their path finder: a plain point inside a rectangle), so that
// they can stand where the nav grid says they can (a use point right next to a prop), unlike the bigger player box.
func (l *Level) solidNPC(x, y float64) bool {
	for _, r := range l.solidsN {
		if r.Has(x, y) {
			return true
		}
	}
	return false
}

// iceMove moves pos by vel*dt, axis by axis, and loses the component that hits something solid (npc: with solidNPC).
func iceMove(l *Level, pos, vel *Vec, dt float64, npc ...bool) {
	solid := l.SolidP
	if len(npc) > 0 && npc[0] {
		solid = l.solidNPC
	}
	nx := pos.X + vel.X*dt
	if !solid(nx, pos.Y) {
		pos.X = nx
	} else {
		vel.X *= -iceGrip
	}
	ny := pos.Y + vel.Y*dt
	if !solid(pos.X, ny) {
		pos.Y = ny
	} else {
		vel.Y *= -iceGrip
	}
}

// iceMovePlayer: the movement of a player standing on ice. It returns false when the ground is normal (the caller
// then moves as always). dx,dy is the raw input direction, speed the wish speed already multiplied by every perk.
func (g *Game) iceMovePlayer(p *Player, dx, dy, speed, dt float64) bool {
	if !g.lvl.OnIce(p.pos) {
		p.vel = Vec{}
		return false
	}
	var wish Vec
	if p.moving {
		l := math.Hypot(dx, dy)
		dx, dy = dx/l, dy/l
		wish = Vec{dx * speed * paceK, dy * speed * paceK}
		if math.Abs(dx) > math.Abs(dy) {
			if dx < 0 {
				p.dir = DirLeft
			} else {
				p.dir = DirRight
			}
		} else if dy < 0 {
			p.dir = DirUp
		} else {
			p.dir = DirDown
		}
	}
	before := p.pos
	iceStep(&p.vel, wish, dt)
	iceMove(g.lvl, &p.pos, &p.vel, dt)
	if v := math.Hypot(p.vel.X, p.vel.Y); v > 3 {
		p.anim += dt * stridePhase(v/paceK) * 0.8 // the legs keep skating while it glides
		g.iceTrace(p.pos, p.vel)
	}
	if p.moving {
		g.watchStuck(before, dx, dy, dt)
	}
	return true
}

// followIce is follow() for walkers that may be on ice: off the ice it IS follow() (and the velocity is cleared).
func followIce(pos, vel *Vec, dir *Dir, path *[]Vec, speed, dt float64) (moved, arrived bool) {
	l := iceLevel
	if l == nil || !l.hasIce() || !l.OnIce(*pos) {
		if l != nil && (vel.X != 0 || vel.Y != 0) && len(*path) > 0 {
			// stepped off the ice with momentum: the skater drifted away from the line of his path, so a straight walk
			// to the next waypoint could cut through something; plan the rest again from where he is
			if np := l.Path(*pos, (*path)[len(*path)-1]); np != nil {
				*path = np
			}
		}
		*vel = Vec{}
		return follow(pos, dir, path, speed, dt)
	}
	if len(*path) == 0 { // nothing to walk to: glide to a stop
		if vel.X == 0 && vel.Y == 0 {
			return false, true
		}
		iceStep(vel, Vec{}, dt)
		iceMove(l, pos, vel, dt, true)
		iceFaceVel(dir, *vel)
		return false, true
	}
	t := (*path)[0]
	dx, dy := t.X-pos.X, t.Y-pos.Y
	d := math.Hypot(dx, dy)
	last := len(*path) == 1
	passed := d < 18 && vel.X*dx+vel.Y*dy < 0 // flew past it
	switch {
	case last && d < 9:
		*pos, *vel = t, Vec{}
		*path = (*path)[:0]
		return true, true
	case !last && (d < 14 || passed):
		*path = (*path)[1:]
		return true, false
	}
	w := speed
	if last { // slow down on the approach so that it stops on the spot
		w = math.Min(speed, d*2.4)
	}
	wish := Vec{dx / d * w, dy / d * w}
	iceStep(vel, wish, dt)
	iceMove(l, pos, vel, dt, true)
	iceFaceVel(dir, Vec{dx, dy})
	if iceFx.g != nil {
		iceFx.g.iceTrace(*pos, *vel)
	}
	return true, false
}

func iceFaceVel(dir *Dir, v Vec) {
	if math.Hypot(v.X, v.Y) < 3 {
		return
	}
	if math.Abs(v.X) > math.Abs(v.Y)*1.3 {
		if v.X < 0 {
			*dir = DirLeft
		} else {
			*dir = DirRight
		}
	} else if math.Abs(v.Y) > math.Abs(v.X)*1.3 {
		if v.Y < 0 {
			*dir = DirUp
		} else {
			*dir = DirDown
		}
	}
}

// ---- skate marks: faint short lines that fade, nothing allocated per frame ----------------------------------

const iceMarkN = 160

type iceMark struct {
	x0, y0, x1, y1 float64
	born           float64
	ok             bool
}

type iceFxState struct {
	g     *Game
	marks [iceMarkN]iceMark
	n     int
	acc   float64
}

var iceFx iceFxState

// iceTrace leaves a skate mark behind a skater who is moving fast enough (called for everybody on the ice).
func (g *Game) iceTrace(pos, vel Vec) {
	f := &iceFx
	f.g = g
	v := math.Hypot(vel.X, vel.Y)
	if v < 12 {
		return
	}
	f.acc += v
	if f.acc < 90 { // roughly one mark per 90 px of travel in total: sparse and subtle
		return
	}
	f.acc = 0
	m := &f.marks[f.n%iceMarkN]
	f.n++
	k := 7 / v
	*m = iceMark{pos.X - vel.X*k, pos.Y - vel.Y*k, pos.X, pos.Y, g.time, true}
}

// drawIceMarks paints the marks on the ice (1 px lines, fading over 5 s).
func (g *Game) drawIceMarks(dst *ebiten.Image) {
	if g.lvl == nil || !g.lvl.hasIce() {
		return
	}
	f := &iceFx
	for i := range f.marks {
		m := &f.marks[i]
		if !m.ok {
			continue
		}
		age := g.time - m.born
		if age < 0 || age > 5 {
			m.ok = false
			continue
		}
		a := uint8(70 * (1 - age/5))
		if a == 0 {
			continue
		}
		x0, y0, x1, y1 := g.wx(m.x0), g.wy(m.y0), g.wx(m.x1), g.wy(m.y1)
		n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
		for s := 0; s <= n; s++ {
			t := float64(s) / float64(n)
			fillRect(dst, math.Round(x0+(x1-x0)*t), math.Round(y0+(y1-y0)*t), 1, 1, rgba(0xdff2ff, a))
		}
	}
}

// iceBegin: a party starts: remember the level and forget the marks of the last one.
func iceBegin(g *Game) {
	iceFx = iceFxState{g: g}
	iceLevel = g.lvl
}
