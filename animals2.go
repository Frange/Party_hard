package main

import (
	"math"
)

// ---------------------------------------------------------------------------
// More animals: cat, chick, bull, elephant, giraffe and kangaroo.
// All sprites face right; the same frame constants as the other animals are used.
// ---------------------------------------------------------------------------

const (
	animCat = iota + 3
	animBull
	animElephant
	animGiraffe
	animKangaroo
	animChick
)

const (
	aCharge = 6
	aSpray  = 7
)

func init() {
	animalKinds = []string{"horse", "hen", "dog", "cat", "bull", "elephant", "giraffe", "kangaroo", "chick"}
}

func legSwing(frame int, i int) (dx, lift float64) {
	if frame < fWalk0 || frame > fWalk3 {
		return 0, 0
	}
	ph := float64(frame-fWalk0) / 4 * 2 * math.Pi
	p := ph + [4]float64{0, math.Pi, math.Pi, 0}[i]
	return math.Sin(p), math.Max(0, math.Cos(p))
}

var bullCoats = []coatCols{
	{hx(0x6a3a22), hx(0x2e1a0e), hx(0x8a5a38), hx(0xe8e0d0)},
	{hx(0x26222a), hx(0x0e0c10), hx(0x3e3a44), hx(0xe8e0d0)},
	{hx(0xd8c8a8), hx(0x8a6a48), hx(0xf0e4cc), hx(0xf8f4ec)},
}

func genBullFrame(frame, coat int) *pixBuf {
	b := newPix(52, 34)
	b.ox, b.oy = 2, 2
	cc := bullCoats[coat%len(bullCoats)]
	body, dark, light, horn := cc.body, cc.dark, cc.light, cc.mane
	const ground = 32.0
	charge := frame == fKick
	bob := 0.0
	if frame >= fWalk0 && frame <= fWalk3 {
		bob = float64((frame - fWalk0) % 2)
	}
	by := 16.0 - bob
	// tail
	b.line(12, int(by)-4, 8, int(by)+8, dark)
	b.rect(7, int(by)+8, 3, 3, dark)
	leg := func(x float64, i int, col c4) {
		dx, lift := legSwing(frame, i)
		fx, fy := x+dx*4.5, ground-lift*3.5
		if frame == fWarn && i == 1 {
			fx, fy = x-6, ground-6 // pawing
		}
		if charge {
			fx = x + []float64{-6, 6, -4, 4}[i]
			fy = ground - []float64{0, 2, 3, 0}[i]
		}
		mx, my, ex, ey := ik2(x, by+5, fx, fy, 7, 0.5, 0.2)
		thickLine(b, x, by+5, mx, my, 3, col)
		thickLine(b, mx, my, ex, ey, 2, shade(col, 0.92))
		b.rect(int(ex)-1, int(ey), 4, 2, c(0x1a1410))
	}
	leg(14, 2, shade(body, 0.7))
	leg(36, 0, shade(body, 0.7))
	b.ellipse(25, by, 15, 8.5, body)
	b.ellipse(26, by-3, 11, 4, light)
	b.ellipse(15, by+1, 7, 7, shade(body, 0.92))
	b.ellipse(35, by-3, 8, 7, shade(body, 1.08)) // hump
	leg(16, 3, body)
	leg(34, 1, body)
	hy := 19.0
	hx0 := 43.0
	switch frame {
	case fGraze:
		hy, hx0 = 27, 43
	case fWarn:
		hy = 22
	case fKick:
		hy, hx0 = 22, 45
	}
	thickLine(b, 36, by-1, hx0-3, hy-2, 7, shade(body, 1.02))
	b.ellipse(hx0, hy, 6, 5, body)
	b.ellipse(hx0+4, hy+2, 3.4, 3, light)
	b.set(int(hx0)+6, int(hy)+2, c(0x14100c))
	b.set(int(hx0)+1, int(hy)-1, c(0xf4f4f0))
	b.set(int(hx0)+2, int(hy)-1, c(0x14100c))
	// horns
	b.line(int(hx0)-3, int(hy)-4, int(hx0)-5, int(hy)-9, horn)
	b.line(int(hx0)-5, int(hy)-9, int(hx0)-3, int(hy)-11, horn)
	b.line(int(hx0)+2, int(hy)-4, int(hx0)+3, int(hy)-9, horn)
	b.line(int(hx0)+3, int(hy)-9, int(hx0)+5, int(hy)-10, horn)
	b.ellipse(hx0-4, hy-1, 2, 2.6, dark)
	return b
}

var elephantCoats = []coatCols{
	{hx(0x8a8c9a), hx(0x5a5c6a), hx(0xa8aab8), hx(0xf2eee0)},
	{hx(0x6e707e), hx(0x464854), hx(0x8a8c9a), hx(0xf2eee0)},
	{hx(0xa8a49c), hx(0x74706a), hx(0xc4c0b6), hx(0xf2eee0)},
}

func genElephantFrame(frame, coat int) *pixBuf {
	b := newPix(66, 52)
	b.ox, b.oy = 2, 2
	cc := elephantCoats[coat%len(elephantCoats)]
	body, dark, light, tusk := cc.body, cc.dark, cc.light, cc.mane
	const ground = 50.0
	bob := 0.0
	if frame >= fWalk0 && frame <= fWalk3 {
		bob = float64((frame - fWalk0) % 2)
	}
	by := 25.0 - bob
	leg := func(x float64, i int, col c4) {
		dx, lift := legSwing(frame, i)
		fx, fy := x+dx*3.5, ground-lift*3
		thickLine(b, x, by+8, fx, fy, 7, col)
		b.rect(int(fx)-3, int(fy)-2, 7, 3, shade(col, 0.7))
		b.rect(int(fx)-2, int(fy), 1, 1, tusk)
		b.rect(int(fx)+1, int(fy), 1, 1, tusk)
	}
	b.line(15, int(by)-3, 12, int(by)+10, dark)
	b.rect(11, int(by)+10, 3, 3, dark)
	leg(17, 2, shade(body, 0.72))
	leg(41, 0, shade(body, 0.72))
	b.ellipse(31, by, 20, 14, body)
	b.ellipse(31, by-6, 15, 5, light)
	b.ellipse(19, by+2, 9, 10, shade(body, 0.94))
	leg(21, 3, body)
	leg(39, 1, body)
	hx0, hy := 52.0, 23.0-bob*0.5
	// ear
	flap := 0.0
	if frame == fWarn {
		flap = -2
	}
	b.ellipse(hx0-6, hy+1+flap, 7, 10, dark)
	b.ellipse(hx0-5, hy+1+flap, 5.5, 8.5, shade(body, 1.05))
	// head
	b.ellipse(hx0+1, hy, 9.5, 9.5, body)
	b.ellipse(hx0+3, hy-3, 6, 4, light)
	b.set(int(hx0)+6, int(hy)-2, c(0x14100c))
	b.set(int(hx0)+7, int(hy)-2, c(0xf4f4f0))
	// trunk
	switch frame {
	case fWarn: // raised, spraying
		thickLine(b, hx0+7, hy+4, hx0+13, hy-2, 5, body)
		thickLine(b, hx0+13, hy-2, hx0+14, hy-14, 4, body)
		b.rect(int(hx0)+12, int(hy)-15, 4, 2, dark)
	case fGraze:
		thickLine(b, hx0+7, hy+4, hx0+9, hy+14, 5, body)
		thickLine(b, hx0+9, hy+14, hx0+11, 46, 4, body)
		b.rect(int(hx0)+9, 47, 5, 2, dark)
	default:
		thickLine(b, hx0+7, hy+4, hx0+11, hy+14, 5, body)
		thickLine(b, hx0+11, hy+14, hx0+9, hy+22, 4, body)
		b.rect(int(hx0)+7, int(hy)+22, 5, 2, dark)
	}
	// tusk
	b.line(int(hx0)+7, int(hy)+6, int(hx0)+11, int(hy)+10, tusk)
	b.line(int(hx0)+8, int(hy)+6, int(hx0)+12, int(hy)+9, tusk)
	return b
}

var giraffeCoats = []coatCols{
	{hx(0xe0b050), hx(0x8a5a22), hx(0xf0cc78), hx(0x5a3a18)},
	{hx(0xd8a040), hx(0x7a4a1a), hx(0xe8c068), hx(0x4a2a10)},
	{hx(0xe8c878), hx(0x9a7a3a), hx(0xf4dc98), hx(0x6a4a20)},
}

func genGiraffeFrame(frame, coat int) *pixBuf {
	b := newPix(46, 70)
	b.ox, b.oy = 2, 2
	cc := giraffeCoats[coat%len(giraffeCoats)]
	body, spot, light, mane := cc.body, cc.dark, cc.light, cc.mane
	const ground = 68.0
	bob := 0.0
	if frame >= fWalk0 && frame <= fWalk3 {
		bob = float64((frame - fWalk0) % 2)
	}
	by := 38.0 - bob
	leg := func(x float64, i int, col c4) {
		dx, lift := legSwing(frame, i)
		fx, fy := x+dx*4, ground-lift*3
		mx, my, ex, ey := ik2(x, by+3, fx, fy, 15, 0.3, 0.2)
		thickLine(b, x, by+3, mx, my, 3, col)
		thickLine(b, mx, my, ex, ey, 2, shade(col, 0.95))
		b.rect(int(ex)-1, int(ey)-1, 4, 3, c(0x2a1a10))
	}
	b.line(9, int(by)-3, 6, int(by)+12, mane)
	b.rect(5, int(by)+12, 3, 3, mane)
	leg(11, 2, shade(body, 0.75))
	leg(28, 0, shade(body, 0.75))
	b.ellipse(19, by, 13, 7, body)
	b.ellipse(20, by-3, 10, 3, light)
	leg(13, 3, body)
	leg(26, 1, body)
	// neck and head
	nx0, ny0 := 28.0, by-3
	hx0, hy := 36.0, 12.0
	switch frame {
	case fGraze:
		hx0, hy = 37, 62
		thickLine(b, nx0, ny0, 35, 46, 6, body)
		thickLine(b, 35, 46, hx0, hy-3, 5, body)
	case fWarn:
		hx0, hy = 38, 8
		thickLine(b, nx0, ny0, hx0-3, hy+3, 6, body)
	default:
		thickLine(b, nx0, ny0, hx0-2, hy+3, 6, body)
	}
	b.ellipse(hx0, hy, 4.6, 3.6, body)
	b.ellipse(hx0+4, hy+1, 3.2, 2.4, light)
	b.set(int(hx0)+1, int(hy)-1, c(0x14100c))
	b.line(int(hx0)-2, int(hy)-3, int(hx0)-3, int(hy)-7, mane)
	b.set(int(hx0)-3, int(hy)-8, mane)
	b.line(int(hx0)+1, int(hy)-3, int(hx0)+1, int(hy)-7, mane)
	b.set(int(hx0)+1, int(hy)-8, mane)
	b.ellipse(hx0-4, hy-1, 1.6, 2.4, shade(body, 0.8))
	// spots (fixed pattern so frames match)
	for _, p := range [][3]float64{{14, 0, 2}, {20, -2, 2.2}, {25, 1, 2}, {17, 3, 1.6}, {23, 3, 1.6}, {11, 2, 1.6}} {
		b.ellipse(p[0], by+p[1], p[2], p[2]*0.8, spot)
	}
	for i := 0; i < 5; i++ {
		t := float64(i) / 5
		sx := nx0 + (hx0-3-nx0)*t
		sy := ny0 + (hy+3-ny0)*t
		if frame == fGraze {
			sx, sy = nx0+(35-nx0)*t, ny0+(46-ny0)*t
		}
		b.set(int(sx)-1, int(sy), spot)
		b.set(int(sx)+1, int(sy)+1, spot)
	}
	return b
}

var kangarooCoats = []coatCols{
	{hx(0xb8683a), hx(0x7a4022), hx(0xe0a070), hx(0x3a2014)},
	{hx(0x8a8a92), hx(0x5a5a62), hx(0xb4b4bc), hx(0x2a2a30)},
	{hx(0xd8b080), hx(0x9a7448), hx(0xf0d4a8), hx(0x4a3020)},
}

func genKangarooFrame(frame, coat int) *pixBuf {
	b := newPix(30, 44)
	b.ox, b.oy = 1, 1
	cc := kangarooCoats[coat%len(kangarooCoats)]
	body, dark, light, eye := cc.body, cc.dark, cc.light, cc.mane
	const ground = 42.0
	hop := 0.0
	tuck := false
	if frame >= fWalk0 && frame <= fWalk3 {
		hop = []float64{0, 4, 7, 3}[frame-fWalk0]
		tuck = frame == fWalk2
	}
	by := 26.0 - hop
	lean := 0.0
	if frame == fKick {
		lean = 3
	}
	// tail
	thickLine(b, 9, by+5, 2, ground-2-hop*0.4, 3, dark)
	// legs
	if tuck {
		b.ellipse(11, by+9, 5, 4, body)
		b.rect(9, int(by)+12, 9, 2, dark)
	} else {
		b.ellipse(11, by+8, 5, 7, shade(body, 0.9))
		b.rect(8, int(ground-hop*0.3)-2, 11, 3, dark)
		b.rect(18, int(ground-hop*0.3)-1, 2, 2, c(0x1a1410))
	}
	// torso
	b.ellipse(13+lean*0.5, by, 6.5, 11, body)
	b.ellipse(15+lean*0.5, by+2, 3.5, 6, light)
	// head
	hx0, hy := 17+lean, by-13
	b.ellipse(hx0, hy, 4, 4.6, body)
	b.ellipse(hx0+3.5, hy+1.5, 3, 2.4, light)
	b.set(int(hx0)+1, int(hy), eye)
	b.set(int(hx0)+6, int(hy)+1, eye)
	// ears
	b.line(int(hx0)-2, int(hy)-3, int(hx0)-3, int(hy)-9, body)
	b.line(int(hx0)-1, int(hy)-3, int(hx0)-2, int(hy)-9, body)
	b.line(int(hx0)+1, int(hy)-4, int(hx0)+1, int(hy)-10, shade(body, 0.85))
	b.set(int(hx0)+1, int(hy)-10, dark)
	// arms
	switch frame {
	case fWarn:
		thickLine(b, 17, by-6, 23, by-9, 2, shade(body, 0.85))
		thickLine(b, 16, by-4, 22, by-6, 2, body)
		b.ellipse(23, by-9, 2, 2, dark)
		b.ellipse(22, by-6, 2, 2, dark)
	case fKick:
		thickLine(b, 18, by-5, 28, by-6, 2, body)
		b.ellipse(28, by-6, 2.4, 2.4, c(0xd83030)) // boxing glove
		thickLine(b, 16, by-3, 21, by-1, 2, shade(body, 0.85))
	default:
		thickLine(b, 17, by-4, 20, by+1, 2, body)
		thickLine(b, 15, by-3, 18, by+2, 2, shade(body, 0.85))
	}
	return b
}

var catCoats = []coatCols{
	{hx(0xd88a3a), hx(0x8a4a1a), hx(0xf0b868), hx(0x1a1410)},
	{hx(0x2a2830), hx(0x0e0c12), hx(0x44424c), hx(0xe8d040)},
	{hx(0xc8c8d0), hx(0x7a7a86), hx(0xf4f4f8), hx(0x1a1410)},
}

func genCatFrame(frame, coat int) *pixBuf {
	b := newPix(24, 18)
	b.ox, b.oy = 1, 1
	cc := catCoats[coat%len(catCoats)]
	body, dark, light, eye := cc.body, cc.dark, cc.light, cc.mane
	const ground = 16.0
	if frame == fSit {
		b.line(6, 15, 2, 12, dark)
		b.line(6, 15, 3, 14, dark)
		b.ellipse(9, 11, 4, 5.5, body)
		b.ellipse(10, 12, 2, 3.5, light)
		b.ellipse(11, 5, 3.4, 3, body)
		b.set(10, 5, eye)
		b.set(13, 5, eye)
		b.rect(9, 1, 2, 2, dark)
		b.rect(13, 1, 2, 2, dark)
		b.rect(9, 3, 1, 1, body)
		b.rect(14, 3, 1, 1, body)
		return b
	}
	bob := 0.0
	if frame >= fWalk0 && frame <= fWalk3 {
		bob = float64((frame - fWalk0) % 2)
	}
	by := 10.0 - bob*0.5
	curl := 1.0
	if frame == fWalk1 || frame == fWalk3 {
		curl = 3
	}
	thickLine(b, 5, by-1, 1, by-3-curl, 1, dark)
	b.set(1, int(by-4-curl), body)
	for i, x := range []float64{7, 9, 14, 16} {
		dx, lift := legSwing(frame, i)
		col := body
		if i < 2 {
			col = shade(body, 0.75)
		}
		thickLine(b, x, by+2, x+dx*2, ground-lift, 1, col)
	}
	b.ellipse(11, by, 6.5, 3.4, body)
	b.ellipse(11, by-1, 4.5, 1.6, light)
	b.ellipse(18, by-2.5, 3.2, 2.8, body)
	b.set(19, int(by)-3, eye)
	b.set(21, int(by)-2, c(0xf08a9a))
	b.rect(16, int(by)-6, 2, 2, dark)
	b.rect(19, int(by)-6, 2, 2, dark)
	b.set(17, int(by)-5, body)
	b.set(20, int(by)-5, body)
	return b
}

func genChickFrame(frame int) *pixBuf {
	b := newPix(14, 12)
	b.ox, b.oy = 1, 1
	body, light := hx(0xf8d838), hx(0xfff08a)
	bob := 0
	hy, hx0 := 4, 10
	if frame == fPeck0 || frame == fPeck1 {
		hy, hx0 = 7+(frame-fPeck0), 11
	} else if frame >= fWalk0 && frame <= fWalk3 {
		bob = (frame - fWalk0) % 2
	}
	b.ellipse(6, 7+float64(bob), 4.2, 3.4, body)
	b.ellipse(7, 6+float64(bob), 2.5, 1.6, light)
	b.ellipse(float64(hx0), float64(hy), 2.4, 2.4, body)
	b.rect(hx0+2, hy, 2, 1, hx(0xf0902a))
	b.set(hx0+1, hy-1, hx(0x14100c))
	o := 0
	if frame >= fWalk0 && frame <= fWalk3 {
		o = []int{1, 0, -1, 0}[frame-fWalk0]
	}
	b.line(5, 10, 5+o, 11, hx(0xf0902a))
	b.line(8, 10, 8-o, 11, hx(0xf0902a))
	return b
}

func buildMoreAnimalSprites(out AnimalSprites) {
	for coat := 0; coat < 3; coat++ {
		for f := fStand; f <= fKick; f++ {
			out[animalKey(animBull, coat, f)] = genBullFrame(f, coat).outlinedSel(0.4).img()
			out[animalKey(animKangaroo, coat, f)] = genKangarooFrame(f, coat).outlinedSel(0.4).img()
		}
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fGraze, fWarn} {
			out[animalKey(animElephant, coat, f)] = genElephantFrame(f, coat).outlinedSel(0.4).img()
			out[animalKey(animGiraffe, coat, f)] = genGiraffeFrame(f, coat).outlinedSel(0.4).img()
		}
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fSit} {
			out[animalKey(animCat, coat, f)] = genCatFrame(f, coat).outlinedSel(0.4).img()
		}
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fPeck0, fPeck1} {
			out[animalKey(animChick, coat, f)] = genChickFrame(f).img()
		}
	}
}

// ---------------------------------------------------------------------------
// behaviour
// ---------------------------------------------------------------------------

func (g *Game) animalWander(a *Animal, speed float64, dt float64, grazeP float64) {
	a.timer -= dt
	switch a.state {
	case aIdle, aGraze:
		if a.timer <= 0 {
			if g.rng.Float64() < 0.55 {
				a.path = g.lvl.Path(a.pos, g.wanderTarget(a))
				a.state, a.timer = aWalk, 20
			} else if a.state == aIdle && g.rng.Float64() < grazeP {
				a.state, a.timer = aGraze, 3+g.rng.Float64()*4
			} else {
				a.state, a.timer = aIdle, 2+g.rng.Float64()*3
			}
		}
	case aWalk:
		if g.animalWalk(a, speed, dt) || a.timer <= 0 {
			a.state, a.timer = aIdle, 1.5+g.rng.Float64()*3
		}
	}
}

// victimAhead returns someone standing in front of the animal within reach.
func (g *Game) victimAhead(a *Animal, reach, width float64) (any, Vec) {
	front := func(p Vec) bool {
		dx, dy := p.X-a.pos.X, p.Y-a.pos.Y
		return dx*a.face > 2 && math.Abs(dx) < reach && math.Abs(dy) < width
	}
	if p := g.p; !p.dead && !p.hidden && front(p.pos) {
		return p, p.pos
	}
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" && gu.state != gSleep && front(gu.pos) {
			return gu, gu.pos
		}
	}
	return nil, Vec{}
}

func (g *Game) animalKill(v any, msg string) bool {
	switch t := v.(type) {
	case *Player:
		if !t.dead {
			g.die(msg)
			return true
		}
	case *Guest:
		if !t.out() {
			g.killGuest(t, false)
			return true
		}
	}
	return false
}

func (g *Game) updateBull(a *Animal, dt float64) {
	switch a.state {
	case aIdle, aGraze, aWalk:
		g.animalWander(a, 18, dt, 0.4)
		if v, _ := g.victimAhead(a, 80, 26); v != nil {
			a.warnT += dt
			if a.warnT > 0.9 {
				a.state, a.timer, a.victim, a.warnT = aWarn, 0.9, v, 0
				a.path = nil
				a.icon, a.iconT = "MUUU!", 1.2
				g.snd.Play("alert")
			}
		} else {
			a.warnT = math.Max(0, a.warnT-dt)
		}
	case aWarn:
		a.timer -= dt
		if t := posOf(a.victim); t != (Vec{}) {
			if t.X < a.pos.X {
				a.face = -1
			} else {
				a.face = 1
			}
		}
		if a.timer <= 0 {
			a.state, a.timer = aCharge, 1.9
			a.hit = false
		}
	case aCharge:
		a.timer -= dt
		t := posOf(a.victim)
		if t == (Vec{}) {
			a.state, a.timer = aIdle, 3
			return
		}
		a.moving = true
		a.victimRepath(g, t, dt)
		g.animalWalk(a, 125, dt)
		if dist(a.pos, t) < 16 {
			if g.animalKill(a.victim, "Un toro te ha embestido") {
				g.snd.Play("thud")
				g.burst(Vec{a.pos.X, a.pos.Y - 14}, 10, 80, rgb(0xd8d0c0), 0.4, 2, 100)
			}
			a.state, a.timer = aIdle, 4
		} else if a.timer <= 0 {
			a.state, a.timer = aIdle, 3
		}
	}
}

func (a *Animal) victimRepath(g *Game, t Vec, dt float64) {
	a.repath -= dt
	if a.repath <= 0 || len(a.path) == 0 {
		a.repath = 0.25
		a.path = g.lvl.Path(a.pos, t)
	}
}

func posOf(v any) Vec {
	switch t := v.(type) {
	case *Player:
		return t.pos
	case *Guest:
		return t.pos
	}
	return Vec{}
}

func (g *Game) updateElephant(a *Animal, dt float64) {
	switch a.state {
	case aSpray:
		a.timer -= dt
		a.sprayT += dt
		if a.sprayT > 0.05 {
			a.sprayT = 0
			g.burst(Vec{a.pos.X + a.face*28, a.pos.Y - 46}, 2, 60, rgb(0x90e0ff), 0.6, 2, 160)
		}
		if a.timer <= 0 {
			a.state, a.timer = aIdle, 3
		}
	default:
		g.animalWander(a, 13, dt, 0.35)
		if a.state == aIdle && g.rng.Float64() < 0.0025 {
			a.state, a.timer = aSpray, 2
			a.icon, a.iconT = "TUUUT!", 1.4
			g.snd.Play("splash")
		}
	}
}

func (g *Game) updateGiraffe(a *Animal, dt float64) {
	g.animalWander(a, 15, dt, 0.5)
}

func (g *Game) updateKangaroo(a *Animal, dt float64) {
	switch a.state {
	case aWarn:
		a.timer -= dt
		if a.timer <= 0 {
			if v, _ := g.victimAhead(a, 44, 22); v != nil {
				a.state, a.timer, a.victim, a.hit = aKick, 0.45, v, false
			} else {
				a.state, a.timer = aIdle, 2
			}
		}
	case aKick:
		a.timer -= dt
		if !a.hit && a.timer < 0.25 {
			a.hit = true
			g.snd.Play("thud")
			switch v := a.victim.(type) {
			case *Player:
				if !v.dead && !v.hidden && dist(v.pos, a.pos) < 50 {
					v.knockT, v.danceT = 3.5, 0
					g.say("¡Un canguro te ha noqueado!")
					a.icon, a.iconT = "BOING!", 1.2
				}
			case *Guest:
				if !v.out() && dist(v.pos, a.pos) < 50 {
					v.state, v.timer, v.path = gIdle, 3, nil
					v.icon, v.iconT = "x_x", 2
				}
			}
		}
		if a.timer <= 0 {
			a.state, a.timer = aIdle, 3
		}
	default:
		g.animalWander(a, 38, dt, 0)
		if v, _ := g.victimAhead(a, 40, 20); v != nil && a.state != aWalk {
			a.warnT += dt
			if a.warnT > 1.2 {
				a.state, a.timer, a.victim, a.warnT = aWarn, 0.7, v, 0
			}
		} else {
			a.warnT = math.Max(0, a.warnT-dt)
		}
	}
}

func (g *Game) updateCat(a *Animal, dt float64) {
	scare := false
	if d := dist(g.p.pos, a.pos); d < 30 && !g.p.hidden {
		scare = true
	}
	for _, o := range g.animals {
		if o.kind == animDog && dist(o.pos, a.pos) < 40 {
			scare = true
		}
	}
	if scare && a.state != aFlee {
		a.state, a.timer = aFlee, 1.6
		a.path = g.lvl.Path(a.pos, g.lvl.RandomIn(g.rng, a.area))
		a.icon, a.iconT = "MIAU!", 1
	}
	switch a.state {
	case aFlee:
		a.timer -= dt
		if g.animalWalk(a, 70, dt) || a.timer <= 0 {
			a.state, a.timer = aIdle, 1
		}
	default:
		g.animalWander(a, 22, dt, 0)
		if a.state == aIdle && a.timer < 0.05 && g.rng.Float64() < 0.5 {
			a.state, a.timer = aGraze, 4+g.rng.Float64()*5 // sits
		}
	}
}

func (g *Game) updateChick(a *Animal, dt float64) {
	scare := dist(g.p.pos, a.pos) < 26 && !g.p.hidden
	if scare && a.state != aFlee {
		a.state, a.timer = aFlee, 1.2
		a.path = g.lvl.Path(a.pos, g.lvl.RandomIn(g.rng, a.area))
		a.icon, a.iconT = "PIO!", 0.8
	}
	switch a.state {
	case aFlee:
		a.timer -= dt
		if g.animalWalk(a, 40, dt) || a.timer <= 0 {
			a.state, a.timer = aIdle, 1
		}
	case aIdle, aGraze:
		a.timer -= dt
		if a.timer <= 0 {
			if g.rng.Float64() < 0.5 {
				a.state, a.timer = aGraze, 1+g.rng.Float64()*2
			} else {
				a.path = g.lvl.Path(a.pos, g.lvl.RandomIn(g.rng, Rect{a.pos.X - 30, a.pos.Y - 20, 60, 40}.clip(a.area)))
				a.state, a.timer = aWalk, 8
			}
		}
	case aWalk:
		a.timer -= dt
		if g.animalWalk(a, 16, dt) || a.timer <= 0 {
			a.state, a.timer = aIdle, 0.6
		}
	}
}

// animalFrameMore: sprite frame for the newer animals.
func (g *Game) animalFrameMore(a *Animal) int {
	switch a.kind {
	case animBull:
		switch a.state {
		case aGraze:
			return fGraze
		case aWarn:
			return fWarn
		case aCharge:
			return fKick
		case aWalk:
			return fWalk0 + int(a.anim*6)%4
		}
	case animElephant:
		switch a.state {
		case aGraze:
			return fGraze
		case aSpray:
			return fWarn
		case aWalk:
			return fWalk0 + int(a.anim*4)%4
		}
	case animGiraffe:
		switch a.state {
		case aGraze:
			return fGraze
		case aWalk:
			return fWalk0 + int(a.anim*4)%4
		}
	case animKangaroo:
		switch a.state {
		case aWarn:
			return fWarn
		case aKick:
			return fKick
		case aWalk:
			return fWalk0 + int(a.anim*7)%4
		}
	case animCat:
		switch {
		case a.state == aGraze:
			return fSit
		case a.moving:
			return fWalk0 + int(a.anim*9)%4
		}
	case animChick:
		switch {
		case a.state == aGraze:
			return fPeck0 + int(a.anim*5)%2
		case a.moving:
			return fWalk0 + int(a.anim*12)%4
		}
	}
	return fStand
}
