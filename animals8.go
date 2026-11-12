package main

import (
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Capybaras (2026-11-11): big, barrel-shaped, tail-less rodents with a blunt square snout, small round ears and a
// sleepy half-closed eye. Calm and sociable: little groups of 1-2 adults, sometimes with a baby (coat capiBaby, a
// smaller copy). Free (no shop), like the guinea pig. They come now and then on levels with water (beside it, never
// in it) or, failing that, on grass/dirt; never indoors, on ships without outdoors or on the station.
//
// They are critters (updateCritter: wander, scatter when somebody runs at them) but slow and hard to startle, and
// every other stop is a long rest lying down (fSit) instead of a nibble (fPeck0/1). When startled they give a soft
// procedural whistle ("capi_whistle", built once with sfxOnce; no audio files). They die like every other animal
// (killAnimal: counts as one fallen animal, 10 points, the same as the guinea pig) and the police never charge an
// animal. Placed by scatterCapibaras at the END of scatterWild with the wild dice, so the draws of the other wild
// animals of the party do not move, and g.rng is never touched; capAnimals (cap.go) still applies the cap after.
// -wildnow capibara puts a group next to the start.
// ---------------------------------------------------------------------------

const animCapibara = animCobaya + 1

const (
	wildCapibaraChance = 0.3 // level with water or grass
	capiBaby           = 3   // coat of the baby: a smaller copy of coat 0
)

var capiSpawn = true // tests: false = the parties exactly as they were before the capybara

func init() {
	if len(animalKinds) != animCapibara {
		panic("animals: the capybara is out of step with animalKinds")
	}
	animalKinds = append(animalKinds, "capibara")
}

// scatterCapibaras: sometimes, a small group beside the water (or on the grass if the level has no water).
func (g *Game) scatterCapibaras(r *rand.Rand, green, wet bool) {
	if !capiSpawn || (!green && !wet) {
		return
	}
	if r.Float64() >= wildCapibaraChance {
		return
	}
	pick := g.nearWater
	if !wet {
		pick = g.greenSpot
	}
	p, ok := g.wildSpot(r, pick)
	if !ok {
		return
	}
	adults := 1 + r.Intn(2)
	g.addCapibaraGroup(r, p, adults, r.Float64() < 0.4, pick)
}

// addCapibaraGroup: n adults (and a baby) around p; a spot that is not good ground makes it huddle on p.
func (g *Game) addCapibaraGroup(r *rand.Rand, p Vec, n int, baby bool, ok func(Vec) bool) {
	if baby {
		n++
	}
	for i := 0; i < n; i++ {
		q := Vec{p.X + r.Float64()*48 - 24, p.Y + r.Float64()*28 - 14}
		if i == 0 || !g.lvl.Outdoor(q) || g.lvl.InPool(q) || g.lvl.badGround(q) || g.inWater(q) || (ok != nil && !ok(q)) {
			q = p
		}
		a := g.addWild(r, animCapibara, q, 110, 70)
		if baby && i == n-1 {
			a.coat, a.label = capiBaby, "Cría de capibara"
		}
	}
}

// inWater: p is inside a pool or a water/sea floor.
func (g *Game) inWater(p Vec) bool {
	for _, w := range g.lvl.waterRects() {
		if w.Has(p.X, p.Y) {
			return true
		}
	}
	return false
}

// updateCapibara: a slow critter; every other stop is a long rest lying down (a.hit = resting).
func (g *Game) updateCapibara(a *Animal, dt float64) {
	prev := a.state
	g.updateCritter(a, dt)
	if a.state == aGraze && prev != aGraze {
		a.hit = !a.hit
		if a.hit {
			a.timer += 3 // a proper siesta
		}
	}
	if a.state == aFlee && prev != aFlee {
		a.hit = false
		if g.snd != nil {
			g.sfxOnce("capi_whistle", capiWhistlePCM)
			g.snd.Play("capi_whistle")
		}
	}
}

func capibaraFrame(a *Animal) int {
	switch {
	case a.state == aGraze && a.hit:
		return fSit
	case a.state == aGraze:
		return fPeck0 + int(a.anim*3)%2
	case a.moving:
		return fWalk0 + int(a.anim*6)%4
	}
	return fStand
}

// capiWhistlePCM: two soft rising whistles over a low purr.
func capiWhistlePCM() []byte {
	sweep := func(f0, f1, dur, vol float64) []float64 {
		out := samples(dur)
		ph := 0.0
		for i := range out {
			t := float64(i) / sampleRate
			f := f0 + (f1-f0)*t/dur
			ph += 2 * math.Pi * f / sampleRate
			env := math.Sin(math.Pi * t / dur) // soft in, soft out
			out[i] = math.Sin(ph) * env * vol
		}
		return out
	}
	return mix(sweep(620, 900, 0.12, 0.28), shift(sweep(660, 980, 0.14, 0.24), 0.17), lowpass(tone(110, 0.3, 0.25, 8, 2), 8))
}

// ---------------------------------------------------------------------------
// art: profile facing right (32x22), and the front / back views for walking up or down the screen
// ---------------------------------------------------------------------------

type capiCols struct{ body, light, dark, snout, leg, inner c4 }

func capibaraCols(coat int) capiCols {
	switch coat {
	case 1: // golden
		return capiCols{hx(0xc8945a), hx(0xe0b47e), hx(0x8e6034), hx(0x6a5242), hx(0x4e3a2c), hx(0x3a281c)}
	case 2: // chestnut
		return capiCols{hx(0x96603a), hx(0xb27c52), hx(0x5e3a20), hx(0x4a3a32), hx(0x3a2a20), hx(0x2a1a12)}
	}
	return capiCols{hx(0xb47a4c), hx(0xcc9a6a), hx(0x7e5232), hx(0x5c4a40), hx(0x463428), hx(0x34241a)}
}

func genCapibaraFrame(frame, coat int) *pixBuf {
	b := newPix(32, 22)
	b.ox, b.oy = 1, 1
	c := capibaraCols(coat)
	bob := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		bob = (frame - fWalk0) % 2
	}
	rest := frame == fSit
	if rest {
		bob = -1 // lying down: legs tucked, the barrel sinks onto the ground
	}
	y := 12 - bob
	leg := func(x, i int, col c4) {
		if rest {
			return
		}
		dx, lift := legSwing(frame, i)
		fx := x + int(math.Round(dx))
		h := 5 - int(lift)
		b.rect(fx, 20-h, 3, h, col)
		b.rect(fx, 19-int(lift), 3, 1, shade(col, 0.75)) // the toes
	}
	// far legs, then the barrel, then the near legs
	leg(5, 2, shade(c.leg, 0.8))
	leg(16, 0, shade(c.leg, 0.8))
	b.ellipse(12, float64(y), 10.5, 6, c.body)               // the barrel, no tail
	b.ellipse(12, float64(y+3), 8, 2.4, shade(c.dark, 1.15)) // darker belly
	b.ellipse(11, float64(y-3), 7, 1.8, c.light)             // light back
	b.ellipse(3, float64(y+1), 2.2, 3.4, shade(c.body, 0.88))
	leg(8, 3, c.leg)
	leg(19, 1, c.leg)
	// head: big and square, lowered when it eats
	hy := y - 3
	if frame == fPeck0 || frame == fPeck1 {
		hy = y + 1 + (frame - fPeck0)
	}
	if rest {
		hy = y - 2
	}
	b.ellipse(20, float64(hy), 5.5, 5, c.body)                  // big tall head
	b.rect(21, hy-4, 5, 8, c.body)                              // the long box of the face
	b.ellipse(25.5, float64(hy)+0.5, 2.8, 3.6, c.body)          // blunt front
	b.ellipse(19, float64(hy+3), 3.6, 1.8, shade(c.body, 0.9))  // jowl
	b.ellipse(26.6, float64(hy+1), 2, 3.2, c.snout)             // dark square muzzle
	b.rect(25, hy-2, 2, 6, c.snout)                             // ...shading into the face
	b.ellipse(21, float64(hy-3), 3.5, 1.2, shade(c.body, 1.12)) // light brow
	b.set(27, hy-1, shade(c.snout, 0.5))                        // big nostril
	b.set(28, hy-1, shade(c.snout, 0.5))
	b.set(26, hy+3, shade(c.snout, 0.6)) // mouth
	b.set(27, hy+3, shade(c.snout, 0.6))
	// small round ear with a dark inside
	b.rect(16, hy-6, 3, 2, c.body)
	b.set(17, hy-7, c.body)
	b.set(17, hy-6, c.inner)
	// the sleepy eye: a dark line under a heavy lid (closed when it rests)
	b.set(21, hy-2, shade(c.body, 0.7))
	b.set(22, hy-2, shade(c.body, 0.7))
	b.set(23, hy-2, shade(c.body, 0.7))
	if rest {
		b.set(21, hy-1, shade(c.dark, 0.6))
		b.set(22, hy-1, shade(c.dark, 0.6))
	} else {
		b.set(21, hy-1, hx(0x140e0a))
		b.set(22, hy-1, hx(0x140e0a))
		b.set(22, hy, shade(c.body, 0.8))
	}
	return b
}

// genCapibaraView: the chest (front) or the back, 24x22, for walking down / up the screen. step moves the legs.
func genCapibaraView(coat int, front bool, step int) *pixBuf {
	b := newPix(24, 22)
	b.ox, b.oy = 1, 1
	c := capibaraCols(coat)
	l1, l2 := 0, 1
	if step%2 == 1 {
		l1, l2 = 1, 0
	}
	b.rect(5, 15+l1, 3, 5-l1, c.leg)
	b.rect(14, 15+l2, 3, 5-l2, c.leg)
	b.ellipse(11, 11, 8.5, 7, c.body)
	if front {
		b.ellipse(11, 14, 5, 3, shade(c.dark, 1.15)) // chest
		b.ellipse(11, 9, 5.5, 5, c.body)             // head
		b.rect(5, 2, 3, 3, c.body)                   // ears
		b.rect(14, 2, 3, 3, c.body)
		b.set(6, 3, c.inner)
		b.set(15, 3, c.inner)
		b.ellipse(11, 12, 3.6, 2.8, c.snout) // snout, head-on
		b.set(10, 11, shade(c.snout, 0.55))
		b.set(12, 11, shade(c.snout, 0.55))
		b.set(11, 14, shade(c.snout, 0.6))
		for _, x := range []int{7, 15} { // sleepy eyes
			b.set(x, 7, shade(c.body, 0.7))
			b.set(x, 8, hx(0x140e0a))
		}
	} else {
		b.ellipse(11, 7, 7, 3, c.light) // light back
		b.ellipse(11, 4, 4, 2.6, c.body)
		b.rect(6, 1, 2, 2, c.body)
		b.rect(14, 1, 2, 2, c.body)
		b.ellipse(11, 15, 6, 3, shade(c.body, 0.85)) // round rump, no tail
	}
	return b
}

func capibaraSprite(b *pixBuf, coat int) *pixBuf {
	if coat == capiBaby {
		b = b.scaled(0.6)
	}
	return b.outlinedSel(0.4)
}

func buildCapibaraSprites(out AnimalSprites) {
	for coat := 0; coat <= capiBaby; coat++ {
		src := coat
		if coat == capiBaby {
			src = 0
		}
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fPeck0, fPeck1, fSit} {
			out[animalKey(animCapibara, coat, f)] = capibaraSprite(genCapibaraFrame(f, src), coat).img()
		}
	}
}

var capiViewCache = map[int]PropImg{}

// capibaraFreeView: the front or back picture (drawAnimal anchors it at its bottom centre).
func capibaraFreeView(a *Animal) PropImg {
	coat, step := a.coat, int(a.anim*6)%2
	key := coat<<4 | b2i(a.vert > 0)<<1 | step
	if pi, ok := capiViewCache[key]; ok {
		return pi
	}
	src := coat
	if coat == capiBaby {
		src = 0
	}
	sb := capibaraSprite(genCapibaraView(src, a.vert > 0, step), coat)
	pi := PropImg{img: sb.img(), ox: float64(sb.w) / 2, oy: float64(sb.h)}
	capiViewCache[key] = pi
	return pi
}
