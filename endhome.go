package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The secret ending, last scene before "FIN": home at last.
//
// The same bedroom as the intro (and the menu), but the other way round: he comes in through the door,
// gets into bed and the window shows the house across the street dark and silent. No music: only a
// cricket. The neighbour asks HIM for silence, the alarm clock tries its luck and loses, and he sleeps.
// ---------------------------------------------------------------------------

// endHome is a stage of the ending (ending.go). Its number skips 10, which -endstage keeps for the
// prelude of the letter: "-ending 2 -endstage 11" plays this scene on its own.
const endHome = 11

const homeLen = 18.0

// the timeline, in seconds
const (
	homeDoorOpen  = 0.6  // the door opens onto the stairwell
	homeIn        = 1.3  // he comes in
	homeAtBed     = 4.6  // he is by the bed
	homeDoorShut  = 3.0  // the door closes behind him
	homeInBed     = 5.4  // he is in bed
	homeShh0      = 8.4  // the neighbour's light: "Shhh!"
	homeShh1      = 11.0 // and off again
	homeRing      = 13.6 // the alarm clock rings...
	homePlaf      = 14.9 // ...and is silenced
	homeFadeStart = 16.4
)

// homeDoor: 0 closed .. 1 open.
func homeDoor(t float64) float64 {
	o := introEase((t - homeDoorOpen) / 0.6)
	if t > homeDoorShut {
		o *= 1 - introEase((t-homeDoorShut)/0.5)
	}
	return o
}

// homeHeroPos: the path of the intro, backwards: from the door to the side of the bed.
func homeHeroPos(t float64, i int) (x, y, s float64) {
	u := introEase((t - homeIn - 0.5*float64(i)) / (homeAtBed - homeIn))
	x = 72 + (250+90*float64(i)-72)*u
	y = 318 + (432-318)*u
	s = 3 + u
	return
}

func homeCaption(t float64) string {
	switch {
	case t < homeAtBed:
		return "Mientras, en casa: llave, puerta y pijama."
	case t < homeShh0:
		return "La ventana: ni luces, ni música, ni vecinos."
	case t < homeShh1:
		return "Ahora es el vecino el que pide silencio."
	case t < homeRing:
		return "Sólo canta un grillo. Bajito, por educación."
	case t < homePlaf+1:
		return "El despertador lo intenta. Hoy no, amigo."
	}
	return "Silencio total. Zzz..."
}

// homeSounds: the door, the alarm clock and its end. Crickets come from updateEnding.
func (g *Game) homeSounds(e *endState) {
	for i, s := range []struct {
		at   float64
		name string
	}{{homeDoorOpen, "thud"}, {homeDoorShut + 0.4, "thud"}, {homeRing, "ring"}, {homeRing + 0.45, "ring"}, {homeRing + 0.9, "ring"}, {homePlaf, "thud"}} {
		if e.t >= s.at && !e.x.home[i] {
			e.x.home[i] = true
			g.snd.Play(s.name)
		}
	}
}

func (g *Game) drawHome(dst *ebiten.Image, e *endState) {
	if e.bed == nil {
		e.bed = ebiten.NewImage(screenW, screenH)
	}
	t := e.t
	two := g.scene2P()
	c := e.bed
	c.Fill(rgb(0x08060c))
	// the room of the intro, quiet: the bed is empty until he gets in; his Zs are drawn crisp, below
	oq, oe, on := g.bedQuiet, g.bedEmpty, g.bedNoTalk
	g.bedQuiet, g.bedEmpty, g.bedNoTalk = true, t < homeInBed, true
	g.time = e.total
	if two {
		bedTwo, bedPoses = true, [2]bedPose{}
	}
	g.drawBedroom(c)
	bedTwo = false
	g.bedQuiet, g.bedEmpty, g.bedNoTalk = oq, oe, on
	g.homeDoorDraw(c, t)
	g.homeGags(c, t, two)
	if t < homeInBed {
		g.homeHeroes(c, t, two)
	}
	// the camera: the whole room, then a slow zoom on the bed
	z := 1 + 0.18*introEase((t-homeInBed)/6)
	fx, fy := 480.0, 300.0
	ox, oy := math.Round(fx-fx*z), math.Round(fy-fy*z)
	dst.Fill(rgb(0x05060c))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(z, z)
	op.GeoM.Translate(ox, oy)
	dst.DrawImage(c, op)
	at := func(x, y float64) (float64, float64) { return math.Round(ox + x*z), math.Round(oy + y*z) }
	g.homeTexts(dst, t, two, at)
	if t < 1 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-t))))
	}
	if t > homeFadeStart {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000008, uint8(255*introEase((t-homeFadeStart)/(homeLen-homeFadeStart-0.2)))))
	}
	g.endCaption(dst, homeCaption(t))
}

// homeDoorDraw: the door of the intro, without the bass shaking it.
func (g *Game) homeDoorDraw(c *ebiten.Image, t float64) {
	x, y := doorX, doorBase-doorH
	o := homeDoor(t)
	fillRect(c, x-6, y-6, doorW+12, doorH+6, rgb(0x2a1e18))
	fillRect(c, x-6, y-6, doorW+12, 2, rgb(0x4a3528))
	if o > 0 { // the stairwell, lit and quiet
		fillRect(c, x, y, doorW, doorH, rgb(0xf0c870))
		fillRect(c, x, y, doorW, 10, rgb(0xd8a850))
	}
	leaf := doorW * (1 - 0.86*o)
	if leaf > 1 {
		fillRect(c, x, y, leaf, doorH, rgb(0x4a3322))
		fillRect(c, x+leaf*0.14, y+8, leaf*0.72, 48, rgb(0x3a281a))
		fillRect(c, x+leaf*0.14, y+66, leaf*0.72, 58, rgb(0x3a281a))
		fillRect(c, x+leaf-3, y, 3, doorH, rgb(0x2a1e14))
		fillRect(c, x+leaf*0.86-3, y+doorH*0.55, 5, 5, rgb(0xe8c060))
	}
}

// homeHeroes: he (or they) walk from the door to the bed and hop in.
func (g *Game) homeHeroes(c *ebiten.Image, t float64, two bool) {
	n := 1
	if two {
		n = 2
	}
	cs0, cs1 := g.sceneHero(0, false), g.sceneHero(1, false)
	for i := n - 1; i >= 0; i-- {
		cs := cs0
		if i == 1 {
			cs = cs1
		}
		if cs == nil {
			continue
		}
		x, y, s := homeHeroPos(t, i)
		start := homeIn + 0.5*float64(i)
		alpha := clampF((t-start+0.3)/0.6, 0, 1)
		anim, dir, ph := AnimWalk, DirRight, t*5
		if t < start+(homeAtBed-homeIn)*0.35 {
			dir = DirDown
		}
		hop := 0.0
		if t > homeAtBed+0.3*float64(i) { // a jump into bed
			u := clampF((t-homeAtBed-0.3*float64(i))/(homeInBed-homeAtBed-0.2), 0, 1)
			anim, dir, ph = AnimIdle, DirDown, 0
			hop = math.Sin(u*math.Pi) * 30
			alpha *= 1 - introEase((u-0.5)/0.5)
		}
		tint := [3]float32{1, 1, 1}
		if i == 1 {
			tint = sceneTint(1, cs0, cs1)
		}
		g.sceneChar(c, cs, anim, dir, ph, x, y, s, alpha, hop, 0, tint)
	}
}

// homeGags: the neighbour's light, the cricket on the sill, the alarm clock.
func (g *Game) homeGags(c *ebiten.Image, t float64, two bool) {
	if t >= homeShh0 && t < homeShh1 { // a window of the house across the street lights up
		fillRect(c, 500, 98, 14, 10, rgb(0xe8c860))
		fillRect(c, 505, 101, 4, 5, rgb(0x2a1e18)) // somebody with a finger on his lips
	}
	// the cricket on the window sill
	cx, cy := 640.0, 204.0
	leg := math.Round(math.Abs(math.Sin(t * 9)))
	fillRect(c, cx, cy, 8, 3, rgb(0x3a8a30))
	fillRect(c, cx+7, cy-1, 3, 3, rgb(0x4ab040))
	fillRect(c, cx+1, cy+3, 1, 2, rgb(0x2a6a20))
	fillRect(c, cx+5, cy+3-leg, 1, 2+leg, rgb(0x2a6a20))
	fillRect(c, cx+9, cy-4, 1, 3, rgb(0x2a6a20))
	if two {
		return
	}
	// the alarm clock (1P: on the bedside table): it rings, shakes, and after the slap it says nothing at all
	const kx, ky = 288.0, 306.0
	if t >= homeRing && t < homePlaf {
		sh := math.Round(math.Sin(t*60) * 2)
		fillRect(c, kx-2, ky-2, 28, 22, rgb(0x0a0a0e))
		fillRect(c, kx+sh, ky, 24, 18, rgb(0x1a1a22))
		g.drawText(c, "6:30", kx+2+sh, ky+4, rgb(0xff3030), 0.9)
	}
	if t >= homePlaf {
		fillRect(c, kx, ky, 24, 18, rgb(0x0a0a0e))
		fillRect(c, kx-6, ky+12, 36, 6, rgb(0xa2aabb)) // the pillow that landed on it
	}
}

// homeTexts: the little texts, on the screen at whole scales (the room is zoomed).
func (g *Game) homeTexts(dst *ebiten.Image, t float64, two bool, at func(x, y float64) (float64, float64)) {
	if t >= homeShh0+0.3 && t < homeShh1 {
		x, y := at(507, 96)
		g.speechBubble(dst, x, y, "¡Shhh! Que hay gente durmiendo.")
	}
	if t >= homeShh1 && t < homeRing {
		x, y := at(644, 196)
		a := uint8(255 * math.Abs(math.Sin(t*2.4)))
		g.drawTextC(dst, "cri, cri...", x, y-24, rgba(0xa8e890, a), 1.5)
	}
	if t >= homeRing && t < homePlaf {
		x, y := at(300, 300)
		if two {
			x, y = 480, 300
		}
		g.sceneBang(dst, "¡RIIING!", x+40, y-40, (t-homeRing)/(homePlaf-homeRing), 0xff5050, 3)
	}
	if t >= homePlaf && t < homePlaf+1.2 {
		x, y := at(300, 300)
		if two {
			x, y = 480, 300
		}
		g.sceneBang(dst, "¡PLAF!", x+40, y-40, (t-homePlaf)/1.2, 0xf0e060, 4)
	}
	if t < homeInBed+0.4 {
		return
	}
	heads := [][2]float64{{g.talkX, g.talkY}}
	if two {
		heads = [][2]float64{{bed1X0 + 70, 330}, {bed2X1 - 70, 330}}
	}
	for _, h := range heads { // the Zs: slow, big, sharp
		x0, y0 := at(h[0], h[1])
		for i := 0; i < 3; i++ {
			f := math.Mod(t*0.45+float64(i)/3, 1)
			g.drawText(dst, "Z", math.Round(x0+20+f*40+float64(i)*3), math.Round(y0-80-f*46), rgba(0xe8f0ff, uint8(240*(1-f))), float64(2+i))
		}
	}
}
