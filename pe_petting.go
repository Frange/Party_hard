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
// PETTING THE ANIMALS (idea 286, approved by the user: "que interactúen con algunos animales, les acaricien si están
// cerca a veces (perros, caballos, gatos)").
//
// A reactive, Minor party event "caricia" (it never takes one of the places of the frame, like the mechanical bull):
// now and then (a look every petLookGap seconds, a low chance petChance, a pause after each petting) a free guest who
// happens to be near a dog, a cat or a horse of the level stops, walks to its head, crouches (stands, for the horse)
// and pets it for 3-6 s: little hearts rise, the dog wags its tail, the cat purrs, the horse nods and neighs softly.
// Then he goes on his way. Cooldowns per guest and per animal.
//
// The animal is HELD while it waits and is petted (pevAnimalHold, one line in updateAnimals): it stays still and does
// not wander. It is never held against its own fear: a fleeing animal is never chosen, a cat that the player (or a
// dog) comes near to is let go at once and its own AI makes it run (updateCat), and a horse with the player behind it
// is let go too (it may kick). A guest who is frightened, killed or alarmed ends it; an alarm near ends it.
// Nothing is added: the animal is one of g.animals (the caps of capAnimals are untouched) and the guest stays a guest.
// Dice: petDice() (never g.rng nor the frame's dice). -pevent caricia[:perro|:gato|:caballo] forces it (progress frozen).
// ---------------------------------------------------------------------------

const (
	petNearR      = 46.0 // a guest this close to the animal may stop to pet it
	petChance     = 0.12 // per look, when some pair is near
	petLookGap    = 1.5
	petMinT       = 3.0
	petMaxT       = 6.0
	petWalkMax    = 8.0
	petGuestCool  = 150.0
	petAnimalCool = 60.0
	petPauseMin   = 35.0
	petPauseVar   = 45.0
)

var (
	petLinesDog   = pevTexts("¿Quién es un perro bueno?", "¡Hola, bonito!", "Qué orejitas...", "¡Toma mimos!")
	petLinesCat   = pevTexts("Michi, michi...", "Qué suave eres", "¡Hola, gatito!", "¿Me dejas, minino?")
	petLinesHorse = pevTexts("Tranquilo, caballito", "¡Qué crin tan bonita!", "Sooo, bonito, sooo")
	petAnDog      = pevTexts("*mueve la cola*", "*jadea feliz*")
	petAnCat      = pevTexts("*rrrrrr*", "*ronronea*")
	petAnHorse    = pevTexts("*relincha bajito*", "*hiii...*")
)

var (
	petRng     *rand.Rand
	petSeed    int64 // tests: non-zero = deterministic dice
	petInTests bool  // tests: let it happen under go test (off by default so the other tests' parties stay as they were)
)

func petDice() *rand.Rand {
	if petRng == nil {
		sd := petSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x9e7c0a7
		}
		petRng = rand.New(rand.NewSource(sd))
	}
	return petRng
}

// ---- holding an animal (shared with the ball thrower, pe_fetch.go) ---------------------------------------

// pevAnimalHolds: the animals an event holds now and the function that moves them instead of their own AI.
var (
	pevAnimalHolds   = map[*Animal]func(g *Game, a *Animal, dt float64) bool{}
	pevAnimalHoldLvl *Level
)

// pevAnimalHold is called by updateAnimals for every live animal: true = an event moved it this frame (its own AI
// is skipped). The function may return false to let the animal's AI take over (a cat that gets frightened).
func (g *Game) pevAnimalHold(a *Animal, dt float64) bool {
	if len(pevAnimalHolds) == 0 {
		return false
	}
	if pevAnimalHoldLvl != g.lvl {
		pevAnimalHolds, pevAnimalHoldLvl = map[*Animal]func(*Game, *Animal, float64) bool{}, g.lvl
		return false
	}
	f := pevAnimalHolds[a]
	return f != nil && f(g, a, dt)
}

func pevHoldAnimal(g *Game, a *Animal, f func(*Game, *Animal, float64) bool) {
	if pevAnimalHoldLvl != g.lvl {
		pevAnimalHolds, pevAnimalHoldLvl = map[*Animal]func(*Game, *Animal, float64) bool{}, g.lvl
	}
	pevAnimalHolds[a] = f
}

// pevFreeAnimal lets go of a held animal: it stands a moment and its own AI goes on.
func pevFreeAnimal(a *Animal) {
	if a == nil {
		return
	}
	if _, ok := pevAnimalHolds[a]; !ok {
		return
	}
	delete(pevAnimalHolds, a)
	if !a.dead && a.state != aFlee {
		a.state, a.timer, a.path = aIdle, 1+math.Mod(a.seed, 1.5), nil
	}
}

func pevAnimalHeld(a *Animal) bool { _, ok := pevAnimalHolds[a]; return ok }

// pevAnimalCalm: an animal that is not running away, not kicking and not dead.
func pevAnimalCalm(a *Animal) bool {
	return a != nil && !a.dead && !a.gone && a.state != aFlee && a.state != aWarn && a.state != aKick
}

// pevAlarmNear: somebody near p is alarmed, running or calling, or the police are close: no tender moments now.
// (A call made earlier in the party does not stop them for the rest of it: only what is near now.)
func (g *Game) pevAlarmNear(p Vec, r float64) bool {
	for _, c := range g.cops {
		if c != nil && dist(c.pos, p) < r+80 {
			return true
		}
	}
	for _, gu := range g.guests {
		if gu.out() || dist(gu.pos, p) > r {
			continue
		}
		switch gu.state {
		case gAlarm, gFlee, gScared, gToPhone, gCall, gReport, gRun:
			return true
		}
	}
	return false
}

// ---- the petting -------------------------------------------------------------------------------------------

func petKindOK(a *Animal) bool {
	return a != nil && (a.kind == animDog || a.kind == animCat || a.kind == animHorse)
}

// petDebugKind: -pevent caricia:perro|gato|caballo (-1 = any).
func petDebugKind() int {
	switch pevDemo("caricia") {
	case "perro", "dog":
		return animDog
	case "gato", "cat":
		return animCat
	case "caballo", "horse":
		return animHorse
	}
	return -1
}

type petLevel struct {
	lvl   *Level
	next  float64 // no petting before this game time (the pause)
	look  float64
	gCool map[*Guest]float64
	aCool map[*Animal]float64
	n     int // pettings in this party
	looks int
}

var petLv petLevel

const (
	ptWalk = iota
	ptPet
)

type petState struct {
	m     pevMem
	a     *Animal
	phase int
	t     float64
	dur   float64
	spot  Vec
	said  bool
	purr  bool
	// for the tests
	petted bool
}

func (s *petState) pevHas(gu *Guest) bool { return s.m.gu == gu }

func init() {
	registerPartyEvent(&partyEvent{ID: "caricia", Name: "Caricias a animales", Group: "caricia", Reactive: true, Minor: true,
		Prob: petChance, AtMin: 15, AtMax: 600, Dur: 15,
		Eligible: petDue, Start: petStart, Items: petItems,
		Update: pevGuard(petUpdate, func(g *Game, a *partyAct) { petEnd(g, a.D.(*petState)) })})
}

func petReset(g *Game) {
	r := petDice()
	petLv = petLevel{lvl: g.lvl, gCool: map[*Guest]float64{}, aCool: map[*Animal]float64{}}
	petLv.next = g.time + 15 + r.Float64()*25
	if pevDebugOn("caricia") {
		petLv.next = g.time + 2
	}
}

// petNearR2: how near the guest must be (the debug flag sends the nearest one from anywhere).
func petNearR2() float64 {
	if pevDebugOn("caricia") {
		return 600
	}
	return petNearR
}

// petPair: the nearest pair of a free guest and a calm pettable animal (nil, nil = none).
func (g *Game) petPair() (*Guest, *Animal) {
	var bg *Guest
	var ba *Animal
	bd := petNearR2()
	want := petDebugKind()
	for _, a := range g.animals {
		if !petKindOK(a) || !pevAnimalCalm(a) || pevAnimalHeld(a) || petLv.aCool[a] > g.time || want >= 0 && a.kind != want {
			continue
		}
		for _, gu := range g.guests {
			if petLv.gCool[gu] > g.time || !g.pevFreeGuest(gu) {
				continue
			}
			if d := dist(gu.pos, a.pos); d < bd {
				bg, ba, bd = gu, a, d
			}
		}
	}
	return bg, ba
}

// petDue: the condition of the reactive event (a pair near now, the pause over, the dice said yes).
func petDue(g *Game, t partyTraits) bool {
	if g.lvl == nil || len(g.animals) == 0 {
		return false
	}
	if testing.Testing() && !petInTests && !pevDebugOn("caricia") {
		return false
	}
	if petLv.lvl != g.lvl {
		petReset(g)
	}
	if g.time < petLv.next || g.time < petLv.look {
		return false
	}
	petLv.look = g.time + petLookGap
	gu, _ := g.petPair()
	if gu == nil {
		return false
	}
	petLv.looks++
	roll := petDice().Float64()
	return pevDebugOn("caricia") || roll < petChance
}

// petSpot: where the guest stands: at the animal's head (never behind a horse), on free ground.
func (g *Game) petSpot(a *Animal, from Vec) (Vec, bool) {
	l := g.lvl
	ok := func(p Vec) bool { return l.Free(p) && !l.SolidP(p.X, p.Y) && !l.InPool(p) && !l.badGround(p) }
	off := 13.0
	if a.kind == animHorse {
		off = 24
	}
	cands := []Vec{{a.pos.X + a.face*off, a.pos.Y + 2}, {a.pos.X + a.face*off, a.pos.Y - 4}}
	if a.kind != animHorse {
		cands = append(cands, Vec{a.pos.X - a.face*off, a.pos.Y + 2})
		if from.X < a.pos.X == (a.face > 0) { // the guest comes from behind: the near side first
			cands[0], cands[2] = cands[2], cands[0]
		}
	}
	for _, p := range cands {
		p = Vec{math.Round(p.X), math.Round(p.Y)}
		if ok(p) && l.PathG(from, p) != nil {
			return p, true
		}
	}
	return Vec{}, false
}

func petStart(g *Game, a *partyAct) bool {
	r := petDice()
	gu, an := g.petPair()
	petLv.next = g.time + 3 // if it cannot start now, ask again in a while
	if gu == nil {
		return false
	}
	sp, ok := g.petSpot(an, gu.pos)
	if !ok {
		petLv.aCool[an] = g.time + 20
		return false
	}
	s := &petState{a: an, spot: sp, dur: petMinT + r.Float64()*(petMaxT-petMinT)}
	s.m.gu = gu
	if !g.pevSendTo(&s.m, sp) {
		return false
	}
	pevHoldAnimal(g, an, s.holdAnimal)
	a.D = s
	if pevDebugOn("caricia") {
		fmt.Printf("PEVDEMO caricia kind %d at %.0f %.0f\n", an.kind, an.pos.X, an.pos.Y)
	}
	petLv.n++
	return true
}

// holdAnimal: the animal waits for the guest and lets itself be petted. false = its own fear wins (a cat, a horse).
func (s *petState) holdAnimal(g *Game, a *Animal, dt float64) bool {
	switch a.kind {
	case animCat:
		for _, p := range g.players() {
			if p != nil && !p.dead && !p.hidden && dist(p.pos, a.pos) < 30 {
				pevFreeAnimal(a)
				return false
			}
		}
		for _, o := range g.animals {
			if o.kind == animDog && !o.dead && dist(o.pos, a.pos) < 40 {
				pevFreeAnimal(a)
				return false
			}
		}
	case animHorse:
		for _, p := range g.players() {
			if p != nil && !p.dead && !p.hidden && a.behind(p.pos, 28, 16) {
				pevFreeAnimal(a)
				return false
			}
		}
	}
	a.path, a.moving, a.vert = nil, false, 0
	gu := s.m.gu
	if gu != nil && math.Abs(gu.pos.X-a.pos.X) > 3 && a.kind != animHorse {
		a.face = math.Copysign(1, gu.pos.X-a.pos.X) // it turns to the one who comes
	}
	switch {
	case a.kind == animHorse && s.phase == ptPet:
		a.state = aIdle // it nods: head down and up
		if int(s.t*1.4)%2 == 1 {
			a.state = aGraze
		}
	case a.kind == animHorse:
		a.state = aIdle
	case s.phase == ptPet && a.kind == animDog && int(s.t*2)%3 == 2:
		a.state = aIdle // the dog stands up a moment, wagging
	default:
		a.state = aGraze // sits
	}
	a.timer = 2
	return true
}

func petEnd(g *Game, s *petState) {
	if s == nil {
		return
	}
	r := petDice()
	if s.m.mine() {
		pevRelease(&s.m, r)
	} else {
		pevForget(&s.m)
	}
	pevFreeAnimal(s.a)
	if s.m.gu != nil {
		petLv.gCool[s.m.gu] = g.time + petGuestCool
	}
	petLv.aCool[s.a] = g.time + petAnimalCool
	petLv.next = g.time + petPauseMin + r.Float64()*petPauseVar
	if pevDebugOn("caricia") {
		petLv.next = g.time + 4
	}
}

func petUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*petState)
	r := petDice()
	an := s.a
	if !s.m.mine() || an.dead || an.gone || !pevAnimalHeld(an) || g.pevAlarmNear(an.pos, 120) {
		petEnd(g, s)
		return true
	}
	gu := s.m.gu
	s.t += dt
	switch s.phase {
	case ptWalk:
		if gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, s.spot) < 6 {
			s.phase, s.t = ptPet, 0
			if pevDebugOn("caricia") {
				fmt.Printf("PEVDEMO caricia pets at t=%.1f\n", g.time)
			}
			if an.kind == animHorse {
				s.m.hold(gIdle)
			} else {
				s.m.pose(AnimCrouch)
			}
			gu.dir = faceTo(gu.pos, an.pos)
			if gu.icon == "" {
				lines := petLinesDog
				switch an.kind {
				case animCat:
					lines = petLinesCat
				case animHorse:
					lines = petLinesHorse
				}
				gu.icon, gu.iconT = pevPick(r, lines), 2
			}
		} else if s.t > petWalkMax && !(pevDebugOn("caricia") && s.t < 40) {
			petEnd(g, s)
			return true
		}
	case ptPet:
		s.m.keep()
		s.petted = true
		gu.dir = faceTo(gu.pos, an.pos)
		if !s.purr && s.t > 1.2 {
			s.purr = true
			lines := petAnDog
			switch an.kind {
			case animCat:
				lines = petAnCat
			case animHorse:
				lines = petAnHorse
			}
			an.icon, an.iconT = pevPick(r, lines), 1.8
		}
		if s.t >= s.dur {
			petEnd(g, s)
			return true
		}
	}
	return false
}

// ---- drawing -------------------------------------------------------------------------------------------------

var (
	petHeartCol  = color.RGBA{0xf0, 0x4a, 0x78, 255}
	petHeartLite = color.RGBA{0xff, 0xb0, 0xc8, 255}
	petWagCol    = color.RGBA{0xf4, 0xf0, 0xe0, 255}
)

// petHeartPix: a 5x4 heart (top-left x, y).
func petHeartPix(plot plotFn, x, y, px int) {
	plot(x+px, y, px, px, petHeartCol)
	plot(x+3*px, y, px, px, petHeartCol)
	plot(x, y+px, 5*px, px, petHeartCol)
	plot(x+px, y+2*px, 3*px, px, petHeartCol)
	plot(x+2*px, y+3*px, px, px, petHeartCol)
	plot(x+px, y+px, px, px, petHeartLite)
}

// animalSize: the size of the animal's sprite (0, 0 when the sprites are not loaded).
func (g *Game) animalSize(a *Animal) (int, int) {
	if g.as == nil || g.as.animals == nil {
		return 0, 0
	}
	im := g.as.animals[animalKey(a.kind, a.coat, fStand)]
	if im == nil {
		return 0, 0
	}
	return im.Bounds().Dx(), im.Bounds().Dy()
}

// petHearts: three hearts that rise over the animal and start again (t in seconds).
func petHearts(plot plotFn, cx, top int, t float64) {
	for i := 0; i < 3; i++ {
		u := math.Mod(t*0.7+float64(i)/3, 1)
		x := cx + (i-1)*7 + int(math.Round(math.Sin(t*3+float64(i)*2)*1.5))
		y := top - int(math.Round(u*14))
		if u < 0.85 {
			petHeartPix(plot, x-2, y, 1)
		}
	}
}

func petItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s, ok := a.D.(*petState)
	if !ok || s.phase != ptPet || s.a == nil || s.a.dead {
		return items
	}
	an := s.a
	w, h := g.animalSize(an)
	if h == 0 {
		return items
	}
	plot := g.plotter(dst)
	items = append(items, drawItem{an.pos.Y + 0.4, func() {
		cx, cy := g.pevXY(an.pos)
		petHearts(plot, cx, cy-h-4, s.t)
		if an.kind == animDog { // the tail wags: two little strokes behind it, left and right in turn
			tx := cx - int(an.face)*(w/2+1)
			ty := cy - h/2 - 2
			if int(s.t*8)%2 == 0 {
				plot(tx-int(an.face)*2, ty-2, 1, 3, petWagCol)
			} else {
				plot(tx-int(an.face)*2, ty+2, 1, 3, petWagCol)
			}
		}
	}})
	return items
}
