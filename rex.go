package main

import (
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The island's inhabitants.
//
//   - ONE Tyrannosaurus rex is loose: it wanders, chases the guests (and you, if
//     it sees you), eats them and throws bits of them around. It roars every
//     so often. It is faster than anybody on foot and it never goes into the
//     lake. What it kills counts like any other death that is not yours (no
//     stars, no points) so the party can end with it; if it catches the player
//     the game is over ("Te ha comido el T-Rex") unless the player is immune.
//   - The brachiosaurs, the triceratops and the velociraptors only walk about
//     their pens for show: they are not props (nothing blocks) and they do
//     not touch anybody.
// ---------------------------------------------------------------------------

const (
	jrSleep = iota // not out of the tunnel yet
	jrWander
	jrChase
	jrEat
	jrRoar
)

type JRex struct {
	pos      Vec
	face     float64 // -1 looks left, +1 right
	state    int
	timer    float64
	path     []Vec
	cur      rexPrey // what it is after (the nearest person it sees: nobody has priority)
	pickT    float64
	lostT    float64
	roarIn   float64
	roarT    float64
	phase    float64
	moving   bool
	repath   float64
	scaredAt map[*Guest]float64
	eatCS    *CharSet
	eatBody  *Body           // the victim lies here while the rex tears at it
	skip     map[any]float64 // people it cannot reach: left alone until that time
	biteT    float64         // timer of the warning effects
	noPath   float64         // how long the chase has had no way to the prey
	eatE     float64         // seconds into the meal
	biteOn   bool
}

// jpDino is a dinosaur that only walks about its pen: drawn, nothing else.
type jpDino struct {
	name        string
	v           int // first frame (the next one is v+1 when the pose changes)
	area        Rect
	pos, to     Vec
	face        float64
	speed       float64
	pause       float64
	phase       float64
	moving      bool
	flipEvery   float64
	still       bool // stays where it is and only moves its head
	frameT      float64
	frame       int
	shiftFrames int
}

// jpEgg is a nest of eggs that rock, hatch and let a tiny baby wander about: only drawn.
type jpEgg struct {
	pos       Vec
	t         float64 // time in the current state
	state     int     // 0 still, 1 rocking, 2 cracking, 3 hatched, 4 baby about
	wait      float64
	baby      Vec // offset of the baby from the nest
	babyTo    Vec
	babyPause float64
	babyFace  float64
	phase     float64
}

// Gib is a piece of a victim that flies off the rex's mouth.
type Gib struct {
	pos     Vec // on the ground
	z       float64
	vel     Vec
	vz      float64
	kind    int // 0 arm, 1 leg, 2 chunk, 3 bone
	col     color.RGBA
	rot     float64
	vrot    float64
	life    float64
	landed  bool
	w, h    float64
	flesh   bool
	bounces int
}

func (g *Game) isIsland() bool { return g.levelIdx == specialJurassic }

func (g *Game) spawnJurassic() {
	g.rex, g.jpDinos, g.gibs, g.jpEggs = nil, nil, nil, nil
	if !g.isIsland() {
		return
	}
	if _, ok := g.snd.sfx["rex_roar"]; !ok {
		g.snd.sfx["rex_roar"] = jrRoarSound()
		g.snd.sfx["rex_step"] = mix(tone(46, 0.22, 0.9, 12, 0), lowpass(noise(0.12, 0.5, 22), 6))
		g.snd.sfx["rex_bite"] = mix(lowpass(noise(0.3, 0.8, 9), 4), tone(70, 0.25, 0.7, 10, 2))
	}
	first := 20 + g.rng.Float64()*5 // out of the tunnel 20 to 25 seconds into the party
	if fastEv {
		first = 4
	}
	g.rex = &JRex{pos: Vec{790, 256}, face: -1, state: jrSleep, timer: first, roarIn: 9 + g.rng.Float64()*6, scaredAt: map[*Guest]float64{}, skip: map[any]float64{}}
	rng := g.rng
	mk := func(name string, v int, area Rect, sp float64) *jpDino {
		d := &jpDino{name: name, v: v, area: area, speed: sp, face: -1, pause: rng.Float64() * 3, flipEvery: 0.7 + rng.Float64()*0.6}
		d.pos = Vec{area.X + rng.Float64()*area.W, area.Y + rng.Float64()*area.H}
		d.to = d.pos
		return d
	}
	g.jpDinos = append(g.jpDinos,
		mk("jp_brachio", 0, Rect{240, 168, 90, 22}, 5),
		mk("jp_raptor", 0, Rect{616, 232, 96, 52}, 22), mk("jp_raptor", 2, Rect{616, 232, 96, 52}, 24), mk("jp_raptor", 4, Rect{616, 232, 96, 52}, 20),
	)
	g.jpDinos = append(g.jpDinos, mk("jp_hadro", 0, Rect{722, 398, 40, 12}, 6))
	for _, p := range []Vec{{650, 226}, {220, 214}, {168, 356}, {196, 270}, {736, 322}} {
		g.jpEggs = append(g.jpEggs, &jpEgg{pos: p, wait: 1 + g.rng.Float64()*6, phase: g.rng.Float64() * 6, babyFace: 1})
	}
	tri := mk("jp_trike", 0, Rect{200, 300, 1, 1}, 0)
	tri.pos, tri.still = Vec{200, 300}, true
	g.jpDinos = append(g.jpDinos, tri)
}

// ---- update ----------------------------------------------------------------------------

func (g *Game) updateJurassic(dt float64) {
	if !g.isIsland() || g.mode != ModePlay {
		return
	}
	for _, d := range g.jpDinos {
		g.updateJPDino(d, dt)
	}
	for _, e := range g.jpEggs {
		g.updateJPEgg(e, dt)
	}
	g.updateGibs(dt)
	if g.rex != nil {
		g.updateRex(dt)
	}
}

func (g *Game) updateJPEgg(e *jpEgg, dt float64) {
	e.t += dt
	e.phase += dt
	switch e.state {
	case 0: // quiet, then it starts to rock
		if e.t > e.wait {
			e.state, e.t = 1, 0
		}
	case 1:
		if e.t > 1.3 {
			e.state, e.t = 2, 0
		}
	case 2:
		if e.t > 1.6 {
			e.state, e.t = 3, 0
		}
	case 3:
		if e.t > 1.2 {
			e.state, e.t = 4, 0
			e.baby, e.babyTo = Vec{}, Vec{}
		}
	case 4: // the baby walks about the nest
		e.babyPause -= dt
		if e.babyPause <= 0 {
			e.babyTo = Vec{(g.rng.Float64() - 0.5) * 40, (g.rng.Float64() - 0.2) * 16}
			e.babyPause = 0.8 + g.rng.Float64()*1.2
		}
		dx, dy := e.babyTo.X-e.baby.X, e.babyTo.Y-e.baby.Y
		if l := math.Hypot(dx, dy); l > 0.5 {
			st := math.Min(l, 14*dt)
			e.baby.X += dx / l * st
			e.baby.Y += dy / l * st
			if dx > 0.2 {
				e.babyFace = 1
			} else if dx < -0.2 {
				e.babyFace = -1
			}
		}
		if e.t > 7 { // the eggs are back: a new clutch
			e.state, e.t, e.wait = 0, 0, 3+g.rng.Float64()*6
		}
	}
}

func (g *Game) updateJPDino(d *jpDino, dt float64) {
	d.frameT += dt
	if d.frameT > d.flipEvery {
		d.frameT = 0
		d.frame = 1 - d.frame
	}
	if d.still {
		d.moving = false
		return
	}
	d.moving = false
	if d.pause > 0 {
		d.pause -= dt
		if d.pause <= 0 {
			d.to = Vec{d.area.X + g.rng.Float64()*d.area.W, d.area.Y + g.rng.Float64()*d.area.H}
		}
		return
	}
	dx, dy := d.to.X-d.pos.X, d.to.Y-d.pos.Y
	l := math.Hypot(dx, dy)
	if l < 2 {
		d.pause = 1.5 + g.rng.Float64()*4
		return
	}
	step := math.Min(l, d.speed*dt)
	d.pos.X += dx / l * step
	d.pos.Y += dy / l * step
	d.phase += dt * 6
	d.moving = true
	if math.Abs(dx) > 1 {
		if dx > 0 {
			d.face = 1
		} else {
			d.face = -1
		}
	}
}

// rexPrey is anybody the rex can eat: a guest, a guard, an officer or the rest of the units (and the player).
type rexPrey struct {
	id     any
	at     func() Vec
	cs     *CharSet
	player bool
	alive  func() bool
	kill   func() // kills it for nothing: no stars, no points
}

func (g *Game) rexCandidates() []rexPrey {
	var out []rexPrey
	for _, gu := range g.guests {
		gu := gu
		if gu.out() || gu.state == gSwim || g.lvl.InPool(gu.pos) {
			continue
		}
		out = append(out, rexPrey{id: gu, at: func() Vec { return gu.pos }, cs: gu.cs, alive: func() bool { return !gu.out() },
			kill: func() { g.killGuest(gu, false) }})
	}
	for _, gd := range g.guards {
		gd := gd
		if gd.dead {
			continue
		}
		out = append(out, rexPrey{id: gd, at: func() Vec { return gd.pos }, cs: gd.cs, alive: func() bool { return !gd.dead },
			kill: func() {
				gd.dead, gd.icon = true, ""
				g.spawnBody(gd.pos, gd.cs, gd.dir)
				g.bodies[len(g.bodies)-1].label = "Guardia"
				g.checkDone()
			}})
	}
	for _, c := range g.cops {
		c := c
		if c.state == cLeave && c.veh == nil {
			continue
		}
		cs := g.copCSFor(c)
		out = append(out, rexPrey{id: c, at: func() Vec { return c.pos }, cs: cs, alive: func() bool { return g.copInList(c) },
			kill: func() { g.rexKillCop(c, cs) }})
	}
	for _, f := range g.fireCrew {
		f := f
		out = append(out, rexPrey{id: f, at: func() Vec { return f.pos }, cs: f.cs, alive: func() bool { return f.state != 3 },
			kill: func() { f.state = 3; g.spawnBody(f.pos, f.cs, f.dir); g.checkDone() }})
	}
	for _, t := range g.techCrew {
		t := t
		out = append(out, rexPrey{id: t, at: func() Vec { return t.pos }, cs: t.cs, alive: func() bool { return t.state != 3 },
			kill: func() { t.state = 3; g.spawnBody(t.pos, t.cs, t.dir); g.checkDone() }})
	}
	if p := g.p; p != nil && !p.dead && !p.hidden && !g.lvl.InPool(p.pos) {
		out = append(out, rexPrey{id: p, at: func() Vec { return p.pos }, cs: p.cs, player: true, alive: func() bool { return !p.dead }})
	}
	return out
}

func (g *Game) copInList(c *Cop) bool {
	for _, o := range g.cops {
		if o == c {
			return true
		}
	}
	return false
}

// rexKillCop takes an officer out of the party the way leaving does: no dangling references.
func (g *Game) rexKillCop(c *Cop, cs *CharSet) {
	g.releaseCop(c)
	for i, x := range g.cops {
		if x == c {
			g.cops = append(g.cops[:i], g.cops[i+1:]...)
			break
		}
	}
	if c.veh != nil { // the vehicle leaves when nobody of its crew is left outside
		crew := false
		for _, o := range g.cops {
			if o.veh == c.veh {
				crew = true
			}
		}
		if !crew {
			c.veh.state = 2
		}
	}
	if c.medic {
		left := false
		for _, o := range g.cops {
			if o.medic {
				left = true
			}
		}
		if !left {
			g.ambCalled, g.ambCount = false, 0
		}
	}
	g.spawnBody(c.pos, cs, c.dir)
	g.checkDone()
}

// rexPick: the nearest person the rex SEES in front of it (the same criterion as the killers: a cone,
// a clear line of sight and no priority for the player). ok is false when it sees nobody.
func (g *Game) rexPick() (rexPrey, bool) {
	r := g.rex
	dir := DirLeft
	if r.face > 0 {
		dir = DirRight
	}
	eye := Vec{r.pos.X + r.face*20, r.pos.Y - 6}
	var best rexPrey
	bd, ok := 1e9, false
	for _, c := range g.rexCandidates() {
		if r.skip[c.id] > g.time {
			continue
		}
		pos := c.at()
		if d := dist(eye, pos); d < bd && g.sees(eye, dir, pos, 260, 0.3) {
			best, bd, ok = c, d, true
		}
	}
	return best, ok
}

func (g *Game) updateRex(dt float64) {
	r := g.rex
	p := g.p
	r.phase += dt
	if r.state == jrSleep {
		r.timer -= dt
		if r.timer < 2.6 && r.timer > 0 { // the ground trembles, dust falls from the cave, the birds leave
			if g.shake < 0.05 {
				g.shake = 0.05
			}
			r.biteT -= dt
			if r.biteT <= 0 {
				r.biteT = 0.3
				g.burst(Vec{r.pos.X - 20 + g.rng.Float64()*40, r.pos.Y - 14}, 6, 40, rgba(0xb8a888, 200), 0.9, 2, 30)
				if g.rng.Float64() < 0.5 { // a bird: a dark speck that flies off
					g.burst(Vec{r.pos.X + (g.rng.Float64()-0.5)*90, r.pos.Y - 50}, 2, 150, rgb(0x20202a), 1.3, 2, -40)
				}
			}
		}
		if r.timer <= 0 {
			g.burst(Vec{r.pos.X, r.pos.Y - 10}, 40, 110, rgba(0xb8a888, 210), 1.2, 3, 40)
			r.state, r.roarT = jrRoar, 1.7
			g.snd.Play("rex_roar")
			g.shake = 0.5
			g.say("¡Un T-Rex suelto! Dice el cartel que no muerde. El cartel miente.")
			g.scareAround(r.pos, 380)
		}
		return
	}
	r.roarIn -= dt
	r.moving = false
	// the player: caught if too near
	if !p.dead && !g.invulnerable() && !p.hidden && !g.lvl.InPool(p.pos) && dist(Vec{r.pos.X, r.pos.Y - 6}, p.pos) < 30 {
		g.lose("Te ha comido el T-Rex")
		g.snd.Play("rex_bite")
		g.rexSplat(p.pos, nil)
		return
	}
	switch r.state {
	case jrRoar:
		r.roarT -= dt
		if r.roarT <= 0 {
			r.state, r.timer = jrWander, 0
			r.roarIn = 14 + g.rng.Float64()*12
		}
		return
	case jrEat:
		g.updateRexMeal(r, dt)
		return
	}
	if r.roarIn <= 0 && r.state != jrRoar {
		r.state, r.roarT = jrRoar, 1.5
		g.snd.Play("rex_roar")
		g.shake = 0.4
		g.scareAround(r.pos, 300)
		return
	}
	// what to chase: the nearest person it sees; if it sees nobody it keeps going for a moment and then wanders
	r.repath -= dt
	r.pickT -= dt
	if r.pickT <= 0 {
		r.pickT = 0.4
		if pr, ok := g.rexPick(); ok {
			if r.cur.id != pr.id {
				r.repath = 0
			}
			r.cur, r.lostT, r.state = pr, 0, jrChase
		} else if r.state == jrChase {
			if r.lostT += 0.4; r.lostT > 2 {
				r.cur, r.state, r.path = rexPrey{}, jrWander, nil
			}
		}
	}
	if r.cur.at != nil && !r.cur.alive() {
		r.cur, r.state, r.path = rexPrey{}, jrWander, nil
	}
	speed := 26.0
	var goal Vec
	if r.state == jrChase && r.cur.at != nil {
		tp := r.cur.at()
		sgn := 1.0
		if tp.X < r.pos.X {
			sgn = -1
		}
		speed, goal = 50, Vec{tp.X - sgn*rexK*40, tp.Y + rexK*16}
		if r.cur.player {
			speed = 44
		}
		if r.repath <= 0 {
			r.path = g.rexPath(r.pos, goal)
			r.repath = 0.6
			if r.path == nil { // no way to it (the lake, a fence): give up after a moment
				r.noPath += 0.6
				if r.noPath > 2 {
					r.skip[r.cur.id] = g.time + 12
					r.noPath, r.cur, r.state = 0, rexPrey{}, jrWander
				}
			} else {
				r.noPath = 0
			}
		}
	} else if len(r.path) == 0 || r.repath <= 0 {
		t := g.lvl.RandomRoam(g.rng)
		r.path = g.rexPath(r.pos, t)
		r.repath = 8 + g.rng.Float64()*6
	}
	// eat what is in reach (the player is only caught by touching it, above)
	if r.state == jrChase && r.cur.at != nil && !r.cur.player && dist(r.biteSpot(), r.cur.at()) < 22 {
		g.jrEats(r.cur)
		return
	}
	// make the guests nearby hesitate, a little
	for _, gu := range g.guests {
		if gu.out() || gu.state == gScared || gu.state == gSwim {
			continue
		}
		if dist(r.pos, gu.pos) < 150 && g.time-r.scaredAt[gu] > 7 {
			r.scaredAt[gu] = g.time
			g.scare(gu)
			gu.icon = ""                         // silent: the roar is what makes noise
			gu.timer = 1.8 + g.rng.Float64()*1.4 // they run a little and stop
		}
	}
	if len(r.path) > 0 {
		t := r.path[0]
		dx := t.X - r.pos.X
		var dir Dir
		m, _ := follow(&r.pos, &dir, &r.path, speed, dt)
		r.moving = m
		if math.Abs(dx) > 0.5 {
			if dx > 0 {
				r.face = 1
			} else {
				r.face = -1
			}
		}
		if m && int(r.phase*3.2)%2 == 0 && math.Mod(r.phase*3.2, 1) < dt*3.2 {
			g.snd.Play("rex_step")
			if g.shake < 0.06 {
				g.shake = 0.06
			}
		}
	}
}

// scareAround: everybody near hesitates (a short, small run).
func (g *Game) scareAround(at Vec, rad float64) {
	var near []*Guest
	for _, gu := range g.guests {
		if gu.out() || gu.state == gSwim || dist(at, gu.pos) > rad {
			continue
		}
		g.scare(gu)
		gu.timer = 2 + g.rng.Float64()*2
		gu.icon = "" // everybody runs, but only the closest few shout (see bubbles.go)
		near = append(near, gu)
	}
	sort.Slice(near, func(i, j int) bool { return dist(at, near[i].pos) < dist(at, near[j].pos) })
	emotes := []string{"AAAH!", "¡¡!!", "!?", "¡CORRE!"}
	for i, gu := range near {
		if i >= 3 {
			break
		}
		gu.icon, gu.iconT = emotes[g.rng.Intn(len(emotes))], 1.4
	}
}

// the mouth of the rex when it bends down: a bit in front of it and "up" in the picture
func (r *JRex) biteSpot() Vec { return Vec{r.pos.X + r.face*rexK*40, r.pos.Y - rexK*16} }

// rexK: the rex is drawn a bit bigger than its sprite, so it looks enormous next to the people
const rexK = 1.18

const rexMeal = 3.7

func (g *Game) jrEats(pr rexPrey) {
	r := g.rex
	if !pr.alive() {
		return
	}
	r.state, r.eatCS, r.eatE, r.biteOn = jrEat, pr.cs, 0, false
	r.path, r.cur = nil, rexPrey{}
	spot := r.biteSpot()
	at := pr.at()
	nb := len(g.bodies)
	pr.kill()
	// the body stays on the ground, under its mouth, until it is swallowed
	r.eatBody = nil
	if len(g.bodies) > nb {
		r.eatBody = g.bodies[len(g.bodies)-1]
		if dist(r.eatBody.pos, at) < 2 {
			r.eatBody.pos = spot
		}
	}
	g.snd.Play("rex_bite")
	g.burst(Vec{spot.X, spot.Y - 6}, 14, 90, rgb(0xb01818), 0.7, 2, 240)
}

// updateRexMeal: lower the neck, bite and tear (bits fly), raise the head and swallow, then maybe roar.
func (g *Game) updateRexMeal(r *JRex, dt float64) {
	r.eatE += dt
	e := r.eatE
	spot := r.biteSpot()
	if r.eatBody != nil && e < 2.4 {
		r.eatBody.pos = spot
	}
	switch {
	case e >= 0.55 && e < 2.4:
		on := int((e-0.55)/0.3)%2 == 1 // jaws closed on odd beats
		if on && !r.biteOn {
			g.snd.Play("rex_bite")
			g.rexSplat(spot, r.eatCS)
			if g.shake < 0.1 {
				g.shake = 0.1
			}
		}
		r.biteOn = on
	case e >= 2.4 && r.eatBody != nil:
		// head up, the rest goes down the throat
		for i, b := range g.bodies {
			if b == r.eatBody {
				g.bodies = append(g.bodies[:i], g.bodies[i+1:]...)
				break
			}
		}
		r.eatBody = nil
		g.burst(Vec{spot.X, spot.Y - 20}, 16, 110, rgb(0xb01818), 0.8, 2, 240)
		g.rexSplat(spot, r.eatCS)
	}
	if e >= rexMeal {
		r.eatCS, r.path = nil, nil
		r.state, r.timer = jrWander, 0
		if g.rng.Float64() < 0.55 { // satisfied, it roars with its head up
			r.state, r.roarT = jrRoar, 1.5
			r.roarIn = 14 + g.rng.Float64()*12
			g.snd.Play("rex_roar")
			g.shake = 0.35
			g.scareAround(r.pos, 260)
		}
	}
}

// rexSplat: blood, and arms, legs and bones that fly out of the mouth.
func (g *Game) rexSplat(at Vec, cs *CharSet) {
	g.burst(Vec{at.X, at.Y - 18}, 22, 120, rgb(0xb01818), 0.8, 2, 240)
	g.burst(Vec{at.X, at.Y - 22}, 8, 90, rgb(0xe03030), 0.6, 1, 240)
	g.decals = append(g.decals, Decal{pos: Vec{at.X + (g.rng.Float64()-0.5)*20, at.Y + (g.rng.Float64()-0.5)*8}, kind: 0, r: 7 + g.rng.Float64()*6})
	skin, cloth, trou := rgb(0xe0b088), rgb(0x3a6ae0), rgb(0x2a3a6a)
	if cs != nil && cs.Spec != nil {
		sp := cs.Spec
		skin, cloth, trou = sp.Skin, sp.Shirt, sp.Pants
	}
	n := 2 + g.rng.Intn(3)
	for i := 0; i < n; i++ {
		kind := g.rng.Intn(4)
		gb := Gib{pos: at, z: 14 + g.rng.Float64()*8, kind: kind, life: 14 + g.rng.Float64()*8,
			vel: Vec{(g.rng.Float64() - 0.5) * 150, (g.rng.Float64() - 0.5) * 70}, vz: 70 + g.rng.Float64()*90,
			rot: g.rng.Float64() * 6, vrot: (g.rng.Float64() - 0.5) * 18}
		switch kind {
		case 0:
			gb.col, gb.w, gb.h, gb.flesh = skin, 5, 1.6, true
		case 1:
			gb.col, gb.w, gb.h, gb.flesh = trou, 6, 2, true
		case 2:
			gb.col, gb.w, gb.h, gb.flesh = rgb(0xa01818), 3, 3, true
			if i%2 == 0 {
				gb.col = cloth
			}
		default:
			gb.col, gb.w, gb.h, gb.flesh = rgb(0xece4cc), 5, 1.2, false
		}
		g.gibs = append(g.gibs, gb)
	}
	if len(g.gibs) > 70 {
		g.gibs = g.gibs[len(g.gibs)-70:]
	}
}

func (g *Game) updateGibs(dt float64) {
	out := g.gibs[:0]
	for _, b := range g.gibs {
		if !b.landed {
			b.vz -= 340 * dt
			b.z += b.vz * dt
			b.pos.X += b.vel.X * dt
			b.pos.Y += b.vel.Y * dt
			b.rot += b.vrot * dt
			if b.z <= 0 {
				b.z = 0
				if b.bounces < 1 {
					b.bounces++
					b.vz = -b.vz * 0.3
					b.vel = Vec{b.vel.X * 0.4, b.vel.Y * 0.4}
				} else {
					b.landed = true
					g.decals = append(g.decals, Decal{pos: b.pos, kind: 0, r: 2 + g.rng.Float64()*2})
				}
			}
		} else {
			b.life -= dt
		}
		if b.life > 0 {
			out = append(out, b)
		}
	}
	g.gibs = out
}

// rexPath is a walking path for the rex: the lake is closed to it.
func (g *Game) rexPath(from, to Vec) []Vec {
	l := g.lvl
	ok := func(cx, cy int) bool {
		if !l.cellFree(cx, cy) {
			return false
		}
		return !l.InPool(cellCenter(cx, cy))
	}
	near := func(p Vec) (int, int) {
		cx, cy := l.cellOf(p)
		for r := 0; r < 12; r++ {
			for dy := -r; dy <= r; dy++ {
				for dx := -r; dx <= r; dx++ {
					if (abs(dx) == r || abs(dy) == r) && ok(cx+dx, cy+dy) {
						return cx + dx, cy + dy
					}
				}
			}
		}
		return cx, cy
	}
	sx, sy := near(from)
	ex, ey := near(to)
	if sx == ex && sy == ey {
		return []Vec{to}
	}
	prev := make([]int32, l.cw*l.ch)
	for i := range prev {
		prev[i] = -2
	}
	start, goal := sy*l.cw+sx, ey*l.cw+ex
	prev[start] = -1
	queue := []int{start}
	dirs := [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}}
	found := false
	for h := 0; h < len(queue) && !found; h++ {
		cur := queue[h]
		cx, cy := cur%l.cw, cur/l.cw
		for _, d := range dirs {
			nx, ny := cx+d[0], cy+d[1]
			if !ok(nx, ny) {
				continue
			}
			if d[0] != 0 && d[1] != 0 && (!ok(cx+d[0], cy) || !ok(cx, cy+d[1])) {
				continue
			}
			ni := ny*l.cw + nx
			if prev[ni] != -2 {
				continue
			}
			prev[ni] = int32(cur)
			if ni == goal {
				found = true
				break
			}
			queue = append(queue, ni)
		}
	}
	if !found {
		return nil
	}
	var cells []Vec
	for i := goal; i != -1; i = int(prev[i]) {
		cells = append(cells, cellCenter(i%l.cw, i/l.cw))
	}
	for i, j := 0, len(cells)-1; i < j; i, j = i+1, j-1 {
		cells[i], cells[j] = cells[j], cells[i]
	}
	// keep one point in three: a straighter, calmer walk
	var out []Vec
	for i, c := range cells {
		if i%3 == 2 || i == len(cells)-1 {
			out = append(out, c)
		}
	}
	return out
}

// ---- drawing -------------------------------------------------------------------------

func (g *Game) jurassicItems(screen *ebiten.Image, items []drawItem) []drawItem {
	if !g.isIsland() {
		return items
	}
	for _, d := range g.jpDinos {
		d := d
		items = append(items, drawItem{d.pos.Y, func() { g.drawJPDino(screen, d) }})
	}
	for _, e := range g.jpEggs {
		e := e
		items = append(items, drawItem{e.pos.Y, func() { g.drawJPEgg(screen, e) }})
	}
	for i := range g.gibs {
		b := &g.gibs[i]
		items = append(items, drawItem{b.pos.Y + 0.1, func() { g.drawGib(screen, b) }})
	}
	if r := g.rex; r != nil && r.state != jrSleep {
		items = append(items, drawItem{r.pos.Y, func() { g.drawJRex(screen, r) }})
	}
	return items
}

func (g *Game) drawJPDino(dst *ebiten.Image, d *jpDino) {
	imgs := g.as.props[d.name]
	idx := d.v + d.frame
	if idx >= len(imgs) {
		idx = d.v
	}
	if idx >= len(imgs) {
		return
	}
	pi := imgs[idx]
	w := float64(pi.img.Bounds().Dx())
	x, y := g.wx(d.pos.X), g.wy(d.pos.Y)
	bob := 0.0
	if d.moving {
		bob = -math.Abs(math.Sin(d.phase)) * 1.4
	}
	op := &ebiten.DrawImageOptions{}
	if d.face > 0 { // the sprites look left
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(w, 0)
		op.GeoM.Translate(math.Round(x-(w-pi.ox)), math.Round(y-pi.oy+bob))
	} else {
		op.GeoM.Translate(math.Round(x-pi.ox), math.Round(y-pi.oy+bob))
	}
	dst.DrawImage(pi.img, op)
}

func (g *Game) drawJPEgg(dst *ebiten.Image, e *jpEgg) {
	imgs := g.as.props["jp_egg"]
	if len(imgs) < 5 {
		return
	}
	v := 0
	switch e.state {
	case 1:
		v = 1 + int(e.t*7)%2
	case 2:
		v = 3
		if int(e.t*9)%4 == 0 { // it still rocks a little while cracking
			v = 1
		}
	case 3:
		v = 4
	case 4:
		v = 0
	}
	pi := imgs[v]
	x, y := g.wx(e.pos.X), g.wy(e.pos.Y)
	dst.DrawImage(pi.img, &ebiten.DrawImageOptions{GeoM: func() ebiten.GeoM {
		var m ebiten.GeoM
		m.Translate(math.Round(x-pi.ox), math.Round(y-pi.oy))
		return m
	}()})
	if e.state == 4 { // the baby: a few pixels of dinosaur
		bx, by := x+e.baby.X+8, y+e.baby.Y-6
		fillRect(dst, bx-3, by+3, 7, 1, color.RGBA{0, 0, 0, 60})
		body, dark := rgb(0x6aa84a), rgb(0x3a7a30)
		hop := math.Abs(math.Sin(e.phase*9)) * 1.2
		fillRect(dst, bx-2, by-hop, 5, 3, body)
		fillRect(dst, bx-2, by-hop, 5, 1, rgb(0x8acc60))
		hx := bx + 3*e.babyFace
		fillRect(dst, math.Min(hx, hx+e.babyFace)-1, by-2-hop, 3, 3, body)
		fillRect(dst, hx+e.babyFace*0.5, by-1.5-hop, 1, 1, rgb(0x14101a))
		fillRect(dst, bx-e.babyFace*3, by+1-hop, 2, 1, dark) // tail
		lf := math.Sin(e.phase * 14)
		fillRect(dst, bx-1+lf, by+2, 1, 1, dark)
		fillRect(dst, bx+1-lf, by+2, 1, 1, dark)
	}
}

func (g *Game) drawGib(dst *ebiten.Image, b *Gib) {
	x, y := g.wx(b.pos.X), g.wy(b.pos.Y)-b.z
	a := 1.0
	if b.landed && b.life < 3 {
		a = b.life / 3
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(b.w, b.h)
	op.GeoM.Translate(-b.w/2, -b.h/2)
	op.GeoM.Rotate(b.rot)
	op.GeoM.Translate(math.Round(x), math.Round(y))
	op.ColorScale.ScaleWithColor(b.col)
	op.ColorScale.ScaleAlpha(float32(a))
	dst.DrawImage(px, op)
	if b.flesh && b.kind != 2 { // a red stump at one end
		op2 := &ebiten.DrawImageOptions{}
		op2.GeoM.Scale(1.6, b.h)
		op2.GeoM.Translate(-b.w/2-0.8, -b.h/2)
		op2.GeoM.Rotate(b.rot)
		op2.GeoM.Translate(math.Round(x), math.Round(y))
		op2.ColorScale.ScaleWithColor(rgb(0xb01818))
		op2.ColorScale.ScaleAlpha(float32(a))
		dst.DrawImage(px, op2)
	}
}

func (g *Game) drawJRex(dst *ebiten.Image, r *JRex) {
	imgs := g.as.props["jp_trex"]
	if len(imgs) == 0 {
		return
	}
	// the pose of the moment
	v := 0
	switch r.state {
	case jrRoar:
		v = 4
	case jrEat:
		e := r.eatE
		switch {
		case e < 0.2:
			v = 0
		case e < 2.4:
			v = 1
			if r.biteOn {
				v = 2
			}
		default:
			v = 3
		}
	}
	if v >= len(imgs) {
		v = 0
	}
	pi := imgs[v]
	x, y := g.wx(r.pos.X), g.wy(r.pos.Y)
	for i := 0; i < 3; i++ { // a soft shadow
		fillRect(dst, x-34+float64(i)*3, y-2+float64(i), 68-float64(i)*6, 1, color.RGBA{0, 0, 0, uint8(40 + i*25)})
	}
	bob, lean, jx, jy := 0.0, 0.0, 0.0, 0.0
	switch r.state {
	case jrRoar:
		jx, jy = (rand.Float64()-0.5)*1.6, (rand.Float64()-0.5)*1.2
	case jrEat:
		if v == 2 { // it pulls: the whole body lunges and shakes
			jx, jy = -r.face*2.5+(rand.Float64()-0.5)*1.5, 1+(rand.Float64()-0.5)
			lean = 0.03 * -r.face
		}
		if v == 3 {
			jy = -math.Abs(math.Sin(r.eatE*14)) * 1.2
		}
	default:
		if r.moving {
			bob = -math.Abs(math.Sin(r.phase*6.4)) * 2.4
			lean = math.Sin(r.phase*6.4) * 0.025
		}
	}
	op := &ebiten.DrawImageOptions{}
	// the sprite looks left; pivot on the feet
	op.GeoM.Translate(-pi.ox, -pi.oy)
	if r.face > 0 {
		op.GeoM.Scale(-1, 1)
	}
	op.GeoM.Rotate(lean)
	op.GeoM.Scale(rexK, rexK)
	op.GeoM.Translate(math.Round(x+jx), math.Round(y+bob+jy))
	dst.DrawImage(pi.img, op)
	if r.state == jrRoar { // sound waves from the raised head
		for i := 0; i < 3; i++ {
			rad := 14 + float64(i)*10 + math.Mod(r.phase*40, 10)
			a := uint8(120 - i*35)
			for ang := -0.7; ang <= 0.7; ang += 0.1 {
				fillRect(dst, math.Round(x+r.face*(36+rad*math.Cos(ang))), math.Round(y-70+rad*math.Sin(ang)), 1, 1, color.RGBA{a, a, a, a})
			}
		}
	}
}

// ---- the roar ---------------------------------------------------------------------------

// jrRoar synthesises a roar: a low rumbling growl that climbs into a rasping shriek
// and falls away again, with the stutter of the real thing and a tail of echo.
func jrRoarSound() []byte {
	const dur = 2.6
	n := int(dur * sampleRate)
	out := make([]float64, n)
	rng := rand.New(rand.NewSource(9))
	var ph, ph2 float64
	var nz float64
	for i := 0; i < n; i++ {
		t := float64(i) / sampleRate
		// pitch: a rise to the shriek at about 0.7s, then a long fall
		var f float64
		switch {
		case t < 0.7:
			f = 70 + 330*math.Pow(t/0.7, 1.6)
		default:
			f = 400 - 330*math.Min(1, (t-0.7)/1.7)
		}
		f *= 1 + 0.025*math.Sin(2*math.Pi*7*t) // wobble
		ph += 2 * math.Pi * f / sampleRate
		ph2 += 2 * math.Pi * f * 0.5 / sampleRate
		// buzz: saw-ish stack of harmonics
		v := 0.0
		for k := 1; k <= 9; k++ {
			v += math.Sin(float64(k)*ph) / float64(k)
		}
		v *= 0.55
		v += 0.5 * math.Sin(ph2)
		// the growl: amplitude chopped at about 30Hz
		am := 0.62 + 0.38*math.Sin(2*math.Pi*(26+10*t)*t)
		v *= am
		// rasp
		nz = 0.82*nz + 0.18*(rng.Float64()*2-1)
		v += nz * 1.4 * math.Min(1, t/0.5)
		env := 1.0
		switch {
		case t < 0.08:
			env = t / 0.08
		case t > 1.5:
			env = math.Exp(-(t - 1.5) * 2.2)
		}
		out[i] = v * env * 0.34
	}
	// a few echoes
	for _, e := range []struct{ d, g float64 }{{0.19, 0.3}, {0.43, 0.18}, {0.8, 0.1}} {
		off := int(e.d * sampleRate)
		for i := n - 1; i >= off; i-- {
			out[i] += out[i-off] * e.g
		}
	}
	return toPCM(lowpass(out, 2))
}
