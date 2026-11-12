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
// THE BALL THROWER (idea 286, approved by the user): in a park, a garden or a square with a dog, now and then a guest
// throws a ball (a small sprite that flies on a curve with its shadow and bounces), the dog runs after it, picks it
// up and brings it back; 2-4 throws (20-40 s).
//
// A reactive, Minor party event "pelota" (never one of the places of the frame), with its OWN dice per party:
// fetchChance of the parties of the venues in fetchIcons, at a moment of fetchAtMin..Max, once. The dog is one of
// g.animals (held with pevAnimalHold, pe_petting.go: nothing is added, the caps of capAnimals stay as they are); the
// thrower is a plain guest (pevMem): he can be killed, he is a witness and he counts for every goal.
//   - the thrower is killed: if the dog has the ball (or is bringing it), it goes to the body and waits beside it with
//     the ball in its mouth for a while (fetchMourn), whining; then it goes back to its own life.
//   - the dog is killed (killAnimal): the event ends, the thrower goes back to the party.
//   - an alarm near, the thrower frightened: it ends at once. Every phase has its own time limit.
// Dice: fetchDice() (never g.rng). -pevent pelota (or pelota:muerte: the thrower drops dead after the first throw, to
// see the dog waiting) forces it in the level given (progress frozen).
// ---------------------------------------------------------------------------

const (
	fetchChance   = 0.12
	fetchAtMin    = 40.0
	fetchAtMax    = 240.0
	fetchMaxT     = 40.0 // the throws stop after this long
	fetchWalkMax  = 15.0
	fetchChaseMax = 12.0
	fetchRun      = 64.0 // the dog's speed after the ball
	fetchTrot     = 46.0 // the dog's speed back with it
	fetchFlyMin   = 0.9
	fetchFlyVar   = 0.3
	fetchArc      = 26.0 // px of the highest point of the throw
	fetchMourn    = 10.0
	fetchHandH    = 14.0 // the ball leaves the hand this high
)

// fetchIcons: the venues with a park, a garden or a square.
var fetchIcons = map[string]bool{"house": true, "eiffel": true, "festival": true, "cake": true, "italy": true, "skate": true, "funfair": true}

var (
	fetchThrow  = pevTexts("¡Busca!", "¡Ve a por ella!", "¡Tráela!", "¡Allá va!")
	fetchGood   = pevTexts("¡Buen chico!", "¡Muy bien!", "¡Qué rápido!", "¡Otra vez!")
	fetchDogL   = pevTexts("¡Guau!", "¡Guau, guau!")
	fetchMournL = pevTexts("*gime*", "*espera con la pelota*")
	fetchCall   = pevTexts("¿Jugamos a la pelota?", "¡Mira lo que tengo!")
)

var (
	fetchRng     *rand.Rand
	fetchSeed    int64
	fetchInTests bool // tests: roll the real dice under go test (off by default)
	fetchForce   bool // tests: the event comes for sure in a venue that fits
)

func fetchDice() *rand.Rand {
	if fetchRng == nil {
		sd := fetchSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0xfe7c4
		}
		fetchRng = rand.New(rand.NewSource(sd))
	}
	return fetchRng
}

type fetchLevel struct {
	lvl   *Level
	want  bool
	at    float64
	done  bool
	tries int
}

var fetchLv fetchLevel

const (
	fcWalk   = iota // the thrower walks to the dog
	fcAim           // he shows the ball, the dog is ready
	fcChase         // the ball flies and bounces, the dog runs after it
	fcBack          // the dog brings it back
	fcPraise        // the dog drops it at his feet, he praises it
	fcMourn         // the thrower was killed: the dog waits by the body
)

type fetchState struct {
	m      pevMem
	dog    *Animal
	phase  int
	t      float64
	t0     float64
	n      int // throws planned
	throws int
	backs  int // balls brought back
	spot   Vec
	walkT  float64 // the time he has to reach the dog
	// the ball
	from, land, rest Vec
	fly              float64
	inMouth          bool
	onGround         bool
	ballAt           Vec // ground point now
	ballH            float64
	body             Vec
	mourned          bool
	killedThrower    bool // pelota:muerte
}

func (s *fetchState) pevHas(gu *Guest) bool { return s.m.gu == gu }

func init() {
	registerPartyEvent(&partyEvent{ID: "pelota", Name: "Lanzador de pelotas", Group: "pelota", Reactive: true, Minor: true,
		Prob: fetchChance, AtMin: fetchAtMin, AtMax: fetchAtMax, Dur: 32,
		Eligible: fetchDue, Start: fetchStart, Items: fetchItems,
		Update: pevGuard(fetchUpdate, func(g *Game, a *partyAct) { fetchEnd(g, a.D.(*fetchState)) })})
}

// fetchFits: a venue with a park, a garden or a square (campaign icon), not a special mode.
func fetchFits(g *Game) bool {
	d := g.lvl.Def
	return d != nil && d.Title != "" && fetchIcons[d.Icon] && len(d.Roam) > 0 && !g.stealth && g.dc == nil
}

func fetchReset(g *Game) {
	r := fetchDice()
	roll, at := r.Float64(), fetchAtMin+r.Float64()*(fetchAtMax-fetchAtMin)
	fetchLv = fetchLevel{lvl: g.lvl, at: g.time + at}
	switch {
	case pevDebugOn("pelota"):
		fetchLv.want, fetchLv.at = true, g.time+2
	case !fetchFits(g):
	case fetchForce:
		fetchLv.want = true
	case testing.Testing() && !fetchInTests:
	default:
		fetchLv.want = roll < fetchChance
	}
}

// fetchDog: a calm dog of the level that nobody holds (nil = none).
func (g *Game) fetchDog() *Animal {
	for _, a := range g.animals {
		if a.kind == animDog && pevAnimalCalm(a) && !pevAnimalHeld(a) && g.lvl.Outdoor(a.pos) {
			return a
		}
	}
	return nil
}

func fetchDue(g *Game, t partyTraits) bool {
	if g.lvl == nil {
		return false
	}
	if fetchLv.lvl != g.lvl {
		fetchReset(g)
	}
	if !fetchLv.want || fetchLv.done || g.time < fetchLv.at {
		return false
	}
	d := g.fetchDog()
	return d != nil && !g.pevAlarmNear(d.pos, 140)
}

// fetchLanding: where the ball falls (and where it stops after the bounces): open, outdoor, reachable by the dog.
func (g *Game) fetchLanding(r *rand.Rand, from, dogAt Vec) (Vec, Vec, bool) {
	l := g.lvl
	ok := func(p Vec) bool {
		return l.Free(p) && !l.SolidP(p.X, p.Y) && !l.InPool(p) && !l.badGround(p) && l.Outdoor(p) && p.X > 16 && p.Y > 24 &&
			p.X < l.W-16 && p.Y < l.H-16
	}
	for i := 0; i < 40; i++ {
		ang := r.Float64() * 2 * math.Pi
		d := 70 + r.Float64()*55
		land := Vec{math.Round(from.X + math.Cos(ang)*d), math.Round(from.Y + math.Sin(ang)*d*0.7)}
		if !ok(land) || !l.clearLine(from, land) {
			continue
		}
		u := faceVec(from, land)
		rest := Vec{math.Round(land.X + u.X*16), math.Round(land.Y + u.Y*16)}
		if !ok(rest) || !l.clearLine(land, rest) {
			rest = land
		}
		if p := l.Path(dogAt, rest); p == nil || dist(p[len(p)-1], rest) > 8 {
			continue
		}
		return land, rest, true
	}
	return Vec{}, Vec{}, false
}

func fetchStart(g *Game, a *partyAct) bool {
	r := fetchDice()
	fetchLv.at = g.time + 6
	if fetchLv.tries++; fetchLv.tries > 25 {
		fetchLv.done = true
		return false
	}
	d := g.fetchDog()
	if d == nil {
		return false
	}
	// the thrower: the nearest free guest who can walk up to the dog (from farther if nobody comes near for a while)
	reach := 170.0
	if fetchLv.tries > 5 {
		reach = 320
	}
	var best *Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && dist(gu.pos, d.pos) < reach && (best == nil || dist(gu.pos, d.pos) < dist(best.pos, d.pos)) {
			best = gu
		}
	}
	if best == nil && pevDebugOn("pelota") {
		for _, gu := range g.guests {
			if g.pevFreeGuest(gu) && (best == nil || dist(gu.pos, d.pos) < dist(best.pos, d.pos)) {
				best = gu
			}
		}
	}
	if best == nil {
		return false
	}
	l := g.lvl
	var spot Vec
	found := false
	for _, off := range []Vec{{18, 2}, {-18, 2}, {0, 14}, {0, -12}} {
		p := Vec{math.Round(d.pos.X + off.X), math.Round(d.pos.Y + off.Y)}
		if l.Free(p) && !l.SolidP(p.X, p.Y) && !l.InPool(p) && !l.badGround(p) && l.PathG(best.pos, p) != nil {
			spot, found = p, true
			break
		}
	}
	if !found {
		return false
	}
	if _, _, ok := g.fetchLanding(r, spot, d.pos); !ok {
		return false
	}
	s := &fetchState{dog: d, spot: spot, n: 2 + r.Intn(3), t0: g.time, walkT: math.Max(fetchWalkMax, 6+dist(best.pos, spot)/12)}
	s.m.gu = best
	if !g.pevSendTo(&s.m, spot) {
		return false
	}
	pevHoldAnimal(g, d, s.holdDog)
	if best.icon == "" {
		best.icon, best.iconT = pevPick(r, fetchCall), 1.8
	}
	fetchLv.done = true
	a.D = s
	if pevDebugOn("pelota") {
		fmt.Printf("PEVDEMO pelota dog at %.0f %.0f\n", d.pos.X, d.pos.Y)
	}
	return true
}

// holdDog: the dog of the game, moved by the event (sits waiting, runs, trots back, waits by the body).
func (s *fetchState) holdDog(g *Game, a *Animal, dt float64) bool {
	a.moving, a.vert = false, 0
	a.timer = 2
	switch s.phase {
	case fcChase, fcBack, fcMourn:
		if len(a.path) > 0 {
			sp := fetchRun
			if s.phase != fcChase {
				sp = fetchTrot
			}
			a.state = aWalk
			g.animalWalk(a, sp, dt)
			return true
		}
	}
	a.path = nil
	if gu := s.m.gu; gu != nil && s.phase != fcMourn && math.Abs(gu.pos.X-a.pos.X) > 3 {
		a.face = math.Copysign(1, gu.pos.X-a.pos.X)
	}
	switch s.phase {
	case fcAim:
		a.state = aIdle // standing, ready to run
	case fcChase:
		a.state = aIdle
	default:
		a.state = aGraze // sits
	}
	return true
}

func fetchEnd(g *Game, s *fetchState) {
	if s == nil {
		return
	}
	r := fetchDice()
	if s.m.mine() {
		pevRelease(&s.m, r)
	} else {
		pevForget(&s.m)
	}
	pevFreeAnimal(s.dog)
	s.inMouth, s.onGround = false, false
}

// fetchThrowNow: the ball leaves the hand; the dog runs to where it will stop.
func (g *Game) fetchThrowNow(s *fetchState, r *rand.Rand) bool {
	gu := s.m.gu
	land, rest, ok := g.fetchLanding(r, gu.pos, s.dog.pos)
	if !ok {
		return false
	}
	s.from, s.land, s.rest = gu.pos, land, rest
	s.fly = fetchFlyMin + r.Float64()*fetchFlyVar
	s.inMouth, s.onGround = false, false
	s.ballAt, s.ballH = gu.pos, fetchHandH
	s.throws++
	if pevDebugOn("pelota") {
		fmt.Printf("PEVDEMO pelota throw %d at t=%.1f to %.0f %.0f\n", s.throws, g.time, rest.X, rest.Y)
	}
	s.dog.path = g.lvl.Path(s.dog.pos, rest)
	gu.dir = faceTo(gu.pos, land)
	s.m.pose(AnimPunch) // the arm swings
	if gu.icon == "" {
		gu.icon, gu.iconT = pevPick(r, fetchThrow), 1.5
	}
	return true
}

// fetchBallAt: the ball at time t of the throw: ground point and height (the flight, then two bounces that roll on).
func (s *fetchState) fetchBallAt(t float64) (Vec, float64, bool) {
	lerp := func(a, b Vec, u float64) Vec { return Vec{a.X + (b.X-a.X)*u, a.Y + (b.Y-a.Y)*u} }
	if t < s.fly {
		u := t / s.fly
		return lerp(s.from, s.land, u), fetchHandH*(1-u) + fetchArc*4*u*(1-u), false
	}
	t -= s.fly
	b1, b2 := 0.32, 0.2
	switch {
	case t < b1:
		u := t / b1
		return lerp(s.land, s.rest, u*0.6), 8 * 4 * u * (1 - u), false
	case t < b1+b2:
		u := (t - b1) / b2
		return lerp(s.land, s.rest, 0.6+u*0.3), 3 * 4 * u * (1 - u), false
	case t < b1+b2+0.3:
		u := (t - b1 - b2) / 0.3
		return lerp(s.land, s.rest, 0.9+u*0.1), 0, false
	}
	return s.rest, 0, true
}

func fetchUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*fetchState)
	r := fetchDice()
	d := s.dog
	if d.dead || d.gone || !pevAnimalHeld(d) {
		fetchEnd(g, s)
		return true
	}
	s.t += dt
	gu := s.m.gu
	// the thrower is gone
	if s.phase != fcMourn && !s.m.mine() {
		pevForget(&s.m)
		if gu.dead && s.phase != fcWalk { // it picks the ball up (if it did not have it) and goes to him
			s.body = gu.pos
			p := g.lvl.Path(d.pos, g.lvl.NearestFree(Vec{gu.pos.X + 12, gu.pos.Y + 3}))
			if p == nil {
				p = []Vec{d.pos}
			}
			s.inMouth, s.onGround = true, false
			d.path, s.phase, s.t = p, fcMourn, 0
			return false
		}
		fetchEnd(g, s)
		return true
	}
	if s.phase != fcMourn && g.pevAlarmNear(d.pos, 120) {
		fetchEnd(g, s)
		return true
	}
	if pevDebugOn("pelota") && int(g.time*2) != int((g.time-dt)*2) {
		fmt.Printf("PEVDEMO pelota t=%.1f phase %d dog %.0f %.0f ball %.0f %.0f h %.0f mouth %v thrower %.0f %.0f\n", g.time, s.phase, d.pos.X, d.pos.Y, s.ballAt.X, s.ballAt.Y, s.ballH, s.inMouth, gu.pos.X, gu.pos.Y)
	}
	next := func(ph int) { s.phase, s.t = ph, 0 }
	switch s.phase {
	case fcWalk:
		if gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, s.spot) < 6 {
			s.m.hold(gIdle)
			gu.dir = faceTo(gu.pos, d.pos)
			s.inMouth = false
			next(fcAim)
		} else if s.t > s.walkT {
			fetchEnd(g, s)
			return true
		}
	case fcAim:
		s.m.keep()
		if s.t > 0.4 && s.t-dt <= 0.4 && d.icon == "" {
			d.icon, d.iconT = pevPick(r, fetchDogL), 1.2
		}
		if s.t >= 1.1 {
			if !g.fetchThrowNow(s, r) {
				fetchEnd(g, s)
				return true
			}
			next(fcChase)
		}
	case fcChase:
		s.m.keep()
		if s.t > 0.45 && gu.state == gFun {
			s.m.hold(gIdle) // the arm comes down
		}
		at, h, still := s.fetchBallAt(s.t)
		s.ballAt, s.ballH, s.onGround = at, h, still
		if still && dist(d.pos, s.rest) < 10 {
			s.inMouth, s.onGround = true, false
			d.path = g.lvl.Path(d.pos, g.lvl.NearestFree(Vec{gu.pos.X + math.Copysign(12, d.pos.X-gu.pos.X), gu.pos.Y + 2}))
			if d.path == nil {
				d.path = []Vec{d.pos}
			}
			next(fcBack)
		} else if still && len(d.path) == 0 {
			d.path = g.lvl.Path(d.pos, s.rest) // it stopped short: once more
			if d.path == nil || s.t > fetchChaseMax {
				fetchEnd(g, s)
				return true
			}
		} else if s.t > fetchChaseMax {
			fetchEnd(g, s)
			return true
		}
	case fcBack:
		s.m.keep()
		gu.dir = faceTo(gu.pos, d.pos)
		if len(d.path) == 0 {
			s.inMouth, s.onGround = false, true
			s.ballAt, s.ballH = Vec{math.Round((d.pos.X + gu.pos.X) / 2), math.Round(gu.pos.Y + 2)}, 0
			s.backs++
			s.m.pose(AnimCheer)
			if gu.icon == "" {
				gu.icon, gu.iconT = pevPick(r, fetchGood), 1.6
			}
			next(fcPraise)
		} else if s.t > fetchChaseMax {
			fetchEnd(g, s)
			return true
		}
		if pevDemo("pelota") == "muerte" && !s.killedThrower && s.t > 0.8 {
			s.killedThrower = true
			g.killGuest(gu, false)
		}
	case fcPraise:
		s.m.keep()
		if s.t >= 1.6 {
			if s.throws >= s.n || g.time-s.t0 > fetchMaxT-6 {
				fetchEnd(g, s)
				return true
			}
			s.m.hold(gIdle)
			s.onGround = false // he picks it up
			next(fcAim)
		}
	case fcMourn:
		if len(d.path) == 0 && !s.mourned {
			s.mourned = true
			if math.Abs(s.body.X-d.pos.X) > 2 {
				d.face = math.Copysign(1, s.body.X-d.pos.X)
			}
			d.icon, d.iconT = pevPick(r, fetchMournL), 2.4
		}
		if s.t > fetchMourn+4 || s.mourned && s.t > fetchMourn {
			fetchEnd(g, s)
			return true
		}
	}
	return false
}

// ---- drawing -------------------------------------------------------------------------------------------------

var (
	fetchBallCol  = color.RGBA{0xd8, 0xf0, 0x40, 255}
	fetchBallLine = color.RGBA{0xf8, 0xf8, 0xf0, 255}
	fetchBallDark = color.RGBA{0x8a, 0xa0, 0x20, 255}
	fetchShadow   = color.RGBA{0, 0, 0, 0x50} // premultiplied: a soft dark
)

// fetchBallPix: a 3x3 tennis ball centred on (x, y) with its seam.
func fetchBallPix(plot plotFn, x, y int) {
	plot(x-1, y-1, 3, 3, fetchBallCol)
	plot(x-1, y+1, 3, 1, fetchBallDark)
	plot(x-1, y-1, 1, 1, fetchBallLine)
	plot(x, y, 1, 1, fetchBallLine)
}

func fetchItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s, ok := a.D.(*fetchState)
	if !ok {
		return items
	}
	plot := g.plotter(dst)
	d := s.dog
	switch {
	case s.inMouth && !d.dead:
		w, h := g.animalSize(d)
		if h == 0 {
			return items
		}
		items = append(items, drawItem{d.pos.Y + 0.5, func() {
			cx, cy := g.pevXY(d.pos)
			fetchBallPix(plot, cx+int(d.face)*(w/2-1), cy-h/2+1)
		}})
	case s.phase == fcChase || s.onGround:
		at, hh := s.ballAt, s.ballH
		items = append(items, drawItem{at.Y + 0.2, func() {
			cx, cy := g.pevXY(at)
			sw := 3
			if hh > 12 {
				sw = 2
			}
			plot(cx-sw/2-1, cy, sw+2, 1, fetchShadow)
			fetchBallPix(plot, cx, cy-1-int(math.Round(hh)))
		}})
	case s.phase == fcAim: // the ball in the thrower's hand
		gu := s.m.gu
		items = append(items, drawItem{gu.pos.Y + 0.5, func() {
			cx, cy := g.pevXY(gu.pos)
			side := 5
			if gu.dir == DirLeft {
				side = -5
			}
			fetchBallPix(plot, cx+side, cy-g.pevK()(13))
		}})
	}
	return items
}
