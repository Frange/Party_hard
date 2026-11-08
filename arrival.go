package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The arrival on the island: after the letter blows up, a yellow rented helicopter flies over the
// sea to the coast of Isla Nublar and lands on the helipad. The players are seen in the cockpit from the
// start; the door slides open, they get out (the blue one calm, the red one in a hurry and dizzy),
// they step away and the helicopter blows up (a show: nobody gets hurt and nothing is left behind but
// a scorch mark that fades). Then the party of the island begins.
//
// It plays only in the story (letter -> arrival -> island). The island played from the list of parties or
// from DEBUG starts directly (see finishInvite); ESCENAS and DEBUG can watch it again (-jparrival too).
// ---------------------------------------------------------------------------

// the timeline, in seconds
const (
	arrFlyEnd  = 2.2  // the camera has reached the helipad; the landing starts
	arrTouch   = 3.4  // the skids touch the pad
	arrDoor    = 3.7  // the door starts to slide open
	arrDoorLen = 0.4  // ...and takes this long
	arrExit1   = 4.1  // the first player steps out
	arrExit2   = 4.3  // the second one
	arrBoom    = 6.1  // the helicopter blows up
	arrGagEnd  = 8.7  // the gag ends and the picture fades
	arrTotal   = 9.5  // the end of the scene
	arrCamEnd  = 620. // how far the camera travels down the world
	arrWorldH  = 1160
	arrK       = 3.0 // scale of the helicopter (whole number: crisp pixels)
)

// where things are on the screen once it has landed
const (
	arrCX      = 480.0                // the helicopter's anchor (the centre of the fuselage)
	arrGroundY = 332.0                // the skids rest here (the centre of the helipad is a little lower)
	arrCY      = arrGroundY - 18*arrK // the skids hang 18 sprite pixels under the anchor
	arrPadWX   = 480.0                // the helipad in the world picture
	arrPadWY   = 960.0
)

type arrivalState struct {
	t, skipArm, rot float64
	touched, boomed bool
	parts           []carPart
	world, canvas   *ebiten.Image
}

// the rotor speed: full until the touch-down, slowing after it, stopped when it blows
func arrRPM(t float64) float64 {
	switch {
	case t >= arrBoom:
		return 0
	case t < arrTouch:
		return 1
	}
	return 1 - 0.85*arrSmooth((t-arrTouch)/1.8)
}

func arrSmooth(u float64) float64 { u = clampF(u, 0, 1); return u * u * (3 - 2*u) }

func arrHash(i, k int) float64 { return jpHash(i, k, 131) }

// arrWind: how hard the rotor wash blows (palms, grass, dust)
func arrWind(t float64) float64 {
	if t >= arrBoom {
		return 0
	}
	return clampF((t-1.4)/1.0, 0, 1) * arrRPM(t)
}

// arrAlt: the height of the helicopter above the ground in pixels
func arrAlt(t float64) float64 {
	switch {
	case t < arrFlyEnd:
		return 130 + math.Sin(t*3)*4
	case t < arrTouch:
		return 130 * (1 - arrSmooth((t-arrFlyEnd)/(arrTouch-arrFlyEnd)))
	}
	return 0
}

// arrCam: the top of the screen in the world picture
func arrCam(t float64) float64 {
	return arrCamEnd * arrSmooth(t/arrFlyEnd)
}

// the pieces that fly off: size of each kind
var arrPartSize = [][2]float64{
	{120, 6}, {96, 6}, {70, 5}, {70, 5}, {100, 22}, {14, 40}, {40, 30}, {40, 30}, {30, 24}, {36, 40}, {26, 18}, {8, 26},
}

// advance moves the scene on by dt. A first press arms the skip, a second one within 2.2 s ends it.
// It returns the sounds to play and whether the scene is over.
func (a *arrivalState) advance(dt float64, press bool) (sounds []string, done bool) {
	a.t += dt
	if a.skipArm > 0 {
		a.skipArm -= dt
	}
	if press {
		if a.skipArm > 0 {
			return nil, true
		}
		a.skipArm = 2.2
	}
	a.rot += dt * arrRPM(a.t) * 38
	if !a.touched && a.t >= arrTouch {
		a.touched = true
		sounds = append(sounds, "thud")
	}
	if !a.boomed && a.t >= arrBoom {
		a.boomed = true
		a.spawnParts()
		sounds = append(sounds, "boom", "spark")
	}
	if a.boomed {
		a.stepParts(dt)
	}
	return sounds, a.t >= arrTotal
}

// spawnParts: the blades, the skids, the tail and the panels of the helicopter, in its own colours.
func (a *arrivalState) spawnParts() {
	body, trim, dark := rgb(0xf2c230), rgb(0xd0402a), rgb(0x2a2a34)
	cols := []color.RGBA{dark, dark, rgb(0x3a3a46), rgb(0x3a3a46), body, trim, body, shade(body, 0.8), trim, body, rgb(0x7ab0d8), dark}
	for k := range arrPartSize {
		h1, h2, h3 := arrHash(k, 1), arrHash(k, 2), arrHash(k, 3)
		ang := h1 * 2 * math.Pi
		sp := 110 + h2*170
		a.parts = append(a.parts, carPart{x: arrCX + (h3-0.5)*30, y: arrGroundY, z: 18 * arrK,
			vx: math.Cos(ang) * sp, vy: math.Sin(ang) * sp * 0.4, vz: 150 + h2*200,
			kind: k, col: cols[k], spin: (h3 - 0.5) * 32})
	}
}

// stepParts: the pieces fly, fall and bounce (the same physics as the car that blows up).
func (a *arrivalState) stepParts(dt float64) {
	for i := range a.parts {
		p := &a.parts[i]
		if p.rest {
			continue
		}
		p.x += p.vx * dt
		p.y += p.vy * dt
		p.z += p.vz * dt
		p.vz -= 520 * dt
		p.rot += p.spin * dt
		if p.z <= 0 {
			p.z = 0
			if math.Abs(p.vz) < 50 {
				p.rest = true
				continue
			}
			p.vz = -p.vz * 0.4
			p.vx *= 0.6
			p.vy *= 0.6
			p.spin *= 0.5
		}
	}
}

// ---- the people ----------------------------------------------------------------------------------

type arrActor struct {
	vis       bool
	x, y      float64 // the feet
	anim      int
	dir       Dir
	ph        float64
	lean, hop float64
	clip      bool // still inside the door: only what is in the doorway shows
	dizzy     bool
	soot      float64 // 0..1: blackened by the blast
}

// arrActorAt: where player i (of n) is and what he does at time t.
func arrActorAt(i, n int, t float64) arrActor {
	x0, fx, fy := arrCX+15, 410.0, 470.0
	if n == 2 {
		if i == 0 {
			x0, fx, fy = arrCX+6, 400, 470
		} else {
			x0, fx, fy = arrCX+24, 570, 478
		}
	}
	start, walk := arrExit1, 1.4
	if i == 1 {
		start, walk = arrExit2, 0.95
	}
	inY, outY := arrCY+12*arrK-4, arrCY+18*arrK+14
	a := arrActor{dir: DirDown}
	if t < arrDoor+0.15 {
		return a // still seen through the glass
	}
	a.vis = true
	tt := t - start
	switch {
	case tt < 0: // in the doorway
		a.x, a.y, a.clip = x0, inY, true
		if i == 1 {
			a.anim, a.ph = AnimPanic, t*9
		}
	case tt < 0.45: // steps down out of the cabin
		u := tt / 0.45
		a.x, a.y, a.clip = x0, inY+(outY-inY)*u, true
		a.anim, a.ph = AnimWalk, tt*8
	default:
		u := clampF((tt-0.45)/walk, 0, 1)
		e := arrSmooth(u)
		a.x, a.y = x0+(fx-x0)*e, outY+(fy-outY)*e
		switch {
		case u >= 1:
			a.dir = DirUp
			if i == 1 {
				a.anim, a.ph, a.dizzy = AnimTired, t*3, true
				a.lean = 0.1 * math.Sin(t*3)
			}
		default:
			a.dir = DirDown
			if u > 0.3 {
				a.dir = DirRight
				if fx < x0 {
					a.dir = DirLeft
				}
			}
			if i == 0 {
				a.anim, a.ph = AnimWalk, tt*7
			} else {
				a.anim, a.ph = AnimRun, tt*11
				a.x += math.Sin(u*10) * 16 * (1 - u)
				a.lean = 0.14 * math.Sin(tt*10) * (1 - u*0.5)
			}
		}
	}
	if t >= arrBoom { // the blast pushes them back, blackens them, and they look at each other
		bt := t - arrBoom
		sgn := 1.0
		if a.x < arrCX {
			sgn = -1
		}
		a.x += sgn * 14 * easeOut(bt/0.35)
		a.soot = arrSmooth(bt / 0.3)
		a.dizzy, a.clip = false, false
		a.dir, a.anim, a.ph, a.hop = DirUp, AnimIdle, 0, 0
		a.lean = 0.15 * sgn * (1 - clampF(bt/0.6, 0, 1))
		if bt >= 0.9 {
			a.anim, a.ph, a.lean = AnimTalk, t*6, 0
			switch {
			case n == 1:
				a.dir = DirDown
			case i == 0:
				a.dir = DirRight
			default:
				a.dir = DirLeft
			}
		}
		if i == 1 && bt < 1.4 {
			a.anim, a.ph, a.hop = AnimPanic, t*10, math.Abs(math.Sin(t*12))*6
		}
	}
	return a
}

// arrLine: what somebody says and when.
type arrLine struct {
	who      int
	from, to float64
	text     string
}

// arrLines: the short jokes (each at most 48 characters; nobody speaks over somebody else).
func arrLines(n int) []arrLine {
	out := []arrLine{
		{0, 0.4, 1.6, "Qué vistas, ¿eh?"},
		{0, 2.9, 3.7, "Aterrizaje de diez."},
	}
	if n == 2 {
		out = append(out,
			arrLine{1, 1.7, 2.8, "¿Falta mucho? ¡Me mareo!"},
			arrLine{1, 4.75, 5.9, "¡Suelo! ¡Cómo te echaba de menos!"},
			arrLine{0, arrBoom + 0.7, arrBoom + 1.6, "Eso no sonaba bien."},
			arrLine{1, arrBoom + 1.7, arrTotal, "¡Era alquilado!"})
	} else {
		out = append(out,
			arrLine{0, 4.5, 5.8, "Qué sitio tan tranquilo."},
			arrLine{0, arrBoom + 0.7, arrBoom + 1.6, "Eso no sonaba bien."},
			arrLine{0, arrBoom + 1.7, arrTotal, "Y era alquilado..."})
	}
	return out
}

// ---- the flow ------------------------------------------------------------------------------------

// startArrival: the helicopter ride to the island (after the letter, or watched again from the scenes).
func (g *Game) startArrival() {
	g.end = &endState{stage: endArrival}
	g.mode = ModeEnding
	g.sceneReplay = false
	g.markSeen("seen_arrival")
}

// updateArrival runs the scene; true while it has the control of the ending.
func (g *Game) updateArrival(e *endState) bool {
	a := &e.x.arr
	press := actConfirm.just() || actBack.just() || ebitenClick()
	sounds, done := a.advance(1.0/60, press)
	for _, s := range sounds {
		g.snd.Play(s)
	}
	if done {
		g.finishArrival(e)
	}
	return true
}

// finishArrival: the party of the island begins (or back to the list of scenes when it was only watched).
func (g *Game) finishArrival(e *endState) {
	if g.sceneReplay {
		g.leaveEnding()
		return
	}
	g.end = nil
	g.snd.StopScene()
	g.startLevel(specialJurassic)
}

// arrMusicVol: the music (the theme of the invitation) swells in and fades with the picture.
func arrMusicVol(t float64) float64 {
	return 0.55 * math.Min(1, t/0.8) * (1 - clampF((t-(arrTotal-0.9))/0.9, 0, 1))
}

// ---- the world picture ---------------------------------------------------------------------------

func arrCoastY(x float64) float64 { return 700 + 18*math.Sin(x/90) + 10*math.Sin(x/37+1) }

var (
	arrHeliSide, arrHeliFront *ebiten.Image
)

func arrHeliImgs() (side, front *ebiten.Image) {
	if arrHeliSide == nil {
		arrHeliSide = genHeliPal(0xf2c230, 0xd0402a, 0).scaled(wk).outlinedSel(0.45).img()
		arrHeliFront = genHeliView(0xf2c230, 0xd0402a, 0, true).scaled(wk).outlinedSel(0.45).img()
	}
	return arrHeliSide, arrHeliFront
}

// arrBuildWorld paints, once, the sea, the beach, the jungle and the helipad (960 x 1160).
func arrBuildWorld() *ebiten.Image {
	w := ebiten.NewImage(screenW, arrWorldH)
	for y := 0.0; y < 760; y += 4 { // the sea, deeper far out
		fillRect(w, 0, y, screenW, 4, mixc(rgb(0x1a5a96), rgb(0x2f8cc0), clampF(y/700, 0, 1)))
	}
	for x := 0.0; x < screenW; x += 8 { // the shallows
		top := arrCoastY(x) - 70 - 16*math.Sin(x/53+2)
		fillRect(w, x, top, 8, arrCoastY(x)-top, rgba(0x58c0d0, 120))
	}
	for x := 0.0; x < screenW; x += 8 {
		cy := arrCoastY(x)
		fillRect(w, x, cy-12-arrHash(int(x), 5)*8, 8, 6, rgb(0xcfeef0)) // the foam
		fillRect(w, x, cy-4, 8, 4, rgb(0xa8e0e4))
		fillRect(w, x, cy, 8, 56, rgb(0xe4cc8c)) // the sand
		fillRect(w, x, cy, 8, 5, rgb(0xc8a868))
		fillRect(w, x, cy+56, 8, arrWorldH-cy-56, rgb(0x3f9a46)) // the jungle
		fillRect(w, x, cy+56, 8, 44, rgb(0x2f7a3a))
	}
	for i := 0; i < 120; i++ { // tufts and patches of grass
		x, y := arrHash(i, 20)*screenW, 800+arrHash(i, 21)*360
		col := rgb(0x358a3c)
		if i%3 == 0 {
			col = rgb(0x4aa850)
		}
		filledEllipse(w, x, y, 16+arrHash(i, 22)*14, 5+arrHash(i, 23)*4, col)
	}
	fillDisc(w, arrPadWX, arrPadWY, 38*arrK+14, rgb(0x6f7378)) // the concrete round the pad
	fillDisc(w, arrPadWX, arrPadWY, 38*arrK+8, rgb(0x8e9296))
	pb := newPix(80, 80)
	jpHelipad(pb, Vec{40, 40}, 34)
	pad := pb.img()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(arrK, arrK)
	op.GeoM.Translate(arrPadWX-40*arrK, arrPadWY-40*arrK)
	w.DrawImage(pad, op)
	return w
}

var arrPalms = [][2]float64{
	{110, 812}, {300, 790}, {660, 795}, {850, 820}, {190, 880}, {770, 885}, // behind the pad
	{130, 1060}, {840, 1066}, {300, 1100}, {680, 1104}, // in front of it
}

// arrPalm draws one palm: the trunk bends and the fronds flutter with the wind.
func arrPalm(cv *ebiten.Image, bx, by float64, i int, t, wind float64) {
	lean := wind*0.20*math.Sin(t*14+float64(i)) + 0.03*math.Sin(t*1.7+float64(i))
	var x, y float64
	for j := 0; j < 5; j++ {
		f := float64(j) / 5
		x, y = bx+lean*f*f*130, by-f*120
		w := 9 - f*3
		fillRect(cv, math.Round(x-w/2), math.Round(y-24), w, 26, rgb(0x8a5a2a))
		fillRect(cv, math.Round(x-w/2), math.Round(y-24), 2, 26, rgb(0x6a4220))
	}
	x, y = bx+lean*130, by-120
	for k := 0; k < 7; k++ {
		ang := float64(k)*2*math.Pi/7 + 0.3 + wind*0.25*math.Sin(t*20+float64(k))
		for m := 1; m <= 4; m++ {
			d := float64(m) * 14
			fx := x + math.Cos(ang)*d
			fy := y - 8 + math.Sin(ang)*d*0.45 + float64(m*m)*1.6
			col := rgb(0x2e8a3a)
			if (k+m)%2 == 0 {
				col = rgb(0x3aa04a)
			}
			fillRect(cv, math.Round(fx-6), math.Round(fy-3), 12, 6, col)
		}
	}
	fillDisc(cv, math.Round(x), math.Round(y-6), 5, rgb(0x6a4220))
}

// arrRotor: the blades round a hub.
func arrRotor(cv *ebiten.Image, hx, hy, rpm, rot float64) {
	const L = 29 * arrK
	if rpm > 0.5 {
		filledEllipse(cv, hx, hy, L, 16, rgba(0x22222a, uint8(46*rpm)))
	}
	for k := 0; k < 3; k++ {
		ang := rot + float64(k)*2*math.Pi/3
		dx := math.Cos(ang) * L
		yy := hy + math.Sin(ang)*14
		al := uint8(235)
		if rpm > 0.5 {
			al = 110
		}
		x0, x1 := math.Min(hx, hx+dx), math.Max(hx, hx+dx)
		fillRect(cv, math.Round(x0), math.Round(yy-2), math.Max(4, math.Round(x1-x0)), 5, rgba(0x22222a, al))
	}
	fillRect(cv, hx-6, hy-4, 12, 8, rgb(0x3a3a46))
}

// arrHead draws the top of a player (his head) at scale 2 (like his whole body), centred on cx with the top of the head at topY.
func (g *Game) arrHead(cv *ebiten.Image, cs *CharSet, cx, topY float64, dir Dir, tint [3]float32) {
	if cs == nil {
		return
	}
	img, flip := cs.img(AnimIdle, dir, 0)
	if img == nil {
		return
	}
	b := img.Bounds()
	sub := img.SubImage(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+20)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(float64(b.Dx()), 0)
	}
	op.GeoM.Translate(-charCX, 0)
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(math.Round(cx), math.Round(topY))
	op.ColorScale.Scale(tint[0], tint[1], tint[2], 1)
	cv.DrawImage(sub, op)
}

// arrRect draws a turned rectangle (a piece of the helicopter).
func arrRect(cv *ebiten.Image, x, y, w, h, rot float64, col color.RGBA) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Rotate(rot)
	op.GeoM.Translate(math.Round(x), math.Round(y))
	op.ColorScale.ScaleWithColor(col)
	cv.DrawImage(px, op)
}

// ---- drawing -------------------------------------------------------------------------------------

func (g *Game) arrNumPlayers() int {
	if g.scene2P() {
		return 2
	}
	return 1
}

func (g *Game) drawArrival(dst *ebiten.Image, e *endState) {
	a := &e.x.arr
	if a.world == nil {
		a.world = arrBuildWorld()
	}
	if a.canvas == nil {
		a.canvas = ebiten.NewImage(screenW, screenH)
	}
	cv, t := a.canvas, a.t
	n := g.arrNumPlayers()
	var cs [2]*CharSet
	cs[0] = g.sceneHero(0, false)
	if n == 2 {
		cs[1] = g.sceneHero(1, false)
	}
	cam := math.Round(arrCam(t))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, -cam)
	cv.DrawImage(a.world, op)
	// the waves
	for i := 0; i < 46; i++ {
		sy := 20 + arrHash(i, 1)*640 - cam
		if sy < -4 || sy > screenH {
			continue
		}
		wx := math.Mod(arrHash(i, 2)*screenW+t*(14+arrHash(i, 3)*14), screenW+40) - 20
		fillRect(cv, math.Round(wx), math.Round(sy), 14+arrHash(i, 4)*16, 2, rgba(0x9ad8f0, 150))
	}
	bt := t - arrBoom
	wind := arrWind(t)
	padSY := arrPadWY - cam
	// the scorch mark the helicopter leaves: it fades away
	if a.boomed {
		k := clampF((bt-0.15)/0.3, 0, 1) * (1 - clampF((bt-1.6)/1.7, 0, 1))
		if k > 0 {
			filledEllipse(cv, arrPadWX, padSY+4, 110, 40, rgba(0x3a2a1c, uint8(120*k)))
			filledEllipse(cv, arrPadWX, padSY+4, 84, 30, rgba(0x14141a, uint8(190*k)))
		}
	}
	// the grass lies down under the rotor wash
	for k := 0; k < 30; k++ {
		ang := arrHash(k, 30) * 2 * math.Pi
		r := 190 + arrHash(k, 31)*90
		gx, gy := arrPadWX+math.Cos(ang)*r, padSY+math.Sin(ang)*r*0.5
		lean := wind * 8 * math.Cos(ang) * (0.6 + 0.4*math.Sin(t*18+float64(k)))
		for b := -1; b <= 1; b++ {
			fillRect(cv, math.Round(gx+float64(b)*3+lean), math.Round(gy-9), 2, 9, rgb(0x2a7a34))
		}
	}
	// palms behind the pad
	for i, p := range arrPalms {
		if sy := p[1] - cam; sy < arrGroundY && sy > -10 {
			arrPalm(cv, p[0], sy, i, t, wind)
		}
	}
	// the helicopter
	if !a.boomed {
		g.arrDrawHeli(cv, a, cs, n)
	}
	// the fire ball, behind the people
	if a.boomed {
		arrFire(cv, bt)
	}
	for i, p := range arrPalms {
		if sy := p[1] - cam; sy >= arrGroundY && sy < screenH+130 {
			arrPalm(cv, p[0], sy, i, t, wind)
		}
	}
	// dust and leaves blown by the rotor
	for k := 0; k < 46 && wind > 0.05; k++ {
		ang := arrHash(k, 10) * 2 * math.Pi
		sp := 70 + arrHash(k, 11)*120
		r := math.Mod(t*sp*0.9+arrHash(k, 12)*220, 260)
		al := wind * (1 - r/260) * 170
		if al < 8 {
			continue
		}
		size := 6 + (1-r/260)*10
		hex := uint32(0xd8c08a)
		if k%4 == 0 {
			hex = 0x4aa850
		}
		col := rgba(hex, uint8(al))
		fillRect(cv, math.Round(arrPadWX+math.Cos(ang)*(60+r)), math.Round(padSY+math.Sin(ang)*(60+r)*0.42-4), math.Round(size), math.Round(size*0.45), col)
	}
	if a.touched && t < arrTouch+0.5 { // the cloud of the landing
		u := (t - arrTouch) / 0.5
		for k := 0; k < 8; k++ {
			side := float64(k%2*2 - 1)
			x := arrCX + side*(60+u*(60+float64(k)*14))
			filledEllipse(cv, x, arrGroundY+2-u*10, 26+u*30, 14+u*10, rgba(0xc8b080, uint8(170*(1-u))))
		}
	}
	// smoke of the wreck, behind the people
	if a.boomed {
		arrSmoke(cv, bt)
	}
	// the people (the lower ones in front)
	order := []int{0, 1}
	if n == 2 && arrActorAt(1, 2, t).y < arrActorAt(0, 2, t).y {
		order = []int{1, 0}
	}
	var acts [2]arrActor
	for i := 0; i < n; i++ {
		acts[i] = arrActorAt(i, n, t)
	}
	for _, i := range order {
		if i >= n {
			continue
		}
		g.arrDrawActor(cv, cs, i, n, acts[i], t)
	}
	// the pieces, the shock wave and the sparks fly in front of everything
	if a.boomed {
		arrDebris(cv, a, bt)
		arrShock(cv, bt)
		if bt < 0.25 {
			fillRect(cv, 0, 0, screenW, screenH, rgba(0xfff4d0, uint8(230*(1-bt/0.25))))
		}
	}
	// the jokes
	for _, l := range arrLines(n) {
		if t < l.from || t >= l.to || cs[l.who] == nil {
			continue
		}
		var x, y float64
		if ac := acts[l.who]; ac.vis {
			x, y = ac.x, ac.y-92-ac.hop
		} else {
			x, y = arrCabinHead(l.who, n, t)
			y -= 4
		}
		g.speechBubble(cv, x, y, l.text)
	}
	// the picture, shaken by the landing and the blast
	var sx, sy float64
	amp := 0.0
	if a.touched && t < arrTouch+0.25 {
		amp = 3 * (1 - (t-arrTouch)/0.25)
	}
	if a.boomed {
		amp = math.Max(amp, 11*math.Max(0, 1-bt/0.9))
	}
	if amp > 0 {
		sx, sy = math.Round(math.Sin(t*71)*amp), math.Round(math.Cos(t*67)*amp)
	}
	dst.Fill(rgb(0x02060e))
	dop := &ebiten.DrawImageOptions{}
	dop.GeoM.Translate(sx, sy)
	dst.DrawImage(cv, dop)
	if t < 2.4 {
		al := clampF(t/0.5, 0, 1) * (1 - clampF((t-1.8)/0.6, 0, 1))
		g.drawTextC(dst, "VUELO A ISLA NUBLAR: SIN TURBULENCIAS (CASI)", 480, 468, rgba(0xf4f0ff, uint8(255*al)), 2)
	}
	if t < 0.5 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-t/0.5))))
	}
	if t > arrGagEnd {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*clampF((t-arrGagEnd)/(arrTotal-arrGagEnd), 0, 1))))
	}
	hint := actionHint(actConfirm) + " / " + actionHint(actBack) + ": saltar (dos veces)"
	col := rgba(0x8a86a0, 170)
	if a.skipArm > 0 {
		hint, col = "Otra vez para saltar", rgb(0xffe060)
	}
	g.drawTextC(dst, hint, 480, 524, col, 1)
}

// arrCabinHead: where the head of player i is seen through the glass (for his joke).
func arrCabinHead(i, n int, t float64) (x, topY float64) {
	cx, cy := arrCX, arrCY-arrAlt(t)
	if t < arrTouch {
		if n == 2 {
			return cx + float64(i*2-1)*16, cy - 4*arrK - 22
		}
		return cx, cy - 4*arrK - 22
	}
	if n == 2 {
		return cx - 15*arrK + float64(i*2-1)*18, cy - 6*arrK - 22
	}
	return cx - 15*arrK, cy - 6*arrK - 22
}

// arrDrawHeli: the helicopter (seen from the front while it comes down, from the side once it has landed).
func (g *Game) arrDrawHeli(cv *ebiten.Image, a *arrivalState, cs [2]*CharSet, n int) {
	t := a.t
	alt := arrAlt(t)
	cx, cy := arrCX, arrCY-alt
	side, front := arrHeliImgs()
	// the shadow on the sea or the ground
	sw := 40*arrK - 10 - alt*0.25
	filledEllipse(cv, cx+alt*0.3, arrGroundY+4+alt*0.1, sw, 34-alt*0.05, rgba(0x000000, 80))
	var tint [2][3]float32
	for i := 0; i < n; i++ {
		tint[i] = sceneTint(i, cs[0], cs[i])
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(arrK, arrK)
	if !a.touched {
		op.GeoM.Translate(math.Round(cx-33*arrK), math.Round(cy-35*arrK))
		cv.DrawImage(front, op)
		for i := 0; i < n; i++ {
			hx, hy := arrCabinHead(i, n, t)
			g.arrHead(cv, cs[i], hx, hy, DirDown, tint[i])
		}
		arrRotor(cv, cx, cy-25*arrK, arrRPM(t), a.rot)
		return
	}
	op.GeoM.Translate(math.Round(cx-42*arrK), math.Round(cy-35*arrK))
	cv.DrawImage(side, op)
	if t < arrDoor+0.15 {
		for i := 0; i < n; i++ {
			hx, hy := arrCabinHead(i, n, t)
			g.arrHead(cv, cs[i], hx, hy, DirLeft, tint[i])
		}
	}
	// the sliding door
	dx0, dy0, dw, dh := cx-6*arrK, cy-12*arrK, 22*arrK, 24*arrK
	slide := clampF((t-arrDoor)/arrDoorLen, 0, 1)
	if slide > 0 {
		fillRect(cv, dx0, dy0, dw, dh, rgb(0x1c1a24))
		fillRect(cv, dx0+dw-20, dy0+dh-32, 18, 28, rgb(0x3a3a48)) // the seats
		fillRect(cv, dx0+4, dy0+dh-28, 16, 24, rgb(0x3a3a48))
	}
	if slide < 1 {
		w := math.Round(dw * (1 - slide))
		x := dx0 + dw - w
		fillRect(cv, x, dy0, w, dh, rgb(0xc89a1c))
		fillRect(cv, x+2, dy0+2, math.Max(0, w-4), dh-4, rgb(0xf2c230))
		if w > 24 {
			fillRect(cv, x+6, dy0+8, w-12, 18, rgb(0x7ab0d8))
			fillRect(cv, x+6, dy0+8, w-12, 3, rgb(0xa8d8f0))
		}
	}
	arrRotor(cv, cx-arrK, cy-25*arrK, arrRPM(t), a.rot)
	if rpm := arrRPM(t); rpm > 0.1 { // the tail rotor
		fillRect(cv, cx+50*arrK, cy-10*arrK-14, 3, 8+28*math.Abs(math.Cos(a.rot*1.7)), rgba(0x22222a, 170))
	}
}

// arrDrawActor: a player on the pad, with the ring of his colour under his feet.
func (g *Game) arrDrawActor(cv *ebiten.Image, cs [2]*CharSet, i, n int, ac arrActor, t float64) {
	if !ac.vis || cs[i] == nil {
		return
	}
	ringCol := rgba(0x4a9aff, 150)
	if i == 1 {
		ringCol = rgba(0xff5a5a, 150)
	}
	tint := sceneTint(i, cs[0], cs[i])
	k := float32(ac.soot)
	tint[0] *= 1 - 0.28*k // a smudge of soot: still easy to tell who is who
	tint[1] *= 1 - 0.32*k
	tint[2] *= 1 - 0.36*k
	target := cv
	if ac.clip {
		target = cv.SubImage(image.Rect(int(arrCX-6*arrK), int(arrCY-12*arrK), int(arrCX+16*arrK), screenH)).(*ebiten.Image)
	}
	filledEllipse(target, math.Round(ac.x), math.Round(ac.y), 22, 8, ringCol)
	g.sceneChar(target, cs[i], ac.anim, ac.dir, ac.ph, ac.x, ac.y, 2, 1, ac.hop, ac.lean, tint)
	if ac.dizzy { // little stars go round his head
		for s := 0; s < 3; s++ {
			a := t*6 + float64(s)*2.1
			fillDisc(cv, math.Round(ac.x+math.Cos(a)*16), math.Round(ac.y-90+math.Sin(a)*5), 3, rgb(0xf0d040))
		}
	}
}

// arrFire: the fire ball (flash, then three layers that rise and fade).
func arrFire(cv *ebiten.Image, bt float64) {
	if bt > 1.1 {
		return
	}
	cx, cy := arrCX, arrCY+10
	grow := math.Min(1, bt/0.35)
	fade := 1 - math.Max(0, (bt-0.35)/0.75)
	r := 40 + 110*grow
	al := func(v float64) uint8 { return uint8(255 * math.Max(0, math.Min(1, v*fade))) }
	fillDisc(cv, cx, cy-40*grow, r, rgba(0xe84610, al(0.9)))
	fillDisc(cv, cx, cy-44*grow, r*0.72, rgba(0xffa01e, al(0.95)))
	fillDisc(cv, cx, cy-48*grow, r*0.42, rgba(0xffec8c, al(1)))
	for k := 0; k < 5; k++ { // smaller blasts a moment later (the fuel)
		s := 0.1 + 0.12*float64(k)
		u := (bt - s) / 0.55
		if u < 0 || u > 1 {
			continue
		}
		x := cx + (arrHash(k, 40)-0.5)*210
		y := cy + (arrHash(k, 41)-0.5)*90 - 20 - u*30
		fillDisc(cv, x, y, 18+30*math.Sin(u*math.Pi), rgba(0xff8c1c, uint8(230*(1-u))))
		fillDisc(cv, x, y, 9+16*math.Sin(u*math.Pi), rgba(0xffe278, uint8(230*(1-u))))
	}
	if bt > 0.4 && bt < 1.7 { // flames on the pad
		for k := 0; k < 5; k++ {
			h := 10 + 12*arrHash(k+int(bt*12), 43)
			fillRect(cv, arrCX-90+float64(k)*44, arrGroundY-h+8, 14, h, rgba(0xff8218, uint8(200*(1-(bt-0.4)/1.3))))
		}
	}
}

// arrSmoke: the black smoke that rises from the wreck and thins out.
func arrSmoke(cv *ebiten.Image, bt float64) {
	for i := 0; i < 16; i++ {
		age := (bt - 0.1 - 0.04*float64(i)) / 2.8
		if age <= 0 || age >= 1 {
			continue
		}
		x := arrCX + (arrHash(i, 50)-0.5)*180 + age*40
		y := arrCY + 50 - age*190 - arrHash(i, 51)*40
		filledEllipse(cv, x, y, 18+40*age, 15+34*age, rgba(0x3a3a46, uint8(190*(1-age))))
	}
}

// arrShock: the shock wave along the ground and the sparks.
func arrShock(cv *ebiten.Image, bt float64) {
	if bt < 0.7 {
		rr := 20 + bt*700
		al := uint8(220 * (1 - bt/0.7))
		for ang := 0.0; ang < 2*math.Pi; ang += 1.5 / rr {
			fillRect(cv, math.Round(arrCX+math.Cos(ang)*rr), math.Round(arrGroundY+math.Sin(ang)*rr*0.38), 3, 2, rgba(0xffe8b4, al))
		}
	}
	for i := 0; i < 60; i++ {
		ang := arrHash(i, 60) * 2 * math.Pi
		sp := 120 + 380*arrHash(i, 61)
		al := clampF(1-bt/(0.5+arrHash(i, 62)), 0, 1)
		if al <= 0 {
			continue
		}
		x := arrCX + math.Cos(ang)*sp*bt
		y := arrCY + math.Sin(ang)*sp*bt*0.6 + 300*bt*bt
		fillRect(cv, math.Round(x), math.Round(y), 3, 3, rgba([]uint32{0xffc828, 0xff9628, 0xff6428}[i%3], uint8(255*al)))
	}
}

// arrDebris: the blades, the skids, the tail and the panels flying through the air, fading out.
func arrDebris(cv *ebiten.Image, a *arrivalState, bt float64) {
	fade := 1 - clampF((bt-1.7)/0.6, 0, 1)
	if fade <= 0 {
		return
	}
	for _, p := range a.parts {
		sz := arrPartSize[p.kind]
		sz = [2]float64{sz[0] * arrK / 4, sz[1] * arrK / 4}
		if !p.rest && p.z > 0.5 { // the shadow under a flying piece
			fillRect(cv, math.Round(p.x-sz[0]/4), math.Round(p.y), sz[0]/2, 2, rgba(0x000000, uint8(60*fade)))
		}
		col := p.col // premultiplied by hand: the alpha fades the whole piece
		col.R, col.G, col.B, col.A = uint8(float64(col.R)*fade), uint8(float64(col.G)*fade), uint8(float64(col.B)*fade), uint8(255*fade)
		arrRect(cv, p.x, p.y-p.z, sz[0], sz[1], p.rot, col)
	}
}
