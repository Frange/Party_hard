package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The third interlude, between parties 20 (the train station) and 21 (the space station).
//
// The same angry walk as the other two, this time over the roof of the station, with the memories of
// the last parties. At the end of the roof there is a home-made rocket: a rusty barrel with a toilet for
// a nose, bundles of firecrackers for an engine and a sign that says CORREOS ESPACIALES. A guard asks him
// for his ticket, the countdown fails (pfffft), a kick, a second countdown, and off it goes towards the
// stars, where a few big shiny stars of real shape drift by. Then the space station.
// ---------------------------------------------------------------------------

const interRocket = 3

// the timeline of the part after the roof stops (interStopX / itStop are shared with the other two)
const (
	rkGuard   = 25.8 // the guard walks in
	rkAsk     = 26.4 // "your ticket, please"
	rkTicket  = 28.4 // he shows his knife: valid
	rkBoard   = 29.6 // he climbs into the toilet
	rkCount0  = 31.0 // 5 4 3 2 1
	rkTick0   = 0.75
	rkFizzle  = 35.0 // pfffft
	rkKick    = 36.8 // the guard kicks the barrel
	rkCount1  = 38.0 // 3 2 1
	rkTick1   = 0.8
	rkLift    = 40.5 // lift off
	rkSpace   = 43.5 // the stars
	rkFlash   = 48.0 // white flash and cut
	rocketLen = 49.5
)

const (
	rkX    = 585.0 // the rocket, on the roof
	rkBase = interGround + 16
	rkTop  = 150.0 // height of the barrel
)

// rocketWorldX: where the rocket is on the screen while the roof scrolls by (it waits at the end of it).
func rocketWorldX(t float64) float64 {
	return rkX + interScroll(itStop) - interScroll(math.Min(t, itStop))
}

// rocketHeight: how far the rocket has risen above its pad, in pixels.
func rocketHeight(t float64) float64 {
	if t < rkLift {
		return 0
	}
	u := t - rkLift
	return 0.5*150*u*u + 20*u
}

// rocketCam: how far the camera has followed it up (the world slides down the screen).
func rocketCam(t float64) float64 { return math.Max(0, rocketHeight(t)-170) }

// ---- music ------------------------------------------------------------------------------------

func rocketFade(t float64) float64 {
	v := math.Min(1, t/2.5)
	if t > rocketLen-3 {
		v *= 1 - introEase((t-(rocketLen-3))/2.6)
	}
	return v
}

func rocketBaseVol(t float64, n int) float64 {
	v := 0.55
	if _, _, a, ok := interMemory(t, n); ok {
		v *= 1 - a
	}
	return v * rocketFade(t)
}

func rocketMemVol(t float64, n int) float64 {
	if _, _, a, ok := interMemory(t, n); ok {
		return 0.6 * a * rocketFade(t)
	}
	return 0
}

// ---- update ------------------------------------------------------------------------------------

func (g *Game) updateRocket(in *interState) {
	if !in.music {
		in.music = true
		g.sceneMusic(sceneRocketN, ambRocket)
		if !g.snd.SceneIsSong("base") {
			g.snd.StartScene("memory", "memory:"+ambMemC, ambientBuilder(ambMemC), "", true)
		}
	}
	g.snd.SetSceneVolume("base", rocketBaseVol(in.t, len(in.bgs)))
	g.snd.SetSceneVolume("memory", rocketMemVol(in.t, len(in.bgs)))
	once := func(key string, at float64, name string) {
		if in.t >= at && !in.did[key] {
			in.did[key] = true
			g.snd.Play(name)
		}
	}
	if in.t < itBoard0 { // stomps
		if st := int(g.interPhase(in) * 2 / float64(animFrames[AnimWalk])); st != in.step && in.t > 1 {
			in.step = st
			g.snd.Play("thud")
		}
	}
	once("guard", rkGuard, "alert")
	once("ticket", rkTicket, "ring")
	once("board", rkBoard, "thud")
	for i := 0; i < 5; i++ {
		once(fmt.Sprint("c0", i), rkCount0+float64(i)*rkTick0, "ring")
	}
	once("fizzle", rkFizzle, "splash")
	once("fizzle2", rkFizzle+0.35, "spark")
	once("kick", rkKick+0.5, "thud")
	for i := 0; i < 3; i++ {
		once(fmt.Sprint("c1", i), rkCount1+float64(i)*rkTick1, "ring")
	}
	once("lift", rkLift, "boom")
	once("lift2", rkLift+0.25, "boom")
	once("space", rkSpace, "win")
	if in.t >= rocketLen {
		g.finishInterlude()
	}
}

// ---- the scene ---------------------------------------------------------------------------------

func (g *Game) drawRocketScene(dst *ebiten.Image, in *interState) {
	if in.canvas == nil {
		in.canvas = ebiten.NewImage(screenW, screenH)
	}
	c := in.canvas
	t := in.t
	W := interScroll(math.Min(t, itStop))
	cam := rocketCam(t)
	// the space behind everything: it shows when the camera climbs
	if cam > 0 {
		g.rocketSpace(dst, cam, t)
	} else {
		dst.Fill(rgb(0x070a1c))
	}
	c.Fill(rgb(0x070a1c))
	g.interPanel(c, in)
	g.rocketRoof(c, in, W)
	g.rocketActors(c, in)
	// camera: a stomp shake while walking, a rumble before the lift off, a hard shake while it takes off
	ox, oy := 0.0, cam
	if t < itBoard0 && t > 1 {
		k := math.Exp(-math.Mod(g.interPhase(in)*2/float64(animFrames[AnimWalk]), 1) * 7)
		oy += k * 3
		ox = math.Sin(t*50) * k
	}
	if t > rkCount1+1.6 && t < rkLift+3 {
		k := math.Min(1, (t-rkCount1-1.6)/1.2) * (1 - introEase((t-rkLift-1.5)/1.5))
		ox += math.Sin(t*67) * 2.6 * k
		oy += math.Cos(t*59) * 2.2 * k
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox, oy)
	dst.DrawImage(c, op)
	if t >= rkLift {
		g.rocketAscend(dst, in, oy)
	}
	g.rocketFlight(dst, in, cam)
	g.rocketCaption(dst, in)
	if t < 1 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-t))))
	}
	if t > rkFlash { // a white flash that stays until the cut to the space station
		fillRect(dst, 0, 0, screenW, screenH, rgba(0xf4f8ff, uint8(255*math.Min(1, (t-rkFlash)/0.6))))
	}
	hint := actionHint(actConfirm) + " / " + actionHint(actBack) + ": saltar (dos veces)"
	col := rgba(0x8a86a0, 170)
	if in.skipArm > 0 {
		hint, col = "Otra vez para saltar", rgb(0xffe060)
	}
	g.drawTextC(dst, hint, 480, 524, col, 1)
}

// the roof of the station: the city goes by behind the parapet and the roof scrolls with the walk
func (g *Game) rocketRoof(c *ebiten.Image, in *interState, W float64) {
	fillRect(c, 0, interPanelH, screenW, interGround-interPanelH, rgb(0x0a0e24))
	// water tanks, vents and an aerial of the roof (they move with the roof)
	for k := -1; k < 5; k++ {
		wx := float64(k)*360 - math.Mod(W*0.9, 360) + 60
		switch ((k+int(W*0.9/360))%3 + 3) % 3 {
		case 0: // a water tank on legs
			fillRect(c, wx, interGround-90, 6, 56, rgb(0x2a2a36))
			fillRect(c, wx+54, interGround-90, 6, 56, rgb(0x2a2a36))
			fillRect(c, wx-6, interGround-132, 72, 46, rgb(0x3a3a4c))
			fillRect(c, wx-6, interGround-132, 72, 5, rgb(0x56566c))
			for h := 0; h < 3; h++ {
				fillRect(c, wx-6, interGround-120+float64(h)*14, 72, 2, rgb(0x2a2a3a))
			}
		case 1: // a vent
			fillRect(c, wx+10, interGround-64, 34, 30, rgb(0x4a4a5c))
			fillRect(c, wx+4, interGround-72, 46, 10, rgb(0x5a5a70))
		default: // an aerial
			fillRect(c, wx+26, interGround-150, 4, 116, rgb(0x2a2a36))
			fillRect(c, wx+8, interGround-132, 40, 3, rgb(0x2a2a36))
			fillRect(c, wx+14, interGround-110, 28, 3, rgb(0x2a2a36))
			fillRect(c, wx+23, interGround-156, 10, 6, rgb(0xe03a3a)) // the red light
		}
	}
	// the parapet at the back and the roof
	fillRect(c, 0, interGround-34, screenW, 34, rgb(0x3a3a48))
	fillRect(c, 0, interGround-34, screenW, 5, rgb(0x5a5a6c))
	for x := -math.Mod(W, 110); x < screenW; x += 110 {
		fillRect(c, x, interGround-29, 3, 29, rgb(0x2e2e3a))
	}
	fillRect(c, 0, interGround, screenW, screenH-interGround, rgb(0x26262e))
	fillRect(c, 0, interGround, screenW, 3, rgb(0x3e3e4c))
	for x := -math.Mod(W, 150); x < screenW; x += 150 { // seams of the roofing felt
		fillRect(c, x, interGround+3, 3, screenH-interGround, rgb(0x1c1c24))
	}
	for x := -math.Mod(W, 64); x < screenW; x += 64 { // the safety line along the edge
		fillRect(c, x, interGround+60, 32, 6, rgb(0xe8c030))
		fillRect(c, x+32, interGround+60, 32, 6, rgb(0x14141a))
	}
	// and the leftovers of the parties he has left behind
	for k := 0; k < 30; k++ {
		wx := float64(k)*191 + jpHash(k, 9, 183)*60 - W
		if wx < -20 || wx > screenW+20 {
			continue
		}
		switch k % 3 {
		case 0:
			fillRect(c, wx, interGround+30, 9, 12, rgb(0xd8503a))
		case 1:
			filledEllipse(c, wx, interGround+38, 9, 6, rgb(0xe03a8a))
		default:
			fillRect(c, wx, interGround+28, 4, 14, rgb(0x3aa050))
		}
	}
}

// ---- hero, guard and the rocket ----------------------------------------------------------------------

// rocketHeroPos: where the hero stands, and whether he is already inside the toilet.
func (g *Game) rocketHeroPos(t float64) (x, y float64, inside bool) {
	x, y = interHeroX, interGround+12
	if t > itBoard0 {
		u := introEase((t - itBoard0) / 2.0)
		x = interHeroX + (rkX-120-interHeroX)*u
	}
	if t >= rkBoard {
		u := math.Min(1, (t-rkBoard)/0.9)
		topY := rkBase - 14 - rkTop + 72 // the hero's feet are hidden behind the barrel: he sits in the toilet
		seat := rkX - 4
		if g.inter != nil && g.inter.two {
			seat = rkX - 22 // two seats on the same barrel: the blue one on the left one
		}
		x += (seat - (rkX - 120)) * introEase(u)
		y += (topY - (interGround + 12)) * introEase(u)
		y -= math.Sin(math.Pi*u) * 70
		inside = u >= 1
	}
	return
}

func (g *Game) rocketActors(c *ebiten.Image, in *interState) {
	t := in.t
	h := rocketHeight(t)
	rx := rocketWorldX(t) // the roof brings the rocket in from the right; it stops at rkX
	g.rocketPad(c, rx)
	g.rocketSmokeGround(c, t)
	// the guard
	if t >= rkGuard && t < rkLift+0.5 {
		g.rocketGuard(c, in)
	}
	hx, hy, _ := g.rocketHeroPos(t)
	if t >= rkLift { // see rocketAscend
		return
	}
	heroDraw := func() {
		g.rocketHero(c, in, c, hx, hy-h)
		if in.two {
			g.rocketHero2(c, in, h, 0)
		}
	}
	if t < rkBoard {
		heroDraw()
		g.drawRocketBody(c, in, rx, rkBase-h, nil)
	} else {
		g.drawRocketBody(c, in, rx, rkBase-h, heroDraw)
	}
}

// rocketAscend draws the rocket once it has left: straight on the screen, rising past the sky.
func (g *Game) rocketAscend(dst *ebiten.Image, in *interState, oy float64) {
	t := in.t
	h := rocketHeight(t)
	hx, hy, _ := g.rocketHeroPos(t)
	by := rkBase - h + oy
	g.drawRocketBody(dst, in, rkX, by, func() {
		g.rocketHero(dst, in, dst, hx, hy-h+oy)
		if in.two {
			g.rocketHero2(dst, in, h, oy)
		}
	})
}

func (g *Game) rocketPad(c *ebiten.Image, rkX float64) {
	by := float64(rkBase)
	fillRect(c, rkX-96, by-6, 192, 14, rgb(0x6a4a2c)) // a pallet
	fillRect(c, rkX-96, by-6, 192, 3, rgb(0x8a6a44))
	for x := rkX - 90; x < rkX+90; x += 30 {
		fillRect(c, x, by-6, 3, 14, rgb(0x3a2a1a))
	}
	fillRect(c, rkX-80, by+8, 28, 12, rgb(0x6a6a74)) // cinder blocks
	fillRect(c, rkX+52, by+8, 28, 12, rgb(0x6a6a74))
	fillRect(c, rkX-80, by+8, 28, 2, rgb(0x8a8a96))
	fillRect(c, rkX+52, by+8, 28, 2, rgb(0x8a8a96))
	// scorch marks, from the earlier attempts
	filledEllipse(c, rkX+10, by-2, 70, 5, rgba(0x14100c, 130))
}

// the rocket itself: a rusty barrel, a toilet for a nose, firecrackers for an engine
func (g *Game) drawRocketBody(c *ebiten.Image, in *interState, cx, by float64, hero func()) {
	t := in.t
	x0, y0 := cx-38, by-14-rkTop
	rumble := 0.0
	if t > rkCount1+1.6 && t < rkLift {
		rumble = math.Round(math.Sin(t*70) * 1.5)
	}
	if t >= rkKick+0.45 && t < rkKick+0.9 { // the kick makes it jump
		rumble = math.Round(math.Sin((t-rkKick-0.45)*40) * 3)
	}
	cx += rumble
	x0 += rumble
	// the flame
	fire := 0.0
	switch {
	case t >= rkLift:
		fire = math.Min(1, (t-rkLift)/0.4)
	case t >= rkCount1+1.9: // the fuses are burning
		fire = 0.18 + 0.1*math.Sin(t*40)
	}
	if fire > 0 {
		for i := 0; i < 7; i++ {
			k := float64(i)
			j := math.Sin(t*50+k*1.7) * 3 * fire
			ry := (46 - k*5) * fire
			if ry < 3 {
				break
			}
			col := []uint32{0xff4a14, 0xff7a20, 0xff9a30, 0xffc040, 0xffe070, 0xfff4b0, 0xffffff}[i]
			filledEllipse(c, cx+j, by-6+k*16*fire+ry*0.4, 26-k*3, ry, rgb(col))
		}
	}
	// the fins of cardboard
	for r := 0.0; r < 54; r++ {
		w := 34 * (r / 54)
		fillRect(c, x0-w, by-70+r, w+1, 1, rgb(0xb48a52))
		fillRect(c, x0+76, by-70+r, w+1, 1, rgb(0xb48a52))
		fillRect(c, x0-w, by-70+r, 2, 1, rgb(0x7a5a30))
		fillRect(c, x0+76+w-1, by-70+r, 2, 1, rgb(0x7a5a30))
	}
	// the tank of the toilet, behind
	fillRect(c, cx+14, y0-80, 38, 54, rgb(0xe6e6ee))
	fillRect(c, cx+14, y0-80, 38, 6, rgb(0xc8c8d8))
	fillRect(c, cx+14, y0-26, 38, 4, rgb(0xaaaabc))
	fillRect(c, cx+52, y0-72, 8, 4, rgb(0x8a8a9c)) // the flush handle
	// the barrel (drawn again below the bowl after the hero, so that it covers his legs)
	barrel := func(from float64) {
		fillRect(c, x0, y0+from, 76, rkTop-from, rgb(0x9c3a2a))
		fillRect(c, x0+6, y0+from, 7, rkTop-from, rgb(0xc25a42))
		fillRect(c, x0+62, y0+from, 14, rkTop-from, rgb(0x6e2418))
		for _, hy := range []float64{14, 66, 118} {
			if hy+5 > from {
				top := math.Max(hy, from)
				fillRect(c, x0, y0+top, 76, hy+5-top, rgb(0x7a7e8a))
				if hy >= from {
					fillRect(c, x0, y0+hy, 76, 1, rgb(0xa0a4b0))
				}
			}
		}
		if from < 36 {
			fillRect(c, x0+16, y0+36, 4, 4, rgb(0x5a1a10)) // rust and rivets
		}
		fillRect(c, x0+50, y0+88, 5, 5, rgb(0x5a1a10))
	}
	barrel(0)
	// the bowl: the back of the rim, the hole, the hero inside, the front lip
	filledEllipse(c, cx, y0-6, 46, 20, rgb(0xf2f2f6))
	filledEllipse(c, cx, y0-8, 33, 12, rgb(0x2a2a3a))
	if hero != nil {
		hero()
	}
	filledEllipse(c, cx, y0+2, 46, 12, rgb(0xf2f2f6))
	filledEllipse(c, cx, y0+4, 40, 8, rgb(0xd4d4e0))
	if hero != nil {
		barrel(16)
	}
	// the firecrackers of the engine, tied to the barrel with a string
	for i, off := range []float64{-30, -12, 6, 24} {
		fillRect(c, cx+off, by-76, 14, 60, rgb(0xd02a2a))
		fillRect(c, cx+off, by-76, 14, 5, rgb(0x14141a))
		fillRect(c, cx+off, by-52, 14, 8, rgb(0xf0e8d0))
		fillRect(c, cx+off+6, by-84, 2, 9, rgb(0x14141a)) // the fuse
		if fire > 0 || (t < rkFizzle && t > rkCount0-1) || (t > rkKick+0.9 && t < rkLift) {
			// the spark on the fuse
			if int(t*20+float64(i))%2 == 0 && t < rkLift {
				fillRect(c, cx+off+4, by-90, 6, 6, rgb(0xffd040))
				fillRect(c, cx+off+6, by-92, 2, 10, rgb(0xfff4b0))
			}
		}
	}
	fillRect(c, cx-34, by-60, 66, 4, rgb(0x4a3a20)) // the string
	// the sign
	fillRect(c, cx-68, by-118, 136, 32, rgb(0x3a2a10))
	fillRect(c, cx-66, by-116, 132, 28, rgb(0xe8c030))
	g.drawTextC(c, "CORREOS", cx, by-114, rgb(0x1c1408), 1.1)
	g.drawTextC(c, "ESPACIALES", cx, by-101, rgb(0x1c1408), 1.1)
	fillRect(c, cx-70, by-110, 4, 4, rgb(0x3a2a10)) // the ropes
	fillRect(c, cx+66, by-110, 4, 4, rgb(0x3a2a10))
}

// the hero, at 4x, in the pose it needs
func (g *Game) rocketHero(c *ebiten.Image, in *interState, _ *ebiten.Image, hx, hy float64) {
	t := in.t
	cs := in.hero
	if cs == nil {
		cs = g.as.playerCS
	}
	if cs == nil {
		return
	}
	anim, ph := AnimWalk, g.interPhase(in)
	dir := DirRight
	walking := t < rkBoard && (t < itBoard0 || t < itBoard0+2.0)
	if t >= itBoard0 {
		ph = t * 7
	}
	if !walking || (t >= itBoard0+2.0 && t < rkBoard) {
		anim, ph = AnimIdle, 0
	}
	if t >= rkBoard && t < rkBoard+0.9 {
		anim, ph = AnimWalk, t*10 // the hop
	}
	if t >= rkAsk && t < rkBoard-0.2 {
		anim, ph, dir = AnimIdle, 0, DirLeft // he turns to the guard
	}
	if t >= rkBoard+0.9 {
		anim, ph, dir = AnimIdle, 0, DirDown // sitting in the bowl, facing us
	}
	img, flip := cs.img(anim, dir, ph)
	if img == nil {
		return
	}
	bob := 0.0
	if t < itBoard0 {
		bob = -math.Abs(math.Sin(ph*math.Pi*2/float64(animFrames[AnimWalk]))) * 5
	}
	if t >= rkBoard+0.9 && t < rkFizzle { // he taps his foot, hurried
		bob = -math.Abs(math.Sin(t*6)) * 1.5
	}
	if t >= rkFizzle && t < rkKick { // he droops
		bob = 3
	}
	fillShadow(c, hx, hy)
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(charW, 0)
	}
	op.GeoM.Translate(-charCX, -charBase)
	op.GeoM.Scale(interScale, interScale)
	if t < itBoard0 {
		op.GeoM.Rotate(0.07)
	}
	op.GeoM.Translate(math.Round(hx), math.Round(hy+bob))
	c.DrawImage(img, op)
	if t < itBoard0+0.8 {
		g.interAnger(c, hx, hy+bob, t)
	}
}

// the station guard that wants to see a ticket
func (g *Game) rocketGuard(c *ebiten.Image, in *interState) {
	t := in.t
	if in.guard == nil {
		sp := guardSpec(rand.New(rand.NewSource(77)))
		in.guard = newCharSet(&sp)
	}
	walk := introEase((t - rkGuard) / 1.6)
	x := -80 + (rkX-215+80)*walk // he comes after the hero, from behind
	anim, ph := AnimIdle, 0.0
	if walk < 1 {
		anim, ph = AnimWalk, t*6
	}
	if t >= rkKick-0.9 && t < rkKick+0.6 { // goes to the barrel for the kick
		u := introEase((t - (rkKick - 0.9)) / 0.9)
		x = rkX - 215 + (rkX-86-(rkX-215))*u
		anim, ph = AnimWalk, t*7
		if t >= rkKick {
			x = rkX - 86
			anim, ph = AnimPunch, (t-rkKick)*12
		}
	}
	if t >= rkKick+0.6 { // and steps back
		u := introEase((t - rkKick - 0.6) / 0.9)
		x = rkX - 86 - 150*u
		anim, ph = AnimWalk, t*6
		if u >= 1 {
			anim, ph = AnimIdle, 0
		}
	}
	img, flip := in.guard.img(anim, DirRight, ph)
	if img == nil {
		return
	}
	// he faces the rocket (he comes from the right: left), or looks at it from the left
	faceLeft := t >= rkKick+0.6 && introEase((t-rkKick-0.6)/0.9) < 1 // stepping back, he walks to the left
	op := &ebiten.DrawImageOptions{}
	f := flip
	if faceLeft {
		f = !flip
	}
	if f {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(charW, 0)
	}
	op.GeoM.Translate(-charCX, -charBase)
	op.GeoM.Scale(interScale, interScale)
	y := interGround + 14.0
	fillShadow(c, x, y)
	op.GeoM.Translate(math.Round(x), math.Round(y))
	c.DrawImage(img, op)
	in.guardX = x
}

// the smoke of the failed ignition and of the lift off, along the roof
func (g *Game) rocketSmokeGround(c *ebiten.Image, t float64) {
	by := float64(rkBase)
	add := func(t0, dur float64, n int, big float64) {
		if t < t0 || t > t0+dur {
			return
		}
		u := (t - t0) / dur
		for i := 0; i < n; i++ {
			side := -1.0
			if i%2 == 0 {
				side = 1
			}
			r := (10 + jpHash(i, 3, 301)*16) * (0.5 + u*big)
			px := rkX + side*(20+u*(60+jpHash(i, 5, 303)*90)*big)
			py := by + 4 - u*(10+jpHash(i, 7, 305)*30) - float64(i%3)*3
			filledEllipse(c, px, py, r*1.3, r, rgba(0xb4b4c4, uint8(190*(1-u))))
		}
	}
	add(rkFizzle, 1.6, 8, 1.0)     // pffft
	add(rkLift, 3.5, 20, 2.2)      // the take off
	add(rkCount1+1.9, 0.9, 5, 0.5) // the fuses
}

// the sky above: from the night of the city to the black of space, with a few big stars of real shape
func (g *Game) rocketSpace(dst *ebiten.Image, cam, t float64) {
	for y := 0.0; y < screenH; y += 4 {
		alt := cam + (screenH - y) // how high above the roof this row is
		k := math.Max(0, math.Min(1, (alt-200)/1500))
		fillRect(dst, 0, y, screenW, 4, mixc(rgb(0x070a1c), rgb(0x02030a), k))
	}
	// a planet with a ring, far above
	pa := 1900.0
	py := screenH - (pa - cam*0.55)
	if py > -140 && py < screenH+140 {
		filledEllipse(dst, 760, py, 80, 80, rgb(0xd8905a))
		filledEllipse(dst, 790, py-12, 60, 60, rgb(0xe8a870))
		for i := 0; i < 4; i++ {
			filledEllipse(dst, 760+float64(i)*3, py+10+float64(i)*14, 70, 10, rgba(0xb8683a, 150))
		}
		filledEllipse(dst, 760, py, 118, 18, rgba(0xf0d8a0, 180))
		filledEllipse(dst, 760, py+2, 96, 11, rgba(0x14102a, 120))
	}
	// stars: few, big, with shape and a clear movement downwards (they rise past us)
	for i := 0; i < 16; i++ {
		x := 40 + jpHash(i, 1, 401)*880
		wy := 650 + jpHash(i, 2, 403)*3200
		par := 0.45 + jpHash(i, 3, 405)*0.4
		sy := screenH - (wy - cam*par)
		if sy < -40 || sy > screenH+40 || wy < cam*par+screenH*0.15 && false {
			continue
		}
		alt := cam + (screenH - sy)
		vis := math.Min(1, math.Max(0, (alt-420)/500))
		if vis <= 0 {
			continue
		}
		r := 7 + jpHash(i, 4, 407)*11
		pulse := 1 + 0.18*math.Sin(t*3+float64(i))
		g.drawStar(dst, x, sy, r*pulse, vis, i)
	}
}

// drawStar paints a four pointed star with a soft glow
func (g *Game) drawStar(dst *ebiten.Image, x, y, r, a float64, i int) {
	cols := []uint32{0xfff4b0, 0xbfe0ff, 0xffd0e8, 0xd8ffd0}
	col := cols[i%len(cols)]
	filledEllipse(dst, x, y, r*1.7, r*1.7, rgba(col, uint8(34*a)))
	filledEllipse(dst, x, y, r*1.05, r*1.05, rgba(col, uint8(50*a)))
	for k := 0.0; k <= r; k++ { // the four points: tall narrow diamonds
		w := math.Max(1, (r-k)/r*r*0.28)
		fillRect(dst, x-w, y-k, w*2, 1, rgba(col, uint8(255*a)))
		fillRect(dst, x-w, y+k, w*2, 1, rgba(col, uint8(255*a)))
		fillRect(dst, x-k, y-w, 1, w*2, rgba(col, uint8(255*a)))
		fillRect(dst, x+k, y-w, 1, w*2, rgba(col, uint8(255*a)))
	}
	fillRect(dst, x-r*0.28, y-r*0.28, r*0.56, r*0.56, rgba(0xffffff, uint8(255*a)))
}

// what flies: the rocket itself once the camera follows it, over the space (the world was drawn shifted)
func (g *Game) rocketFlight(dst *ebiten.Image, in *interState, cam float64) {
	t := in.t
	if t < rkLift+0.001 || cam <= 0 {
		return
	}
	// a trail of smoke and sparks behind it, in the sky
	h := rocketHeight(t)
	sy := rkBase - h + cam // where it is on the screen
	for i := 0; i < 16; i++ {
		age := float64(i) / 16
		py := sy + 10 + age*260
		r := 14 + age*22
		a := uint8(150 * (1 - age))
		filledEllipse(dst, rkX+math.Sin(t*30+float64(i)*2)*6*age, py, r, r*0.9, rgba(0xc8c8d8, a))
	}
}

// ---- captions and speech -------------------------------------------------------------------------------

var rocketMutters = []string{"Otra fiesta más...", "¿Dónde dejé el cohete?", "Esto es de locos.", "Y nadie me da las gracias."}

func (g *Game) rocketCaption(dst *ebiten.Image, in *interState) {
	t := in.t
	var line string
	switch {
	case t < 2:
	case t < itDecel0:
		line = "La última fiesta queda lejos. Muy, muy lejos."
	case t < rkGuard:
		line = "Para llegar hasta allí hace falta un cohete."
	case t < rkTicket:
		line = "Un cohete casero, todo hay que decirlo."
	case t < rkBoard:
		line = "Cuchillo válido como billete. Qué tiempos."
	case t < rkCount0:
		line = "Combustible: petardos. Piloto: cabreado."
	case t < rkFizzle:
		line = "Cuenta atrás..."
	case t < rkKick:
		line = "...¿Por qué siempre falla algo?"
	case t < rkCount1:
		line = "Un buen patadón y a otra cosa."
	case t < rkLift:
		line = "Segundo intento. Que nadie respire."
	case t < rkSpace+1:
		line = "¡¡Todos a sus puestos!!"
	case t < rkFlash:
		line = "Rumbo a la última fiesta."
	}
	if line != "" {
		w := g.textW(tr(line), 1.5) + 40
		fillRect(dst, 480-w/2, 488, w, 32, rgba(0x000000, 170))
		g.drawTextC(dst, line, 480, 495, rgb(0xf4f0ff), 1.5)
	}
	cam := rocketCam(t)
	// the countdown, big, over the sky
	count := func(t0, tick float64, from int) (string, bool) {
		if t < t0 {
			return "", false
		}
		i := int((t - t0) / tick)
		if i >= from {
			return "", false
		}
		return fmt.Sprint(from - i), true
	}
	if s, ok := count(rkCount0, rkTick0, 5); ok && t < rkFizzle {
		g.rocketNumber(dst, s, (t-rkCount0)/rkTick0)
	}
	if s, ok := count(rkCount1, rkTick1, 3); ok && t < rkLift {
		g.rocketNumber(dst, s, (t-rkCount1)/rkTick1)
	}
	if t >= rkLift && t < rkLift+1.2 {
		g.drawTextC(dst, "¡DESPEGUE!", 480, 110+cam*0, rgba(0xff6a8a, uint8(255*(1-(t-rkLift)/1.2))), 5)
	}
	if t >= rkFizzle && t < rkFizzle+1.5 {
		g.drawTextC(dst, "pffffft...", rkX, 150, rgba(0xc8c8d8, uint8(255*(1-(t-rkFizzle)/1.5))), 2.6)
	}
	// the talk
	say := func(x, y float64, s string) { g.speechBubble(dst, x, y, s) }
	if in.two {
		g.rocketTwoSay(dst, in)
	}
	hx, hy, _ := g.rocketHeroPos(t)
	h := rocketHeight(t)
	switch {
	case t > 2 && t < itBoard0:
		const per = 3.8
		k := int((t - 2) / per)
		if u := t - 2 - float64(k)*per; u < 2.4 {
			say(hx+10, hy-34*interScale-4, rocketMutters[k%len(rocketMutters)])
		}
	case t >= rkAsk && t < rkTicket-0.2:
		say(in.guardX+10, interGround+14-34*interScale-4, "¡Su billete, por favor!")
	case t >= rkTicket-0.2 && t < rkTicket+0.9:
		g.drawKnifeFlash(dst, hx+30, hy-70)
	case t >= rkTicket+0.9 && t < rkBoard:
		say(in.guardX+10, interGround+14-34*interScale-4, "¡Válido! Pase usted.")
	case t >= rkBoard+1.2 && t < rkBoard+2.6:
		say(in.guardX+10, interGround+14-34*interScale-4, "¡Buen viaje!")
	case t >= rkFizzle+0.6 && t < rkKick-0.3:
		say(hx+10, hy-h+cam-34*interScale+70, "¿En serio?")
	case t >= rkKick-0.9 && t < rkKick+0.5:
		say(in.guardX+10, interGround+14-34*interScale-4, "Déjame a mí...")
	}
}

func (g *Game) drawKnifeFlash(dst *ebiten.Image, x, y float64) {
	g.drawText(dst, "*", x, y, rgb(0xffffff), 3)
}

// rocketNumber: a countdown number that pops and fades
func (g *Game) rocketNumber(dst *ebiten.Image, s string, u float64) {
	u = math.Mod(u, 1)
	k := 1 + 0.5*math.Exp(-u*8)
	a := uint8(255 * math.Min(1, 1.6*(1-u)+0.2))
	col := rgba(0xffe060, a)
	if s == "1" {
		col = rgba(0xff6a8a, a)
	}
	g.drawTextC(dst, s, 482, 104, rgba(0x14102a, a), 8*k)
	g.drawTextC(dst, s, 480, 100, col, 8*k)
}

var _ = color.RGBA{}
