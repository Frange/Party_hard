package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Firefighters: when something burns (a wrecked car) and nobody puts it out,
// a fire engine drives in and the crew hoses the flames down.
// ---------------------------------------------------------------------------

func init() {
	propKinds["firetruck"] = &PropKind{W: 96, H: 46, FW: 90, FH: 12, NoSolid: true, Vars: 1, Category: "vehicle", Draw: func(b *pixBuf, v int) { vaFireTruckSide(b) }}
}

type Fireman struct {
	pos    Vec
	dir    Dir
	cs     *CharSet
	path   []Vec
	state  int // 0 to the fire, 1 spraying, 2 back to the truck, 3 gone
	timer  float64
	anim   float64
	moving bool
	curSpd float64
	truck  *Vehicle
	fire   *fireRef
}

type fireRef struct {
	pos     Vec
	prop    *Prop
	wreck   Vec
	claimed *Fireman
	fire    *Fire // flames of the arsonist (fire.go)
}

func (f *fireRef) out(g *Game) bool {
	if f.fire != nil {
		return f.fire.out
	}
	if f.prop != nil {
		return f.prop.fireOut
	}
	return g.wreckOut[f.wreck]
}

func (f *fireRef) extinguish(g *Game) {
	if f.fire != nil {
		f.fire.putOut(g, true)
		return
	}
	if f.prop != nil {
		f.prop.fireOut = true
		return
	}
	if g.wreckOut == nil {
		g.wreckOut = map[Vec]bool{}
	}
	g.wreckOut[f.wreck] = true
}

func fireSpec() CharSpec {
	return CharSpec{Name: "firefighter", DisplayName: "Bombero", Skin: hx(0xe8b48a), Hair: hx(0x2b1d12), Shirt: hx(0x2a2a30), Shirt2: hx(0xf0c820),
		Pants: hx(0x2a2a30), Shoes: hx(0x14141a), Top: TopVest, Hat: HatHardhat, HatCol: hx(0xe83030), Accent: hx(0xf0c820), Hair2: HairShort, Body: BodyBig,
		Carry: HoldExt, CarryCol: hx(0xd02a2a)}
}

// openFires lists every flame that nobody is handling yet.
func (g *Game) openFires() []*fireRef {
	var out []*fireRef
	for _, w := range g.wrecks {
		if !g.wreckOut[w] {
			out = append(out, &fireRef{pos: w, wreck: w})
		}
	}
	for _, pr := range g.lvl.Props {
		if pr.wreck && !pr.fireOut {
			out = append(out, &fireRef{pos: Vec{pr.X, pr.Y}, prop: pr})
		}
	}
	for _, bf := range g.blaze.fires {
		if !bf.out {
			out = append(out, &fireRef{pos: bf.pos, fire: bf})
		}
	}
	return out
}

func (g *Game) updateFire(dt float64) {
	if g.lvl == nil || g.mode != ModePlay {
		return
	}
	// levels without a road for the engine still get firefighters when something burns: they run in on foot
	onFoot := g.lvl.Def.Police != "car"
	if onFoot && g.fireTruck == nil && len(g.fireCrew) == 0 && g.liveFires() == 0 {
		return
	}
	if fastEv && !g.fireTested && g.time > 4 {
		g.fireTested = true
		g.wrecks = append(g.wrecks, Vec{g.lvl.Def.CarFrom.X - 300, g.lvl.Def.CarFrom.Y + g.laneOff()})
	}
	fires := g.openFires()
	// call the engine once something has been burning for a while
	if g.fireTruck == nil && len(fires) > 0 {
		if g.fireWait <= 0 {
			g.fireWait = 10 + g.rng.Float64()*6
			if g.liveFires() > 0 {
				g.fireWait = 6 + g.rng.Float64()*3 // somebody has just called: they are quicker for a blaze
			}
		}
		g.fireWait -= dt
		if g.fireWait <= 0 {
			if onFoot {
				g.dispatchFireOnFoot()
			} else {
				g.dispatchFire()
			}
			g.fireWait = 0
		}
	}
	if g.fireTruck != nil && g.fireTruck.dead {
		g.fireTruck, g.fireCrew = nil, nil
	}
	alive := g.fireCrew[:0]
	for _, f := range g.fireCrew {
		g.updateFireman(f, dt)
		if f.state != 3 {
			alive = append(alive, f)
		}
	}
	g.fireCrew = alive
	if g.fireTruck != nil && g.fireTruck.spawned && len(g.fireCrew) == 0 && g.fireTruck.state == 1 {
		g.fireTruck.state = 2
		g.fireTruck = nil
	}
}

// dispatchFireOnFoot: no engine can drive in (boats, helicopters, the island): the crew runs in from the edge.
func (g *Game) dispatchFireOnFoot() {
	def := g.lvl.Def
	from := g.lvl.NearestFree(Vec{def.Start.X, def.Start.Y})
	if def.CarFrom.X != 0 || def.CarFrom.Y != 0 {
		from = g.lvl.NearestFree(Vec{def.CarFrom.X, def.CarFrom.Y})
	}
	home := &Vehicle{kind: 7, x: from.X, y: from.Y + 24, spawned: true, state: 1}
	g.fireTruck = home
	g.snd.Play("siren")
	g.say("Los bomberos corren hacia el incendio")
	g.firemenArrive(home)
}

func (g *Game) dispatchFire() {
	def := g.lvl.Def
	n := 0
	for _, v := range g.veh {
		if v.state != 2 {
			n++
		}
	}
	tx := def.CarTo.X - float64(n)*100
	if def.ParkX1 > def.ParkX0 {
		if def.CarFrom.X > def.CarTo.X {
			tx = math.Max(def.ParkX0, def.ParkX1-float64(n)*100)
		} else {
			tx = math.Min(def.ParkX1, def.ParkX0+float64(n)*100)
		}
	}
	ly := def.CarFrom.Y + g.laneOff()
	if def.CarFrom.X > def.CarTo.X {
		ly = def.CarFrom.Y - g.laneOff()
	}
	v := &Vehicle{kind: 7, x: def.CarFrom.X + 60, y: ly, tx: tx, ty: ly}
	g.veh = append(g.veh, v)
	g.fireTruck = v
	g.snd.Play("siren")
	g.say("Los bomberos van de camino al incendio")
}

// firemenArrive: the engine has parked, the crew gets out.
func (g *Game) firemenArrive(v *Vehicle) {
	fires := g.openFires()
	n := 2 + g.rng.Intn(2)
	if len(fires) > 2 {
		n = 4
	}
	spec := fireSpec()
	for i := 0; i < n; i++ {
		sp := spec
		sp.Hair = hx(pick(g.rng, hairTones))
		cs := newCharSet(&sp)
		if len(g.as.fireCS) > 0 {
			cs = g.as.fireCS[i%len(g.as.fireCS)]
		}
		pos := g.lvl.NearestFree(Vec{v.x - 14 + float64(i)*10, v.y - 24 - float64(i%2)*6})
		f := &Fireman{pos: pos, dir: DirDown, cs: cs, truck: v}
		g.fireCrew = append(g.fireCrew, f)
		g.firePick(f)
	}
	g.say("Llegan los bomberos")
}

// firePick sends a fireman to the nearest fire that no colleague is on yet.
func (g *Game) firePick(f *Fireman) {
	var best *fireRef
	bd := 1e9
	for _, r := range g.openFires() {
		taken := false
		for _, o := range g.fireCrew {
			if o != f && o.fire != nil && o.fire.pos == r.pos && o.state <= 1 {
				taken = true
			}
		}
		if taken && len(g.fireCrew) > 1 {
			continue
		}
		if d := dist(f.pos, r.pos); d < bd {
			best, bd = r, d
		}
	}
	if best == nil {
		f.state, f.fire = 2, nil
		f.path = g.lvl.Path(f.pos, g.lvl.NearestFree(Vec{f.truck.x - 12, f.truck.y - 24}))
		return
	}
	f.fire, f.state = best, 0
	f.path = g.lvl.Path(f.pos, g.lvl.NearestFree(Vec{best.pos.X, best.pos.Y + 22}))
}

func (g *Game) updateFireman(f *Fireman, dt float64) {
	f.moving = false
	switch f.state {
	case 0:
		m, arr := follow(&f.pos, &f.dir, &f.path, 70*paceK, dt)
		f.moving, f.curSpd = m, 70
		if arr || dist(f.pos, f.fire.pos) < 30 {
			f.state, f.timer = 1, 5+g.rng.Float64()*3
			f.dir = DirUp
		}
	case 1:
		f.timer -= dt
		// the jet of water
		for i := 0; i < 2; i++ {
			g.parts = append(g.parts, Particle{pos: Vec{f.pos.X + 5, f.pos.Y - 20}, vel: Vec{(f.fire.pos.X - f.pos.X) * 2.2 * (0.7 + g.rng.Float64()*0.5), (f.fire.pos.Y - 14 - f.pos.Y + 20) * 2.2 * (0.7 + g.rng.Float64()*0.5)}, life: 0.45, max: 0.45, col: rgb(0xa0d8f8), size: 2, grav: 40})
		}
		if f.timer <= 0 {
			if f.fire != nil && !f.fire.out(g) {
				f.fire.extinguish(g)
				g.burst(Vec{f.fire.pos.X, f.fire.pos.Y - 10}, 14, 40, rgb(0xd0d0d8), 0.8, 3, -30)
				g.say("Los bomberos apagan el fuego")
			}
			g.firePick(f)
		}
	case 2:
		m, arr := follow(&f.pos, &f.dir, &f.path, 70*paceK, dt)
		f.moving, f.curSpd = m, 70
		if arr || (f.truck != nil && dist(f.pos, Vec{f.truck.x - 12, f.truck.y - 24}) < 16) {
			f.state = 3
		}
	}
	if f.moving {
		f.anim += dt * stridePhase(f.curSpd)
	}
}

func (g *Game) drawFireman(dst *ebiten.Image, f *Fireman) {
	anim, phase := AnimIdle, g.time*1.2+f.pos.X
	switch {
	case f.moving:
		anim, phase = AnimWalk, f.anim
	case f.state == 1:
		anim, phase = AnimPoint, g.time*2
	}
	g.drawActor(dst, f.cs, anim, phase, f.pos, f.dir, 0, false)
}

func (g *Game) drawFireTruck(dst *ebiten.Image, v *Vehicle) {
	imgs := g.as.props["firetruck"]
	if len(imgs) == 0 {
		return
	}
	pi := imgs[0]
	w := float64(pi.img.Bounds().Dx())
	op := &ebiten.DrawImageOptions{}
	// the sprite faces right: mirror it while it drives in from the right
	if v.state != 2 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(w, 0)
		op.GeoM.Translate(math.Round(g.wx(v.x)-(w-pi.ox)), math.Round(g.wy(v.y)-pi.oy))
	} else {
		op.GeoM.Translate(math.Round(g.wx(v.x)-pi.ox), math.Round(g.wy(v.y)-pi.oy))
	}
	dst.DrawImage(pi.img, op)
	if v.state != 1 || int(g.time*6)%2 == 0 {
		g.glowAt(dst, Vec{v.x - 6, v.y - 36}, 26, color.RGBA{220, 30, 30, 120})
	} else {
		g.glowAt(dst, Vec{v.x + 6, v.y - 36}, 26, color.RGBA{30, 70, 220, 120})
	}
}
