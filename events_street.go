package main

import (
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// STREET EVENTS (ideas 249 and 250): two "sometimes" things of the street in front of the parties that have one (the
// two one-way lanes of the police cars along Def.CarFrom.Y: Police == "car", no sea or space transit, not the highway).
// Both are party events of the frame (partyevents.go): own dice, only in some parties, a time window, never in STEALTH,
// the RETO DE HOY or the island, and the group "street" keeps them apart.
//
//   - "averia": an ordinary car of the traffic (the new car of vehart.go, drawn by drawAmbient) runs out of fuel. It is
//     taken out of g.amb while it stands (so nobody is run over by a car that does not move), the driver gets out, looks
//     at the smoking engine, asks for help, pushes it a bit and it starts again: it goes back to the lane with its own
//     speed. The cars behind it wait in a queue (streetHold) and honk now and then in speech bubbles; they never drive
//     through it. No stars, no points, nobody hurt: the driver is not a guest.
//   - "taxi": a taxi (the yellow taxi of the kit, with a generic "TAXI" plate drawn over its roof sign) stops by the
//     kerb and 1-4 guests who leave the party walk to it and get in. They are GONE (not dead): out of the party and out
//     of the kill target (g.total--, like the cleaners), no points, no casualty. Tens of seconds later a taxi comes back
//     and brings new guests through addGuestAt (it asks guestRoom: never more than the caps of cap.go allow). Never the
//     dead, the cuffed or blamed, a named or famous guest, a performer, a guest of a scene (selfie, contest, conga...:
//     pevFreeGuest) or the target of a killer. Never when the party is nearly over (lateJoinMin + a margin alive).
// ---------------------------------------------------------------------------

var (
	averiaOut   = pevTexts("¡Vaya, sin gasolina!", "¿Pero qué le pasa ahora?", "¡Me he quedado tirado!")
	averiaLook  = pevTexts("¡Humo! Eso no es buena señal", "¿Y ahora qué hago?", "¿Alguien me echa una mano?", "Venga, bonito, arranca...")
	averiaPush  = pevTexts("Uf... cómo pesa", "¡Un poco más!", "¡Empujando, que es gerundio!")
	averiaGo    = pevTexts("¡Arrancó!", "¡Por fin!", "¡Perdón por el atasco!")
	averiaHonk  = pevTexts("¡PIII!", "¡PIII, PIII!", "¡Venga ya!", "¡Mueve eso!", "Bueno, esperamos...", "Sin prisa, eh")
	taxiCall    = pevTexts("¡Taxi!", "Me voy, que mañana curro", "Yo me piro", "¡Esperadme!", "Nos vamos a otra fiesta")
	taxiSayGo   = pevTexts("Un taxi se lleva a unos invitados")
	taxiSayBack = pevTexts("Llega un taxi con más invitados")
)

const (
	streetGapStop = 18.0 // a held car stops this far behind the one in front
	averiaPushV   = 9.0  // px/s while the driver pushes
	taxiStayMax   = 16.0 // seconds the taxi waits for its riders
	taxiBackMin   = 30.0 // the taxi with new guests comes back 30-55 s later
	taxiBackVar   = 25.0
	taxiAliveKeep = lateJoinMin + 12 // never empty a party that is nearly over
)

func init() {
	registerPartyEvent(&partyEvent{ID: "averia", Name: "Coche sin gasolina", Group: "street",
		Prob: 0.13, AtMin: 40, AtMax: 220, Dur: 26, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool { return streetOK(partyDefOf(g, t)) },
		Start:    averiaStart, Items: averiaItems,
		Update: pevGuard(averiaUpdate, func(g *Game, a *partyAct) { averiaEnd(g, a.D.(*averiaState)) })})
	registerPartyEvent(&partyEvent{ID: "taxi", Name: "Taxi que recoge invitados", Group: "street",
		Prob: 0.13, AtMin: 60, AtMax: 240, Dur: 62, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool { return streetOK(partyDefOf(g, t)) && t.guests >= 20 },
		Start:    taxiStart, Items: taxiItems,
		Update: pevGuard(taxiUpdate, func(g *Game, a *partyAct) { taxiEnd(g, a.D.(*taxiState)) })})
}

// partyDefOf: the level definition the traits were taken from (the plan is also rolled for levels that are not loaded).
func partyDefOf(g *Game, t partyTraits) *LevelDef {
	if g != nil && g.lvl != nil && g.levelIdx == t.idx && g.lvl.Def != nil {
		return g.lvl.Def
	}
	if t.idx >= 0 && t.idx < len(levelDefs) {
		return levelDefs[t.idx]
	}
	return nil
}

// streetOK: the level has the street of the police cars with ordinary traffic.
func streetOK(d *LevelDef) bool {
	return d != nil && d.Police == "car" && d.Transit == "" && !d.Highway && d.Sky != "space"
}

// ---- the street ------------------------------------------------------------------------------------------

// streetLanes: the y of each lane and the way it drives (+1 right, -1 left).
func (g *Game) streetLanes() (ys, dirs []float64) {
	def := g.lvl.Def
	if len(def.Lanes) > 0 {
		for _, y := range def.Lanes {
			d := -1.0
			if y > def.CarFrom.Y {
				d = 1
			}
			ys, dirs = append(ys, y), append(dirs, d)
		}
		return
	}
	return []float64{def.CarFrom.Y + g.laneOff(), def.CarFrom.Y - g.laneOff()}, []float64{1, -1}
}

func ambIn(g *Game, a *Ambient) bool {
	for _, o := range g.amb {
		if o == a {
			return true
		}
	}
	return false
}

func ambOut(g *Game, a *Ambient) {
	for i, o := range g.amb {
		if o == a {
			g.amb = append(g.amb[:i], g.amb[i+1:]...)
			return
		}
	}
}

// streetHold: the traffic behind a vehicle that stands at x in the lane y (and is not in g.amb) waits behind it, one
// car after the other (driveAmbient drives through stopped cars: here they are told to wait). Returns the first of the
// queue (nil = nobody waits).
func (g *Game) streetHold(x, y, dir, length float64) *Ambient {
	var qs []*Ambient
	for _, o := range g.amb {
		if o.dead || !roadKind(o.kind) || o.kind == ambTank || math.Abs(o.y-y) > 8 || o.v0 == 0 || math.Copysign(1, o.v0) != dir {
			continue
		}
		if (x-o.x)*dir <= 0 {
			continue // already past it
		}
		qs = append(qs, o)
	}
	sort.Slice(qs, func(i, j int) bool { return (x-qs[i].x)*dir < (x-qs[j].x)*dir })
	var first *Ambient
	leadX, leadL := x, length
	for _, o := range qs {
		if (leadX-o.x)*dir-(leadL+o.length)/2 >= streetGapStop {
			break
		}
		o.wait = math.Max(o.wait, 0.15)
		if first == nil {
			first = o
		}
		leadX, leadL = o.x, o.length
	}
	return first
}

// streetBehindGap: the free road behind a car of the lane (1e9 = nobody behind it).
func (g *Game) streetBehindGap(c *Ambient) float64 {
	dir := math.Copysign(1, c.v0)
	best := 1e9
	for _, o := range g.amb {
		if o == c || o.dead || !roadKind(o.kind) || math.Abs(o.y-c.y) > 8 || math.Copysign(1, o.v0) != dir || (c.x-o.x)*dir <= 0 {
			continue
		}
		best = math.Min(best, (c.x-o.x)*dir-(c.length+o.length)/2)
	}
	return best
}

// streetHonk: now and then the first car of the queue honks (a speech bubble, no sound: no repeated beeps).
type streetHonk struct {
	car   *Ambient
	txt   string
	until float64
	next  float64
}

func (h *streetHonk) update(g *Game, r *rand.Rand, first *Ambient) {
	if first == nil {
		h.car = nil
		return
	}
	if h.next == 0 {
		h.next = g.time + 1.5 + r.Float64()*1.5
	}
	if g.time >= h.next {
		h.car, h.txt, h.until, h.next = first, pevPick(r, averiaHonk), g.time+1.6, g.time+3+r.Float64()*3
	}
	if h.car != nil && g.time < h.until && !h.car.dead {
		g.bubble(h.car, bubGuest, Vec{h.car.x, h.car.y + 4}, h.txt, false, 0)
	}
}

// ---- 249: a car runs out of fuel --------------------------------------------------------------------------

const (
	avCome = iota - 1 // the chosen car is still driving to the place where it stops
	avOut
	avLook
	avToTail
	avPush
	avBack
	avDone
)

type averiaState struct {
	car     *Ambient
	v0      float64
	dir     float64
	drv     *pevPerf
	phase   int
	t0      float64
	say     string
	sayT    float64
	lookFor float64
	pushFor float64
	honk    streetHonk
	nextFx  float64
	said    bool
	waitGo  float64
	stallX  float64
}

func averiaStart(g *Game, a *partyAct) bool {
	r := partyDice()
	var cand []*Ambient
	for _, o := range g.amb {
		if !averiaCar(o) || o.dead || o.v0 == 0 || math.Abs(o.vx) < 20 {
			continue
		}
		// a car on the road that has not passed the middle of the street yet
		if ahead := (g.lvl.W/2 - o.x) * math.Copysign(1, o.v0); ahead < 40 || ahead > g.lvl.W/2+120 {
			continue
		}
		cand = append(cand, o)
	}
	if len(cand) == 0 {
		return false
	}
	car := cand[r.Intn(len(cand))]
	dir := math.Copysign(1, car.v0)
	s := &averiaState{car: car, v0: car.v0, dir: dir, t0: g.time, lookFor: 5 + r.Float64()*3, pushFor: 4 + r.Float64()*3, phase: avCome}
	// it stops somewhere in the middle of the street, ahead of it
	lo, hi := math.Max(220, car.x+dir*30), g.lvl.W-220
	if dir < 0 {
		lo, hi = 220, math.Min(g.lvl.W-220, car.x-30)
	}
	if hi < lo {
		return false
	}
	s.stallX = lo + r.Float64()*(hi-lo)
	a.D = s
	return true
}

// averiaCar: an ordinary car of the traffic (the plain car, a van or a taxi of the vehicle kit), not a set piece.
func averiaCar(o *Ambient) bool {
	return o.kind == ambCar || o.kind == ambObj && (o.name == "van" || o.name == "taxi")
}

// averiaStall: the car has reached its place and dies there.
func averiaStall(g *Game, s *averiaState, r *rand.Rand) {
	car := s.car
	ambOut(g, car)
	car.vx = 0
	spec := genericSpec(r)
	d := DirRight
	if s.dir < 0 {
		d = DirLeft
	}
	s.drv = newPevPerf(spec, averiaDoor(s), d)
	s.drv.path = []Vec{{s.drv.pos.X, s.drv.pos.Y + 6}}
	s.talk(g, pevPick(r, averiaOut))
	g.snd.Play("thud")
	s.phase, s.t0 = avOut, g.time
}

func averiaDoor(s *averiaState) Vec { return Vec{s.car.x + s.dir*6, s.car.y + 2} }
func averiaNose(s *averiaState) Vec { return Vec{s.car.x + s.dir*40, s.car.y + 4} }
func averiaTail(s *averiaState) Vec { return Vec{s.car.x - s.dir*40, s.car.y + 4} }

func (s *averiaState) talk(g *Game, txt string) { s.say, s.sayT = txt, g.time+2.6 }

func averiaUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*averiaState)
	r := partyDice()
	age := g.time - s.t0
	if s.phase == avCome {
		if s.car.dead || !ambIn(g, s.car) || age > 20 {
			s.phase = avDone // it never got there (gone, crushed): nothing happened
			return true
		}
		// past its place, it stops as soon as nobody drives right behind it (a sudden stop is never rammed)
		if (s.stallX-s.car.x)*s.dir <= 0 && g.streetBehindGap(s.car) > 30 && math.Abs(s.car.x-g.lvl.W/2) < g.lvl.W/2-150 {
			averiaStall(g, s, r)
		}
		return false
	}
	if s.phase < avDone {
		s.honk.update(g, r, g.streetHold(s.car.x, s.car.y, s.dir, s.car.length))
	}
	if s.drv != nil && g.time < s.sayT {
		g.bubble(s.drv, bubGuest, s.drv.pos, s.say, false, 0)
	}
	switch s.phase {
	case avOut:
		s.drv.step(dt)
		if age > 1.4 {
			s.phase, s.t0 = avLook, g.time
			s.drv.path = []Vec{averiaNose(s)}
		}
	case avLook:
		if s.drv.step(dt) {
			s.drv.dir = DirLeft
			if s.dir < 0 {
				s.drv.dir = DirRight
			}
		}
		if g.time >= s.nextFx { // smoke from the engine
			s.nextFx = g.time + 0.22
			hood := Vec{s.car.x + s.dir*22, s.car.y - 20}
			l := 0.9 + r.Float64()*0.5
			g.parts = append(g.parts, Particle{pos: hood, vel: Vec{(r.Float64() - 0.5) * 8, -14 - r.Float64()*8}, life: l, max: l,
				col: color.RGBA{0x9a, 0x9a, 0xa2, 255}, size: 3, grav: -6})
		}
		if !s.said && g.time > s.sayT {
			s.said = true
			s.talk(g, pevPick(r, averiaLook))
		}
		if age > s.lookFor {
			s.phase, s.t0 = avToTail, g.time
			s.drv.path = []Vec{{s.drv.pos.X, s.car.y + 12}, {averiaTail(s).X, s.car.y + 12}, averiaTail(s)}
		}
	case avToTail:
		if s.drv.step(dt) || age > 8 {
			s.drv.pos, s.phase, s.t0 = averiaTail(s), avPush, g.time
			s.talk(g, pevPick(r, averiaPush))
		}
	case avPush:
		s.car.x += s.dir * averiaPushV * dt
		s.car.phase += dt * 0.4
		s.drv.pos = averiaTail(s)
		s.drv.dir = DirRight
		if s.dir < 0 {
			s.drv.dir = DirLeft
		}
		s.drv.anim += dt * 4
		if age > s.pushFor || s.car.x < 160 || s.car.x > g.lvl.W-160 {
			s.phase, s.t0 = avBack, g.time
			s.drv.path = []Vec{{s.drv.pos.X, s.car.y + 12}, {averiaDoor(s).X, s.car.y + 12}, averiaDoor(s)}
		}
	case avBack:
		if s.drv.step(dt) || age > 8 {
			s.drv.pos = averiaDoor(s)
			// he gets in; the car starts once nobody stands in front of it (4 s at most)
			if g.streetClear(s.car.x, s.car.y, s.dir) || s.waitGo > 4 {
				averiaGoes(g, s, r)
				return true
			}
			s.waitGo += dt
		}
	case avDone:
		return true
	}
	return false
}

// averiaGoes: the driver gets in and the car starts: back in its lane at its own speed.
func averiaGoes(g *Game, s *averiaState, r *rand.Rand) {
	if s.phase == avDone || s.phase == avCome {
		s.phase = avDone
		return
	}
	s.phase = avDone
	if !s.car.dead && !ambIn(g, s.car) {
		s.car.v0, s.car.vx, s.car.wait = s.v0, s.v0, 0
		g.amb = append(g.amb, s.car)
	}
	if s.drv != nil {
		g.bubbleAt(Vec{s.car.x, s.car.y}, pevPick(r, averiaGo), 1.8)
		pevPuff(g, Vec{s.car.x - s.dir*34, s.car.y + 2}, 6, r)
	}
	s.drv = nil
}

func averiaEnd(g *Game, s *averiaState) { averiaGoes(g, s, partyDice()) }

func averiaItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*averiaState)
	if s.phase == avDone || s.phase == avCome {
		return items
	}
	car := s.car
	items = append(items, drawItem{car.y, func() { g.drawAmbient(dst, car) }})
	if drv := s.drv; drv != nil {
		anim, ph := AnimIdle, g.time*2
		switch s.phase {
		case avLook:
			if !drv.moving {
				anim, ph = AnimCrouch, g.time*1.5
			}
		case avPush:
			anim, ph = AnimCarry, drv.anim
		}
		items = append(items, drawItem{drv.pos.Y, func() { drv.draw(g, dst, anim, ph) }})
	}
	return items
}

// ---- 250: a taxi takes some guests home (and later brings new ones) ------------------------------------

const (
	txCome = iota
	txBoard
	txLeave
	txWait
	txBack
	txDrop
	txLeave2
	txDone
)

type taxiState struct {
	cab          *Ambient
	lane, dir    float64
	side         float64 // +1: the guests come from below the lane, -1: from above
	stopX        float64
	phase        int
	t0           float64
	riders       []*pevMem
	n            int // got in
	backAt       float64
	dropped      int
	nextDrop     float64
	honk         streetHonk
	saidGo, said bool
	waitGo       float64
}

func (s *taxiState) pevHas(gu *Guest) bool { return pevHasIn(s.riders, gu) }

// taxiCanTake: a plain guest who may leave the party now.
func taxiCanTake(g *Game, gu *Guest) bool {
	if !g.pevFreeGuest(gu) || gu.name != "" || gu.keep || gu.vipCard || gu.famousOK || gu == g.suspect || pevFxs[gu] != nil {
		return false
	}
	if gu.cs != nil && gu.cs.Spec != nil && (g.isFamous(gu.cs.Spec.Name) || isKidSlug(gu.cs.Spec.Name)) {
		return false
	}
	for _, sl := range g.allSlashers() {
		if sl != nil && sl.tgtG == gu {
			return false
		}
	}
	return gu.state != gLounge
}

func taxiNewCab(g *Game, lane, dir float64) *Ambient {
	x := -100.0
	if dir < 0 {
		x = g.lvl.W + 100
	}
	v := 105 * dir
	if k := propKinds["taxi"]; k != nil && k.Ambient > 0 {
		v = k.Ambient * dir
	}
	cab := &Ambient{kind: ambObj, name: "taxi", x: x, y: lane, vx: v, v0: v, phase: 1, stopX: -1e9}
	cab.length = ambLength(cab)
	return cab
}

// kerb: where rider k gets in (by the door, on the side the guests come from); wait: where he waits for the taxi,
// clear of the lane (a moving car runs over whoever is within 11 px of its lane).
func (s *taxiState) kerb(k int) Vec {
	dy := 8.0
	if s.side < 0 {
		dy = 6
	}
	return Vec{s.stopX + s.dir*(float64(k)*9-14), s.lane + s.side*dy}
}

func (s *taxiState) wait(k int) Vec {
	return Vec{s.stopX + s.dir*(float64(k)*11-16), s.lane + s.side*28}
}

// streetClear: nobody (guest or player) stands where a car at (x, y) would run him over when it starts.
func (g *Game) streetClear(x, y, dir float64) bool {
	near := func(p Vec) bool { return math.Abs(p.Y-y) < 13 && (p.X-x)*dir > -36 && (p.X-x)*dir < 70 }
	for _, gu := range g.guests {
		if !gu.out() && near(gu.pos) {
			return false
		}
	}
	for _, pl := range g.players() {
		if pl != nil && !pl.dead && near(pl.pos) {
			return false
		}
	}
	return true
}

func taxiStart(g *Game, a *partyAct) bool {
	if propKinds["taxi"] == nil || g.aliveCount() < taxiAliveKeep+2 {
		return false
	}
	r := partyDice()
	var cand []*Guest
	for _, gu := range g.guests {
		if taxiCanTake(g, gu) && math.Abs(gu.pos.Y-g.lvl.Def.CarFrom.Y) < 340 {
			cand = append(cand, gu)
		}
	}
	n := minI(1+r.Intn(4), g.aliveCount()-taxiAliveKeep)
	if n < 1 || len(cand) < n {
		return false
	}
	// the lane on the side of the guests: they never cross the other one to get in
	my := 0.0
	for _, gu := range cand {
		my += gu.pos.Y
	}
	my /= float64(len(cand))
	ys, dirs := g.streetLanes()
	li := 0
	for i := range ys {
		if math.Abs(ys[i]-my) < math.Abs(ys[li]-my) {
			li = i
		}
	}
	s := &taxiState{lane: ys[li], dir: dirs[li], t0: g.time, side: 1}
	if my < s.lane {
		s.side = -1
	}
	sort.Slice(cand, func(i, j int) bool { return math.Abs(cand[i].pos.Y-s.lane) < math.Abs(cand[j].pos.Y-s.lane) })
	cand = cand[:minI(len(cand), n*3)]
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	mx := 0.0
	for _, gu := range cand[:n] {
		mx += gu.pos.X
	}
	mx /= float64(n)
	// a stop where the kerb and the waiting points can be walked to
	best, bestD := 0.0, 1e9
	for x := 140.0; x <= g.lvl.W-140; x += 20 {
		s.stopX = x
		ok := true
		for k := 0; k < 4 && ok; k++ {
			for _, p := range []Vec{s.kerb(k), s.wait(k)} {
				q := g.lvl.NearestFree(p)
				ok = ok && dist(p, q) < 4 && !g.lvl.InPool(q) && !g.lvl.SolidP(q.X, q.Y)
			}
		}
		if ok && math.Abs(x-mx) < bestD {
			best, bestD = x, math.Abs(x-mx)
		}
	}
	if bestD > 1e8 {
		return false
	}
	s.stopX = best
	for _, gu := range cand {
		if len(s.riders) >= n {
			break
		}
		k := len(s.riders)
		m := &pevMem{gu: gu, pt: s.kerb(k), idx: k}
		if g.lvl.PathG(gu.pos, s.kerb(k)) == nil || !g.pevSendTo(m, s.wait(k)) {
			pevForget(m)
			continue
		}
		s.riders = append(s.riders, m)
	}
	if len(s.riders) == 0 {
		return false
	}
	s.riders[0].gu.icon, s.riders[0].gu.iconT = pevPick(r, taxiCall), 2
	s.cab = taxiNewCab(g, s.lane, s.dir)
	g.amb = append(g.amb, s.cab)
	a.D = s
	return true
}

// taxiArrive: the cab reached its stop: it leaves the lane (the traffic behind waits) and stands there.
func taxiArrive(g *Game, s *taxiState) bool {
	c := s.cab
	if !ambIn(g, c) || (s.stopX-c.x)*s.dir > 1 {
		return false
	}
	// a sudden stop is never rammed: with a car right behind it pulls in a little further on
	if over := (c.x - s.stopX) * s.dir; g.streetBehindGap(c) < 30 && over < 60 {
		return false
	}
	ambOut(g, c)
	c.vx = 0
	s.stopX = c.x
	for k, m := range s.riders {
		m.pt = g.lvl.NearestFree(s.kerb(k))
	}
	return true
}

// taxiDepart: back into the lane once nobody stands in its way (true = gone; at most 4 s of waiting).
func taxiDepart(g *Game, s *taxiState, dt float64, force bool) bool {
	c := s.cab
	if c == nil || c.dead || ambIn(g, c) {
		return true
	}
	if !force && !g.streetClear(c.x, c.y, s.dir) && s.waitGo < 4 {
		s.waitGo += dt
		return false
	}
	s.waitGo = 0
	c.vx, c.wait = c.v0, 0
	g.amb = append(g.amb, c)
	return true
}

func taxiUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*taxiState)
	r := partyDice()
	age := g.time - s.t0
	if (s.phase == txBoard || s.phase == txDrop) && !ambIn(g, s.cab) {
		s.honk.update(g, r, g.streetHold(s.cab.x, s.cab.y, s.dir, s.cab.length))
	}
	switch s.phase {
	case txCome, txBack:
		s.riders = pevPruneMems(s.riders)
		for _, m := range s.riders {
			if m.gu.state == gIdle {
				m.want = gIdle
				m.keep()
				m.gu.dir = DirDown
				if s.side > 0 {
					m.gu.dir = DirUp
				}
			}
		}
		if s.cab.dead || !ambIn(g, s.cab) || age > 30 {
			taxiEnd(g, s)
			return true
		}
		if taxiArrive(g, s) {
			if s.phase == txCome {
				s.phase, s.t0 = txBoard, g.time
				for _, m := range s.riders {
					g.pevSendTo(m, m.pt)
				}
			} else {
				s.phase, s.t0, s.nextDrop = txDrop, g.time, g.time+0.6
			}
		}
	case txBoard:
		live := s.riders[:0]
		for _, m := range s.riders {
			if !m.mine() {
				pevForget(m)
				continue
			}
			if gu := m.gu; gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, m.pt) < 6 {
				taxiBoard(g, gu)
				s.n++
				continue
			}
			live = append(live, m)
		}
		s.riders = live
		if len(s.riders) == 0 || age > taxiStayMax {
			if len(s.riders) > 0 {
				pevReleaseAll(s.riders, r)
				s.riders = nil
			}
			if s.n > 0 && !s.saidGo {
				s.saidGo = true
				g.say(taxiSayGo[0])
			}
			if taxiDepart(g, s, dt, false) {
				s.phase, s.t0 = txLeave, g.time
			}
		}
	case txLeave, txLeave2:
		if !ambIn(g, s.cab) || age > 20 {
			if s.phase == txLeave2 || s.n == 0 {
				s.phase = txDone
				return true
			}
			s.phase, s.backAt = txWait, g.time+taxiBackMin+r.Float64()*taxiBackVar
		}
	case txWait:
		if g.time >= s.backAt {
			if g.guestRoom() < 1 {
				s.phase = txDone
				return true // the party is full or closing: nobody comes
			}
			s.cab = taxiNewCab(g, s.lane, s.dir)
			g.amb = append(g.amb, s.cab)
			s.phase, s.t0 = txBack, g.time
		}
	case txDrop:
		if g.time >= s.nextDrop && s.dropped < s.n && g.guestRoom() > 0 {
			s.nextDrop = g.time + 0.55
			before := len(g.guests)
			g.addGuestAt(s.wait(s.dropped))
			if len(g.guests) > before {
				gu := g.guests[len(g.guests)-1]
				gu.dir = DirDown
				if s.side < 0 {
					gu.dir = DirUp
				}
				taxiBackN++
				if !s.said {
					s.said = true
					g.say(taxiSayBack[0])
				}
			}
			s.dropped++
		}
		if s.dropped >= s.n || g.guestRoom() < 1 || age > 8 {
			if taxiDepart(g, s, dt, false) {
				s.phase, s.t0 = txLeave2, g.time
			}
		}
	case txDone:
		return true
	}
	return false
}

// taxiBoard: the guest gets in and leaves the party: gone (not dead), out of the kill target, no points.
func taxiBoard(g *Game, gu *Guest) {
	if gu.out() {
		return
	}
	gu.gone, gu.icon, gu.iconT, gu.path, gu.moving = true, "", 0, nil, false
	g.total--
	taxiGoneN++
	g.checkDone()
}

// taxiGoneN / taxiBackN: guests who left in a taxi / came in one (tests).
var taxiGoneN, taxiBackN int

func taxiEnd(g *Game, s *taxiState) {
	pevReleaseAll(s.riders, partyDice())
	s.riders = nil
	taxiDepart(g, s, 0, true)
	s.phase = txDone
}

// taxiItems: the cab while it stands (it is out of g.amb), and the TAXI plate on its roof whenever it is on screen.
func taxiItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*taxiState)
	c := s.cab
	if c == nil || c.dead || s.phase == txDone || s.phase == txWait {
		return items
	}
	if !ambIn(g, c) {
		items = append(items, drawItem{c.y, func() { g.drawAmbient(dst, c) }})
	}
	items = append(items, drawItem{c.y + 0.01, func() { g.drawTaxiPlate(dst, c) }})
	return items
}

// taxiFont: the four letters of the plate, 3x5.
var taxiFont = map[byte][5]uint8{
	'T': {7, 2, 2, 2, 2}, 'A': {2, 5, 7, 5, 5}, 'X': {5, 5, 2, 5, 5}, 'I': {7, 2, 2, 2, 7},
}

// taxiPlateRect: the plate over the roof sign of the taxi standing (or driving) at (x, y) of the world, in screen
// pixels: 19x9 (a dark edge, a white plate and 4 letters of 3x5 with a pixel between them and around them).
func (g *Game) taxiPlateRect(c *Ambient) (int, int, int, int) {
	x, y := int(math.Round(g.wx(c.x))), int(math.Round(g.wy(c.y)))
	return x - 10, y - 40, 19, 9
}

func (g *Game) drawTaxiPlate(dst *ebiten.Image, c *Ambient) {
	px, py, w, h := g.taxiPlateRect(c)
	drawTaxiPlatePix(g.plotter(dst), px, py, w, h)
}

func drawTaxiPlatePix(plot plotFn, px, py, w, h int) {
	ink := color.RGBA{0x15, 0x15, 0x1b, 255}
	plot(px, py, w, h, ink)
	plot(px+1, py+1, w-2, h-2, color.RGBA{0xf4, 0xf4, 0xf6, 255})
	x := px + 2
	for _, ch := range []byte("TAXI") {
		f := taxiFont[ch]
		for r := 0; r < 5; r++ {
			for k := 0; k < 3; k++ {
				if f[r]>>(2-k)&1 == 1 {
					plot(x+k, py+2+r, 1, 1, ink)
				}
			}
		}
		x += 4
	}
}
