package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// worldBubble: a speech bubble that belongs to the scenery (an event actor), not to a guest.
type worldBubble struct {
	pos Vec
	txt string
	t   float64
}

func (g *Game) updateBubbles(dt float64) {
	out := g.bubbles[:0]
	for _, b := range g.bubbles {
		b.t -= dt
		if b.t > 0 {
			out = append(out, b)
		}
	}
	g.bubbles = out
}

type worldBubbleKey struct {
	pos Vec
	txt string
}

func (g *Game) drawBubbles(dst *ebiten.Image) {
	for _, b := range g.bubbles {
		g.bubble(worldBubbleKey{b.pos, b.txt}, bubGuest, b.pos, b.txt, false, 0)
	}
}

// ---- ball game --------------------------------------------------------------------

// startSport: four or five idle guests meet in a clear patch and pass a ball
// around for half a minute; then they go back to the party.
func (g *Game) startSport(e *Event) bool {
	var idle []*Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" && gu.state == gIdle && gu.timer < 1000 && gu.partner == nil && gu.leader == nil {
			idle = append(idle, gu)
		}
	}
	if len(idle) < 4 {
		return false
	}
	// a patch of open ground around somebody's spot
	var spot Vec
	ok := false
	for try := 0; try < 40 && !ok; try++ {
		p := g.lvl.RandomRoam(g.rng)
		if g.lvl.badGround(p) || g.lvl.InPool(p) || !g.lvl.Outdoor(p) {
			continue
		}
		ok = true
		for a := 0.0; a < 6.3; a += 0.8 {
			for _, r := range []float64{20, 46} {
				if !g.lvl.Free(Vec{p.X + math.Cos(a)*r, p.Y + math.Sin(a)*r*0.8}) {
					ok = false
				}
			}
		}
		spot = p
	}
	if !ok {
		return false
	}
	n := 4 + g.rng.Intn(2)
	if n > len(idle) {
		n = len(idle)
	}
	e.n = g.rng.Intn(2) // 0 football, 1 basketball
	dur := 26 + g.rng.Float64()*10
	for i := 0; i < n; i++ {
		gu := idle[g.rng.Intn(len(idle))]
		for _, o := range e.crew {
			if o == gu {
				gu = nil
				break
			}
		}
		if gu == nil {
			i--
			continue
		}
		a := float64(len(e.crew)) * 2 * math.Pi / float64(n)
		slot := g.lvl.NearestFree(Vec{spot.X + math.Cos(a)*40, spot.Y + math.Sin(a)*32})
		gu.path = g.lvl.PathG(gu.pos, slot)
		gu.state, gu.next, gu.dur, gu.fun = gGoto, gFun, dur, 15
		gu.hurry, gu.hurryT = true, -99
		e.crew = append(e.crew, gu)
	}
	e.x, e.y = spot.X, spot.Y
	e.sp = dur
	if e.n == 0 {
		g.say("Unos invitados se ponen a jugar al fútbol en plena fiesta")
	} else {
		g.say("Unos invitados juegan al baloncesto entre ellos")
	}
	return true
}

func (g *Game) gamePlayers(e *Event) []*Guest {
	var out []*Guest
	for _, gu := range e.crew {
		if !gu.out() && gu.fun == 15 && (gu.state == gFun || gu.state == gGoto && gu.next == gFun) {
			out = append(out, gu)
		}
	}
	return out
}

func (g *Game) stepSport(e *Event, dt float64) bool {
	pl := g.gamePlayers(e)
	if len(pl) < 2 {
		for _, gu := range e.crew {
			gu.rcv = false
		}
		return false
	}
	// everybody on the field: the ball starts moving
	ready := true
	for _, gu := range pl {
		if gu.state != gFun {
			ready = false
		}
	}
	if !ready && e.phase == 0 {
		return e.t < 20
	}
	if e.phase == 0 {
		e.phase, e.t = 1, 0
		e.boltT = 0
	}
	e.boltT -= dt
	if e.boltT <= 0 {
		// pass the ball to somebody else
		e.boltT = 1.1 + g.rng.Float64()*0.5
		var next *Guest
		for tries := 0; tries < 8 && next == nil; tries++ {
			if c := pl[g.rng.Intn(len(pl))]; c != e.target {
				next = c
			}
		}
		if next != nil {
			e.crewFrom, e.target = e.target, next
			if e.crewFrom == nil {
				e.crewFrom = pl[0]
			}
			e.sp2 = 0
			for _, gu := range pl {
				gu.rcv = gu == next
			}
			shout := []string{"¡PASA!", "¡AQUÍ!", "¡TOMA!", "¡VAMOS!", "¡YO, YO!"}
			if e.n == 1 {
				shout = []string{"¡CANASTA!", "¡TRIPLE!", "¡PASA!", "¡AQUÍ!"}
			}
			if g.rng.Float64() < 0.5 {
				e.crewFrom.icon, e.crewFrom.iconT = shout[g.rng.Intn(len(shout))], 1.1
			}
		}
	}
	if e.target != nil && e.crewFrom != nil {
		e.sp2 = math.Min(1, e.sp2+dt/0.8)
	}
	if e.t > e.sp-3 {
		for _, gu := range pl {
			gu.rcv = false
		}
	}
	return e.t < e.sp
}

func (g *Game) drawSport(dst *ebiten.Image, e *Event) {
	if e.phase == 0 || e.target == nil || e.crewFrom == nil {
		return
	}
	a, b := e.crewFrom.pos, e.target.pos
	t := e.sp2
	x := a.X + (b.X-a.X)*t
	y := a.Y + (b.Y-a.Y)*t - 16 - math.Sin(t*math.Pi)*18
	sx, sy := g.wx(x), g.wy(y)
	fillRect(dst, g.wx(a.X+(b.X-a.X)*t)-3, g.wy(a.Y+(b.Y-a.Y)*t)-1, 6, 2, color.RGBA{0, 0, 0, 70})
	col, dark := hx(0xf4f4f0), hx(0x202028)
	if e.n == 1 {
		col, dark = hx(0xe87820), hx(0x40200c)
	}
	filledEllipse(dst, sx, sy, 3.6, 3.6, dark)
	filledEllipse(dst, sx, sy, 3, 3, col)
	if e.n == 1 {
		fillRect(dst, sx-3, sy, 6, 1, dark)
		fillRect(dst, sx, sy-3, 1, 6, dark)
	} else {
		fillRect(dst, sx-1, sy-1, 2, 2, dark)
	}
}

// ---- costume groups ----------------------------------------------------------------

type themeKit struct {
	name, announce string
	make           func(g *Game, i int) CharSpec
}

func themedSpec(g *Game, female bool) CharSpec {
	spec := genericSpec(g.rng)
	spec.Female = female
	spec.Stature = 0
	spec.Hat, spec.Glasses, spec.Mask = HatNone, GlassNone, color.RGBA{}
	spec.Beard = BeardNone
	if female && spec.Hair2 == HairBald {
		spec.Hair2 = HairLong
	}
	return spec
}

var themeKits = []themeKit{
	{"halloween", "¡Llega un grupo disfrazado de Halloween!", func(g *Game, i int) CharSpec {
		s := themedSpec(g, i%2 == 0)
		switch i % 5 {
		case 0:
			s.Hat, s.HatCol = HatPumpkin, hx(0xf08020)
			s.Top, s.Shirt, s.Shirt2 = TopRobe, hx(0x20202a), hx(0x20202a)
		case 1:
			s.Hat, s.HatCol = HatWizard, hx(0x201830)
			s.Top, s.Shirt, s.Shirt2 = TopRobe, hx(0x40206a), hx(0x40206a)
		case 2:
			s.Mask, s.MaskKind = hx(0xf4f4f0), MaskGhost
			s.Top, s.Shirt, s.Shirt2 = TopRobe, hx(0xf4f4f0), hx(0xf4f4f0)
		case 3:
			s.Hat, s.HatCol = HatHorns, hx(0xc01818)
			s.Top, s.Shirt, s.Shirt2 = TopTee, hx(0xc01818), hx(0xc01818)
			s.Skin = hx(0xd04030)
		default:
			s.Mask, s.MaskKind = hx(0xe8e8e0), MaskSkull
			s.Top, s.Shirt, s.Shirt2 = TopTee, hx(0x14141a), hx(0xe8e8e0)
		}
		return s
	}},
	{"navidad", "¡Llega un grupo con gorros navideños, ho ho ho!", func(g *Game, i int) CharSpec {
		s := themedSpec(g, i%2 == 1)
		red, white := hx(0xc01818), hx(0xf4f4f4)
		switch i % 4 {
		case 0: // Santa
			s.Female = false
			s.Body, s.Beard, s.Hat, s.HatCol = BodyFat, BeardFull, HatBeanie, red
			s.Hair = white
			s.Top, s.Shirt, s.Shirt2 = TopTee, red, red
			s.Pants = hx(0xc01818)
		case 1: // reindeer
			s.Hat, s.HatCol = HatHorns, hx(0x8a5a2e)
			s.Top, s.Shirt, s.Shirt2 = TopHoodie, hx(0x7a4a24), hx(0x7a4a24)
		case 2: // elf
			s.Hat, s.HatCol = HatBeanie, hx(0x20a040)
			s.Top, s.Shirt, s.Shirt2 = TopTee, hx(0x20a040), hx(0xc01818)
			s.Stature = -1
		default:
			s.Hat, s.HatCol = HatBeanie, red
			s.Top, s.Shirt, s.Shirt2 = TopLong, white, red
		}
		return s
	}},
	{"verano", "¡Llega un grupo en bañador y gafas de sol! Es verano en la fiesta", func(g *Game, i int) CharSpec {
		s := themedSpec(g, i%2 == 0)
		s.Glasses = GlassShades
		if i%3 == 0 {
			s.Hat, s.HatCol = HatSombrero, hx(0xe8c070)
		}
		if s.Female {
			s.Top, s.Bottom = TopSwim, BotShorts
		} else {
			s.Top, s.Bottom = TopBare, BotTrunks
		}
		s.Shirt, s.Shirt2 = hx(pick(g.rng, []uint32{0xf04880, 0x20b0e0, 0xf0c020, 0x30d070})), hx(0xf4f4f4)
		s.Pants = hx(pick(g.rng, []uint32{0xf04880, 0x20b0e0, 0xf0c020, 0x30d070}))
		return s
	}},
}

func (g *Game) startTheme(e *Event) bool {
	if g.guestRoom() < 3 {
		return false
	}
	kit := themeKits[g.rng.Intn(len(themeKits))]
	e.n = 0
	for i, k := range themeKits {
		if k.name == kit.name {
			e.n = i
		}
	}
	count := minI(5+g.rng.Intn(3), g.guestRoom())
	st := g.lvl.Def.Start
	for i := 0; i < count; i++ {
		spec := kit.make(g, i)
		cs := newCharSet(&spec)
		pos := g.lvl.NearestFree(Vec{st.X + float64(i)*14 - 40, st.Y})
		gu := &Guest{pos: pos, dir: DirDown, cs: cs, style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: 0.5, role: g.rng.Intn(2), keep: true}
		g.guests = append(g.guests, gu)
		g.total++
		e.crew = append(e.crew, gu)
	}
	g.say(kit.announce)
	g.snd.Play("alert")
	if kit.name == "verano" {
		e.phase = 1
	}
	return true
}

func (g *Game) stepTheme(e *Event, dt float64) bool {
	// the summer crowd heads for the water, or lies down in the sun
	if e.phase == 1 && e.t > 1.5 {
		e.phase = 2
		pools := g.lvl.Pools
		var sun Vec
		sunOK := false
		for try := 0; try < 40 && !sunOK; try++ {
			p := g.lvl.RandomRoam(g.rng)
			sunOK = g.lvl.Outdoor(p) && !g.lvl.InPool(p) && !g.lvl.badGround(p)
			sun = p
		}
		for i, gu := range e.crew {
			if gu.out() {
				continue
			}
			if len(pools) > 0 && g.lvl.Def.Swim && i%3 != 0 {
				t := g.lvl.RandomIn(g.rng, pools[g.rng.Intn(len(pools))].Grow(-8))
				gu.path = g.lvl.PathG(gu.pos, t)
				gu.state, gu.next, gu.dur = gGoto, gSwim, 14+g.rng.Float64()*10
			} else if sunOK {
				t := g.lvl.NearestFree(Vec{sun.X + float64(i%4)*24 - 36, sun.Y + float64(i/4)*18})
				gu.path = g.lvl.PathG(gu.pos, t)
				gu.state, gu.next, gu.dur, gu.fun = gGoto, gFun, 20+g.rng.Float64()*10, 16
				gu.icon, gu.iconT = []string{"¡Qué calorcito!", "A broncearse", "Protector, ¿quién?"}[g.rng.Intn(3)], 2.5
			}
		}
	}
	return e.t < 3
}
