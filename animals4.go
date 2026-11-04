package main

// Deer, and the two famous riders: Don Quijote on his skinny little horse
// (lance and shield) and Sancho Panza on a small donkey.

const (
	animDeer = iota + 13
	animQuijote
	animSancho
)

func init() {
	animalKinds = append(animalKinds, "deer", "quijote", "sancho")
}

func genDeerFrame(frame, coat int) *pixBuf {
	b := newPix(50, 40)
	b.ox, b.oy = 2, 2
	body, dark, light := hx(0xa8703a), hx(0x6a4220), hx(0xd0a070)
	if coat == 1 {
		body, dark, light = hx(0x8a5a30), hx(0x4a2c14), hx(0xb88a5a)
	} else if coat == 2 {
		body, dark, light = hx(0xc09060), hx(0x7a5430), hx(0xe8c898)
	}
	const ground = 37.0
	by := 20.0
	if frame >= fWalk0 && frame <= fWalk3 {
		by -= float64((frame - fWalk0) % 2)
	}
	leg := func(x float64, i int, col c4) {
		dx, lift := legSwing(frame, i)
		fx, fy := x+dx*4, ground-lift*3
		mx, my, ex, ey := ik2(x, by+4, fx, fy, 8, 0.5, 0.2)
		thickLine(b, x, by+4, mx, my, 2, col)
		thickLine(b, mx, my, ex, ey, 1, shade(col, 0.9))
		b.rect(int(ex)-1, int(ey), 3, 2, c(0x2a1a10))
	}
	leg(14, 2, shade(body, 0.72))
	leg(33, 0, shade(body, 0.72))
	b.ellipse(23, by, 14, 7.5, body)
	b.ellipse(24, by-2, 11, 3.5, light)
	b.ellipse(23, by+4, 9, 3, hx(0xf0e4d0)) // pale belly
	b.rect(10, int(by)-3, 3, 3, hx(0xf4f0e4))
	for _, sp := range [][2]int{{18, -3}, {24, -4}, {29, -2}, {21, 0}} {
		b.set(sp[0], int(by)+sp[1], hx(0xf4f0e0))
	}
	leg(17, 3, body)
	leg(36, 1, body)
	// neck and head
	hx0, hy0 := 40.0, by-14
	if frame == fPeck0 || frame == fPeck1 {
		hx0, hy0 = 43, by+8+float64(frame-fPeck0)*2
	}
	thickLine(b, 33, by-4, hx0, hy0+2, 4, body)
	b.ellipse(hx0+2, hy0+1, 4, 3.2, body)
	b.rect(int(hx0)+5, int(hy0)+1, 3, 2, dark) // muzzle
	b.set(int(hx0)+1, int(hy0), c(0x14100c))
	b.rect(int(hx0)-2, int(hy0)-3, 2, 3, dark) // ear
	if frame != fPeck0 && frame != fPeck1 {
		// antlers
		for _, a := range [][4]int{{0, -4, 2, -10}, {2, -10, 0, -14}, {2, -10, 5, -13}, {1, -7, 5, -8}} {
			b.line(int(hx0)+a[0]+1, int(hy0)+a[1], int(hx0)+a[2]+1, int(hy0)+a[3], hx(0xd8c8a0))
		}
	}
	b.line(10, int(by)-2, 7, int(by)+1, dark) // tail
	return b
}

// genRiderFrame draws a mounted rider: quijote (lance, shield, helmet) or sancho (hat, jerkin).
func genRiderFrame(frame, coat int, sancho bool) *pixBuf {
	b := newPix(52, 50)
	b.ox, b.oy = 2, 2
	const ground = 47.0
	bob := 0.0
	if frame >= fWalk0 && frame <= fWalk3 {
		bob = float64((frame - fWalk0) % 2)
	}
	by := 33.0 - bob
	bodyC, darkC := hx(0xb8a890), hx(0x7a6a58)
	if sancho {
		bodyC, darkC = hx(0x8a8a94), hx(0x5a5a64)
	}
	leg := func(x float64, i int, col c4) {
		dx, lift := legSwing(frame, i)
		fx, fy := x+dx*3.5, ground-lift*3
		mx, my, ex, ey := ik2(x, by+4, fx, fy, 6.5, 0.5, 0.2)
		thickLine(b, x, by+4, mx, my, 2, col)
		thickLine(b, mx, my, ex, ey, 1, shade(col, 0.9))
		b.rect(int(ex)-1, int(ey), 3, 2, c(0x2a1a10))
	}
	leg(16, 2, shade(bodyC, 0.75))
	leg(32, 0, shade(bodyC, 0.75))
	b.ellipse(24, by, 13, 6.5, bodyC)
	b.ellipse(24, by-2, 10, 3, shade(bodyC, 1.12))
	leg(19, 3, bodyC)
	leg(35, 1, bodyC)
	// head of the mount
	hx0, hy0 := 40.0, by-6
	if frame == fPeck0 || frame == fPeck1 {
		hx0, hy0 = 42, by+5
	}
	thickLine(b, 33, by-3, hx0, hy0+1, 4, bodyC)
	b.ellipse(hx0+2, hy0+1, 4, 3, bodyC)
	b.rect(int(hx0)+4, int(hy0)+1, 3, 3, darkC)
	b.set(int(hx0)+1, int(hy0), c(0x14100c))
	earH := 5
	if sancho {
		earH = 8
	}
	b.rect(int(hx0)-1, int(hy0)-earH, 2, earH, bodyC)
	if sancho {
		b.rect(int(hx0)+2, int(hy0)-7, 2, 7, bodyC)
	}
	b.line(11, int(by)-2, 8, int(by)+4, darkC) // tail
	// rider
	rx, ry := 24.0, by-9
	if sancho { // round, jolly
		b.ellipse(rx, ry+2, 6.5, 7, hx(0x8a5a2a))
		b.rect(int(rx)-6, int(ry)+1, 12, 2, hx(0xe8d8a0))
		b.ellipse(rx, ry-8, 3.6, 3.6, hx(0xe8b48a))
		b.rect(int(rx)-5, int(ry)-12, 10, 3, hx(0x6a4a2a)) // hat brim
		b.rect(int(rx)-3, int(ry)-15, 6, 4, hx(0x6a4a2a))
		b.set(int(rx)+1, int(ry)-8, c(0x14100c))
		b.rect(int(rx)-1, int(ry)-6, 3, 1, hx(0x3a2a1a))
		b.line(int(rx)+5, int(ry)+2, int(rx)+9, int(ry)+6, hx(0xe8b48a))
		b.rect(int(rx)+7, int(ry)+6, 3, 4, hx(0x7a3a1a)) // wineskin
	} else { // thin knight
		b.rect(int(rx)-3, int(ry)-4, 6, 11, hx(0xb0b4c0))
		b.rect(int(rx)-3, int(ry)-4, 6, 1, hx(0xe0e4ee))
		b.rect(int(rx)-2, int(ry)+6, 4, 3, hx(0x6a4a2a))
		b.ellipse(rx, ry-8, 3, 3.4, hx(0xe8c8a0))
		b.rect(int(rx)-3, int(ry)-12, 6, 3, hx(0xd8c060)) // barber's basin helmet
		b.rect(int(rx)-4, int(ry)-10, 8, 1, hx(0xb89a40))
		b.set(int(rx)+1, int(ry)-8, c(0x14100c))
		b.rect(int(rx)+1, int(ry)-6, 3, 1, hx(0x2a1a10)) // moustache
		// shield on the back arm
		b.ellipse(rx-5, ry+1, 4.2, 5.2, hx(0xc8c8d0))
		b.ellipse(rx-5, ry+1, 2.6, 3.4, hx(0xc03030))
		// the lance, long and raised
		b.line(int(rx)+3, int(ry)+2, int(rx)+26, int(ry)-14, hx(0x8a5a2a))
		b.line(int(rx)+3, int(ry)+3, int(rx)+26, int(ry)-13, hx(0x6a4220))
		b.rect(int(rx)+26, int(ry)-16, 3, 3, hx(0xe8eef8))
		b.rect(int(rx)+10, int(ry)-7, 2, 2, hx(0xc03030)) // pennant
	}
	return b
}

func buildRiderSprites(out AnimalSprites) {
	for coat := 0; coat < 3; coat++ {
		for _, f := range []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fPeck0, fPeck1} {
			out[animalKey(animDeer, coat, f)] = genDeerFrame(f, coat).scaled(0.8).outlinedSel(0.4).img()
			out[animalKey(animQuijote, coat, f)] = genRiderFrame(f, coat, false).scaled(0.8).outlinedSel(0.4).img()
			out[animalKey(animSancho, coat, f)] = genRiderFrame(f, coat, true).scaled(0.8).outlinedSel(0.4).img()
		}
	}
}

// scatterRiders: sometimes Don Quijote and Sancho wander the party together.
func (g *Game) scatterRiders() {
	if base, ok := g.outdoorSpot(); ok && g.rng.Float64() < 0.4 && quijoteUnlocked() {
		area := Rect{base.X - 110, base.Y - 70, 220, 140}
		for i, k := range []int{animQuijote, animSancho} {
			p := g.lvl.NearestFree(Vec{base.X + float64(i)*34, base.Y})
			g.animals = append(g.animals, &Animal{kind: k, pos: p, face: 1, area: area, seed: g.rng.Float64() * 10, timer: g.rng.Float64() * 2, coat: g.rng.Intn(3)})
		}
	}
	if g.rng.Float64() < 0.55 && ownedID("animal:deer") {
		n := 1 + g.rng.Intn(2)
		for i := 0; i < n; i++ {
			pos, ok := g.outdoorSpot()
			if !ok {
				break
			}
			g.animals = append(g.animals, &Animal{kind: animDeer, pos: g.lvl.NearestFree(pos), face: 1, area: Rect{pos.X - 100, pos.Y - 60, 200, 120}, seed: g.rng.Float64() * 10, timer: g.rng.Float64() * 3, coat: g.rng.Intn(3)})
		}
	}
	if pos, ok := g.outdoorSpot(); ok && g.rng.Float64() < 0.3 && ownedID("animal:elephant") {
		g.animals = append(g.animals, &Animal{kind: animElephant, pos: g.lvl.NearestFree(pos), face: -1, area: Rect{pos.X - 120, pos.Y - 70, 240, 140}, seed: g.rng.Float64() * 10, timer: g.rng.Float64() * 3, coat: g.rng.Intn(3)})
	}
}

// aggressiveAnimal: the ones that may roam anywhere, buildings included.
func aggressiveAnimal(k int) bool { return k == animBull || k == animKangaroo }

// outdoorSpot picks a random walkable roam point that is outside any building.
func (g *Game) outdoorSpot() (Vec, bool) {
	if g.lvl.Def.Sky == "space" || len(g.lvl.Def.Roam) == 0 {
		return Vec{}, false
	}
	for i := 0; i < 60; i++ {
		if p := g.lvl.RandomRoam(g.rng); g.lvl.Outdoor(p) {
			return p, true
		}
	}
	return Vec{}, false
}
