package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"testing"
)

// lifeTestGame: a light game (level 1) with n idle guests spread over the roaming area, own dice seeded.
func lifeTestGame(t *testing.T, n int) *Game {
	t.Helper()
	oldR, oldS := lifeRng, lifeSeed
	lifeRng = rand.New(rand.NewSource(5))
	t.Cleanup(func() { lifeRng, lifeSeed = oldR, oldS })
	g := twoGame(false)
	g.guests, g.guards, g.cops, g.animals = nil, nil, nil, nil
	rr := rand.New(rand.NewSource(9))
	for i := 0; i < n; i++ {
		g.guests = append(g.guests, &Guest{cs: g.plainGuestCS(), pos: g.lvl.RandomRoam(rr), state: gIdle, timer: 30, style: i})
	}
	g.total = len(g.guests)
	g.levelIdx = 0
	g.time = 100
	g.lifeReset()
	g.life.contestAt, g.life.pizzaAt, g.life.selfieT = 0, 0, 1e9
	g.life.mascot = nil
	return g
}

func stepGuests(g *Game, secs float64) {
	for i := 0; i < int(secs*30); i++ {
		g.time += 1.0 / 30
		for _, gu := range g.guests {
			g.updateGuest(gu, 1.0/30)
		}
		g.updateLife(1.0 / 30)
	}
}

func TestSongChangeSendsSomeGuestsDancing(t *testing.T) {
	g := lifeTestGame(t, 40)
	if len(g.lvl.Def.Dance) == 0 {
		t.Fatal("level 1 must have a dance floor")
	}
	before := g.rng.Int63()
	g.rng = rand.New(rand.NewSource(11))
	g.lifeSongChanged()
	going := 0
	for _, gu := range g.guests {
		if gu.state == gGoto && gu.next == gDance {
			going++
		}
	}
	if going == 0 || going >= len(g.guests) || going > waveMax {
		t.Fatalf("some (not all, at most %d) guests must go dancing: %d of %d", waveMax, going, len(g.guests))
	}
	_ = before
	// own dice: g.rng is not moved by the wave
	a, b := rand.New(rand.NewSource(11)), g.rng
	if a.Int63() != b.Int63() {
		t.Fatal("the wave moved g.rng")
	}
	// a second change right away does not send more
	g.lifeSongChanged()
	again := 0
	for _, gu := range g.guests {
		if gu.state == gGoto && gu.next == gDance {
			again++
		}
	}
	if again != going {
		t.Fatalf("two waves less than %.0f s apart: %d -> %d", waveGap, going, again)
	}
	// and they arrive and dance
	stepGuests(g, 30)
	dancing := 0
	for _, gu := range g.guests {
		if gu.state == gDance {
			dancing++
		}
	}
	if dancing == 0 {
		t.Fatal("nobody reached the floor to dance")
	}
}

func TestDJBoothAliveAndOops(t *testing.T) {
	g := lifeTestGame(t, 20)
	g.djDead = false
	b := g.p.pos
	// cooldown: the second press does not change the song
	g.djBoothUse(b)
	if !g.skipSong {
		t.Fatal("E next to the booth must change the song")
	}
	g.skipSong = false
	g.djBoothUse(b)
	if g.skipSong {
		t.Fatal("the DJ must make the player wait between two songs")
	}
	// the mistake happens sometimes and plays an existing synthesised genre
	oops := 0
	for i := 0; i < 200; i++ {
		g.time += djCoolTime + 1
		g.life.oopsNew = -999
		g.djBoothUse(b)
		if g.life.oopsNew == g.time {
			oops++
		}
	}
	if oops == 0 || oops > 100 {
		t.Fatalf("the DJ must get it wrong sometimes, not always: %d of 200", oops)
	}
	sp := lifeOopsSpec()
	if _, ok := genreSongs[sp.style]; !ok || sp.bpm <= 0 {
		t.Fatalf("the mistake must be an existing genre: %+v", sp)
	}
	// the mistake only lasts one song
	g.life.oopsNew = g.time
	g.lifeOopsOnChange(true)
	if !g.life.oops {
		t.Fatal("the mistake must play")
	}
	g.lifeOopsOnChange(false)
	if g.life.oops {
		t.Fatal("the next song is normal again")
	}
	// dead DJ: nobody stops the player and there is no cooldown
	g.djDead = true
	g.skipSong = false
	g.djBoothUse(b)
	if !g.skipSong {
		t.Fatal("dead DJ: the song changes")
	}
}

func TestSelfies(t *testing.T) {
	g := lifeTestGame(t, 30)
	for _, gu := range g.guests { // group them so some selfies are shared
		gu.pos = g.lvl.RandomRoam(rand.New(rand.NewSource(int64(gu.style / 4))))
	}
	g.life.selfieT = 0
	seen, group := 0, false
	states := map[*Guest]gState{}
	for _, gu := range g.guests {
		states[gu] = gu.state
	}
	for i := 0; i < 30*400; i++ {
		g.time += 1.0 / 30
		g.updateLife(1.0 / 30)
		if len(g.life.selfies) > selfieMax {
			t.Fatalf("more than %d selfies at once", selfieMax)
		}
		for _, s := range g.life.selfies {
			if len(s.gs) > 1 {
				group = true
			}
			for _, gu := range s.gs {
				if gu.state != gIdle || gu.dir != DirDown {
					t.Fatal("a guest in a selfie keeps his state and faces the front")
				}
			}
		}
		if len(g.life.selfies) > 0 {
			seen++
		}
	}
	if seen == 0 {
		t.Fatal("no selfie in 400 s")
	}
	if seen > 30*400*3/4 {
		t.Fatalf("selfies all the time (%d frames)", seen)
	}
	if !group {
		t.Log("no group selfie in this run (random)")
	}
}

func TestSelfiePixels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 60, 70))
	plot := func(x, y, w, h int, c color.RGBA) {
		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				img.SetRGBA(x+i, y+j, c)
			}
		}
	}
	skin := rgb(0xe0b090)
	count := func() (n int) {
		for _, p := range img.Pix[3:] {
			_ = p
		}
		for y := 0; y < 70; y++ {
			for x := 0; x < 60; x++ {
				if img.RGBAAt(x, y).A != 0 {
					n++
				}
			}
		}
		return
	}
	drawSelfiePix(plot, 30, 60, 1, 0.1, skin)
	short := count()
	img = image.NewRGBA(img.Rect)
	drawSelfiePix(plot, 30, 60, 1, 1.0, skin)
	arm := count()
	img = image.NewRGBA(img.Rect)
	drawSelfiePix(plot, 30, 60, 1, selfieFlashAt+0.05, skin)
	flash := count()
	if !(short < arm && arm < flash) {
		t.Fatalf("arm grows and the flash adds light: %d %d %d", short, arm, flash)
	}
	if dir := os.Getenv("PH_LIFE_PNG"); dir != "" {
		sheet := image.NewRGBA(image.Rect(0, 0, 4*60, 70))
		img2 := func(k int, f func(p plotFn)) {
			f(func(x, y, w, h int, c color.RGBA) {
				for j := 0; j < h; j++ {
					for i := 0; i < w; i++ {
						sheet.SetRGBA(k*60+x+i, y+j, c)
					}
				}
			})
		}
		img2(0, func(p plotFn) { drawSelfiePix(p, 30, 60, 1, 1.0, skin) })
		img2(1, func(p plotFn) { drawSelfiePix(p, 30, 60, 1, selfieFlashAt+0.05, skin) })
		img2(2, func(p plotFn) { drawPizzaBoxPix(p, 30, 60, DirDown, 1); drawCrownPix(p, 30, 10) })
		img2(3, func(p plotFn) {
			drawMascotPix(p, "cat", 12, 30, false, 0, 1)
			drawMascotPix(p, "crab", 40, 30, false, 0, 1)
			drawMascotPix(p, "penguin", 12, 60, false, 0, 1)
			drawMascotPix(p, "duck", 40, 60, true, 0, 1)
		})
		f, err := os.Create(dir + "/life.png")
		if err == nil {
			big := image.NewRGBA(image.Rect(0, 0, sheet.Rect.Dx()*6, sheet.Rect.Dy()*6))
			for y := 0; y < big.Rect.Dy(); y++ {
				for x := 0; x < big.Rect.Dx(); x++ {
					c := sheet.RGBAAt(x/6, y/6)
					if c.A == 0 {
						c = rgb(0x3c4a3c)
					}
					big.SetRGBA(x, y, c)
				}
			}
			png.Encode(f, big)
			f.Close()
		}
	}
}

func TestDanceContest(t *testing.T) {
	g := lifeTestGame(t, 30)
	g.life.contestAt = g.time + 1
	pts := g.points
	rngBefore := rand.New(rand.NewSource(11))
	g.rng = rand.New(rand.NewSource(11))
	stepGuests(g, 3)
	if g.life.contest == nil {
		t.Fatal("the contest did not start")
	}
	n := len(g.life.contest.gs)
	if n < contestMinN-1 || n > contestMaxN {
		t.Fatalf("group size %d", n)
	}
	won, danced := false, false
	for i := 0; i < 30*60 && g.life.contest != nil; i++ {
		stepGuests(g, 1.0/30)
		if c := g.life.contest; c != nil {
			if c.phase == ctDance {
				danced = true
			}
			if c.phase == ctWin && c.winner != nil {
				won = true
			}
		}
	}
	if !danced || !won {
		t.Fatalf("the contest must have a dance (%v) and a winner (%v)", danced, won)
	}
	if g.life.contest != nil {
		t.Fatal("the contest must end")
	}
	for _, gu := range g.guests {
		if gu.out() {
			t.Fatal("nobody dies in the contest")
		}
	}
	if g.points != pts {
		t.Fatalf("the contest gives no points: %d -> %d", pts, g.points)
	}
	_ = rngBefore
}

func TestPizzaBoyHelped(t *testing.T) {
	g := lifeTestGame(t, 20)
	g.life.pizzaAt = g.time + 1
	alive := g.aliveCount()
	stepGuests(g, 3)
	e := g.life.pizza
	if e == nil {
		t.Fatal("the pizza boy did not come in")
	}
	if pizzaLine(e, false) != "¿Es aquí el 14B?" && pizzaLine(e, false) != "Pizza para el 14B... ¿alguien?" {
		t.Fatalf("his line: %q", pizzaLine(e, false))
	}
	if g.aliveCount() != alive || len(g.animals) != 0 {
		t.Fatal("he is not a guest and not an animal")
	}
	g.perkStamina, g.p.stamina = 6, 0
	g.p.pos = Vec{e.pos.X + 8, e.pos.Y}
	pts := g.points
	if !g.lifeInteract(g.p) {
		t.Fatal("E next to him must help")
	}
	if g.p.stamina < g.perkStamina || g.p.stamina <= 0 || g.points != pts+pizzaStamina {
		t.Fatalf("the pizza gives stamina (%v) and %d points (%d -> %d)", g.p.stamina, pizzaStamina, pts, g.points)
	}
	entry := e.entry
	for i := 0; i < 30*60 && g.life.pizza != nil; i++ {
		g.time += 1.0 / 30
		g.updateLife(1.0 / 30)
	}
	if g.life.pizza != nil {
		t.Fatal("he must leave")
	}
	if dist(e.pos, entry) > 2 {
		t.Fatalf("he leaves by where he came: %v vs %v", e.pos, entry)
	}
}

func TestPizzaBoyIgnoredLeavesAndKilledIsRemembered(t *testing.T) {
	g := lifeTestGame(t, 20)
	g.life.pizzaAt = g.time + 1
	g.p.pos = Vec{5, 5}
	for i := 0; i < 30*100 && !(g.life.pizza == nil && g.time > 110); i++ {
		g.time += 1.0 / 30
		g.updateLife(1.0 / 30)
	}
	if g.life.pizza != nil || g.life.pizzaDead {
		t.Fatal("ignored: he leaves on his own, alive")
	}
	// now a second one that is killed
	g.life.pizzaAt = g.time + 1
	stepGuests(g, 3)
	e := g.life.pizza
	if e == nil {
		t.Fatal("no second pizza boy")
	}
	g.mode = ModePlay
	g.p.pos = Vec{e.pos.X + 6, e.pos.Y}
	nb := len(g.bodies)
	if !g.lifeStab(g.p, 24) {
		t.Fatal("the blow must reach him")
	}
	if !g.life.pizzaDead || g.life.pizza != nil || len(g.bodies) != nb+1 || g.bodies[nb].label != "Repartidor" {
		t.Fatal("killed: a body and the fact for the newspaper (pizzaDead)")
	}
	if g.run.nonGuest != 1 {
		t.Fatalf("it is a kill that is not a guest: %d", g.run.nonGuest)
	}
}

func TestMascot(t *testing.T) {
	g := lifeTestGame(t, 20)
	g.mascotSpawn(lifeDice())
	m := g.life.mascot
	if m == nil {
		t.Fatal("no mascot")
	}
	if len(g.animals) != 0 || g.aliveCount() != 20 {
		t.Fatal("the mascot is not an animal (animalMax) nor a guest")
	}
	capBefore := len(g.animals)
	g.capAnimals()
	if len(g.animals) != capBefore {
		t.Fatal("capAnimals")
	}
	pts := g.points
	if g.lifeFinish() != mascotBonus || g.points != pts+mascotBonus {
		t.Fatal("alive at the end: bonus")
	}
	// petting: once
	g.p.pos = Vec{m.pos.X + 5, m.pos.Y}
	pts = g.points
	g.lifeInteract(g.p)
	g.lifeInteract(g.p)
	if g.points != pts+mascotPet {
		t.Fatalf("petting gives %d once: %d", mascotPet, g.points-pts)
	}
	// it wanders
	start := m.pos
	moved := false
	for i := 0; i < 30*120; i++ {
		g.time += 1.0 / 30
		g.lifeMascot(1.0 / 30)
		if dist(m.pos, start) > 5 {
			moved = true
		}
		if !g.lvl.Free(m.pos) && i > 0 {
			t.Fatalf("it walked into a wall at %v", m.pos)
		}
	}
	if !moved {
		t.Fatal("the mascot never moves")
	}
	// dies: nothing
	g.p.pos = Vec{m.pos.X + 5, m.pos.Y}
	pts = g.points
	if !g.lifeStab(g.p, 24) || !m.dead {
		t.Fatal("the blow kills the mascot")
	}
	if g.points != pts || g.lifeFinish() != 0 || g.aliveCount() != 20 {
		t.Fatal("dead mascot: no points, no bonus, the guests are still 20")
	}
	// the kinds
	for title, want := range map[string]string{"PLAYA": "crab", "ESTACIÓN DE ESQUÍ": "penguin", "PARQUE DE ATRACCIONES": "duck", "CASINO": "cat", "": "cat"} {
		if mascotKindFor(title) != want {
			t.Errorf("%q -> %q, want %q", title, mascotKindFor(title), want)
		}
	}
	for k := range mascotRows {
		if mascotPal[k] == nil || mascotPetLine[k] == "" {
			t.Errorf("%s incomplete", k)
		}
	}
}

func TestLifeDoesNotRunInStealth(t *testing.T) {
	g := lifeTestGame(t, 10)
	g.stealth = true
	g.lifeReset()
	if g.life.mascot != nil || g.life.pizzaAt != 0 || g.life.contestAt != 0 || g.lifeFinish() != 0 {
		t.Fatal("STEALTH mode is left alone")
	}
}

func TestLifeEnglish(t *testing.T) {
	for _, s := range []string{"¡PATATA!", "¿QUÉ ES ESTO?", "¿Es aquí el 14B?", "Pizza para el 14B... ¿alguien?", "¡Gracias! Toma, invita la casa",
		"Mejor llamo al cliente...", "¡Concurso de baile en la pista!", "La mascota de la fiesta ya no está", "Repartidor", "¡HE GANADO!",
		"Uy: el DJ se equivoca de disco", "El DJ está liado con los discos", "Le indicas el camino y te regala una pizza"} {
		if len(s) > 48 {
			t.Errorf("%q is longer than 48", s)
		}
		if _, ok := langDataEN[s]; !ok {
			t.Errorf("no English for %q", s)
		}
	}
	for _, s := range mascotPetLine {
		if _, ok := langDataEN[s]; !ok {
			t.Errorf("no English for %q", s)
		}
	}
}

// The whole thing is drawn in the real world (headless): a selfie with flash, the crown, the pizza boy with his
// bubble and the mascot, and a real loadLevel rolls the dice without touching g.rng.
func TestLifeDrawsInTheWorld(t *testing.T) {
	g := layerGame(t, false)
	pm := twoPlayerMode
	t.Cleanup(func() { twoPlayerMode = pm })
	auditReset(g)
	twoPlayerMode = false
	oldR, oldS := lifeRng, lifeSeed
	lifeRng = rand.New(rand.NewSource(3))
	t.Cleanup(func() { lifeRng, lifeSeed = oldR, oldS })
	g.loadLevel(0)
	g.lvl.graded = true
	g.titleT = 0
	g.mode = ModePlay
	if g.life.mascot == nil || len(g.animals) > animalMax {
		t.Fatal("a real level has its mascot and respects animalMax")
	}
	g.life.pizzaAt = 0.01
	g.time = 5
	for i := 0; i < 5; i++ {
		g.time += 0.05
		g.updateLife(0.05)
	}
	if g.life.pizza == nil {
		t.Fatal("the pizza boy came in")
	}
	lead := g.guests[0]
	lead.state, lead.timer = gIdle, 30
	g.life.selfies = []selfie{{gs: []*Guest{lead}, t0: g.time - selfieFlashAt - 0.05}}
	g.life.contest = &danceContest{gs: []*Guest{g.guests[1], g.guests[2], g.guests[3]}, phase: ctWin, winner: g.guests[1], t0: g.time}
	img := ebiten.NewImage(screenW, screenH)
	g.Draw(img)
	img.Deallocate()
}
