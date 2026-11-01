package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Small critters: rabbit, pig, duck and (blue) budgie. Like the hens and chicks
// they scatter when somebody gets close, and the player can kill them.
// ---------------------------------------------------------------------------

const (
	animRabbit = iota + 9
	animPig
	animDuck
	animBudgie
)

func init() {
	animalKinds = append(animalKinds, "rabbit", "pig", "duck", "budgie")
}

var rabbitCoats = []coatCols{
	{hx(0x9a8470), hx(0x5a4a3c), hx(0xc4b09a), hx(0xf4f0ea)},
	{hx(0xf0ece8), hx(0xa8a4a0), hx(0xffffff), hx(0xf8b0b8)},
	{hx(0x48403c), hx(0x241e1c), hx(0x6a625c), hx(0xe8e4e0)},
}

func genRabbitFrame(frame, coat int) *pixBuf {
	b := newPix(26, 22)
	b.ox, b.oy = 1, 1
	cc := rabbitCoats[coat%len(rabbitCoats)]
	body, dark, light, tail := cc.body, cc.dark, cc.light, cc.mane
	hop := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		hop = []int{0, 3, 3, 0}[frame-fWalk0]
	}
	hy := 9 - hop
	b.ellipse(10, float64(14-hop), 7.5, 5, body)
	b.ellipse(7, float64(16-hop), 4, 4, shade(body, 0.9)) // haunch
	b.ellipse(11, float64(12-hop), 4.5, 2.4, light)
	b.ellipse(5, float64(13-hop), 2.4, 2.4, tail) // fluffy tail
	b.ellipse(17, float64(hy), 4, 3.4, body)      // head
	b.rect(15, hy-9, 2, 7, body)                  // ears
	b.rect(18, hy-8, 2, 6, body)
	b.rect(15, hy-8, 1, 5, c(0xe8a0a8))
	b.rect(18, hy-7, 1, 4, c(0xe8a0a8))
	b.set(18, hy-1, c(0x14100c))
	b.set(21, hy+1, c(0xe8a0a8))
	if frame == fPeck0 || frame == fPeck1 {
		b.ellipse(17, float64(hy+3+(frame-fPeck0)), 4, 3.4, body)
	}
	b.rect(13, 18-hop, 4, 3, dark) // front paws
	b.rect(7, 19-hop, 5, 2, dark)
	return b
}

func genPigFrame(frame, coat int) *pixBuf {
	b := newPix(36, 24)
	b.ox, b.oy = 1, 1
	body, dark, light := hx(0xf0a8b0), hx(0xc87884), hx(0xffc8cc)
	if coat == 1 {
		body, dark, light = hx(0xd890a0), hx(0xa86070), hx(0xf0b0b8)
	} else if coat == 2 {
		body, dark, light = hx(0x8a6a58), hx(0x4a3428), hx(0xb09078)
	}
	bob := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		bob = (frame - fWalk0) % 2
	}
	leg := func(x int, i int) {
		dx, lift := legSwing(frame, i)
		fx := float64(x) + dx*2.5
		b.rect(int(fx)-1, 17, 3, 5-int(lift*2), shade(body, 0.85))
		b.rect(int(fx)-1, 21-int(lift*2), 3, 2, hx(0x3a2a24))
	}
	leg(10, 2)
	leg(24, 0)
	b.ellipse(16, float64(13-bob), 12, 7.5, body)
	b.ellipse(16, float64(10-bob), 9, 3.5, light)
	leg(13, 3)
	leg(27, 1)
	b.ellipse(27, float64(12-bob), 6, 6, body) // head
	b.rect(22, 5-bob, 4, 5, dark)              // ear
	b.ellipse(32, float64(14-bob), 3, 2.8, light)
	b.set(31, 13-bob, hx(0x3a2028))
	b.set(33, 13-bob, hx(0x3a2028))
	b.set(26, 11-bob, hx(0x14100c))
	b.line(4, 11, 2, 8, dark) // curly tail
	b.set(1, 8, dark)
	b.set(2, 6, dark)
	if frame == fPeck0 || frame == fPeck1 {
		b.ellipse(31, float64(18+(frame-fPeck0)), 3.5, 2.8, light)
	}
	return b
}

func genDuckFrame(frame, coat int) *pixBuf {
	b := newPix(24, 22)
	b.ox, b.oy = 1, 1
	body, dark, light, head := hx(0xf8f8f0), hx(0xc8c8c0), hx(0xffffff), hx(0xf8f8f0)
	if coat == 1 { // mallard
		body, dark, light, head = hx(0x9a7a58), hx(0x5a4a38), hx(0xc8b090), hx(0x1f7a4a)
	} else if coat == 2 {
		body, dark, light, head = hx(0xe8d070), hx(0xb09840), hx(0xf8e8a0), hx(0xe8d070)
	}
	bob := 0
	hx0, hy0 := 16, 7
	if frame >= fWalk0 && frame <= fWalk3 {
		bob = (frame - fWalk0) % 2
	}
	if frame == fPeck0 || frame == fPeck1 {
		hx0, hy0 = 18, 12+(frame-fPeck0)*2
	}
	b.ellipse(10, float64(13+bob), 9, 5.5, body)
	b.ellipse(10, float64(11+bob), 6, 3, light)
	b.ellipse(7, float64(14+bob), 5, 3.5, dark)
	b.rect(1, 10+bob, 3, 3, body) // tail
	thickLine(b, 15, float64(11+bob), float64(hx0), float64(hy0), 3, head)
	b.ellipse(float64(hx0), float64(hy0), 3, 2.8, head)
	b.rect(hx0+2, hy0, 4, 2, hx(0xf0902a))
	b.set(hx0+1, hy0-1, hx(0x14100c))
	o := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		o = []int{2, 0, -2, 0}[frame-fWalk0]
	}
	for _, lx := range []int{8, 12} {
		off := o
		if lx == 12 {
			off = -o
		}
		b.line(lx, 17+bob, lx+off, 20, hx(0xf0902a))
		b.rect(lx+off-1, 20, 4, 1, hx(0xf0902a))
	}
	return b
}

func genBudgieFrame(frame, coat int) *pixBuf {
	b := newPix(18, 16)
	b.ox, b.oy = 1, 1
	body, dark, light := hx(0x3a78e0), hx(0x24509a), hx(0x7aaef4)
	if coat == 1 {
		body, dark, light = hx(0x4ac060), hx(0x2a8a40), hx(0x9ae8a0)
	} else if coat == 2 {
		body, dark, light = hx(0xf0d830), hx(0xb09c18), hx(0xfff088)
	}
	hop := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		hop = []int{0, 2, 1, 0}[frame-fWalk0]
	}
	hy := 4 - hop
	b.line(4, 10-hop, 0, 14-hop, dark) // long tail
	b.line(5, 10-hop, 1, 15-hop, body)
	b.ellipse(8, float64(9-hop), 5, 4, body)
	b.ellipse(9, float64(8-hop), 3, 2, light)
	b.ellipse(6, float64(10-hop), 3, 2.4, dark) // wing
	b.ellipse(11, float64(hy), 3.2, 3, hx(0xf4f0b0))
	b.rect(9, hy-2, 5, 1, body)
	b.rect(14, hy, 2, 2, hx(0xf09a30))
	b.set(12, hy-1, hx(0x14100c))
	b.set(10, hy+2, hx(0x2a2a8a))
	b.set(11, hy+2, hx(0x2a2a8a))
	b.line(7, 13-hop, 7, 15, hx(0xc88848))
	b.line(10, 13-hop, 10, 15, hx(0xc88848))
	return b
}

func buildCritterSprites(out AnimalSprites) {
	for coat := 0; coat < 3; coat++ {
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fPeck0, fPeck1} {
			out[animalKey(animRabbit, coat, f)] = genRabbitFrame(f, coat).outlinedSel(0.4).img()
			out[animalKey(animPig, coat, f)] = genPigFrame(f, coat).outlinedSel(0.4).img()
			out[animalKey(animDuck, coat, f)] = genDuckFrame(f, coat).outlinedSel(0.4).img()
			out[animalKey(animBudgie, coat, f)] = genBudgieFrame(f, coat).outlinedSel(0.4).img()
		}
	}
}

// critterFrame: sprite frame of the small animals.
func critterFrame(a *Animal) int {
	switch {
	case a.state == aGraze:
		return fPeck0 + int(a.anim*4)%2
	case a.moving:
		rate := 9.0
		if a.kind == animPig || a.kind == animDuck {
			rate = 7
		}
		return fWalk0 + int(a.anim*rate)%4
	}
	return fStand
}

// updateCritter drives rabbit, pig, duck and budgie: wander, peck, scatter when approached.
func (g *Game) updateCritter(a *Animal, dt float64) {
	scareD, fleeV, walkV, cry := 30.0, 52.0, 14.0, "!"
	switch a.kind {
	case animRabbit:
		scareD, fleeV, walkV, cry = 44, 72, 18, "¡!"
	case animPig:
		scareD, fleeV, walkV, cry = 24, 38, 12, "OINK!"
	case animDuck:
		scareD, fleeV, walkV, cry = 28, 44, 12, "CUAC!"
	case animBudgie:
		scareD, fleeV, walkV, cry = 36, 90, 16, "PIO PIO!"
	case animDeer:
		scareD, fleeV, walkV, cry = 64, 95, 16, "¡!"
	}
	scare := dist(g.p.pos, a.pos) < scareD && !g.p.hidden
	if !scare {
		for _, gu := range g.guests {
			if !gu.out() && gu.moving && dist(gu.pos, a.pos) < scareD*0.8 {
				scare = true
				break
			}
		}
	}
	if scare && a.state != aFlee {
		a.state, a.timer = aFlee, 1.3
		a.path = g.lvl.Path(a.pos, g.lvl.RandomIn(g.rng, Rect{a.pos.X - 70, a.pos.Y - 45, 140, 90}.clip(a.area)))
		a.icon, a.iconT = cry, 1
	}
	switch a.state {
	case aFlee:
		a.timer -= dt
		if g.animalWalk(a, fleeV, dt) || a.timer <= 0 {
			a.state, a.timer = aIdle, 1
		}
	case aIdle, aGraze:
		a.timer -= dt
		if a.timer <= 0 {
			if g.rng.Float64() < 0.45 {
				a.state, a.timer = aGraze, 1+g.rng.Float64()*2.5
			} else {
				a.path = g.lvl.Path(a.pos, g.lvl.RandomIn(g.rng, Rect{a.pos.X - 40, a.pos.Y - 25, 80, 50}.clip(a.area)))
				a.state, a.timer = aWalk, 8
			}
		}
	case aWalk:
		a.timer -= dt
		if g.animalWalk(a, walkV, dt) || a.timer <= 0 {
			a.state, a.timer = aIdle, 0.8
		}
	}
}

// killableAnimal: everything except the big animals can be stabbed.
func killableAnimal(k int) bool {
	switch k {
	case animHorse, animBull, animElephant, animGiraffe, animQuijote, animSancho:
		return false
	}
	return true
}

func (g *Game) nearAnimal(p Vec, maxD float64) (*Animal, float64) {
	var best *Animal
	bd := maxD
	for _, a := range g.animals {
		if a.dead || !killableAnimal(a.kind) {
			continue
		}
		if d := dist(p, a.pos); d < bd {
			best, bd = a, d
		}
	}
	return best, bd
}

func (g *Game) killAnimal(a *Animal) {
	g.stat("animals", 1)
	if a.kind >= 0 && a.kind < len(animalKinds) {
		switch animalKinds[a.kind] {
		case "pig":
			g.stat("an_pig", 1)
		case "cat":
			g.stat("an_cat", 1)
		case "dog":
			g.stat("an_dog", 1)
		case "hen", "chick", "duck", "budgie":
			g.stat("an_bird", 1)
		case "snake":
			g.stat("an_snake", 1)
		}
	}
	a.dead = true
	g.award(a.pos, 10+5*b2i(a.kind == animSnake), "")
	a.icon, a.state = "", aIdle
	g.decals = append(g.decals, Decal{pos: a.pos, kind: 0, r: 5 + g.rng.Float64()*3})
	g.burst(Vec{a.pos.X, a.pos.Y - 6}, 10, 60, rgb(0xb01818), 0.5, 2, 200)
	g.snd.Play("stab")
	g.say("Has matado a " + animalNameEs(a.kind) + "...")
	// whoever sees it gets a fright (it is not a reason to call the police)
	for _, gu := range g.guests {
		if g.canWatch(gu) && gu.perf == "" && g.sees(gu.pos, gu.dir, a.pos, guestRange, guestDot) {
			g.alarm(gu, "!?")
		}
	}
}

func animalNameEs(k int) string {
	if k >= 0 && k < len(animalKinds) {
		if n, ok := map[string]string{"hen": "una gallina", "dog": "un perro", "cat": "un gato", "kangaroo": "un canguro", "chick": "un pollito", "rabbit": "un conejo", "pig": "un cerdo", "duck": "un pato", "budgie": "un periquito", "snake": "una serpiente"}[animalKinds[k]]; ok {
			return n
		}
	}
	return "un animal"
}

// scatterCritters adds a few small animals to every party.
func (g *Game) scatterCritters() {
	kinds := []int{animRabbit, animPig, animDuck, animBudgie, animHen, animChick, animRabbit, animDuck}
	n := 4 + g.rng.Intn(4)
	for i := 0; i < n; i++ {
		pos, ok := g.outdoorSpot()
		if !ok {
			break
		}
		k := kinds[g.rng.Intn(len(kinds))]
		if !animalAllowed(k) {
			continue
		}
		a := &Animal{kind: k, pos: g.lvl.NearestFree(pos), face: -1, area: Rect{pos.X - 80, pos.Y - 50, 160, 100}, seed: g.rng.Float64() * 10, timer: g.rng.Float64() * 3, coat: g.rng.Intn(3)}
		if g.rng.Intn(2) == 0 {
			a.face = 1
		}
		g.animals = append(g.animals, a)
	}
}

// drawDeadAnimal: belly up on a small pool of blood.
func (g *Game) drawDeadAnimal(dst *ebiten.Image, a *Animal) {
	im := g.as.animals[animalKey(a.kind, a.coat, fStand)]
	if im == nil {
		return
	}
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	x, y := g.wx(a.pos.X), g.wy(a.pos.Y)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1, -1)
	if a.face < 0 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(w, 0)
	}
	op.GeoM.Translate(math.Round(x-w/2), math.Round(y+h*0.15))
	op.ColorScale.Scale(0.8, 0.7, 0.7, 1)
	dst.DrawImage(im, op)
}
