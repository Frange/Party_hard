package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// AMATEUR SUPERHERO (idea 294, approved by the user: "sí").
//
// Now and then a guest in a home-made costume (a bed sheet tied as a cape, a mask or swimming goggles, his underpants
// over his trousers and a fried-egg emblem of his own on the chest: a generic hero of nobody's comic) shouts "¡Voy a
// salvar a alguien!" and goes for a feat:
//   - WATER: if the level has a pool near, he runs up and jumps into it (a belly flop or a clean dive);
//   - JUMP: else, over a small thing of the level (a bench, a chair, a stool, a crate, a barrel, a planter...): run-up,
//     flight pose with the cape flapping, and on the other side either a pose (success) or a crash with stars (50 %),
//     getting up with his cape torn;
//   - FLY: with nothing near, he just "flies" a few steps with his fists ahead and lands (or crashes) on open floor.
// Nothing new is solid and nothing is added: the hero is a plain guest all the time (pevMem, like every crowd event):
// he can be killed anywhere (his cape stays on the floor), he is a witness, he counts for every cap and goal. Every
// phase has its own time limit; with an alarm or the police near it ends at once.
//
// A reactive, Minor party event "heroe" (it never takes one of the places of the frame, like the bull and the petting):
// one party in about three (heroChance, decided when the party starts) gets a hero at a moment of heroAtMin..Max, and
// rarely a second one after a long pause. Dice: heroDice() (never g.rng nor the frame's dice).
// Debug: -pevent heroe[:agua|:salto|:vuelo][-cae|-posa] forces it 2 s into the party near the first player, with a
// slow flight for the captures (progress frozen like every -pevent).
// ---------------------------------------------------------------------------

const (
	heroChance    = 0.34  // a party gets a hero
	heroAgainP    = 0.25  // and, after the first, a second one later
	heroAtMin     = 60.0  // seconds of the party
	heroAtVar     = 200.0 // ... up to heroAtMin+heroAtVar
	heroAgainMin  = 200.0
	heroAgainVar  = 100.0
	heroLookGap   = 3.0
	heroCrashP    = 0.5
	heroWalkMax   = 9.0 // seconds to reach the run-up spot, else he gives up
	heroShoutT    = 1.1
	heroRunSpeed  = 78.0
	heroFlyT      = 0.75
	heroFlyTDemo  = 2.6
	heroLieT      = 2.0
	heroUpT       = 1.4
	heroPoseT     = 2.0
	heroSwimT     = 2.2
	heroRunUp     = 28.0 // pixels of run-up before the jump
	heroWaterR    = 240.0
	heroJumpR     = 200.0
	heroPlayerGap = 60.0 // he never starts this near a player
	heroLift      = 16.0 // the top of the jump arc (sprite pixels)
)

const (
	hkWater = iota
	hkJump
	hkFly
)

const (
	hpWalk = iota
	hpShout
	hpDash
	hpFly
	hpLie
	hpUp
	hpPose
	hpSwim
)

var (
	heroGoL    = pevTexts("¡Voy a salvar a alguien!", "¡Esto es trabajo para mí!", "¡Tranquilos, ya llego yo!", "¡Al rescate!")
	heroFlyL   = pevTexts("¡Puedo volaaar!", "¡Más alto, más lejos!", "¡Allá voooy!")
	heroCrashL = pevTexts("¡Estoy bien! ¡ESTOY BIEN!", "Mi capa... mi preciosa capa", "Mañana vuelo mejor", "El suelo me ha atacado")
	heroPoseL  = pevTexts("¡Misión cumplida!", "¡De nada, ciudadanos!", "¡Otra vida salvada!")
	heroWaterL = pevTexts("¡PLAF!", "¿Alguien se ahogaba? ¿No?", "¡El agua está a salvo!")
	heroSeenL  = pevTexts("¿Ese va en calzoncillos?", "¿Eso es una sábana?", "¡Qué valiente... o qué loco!")
	heroAbortL = pevTexts("¡Uy! Hoy no vuelo")
)

// heroHop: the small things of the levels he may jump over (only their foot is solid; nothing is added).
var heroHop = map[string]bool{"bench": true, "seatbench": true, "rinkbench": true, "seatlong": true, "chair": true,
	"seatchair": true, "stool": true, "crate": true, "barrel": true, "pbarrel": true, "haystack": true, "planter": true,
	"seatpouf": true, "rail": true}

var (
	heroRng     *rand.Rand
	heroSeed    int64 // tests: non-zero = deterministic dice
	heroInTests bool  // tests: let it happen under go test (off by default so the other tests' parties stay as they were)
)

func heroDice() *rand.Rand {
	if heroRng == nil {
		sd := heroSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x4e2015
		}
		heroRng = rand.New(rand.NewSource(sd))
	}
	return heroRng
}

// heroLook: the home-made costume.
type heroLook struct {
	cape, capeDk, trim color.RGBA
	briefs, mask       color.RGBA
	pat                int  // 0 plain sheet, 1 polka dots, 2 stripes
	goggles            bool // swimming goggles instead of a mask
}

var heroCapes = []color.RGBA{rgb(0x8a4ae0), rgb(0x2aa070), rgb(0xe07a2a), rgb(0x2a9ad0), rgb(0xd84a90), rgb(0xc8a020)}
var heroBriefs = []color.RGBA{rgb(0xf4f0e6), rgb(0x60c8f0), rgb(0xf06a6a), rgb(0x80d860), rgb(0xf8d040)}

func heroLookFrom(r *rand.Rand) heroLook {
	c := heroCapes[r.Intn(len(heroCapes))]
	l := heroLook{cape: c, capeDk: shade(c, 0.68), trim: rgb(0xf8f4ea), pat: r.Intn(3), goggles: r.Intn(3) == 0}
	l.briefs = heroBriefs[r.Intn(len(heroBriefs))]
	l.mask = rgb(0x1a1620)
	if r.Intn(2) == 0 {
		l.mask = shade(c, 0.5)
	}
	return l
}

// heroPlan: the feat: run from a to l, fly to land.
type heroPlan struct {
	kind    int
	a, l    Vec
	land    Vec
	f       float64 // +1 to the right, -1 to the left
	prop    *Prop
	crash   bool
	nearest float64
}

type heroLevel struct {
	lvl   *Level
	want  bool
	next  float64
	look  float64
	runs  int
	looks int
	gu    *Guest // the one chosen by the last look (Start takes him)
	plan  heroPlan
	capes []heroCape
	n     [3]int // feats by kind (tests)
	ends  int
}

type heroCape struct {
	pos  Vec
	look heroLook
}

var heroLv heroLevel

type heroState struct {
	m     pevMem
	plan  heroPlan
	look  heroLook
	phase int
	t     float64
	from  Vec // where the flight starts
	torn  bool
	said  bool
	flown bool // for the tests
}

func (s *heroState) pevHas(gu *Guest) bool { return s.m.gu == gu }

func init() {
	registerPartyEvent(&partyEvent{ID: "heroe", Name: "Superhéroe aficionado", Group: "heroe", Reactive: true, Minor: true,
		Prob: heroChance, AtMin: heroAtMin, AtMax: heroAtMin + heroAtVar, Dur: 15,
		Eligible: heroDue, Start: heroStart,
		Update: pevGuard(heroUpdate, func(g *Game, a *partyAct) { heroEnd(g, a.D.(*heroState)) })})
}

// heroDemoKind: -pevent heroe:agua|salto|vuelo (-1 = any) and -cae / -posa.
func heroDemoKind() (kind int, crash int) {
	d := pevDemo("heroe")
	kind, crash = -1, -1
	for _, w := range []struct {
		s string
		k int
	}{{"agua", hkWater}, {"water", hkWater}, {"salto", hkJump}, {"jump", hkJump}, {"vuelo", hkFly}, {"fly", hkFly}} {
		if len(d) >= len(w.s) && d[:len(w.s)] == w.s {
			kind = w.k
		}
	}
	switch {
	case len(d) >= 3 && (d[len(d)-3:] == "cae"):
		crash = 1
	case len(d) >= 4 && (d[len(d)-4:] == "posa"):
		crash = 0
	}
	return
}

func heroReset(g *Game) {
	r := heroDice()
	roll, at := r.Float64(), r.Float64()
	heroLv = heroLevel{lvl: g.lvl, want: roll < heroChance, next: heroAtMin + at*heroAtVar}
	if pevDebugOn("heroe") {
		heroLv.want, heroLv.next = true, g.time+2
	}
}

// heroDue: the condition of the reactive event (the party has a hero, his moment has come, somebody can do it).
func heroDue(g *Game, t partyTraits) bool {
	if g.lvl == nil || len(g.lvl.Def.Roam) == 0 {
		return false
	}
	if testing.Testing() && !heroInTests && !pevDebugOn("heroe") {
		return false
	}
	if heroLv.lvl != g.lvl {
		heroReset(g)
	}
	if !heroLv.want || g.time < heroLv.next || g.time < heroLv.look {
		return false
	}
	heroLv.look = g.time + heroLookGap
	heroLv.looks++
	gu, p, ok := g.heroPick(heroDice())
	if !ok {
		return false
	}
	heroLv.gu, heroLv.plan = gu, p
	return true
}

// heroCanBe: a guest who may put on the costume now.
func (g *Game) heroCanBe(gu *Guest) bool {
	return g.pevFreeGuest(gu) && !isKid(gu) && !g.lvl.InPool(gu.pos) && gu.icon == "" && g.pevPlayerFar(gu.pos, heroPlayerGap)
}

// heroPick: a free guest and a feat for him (a few guests are tried).
func (g *Game) heroPick(r *rand.Rand) (*Guest, heroPlan, bool) {
	var cand []*Guest
	for _, gu := range g.guests {
		if g.heroCanBe(gu) {
			cand = append(cand, gu)
		}
	}
	if len(cand) == 0 {
		return nil, heroPlan{}, false
	}
	want, _ := heroDemoKind()
	if pevDebugOn("heroe") { // the captures: the ones near the first player first
		pp := g.p.pos
		for i := 1; i < len(cand); i++ {
			for j := i; j > 0 && dist(cand[j].pos, pp) < dist(cand[j-1].pos, pp); j-- {
				cand[j], cand[j-1] = cand[j-1], cand[j]
			}
		}
	} else {
		r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	}
	for i, gu := range cand {
		if i >= 10 {
			break
		}
		if p, ok := g.heroPlanFor(gu, r, want); ok {
			return gu, p, true
		}
	}
	return nil, heroPlan{}, false
}

// heroGround: a point where a guest may stand (and land): walkable, not solid, not on a road; water as asked.
func (g *Game) heroGround(p Vec, water bool) bool {
	l := g.lvl
	if p.X < 8 || p.Y < 8 || p.X > l.Def.W-8 || p.Y > l.Def.H-8 {
		return false
	}
	return l.Free(p) && !l.SolidP(p.X, p.Y) && !l.badGround(p) && l.InPool(p) == water
}

// heroLineClear: every few pixels of the run-up from a to b is dry walkable ground.
func (g *Game) heroLineClear(a, b Vec) bool {
	n := int(math.Ceil(dist(a, b)/3)) + 1
	for i := 0; i <= n; i++ {
		u := float64(i) / float64(n)
		if !g.heroGround(Vec{a.X + (b.X-a.X)*u, a.Y + (b.Y-a.Y)*u}, false) {
			return false
		}
	}
	return true
}

func (g *Game) heroNoPlayer(ps ...Vec) bool {
	for _, p := range ps {
		if !g.pevPlayerFar(p, 22) {
			return false
		}
	}
	return true
}

// heroPlans: every feat that fits around the guest (water, jumps over small things, a plain flight).
func (g *Game) heroPlans(gu *Guest) []heroPlan {
	var out []heroPlan
	l := g.lvl
	rnd := func(p Vec) Vec { return Vec{math.Round(p.X), math.Round(p.Y)} }
	for _, pr := range l.Pools {
		for _, f := range []float64{1, -1} {
			for _, k := range []float64{0.5, 0.3, 0.7} {
				y := pr.Y + pr.H*k
				var lp, land Vec
				if f > 0 {
					lp, land = Vec{pr.X - 7, y}, Vec{pr.X + 20, y}
				} else {
					lp, land = Vec{pr.X + pr.W + 7, y}, Vec{pr.X + pr.W - 20, y}
				}
				a := Vec{lp.X - f*heroRunUp, y}
				a, lp, land = rnd(a), rnd(lp), rnd(land)
				if dist(gu.pos, a) > heroWaterR || !g.heroLineClear(a, lp) || !g.heroGround(land, true) {
					continue
				}
				out = append(out, heroPlan{kind: hkWater, a: a, l: lp, land: land, f: f, nearest: dist(gu.pos, a)})
				break
			}
		}
	}
	for _, pr := range l.Props {
		if pr.gone || !heroHop[pr.Name] || pr.Foot.W <= 0 || pr.Foot.W > 48 || pr.Foot.H > 22 {
			continue
		}
		fy := math.Round(pr.Foot.Y + pr.Foot.H/2)
		for _, f := range []float64{1, -1} {
			var lp, land Vec
			if f > 0 {
				lp, land = Vec{pr.Foot.X - 7, fy}, Vec{pr.Foot.X + pr.Foot.W + 9, fy}
			} else {
				lp, land = Vec{pr.Foot.X + pr.Foot.W + 7, fy}, Vec{pr.Foot.X - 9, fy}
			}
			a := Vec{lp.X - f*heroRunUp, fy}
			a, lp, land = rnd(a), rnd(lp), rnd(land)
			if dist(gu.pos, a) > heroJumpR || !g.heroLineClear(a, lp) || !g.heroGround(land, false) {
				continue
			}
			out = append(out, heroPlan{kind: hkJump, a: a, l: lp, land: land, f: f, prop: pr, nearest: dist(gu.pos, a)})
		}
	}
	for _, f := range []float64{1, -1} {
		a := rnd(gu.pos)
		lp, land := Vec{a.X + f*30, a.Y}, Vec{a.X + f*58, a.Y}
		if g.heroLineClear(a, land) {
			out = append(out, heroPlan{kind: hkFly, a: a, l: lp, land: land, f: f})
		}
	}
	return out
}

// heroPlanFor: the feat of this guest (water or a jump if any is near, else a plain flight); want >= 0 forces a kind.
func (g *Game) heroPlanFor(gu *Guest, r *rand.Rand, want int) (heroPlan, bool) {
	all := g.heroPlans(gu)
	var by [3][]heroPlan
	for _, p := range all {
		by[p.kind] = append(by[p.kind], p)
	}
	order := []int{hkWater, hkJump, hkFly}
	if len(by[hkWater]) > 0 && len(by[hkJump]) > 0 && r.Intn(2) == 0 {
		order = []int{hkJump, hkWater, hkFly}
	}
	if want >= 0 {
		order = []int{want}
	}
	for _, k := range order {
		ps := by[k]
		// the nearest few, then reachable
		for i := 1; i < len(ps); i++ {
			for j := i; j > 0 && ps[j].nearest < ps[j-1].nearest; j-- {
				ps[j], ps[j-1] = ps[j-1], ps[j]
			}
		}
		for i, p := range ps {
			if i >= 4 {
				break
			}
			if !g.heroNoPlayer(p.a, p.l, p.land) {
				continue
			}
			if dist(gu.pos, p.a) > 6 && g.lvl.PathG(gu.pos, p.a) == nil {
				continue
			}
			if g.lvl.PathG(p.land, p.a) == nil { // he must be able to walk away from where he lands
				continue
			}
			p.crash = r.Float64() < heroCrashP
			return p, true
		}
	}
	return heroPlan{}, false
}

func heroStart(g *Game, a *partyAct) bool {
	r := heroDice()
	gu, p := heroLv.gu, heroLv.plan
	heroLv.gu = nil
	heroLv.next = g.time + 4 // if it cannot start now, look again in a while
	if gu == nil || !g.heroCanBe(gu) {
		return false
	}
	if _, c := heroDemoKind(); c >= 0 {
		p.crash = c == 1
	}
	s := &heroState{plan: p, look: heroLookFrom(r)}
	s.m.gu = gu
	if !g.pevSendTo(&s.m, p.a) {
		return false
	}
	gu.icon, gu.iconT = pevPick(r, heroGoL), 2.2
	a.D = s
	heroLv.n[p.kind]++
	if pevDebugOn("heroe") {
		fmt.Printf("PEVDEMO heroe kind %d crash %v from %.0f %.0f run %.0f %.0f land %.0f %.0f\n", p.kind, p.crash, gu.pos.X, gu.pos.Y, p.a.X, p.a.Y, p.land.X, p.land.Y)
	}
	return true
}

func heroEnd(g *Game, s *heroState) {
	if s == nil {
		return
	}
	r := heroDice()
	gu := s.m.gu
	if gu != nil && gu.dead { // his cape stays where he fell
		heroLv.capes = append(heroLv.capes, heroCape{pos: Vec{math.Round(gu.pos.X), math.Round(gu.pos.Y)}, look: s.look})
	}
	if s.m.mine() {
		pevRelease(&s.m, r)
	} else {
		pevForget(&s.m)
	}
	heroLv.ends++
	heroLv.runs++
	if heroLv.runs == 1 && r.Float64() < heroAgainP {
		heroLv.next = g.time + heroAgainMin + r.Float64()*heroAgainVar
	} else {
		heroLv.want = false
	}
	if pevDebugOn("heroe") {
		heroLv.want, heroLv.next = true, g.time+6
	}
}

func (s *heroState) flyT() float64 {
	if pevDebugOn("heroe") {
		return heroFlyTDemo
	}
	return heroFlyT
}

func heroUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*heroState)
	r := heroDice()
	gu := s.m.gu
	if !s.m.mine() || g.pevAlarmNear(gu.pos, 110) {
		heroEnd(g, s)
		return true
	}
	s.t += dt
	next := func(ph int) { s.phase, s.t = ph, 0 }
	p := &s.plan
	switch s.phase {
	case hpWalk:
		if gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, p.a) < 6 {
			s.m.hold(gIdle)
			gu.pos = p.a
			gu.dir = DirRight
			if p.f < 0 {
				gu.dir = DirLeft
			}
			next(hpShout)
		} else if s.t > heroWalkMax && !(pevDebugOn("heroe") && s.t < 40) {
			heroEnd(g, s)
			return true
		}
	case hpShout:
		s.m.keep()
		if s.t >= heroShoutT {
			next(hpDash)
		}
	case hpDash:
		s.m.keep()
		d := dist(p.a, p.l)
		u := math.Min(1, s.t*heroRunSpeed/math.Max(d, 1))
		gu.pos = Vec{math.Round(p.a.X + (p.l.X-p.a.X)*u), math.Round(p.a.Y + (p.l.Y-p.a.Y)*u)}
		if u >= 1 {
			if !g.heroNoPlayer(p.land) { // a player stands where he would land: he brakes
				gu.icon, gu.iconT = heroAbortL[0], 1.6
				heroEnd(g, s)
				return true
			}
			s.from = p.l
			gu.pos = p.land // he is already there for the game (drawn flying from s.from)
			s.flown = true
			if gu.icon == "" || gu.iconT < 0.6 {
				gu.icon, gu.iconT = pevPick(r, heroFlyL), 1.6
			}
			g.heroWitnessLine(gu, r)
			next(hpFly)
		}
	case hpFly:
		s.m.keep()
		if s.t >= s.flyT() {
			switch {
			case p.kind == hkWater:
				g.burst(Vec{gu.pos.X, gu.pos.Y - 4}, 14, 60, rgb(0xbfe4ff), 0.5, 2, 120)
				if g.heroHeard(gu) {
					g.snd.Play("splash")
				}
				if p.crash {
					gu.icon, gu.iconT = heroWaterL[0], 1.4
				}
				next(hpSwim)
			case p.crash:
				s.torn = true
				if g.heroHeard(gu) {
					g.snd.Play("thud")
				}
				next(hpLie)
			default:
				s.m.pose(AnimWave)
				gu.dir = DirDown
				gu.icon, gu.iconT = pevPick(r, heroPoseL), 1.8
				next(hpPose)
			}
		}
	case hpLie:
		s.m.keep()
		if s.t >= heroLieT {
			s.m.pose(AnimLaugh)
			gu.dir = DirDown
			gu.icon, gu.iconT = pevPick(r, heroCrashL), 2
			next(hpUp)
		}
	case hpUp, hpPose:
		s.m.keep()
		lim := heroUpT
		if s.phase == hpPose {
			lim = heroPoseT
		}
		if s.t >= lim {
			heroEnd(g, s)
			return true
		}
	case hpSwim:
		s.m.keep()
		if !s.said && s.t > 0.9 {
			s.said = true
			if gu.icon == "" || gu.iconT < 0.3 {
				gu.icon, gu.iconT = pevPick(r, heroWaterL[1:]), 1.8
			}
		}
		if s.t >= heroSwimT {
			heroEnd(g, s)
			return true
		}
	}
	return false
}

// heroHeard: the sounds of the feat only near a player.
func (g *Game) heroHeard(gu *Guest) bool { return g.snd != nil && !g.pevPlayerFar(gu.pos, 220) }

// heroWitnessLine: half of the time somebody near says what everybody thinks.
func (g *Game) heroWitnessLine(hero *Guest, r *rand.Rand) {
	if r.Intn(2) == 0 {
		return
	}
	for _, gu := range g.guests {
		if gu == hero || gu.out() || gu.icon != "" || !lifeFree(gu) || dist(gu.pos, hero.pos) > 90 {
			continue
		}
		gu.icon, gu.iconT = pevPick(r, heroSeenL), 1.8
		return
	}
}

// ---- drawing -------------------------------------------------------------------------------------------------

// drawGuest: the frame's hook in the guests' draw loop (pe_fx.go): the costume always, the flight, the crash, the swim.
func (s *heroState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	if s.m.gu != gu || !s.m.mine() {
		return false
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	sc := g.lifeScale()
	switch s.phase {
	case hpFly:
		u := math.Min(1, s.t/s.flyT())
		at := Vec{s.from.X + (s.plan.land.X-s.from.X)*u, s.from.Y + (s.plan.land.Y-s.from.Y)*u}
		lift := heroLift * 4 * u * (1 - u)
		if s.plan.kind == hkFly {
			lift = heroLift*0.6*math.Min(1, u*4) - heroLift*0.6*math.Max(0, u-0.75)*4
		}
		g.heroDrawFlight(dst, gu, s, at, lift*sc)
		return true
	case hpLie:
		p := Vec{math.Round(gu.pos.X), math.Round(gu.pos.Y)}
		cx, cy := g.pevXY(p)
		heroCapeFloorPix(plot, cx-int(s.plan.f)*k(16), cy-px, px, s.look, true)
		g.drawLying(dst, gu.cs, false, p)
		drawStarsPix(plot, cx, cy-k(12), px, g.time)
		return true
	case hpSwim:
		cx, cy := g.pevXY(gu.pos)
		heroCapeFloorPix(plot, cx-int(s.plan.f)*k(10), cy-k(6), px, s.look, false)
		g.drawActor(dst, gu.cs, AnimPanic, g.time*6, gu.pos, DirDown, 0, true)
		return true
	}
	anim, phase := g.guestPose(gu)
	dir := g.gaitDir(gu)
	moving := gu.moving
	switch s.phase {
	case hpShout:
		anim, phase, moving = AnimPoint, s.t*4, false
		dir = DirRight
		if s.plan.f < 0 {
			dir = DirLeft
		}
	case hpDash:
		anim, phase, moving = AnimRun, s.t*10, true
		dir = DirRight
		if s.plan.f < 0 {
			dir = DirLeft
		}
	}
	cx, cy := g.pevXY(pos)
	heroCapeBehind(plot, cx, cy, px, k, dir, moving, g.time+gu.off, s.look, s.torn)
	g.drawActor(dst, gu.cs, anim, phase, pos, dir, 0, false)
	heroCostumeFront(plot, cx, cy, px, k, dir, moving, g.time+gu.off, s.look, s.torn)
	return true
}

// heroDrawFlight: the hero in the air, flat, fists ahead, the cape flapping behind his feet; his shadow on the floor.
func (g *Game) heroDrawFlight(dst *ebiten.Image, gu *Guest, s *heroState, at Vec, lift float64) {
	im := gu.cs.lying(false)
	w, h := im.Bounds().Dx(), im.Bounds().Dy()
	cx, cy := g.pevXY(at)
	top := cy - int(math.Round(lift)) - h + 2
	fillShadow(dst, float64(cx), float64(cy), g.lifeScale())
	plot := g.plotter(dst)
	f := int(s.plan.f)
	headX := cx + f*(w/2) // the head is at the end of the flight direction
	midY := top + h/2
	heroCapeFlightPix(plot, cx, midY, w, f, g.pevPx(), g.time, s.look)
	op := &ebiten.DrawImageOptions{}
	if f < 0 { // the lying frame has the head on the right: mirrored to fly left
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(float64(w), 0)
	}
	op.GeoM.Translate(float64(cx-w/2), float64(top))
	dst.DrawImage(im, op)
	// the fists ahead, past the head
	skin := rgb(0xf0c8a0)
	if gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Skin.A != 0 {
		skin = gu.cs.Spec.Skin
	}
	fx := headX + f*1
	if f < 0 {
		fx -= 3
	}
	plot(fx, midY-3, 3, 2, skin)
	plot(fx, midY+1, 3, 2, skin)
	plot(fx+f, midY-3, 1, 1, shade(skin, 0.8))
	plot(fx+f, midY+1, 1, 1, shade(skin, 0.8))
}

// heroCapePanel: one column of the sheet (pattern: plain, dots, stripes).
func heroCapeCol(l heroLook, i, j int) color.RGBA {
	switch l.pat {
	case 1:
		if (i+2*j)%5 == 0 && j > 0 {
			return l.trim
		}
	case 2:
		if (i/2)%2 == 1 {
			return shade(l.cape, 0.82)
		}
	}
	return l.cape
}

// heroCapeBehind: the sheet behind the body (drawn before the sprite): a wide flap seen from the front, a trailing
// flap seen from the side (it flaps while he runs). Seen from the back it is drawn after the sprite (heroCostumeFront).
func heroCapeBehind(plot plotFn, cx, cy, px int, k func(float64) int, dir Dir, moving bool, t float64, l heroLook, torn bool) {
	sy, by := cy-k(27), cy-k(8)
	if torn {
		by = cy - k(14)
	}
	switch dir {
	case DirDown:
		for i := -6; i <= 5; i++ {
			bot := by
			if moving && (i+int(t*8))%3 == 0 {
				bot -= px
			}
			if torn && (i*7)%3 == 0 {
				bot -= 2 * px
			}
			for j, y := 0, sy; y < bot; j, y = j+1, y+px {
				plot(cx+i*px, y, px, px, heroCapeCol(l, i+6, j))
			}
		}
	case DirLeft, DirRight:
		f := 1
		if dir == DirLeft {
			f = -1
		}
		L := 4
		if moving {
			L = 7 + int(math.Round(math.Sin(t*14)))
		}
		if torn {
			L -= 2
		}
		for i := 0; i < L; i++ {
			x := cx - f*(2+i)*px
			top := sy + (i*px)/2
			bot := by - i*px/2
			if moving {
				bot = by - i*px - int(math.Round(math.Sin(t*14+float64(i))*float64(px)))
				top = sy + i*px/3
			}
			for j, y := 0, top; y < bot; j, y = j+1, y+px {
				plot(x, y, px, px, heroCapeCol(l, i, j))
			}
		}
	}
}

// heroCostumeFront: what goes over the sprite: the cape from the back, the briefs over the trousers, the mask or the
// goggles, the emblem on the chest (a fried egg: nobody's logo).
func heroCostumeFront(plot plotFn, cx, cy, px int, k func(float64) int, dir Dir, moving bool, t float64, l heroLook, torn bool) {
	ey := cy - k(28)
	if dir == DirUp {
		by := cy - k(8)
		if torn {
			by = cy - k(14)
		}
		for i := -5; i <= 4; i++ {
			bot := by
			if moving && (i+int(t*8))%3 == 0 {
				bot -= px
			}
			if torn && (i*7)%3 == 0 {
				bot -= 2 * px
			}
			for j, y := 0, cy-k(27); y < bot; j, y = j+1, y+px {
				plot(cx+i*px, y, px, px, heroCapeCol(l, i+5, j))
			}
		}
		plot(cx-4*px, ey, 9*px, px, l.mask) // the strap of the mask
		return
	}
	// briefs over the trousers
	bw, bx := 9, cx-4*px
	if dir == DirLeft || dir == DirRight {
		bw, bx = 6, cx-3*px
	}
	plot(bx, cy-k(16), bw*px, 2*px, l.briefs)
	plot(bx, cy-k(16), bw*px, px, shade(l.briefs, 0.85))
	if dir == DirDown {
		// the knot of the cape at the neck and the emblem
		plot(cx-2*px, cy-k(25), px, px, l.capeDk)
		plot(cx+px, cy-k(25), px, px, l.capeDk)
		plot(cx-px, cy-k(22), 3*px, 3*px, rgb(0xf8f4ea))
		plot(cx, cy-k(22)+px, px, px, rgb(0xf4b830))
		if l.goggles {
			plot(cx-4*px, ey, 9*px, px, rgb(0x2a2a3a))
			plot(cx-3*px, ey-px, 2*px, 2*px, rgb(0x60c8f0))
			plot(cx+px, ey-px, 2*px, 2*px, rgb(0x60c8f0))
		} else {
			plot(cx-4*px, ey-px, 9*px, 2*px, l.mask)
			plot(cx-2*px, ey-px, px, px, rgb(0xf8f8f8))
			plot(cx+2*px, ey-px, px, px, rgb(0xf8f8f8))
		}
		return
	}
	f := 1
	if dir == DirLeft {
		f = -1
	}
	x0 := cx
	if f < 0 {
		x0 = cx - 4*px
	}
	if l.goggles {
		plot(x0, ey, 5*px, px, rgb(0x2a2a3a))
		plot(cx+f*2*px, ey-px, px, 2*px, rgb(0x60c8f0))
	} else {
		plot(x0, ey-px, 5*px, 2*px, l.mask)
		plot(cx+f*2*px, ey-px, px, px, rgb(0xf8f8f8))
	}
}

// heroCapeFlightPix: in the air the sheet streams behind his feet and flaps (cx, midY: middle of the flying body).
func heroCapeFlightPix(plot plotFn, cx, midY, w, f, px int, t float64, l heroLook) {
	start := cx + f*(w/2-12) // from his shoulders
	n := w/2 + 6
	for i := 0; i < n; i++ {
		x := start - f*i
		half := 5 + i/8
		wave := int(math.Round(math.Sin(t*16-float64(i)*0.5) * (1 + float64(i)/10)))
		for j := -half; j <= half; j++ {
			plot(x, midY+j+wave, 1, 1, heroCapeCol(l, i, j+half))
		}
	}
	_ = px
}

// heroCapeFloorPix: the cape lying on the floor (or floating on the water), torn at the end.
func heroCapeFloorPix(plot plotFn, cx, cy, px int, l heroLook, torn bool) {
	for i := -5; i <= 5; i++ {
		h := 4
		if torn && (i*5)%3 == 0 {
			h = 2
		}
		for j := 0; j < h; j++ {
			plot(cx+i*px, cy-(j+1)*px, px, px, heroCapeCol(l, i+5, j))
		}
	}
	plot(cx-6*px, cy-2*px, px, px, l.capeDk)
	plot(cx+6*px, cy-3*px, px, px, l.capeDk)
}

// heroItems: the capes left on the floor by the heroes who were killed (draw.go adds them to the y-sorted list).
func (g *Game) heroItems(dst *ebiten.Image, items []drawItem) []drawItem {
	if heroLv.lvl != g.lvl || len(heroLv.capes) == 0 {
		return items
	}
	plot := g.plotter(dst)
	px := g.pevPx()
	for _, c := range heroLv.capes {
		c := c
		items = append(items, drawItem{c.pos.Y - 5, func() {
			cx, cy := g.pevXY(c.pos)
			heroCapeFloorPix(plot, cx+3*px, cy+px, px, c.look, true)
		}})
	}
	return items
}
