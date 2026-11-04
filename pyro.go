package main

// ---------------------------------------------------------------------------
// The arsonist: a madman who turns up in some parties (about one in three, never
// the very first party of the campaign, never the island) and sets fire to the
// furniture. He is a guest in every technical way (he can be stabbed like anybody,
// the hover label and the body work as usual) but he does not count for the guest
// cap nor as somebody who has to die for the party to end: kill him or leave him.
// He lights a prop, laughs, goes for another one (three fires at the most), runs
// away from the firefighters and leaves once he is done. Killing him gives the
// "Bombero honorario" trophy. See fire.go for the flames.
// ---------------------------------------------------------------------------

// pyroChance is the chance that a party has an arsonist.
var pyroChance = 0.3

// pyroNow forces him (debug flag -pyronow).
var pyroNow bool

const maxIgnitions = 3

type pyroSt struct {
	phase    int // 0 choose a prop, 1 walk to it, 2 light it, 3 run from the firefighters, 4 leave
	target   *Prop
	timer    float64
	lit      int
	laughT   float64
	lastPos  Vec
	stuckT   float64
	iconT    float64
	leaveTo  Vec
	flameOut float64
}

func pyroSpec() CharSpec {
	return CharSpec{Name: "piromano", DisplayName: "Pirómano", Skin: hx(0xe0b090), Hair: hx(0xe86a20), Hair2: HairSpiky,
		Shirt: hx(0x4a3a2a), Shirt2: hx(0x2a2420), Pants: hx(0x3a3028), Shoes: hx(0x14141a), Top: TopVest, Accent: hx(0x2a2420),
		Glasses: GlassBig, Beard: BeardStubble, Body: BodySlim, Carry: HoldTorch, CarryCol: hx(0xf08a20)}
}

var pyroLines = []string{"Qué calorcito", "Ardo en deseos", "Fuego, fuego", "¡Que arda Troya!", "Hoy toca barbacoa", "Esto necesita ambiente", "Chispa, chispa", "Un fueguito y me voy"}
var pyroFlee = []string{"Yo no he sido", "¡Ay, el agua!", "¡Que me apagan!", "Era una vela, lo juro"}

// pyroPlan decides at the start of a party whether the arsonist comes and when.
func (g *Game) pyroPlan() {
	g.blaze.pyroAt, g.blaze.pyroDone = 0, false
	if g.lvl == nil || g.lvl.Def.CharScale > 0 || g.lvl.Def.PaintBG != nil {
		return // not on the island
	}
	n := 0
	for _, pr := range g.lvl.Props {
		if g.isFlammable(pr) {
			n++
		}
	}
	if n < 3 {
		return
	}
	if pyroNow {
		g.blaze.pyroAt = 2
		return
	}
	if g.levelIdx == 0 || g.rng.Float64() >= pyroChance {
		return
	}
	g.blaze.pyroAt = 16 + g.rng.Float64()*30
}

// updatePyroWorld runs every frame: brings him in when it is time and keeps the fires going.
func (g *Game) updatePyroWorld(dt float64) {
	if g.lvl == nil || g.mode != ModePlay {
		return
	}
	if pyroNow && g.blaze.pyroAt == 0 && !g.blaze.pyroDone && g.lvl.Def.CharScale == 0 { // the flag is read after the level has been planned
		g.blaze.pyroAt = g.time + 2
	}
	if g.blaze.pyroAt > 0 && !g.blaze.pyroDone && g.time >= g.blaze.pyroAt {
		g.blaze.pyroDone = true
		g.spawnPyro()
	}
	g.updateFires(dt)
}

func (g *Game) spawnPyro() {
	spec := pyroSpec()
	pos := g.lvl.NearestFree(g.farRoam(g.p.pos))
	gu := &Guest{pos: pos, dir: DirDown, cs: newCharSet(&spec), name: "Pirómano", style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: 1, role: g.rng.Intn(2), py: &pyroSt{lastPos: pos}}
	g.guests = append(g.guests, gu)
	g.snd.Play("alert")
	g.say("Un pirómano se ha colado en la fiesta")
	g.sayNews("arson_arrive", nil)
}

// pyroTarget: a flammable prop, preferably one near him, that nobody has lit yet.
func (g *Game) pyroTarget(gu *Guest) *Prop {
	var cand []*Prop
	for _, pr := range g.lvl.Props {
		if g.isFlammable(pr) {
			cand = append(cand, pr)
		}
	}
	if len(cand) == 0 {
		return nil
	}
	best, bd := cand[g.rng.Intn(len(cand))], 1e9
	for i := 0; i < 4; i++ {
		c := cand[g.rng.Intn(len(cand))]
		if d := dist(c.footPoint(), gu.pos); d < bd {
			best, bd = c, d
		}
	}
	return best
}

// footPoint is where somebody stands to reach the prop.
func (pr *Prop) footPoint() Vec { return Vec{pr.X, pr.Y + 12} }

func (g *Game) pyroSay(gu *Guest, cat string, lines []string) {
	if gu.iconT > 0 {
		return
	}
	if s := g.freshPhrase(cat, lines, g.rng.Intn); s != "" {
		gu.icon, gu.iconT = s, 2
	}
}

// firemenNear: a fireman close enough to scare him.
func (g *Game) firemenNear(p Vec) bool {
	for _, f := range g.fireCrew {
		if f.state != 3 && dist(f.pos, p) < 130 {
			return true
		}
	}
	return false
}

func (g *Game) updatePyro(gu *Guest, dt float64) {
	py := gu.py
	if gu.iconT > 0 {
		gu.iconT -= dt
		if gu.iconT <= 0 {
			gu.icon = ""
		}
	}
	gu.moving = false
	// scared of the hoses
	if py.phase != 3 && py.phase != 4 && g.firemenNear(gu.pos) {
		py.phase, py.timer = 3, 6
		gu.path = nil
		g.pyroSay(gu, "pyroflee", pyroFlee)
	}
	switch py.phase {
	case 0: // choose
		gu.state = gIdle
		py.timer -= dt
		if py.timer > 0 {
			break
		}
		if py.lit >= maxIgnitions {
			py.phase = 4
			py.leaveTo = g.farRoam(gu.pos)
			break
		}
		py.target = g.pyroTarget(gu)
		if py.target == nil {
			py.phase = 4
			py.leaveTo = g.farRoam(gu.pos)
			break
		}
		gu.path = g.lvl.PathG(gu.pos, g.lvl.NearestFree(py.target.footPoint()))
		py.phase = 1
		g.pyroSay(gu, "pyro", pyroLines)
	case 1: // walk to the prop
		gu.state = gGoto
		if py.target == nil || !g.isFlammable(py.target) {
			py.phase, py.timer = 0, 0.3
			break
		}
		arr := gu.walk(50, dt)
		if arr || dist(gu.pos, py.target.footPoint()) < 12 {
			py.phase, py.timer = 2, 1.3
			gu.dir = DirUp
			gu.path = nil
		}
	case 2: // light it
		gu.state, gu.fun = gFun, 7
		py.timer -= dt
		if int(py.timer*10)%3 == 0 {
			g.burst(Vec{gu.pos.X + 6, gu.pos.Y - 16}, 1, 30, rgb(0xffb030), 0.3, 1, 60)
		}
		if py.timer <= 0 {
			if py.target != nil && g.isFlammable(py.target) {
				if g.igniteProp(py.target, nil) != nil {
					py.lit++
					g.blaze.lit++
					g.sayNews("arson_fire", nil)
				}
			}
			py.target = nil
			gu.state = gIdle
			py.phase, py.timer = 0, 4+g.rng.Float64()*4
			g.pyroSay(gu, "pyro", pyroLines)
		}
	case 3: // run from the firefighters
		gu.state = gRun
		py.timer -= dt
		if len(gu.path) == 0 {
			gu.path = g.lvl.PathG(gu.pos, g.farRoam(gu.pos))
		}
		gu.walk(66, dt)
		if py.timer <= 0 {
			py.phase, py.timer = 0, 2
			gu.state = gIdle
		}
	case 4: // leave the party
		gu.state = gGoto
		if len(gu.path) == 0 {
			gu.path = g.lvl.PathG(gu.pos, g.lvl.NearestFree(py.leaveTo))
		}
		if gu.walk(56, dt) || dist(gu.pos, py.leaveTo) < 14 {
			gu.gone = true
			return
		}
	}
	g.pyroUnstick(gu, dt)
	if gu.moving {
		gu.anim += dt * stridePhase(gu.curSpd)
	}
}

// pyroUnstick: nothing stays stuck for more than five seconds; he is moved a little to a free spot.
func (g *Game) pyroUnstick(gu *Guest, dt float64) {
	py := gu.py
	if dist(gu.pos, py.lastPos) < 1 && (py.phase == 1 || py.phase == 3 || py.phase == 4) {
		py.stuckT += dt
		if py.stuckT > 5 {
			py.stuckT = 0
			gu.pos = g.lvl.NearestFree(Vec{gu.pos.X + 18*(g.rng.Float64()*2-1), gu.pos.Y + 18*(g.rng.Float64()*2-1)})
			gu.path = nil
		}
	} else {
		py.stuckT = 0
	}
	py.lastPos = gu.pos
}
