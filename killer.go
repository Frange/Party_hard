package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The special enemy ("slasher"): at most one per level. He shows up in the middle of the party
// in a macabre car (or a black helicopter on roof levels), hunts people, and can be killed by
// the player or arrested by the police.
// ---------------------------------------------------------------------------

const (
	slArrive = iota
	slHunt
	slCuffed
	slGone
)

type Slasher struct {
	pos      Vec
	dir      Dir
	cs       *CharSet
	state    int
	path     []Vec
	hp       int
	anim     float64
	moving   bool
	curSpd   float64
	repath   float64
	pause    float64
	timer    float64
	icon     string
	iconT    float64
	sparkT   float64
	sawT     float64
	trail    Vec
	chainsaw bool
	tgtG     *Guest
	tgtD     *Guard
	tgtP     bool
	name     string
	stabT    float64
	copRoll  float64
}

const guestRunSpeed = 112.0

// copOnScene: is there a police officer around (not a paramedic, not leaving)?
func (g *Game) copOnScene() bool {
	for _, c := range g.cops {
		if !c.medic && c.state != cLeave {
			return true
		}
	}
	return false
}

func (v *Vehicle) isHeli() bool { return v.kind == 2 || v.kind == 3 || v.kind == 5 }

// pickKillerCS chooses the look of the special enemy from the assets (or the built-ins).
func (g *Game) pickKiller() (*CharSet, bool, string) {
	if len(g.as.killerCS) > 0 {
		k := g.rng.Intn(len(g.as.killerCS))
		cs := g.as.killerCS[k]
		return cs, cs.Spec.Weapon == HoldChainsaw || (cs.rig != nil && cs.rig.carry == HoldChainsaw), cs.Spec.Name
	}
	e := killerPresets[g.rng.Intn(len(killerPresets))]
	spec := e.F(g.rng)
	return newCharSet(&spec), spec.Weapon == HoldChainsaw, e.Name
}

func (g *Game) updateSlasher(dt float64) {
	def := g.lvl.Def
	s := g.slasher
	if s == nil {
		if g.slasherDone || g.time < g.slasherT || g.testVictim != nil || len(g.guests) < 4 {
			return
		}
		cs, saw, name := g.pickKiller()
		var v *Vehicle
		if def.Police == "heli" {
			pad := g.freePad()
			if pad < 0 {
				g.slasherT = g.time + 5
				return
			}
			p := def.Pads[pad]
			v = &Vehicle{kind: 5, x: p.X, y: -130, tx: p.X, ty: p.Y, pad: pad}
		} else {
			n := 0
			for _, o := range g.veh {
				if o.state != 2 {
					n++
				}
			}
			v = &Vehicle{kind: 4, x: def.CarFrom.X + 40, y: def.CarFrom.Y, tx: def.CarTo.X - float64(n)*100, ty: def.CarFrom.Y}
		}
		g.veh = append(g.veh, v)
		g.slasherDone = true
		g.slasher = &Slasher{cs: cs, state: slArrive, hp: 3, chainsaw: saw, name: name, dir: DirDown}
		g.snd.Play("alert")
		g.say("Algo siniestro se acerca a la fiesta...")
		return
	}
	if s.iconT > 0 {
		s.iconT -= dt
		if s.iconT <= 0 {
			s.icon = ""
		}
	}
	if s.stabT > 0 {
		s.stabT -= dt
	}
	s.moving = false
	switch s.state {
	case slCuffed:
		s.timer -= dt
		if s.timer <= 0 {
			s.state = slGone
			g.points += 250
			g.say("¡La policía ha detenido al asesino!")
			g.snd.Play("win")
			g.checkDone()
		}
		return
	case slHunt:
	default:
		return
	}
	g.slasherThink(s, dt)
}

// slasherArrives is called when the macabre vehicle parks: the killer steps out.
func (g *Game) slasherArrives(v *Vehicle) {
	s := g.slasher
	if s == nil {
		return
	}
	if v.isHeli() {
		s.pos = g.lvl.NearestFree(Vec{v.tx + 14, v.ty + 22})
	} else {
		s.pos = g.lvl.NearestFree(Vec{v.x - 12, v.y - 24})
	}
	s.state = slHunt
	s.pause = 1.2
	s.trail = s.pos
	g.total++
	g.shake = 0.3
	g.snd.Play("alert")
	g.say("¡Un asesino ha llegado a la fiesta!")
}

func (g *Game) slasherThink(s *Slasher, dt float64) {
	p := g.p
	// who do we want?
	s.sawT -= dt
	if s.sawT <= 0 {
		s.sawT = 0.5
		for _, gu := range g.guests {
			if g.canWatch(gu) && gu.perf == "" && g.sees(gu.pos, gu.dir, s.pos, guestRange, guestDot) {
				gu.sawSlash, gu.sawPos = true, s.pos
				if g.callerAvailable() {
					g.becomeCaller(gu)
				} else {
					g.alarm(gu, "!?")
				}
			}
		}
	}
	if s.pause > 0 {
		s.pause -= dt
		return
	}
	s.repath -= dt
	if s.repath <= 0 {
		s.repath = 0.5
		g.slasherPick(s)
		var tp Vec
		ok := true
		switch {
		case s.tgtP:
			tp = p.pos
		case s.tgtG != nil:
			tp = s.tgtG.pos
		case s.tgtD != nil:
			tp = s.tgtD.pos
		default:
			ok = false
		}
		if ok {
			s.path = g.lvl.Path(s.pos, tp)
		} else {
			s.path = g.lvl.Path(s.pos, g.lvl.RandomRoam(g.rng))
		}
	}
	speed := 38.0
	if s.chainsaw {
		speed = 32
	}
	if s.tgtP {
		speed = 76
	}
	m, _ := follow(&s.pos, &s.dir, &s.path, speed, dt)
	s.moving, s.curSpd = m, speed
	if m {
		s.anim += dt * stridePhase(speed)
		if s.chainsaw {
			s.sparkT -= dt
			if s.sparkT <= 0 {
				s.sparkT = 0.05
				v := dirVec(s.dir)
				at := Vec{s.pos.X + v.X*9, s.pos.Y - 1}
				g.burst(at, 2, 60, rgb(0xffb030), 0.35, 2, 260)
				g.burst(at, 1, 40, rgb(0xfff0a0), 0.2, 1, 200)
				if dist(s.trail, s.pos) > 12 && len(g.decals) < 500 {
					s.trail = s.pos
					g.decals = append(g.decals, Decal{pos: Vec{at.X, at.Y + 1}, kind: 1, r: 3})
				}
			}
		}
	}
	// strike
	var tp Vec
	switch {
	case s.tgtP:
		tp = p.pos
	case s.tgtG != nil:
		tp = s.tgtG.pos
	case s.tgtD != nil:
		tp = s.tgtD.pos
	default:
		return
	}
	if dist(s.pos, tp) < 13 {
		s.stabT = 0.3
		s.pause = 1.1
		switch {
		case s.tgtP:
			if !p.hidden && !p.dead {
				g.snd.Play("stab")
				g.die("Un asesino te ha dado caza")
			}
		case s.tgtG != nil:
			v := s.tgtG
			s.tgtG = nil
			g.killGuest(v, false)
			g.slasherWitness(s, v.pos)
		case s.tgtD != nil:
			d := s.tgtD
			s.tgtD = nil
			d.dead, d.icon = true, ""
			g.spawnBody(d.pos, d.cs, d.dir)
			g.snd.Play("stab")
			g.slasherWitness(s, d.pos)
			g.checkDone()
		}
		s.repath = 0
	}
}

func (g *Game) slasherWitness(s *Slasher, at Vec) {
	g.burst(Vec{at.X, at.Y - 14}, 14, 80, rgb(0xb01818), 0.6, 2, 220)
	for _, o := range g.guests {
		if o.out() || !g.canWatch(o) || o.perf != "" {
			continue
		}
		if g.sees(o.pos, o.dir, at, guestRange, guestDot) || dist(o.pos, at) < 45 {
			o.sawSlash, o.sawPos = true, s.pos
			if len(g.bodies) > 0 {
				g.knowBody(o, g.bodies[len(g.bodies)-1])
			}
			if g.callerAvailable() {
				g.becomeCaller(o)
			} else {
				g.alarm(o, "!?")
			}
		}
	}
}

// slasherPick: the player if he is close, otherwise the closest reachable victim.
func (g *Game) slasherPick(s *Slasher) {
	p := g.p
	if !p.dead && !p.hidden && dist(p.pos, s.pos) < 110 && g.lvl.LOS(s.pos, p.pos) {
		s.tgtP, s.tgtG, s.tgtD = true, nil, nil
		return
	}
	s.tgtP = false
	if s.tgtG != nil && !s.tgtG.out() {
		return
	}
	if s.tgtD != nil && !s.tgtD.dead {
		return
	}
	s.tgtG, s.tgtD = nil, nil
	best := 1e9
	for _, gu := range g.guests {
		if gu.out() {
			continue
		}
		if d := dist(gu.pos, s.pos); d < best {
			best, s.tgtG, s.tgtD = d, gu, nil
		}
	}
	for _, gd := range g.guards {
		if gd.dead {
			continue
		}
		if d := dist(gd.pos, s.pos) + 40; d < best {
			best, s.tgtG, s.tgtD = d, nil, gd
		}
	}
}

// hitSlasher: the player stabs the special enemy. It takes several stabs.
func (g *Game) hitSlasher() {
	s := g.slasher
	s.hp--
	s.pause = 0.5
	g.snd.Play("stab")
	g.burst(Vec{s.pos.X, s.pos.Y - 18}, 14, 90, rgb(0xb01818), 0.6, 2, 220)
	g.shake = 0.15
	if s.hp > 0 {
		s.icon, s.iconT = "!!!", 0.8
		// he fights back: pushes the player away and tries to hit
		g.p.knockT = 0.8
		return
	}
	s.state = slGone
	g.spawnBody(s.pos, s.cs, s.dir)
	g.kills++
	g.points += 500
	g.say("¡Has matado al asesino! +500")
	g.snd.Play("win")
	g.witness()
	g.checkDone()
}

// nearCop returns a police officer (not paramedic) who is around and could take a report.
func (g *Game) nearCop(pos Vec) *Cop {
	var best *Cop
	for _, c := range g.cops {
		if c.medic || c.state == cLeave {
			continue
		}
		if d := dist(c.pos, pos); d < 420 && (best == nil || d < dist(best.pos, pos)) {
			best = c
		}
	}
	return best
}

func (g *Game) scare(gu *Guest) {
	gu.state, gu.timer, gu.path = gScared, 6+g.rng.Float64()*4, nil
	gu.icon, gu.iconT = "!!", 1
}

func (g *Game) farRoam(from Vec) Vec {
	var t Vec
	for i := 0; i < 20; i++ {
		t = g.lvl.RandomRoam(g.rng)
		if dist(t, from) > 160 {
			break
		}
	}
	return t
}

func (g *Game) updateReport(gu *Guest, dt float64) {
	gu.timer -= dt
	c := gu.reportTo
	alive := false
	for _, o := range g.cops {
		if o == c {
			alive = true
		}
	}
	if !alive || gu.timer <= 0 || c.state == cLeave {
		g.scare(gu)
		return
	}
	gu.repath -= dt
	if gu.repath <= 0 {
		gu.repath = 0.5
		gu.path = g.lvl.Path(gu.pos, c.pos)
	}
	gu.walk(guestRunSpeed, dt)
	if dist(gu.pos, c.pos) < 28 {
		gu.icon, gu.iconT = "¡FUE ÉL!", 2.5
		gu.state, gu.timer, gu.path = gIdle, 3, nil
		gu.dir = faceTo(gu.pos, c.pos)
		c.icon, c.iconT = "ENTENDIDO", 2
		var bodyAt Vec
		bd := 1e9
		for _, b := range g.bodies {
			if !b.reported && !b.carried && g.knows(gu, b) {
				if d := dist(b.pos, c.pos); d < bd {
					bd, bodyAt = d, b.pos
				}
			}
		}
		g.deliver(gu)
		if s := g.slasher; gu.sawSlash && s != nil && s.state == slHunt {
			g.releaseCop(c)
			c.state, c.path = cSlash, g.lvl.Path(c.pos, gu.sawPos)
			return
		}
		if gu.sawKiller {
			g.wanted = math.Max(g.wanted, 70)
			g.killPos = gu.sawPos
			g.releaseCop(c)
			c.state, c.wait = cGoto, 0
			c.path = g.lvl.Path(c.pos, gu.sawPos)
		} else if bodyAt != (Vec{}) && c.state != cChase {
			c.state, c.wait = cGoto, 0
			c.path = g.lvl.Path(c.pos, bodyAt)
		}
	}
}

// copSlashHook: police that see the special enemy go for him; returns true when it handled the cop.
func (g *Game) copSlashHook(c *Cop, dt float64) bool {
	s := g.slasher
	if c.medic {
		return false
	}
	active := s != nil && (s.state == slHunt || s.state == slCuffed)
	if c.state != cSlash {
		if active && s.state == slHunt && c.state != cChase && c.state != cTired && g.sees(c.pos, c.dir, s.pos, copRange, copDot) {
			g.releaseCop(c)
			c.state = cSlash
			c.icon, c.iconT = "¡ALTO! ¡POLICÍA!", 2
			g.snd.Play("alert")
		} else {
			return false
		}
	}
	if !active {
		g.copNext(c)
		return false
	}
	if s.state == slCuffed {
		c.moving = false
		return true
	}
	c.repath -= dt
	if c.repath <= 0 || len(c.path) == 0 {
		c.repath = 0.3
		c.path = g.lvl.Path(c.pos, s.pos)
	}
	g.copWalk(c, copChaseSpeed*0.9, dt)
	if dist(c.pos, s.pos) < 16 {
		if c.copRollDone() && g.rng.Float64() < 0.3 {
			// the killer gets the officer first
			c.moving = false
			g.snd.Play("stab")
			g.spawnBody(c.pos, g.as.copCS, c.dir)
			s.stabT, s.pause = 0.3, 1.0
			s.icon, s.iconT = "...", 1.5
			g.say("¡El asesino ha matado a un policía!")
			g.removeCop(c)
			return true
		}
		s.state, s.timer = slCuffed, 2.8
		s.icon, s.iconT = "", 0
		c.icon, c.iconT = "¡QUEDA DETENIDO!", 2.5
		g.snd.Play("alert")
	}
	return true
}

func (c *Cop) copRollDone() bool {
	if c.timer < 0 {
		return false
	}
	c.timer = -1
	return true
}

func (g *Game) removeCop(c *Cop) {
	g.releaseCop(c)
	out := g.cops[:0]
	for _, o := range g.cops {
		if o != c {
			out = append(out, o)
		}
	}
	g.cops = out
}

// maybeBackup: with many bodies a second, different-looking officer comes.
func (g *Game) maybeBackup() {
	if g.pendingBackup >= 0 && g.time >= g.pendingBackup {
		g.pendingBackup = -1
		n := len(g.veh)
		g.dispatch(false)
		if len(g.veh) > n {
			g.veh[len(g.veh)-1].alt = 1 + g.rng.Intn(2)
		} else {
			g.backupCalled = false
		}
	}
	if g.backupCalled || !g.policeCalled || g.pendingBackup >= 0 || g.respKind != 0 {
		return
	}
	n := 0
	for _, b := range g.bodies {
		if b.reported && !b.carried {
			n++
		}
	}
	if n >= 4 {
		g.backupCalled = true
		g.pendingBackup = g.time + 9
		g.say("Piden refuerzos: llega otro agente")
	}
}

// unstick: police and medics that stop making progress are moved to a free spot.
func (g *Game) copUnstick(c *Cop, dt float64) {
	walking := false
	switch c.state {
	case cGoto, cSweep, cCollect, cBag, cBagHere, cLeave, cSearch, cArrest, cSlash, cChase:
		walking = len(c.path) > 0
	}
	if !walking {
		c.stuckT, c.stuckPos, c.stuckChk = 0, c.pos, 0
		return
	}
	c.stuckChk += dt
	if c.stuckChk < 1.0 {
		return
	}
	c.stuckChk = 0
	if dist(c.pos, c.stuckPos) > 5 {
		c.stuckT, c.stuckPos = 0, c.pos
		return
	}
	c.stuckT++
	if c.stuckT < 3 {
		c.path = nil
		return
	}
	c.stuckT, c.stuckPos = 0, c.pos
	switch {
	case c.state == cLeave || c.target == nil && c.state == cCollect:
		c.pos = g.vehDoor(c)
		c.path = nil
		if c.veh == nil {
			g.removeCop(c)
		}
	case len(c.path) > 0:
		k := minI(len(c.path)-1, 2)
		c.pos = g.lvl.NearestFree(c.path[k])
		c.path = nil
	default:
		c.pos = g.lvl.NearestFree(c.pos)
	}
}

// ---------------------------------------------------------------------------
// Drawing
// ---------------------------------------------------------------------------

func (g *Game) drawSlasher(dst *ebiten.Image, s *Slasher) {
	anim, phase := AnimIdle, g.time*1.2+s.pos.X
	switch {
	case s.stabT > 0:
		anim, phase = AnimStab, s.stabT*10
	case s.moving:
		anim, phase = AnimWalk, s.anim
	}
	pos := s.pos
	swim := g.lvl.InPool(pos)
	g.drawActor(dst, s.cs, anim, phase, pos, s.dir, 0, swim)
	if s.state == slCuffed {
		g.drawCuffs(dst, pos)
	}
	if s.chainsaw && s.moving && s.state == slHunt {
		v := dirVec(s.dir)
		x, y := g.wx(pos.X+v.X*10), g.wy(pos.Y)
		fillRect(dst, x-1, y-1, 3, 1, color.RGBA{255, 220, 120, 255})
	}
}

func (g *Game) drawHearse(dst *ebiten.Image, pi PropImg, v *Vehicle) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(g.wx(v.x)-pi.ox), math.Round(g.wy(v.y)-pi.oy))
	op.ColorScale.Scale(0.28, 0.24, 0.32, 1)
	dst.DrawImage(pi.img, op)
	// coffin on the roof and ghostly lights
	cx, cy := math.Round(g.wx(v.x)), math.Round(g.wy(v.y))
	fillRect(dst, cx-14, cy-34, 28, 5, color.RGBA{60, 40, 30, 255})
	fillRect(dst, cx-14, cy-34, 28, 1, color.RGBA{110, 80, 60, 255})
	fillRect(dst, cx-8, cy-31, 16, 1, color.RGBA{200, 170, 60, 255})
}

// unitFor: which kind of responders a new vehicle carries (paramedics are always plain).
func (g *Game) unitFor(medic bool) int {
	if medic {
		return 0
	}
	return g.respKind
}

func (g *Game) drawUnitCar(dst *ebiten.Image, pi PropImg, v *Vehicle) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(g.wx(v.x)-pi.ox), math.Round(g.wy(v.y)-pi.oy))
	label, col := "SWAT", color.RGBA{230, 230, 240, 255}
	if v.unit == 1 {
		op.ColorScale.Scale(0.33, 0.35, 0.4, 1)
	} else {
		op.ColorScale.Scale(0.18, 0.2, 0.28, 1)
		label, col = "FBI", color.RGBA{240, 220, 80, 255}
	}
	dst.DrawImage(pi.img, op)
	g.drawText(dst, label, math.Round(g.wx(v.x))-g.textW(label, 1)/2, math.Round(g.wy(v.y))-30, col, 1)
}

// copBrutal: SWAT and FBI units that cannot find the killer start shooting people in the crowd.
func (g *Game) copBrutal(c *Cop, dt float64) {
	if c.kind == 0 || c.medic || (c.state != cSweep && c.state != cGoto) {
		return
	}
	if c.searchT < c.maxSearch*0.4 {
		return
	}
	limit := 6
	if c.kind == 2 {
		limit = 2
	}
	if g.swatKills >= limit {
		return
	}
	c.shootT -= dt
	if c.shootT > 0 {
		return
	}
	c.shootT = 5 + g.rng.Float64()*4
	var best *Guest
	bd := 190.0
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" {
			continue
		}
		if d := dist(gu.pos, c.pos); d < bd && g.lvl.LOS(c.pos, gu.pos) {
			best, bd = gu, d
		}
	}
	if best == nil {
		return
	}
	g.swatKills++
	c.icon, c.iconT = "¡AL SUELO!", 1.6
	c.dir = faceTo(c.pos, best.pos)
	g.snd.Play("boom")
	g.killGuest(best, false)
	if n := len(g.bodies); n > 0 {
		g.bodies[n-1].reported = true
	}
	for _, o := range g.guests {
		if !o.out() && o.perf == "" && dist(o.pos, best.pos) < 150 && g.canWatch(o) {
			g.scare(o)
		}
	}
	g.say("¡Han disparado a un invitado!")
}

// ---------------------------------------------------------------------------
// The tank on the road crushes whatever vehicle is in its way.
// ---------------------------------------------------------------------------

func (g *Game) explodeAt(x, y float64) {
	g.snd.Play("boom")
	g.shake = 0.5
	g.burst(Vec{x, y - 14}, 36, 130, rgb(0xff9030), 0.8, 3, 60)
	g.burst(Vec{x, y - 14}, 20, 90, rgb(0x303030), 1.0, 3, -20)
	g.decals = append(g.decals, Decal{pos: Vec{x, y}, kind: 1, r: 14})
}

func (g *Game) tankCrush(a *Ambient) {
	// parked cars and motorbikes on the street
	for _, pr := range g.lvl.Props {
		if pr.wreck || (pr.Name != "car" && pr.Name != "motorbike") {
			continue
		}
		if math.Abs(pr.X-a.x) < 36 && math.Abs(pr.Y-a.y) < 40 {
			pr.wreck = true
			g.explodeAt(pr.X, pr.Y)
		}
	}
	// cars passing by
	for _, o := range g.amb {
		if o == a || o.kind == ambTank || o.kind == ambUFO || o.dead {
			continue
		}
		if math.Abs(o.x-a.x) < 50 && math.Abs(o.y-a.y) < 22 {
			o.dead = true
			g.explodeAt(o.x, o.y)
			g.wrecks = append(g.wrecks, Vec{o.x, o.y})
		}
	}
	// emergency vehicles
	for _, v := range g.veh {
		if v.isHeli() || v.dead {
			continue
		}
		if math.Abs(v.x-a.x) < 56 && math.Abs(v.y-a.y) < 26 {
			g.vehicleDestroyed(v)
		}
	}
}

func (g *Game) vehicleDestroyed(v *Vehicle) {
	v.dead = true
	g.explodeAt(v.x, v.y)
	g.wrecks = append(g.wrecks, Vec{v.x, v.y})
	switch v.kind {
	case 0, 1:
		if !v.spawned {
			g.say("¡El tanque ha aplastado un vehículo de emergencias con su tripulación!")
			// the crew was still inside: they never come out. Another unit is sent a bit later.
			if v.kind == 0 {
				g.pendingCar = g.time + 9
			} else {
				g.pendingAmb = g.time + 9
				g.ambCalled = true
			}
		} else {
			g.say("¡El tanque ha aplastado el vehículo de emergencias!")
		}
	case 4:
		if !v.spawned && g.slasher != nil && g.slasher.state == slArrive {
			g.slasher.state = slGone
			g.say("¡El tanque ha aplastado el coche fúnebre con el asesino dentro!")
		}
	case 6:
		if !v.spawned {
			g.say("¡El tanque ha aplastado una limusina llena de invitados!")
		}
	}
}

func (g *Game) drawFlames(dst *ebiten.Image, x, y, w float64) {
	for i := 0; i < 5; i++ {
		ph := g.time*9 + float64(i)*1.7 + x
		h := 5 + 4*math.Abs(math.Sin(ph))
		fx := x - w/2 + float64(i)*w/4
		fillRect(dst, math.Round(fx), math.Round(y-h), 3, math.Round(h), color.RGBA{250, 140, 30, 230})
		fillRect(dst, math.Round(fx)+1, math.Round(y-h*0.6), 1, math.Round(h*0.6), color.RGBA{255, 230, 120, 240})
	}
}
