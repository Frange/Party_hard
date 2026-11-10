package main

import "math/rand"

// ---------------------------------------------------------------------------
// Guinea pigs (2026-11-09): small, plump, tail-less critters in little groups of 2-4 ("wheek!"). Free (no shop).
// They live in gardens, parks and farms (grass/dirt, outdoors), never on the space station or without outdoors.
// They behave like the other critters (updateCritter: wander, nibble, scatter and squeak when approached) and
// die like them (killAnimal). Own dice (wildDice), added from scatterWild (animals6.go).
// ---------------------------------------------------------------------------

const animCobaya = animLion + 1

const wildCobayaChance = 0.4 // green level

func init() {
	if len(animalKinds) != animCobaya {
		panic("animals: the guinea pig is out of step with animalKinds")
	}
	animalKinds = append(animalKinds, "cobaya")
}

// scatterCobayas: a little group of 2-4 on the grass.
func (g *Game) scatterCobayas(r *rand.Rand) {
	if r.Float64() >= wildCobayaChance {
		return
	}
	p, ok := g.wildSpot(r, g.greenSpot)
	if !ok {
		return
	}
	g.addCobayaGroup(r, p, 2+r.Intn(3))
}

func (g *Game) addCobayaGroup(r *rand.Rand, p Vec, n int) {
	for i := 0; i < n; i++ {
		q := Vec{p.X + r.Float64()*40 - 20, p.Y + r.Float64()*24 - 12}
		if i == 0 || !g.lvl.Outdoor(q) || g.lvl.InPool(q) || g.lvl.badGround(q) || !g.greenSpot(q) {
			q = p // the spot beside it is not outdoor ground: it huddles on the first one
		}
		g.addWild(r, animCobaya, q, 90, 60)
	}
}

func genCobayaFrame(frame, coat int) *pixBuf {
	b := newPix(20, 13)
	b.ox, b.oy = 1, 1
	body, patch, belly := hx(0xa8703a), hx(0xf2e8d4), hx(0xe0c89a)
	ear := hx(0xd89a8a)
	switch coat {
	case 1: // white with an orange patch
		body, patch, belly = hx(0xf4eee2), hx(0xd88a3a), hx(0xffffff)
	case 2: // black and white
		body, patch, belly = hx(0x2e2a2c), hx(0xf2f0ec), hx(0x6a6468)
		ear = hx(0xb08890)
	}
	bob := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		bob = (frame - fWalk0) % 2
	}
	y := 7 - bob
	// four stubby feet
	for i, x := range []int{5, 8, 12, 15} {
		dx, lift := legSwing(frame, i)
		b.rect(x+int(dx), 10-int(lift), 2, 2, shade(body, 0.7))
	}
	b.ellipse(9, float64(y), 8, 4.2, body)      // plump body, no tail
	b.ellipse(10, float64(y+2), 6, 1.8, belly)  // belly
	b.ellipse(6, float64(y-1), 2.6, 2.2, patch) // patch on the back
	b.ellipse(14, float64(y-1), 4, 3.6, body)   // big round head
	b.ellipse(16, float64(y-2), 2, 1.8, patch)  // white blaze on the nose
	hy := y
	if frame == fPeck0 || frame == fPeck1 {
		hy = y + 1 + (frame - fPeck0) // nibbling the grass
		b.ellipse(16, float64(hy), 3, 2.4, body)
	}
	b.rect(12, y-5, 2, 2, ear) // small round ear
	b.set(12, y-5, shade(ear, 0.8))
	b.set(16, hy-2, hx(0x101010))     // eye
	b.set(18, hy-1, hx(0x3a2018))     // nose
	b.set(17, hy+1, shade(body, 0.6)) // mouth
	return b
}
