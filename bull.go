package main

import "math"

// ---------------------------------------------------------------------------
// The bull: a black pixel-art bull (blue-grey volume tones, dark grey muzzle, big white eyes,
// cream horns curling up, long tail with a tuft, dark hooves). Side view facing right with
// idle / walk / graze / paw / charge / impact frames, plus a front and a back view for when
// it walks up or down the screen (animalFreeView).
// ---------------------------------------------------------------------------

// extra bull frames (after fPeck1)
const (
	fBullIdle0 = 20 + iota // idle cycle: breathing, tail swing, ear flick
	fBullIdle1
	fBullIdle2
	fBullIdle3
	fBullRun0 // charge gallop, head down, horns forward
	fBullRun1
	fBullRun2
	fBullRun3
	fBullHit  // impact: head thrown up tossing the guest
	fBullPaw  // second pawing frame
	fBullLast = fBullPaw
)

// bullCoats: the bull is always the black one of the reference design (the coat only picks a tiny tint).
var bullCoats = []coatCols{
	{hx(0x24252d), hx(0x121217), hx(0x3a4054), hx(0xe6cf9e)},
	{hx(0x26252e), hx(0x131217), hx(0x3c3f52), hx(0xe6cf9e)},
	{hx(0x232630), hx(0x111318), hx(0x38405a), hx(0xe6cf9e)},
}

var (
	bullMuzzle  = hx(0x4c4d56)
	bullMuzzleD = hx(0x34353d)
	bullHoof    = hx(0x0c0c10)
	bullHornHi  = hx(0xf7e8c4)
	bullHornLo  = hx(0xb8996a)
)

// bullHorn paints a horn from (x,y) following a curve that bends forward; rot turns it (radians,
// clockwise on screen: 0 = pointing up, ~1 = pointing forward) and dir = -1 mirrors it.
func bullHorn(b *pixBuf, x, y, rot, dir float64, horn c4) {
	pts := [][2]float64{{0, 0}, {-1, -2.5}, {-1, -5}, {0.5, -7}, {3, -8}}
	cs, sn := math.Cos(rot), math.Sin(rot)
	var px, py [5]float64
	for i, p := range pts {
		lx, ly := p[0]*dir*1.45, p[1]*1.45
		px[i] = x + lx*cs - ly*sn
		py[i] = y + lx*sn + ly*cs
	}
	// stamp discs along the curve: thick cream base, thin light tip
	for i := 0; i < 4; i++ {
		for k := 0; k < 6; k++ {
			t := float64(k) / 6
			x, y := px[i]+(px[i+1]-px[i])*t, py[i]+(py[i+1]-py[i])*t
			r := 2.5 - (float64(i)+t)*0.55
			col := horn
			if i == 0 {
				col = bullHornLo
			} else if i == 3 {
				col = bullHornHi
			}
			b.ellipse(x, y, r, r, col)
			if i == 1 || i == 2 { // light on the outer edge
				b.set(int(math.Floor(x-r*0.6)), int(math.Floor(y)), bullHornHi)
			}
		}
	}
	b.set(int(math.Round(px[4])), int(math.Round(py[4])), bullHornHi)
}

// bullTailPt: point t (0..1) of the tail curve.
func bullTailPt(p0, p1, p2 [2]float64, t float64) (float64, float64) {
	u := 1 - t
	return u*u*p0[0] + 2*u*t*p1[0] + t*t*p2[0], u*u*p0[1] + 2*u*t*p1[1] + t*t*p2[1]
}

func genBullFrame(frame, coat int) *pixBuf {
	b := newPix(60, 40)
	b.ox, b.oy = 2, 2
	cc := bullCoats[coat%len(bullCoats)]
	body, dark, light, horn := cc.body, cc.dark, cc.light, cc.mane
	const ground = 36.0

	if frame == fStand {
		frame = fBullIdle0
	}
	if frame == fKick {
		frame = fBullRun0
	}
	idle := frame >= fBullIdle0 && frame <= fBullIdle3
	run := frame >= fBullRun0 && frame <= fBullRun3
	walk := frame >= fWalk0 && frame <= fWalk3
	hit := frame == fBullHit
	paw := frame == fWarn || frame == fBullPaw
	graze := frame == fGraze

	// per-frame animation values
	breath, tailSw, earUp := 0.0, 0.0, false
	if idle {
		i := frame - fBullIdle0
		breath = []float64{0, 0.5, 1, 0.5}[i]
		tailSw = []float64{0, 2, 4, 2}[i]
		earUp = i == 3
	}
	bob := 0.0
	if walk {
		bob = float64((frame-fWalk0)%2) * 1
		tailSw = []float64{0, 2, 4, 2}[frame-fWalk0]
		earUp = frame == fWalk2
	}
	ph := 0.0
	if run {
		ph = float64(frame-fBullRun0) / 4 * 2 * math.Pi
		bob = math.Round(math.Sin(ph*2)*1.5 + 1.5)
	}
	by := 18.0 - bob - breath*0.5 // body centre
	if run {
		by = 19 - bob
	}
	if hit {
		by = 16
	}

	// tail (behind everything)
	p0 := [2]float64{9, by - 5}
	p1 := [2]float64{1 + tailSw*0.3, by - 7}
	p2 := [2]float64{2 + tailSw*0.9, by + 8}
	if run {
		p1, p2 = [2]float64{0, by - 12}, [2]float64{-1 - math.Sin(ph)*1.5, by - 9}
	} else if hit {
		p1, p2 = [2]float64{1, by - 10}, [2]float64{-1, by - 5}
	}
	prevX, prevY := bullTailPt(p0, p1, p2, 0)
	for k := 1; k <= 10; k++ {
		x, y := bullTailPt(p0, p1, p2, float64(k)/10)
		thickLine(b, prevX, prevY, x, y, 2, shade(body, 0.85))
		prevX, prevY = x, y
	}
	b.ellipse(prevX, prevY+2, 2.2, 3.4, dark)
	b.set(int(prevX), int(prevY)+1, light)

	// legs
	leg := func(x float64, i int, col c4) {
		fx, fy := x, ground
		switch {
		case run:
			front := i < 2
			s := math.Sin(ph + map[bool]float64{true: 0, false: 2.1}[front] + float64(i&1)*0.5)
			fx, fy = x+s*7, ground-math.Max(0, math.Cos(ph+map[bool]float64{true: 0, false: 2.1}[front]+float64(i&1)*0.5))*4
		case hit:
			if i < 2 {
				fx, fy = x+4, ground-7+float64(i)*2 // front legs up in the air
			} else {
				fx, fy = x-2, ground
			}
		case paw && i == 1:
			if frame == fWarn {
				fx, fy = x+5, ground-8
			} else {
				fx, fy = x+3, ground-3
			}
		default:
			dx, lift := legSwing(frame, i)
			fx, fy = x+dx*4.5, ground-lift*3.5
		}
		hipY := by + 4
		mx, my, ex, ey := ik2(x, hipY, fx, fy, 7, 0.5, 0.2)
		thickLine(b, x, hipY, mx, my, 6, col)
		thickLine(b, mx, my, ex, ey, 5, shade(col, 0.93))
		b.rect(int(ex)-2, int(ey)-1, 5, 3, bullHoof)
		b.set(int(ex)-1, int(ey)-1, shade(bullHoof, 1.5))
	}
	farC, nearC := shade(body, 0.72), shade(body, 0.92)
	leg(39, 0, farC)
	leg(13, 2, farC)

	// body
	b.ellipse(24, by, 17, 9.5, body)
	b.ellipse(14, by+1, 8.5, 8.5, shade(body, 0.95)) // haunch
	b.ellipse(35, by-2.5, 10, 9, shade(body, 1.04))  // shoulders / hump
	b.ellipse(26, by+8, 12, 2.4, shade(body, 0.62))  // belly
	// volume: blue-grey light on the back, shoulder and haunch
	b.ellipse(27, by-7, 12, 2.2, light)
	b.ellipse(36, by-5, 5, 3, light)
	b.ellipse(14, by-3, 4, 3, shade(light, 0.9))
	b.ellipse(24, by+1, 5, 2, shade(body, 1.12))
	// muscle creases
	for k := 0; k < 4; k++ {
		b.set(31+k, by2i(by)+1+k/2, dark)
		b.set(10+k, by2i(by)+2-k/3, dark)
	}
	b.set(22, by2i(by)-2, light)
	b.set(25, by2i(by)-4, light)
	leg(34, 1, nearC)
	leg(18, 3, nearC)

	// head
	hx0, hy := 46.0, by-1
	hrot := 0.0 // horn rotation
	switch {
	case graze:
		hx0, hy, hrot = 47, 29, 0.9
	case paw:
		hx0, hy, hrot = 47, by+3, 0.5
	case run:
		hx0, hy, hrot = 48, by+8, 1.1
	case hit:
		hx0, hy, hrot = 47, by-9, -0.3
	case walk:
		hx0, hy = 46, by
	}
	if idle {
		hy += breath * 0.5
	}
	// far horn and ear first
	bullHorn(b, hx0-5, hy-5, hrot, 1, shade(horn, 0.85))
	thickLine(b, 35, by-2, hx0-2, hy, 10, shade(body, 1.0)) // neck
	b.ellipse(hx0-4, hy-3, 4, 3, shade(body, 1.02))         // poll (fur tuft)
	b.ellipse(hx0, hy, 7.2, 6.4, body)
	b.ellipse(hx0-2, hy-3.5, 4, 1.6, light)
	// muzzle
	b.ellipse(hx0+6, hy+2.5, 4.4, 3.6, bullMuzzle)
	b.ellipse(hx0+7, hy+4.2, 3.4, 1.7, bullMuzzleD)
	b.set(int(hx0)+9, int(hy)+2, bullHoof)
	b.set(int(hx0)+9, int(hy)+3, bullHoof)
	b.set(int(hx0)+4, int(hy)+4, bullMuzzleD)
	b.set(int(hx0)+5, int(hy)+1, shade(bullMuzzle, 1.25))
	// ear
	eyAdd := 0.0
	if earUp {
		eyAdd = -2
	}
	b.ellipse(hx0-6, hy-1+eyAdd, 4.2, 2.2, shade(body, 0.8))
	b.set(int(hx0)-5, int(hy)-1+int(eyAdd), shade(light, 0.9))
	// near horn
	bullHorn(b, hx0-1, hy-6, hrot, 1, horn)
	// eye: big and white with a pupil
	ex, ey := int(hx0)+1, int(hy)-3
	b.rect(ex, ey, 4, 4, hx(0xf4f4f0))
	b.rect(ex+2, ey+1, 2, 3, hx(0x0c0c10))
	b.set(ex+2, ey+1, hx(0xf4f4f0))
	if run || paw || hit {
		// angry brow
		b.line(ex-1, ey-2, ex+3, ey-1, dark)
	}
	if hit {
		sp := hx(0xfff2b0)
		b.set(int(hx0)+10, int(hy)-8, sp)
		b.set(int(hx0)+12, int(hy)-5, sp)
		b.set(int(hx0)+9, int(hy)-12, sp)
		b.line(int(hx0)+8, int(hy)-10, int(hx0)+11, int(hy)-12, sp)
	}
	return b
}

func by2i(v float64) int { return int(math.Round(v)) }

// ---- front and back views (it walks up or down the screen) ---------------------------------

var bullFreeCache = map[int]PropImg{}

// bullFreeView: the bull seen head on (front) or from behind, step 0/1 alternates the legs.
// Anchored at the middle of the hooves.
func bullFreeView(front bool, step int) PropImg {
	key := step & 1
	if front {
		key |= 2
	}
	if pi, ok := bullFreeCache[key]; ok {
		return pi
	}
	pi := PropImg{img: bullFreeBuf(front, step).img(), ox: 17, oy: 41}
	bullFreeCache[key] = pi
	return pi
}

// bullFreeBuf paints the front / back picture (34x44, hooves centred on x=17 and ending at y=41).
func bullFreeBuf(front bool, step int) *pixBuf {
	cc := bullCoats[0]
	body, dark, light, horn := cc.body, cc.dark, cc.light, cc.mane
	b := newPix(34, 44)
	b.ox, b.oy = 2, 2
	const cx, ground = 15.0, 38.0
	leg := func(x float64, lift float64, col c4) {
		thickLine(b, x, 24, x, ground-lift, 5, col)
		b.rect(int(x)-2, int(ground-lift)-1, 5, 3, bullHoof)
	}
	l0, l1 := 0.0, 0.0
	if step&1 == 0 {
		l0 = 2
	} else {
		l1 = 2
	}
	if front {
		leg(cx-5, l0, shade(body, 0.85))
		leg(cx+5, l1, shade(body, 0.85))
		b.ellipse(cx, 24, 10, 10, body)               // chest
		b.ellipse(cx, 17, 11.5, 7, shade(body, 1.04)) // shoulders
		b.ellipse(cx, 13, 8, 2.5, light)              // hump light
		b.ellipse(cx-6, 22, 3, 5, shade(light, 0.85)) // shoulder volume
		b.ellipse(cx+6, 22, 3, 5, shade(body, 0.8))
		b.ellipse(cx, 31, 5, 2, shade(body, 0.7)) // belly under the chest
		// horns and ears
		bullHorn(b, cx-6, 14, -0.45, -1, horn)
		bullHorn(b, cx+6, 14, 0.45, 1, horn)
		b.ellipse(cx-8, 18, 4, 2, shade(body, 0.8))
		b.ellipse(cx+8, 18, 4, 2, shade(body, 0.8))
		// head
		b.ellipse(cx, 13, 5.5, 2.4, dark) // forelock
		b.ellipse(cx, 20, 6, 6.5, body)
		b.ellipse(cx, 15, 4, 1.5, light)
		b.ellipse(cx, 24, 4.4, 3.4, bullMuzzle) // muzzle
		b.ellipse(cx, 26, 3.4, 1.4, bullMuzzleD)
		b.set(int(cx)-2, 24, bullHoof)
		b.set(int(cx)+2, 24, bullHoof)
		for _, ex := range []int{int(cx) - 5, int(cx) + 3} {
			b.rect(ex, 17, 3, 3, hx(0xf4f4f0))
			b.rect(ex+1, 18, 2, 2, hx(0x0c0c10))
			b.set(ex+1, 18, hx(0xf4f4f0))
		}
	} else {
		leg(cx-5, l0, shade(body, 0.85))
		leg(cx+5, l1, shade(body, 0.85))
		bullHorn(b, cx-6, 12, -0.45, -1, horn)
		bullHorn(b, cx+6, 12, 0.45, 1, horn)
		b.ellipse(cx-8, 15, 4, 2, shade(body, 0.8))
		b.ellipse(cx+8, 15, 4, 2, shade(body, 0.8))
		b.ellipse(cx, 14, 6, 5, body) // nape
		b.ellipse(cx, 12, 4, 1.6, light)
		b.ellipse(cx, 25, 11.5, 12, body)             // rump
		b.ellipse(cx-5, 23, 4, 5, shade(light, 0.85)) // haunch light
		b.ellipse(cx+5, 25, 4, 5, shade(body, 0.85))
		b.ellipse(cx, 21, 7, 2, shade(body, 1.1))
		// tail down the middle with its tuft
		thickLine(b, cx, 20, cx, 32, 2, shade(body, 0.7))
		b.ellipse(cx, 34, 2, 3.4, dark)
		b.set(int(cx), 33, light)
	}
	return b.outlinedSel(0.4)
}

// animalHasFreeView: animals that show their back or chest when they walk up or down the screen.
func animalHasFreeView(k int) bool { return k == animHorse || k == animBull || k == animCapibara }

// animalFreeView: that front / back picture for the animal (a.vert > 0 walking down = chest).
func animalFreeView(a *Animal) PropImg {
	if a.kind == animBull {
		return bullFreeView(a.vert > 0, int(a.anim*5))
	}
	if a.kind == animCapibara {
		return capibaraFreeView(a) // animals8.go
	}
	return horseFreeView(a.coat, a.vert > 0)
}

// bullFrame: sprite frame of the bull for its current state.
func (g *Game) bullFrame(a *Animal) int {
	switch {
	case a.sprayT > 0: // sprayT is the bull's impact timer (the elephant's spray is another animal)
		return fBullHit
	case a.state == aGraze:
		return fGraze
	case a.state == aWarn:
		return fWarn + int(a.anim*5)%2*(fBullPaw-fWarn)
	case a.state == aCharge:
		return fBullRun0 + int(a.anim*12)%4
	case a.state == aWalk:
		return fWalk0 + int(a.anim*6)%4
	}
	return fBullIdle0 + int(a.anim*2)%4
}
