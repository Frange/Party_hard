package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The intro for two players.
//
// Same night, same room and same music, but two beds and two very different sleepers (see the
// "characters" in twoscenes.go): the blue one is the patient one (lies still on his back, puts paper
// earplugs in, counts sheep in a thought cloud, catches a falling glass without opening his eyes, gets up
// slowly) and the red one the nervous one (pillow over the head, sits up fuming, tosses and turns,
// bangs on the wall, throws a slipper at the neighbours and a pillow at the blue one's dream cloud, jumps out
// of bed at once, plays rock paper scissors too early and loses, runs to the door and cannot open it).
// The timeline is the one-player intro's (introLen, the music, the door...) so the music and the beat
// clock stay in step; only the picture and the jokes change.
// ---------------------------------------------------------------------------

const (
	i2GlassBFall = 8.2  // the red one's glass falls
	i2GlassBreak = 8.65 // and breaks
	i2Sit1       = 2.6  // the red one sits up to fume (first time)
	i2Sit1End    = 4.6
	i2Sit2       = 10.0 // and again
	i2Sit2End    = 13.0
	i2Plug       = 11.6 // the blue one puts his paper earplugs in
	i2WallA      = 13.0 // the red one bangs on the wall on the beat
	i2WallB      = 16.8
	i2Slip0      = 16.9 // he throws a slipper at the window
	i2SlipHit    = 17.5 // it hits the window
	i2SlipBack   = 18.4 // and bounces onto the blue one
	i2Sit3       = 17.6 // sits up once more...
	i2Pillow0    = 18.2 // ...and throws his pillow at the blue one's dream cloud
	i2RPS0       = 20.7 // rock paper scissors
	i2RPS1       = 23.3
	i2Run        = 24.5 // the red one runs to the door; the blue one strolls
	i2AtDoor     = 27.2 // and he is there long before it opens
	i2SlowStart  = 0.8  // the blue one needs this long more to get up
)

// the glasses on the table between the beds
const (
	i2GlassAx0, i2GlassBx0 = 463.0, 494.0
	i2GlassY               = 316.0
	i2CatchX, i2CatchY     = 426.0, 372.0 // where the blue one catches his
)

// introTwoBed: what each of them does in bed at time t (the room draws it, see drawTwoBeds).
func introTwoBed(i int, t float64) bedPose {
	var p bedPose
	up := func(a, b float64) float64 { return introEase((t-a)/0.5) * (1 - introEase((t-b)/0.5)) }
	if i == 0 { // the patient one: still on his back, earplugs, then he sits up slowly
		p.turn = 0
		p.earplug = t >= i2Plug && t < inStandUp+i2SlowStart
		p.away = t >= inStandUp+i2SlowStart
		if t >= inStandUp && t < inStandUp+i2SlowStart {
			p.sit = introEase((t - inStandUp) / (i2SlowStart * 0.9))
		}
		return p
	}
	// the nervous one
	p.turn = 3.2
	p.pillow = math.Max(up(0.6, 2.4), up(4.8, 6.1))
	s := math.Max(up(i2Sit1, i2Sit1End), math.Max(up(i2Sit2, i2Sit2End), up(i2Sit3, inStandUp)))
	p.sit = s
	p.puff = s > 0.6
	p.fist = t >= i2WallA && t < i2WallB
	p.away = t >= inStandUp
	return p
}

// introTwoGlassA: the blue one's glass: where it is, if it is falling, caught and how far his arm reaches.
func introTwoGlassA(t float64) (x, y, rot float64, arm float64) {
	k := introPump(t)
	x, y = i2GlassAx0, i2GlassY
	switch {
	case t < inGlassSlide:
		y -= math.Round(k * 1.6)
	case t < inGlassFall:
		x -= 11 * (t - inGlassSlide) / (inGlassFall - inGlassSlide)
		y -= math.Round(k * 1.6)
	case t < inGlassLand: // the fall...
		u := (t - inGlassFall) / (inGlassLand - inGlassFall)
		x = i2GlassAx0 - 11 - 22*u
		y = i2GlassY + (i2CatchY-i2GlassY)*u*u
		rot = u * 2.2
		arm = introEase(u * 1.6)
	case t < 8.0: // ...caught without a flinch
		x, y = i2CatchX, i2CatchY
		arm = 1
	case t < 8.6: // he puts it back on the table
		u := introEase((t - 8.0) / 0.6)
		x = i2CatchX + (i2GlassAx0-11-i2CatchX)*u
		y = i2CatchY + (i2GlassY-i2CatchY)*u
		arm = 1 - u
	default:
		x = i2GlassAx0 - 11
		y -= math.Round(k * 1.6)
	}
	return
}

// introTwoGlassB: the red one's glass: it creeps to the right edge, falls and breaks at i2GlassBreak.
func introTwoGlassB(t float64) (x, y, rot float64, broken bool) {
	k := introPump(t)
	x, y = i2GlassBx0, i2GlassY
	switch {
	case t < 4.2:
		y -= math.Round(k * 1.6)
	case t < i2GlassBFall:
		x += 12 * (t - 4.2) / (i2GlassBFall - 4.2)
		y -= math.Round(k * 1.6)
	case t < i2GlassBreak:
		u := (t - i2GlassBFall) / (i2GlassBreak - i2GlassBFall)
		x = i2GlassBx0 + 12 + 40*u
		y = i2GlassY + 102*u*u
		rot = -u * 3.0
	default:
		broken = true
	}
	return
}

// introTwoActor: where each hero stands once he is out of bed and what he is doing.
type introTwoActor struct {
	x, y, s float64
	alpha   float64
	hop     float64
	anim    int
	dir     Dir
	ph      float64
	visible bool
	lean    float64
	knife   bool
	bump    float64 // a hop from being shoved
	dizzy   bool
}

func introTwoActorAt(i int, t float64) introTwoActor {
	a := introTwoActor{alpha: 1, s: 4, anim: AnimIdle, dir: DirDown}
	t0 := inStandUp // when he leaves the bed
	startX, midX, endX, startY := 250.0, 410.0, 150.0, 432.0
	if i == 0 { // the patient one is slower in everything
		t0 += i2SlowStart
	} else {
		startX, midX, endX, startY = 700, 540, 95, 444
	}
	if t < t0 {
		return a // in bed: the room draws him
	}
	a.visible = true
	if t < t0+0.5 && i == 1 { // the red one jumps out
		a.hop = math.Sin((t-t0)/0.5*math.Pi) * 26
	}
	dur := 1.0
	if i == 0 {
		dur = 1.6
	}
	u := introEase((t - t0) / dur)
	a.x = startX + (midX-startX)*u
	a.y = startY
	if t >= t0 && t < t0+dur {
		a.anim, a.ph = AnimWalk, t*6
		if i == 1 {
			a.anim, a.ph = AnimRun, t*9
			a.dir = DirLeft
		} else {
			a.dir = DirRight
		}
	}
	if t >= i2RPS0 && t < i2RPS1+0.5 { // face to face
		a.anim = AnimIdle
		if i == 0 {
			a.dir = DirRight
		} else {
			a.dir = DirLeft
		}
		if i == 1 && t > i2RPS0+1.8 { // the loser fumes: punches the air
			a.anim, a.ph = AnimPunch, t*8
		}
	}
	a.knife = t >= inKnife
	// the walk to the door: he runs, the blue one strolls
	if t >= i2Run {
		end := inWalk1
		if i == 1 {
			end = i2AtDoor
		}
		w := introEase((t - i2Run) / (end - i2Run))
		endY, endS := 326.0, 3.1
		if i == 1 {
			endY, endS = 318, 3.0
		}
		a.x = midX + (endX-midX)*w
		a.y = startY + (endY-startY)*w
		a.s = 4 + (endS-4)*w
		a.anim, a.ph = AnimWalk, t*5
		if i == 1 {
			a.anim, a.ph = AnimRun, t*9
		}
		a.dir = DirLeft
		if w > 0.6 {
			a.dir = DirUp
		}
		if w >= 1 {
			a.anim, a.ph = AnimIdle, 0
			if i == 1 && t < inWalk1+0.2 { // he tugs at the handle
				a.anim, a.ph = AnimPanic, t*10
			}
		}
	}
	if i == 1 && t >= i2Run && t < i2Run+0.6 { // shoved on his way
		a.bump = math.Sin((t-i2Run)/0.6*math.Pi) * 14
		a.dizzy = true
	}
	// through the door: the red one first (he lost), the blue one calmly after
	if i == 0 && t > inWalk1+0.9 {
		w := introEase((t - inWalk1 - 0.9) / 0.7)
		a.x = endX + (72-endX)*w
		a.y = 326 + (318-326)*w
		a.anim, a.ph = AnimWalk, t*5
		a.alpha = 1 - introEase((t-inWalk1-1.2)/0.8)
	}
	if i == 1 && t > inWalk1+0.4 {
		a.alpha = 1 - introEase((t-inWalk1-0.4)/0.9)
	}
	return a
}

// introTwoCam: the point of the room the camera follows (between the two of them).
func introTwoCam(t float64) (x, y float64) {
	a, b := introTwoActorAt(0, t), introTwoActorAt(1, t)
	switch {
	case !a.visible && !b.visible:
		return 480, 360
	case !a.visible:
		return b.x, b.y
	case !b.visible:
		return a.x, a.y
	}
	return (a.x + b.x) / 2, (a.y + b.y) / 2
}

// ---- drawing -----------------------------------------------------------------------------------

// introRoomTwo adds everything that happens in the room with two beds.
func (g *Game) introRoomTwo(c *ebiten.Image, in *introState) {
	t := in.t
	g.introDoor(c, t)
	g.introGlassTwo(c, t)
	g.introGagsTwo(c, in)
	g.introHeroesTwo(c, in)
	if t >= inSheepStart && t < inBurst+2.6 { // the blue one's dream goes over everything: door, furniture, heroes
		g.introCloudAt(c, t-inSheepStart, 252, 170)
	}
	if t >= inBurst && t < inBurst+0.35 {
		fillRect(c, 0, 0, screenW, screenH, rgba(0xffffff, uint8(230*(1-(t-inBurst)/0.35))))
	}
}

func (g *Game) introHeroesTwo(c *ebiten.Image, in *introState) {
	t := in.t
	for i := 0; i < 2; i++ {
		a := introTwoActorAt(i, t)
		if !a.visible || a.alpha <= 0 {
			continue
		}
		cs := in.heroes[i]
		if a.knife && in.heroesK[i] != nil {
			cs = in.heroesK[i]
		}
		tint := sceneTint(i, in.heroes[0], in.heroes[1])
		g.sceneChar(c, cs, a.anim, a.dir, a.ph, a.x, a.y, a.s, a.alpha, a.hop+a.bump, a.lean, tint)
		if a.dizzy {
			g.sceneStars(c, a.x, a.y-a.hop-a.bump-34*a.s, t, 1)
		}
	}
	if t >= inKnife && t < inKnife+0.5 { // two glints on the blades
		for i := 0; i < 2; i++ {
			a := introTwoActorAt(i, t)
			g.drawText(c, "*", a.x+8*a.s, a.y-26*a.s, rgba(0xffffff, uint8(255*(1-(t-inKnife)/0.5))), 2)
		}
	}
}

// the two glasses on the table between the beds
func (g *Game) introGlassTwo(c *ebiten.Image, t float64) {
	// the arm of the blue one that catches the glass in mid air: the sleeve of his own colour and a hand
	if _, y, _, arm := introTwoGlassA(t); arm > 0.02 {
		x1 := bed1X1 + 6
		x2 := x1 + (i2CatchX-x1)*arm
		fillRect(c, x1, 362, math.Min(x2-x1, 18), 9, rgb(0x3a6aa0))
		if x2-x1 > 18 {
			fillRect(c, x1+18, 363, x2-x1-18, 7, rgb(0xe0b088))
		}
		fillRect(c, x2-2, math.Min(y, 372)-1, 8, 10, rgb(0xe0b088))
	}
	ax, ay, ar, _ := introTwoGlassA(t)
	g.drawGlass(c, ax, ay, ar, 1)
	if t >= inGlassFall && t < inGlassLand+0.9 {
		g.sceneBang(c, "¡toma!", i2CatchX+14, 352, sceneWindow(t, inGlassLand, inGlassLand+0.9), 0x60e0a0, 1.2)
	}
	bx, by, br, broken := introTwoGlassB(t)
	if !broken {
		g.drawGlass(c, bx, by, br, 1)
		if k := introPump(t); k > 0.45 && t < i2GlassBFall { // ripples and a "tin"
			g.drawText(c, "tin", bx-6, by-20-math.Round(k*1.6), rgba(0xeaf6ff, uint8(255*k)), 0.9)
		}
		return
	}
	u := math.Min(1, (t-i2GlassBreak)/0.5)
	for i := 0; i < 11; i++ { // the bits hop away and stay on the floor
		ang := -math.Pi + jpHash(i, 7, 61)*math.Pi
		sp := 14 + jpHash(i, 9, 63)*26
		x := 556 + math.Cos(ang)*sp*u
		y := 416 - math.Abs(math.Sin(ang))*(10+jpHash(i, 4, 65)*12)*math.Sin(math.Pi*u) + jpHash(i, 6, 67)*5
		w := 2 + float64(i%3)
		col := rgba(0xbfe0f8, 210)
		if i%4 == 0 {
			col = rgba(0x5a9ae0, 220)
		}
		fillRect(c, math.Round(x), math.Round(y), w, 2, col)
	}
	if t < i2GlassBreak+0.8 {
		g.drawTextC(c, "¡CRASH!", 556, 366-(t-i2GlassBreak)*30, rgba(0xff6a8a, uint8(255*(1-(t-i2GlassBreak)/0.8))), 2)
	}
}

// the jokes of the room that are not about the beds themselves
func (g *Game) introGagsTwo(c *ebiten.Image, in *introState) {
	t := in.t
	// the red one's fist on the headboard, on every kick, and the blue one's paper earplugs
	if p := introTwoBed(1, t); p.fist {
		k := beatKick(t)
		x := bed2X1 + 26 + k*8
		fillRect(c, x, 338, 14, 11, rgb(0xe0b088))
		fillRect(c, x, 338, 14, 3, rgb(0xf0c8a0))
		if k > 0.35 {
			for s := 0; s < 3; s++ { // the lines of the hit
				d := float64(s) * 5
				fillRect(c, x+16+d*0.4, 333+d, 5, 1, rgba(0xffffff, 200))
			}
			g.sceneBang(c, "¡PUM!", bed2X1+70, 312, math.Min(1, (1-k)*2.5), 0xffe060, 1.8)
		}
	}
	if p := introTwoBed(0, t); p.earplug { // two little balls of paper in the ears
		drawEarplugs(c, 0)
		if t < i2Plug+1.2 { // the poke
			g.sceneBang(c, "pok", bedHead1X+30, twoBedCenterY-34, sceneWindow(t, i2Plug, i2Plug+1.2), 0xe8e8f4, 1.1)
		}
	}
	// the red one fumes while he sits up: steam over his head and a snort
	if p := introTwoBed(1, t); p.puff {
		hx, hy := bed2X0+60.0, 250.0
		drawSteam(c, t, hx, hy)
		if int(t*1.2)%2 == 0 {
			g.sceneBang(c, "¡PUF!", hx, hy-48, math.Mod(t*1.2, 1), 0xff8a6a, 1.5)
		}
	}
	if t >= i2Slip0 && t < i2SlipBack+0.4 { // the slipper: window, bounce, head
		var x, y float64
		switch {
		case t < i2SlipHit:
			u := (t - i2Slip0) / (i2SlipHit - i2Slip0)
			x = 700 + (520-700)*u
			y = 330 + (150-330)*u - math.Sin(math.Pi*u)*50
		case t < i2SlipBack:
			u := (t - i2SlipHit) / (i2SlipBack - i2SlipHit)
			x = 520 + (190-520)*u
			y = 150 + (332-150)*u*u - math.Sin(math.Pi*u)*70
		default:
			x, y = 190, 334
		}
		rot := (t - i2Slip0) * 14
		for k := 0; k < 8; k++ { // the slipper turns round: a little brown sole with a fluffy top
			dx, dy := math.Cos(rot)*float64(k-4)*1.6, math.Sin(rot)*float64(k-4)*1.6
			fillRect(c, math.Round(x+dx)-1, math.Round(y+dy)-1, 3, 3, rgb(0x8a5a3a))
		}
		fillRect(c, math.Round(x)-2, math.Round(y)-2, 4, 4, rgb(0xf4a0c0))
		if t >= i2SlipHit && t < i2SlipHit+0.7 {
			g.sceneBang(c, "¡CLONC!", 520, 138, (t-i2SlipHit)/0.7, 0xff6a8a, 2)
		}
		if t >= i2SlipBack && t < i2SlipBack+0.4 {
			g.sceneBang(c, "¡TOC!", 190, 320, (t-i2SlipBack)/0.4, 0xffe060, 2)
		}
	}
	// the pillow of the red one flies to the blue one's cloud (and pops it), then falls on the blue one
	if t >= i2Pillow0 && t < inBurst+0.8 {
		var x, y float64
		if t < inBurst {
			u := (t - i2Pillow0) / (inBurst - i2Pillow0)
			x = 640 + (252-640)*u
			y = 330 + (184-330)*u - math.Sin(math.Pi*u)*40
		} else {
			u := (t - inBurst) / 0.8
			x = 252 + (bedHead1X-252)*u
			y = 184 + (twoBedCenterY-60-184)*u*u
		}
		fillRect(c, math.Round(x)-14, math.Round(y)-9, 28, 18, rgb(0xd8dee8))
		fillRect(c, math.Round(x)-14, math.Round(y)-9, 28, 3, rgb(0xf0f4fa))
		fillRect(c, math.Round(x)-14, math.Round(y)+6, 28, 3, rgb(0xa8b0c2))
	}
	if t >= i2SlipBack && t < i2SlipBack+2 { // stars over the blue one's head after the slipper...
		g.sceneStars(c, bedHead1X+8, twoBedCenterY-48, t, 1-sceneWindow(t, i2SlipBack+1.4, i2SlipBack+2))
	}
	if t >= inBurst+0.8 && t < inBurst+2.0 { // ...and after the pillow
		g.sceneStars(c, bedHead1X+8, twoBedCenterY-48, t, 1-sceneWindow(t, inBurst+1.4, inBurst+2.0))
	}
}

// the blue one's dream: a thought cloud with sheep; the fourth wears sunglasses and a party hat
func (g *Game) introCloudAt(c *ebiten.Image, u, cx, cy float64) {
	if u >= inBurst-inSheepStart { // popped: confetti
		g.introConfetti(c, cx, cy, u-(inBurst-inSheepStart))
		return
	}
	a := uint8(255 * math.Min(1, u/1.2))
	al := func(col color.RGBA) color.RGBA { col.A = a; return col }
	pulse := 0.0
	if u > 7.2 {
		pulse = introPump(u+inSheepStart) * 5
	}
	white := al(rgb(0xf4f6ff))
	filledEllipse(c, cx, cy, 130+pulse, 70+pulse, white)
	filledEllipse(c, cx-70, cy-42, 54+pulse, 34, white)
	filledEllipse(c, cx+60, cy-44, 58+pulse, 34, white)
	filledEllipse(c, cx-74, cy+50, 46, 28, white)
	for i, r := range []float64{9, 6, 4} {
		filledEllipse(c, cx+90+float64(i)*26, cy+80+float64(i)*16, r, r, white)
	}
	sky := al(rgb(0xbfe0f8))
	if u > 7.2 {
		sky = al(rgb(partyColors[int(u*8)%len(partyColors)]))
	}
	fillRect(c, cx-104, cy-44, 208, 92, sky)
	fillRect(c, cx-104, cy+20, 208, 28, al(rgb(0x6ab46a)))
	for x := cx - 36; x <= cx+36; x += 18 {
		fillRect(c, x, cy-4, 3, 30, al(rgb(0x8a6a44)))
	}
	fillRect(c, cx-40, cy+4, 80, 3, al(rgb(0x8a6a44)))
	fillRect(c, cx-40, cy+14, 80, 3, al(rgb(0x8a6a44)))
	const per = 2.0
	start := 1.5
	n := -1
	for k := 0; k < 4; k++ {
		s := (u - start - float64(k)*per) / per
		if s < 0 || s > 1 {
			continue
		}
		n = k
		x := cx - 110 + s*220
		jump := 0.0
		if s > 0.34 && s < 0.66 {
			jump = math.Sin((s-0.34)/0.32*math.Pi) * 34
		}
		g.drawSheep(c, x, cy+22-jump, s, a)
		if k == 3 { // sunglasses and a party hat
			fillRect(c, x+14, cy+22-jump-5, 9, 3, al(rgb(0x0a0a10)))
			fillRect(c, x+15, cy+22-jump-5, 2, 1, al(rgb(0x8ab8ff)))
			for r := 0; r < 7; r++ {
				fillRect(c, x+12+float64(r)*0.5, cy+22-jump-8-float64(7-r)*1.6, float64(7-r)+1, 2, al(rgb(partyColors[r%len(partyColors)])))
			}
		}
	}
	txt := ""
	switch {
	case n < 0 && u < start:
	case n == 3 || u > start+3*per:
		txt = "4 ovejitas... ¿¿gafas??"
	case n == 2:
		txt = "3 ovejitas..."
	case n == 1:
		txt = "2 ovejitas..."
	default:
		txt = "1 ovejita..."
	}
	if txt != "" {
		g.drawTextC(c, txt, cx, cy-62, al(rgb(0x3a3a5a)), 1.4)
	}
}

// ---- captions and speech -----------------------------------------------------------------------

// introSay: a comic balloon over a point of the room (room coordinates), through the camera of the frame.
func (g *Game) introSay(dst *ebiten.Image, in *introState, x, y float64, s string) {
	z := in.cam
	g.speechBubble(dst, z.ox+x*z.z, z.oy+y*z.z, s)
}

func (g *Game) introOverlayTwo(dst *ebiten.Image, in *introState) {
	t := in.t
	var line string
	switch {
	case t < 3.5:
		line = "3:47 de la madrugada. Dos camas, dos formas de no dormir."
	case t < inGlassFall:
		line = "El bajo del vecino hace bailar los vasos de la mesilla."
	case t < inGlassLand+0.9:
		line = "¡Un vaso al vacío! El azul ni abre los ojos."
	case t < i2GlassBreak+0.2:
		line = "Lo caza al vuelo. El rojo no tiene tanta suerte."
	case t < inSheepStart:
		line = "El rojo ya está al borde del ataque de nervios."
	case t < 13:
		line = "El azul cuenta ovejas. El rojo cuenta los segundos."
	case t < i2WallB:
		line = "Tapones de papel contra puñetazos a la pared."
	case t < inBurst:
		line = "Una zapatilla, una almohada... y un sueño reventado."
	case t < 20.5:
		line = "¡PAF! Hasta los sueños se han ido de fiesta."
	case t < i2RPS1:
		line = "Piedra, papel o tijera: el que pierda abre la puerta."
	case t < inKnife+1:
		line = "Se acabó. Hay una sola forma de que esta ciudad duerma."
	case t < inFadeOut-1:
		line = "Cuchillo en mano, a poner fin a las fiestas. Juntos."
	default:
		line = "Una fiesta menos. Y otra. Y otra..."
	}
	g.endCaption(dst, line)
	head0 := func(s string) { g.introSay(dst, in, bedHead1X, twoBedCenterY-64, s) }
	head1 := func(s string) { g.introSay(dst, in, bedHead2X, twoBedCenterY-64, s) }
	sit1 := func(s string) { g.introSay(dst, in, bed2X0+60, 196, s) }
	switch { // one balloon at a time, each one says what he would say
	case sceneIn(t, 1.0, 2.4):
		head1("¡¡Mmmpf!!")
	case sceneIn(t, 2.8, 4.4):
		sit1("¡Así no hay quien duerma!")
	case sceneIn(t, inGlassLand+0.1, inGlassLand+1.4):
		head0("Con calma.")
	case sceneIn(t, i2GlassBreak+0.3, i2GlassBreak+1.5):
		head1("¡¡MI VASO!!")
	case sceneIn(t, i2Sit2+0.2, i2Sit2+1.6):
		sit1("Resoplo, luego existo")
	case sceneIn(t, i2Plug+0.9, i2Plug+2.4):
		head0("Respira. Ommm.")
	case sceneIn(t, i2WallA+0.3, i2WallA+1.8):
		head1("¡¡Callaos YA!!")
	case sceneIn(t, i2WallA+2.0, i2WallA+3.4):
		head0("(no oigo nada)")
	case sceneIn(t, i2Slip0, i2Slip0+0.9):
		head1("¡Toma pantufla!")
	case sceneIn(t, i2SlipBack+0.1, i2SlipBack+1.2):
		head0("...no pasa nada.")
	case sceneIn(t, i2Pillow0, inBurst):
		sit1("¡¡Y TOMA ALMOHADA!!")
	case sceneIn(t, inBurst+0.5, inBurst+1.7):
		head0("Iba por la cuarta...")
	}
	// out of bed: the red one first, as usual
	if t >= inStandUp && t < i2RPS0 {
		a, b := introTwoActorAt(0, t), introTwoActorAt(1, t)
		switch {
		case t < inStandUp+1.4 && b.visible:
			g.introSay(dst, in, b.x, b.y-44*b.s, "¡¡YA ESTÁ BIEN!!")
		case a.visible && t < i2RPS0:
			g.introSay(dst, in, a.x, a.y-44*a.s, "Buenos días... supongo.")
		}
	}
	if sceneIn(t, i2RPS0, i2RPS1) { // the red one cannot wait for the count
		a, b := introTwoActorAt(0, t), introTwoActorAt(1, t)
		switch {
		case t < i2RPS0+0.7:
			g.introSay(dst, in, a.x, a.y-46*a.s, "Piedra... papel...")
		case t < i2RPS0+1.5:
			g.introSay(dst, in, b.x, b.y-44*b.s, "¡¡TIJERA!!")
		case t < i2RPS0+2.2:
			g.introSay(dst, in, a.x, a.y-44*a.s, "Papel.")
		case t < i2RPS0+2.7:
			g.introSay(dst, in, b.x, b.y-44*b.s, "¡Mejor de tres!")
		default:
			g.introSay(dst, in, a.x, a.y-44*a.s, "No.")
		}
	}
	if sceneIn(t, i2Run+0.1, i2Run+1.3) {
		b := introTwoActorAt(1, t)
		g.introSay(dst, in, b.x, b.y-44*b.s, "¡¡Abro yo, abro yo!!")
	}
	if sceneIn(t, i2AtDoor+0.3, inWalk1-0.2) {
		g.introSay(dst, in, 90, 180, "¡No abre!")
	}
	if sceneIn(t, inWalk1-0.1, inWalk1+0.9) {
		a := introTwoActorAt(0, t)
		g.introSay(dst, in, a.x, a.y-44*a.s, "Tira, no empujes.")
	}
}

var _ = ebiten.NewImage

// drawEarplugs: two little balls of paper in the ears of the first sleeper (dx slides the room).
func drawEarplugs(c *ebiten.Image, dx float64) {
	for _, dy := range []float64{-15, 15} {
		fillRect(c, bedHead1X+dx+2, twoBedCenterY+dy-2, 5, 5, rgb(0xf4f0e0))
		fillRect(c, bedHead1X+dx+2, twoBedCenterY+dy+2, 5, 1, rgb(0xb8b4a0))
	}
}

// drawSteam: the steam of the red one fuming over his head at (hx, hy).
func drawSteam(c *ebiten.Image, t, hx, hy float64) {
	for i := 0; i < 5; i++ {
		age := math.Mod(t*1.4+float64(i)/5, 1)
		filledEllipse(c, hx+math.Sin(age*5+float64(i))*8, hy-age*40, 4+age*8, 4+age*7, rgba(0xdce0ee, uint8(170*(1-age))))
	}
}
