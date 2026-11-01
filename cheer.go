package main

import "image/color"

// Cheerleaders: a squad turns up now and then, picks a free patch of ground,
// shakes its pompoms for half a minute and then melts into the party. They are
// ordinary guests (they can be killed, they get scared) with a routine.

var cheerTeams = []struct{ top, skirt, pom uint32 }{
	{0xd02030, 0xf4f4f4, 0xf0d030}, // red and white, gold pompoms
	{0x2050d0, 0xf0d030, 0xf4f4f4}, // blue and yellow
	{0xe060a8, 0x3a2a50, 0xf4f4f4}, // pink and plum
	{0x20a050, 0xf4f4f4, 0xf0e060}, // green and white
	{0x8a30c8, 0xf0c030, 0xf0c030}, // purple and gold
}

var cheerShouts = []string{"¡VAMOS!", "¡P-A-R-T-Y!", "¡GO GO GO!", "¡DALE CAÑA!", "¡QUE NO PARE!", "¡SÍ SE PUEDE!", "¡ESA FIESTA!", "¡UH-UH-AH!", "¡GIVE ME A P!", "¡OLE OLE!", "¡MÁS FUERTE!"}

func (g *Game) newCheerleader(team int) *Guest {
	t := cheerTeams[team%len(cheerTeams)]
	spec := genericSpec(g.rng)
	spec.Female = true
	spec.Stature = []int{0, 0, -1}[g.rng.Intn(3)]
	spec.Body = []int{BodyNormal, BodySlim, BodyNormal}[g.rng.Intn(3)]
	spec.Top, spec.Bottom = TopTank, BotSkirt
	spec.Shirt, spec.Shirt2 = hx(t.top), hx(t.top)
	spec.Pants, spec.Shoes = hx(t.skirt), hx(0xf4f4f4)
	spec.Hair2 = []int{HairPony, HairPony, HairBuns, HairLong}[g.rng.Intn(4)]
	spec.Hat, spec.Glasses, spec.Beard, spec.Mask = HatNone, GlassNone, BeardNone, color.RGBA{}
	spec.Makeup = true
	spec.Bust = 1
	spec.CarryCol = hx(t.pom)
	cs := newCharSet(&spec)
	pos := g.lvl.NearestFree(g.lvl.Def.Start)
	gu := &Guest{pos: pos, dir: DirDown, cs: cs, style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: 0.5, role: 1, keep: true}
	gu.name = ""
	g.guests = append(g.guests, gu)
	g.total++
	return gu
}

func (g *Game) cheerAlive() []*Guest {
	var out []*Guest
	for _, m := range g.cheerGroup {
		if !m.out() && m.state != gKiller && m.state != gZombie && m.state != gLift {
			out = append(out, m)
		}
	}
	return out
}

func (g *Game) cheerPerforming() bool {
	for _, m := range g.cheerGroup {
		if m.out() || m.fun != 13 {
			continue
		}
		if m.state == gFun || (m.state == gGoto && m.next == gFun) {
			return true
		}
	}
	return false
}

func (g *Game) updateCheer() {
	if g.mode != ModePlay || g.time < g.cheerT || g.testVictim != nil || g.cheerPerforming() {
		return
	}
	if g.p.dead || g.policeCalled && g.stars >= 2 {
		return
	}
	alive := g.cheerAlive()
	if len(alive) < 3 {
		if g.cheerGroups >= 2 || g.total+5 > g.maxTotal() {
			g.cheerT = 1e9
			return
		}
		g.cheerGroups++
		team := g.rng.Intn(len(cheerTeams))
		n := 4 + g.rng.Intn(2)
		g.cheerGroup = nil
		for i := 0; i < n; i++ {
			m := g.newCheerleader(team)
			m.pos = g.lvl.NearestFree(Vec{g.lvl.Def.Start.X + float64(i)*14 - 24, g.lvl.Def.Start.Y})
			g.cheerGroup = append(g.cheerGroup, m)
		}
		alive = g.cheerGroup
		g.say("¡Llegan las animadoras con sus pompones!")
		g.snd.Play("alert")
	}
	spot, ok := g.openSpot(float64(len(alive))*26+30, 80)
	if !ok {
		g.cheerT = g.time + 40
		return
	}
	dur := 26 + g.rng.Float64()*12
	x0 := spot.X - float64(len(alive)-1)*13
	for i, m := range alive {
		slot := g.lvl.NearestFree(Vec{x0 + float64(i)*26, spot.Y + 16 + float64(i%2)*3})
		m.path = g.lvl.Path(m.pos, slot)
		m.state, m.next, m.dur, m.fun = gGoto, gFun, dur, 13
		m.hurry, m.hurryT = true, -99 // they run to their marks, and never have accidents on the way
		if m.path == nil {
			m.pos = slot
			m.state, m.timer = gFun, dur
		}
	}
	g.cheerT = g.time + dur + 80 + g.rng.Float64()*70
	if g.dbgSnakeOn {
		println("cheer spot", int(spot.X), int(spot.Y), len(alive))
	}
}

func (g *Game) updateCheerer(gu *Guest, dt float64) {
	gu.dir = DirDown
	if gu.iconT <= 0 && g.rng.Float64() < dt*0.45 {
		gu.icon, gu.iconT = cheerShouts[g.rng.Intn(len(cheerShouts))], 1.6
	}
	if g.rng.Float64() < dt*0.9 && gu.cs != nil && gu.cs.Spec != nil {
		g.burst(Vec{gu.pos.X + g.rng.Float64()*20 - 10, gu.pos.Y - 36}, 1, 28, gu.cs.Spec.CarryCol, 0.7, 2, 40)
	}
}
