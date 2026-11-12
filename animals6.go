package main

import (
	"math"
	"math/rand"
	"time"
)

// ---------------------------------------------------------------------------
// Wild animals (2026-11-09): frogs, squirrels, boars, crocodiles, cheetahs and lions. Our own pixel art (a cartoon
// take, side view facing right like every other animal), with walk frames.
//
//   - Frogs and squirrels are free: they come now and then on levels with grass/dirt (squirrels) or by the water
//     and in parks (frogs). Boars (shop) on green levels. They are critters: wander, peck, scatter.
//   - The crocodile (shop) only shows up where the level has water (a pool, a "water"/"sea" floor) and stays by it.
//     It snaps at a guest who stands in front of its jaws; it kills at most ONE guest per party.
//   - Cheetahs and lions (shop) come VERY rarely (wildCatChance) on green levels. Each one makes a single short
//     hunt: a sprint of a couple of seconds at one guest (never the player), that may or may not end in a kill,
//     and then it walks off and leaves the party. They never clear half the screen.
//
// Everything is decided with its OWN dice (wildDice), so g.rng and the rest of the level do not move. They are
// placed before the critters, and capAnimals (cap.go) still applies the single limit afterwards. Every one of them
// can be stabbed and counts as a fallen animal (killAnimal), with the same rules of suspicion as the others.
// -wildnow <kind> puts one next to the start (debug; it freezes the progress like every debug flag).
// ---------------------------------------------------------------------------

const (
	animFrog = iota + animSnake + 1
	animSquirrel
	animBoar
	animCroc
	animCheetah
	animLion
)

func init() {
	if len(animalKinds) != animFrog {
		panic("animals: the wild animals are out of step with animalKinds")
	}
	animalKinds = append(animalKinds, "frog", "squirrel", "boar", "croc", "cheetah", "lion")
}

// chances per party (own dice)
const (
	wildSquirrelChance = 0.45 // green level
	wildFrogChance     = 0.55 // by the water (0.3 in a green level without water)
	wildBoarChance     = 0.35 // green level, bought
	wildCrocChance     = 0.35 // level with water, bought
	wildCatChance      = 1.0 / 20
	bigCatHuntMin      = 25.0 // seconds before the one hunt
	bigCatHuntSpan     = 35.0
)

var (
	wildRng  *rand.Rand
	wildSeed int64  // tests: non-zero = deterministic dice
	dbgWild  string // -wildnow kind
)

func wildDice() *rand.Rand {
	if wildRng == nil {
		sd := wildSeed
		if sd == 0 {
			sd = time.Now().UnixNano()
		}
		wildRng = rand.New(rand.NewSource(sd))
	}
	return wildRng
}

// wildReseed: tests.
func wildReseed(seed int64) { wildSeed, wildRng = seed, nil }

// floorKindAt: the kind of the topmost floor under p ("" = none).
func (l *Level) floorKindAt(p Vec) string {
	kind := ""
	for _, f := range l.Def.Floors {
		if f.R.Has(p.X, p.Y) {
			kind = f.Kind
		}
	}
	return kind
}

// waterRects: the pools and the water floors of the level.
func (l *Level) waterRects() []Rect {
	out := append([]Rect(nil), l.Pools...)
	for _, f := range l.Def.Floors {
		if f.Kind == "water" || f.Kind == "sea" {
			out = append(out, f.R)
		}
	}
	return out
}

// wildSpot: a free outdoor roam point that passes ok (nil = any), drawn with the wild dice.
func (g *Game) wildSpot(r *rand.Rand, ok func(Vec) bool) (Vec, bool) {
	if g.lvl.Def.Sky == "space" || len(g.lvl.Def.Roam) == 0 {
		return Vec{}, false
	}
	for i := 0; i < 80; i++ {
		p := g.lvl.RandomRoam(r)
		if !g.lvl.Outdoor(p) || g.lvl.InPool(p) || g.lvl.badGround(p) {
			continue
		}
		if ok == nil || ok(p) {
			return p, true
		}
	}
	return Vec{}, false
}

func (g *Game) greenSpot(p Vec) bool {
	k := g.lvl.floorKindAt(p)
	return k == "grass" || k == "dirt"
}

func (g *Game) nearWater(p Vec) bool {
	for _, w := range g.lvl.waterRects() {
		if w.Grow(60).Has(p.X, p.Y) && !w.Has(p.X, p.Y) {
			return true
		}
	}
	return false
}

func (g *Game) addWild(r *rand.Rand, kind int, p Vec, w, h float64) *Animal {
	a := &Animal{kind: kind, pos: g.lvl.NearestFree(p), face: 1, area: Rect{p.X - w/2, p.Y - h/2, w, h}, seed: r.Float64() * 10, timer: r.Float64() * 3, coat: r.Intn(3)}
	if r.Intn(2) == 0 {
		a.face = -1
	}
	if kind == animCheetah || kind == animLion {
		a.huntT = bigCatHuntMin + r.Float64()*bigCatHuntSpan
	}
	g.animals = append(g.animals, a)
	return a
}

// scatterWild: the wild animals of this party (none on interiors, ships or the station).
func (g *Game) scatterWild() {
	r := wildDice()
	if k := animalKindIndex(dbgWild); dbgWild != "" && animalKinds[k] == dbgWild {
		st := g.lvl.Def.Start
		g.addWild(r, k, Vec{st.X + 50, st.Y - 20}, 160, 100)
		if k == animCobaya {
			g.addCobayaGroup(r, Vec{st.X + 50, st.Y - 20}, 3) // a group, as in the parties
		}
		if k == animCapibara {
			g.addCapibaraGroup(r, Vec{st.X + 84, st.Y - 12}, 1, true, nil) // another adult and a baby beside it (animals8.go)
		}
		if k == animCheetah || k == animLion {
			g.animals[len(g.animals)-1].huntT = 4 // the debug flag wants to see the hunt
		}
		return
	}
	if _, ok := g.wildSpot(r, nil); !ok {
		return
	}
	_, green := g.wildSpot(r, g.greenSpot)
	_, wet := g.wildSpot(r, g.nearWater)
	if green && r.Float64() < wildSquirrelChance {
		for i, n := 0, 1+r.Intn(2); i < n; i++ {
			if p, ok := g.wildSpot(r, g.greenSpot); ok {
				g.addWild(r, animSquirrel, p, 150, 100)
			}
		}
	}
	frogP := wildFrogChance
	if !wet {
		frogP = 0.3
	}
	if (wet || green) && r.Float64() < frogP {
		pick := g.nearWater
		if !wet {
			pick = g.greenSpot
		}
		for i, n := 0, 1+r.Intn(2); i < n; i++ {
			if p, ok := g.wildSpot(r, pick); ok {
				g.addWild(r, animFrog, p, 110, 70)
			}
		}
	}
	if green {
		g.scatterCobayas(r)
	}
	if green && ownedID("animal:boar") && r.Float64() < wildBoarChance {
		if p, ok := g.wildSpot(r, g.greenSpot); ok {
			g.addWild(r, animBoar, p, 180, 110)
		}
	}
	if wet && ownedID("animal:croc") && r.Float64() < wildCrocChance {
		if p, ok := g.wildSpot(r, g.nearWater); ok {
			g.addWild(r, animCroc, p, 90, 60)
		}
	}
	var cats []int
	for _, k := range []int{animCheetah, animLion} {
		if ownedID("animal:" + animalKinds[k]) {
			cats = append(cats, k)
		}
	}
	if green && len(cats) > 0 && r.Float64() < wildCatChance {
		if p, ok := g.wildSpot(r, g.greenSpot); ok {
			g.addWild(r, cats[r.Intn(len(cats))], p, 240, 150)
		}
	}
	g.scatterCapibaras(r, green, wet) // last: the draws of the animals above do not move (animals8.go)
}

// dropGoneAnimals removes the animals that left the party.
func (g *Game) dropGoneAnimals() {
	for _, a := range g.animals {
		if a.gone {
			out := g.animals[:0]
			for _, b := range g.animals {
				if !b.gone {
					out = append(out, b)
				}
			}
			g.animals = out
			return
		}
	}
}

// wildPrey: the nearest guest in reach that the animal could go for (never the player, never a performer).
func (g *Game) wildPrey(a *Animal, reach float64, ahead bool) *Guest {
	var best *Guest
	bd := reach
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" || gu.state == gSleep {
			continue
		}
		dx := gu.pos.X - a.pos.X
		if ahead && (dx*a.face < 2 || math.Abs(gu.pos.Y-a.pos.Y) > 14) {
			continue
		}
		if d := dist(gu.pos, a.pos); d < bd {
			best, bd = gu, d
		}
	}
	return best
}

// updateCroc: lies by the water; a guest right in front of the jaws gets a warning and then a snap.
func (g *Game) updateCroc(a *Animal, dt float64) {
	a.huntT = math.Max(0, a.huntT-dt)
	switch a.state {
	case aWarn:
		a.timer -= dt
		if a.timer <= 0 {
			a.state, a.timer, a.hit = aKick, 0.35, false
		}
	case aKick:
		a.timer -= dt
		if !a.hit && a.timer < 0.2 {
			a.hit = true
			g.snd.Play("thud")
			if v, ok := a.victim.(*Guest); ok && !v.out() && dist(v.pos, a.pos) < 30 {
				if a.bites < 1 && g.animalKill(v, "", "Un cocodrilo") {
					a.bites++
					g.checkDone()
				} else {
					g.scare(v)
				}
			}
		}
		if a.timer <= 0 {
			a.state, a.timer, a.huntT, a.victim = aIdle, 3, 20, nil
		}
	default:
		g.animalWander(a, 6, dt, 0.6)
		if a.huntT <= 0 {
			if v := g.wildPrey(a, 28, true); v != nil {
				a.state, a.timer, a.victim, a.path = aWarn, 0.8, v, nil
				a.icon, a.iconT = "¡ÑAM!", 1
			}
		}
	}
}

// updateBigCat: wanders, makes ONE short hunt (a sprint of up to 2 s at the nearest guest) and leaves the party.
func (g *Game) updateBigCat(a *Animal, dt float64) {
	lion := a.kind == animLion
	switch a.state {
	case aCharge:
		a.timer -= dt
		v, _ := a.victim.(*Guest)
		if v == nil || v.out() {
			g.bigCatLeave(a)
			return
		}
		a.moving = true
		a.victimRepath(g, v.pos, dt)
		sp := 135.0
		if lion {
			sp = 100
		}
		g.animalWalk(a, sp, dt)
		if dist(a.pos, v.pos) < 14 {
			p := 0.55
			if lion {
				p = 0.65
			}
			if wildDice().Float64() < p && g.animalKill(v, "", map[bool]string{true: "Un león", false: "Un guepardo"}[lion]) {
				a.bites++
				g.snd.Play("thud")
				g.checkDone()
			} else {
				g.scare(v)
				v.icon, v.iconT = "¡POR LOS PELOS!", 1.6
			}
			g.bigCatLeave(a)
		} else if a.timer <= 0 { // too far: it gives up
			g.scare(v)
			g.bigCatLeave(a)
		}
	case aFlee: // leaving: walks away and is gone
		a.timer -= dt
		if g.animalWalk(a, 40, dt) || a.timer <= 0 {
			a.gone = true
		}
	default:
		g.animalWander(a, 16, dt, 0.4)
		a.huntT -= dt
		if a.huntT <= 0 && a.bites == 0 {
			if v := g.wildPrey(a, 150, false); v != nil {
				a.state, a.timer, a.victim, a.path = aCharge, 2.0, v, nil
				a.icon, a.iconT = map[bool]string{true: "¡GRRROAR!", false: "¡FFSSS!"}[lion], 1.2
				g.snd.Play("alert")
			} else {
				a.huntT = 4 // nobody around: tries again in a moment
			}
		}
	}
}

// bigCatLeave: the hunt is over, the cat walks to a far point of the level and disappears there.
func (g *Game) bigCatLeave(a *Animal) {
	a.state, a.timer, a.victim, a.huntT = aFlee, 10, nil, 1e9
	far := a.pos
	for i := 0; i < 20; i++ {
		far = g.lvl.RandomRoam(wildDice())
		if dist(far, a.pos) > 220 {
			break
		}
	}
	a.path = g.lvl.Path(a.pos, far)
}

// ---------------------------------------------------------------------------
// frames and art
// ---------------------------------------------------------------------------

func wildFrame(a *Animal) int {
	switch a.kind {
	case animCroc:
		switch a.state {
		case aWarn:
			return fWarn
		case aKick:
			return fKick
		}
	case animCheetah, animLion:
		if a.state == aCharge {
			return fWalk0 + int(a.anim*14)%4
		}
	case animCapibara:
		return capibaraFrame(a)
	}
	return critterFrame(a)
}

func buildWildSprites(out AnimalSprites) {
	for coat := 0; coat < 3; coat++ {
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fPeck0, fPeck1} {
			out[animalKey(animFrog, coat, f)] = genFrogFrame(f, coat).outlinedSel(0.4).img()
			out[animalKey(animSquirrel, coat, f)] = genSquirrelFrame(f, coat).outlinedSel(0.4).img()
			out[animalKey(animBoar, coat, f)] = genBoarFrame(f, coat).scaled(0.8).outlinedSel(0.4).img()
			out[animalKey(animCheetah, coat, f)] = genBigCatFrame(f, coat, false).scaled(0.8).outlinedSel(0.4).img()
			out[animalKey(animLion, coat, f)] = genBigCatFrame(f, coat, true).scaled(0.8).outlinedSel(0.4).img()
			out[animalKey(animCobaya, coat, f)] = genCobayaFrame(f, coat).outlinedSel(0.4).img()
		}
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fPeck0, fPeck1, fWarn, fKick} {
			out[animalKey(animCroc, coat, f)] = genCrocFrame(f, coat).outlinedSel(0.4).img()
		}
	}
	buildCapibaraSprites(out)
}

func genFrogFrame(frame, coat int) *pixBuf {
	b := newPix(18, 14)
	b.ox, b.oy = 1, 1
	body, dark, belly := hx(0x4aa83a), hx(0x2a6a24), hx(0xc8e08a)
	switch coat {
	case 1:
		body, dark, belly = hx(0x8a9a3a), hx(0x4a5a1a), hx(0xe0e0a0)
	case 2:
		body, dark, belly = hx(0x3a8ac8), hx(0x1a4a7a), hx(0xd0e8f0) // a blue tree frog
	}
	hop := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		hop = []int{0, 3, 4, 1}[frame-fWalk0]
	}
	y := 9 - hop
	b.ellipse(8, float64(y), 6, 3.6, body)
	b.ellipse(9, float64(y+1), 4, 2, belly)
	b.ellipse(4, float64(y+1), 3, 2.4, dark) // folded hind leg
	if hop > 1 {
		b.line(2, y+2, 0, y+4, dark) // the leg stretched out in the jump
	}
	b.ellipse(11, float64(y-3), 2, 2, body) // eyes on top
	b.ellipse(7, float64(y-3), 1.6, 1.6, body)
	b.set(11, y-4, hx(0x101010))
	b.set(7, y-4, hx(0x101010))
	b.line(10, y+1, 14, y+1, dark) // mouth
	b.rect(11, y+3, 3, 1, dark)    // front foot
	if frame == fPeck0 {
		b.line(14, y, 16, y-1, hx(0xe05a6a)) // the tongue after a fly
	}
	return b
}

func genSquirrelFrame(frame, coat int) *pixBuf {
	b := newPix(22, 20)
	b.ox, b.oy = 1, 1
	body, dark, belly := hx(0xc0602a), hx(0x7a3a14), hx(0xf0d8b0)
	switch coat {
	case 1:
		body, dark, belly = hx(0x8a8a90), hx(0x4a4a52), hx(0xe8e8e8)
	case 2:
		body, dark, belly = hx(0x8a5a30), hx(0x4a2c14), hx(0xe0c8a0)
	}
	hop := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		hop = []int{0, 2, 2, 0}[frame-fWalk0]
	}
	y := 13 - hop
	// the big bushy tail, curled up behind
	b.ellipse(4, float64(y-4), 3.6, 6, body)
	b.ellipse(5, float64(y-9), 3, 2.6, body)
	b.ellipse(4, float64(y-4), 1.8, 4, dark)
	b.ellipse(10, float64(y), 5, 3.6, body)
	b.ellipse(11, float64(y+1), 3, 2, belly)
	hy := y - 4
	if frame == fPeck0 || frame == fPeck1 {
		hy = y - 2 + (frame - fPeck0) // nibbling a nut
		b.ellipse(16, float64(hy+3), 1.4, 1.4, hx(0x8a5a2a))
	}
	b.ellipse(15, float64(hy), 3, 2.6, body)
	b.rect(13, hy-4, 2, 2, body) // ear
	b.set(16, hy-1, hx(0x101010))
	b.set(18, hy, hx(0x2a1a14))
	b.rect(12, y+3, 2, 2, dark)
	b.rect(7, y+3, 3, 2, dark)
	return b
}

func genBoarFrame(frame, coat int) *pixBuf {
	b := newPix(38, 26)
	b.ox, b.oy = 1, 1
	body, dark, light := hx(0x5a4030), hx(0x2a1c14), hx(0x7a5a44)
	switch coat {
	case 1:
		body, dark, light = hx(0x4a4440), hx(0x221e1c), hx(0x6a625c)
	case 2:
		body, dark, light = hx(0x6a4a2a), hx(0x3a2814), hx(0x8a6a44)
	}
	bob := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		bob = (frame - fWalk0) % 2
	}
	leg := func(x int, i int, col c4) {
		dx, lift := legSwing(frame, i)
		fx := float64(x) + dx*2.5
		b.rect(int(fx)-1, 17, 3, 6-int(lift*2), col)
		b.rect(int(fx)-1, 22-int(lift*2), 3, 2, hx(0x14100c))
	}
	leg(10, 2, shade(body, 0.75))
	leg(24, 0, shade(body, 0.75))
	b.ellipse(17, float64(13-bob), 13, 7.5, body)
	b.ellipse(16, float64(10-bob), 9, 3, light)
	for x := 8; x <= 26; x += 2 { // the bristly ridge
		b.rect(x, 5-bob+(x%4)/2, 1, 2, dark)
	}
	leg(13, 3, body)
	leg(27, 1, body)
	b.ellipse(29, float64(13-bob), 6, 5.5, body) // head
	b.rect(24, 6-bob, 3, 4, dark)                // ear
	b.ellipse(34, float64(15-bob), 2.6, 2.4, hx(0x8a6a5a))
	b.set(34, 14-bob, hx(0x2a1814))
	b.set(28, 11-bob, hx(0x101010))
	b.line(32, 16-bob, 34, 13-bob, hx(0xf4f0e0)) // tusk
	b.line(3, 11-bob, 1, 14-bob, dark)           // thin tail
	if frame == fPeck0 || frame == fPeck1 {
		b.ellipse(33, float64(19+(frame-fPeck0)), 3, 2.4, hx(0x8a6a5a)) // rooting in the ground
	}
	return b
}

func genCrocFrame(frame, coat int) *pixBuf {
	b := newPix(56, 18)
	b.ox, b.oy = 1, 1
	// bright greens with a pale belly: a dark green crocodile vanished on the night grass
	body, dark, belly := hx(0x78b850), hx(0x2a5a20), hx(0xf0e8b0)
	switch coat {
	case 1:
		body, dark, belly = hx(0x98a858), hx(0x4a5424), hx(0xf4ecc0)
	case 2:
		body, dark, belly = hx(0x58b088), hx(0x1a4a34), hx(0xe8f0c8)
	}
	sw := 0.0
	if frame >= fWalk0 && frame <= fWalk3 {
		sw = float64(frame - fWalk0)
	}
	// tail to snout, low and long
	for i := 0; i < 20; i++ {
		t := float64(i) / 19
		x := 2 + t*34
		y := 11 + math.Sin(t*5-sw*math.Pi/2)*(1-t)*1.6
		r := 1.3 + 3.4*math.Min(1, t*1.6)
		b.ellipse(x, y, r, r*0.8, body)
		b.ellipse(x, y+r*0.5, r*0.8, r*0.3, belly)
		if i%2 == 0 {
			b.set(int(x), int(y-r*0.8), dark) // the ridges on the back
		}
	}
	for i, lx := range []float64{14, 28} { // short legs
		dx, lift := legSwing(frame, i*2)
		b.rect(int(lx+dx*2)-1, 14-int(lift), 3, 3, dark)
	}
	open := 0
	switch frame {
	case fWarn:
		open = 3
	case fKick:
		open = 1
	}
	// head and the long jaws
	b.ellipse(40, 10, 4.5, 3.5, body)
	b.rect(42, 8-open, 12, 3, body) // upper jaw
	b.rect(42, 11, 12, 2, shade(body, 0.85))
	if open > 0 {
		for x := 44; x < 54; x += 2 {
			b.set(x, 11-open+2, hx(0xf4f4e8)) // teeth
			b.set(x+1, 11, hx(0xf4f4e8))
		}
		b.rect(43, 11-open+2, 10, open-1, hx(0xc84a4a))
	} else {
		for x := 44; x < 54; x += 3 {
			b.set(x, 11, hx(0xf4f4e8))
		}
	}
	b.ellipse(40, 7-float64(open)/2, 1.6, 1.4, dark) // eye bump
	b.set(41, 7-open/2, hx(0xe8d040))
	return b
}

// genBigCatFrame: a cheetah (slim, spotted, the black "tear" lines) or a lion (tawny, with a big mane).
func genBigCatFrame(frame, coat int, lion bool) *pixBuf {
	b := newPix(50, 34)
	b.ox, b.oy = 2, 2
	body, dark, light, mane := hx(0xe0b050), hx(0x8a6420), hx(0xf4dc98), hx(0x14100c)
	if lion {
		body, dark, light, mane = hx(0xc89a50), hx(0x7a5a28), hx(0xe8c88a), hx(0x8a4a1a)
		switch coat {
		case 1:
			mane = hx(0x3a2414) // dark mane
		case 2:
			mane = hx(0xc87a2a) // golden mane
		}
	} else if coat == 1 {
		body, light = hx(0xd8a040), hx(0xf0d088)
	}
	const ground = 31.0
	by := 17.0
	if frame >= fWalk0 && frame <= fWalk3 {
		by -= float64((frame - fWalk0) % 2)
	}
	leg := func(x float64, i int, col c4) {
		dx, lift := legSwing(frame, i)
		fx, fy := x+dx*5, ground-lift*3
		mx, my, ex, ey := ik2(x, by+3, fx, fy, 7, 0.5, 0.2)
		thickLine(b, x, by+3, mx, my, 2, col)
		thickLine(b, mx, my, ex, ey, 2, shade(col, 0.92))
		b.rect(int(ex)-1, int(ey), 4, 2, shade(col, 0.8))
	}
	leg(13, 2, shade(body, 0.78))
	leg(31, 0, shade(body, 0.78))
	rx := 13.0
	if lion {
		rx = 14
	}
	b.ellipse(22, by, rx, 6, body)
	b.ellipse(22, by-2, rx-3, 2.6, light)
	b.ellipse(22, by+3, rx-5, 2.2, hx(0xf4ead0)) // pale belly
	if !lion {
		for _, sp := range [][2]int{{12, -2}, {16, 1}, {19, -3}, {23, 0}, {27, -2}, {30, 2}, {14, 3}, {25, 3}, {33, -1}} {
			b.rect(sp[0], int(by)+sp[1], 2, 1, mane)
		}
	}
	leg(16, 3, body)
	leg(34, 1, body)
	// tail: long; the lion's with a dark tuft
	b.line(9, int(by)-2, 4, int(by)+4, body)
	b.line(4, int(by)+4, 3, int(by)+9, body)
	if lion {
		b.ellipse(3, by+10, 1.6, 1.6, mane)
	} else {
		b.set(3, int(by)+9, mane)
		b.set(3, int(by)+8, mane)
	}
	// head
	hx0, hy0 := 38.0, by-5
	if frame == fPeck0 || frame == fPeck1 {
		hx0, hy0 = 40, by+1+float64(frame-fPeck0)
	}
	if lion {
		b.ellipse(hx0-1, hy0+1, 7.5, 7.5, mane)
		b.ellipse(hx0-2, hy0+1, 6, 6.5, shade(mane, 0.85))
	}
	b.ellipse(hx0+1, hy0, 4.5, 4, body)
	b.ellipse(hx0+4, hy0+2, 2.6, 2, light) // muzzle
	b.set(int(hx0)+6, int(hy0)+1, hx(0x2a1a14))
	b.set(int(hx0)+2, int(hy0)-1, hx(0x101010))
	b.rect(int(hx0)-2, int(hy0)-5, 2, 2, dark) // ear
	if !lion {
		b.line(int(hx0)+2, int(hy0), int(hx0)+3, int(hy0)+3, mane) // the black tear line
	}
	return b
}
