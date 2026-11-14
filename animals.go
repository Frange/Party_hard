package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Animals: they wander and act on their own. A horse kicks (and kills) anyone who lingers behind it.
// ---------------------------------------------------------------------------

const (
	animHorse = iota
	animHen
	animDog
)

var animalKinds = []string{"horse", "hen", "dog"}

func animalKindIndex(s string) int {
	for i, k := range animalKinds {
		if k == s {
			return i
		}
	}
	return animHen
}

type AnimalSpawn struct {
	Kind string
	Pos  Vec
	Area Rect
}

const (
	aIdle = iota
	aWalk
	aGraze
	aWarn
	aKick
	aFlee
)

type Animal struct {
	kind   int
	pos    Vec
	face   float64 // -1 left, +1 right
	state  int
	timer  float64
	path   []Vec
	anim   float64
	area   Rect
	moving bool
	warnT  float64
	victim any
	hit    bool
	icon   string
	iconT  float64
	seed   float64
	coat   int
	sprayT float64
	repath float64
	dead   bool
	label  string  // proper name shown on hover (Woof)
	home   bool    // placed by the level (stable, paddock): keeps to its own area
	vert   float64 // -1 walking up the screen, +1 down, 0 sideways or standing (the horse then shows its back or its front)
	gasT   float64 // seconds spent in a gas cloud (trapany.go)
	bites  int     // wild hunters (animals6.go): kills so far (a crocodile or a big cat never makes more than one)
	huntT  float64 // big cats: seconds until their one hunt; crocodile: rest after a snap
	gone   bool    // left the party (a big cat after its hunt): removed at the end of updateAnimals
}

// frames of each animal sprite
const (
	fStand = iota
	fWalk0 // walk frames 0..3
	fWalk1
	fWalk2
	fWalk3
	fGraze
	fWarn
	fKick
	fSit
	fPeck0
	fPeck1
)

// AnimalSprites holds every frame, keyed by animalKey(kind, coat, frame).
type AnimalSprites map[int]*ebiten.Image

func animalKey(kind, coat, frame int) int { return kind<<16 | coat<<8 | frame }

type coatCols struct{ body, dark, light, mane c4 }
type c4 = color.RGBA

var horseCoats = []coatCols{
	{hx(0x7a4a2a), hx(0x3a2010), hx(0x9a6a40), hx(0x1e1008)}, // bay
	{hx(0x2a2024), hx(0x100c10), hx(0x463a40), hx(0x0a0608)}, // black
	{hx(0xd8d4cc), hx(0x8a867e), hx(0xf4f2ec), hx(0xa8a49c)}, // grey
}

// leg draws a two-bone leg with a hoof; kneeFwd selects the direction of the joint.
func horseLeg(b *pixBuf, hx0, hy0, fx, fy float64, kneeFwd float64, upper, lower int, col, dark c4) {
	mx, my, ex, ey := ik2(hx0, hy0, fx, fy, 11, kneeFwd, 0.2)
	thickLine(b, hx0, hy0, mx, my, upper, col)
	thickLine(b, mx, my, ex, ey, lower, shade(col, 0.92))
	b.rect(int(ex)-1, int(ey)-1, lower+2, 3, c(0x1a1410))
	_ = dark
}

func genHorseFrame(frame, coat int) *pixBuf {
	b := newPix(80, 60)
	b.ox, b.oy = 2, 2
	cc := horseCoats[coat%len(horseCoats)]
	body, dark, light, mane := cc.body, cc.dark, cc.light, cc.mane
	const ground = 55.0
	// walk phases: x offsets for front-far, front-near, hind-far, hind-near feet and lifts
	var dx, lift [4]float64
	if frame >= fWalk0 && frame <= fWalk3 {
		ph := float64(frame-fWalk0) / 4 * 2 * math.Pi
		for i := 0; i < 4; i++ {
			p := ph + float64(i)*math.Pi/2*[4]float64{1, 3, 2, 0}[i]/float64(1+i%2)
			p = ph + [4]float64{0, math.Pi, math.Pi / 2, 3 * math.Pi / 2}[i]
			dx[i] = 7 * math.Sin(p)
			lift[i] = 5 * math.Max(0, math.Cos(p))
		}
	}
	kick := frame == fKick
	tilt := 0.0
	if kick {
		tilt = 3
	}
	bodyY := 27.0 + tilt*0.4
	shX, hiX := 50.0, 24.0
	// tail first (behind everything)
	for k := -1; k <= 1; k++ {
		b.line(14, int(bodyY)-8, 8+k, int(bodyY)+10, mane)
		b.line(15, int(bodyY)-8, 9+k, int(bodyY)+12, mane)
	}
	if kick {
		b.line(14, int(bodyY)-8, 4, int(bodyY)-14, mane)
	}
	// far legs
	horseLeg(b, shX-3, bodyY+6, shX-3+dx[0]+2, ground-lift[0], 1, 4, 3, shade(body, 0.72), dark)
	if kick {
		horseLeg(b, hiX-2, bodyY+6, 3, bodyY-1, -1, 4, 3, shade(body, 0.72), dark)
	} else {
		horseLeg(b, hiX-2, bodyY+6, hiX-2+dx[2]-1, ground-lift[2], -1, 4, 3, shade(body, 0.72), dark)
	}
	// body
	b.ellipse(36, bodyY, 20, 10, body)
	b.ellipse(50, bodyY-1, 8, 10.5, body)
	b.ellipse(23, bodyY-1, 9, 10.5, body)
	b.ellipse(36, bodyY-6, 17, 4, light)
	b.ellipse(36, bodyY+8, 17, 2.5, shade(body, 0.8))
	b.ellipse(24, bodyY-4, 5, 4, light)
	// neck and head
	hx0, hy0 := 64.0, 10.0
	switch frame {
	case fGraze:
		hx0, hy0 = 66, 43
	case fWarn:
		hx0, hy0 = 66, 8
	}
	thickLine(b, 51, bodyY-9, hx0-4, hy0+2, 8, body)
	thickLine(b, 50, bodyY-10, hx0-5, hy0-2, 3, mane)
	thickLine(b, 52, bodyY-8, hx0-3, hy0+4, 2, light)
	hd := 0.0
	if frame == fGraze {
		hd = 1
	}
	b.ellipse(hx0, hy0+hd*2, 6.5, 4, body)
	b.ellipse(hx0+5, hy0+2+hd*2, 4, 3, light)
	b.rect(int(hx0)+7, int(hy0)+2+int(hd*2), 1, 1, c(0x14100c))
	b.rect(int(hx0)+1, int(hy0)-1+int(hd*2), 1, 1, c(0x14100c))
	b.rect(int(hx0)+1, int(hy0)-2+int(hd*2), 1, 1, c(0xf0f0e0))
	if frame == fWarn {
		b.rect(int(hx0)-4, int(hy0)-4, 3, 2, mane) // ears flat back
	} else {
		b.rect(int(hx0)-2, int(hy0)-6+int(hd*2), 2, 3, mane)
		b.rect(int(hx0), int(hy0)-6+int(hd*2), 2, 3, mane)
	}
	// near legs
	horseLeg(b, shX+1, bodyY+6, shX+1+dx[1], ground-lift[1], 1, 4, 3, body, dark)
	if kick {
		horseLeg(b, hiX+2, bodyY+6, 8, bodyY-4, -1, 4, 3, body, dark)
	} else {
		horseLeg(b, hiX+2, bodyY+6, hiX+2+dx[3], ground-lift[3], -1, 4, 3, body, dark)
	}
	return b
}

var henCoats = []coatCols{
	{hx(0xf4f0e8), hx(0xc8c0b0), hx(0xffffff), hx(0xd83030)},
	{hx(0xa0602a), hx(0x6a3a14), hx(0xc88040), hx(0xd83030)},
	{hx(0x3a3a40), hx(0x1a1a20), hx(0x58585e), hx(0xd83030)},
}

func genHenFrame(frame, coat int) *pixBuf {
	b := newPix(26, 22)
	b.ox, b.oy = 1, 1
	cc := henCoats[coat%len(henCoats)]
	body, dark, light, comb := cc.body, cc.dark, cc.light, cc.mane
	_ = dark
	bob := 0
	hx0, hy0 := 17, 6
	if frame == fPeck0 || frame == fPeck1 {
		hx0, hy0 = 19, 12+(frame-fPeck0)*2
	} else if frame >= fWalk0 && frame <= fWalk3 {
		bob = (frame - fWalk0) % 2
		hx0 += (frame - fWalk0) % 2
	}
	// tail feathers
	b.line(5, 11, 1, 5, dark)
	b.line(6, 11, 2, 4, body)
	b.line(6, 12, 1, 7, dark)
	b.ellipse(11, 12+float64(bob), 8, 5.5, body)
	b.ellipse(12, 10+float64(bob), 5, 3, light)
	b.ellipse(8, 13+float64(bob), 4, 3, dark)
	// neck + head
	thickLine(b, 16, 10+float64(bob), float64(hx0), float64(hy0)+1, 3, body)
	b.ellipse(float64(hx0)+1, float64(hy0), 3, 2.6, body)
	b.rect(hx0, hy0-3, 3, 2, comb)
	b.set(hx0+3, hy0-1, comb)
	b.rect(hx0+3, hy0, 2, 1, c(0xf0b020))
	b.set(hx0+1, hy0-1, c(0x14100c))
	b.rect(hx0+2, hy0+1, 1, 2, comb)
	// legs
	o := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		o = []int{3, 0, -3, 0}[frame-fWalk0]
	}
	for _, lx := range []int{9, 13} {
		off := o
		if lx == 13 {
			off = -o
		}
		b.line(lx, 16+bob, lx+off, 20, c(0xe8a820))
		b.rect(lx+off-1, 20, 3, 1, c(0xe8a820))
	}
	return b
}

var dogCoats = []coatCols{
	{hx(0xb88a50), hx(0x6a4a28), hx(0xd8aa70), hx(0x3a2a18)},
	{hx(0x2e2a30), hx(0x141216), hx(0x4a444c), hx(0x0c0a0c)},
	{hx(0xe8e4dc), hx(0x8a6a4a), hx(0xffffff), hx(0x4a3a2a)},
}

func genDogFrame(frame, coat int) *pixBuf {
	b := newPix(46, 30)
	b.ox, b.oy = 1, 1
	cc := dogCoats[coat%len(dogCoats)]
	body, light, ear := cc.body, cc.light, cc.mane
	sit := frame == fSit
	wag := 0
	if frame == fWalk1 || frame == fWalk3 || frame == fStand {
		wag = 1
	}
	by := 17.0
	if sit {
		by = 20
	}
	// tail
	if sit {
		b.line(9, 22, 4, 24+wag, body)
	} else {
		b.line(9, 14, 3, 9+wag*3, body)
		b.line(9, 15, 3, 10+wag*3, body)
	}
	var d [4]float64
	var lf [4]float64
	if frame >= fWalk0 && frame <= fWalk3 {
		ph := float64(frame-fWalk0) / 4 * 2 * math.Pi
		for i := 0; i < 4; i++ {
			p := ph + [4]float64{0, math.Pi, math.Pi, 0}[i]
			d[i] = 4.5 * math.Sin(p)
			lf[i] = 3 * math.Max(0, math.Cos(p))
		}
	}
	leg := func(x float64, off, lift float64, col c4, fore bool) {
		if sit && !fore {
			b.ellipse(x, 25, 4, 3, col)
			return
		}
		fx, fy := x+off, 28-lift
		mx, my, ex, ey := ik2(x, by+3, fx, fy, 5.5, 0.4, 0.2)
		thickLine(b, x, by+3, mx, my, 2, col)
		thickLine(b, mx, my, ex, ey, 2, shade(col, 0.92))
		b.rect(int(ex)-1, int(ey), 4, 2, shade(col, 0.7))
	}
	leg(11, d[2], lf[2], shade(body, 0.75), false)
	leg(28, d[0], lf[0], shade(body, 0.75), true)
	b.ellipse(19, by, 11, 5.5, body)
	b.ellipse(21, by-2, 8, 3, light)
	b.ellipse(27, by+1, 6, 5, body)
	hx0, hy0 := 34.0, 9.0
	if sit {
		hx0, hy0 = 33, 6
	}
	if sit {
		b.ellipse(30, 13, 4.5, 6.5, body)
	}
	b.ellipse(hx0, hy0, 4.5, 4, body)
	b.ellipse(hx0+5, hy0+2, 4, 2.6, light)
	b.rect(int(hx0)+8, int(hy0)+1, 2, 2, c(0x14100c))
	b.rect(int(hx0)+1, int(hy0)-1, 1, 1, c(0x14100c))
	b.ellipse(hx0-2, hy0-2, 2, 3.5, ear)
	leg(10, d[3], lf[3], body, false)
	leg(27, d[1], lf[1], body, true)
	return b
}

func buildAnimalSprites() AnimalSprites {
	out := AnimalSprites{}
	for coat := 0; coat < 3; coat++ {
		for f := fStand; f <= fKick; f++ {
			out[animalKey(animHorse, coat, f)] = genHorseFrame(f, coat).scaled(0.7).outlinedSel(0.4).img()
		}
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fPeck0, fPeck1} {
			out[animalKey(animHen, coat, f)] = genHenFrame(f, coat).outlinedSel(0.4).img()
		}
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fSit} {
			out[animalKey(animDog, coat, f)] = genDogFrame(f, coat).scaled(0.5).outlinedSel(0.4).img()
		}
	}
	buildMoreAnimalSprites(out)
	buildCritterSprites(out)
	buildRiderSprites(out)
	buildSnakeSprites(out)
	buildWildSprites(out)
	return out
}

// ---------------------------------------------------------------------------
// behaviour
// ---------------------------------------------------------------------------

func (g *Game) spawnAnimals() {
	g.animals = nil
	for _, sp := range g.lvl.Def.Animals {
		k := animalKindIndex(sp.Kind)
		if !animalAllowed(k) {
			continue
		}
		a := &Animal{kind: k, pos: g.lvl.NearestFree(sp.Pos), face: -1, area: sp.Area, seed: g.rng.Float64() * 10, timer: g.rng.Float64() * 3, coat: g.rng.Intn(3), home: true}
		if g.rng.Intn(2) == 0 {
			a.face = 1
		}
		g.animals = append(g.animals, a)
	}
	g.scatterWild() // animals6.go: frogs, squirrels, boars, crocodiles and (rarely) a big cat; own dice
	g.scatterCritters()
	g.scatterRiders()
	g.scatterSnake()
	g.capAnimals()
}

func (a *Animal) behind(p Vec, reach, width float64) bool {
	dx, dy := p.X-a.pos.X, p.Y-a.pos.Y
	return dx*a.face < -4 && math.Abs(dx) < reach && math.Abs(dy) < width
}

func (g *Game) updateAnimals(dt float64) {
	for _, a := range g.animals {
		if a.dead {
			continue
		}
		a.moving = false
		a.anim += dt
		if a.iconT > 0 {
			a.iconT -= dt
		}
		if g.pevAnimalHold(a, dt) { // petted or fetching a ball (pe_petting.go, pe_fetch.go)
			continue
		}
		restore := g.focus(a.pos) // two players: it reacts to the nearest one
		switch a.kind {
		case animHorse:
			g.updateHorse(a, dt)
		case animHen:
			g.updateHen(a, dt)
		case animDog:
			g.updateDog(a, dt)
		case animCat:
			g.updateCat(a, dt)
		case animBull:
			g.updateBull(a, dt)
		case animElephant:
			g.updateElephant(a, dt)
		case animGiraffe:
			g.updateGiraffe(a, dt)
		case animKangaroo:
			g.updateKangaroo(a, dt)
		case animChick:
			g.updateChick(a, dt)
		case animRabbit, animPig, animDuck, animBudgie, animDeer:
			g.updateCritter(a, dt)
		case animQuijote, animSancho:
			g.animalWander(a, 13, dt, 0.15)
		case animSnake:
			g.updateSnake(a, dt)
		case animFrog, animSquirrel, animBoar, animCobaya:
			g.updateCritter(a, dt)
		case animCapibara:
			g.updateCapibara(a, dt)
		case animCroc:
			g.updateCroc(a, dt)
		case animCheetah, animLion:
			g.updateBigCat(a, dt)
		}
		restore()
	}
	g.dropGoneAnimals()
}

func (g *Game) animalWalk(a *Animal, speed, dt float64) bool {
	prev, prevY := a.pos.X, a.pos.Y
	_, arr := follow(&a.pos, new(Dir), &a.path, speed, dt)
	if dy, dx := a.pos.Y-prevY, a.pos.X-prev; math.Abs(dy) > math.Abs(dx)*1.4 && math.Abs(dy) > 1e-6 {
		a.vert = math.Copysign(1, dy)
	} else {
		a.vert = 0
	}
	if a.pos.Y != prevY {
		a.moving = true
	}
	if a.pos.X != prev {
		a.moving = true
		if a.pos.X < prev {
			a.face = -1
		} else {
			a.face = 1
		}
	}
	return arr
}

func (g *Game) wanderTarget(a *Animal) Vec {
	if aggressiveAnimal(a.kind) {
		return g.lvl.RandomRoam(g.rng)
	}
	if !a.home {
		for i := 0; i < 12; i++ {
			if p := g.lvl.RandomIn(g.rng, a.area); g.lvl.Outdoor(p) {
				return p
			}
		}
		return a.pos
	}
	return g.lvl.RandomIn(g.rng, a.area)
}

// whoIsBehind finds someone standing behind the animal.
func (g *Game) whoIsBehind(a *Animal, reach, width float64) (any, Vec) {
	if p := g.p; !p.dead && !p.hidden && a.behind(p.pos, reach, width) {
		return p, p.pos
	}
	for _, gu := range g.guests {
		if !gu.out() && gu.state != gSleep && a.behind(gu.pos, reach, width) {
			return gu, gu.pos
		}
	}
	for _, gd := range g.guards {
		if !gd.dead && gd.vip == 0 && a.behind(gd.pos, reach, width) { // never the VIP zone staff (vipshape.go)
			return gd, gd.pos
		}
	}
	return nil, Vec{}
}

func (g *Game) updateHorse(a *Animal, dt float64) {
	switch a.state {
	case aIdle, aGraze:
		a.timer -= dt
		if v, _ := g.whoIsBehind(a, 28, 16); v != nil {
			a.warnT += dt
			if a.warnT > 1.1 {
				a.state, a.timer, a.victim, a.hit = aWarn, 0.7, v, false
				a.warnT = 0
			}
		} else {
			a.warnT = math.Max(0, a.warnT-dt)
		}
		if a.timer <= 0 && a.state != aWarn {
			if g.rng.Float64() < 0.5 {
				a.path = g.lvl.Path(a.pos, g.wanderTarget(a))
				a.state, a.timer = aWalk, 20
			} else if a.state == aIdle {
				a.state, a.timer = aGraze, 3+g.rng.Float64()*4
			} else {
				a.state, a.timer = aIdle, 2+g.rng.Float64()*3
			}
		}
	case aWalk:
		a.timer -= dt
		if g.animalWalk(a, 20, dt) || a.timer <= 0 {
			a.state, a.timer = aIdle, 2+g.rng.Float64()*3
		}
		if v, _ := g.whoIsBehind(a, 28, 16); v != nil {
			a.warnT += dt
			if a.warnT > 1.1 {
				a.state, a.timer, a.victim, a.hit = aWarn, 0.7, v, false
				a.warnT = 0
				a.path = nil
			}
		}
	case aWarn:
		a.timer -= dt
		if a.timer <= 0 {
			if v, _ := g.whoIsBehind(a, 32, 18); v != nil && g.rng.Float64() < 0.8 {
				a.state, a.timer, a.victim, a.hit = aKick, 0.5, v, false
			} else {
				a.state, a.timer = aIdle, 2
			}
		}
	case aKick:
		a.timer -= dt
		if !a.hit && a.timer < 0.3 {
			a.hit = true
			g.snd.Play("thud")
			g.burst(Vec{a.pos.X - a.face*19, a.pos.Y - 12}, 8, 70, rgb(0xd8d0c0), 0.4, 2, 100)
			switch v := a.victim.(type) {
			case *Player:
				if !v.dead && dist(v.pos, a.pos) < 38 {
					g.die("Un caballo te ha dado una coz mortal")
				}
			case *Guest:
				if !v.out() && dist(v.pos, a.pos) < 38 {
					g.killGuest(v, false)
					g.sayNews("kill", map[string]string{"actor": "Un caballo", "victim": guestLabel(v)})
				}
			case *Guard:
				if !v.dead && dist(v.pos, a.pos) < 38 {
					v.dead = true
					g.points += 50
					g.spawnBody(v.pos, v.cs, v.dir)
					g.bodies[len(g.bodies)-1].label = "Guardia " + g.personalName(v, v.cs)
					g.checkDone()
				}
			}
		}
		if a.timer <= 0 {
			a.state, a.timer = aIdle, 3
		}
	}
}

func (g *Game) updateHen(a *Animal, dt float64) {
	// scatter when someone comes close
	scare := dist(g.p.pos, a.pos) < 30 && !g.p.hidden
	if !scare {
		for _, gu := range g.guests {
			if !gu.out() && gu.moving && dist(gu.pos, a.pos) < 24 {
				scare = true
				break
			}
		}
	}
	if scare && a.state != aFlee {
		a.state, a.timer = aFlee, 1.4
		a.path = g.lvl.Path(a.pos, g.lvl.RandomIn(g.rng, a.area))
		a.icon, a.iconT = "BAWK!", 1
	}
	switch a.state {
	case aFlee:
		a.timer -= dt
		if g.animalWalk(a, 46, dt) || a.timer <= 0 {
			a.state, a.timer = aIdle, 1
		}
	case aIdle, aGraze:
		a.timer -= dt
		if a.timer <= 0 {
			switch r := g.rng.Float64(); {
			case r < 0.45:
				a.state, a.timer = aGraze, 1.5+g.rng.Float64()*2
			case r < 0.8:
				a.path = g.lvl.Path(a.pos, g.lvl.RandomIn(g.rng, Rect{a.pos.X - 40, a.pos.Y - 30, 80, 60}.clip(a.area)))
				a.state, a.timer = aWalk, 8
			default:
				a.state, a.timer = aIdle, 1+g.rng.Float64()*2
			}
		}
	case aWalk:
		a.timer -= dt
		if g.animalWalk(a, 14, dt) || a.timer <= 0 {
			a.state, a.timer = aIdle, 0.5
		}
	}
}

func (r Rect) clip(o Rect) Rect {
	x0, y0 := math.Max(r.X, o.X), math.Max(r.Y, o.Y)
	x1, y1 := math.Min(r.X+r.W, o.X+o.W), math.Min(r.Y+r.H, o.Y+o.H)
	if x1 <= x0 || y1 <= y0 {
		return o
	}
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

func (g *Game) updateDog(a *Animal, dt float64) {
	d := dist(g.p.pos, a.pos)
	if d < 55 && !g.p.hidden && a.iconT <= 0 && g.rng.Float64() < 0.02 {
		a.icon, a.iconT = "WOOF!", 1.4
	}
	switch a.state {
	case aIdle, aGraze:
		a.timer -= dt
		if a.timer <= 0 {
			if g.rng.Float64() < 0.65 {
				a.path = g.lvl.Path(a.pos, g.wanderTarget(a))
				a.state, a.timer = aWalk, 20
			} else {
				a.state, a.timer = aGraze, 2+g.rng.Float64()*3 // sits
			}
		}
	case aWalk:
		a.timer -= dt
		if g.animalWalk(a, 34, dt) || a.timer <= 0 {
			a.state, a.timer = aIdle, 1+g.rng.Float64()*2
		}
	}
}

// ---------------------------------------------------------------------------
// drawing
// ---------------------------------------------------------------------------

func (g *Game) animalFrame(a *Animal) int {
	if a.kind == animSnake {
		return snakeFrame(a)
	}
	if a.kind > animSnake {
		return wildFrame(a)
	}
	if a.kind >= animRabbit {
		return critterFrame(a)
	}
	if a.kind >= animCat {
		return g.animalFrameMore(a)
	}
	switch a.kind {
	case animHorse:
		switch a.state {
		case aGraze:
			return fGraze
		case aWarn:
			return fWarn
		case aKick:
			return fKick
		case aWalk:
			return fWalk0 + int(a.anim*6)%4
		}
		return fStand
	case animHen:
		switch {
		case a.state == aGraze:
			return fPeck0 + int(a.anim*4)%2
		case a.moving:
			return fWalk0 + int(a.anim*10)%4
		}
		return fStand
	default:
		switch {
		case a.state == aGraze:
			return fSit
		case a.moving:
			return fWalk0 + int(a.anim*8)%4
		}
		return fStand
	}
}

func (g *Game) drawAnimal(dst *ebiten.Image, a *Animal) {
	if a.dead {
		g.drawDeadAnimal(dst, a)
		return
	}
	im := g.as.animals[animalKey(a.kind, a.coat, g.animalFrame(a))]
	if im == nil {
		im = g.as.animals[animalKey(a.kind, a.coat, fStand)]
	}
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	x, y := g.wx(a.pos.X), g.wy(a.pos.Y)
	fillShadow(dst, x, y)
	if a.vert != 0 && a.moving && animalHasFreeView(a.kind) {
		// walking up or down the screen: its back or its chest, not the profile
		vp := animalFreeView(a)
		bob := math.Round(math.Sin(a.anim * 9))
		vop := &ebiten.DrawImageOptions{}
		vop.GeoM.Translate(math.Round(x-vp.ox), math.Round(y-vp.oy+2+bob))
		dst.DrawImage(vp.img, vop)
		if a.iconT > 0 && a.icon != "" {
			g.bubble(a, bubFiller, Vec{a.pos.X, a.pos.Y - 4}, a.icon, false, -h*0.2)
		}
		return
	}
	op := &ebiten.DrawImageOptions{}
	// sprites face right
	if a.face < 0 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(w, 0)
	}
	op.GeoM.Translate(math.Round(x-w/2), math.Round(y-h+2))
	dst.DrawImage(im, op)
	if a.iconT > 0 && a.icon != "" {
		g.bubble(a, bubFiller, Vec{a.pos.X, a.pos.Y - 4}, a.icon, false, -float64(h)*0.2)
	}
}
