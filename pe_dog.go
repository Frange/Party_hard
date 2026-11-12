package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "dog": a stray dog trots into an OUTDOOR party. If there is a corpse lying about it sniffs it, grabs it and
// DRAGS it to the edge of the level (the pavement side) and leaves with it.
//
// DECISIONS
//   - The dog is its own little Animal that is NOT in g.animals: animalMax / the bull cap are never touched, and it is
//     not a guest (no count, no "left to kill").
//   - Only if a corpse is lying there when the event is due (Start gives up otherwise; the frame retries a few times).
//   - While it drags the body, Body.carried is true, so witnesses, guards, cops and the ambulance ignore it (the
//     corpse "is not there"). At the end it is removed from g.bodies exactly like dispose() does. The kill itself was
//     counted when it happened (countKill / located-kills never go down), so no counter, no star and no stat changes.
//   - Killing the dog: +5 points, no blame, no animal penalty, no stat (it is a stray). The body is dropped where
//     the dog was.
//
// WHEN: outdoor parties (not the cruise or the ice rink), 30 %, between 90 and 330 s, once (a second visit after 150 s).
// ---------------------------------------------------------------------------

const (
	dogSpeed   = 30.0
	dogDragSpd = 16.0
	dogReach   = 24.0
	dogKillPts = 5
)

var dogNoIcons = map[string]bool{"boat": true, "hockey": true, "station": true}

var dogBarks = pevTexts("¡Guau!", "¡Guau, guau!", "¡Arf, arf!", "Grrr... ¡mío!", "¡Rrrguau!", "¡Guau! ¿Qué es eso?")
var dogDeadLine = pevTexts("El perro callejero ya no ladrará más")[0]
var dogSniffs = pevTexts("*sniff, sniff*", "*olfatea*", "*sniff* ¡Guau!")

const (
	dogGo = iota
	dogSniff
	dogDrag
	dogLeave
)

type dogState struct {
	dog   *Animal
	phase int
	path  []Vec
	b     *Body
	exit  Vec
	entry Vec
	t0    float64
	chk   float64
	dead  bool
	fade  float64
	dir   Vec // unit vector of the last move (the body trails behind)
	took  bool
	// kept for the tests
	gaveUp bool
}

func init() {
	registerPartyEvent(&partyEvent{ID: "dog", Name: "Stray dog", Group: "dog", Prob: 0.30, AtMin: 90, AtMax: 330, Dur: 60,
		Cooldown: 150, MaxRuns: 2,
		Eligible: func(g *Game, t partyTraits) bool { return t.outdoor && t.roam && !dogNoIcons[t.icon] },
		Start:    dogStart, Update: pevGuard(dogUpdate, dogEnd), Items: dogItems})
}

// dogBodyOK: a corpse the dog may take (nobody carries it, nobody is on the way to it, it is a normal body).
func dogBodyOK(g *Game, b *Body) bool {
	if b == nil || b.carried || b.bagged || b.skeleton || b.fallT > 0 || b.claimed != nil || g.bodyHeld(b) { // an ambulance's body stays (ambulance.go)
		return false
	}
	for _, x := range g.bodies {
		if x == b {
			return true
		}
	}
	return false
}

func dogFindBody(g *Game, from Vec) *Body {
	var best *Body
	for _, b := range g.bodies {
		if dogBodyOK(g, b) && (best == nil || dist(from, b.pos) < dist(from, best.pos)) {
			best = b
		}
	}
	return best
}

// dogEdge: a free, reachable roaming point as near to the border of the level as it gets (the "pavement").
func dogEdge(g *Game, r *rand.Rand, near Vec, avoid *Vec, minGap float64) (Vec, bool) {
	l := g.lvl
	W, H := l.Def.W, l.Def.H
	var best Vec
	bs := math.Inf(1)
	for i := 0; i < 70; i++ {
		p := l.NearestFree(l.RandomRoam(r))
		if !l.Free(p) || l.SolidP(p.X, p.Y) || l.badGround(p) || l.InPool(p) {
			continue
		}
		if avoid != nil && dist(p, *avoid) < minGap {
			continue
		}
		edge := math.Min(math.Min(p.X, W-p.X), math.Min(p.Y, H-p.Y))
		sc := edge*2 + dist(p, near)*0.4
		if sc < bs && g.pevPlayerFar(p, 50) {
			best, bs = p, sc
		}
	}
	return best, !math.IsInf(bs, 1)
}

func dogStart(g *Game, a *partyAct) bool {
	r := partyDice()
	if dbgPartyEvent == "dog" && len(g.bodies) == 0 { // -pevent dog: a body for the capture
		if sp, ok := g.pevSpot(r, 14, true); ok {
			g.spawnBody(sp, g.p.cs, DirDown)
			g.bodies[len(g.bodies)-1].fallT = 0
			g.bodies[len(g.bodies)-1].byPlayer = false
		}
	}
	b := dogFindBody(g, g.p.pos)
	if b == nil {
		return false
	}
	exit, ok := dogEdge(g, r, b.pos, nil, 0)
	if !ok {
		return false
	}
	entry, ok := dogEdge(g, r, b.pos, &exit, 140)
	if !ok {
		return false
	}
	target := g.lvl.NearestFree(Vec{b.pos.X + 12, b.pos.Y + 4})
	path := g.lvl.PathG(entry, target)
	if path == nil || g.lvl.PathG(b.pos, exit) == nil {
		return false
	}
	d := &Animal{kind: animDog, pos: entry, face: 1, coat: r.Intn(3), seed: r.Float64() * 10}
	a.D = &dogState{dog: d, phase: dogGo, path: path, b: b, exit: exit, entry: entry, t0: g.time}
	d.icon, d.iconT = pevPick(r, dogBarks), 2
	return true
}

func dogSetFace(d *Animal, from, to Vec) {
	if math.Abs(to.X-from.X) > 0.2 {
		d.face = 1
		if to.X < from.X {
			d.face = -1
		}
	}
}

func dogFree(g *Game, s *dogState) {
	if s.b != nil && s.b.carried && !s.b.bagged {
		// still ours (nobody else carries it: carried is only set by us in this event)
		if s.took {
			s.b.carried = false
			s.b.pos = g.lvl.NearestFree(s.b.pos)
		}
	}
	s.took = false
}

func dogEnd(g *Game, a *partyAct) {
	if s, ok := a.D.(*dogState); ok {
		dogFree(g, s)
	}
}

func dogUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*dogState)
	r := partyDice()
	d := s.dog
	d.anim += dt
	if d.iconT > 0 {
		if d.iconT -= dt; d.iconT <= 0 {
			d.icon = ""
		}
	}
	if s.dead {
		s.fade -= dt
		return s.fade <= 0
	}
	d.moving = false
	switch s.phase {
	case dogGo:
		if !dogBodyOK(g, s.b) {
			if nb := dogFindBody(g, d.pos); nb != nil {
				s.b = nb
				s.path = g.lvl.PathG(d.pos, g.lvl.NearestFree(Vec{nb.pos.X + 12, nb.pos.Y + 4}))
			} else {
				s.gave(g)
				return false
			}
		}
		var dir Dir
		mv, arr := follow(&d.pos, &dir, &s.path, dogSpeed, dt)
		d.moving = mv
		dogSetFace(d, d.pos, s.b.pos)
		if arr || dist(d.pos, s.b.pos) < dogReach*0.7 {
			s.phase, s.t0 = dogSniff, g.time
			d.icon, d.iconT = pevPick(r, dogSniffs), 2
		}
		if g.time-s.t0 > 40 {
			s.gave(g)
		}
	case dogSniff:
		if !dogBodyOK(g, s.b) {
			s.gave(g)
			return false
		}
		dogSetFace(d, d.pos, s.b.pos)
		d.moving = true // paces on the spot with its nose down
		if g.time-s.t0 > 2.4 {
			path := g.lvl.PathG(s.b.pos, s.exit)
			if path == nil {
				s.gave(g)
				return false
			}
			s.b.carried = true // for the witnesses it is not lying there any more
			s.dir = faceVec(d.pos, path[0])
			s.b.pos = Vec{d.pos.X - s.dir.X*11, d.pos.Y - s.dir.Y*11}
			s.took, s.path, s.phase, s.t0 = true, path, dogDrag, g.time
			d.icon, d.iconT = pevPick(r, dogBarks), 1.6
			g.snd.Play("thud")
		}
	case dogDrag:
		if !s.b.carried || !dogBodyOK2(g, s.b) { // someone took it from us (or Rex ate it): we give up
			s.took = false
			s.gave(g)
			return false
		}
		old := d.pos
		var dir Dir
		mv, arr := follow(&d.pos, &dir, &s.path, dogDragSpd, dt)
		d.moving = mv
		if mv {
			dogSetFace(d, old, d.pos)
			if l := dist(old, d.pos); l > 0 {
				s.dir = Vec{(d.pos.X - old.X) / l, (d.pos.Y - old.Y) / l}
			}
		}
		// the body trails behind the dog
		s.b.pos = Vec{d.pos.X - s.dir.X*11, d.pos.Y - s.dir.Y*11}
		s.b.dir = faceTo(d.pos, s.b.pos)
		if arr {
			dogTakeAway(g, s)
			s.phase, s.t0 = dogLeave, g.time
			return true
		}
		if g.time-s.t0 > 70 {
			s.took = false
			dogFree(g, s)
			s.gave(g)
		}
	case dogLeave:
		var dir Dir
		mv, arr := follow(&d.pos, &dir, &s.path, dogSpeed, dt)
		d.moving = mv
		dogSetFace(d, d.pos, s.entry)
		return arr || g.time-s.t0 > 40
	}
	return false
}

// dogOK2: the body is still among g.bodies (the lenient check while we carry it).
func dogBodyOK2(g *Game, b *Body) bool {
	for _, x := range g.bodies {
		if x == b {
			return true
		}
	}
	return false
}

// gave: no body to take (any more): the dog trots off the way it came.
func (s *dogState) gave(g *Game) {
	s.phase, s.t0 = dogLeave, g.time
	s.path = g.lvl.PathG(s.dog.pos, s.entry)
	if s.path == nil {
		s.path = []Vec{s.dog.pos}
	}
	s.gaveUp = true
}

// dogTakeAway: the dog and the body vanish at the edge (a puff of dust); the body leaves g.bodies like dispose().
func dogTakeAway(g *Game, s *dogState) {
	for i, x := range g.bodies {
		if x == s.b {
			g.bodies = append(g.bodies[:i], g.bodies[i+1:]...)
			break
		}
	}
	s.took = false
	r := partyDice()
	pevPuff(g, s.dog.pos, 14, r)
	s.dog.icon, s.dog.iconT = "", 0
}

// pevPuff: a puff of dust with the party dice (g.rng is never moved).
func pevPuff(g *Game, pos Vec, n int, r *rand.Rand) {
	for i := 0; i < n; i++ {
		a := r.Float64() * 2 * math.Pi
		sp := 30 * (0.3 + r.Float64()*0.7)
		l := 0.5 * (0.6 + r.Float64()*0.4)
		g.parts = append(g.parts, Particle{pos: Vec{pos.X, pos.Y - 6}, vel: Vec{math.Cos(a) * sp, math.Sin(a)*sp - 8}, life: l, max: l,
			col: color.RGBA{0xc8, 0xc0, 0xa8, 255}, size: 2, grav: 40})
	}
}

// dogStab: the player's blow reaches the stray dog (the nearest, within maxD). Small points, no blame, no penalty.
func (g *Game) dogStab(p *Player, maxD float64) bool {
	if !g.partyActive() {
		return false
	}
	a := g.partyRunning("dog")
	if a == nil {
		return false
	}
	s, ok := a.D.(*dogState)
	if !ok || s.dead || dist(p.pos, s.dog.pos) > maxD {
		return false
	}
	s.dead, s.fade = true, 1.2
	s.dog.dead, s.dog.moving = true, false
	if s.took {
		dogFree(g, s)
	}
	g.award(s.dog.pos, dogKillPts, "")
	g.decals = append(g.decals, Decal{pos: s.dog.pos, kind: 0, r: 4})
	pevPuff(g, s.dog.pos, 8, partyDice())
	g.snd.Play("stab")
	g.say(dogDeadLine)
	s.dog.icon = ""
	return true
}

func dogItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*dogState)
	if s.took && s.b != nil && s.b.carried {
		b := s.b
		items = append(items, drawItem{b.pos.Y - 4, func() { g.drawBody(dst, b) }})
	}
	d := s.dog
	items = append(items, drawItem{d.pos.Y, func() {
		if g.as == nil || g.as.animals == nil {
			return
		}
		g.drawAnimal(dst, d)
	}})
	return items
}
