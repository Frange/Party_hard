package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The intro: shown once, before the very first party of a new game.
//
// The hero tries to sleep (3:47 a.m.), the neighbours' bass shakes the glass of
// water on the bedside table, he counts sheep, the fourth one wears sunglasses
// and a party hat, the cloud pops, he gets up, takes his knife and goes out
// to put an end to every party. Same cinematic style as the ending: the
// bedroom of the main menu seen with a camera that zooms in on him.
// ---------------------------------------------------------------------------

type introState struct {
	t         float64
	skipArm   float64 // seconds left in which a second press skips the whole thing
	canvas    *ebiten.Image
	thump     bool            // the music of the neighbours' party has been started
	song      bool            // ...and it is a song of the intros folder, not the synthesised loop
	did       map[string]bool // one-shot sounds already played
	cam       introCam
	heroKnife *CharSet // the hero with the knife in his hand (the real held-item drawing)
	two       bool     // the two player version (intro2p.go)
	heroes    [2]*CharSet
	heroesK   [2]*CharSet // ...and the same two with a knife in the hand
}

type introCam struct{ z, ox, oy float64 } // the transform of the last frame (for the speech bubble)

const introLen = 41.5

// the timeline, in seconds
const (
	inGlassSlide = 3.75 // the glass starts to creep towards the edge of the table
	inGlassFall  = 6.25 // and falls
	inGlassLand  = 6.7  // and breaks on the floor
	inSheepStart = 9.5  // the cloud shows up
	inBurst      = 19.0 // the cloud pops
	inStandUp    = 19.4 // he is out of bed
	inKnife      = 23.0 // he takes the knife
	inWalk0      = 24.5 // walks to the door
	inWalk1      = 29.5 // arrives
	inFadeOut    = 31.4 // the room fades to black
	inCard       = 32.8 // title card
)

func (g *Game) startIntro() {
	g.intro = &introState{did: map[string]bool{}}
	g.mode = ModeIntro
	g.sceneReplay = false
	g.snd.StopMusic()
	g.bedQuiet, g.bedEmpty, g.bedNoTalk = false, false, false
	g.markSeen("seen_intro")
	g.intro.heroKnife = g.heroWithKnife()
	if g.scene2P() {
		in := g.intro
		in.two = true
		for i := 0; i < 2; i++ {
			in.heroes[i] = g.sceneHero(i, false)
			in.heroesK[i] = g.sceneHero(i, true)
		}
	}
}

// heroWithKnife: the player\'s own look carrying a knife, so the knife is drawn in his hand by the
// normal held-item code in every pose and view.
func (g *Game) heroWithKnife() *CharSet {
	if g.as.playerCS == nil {
		return nil
	}
	return knifeHeroSet(g.as.playerCS)
}

// knifeHeroSet copies a CharSet making it carry a knife. The player is normally a painted rig (the
// character of the editor): its own carry setting is changed, never a blank spec (that would draw nothing).
func knifeHeroSet(src *CharSet) *CharSet {
	cs := *src
	cs.cache = map[int]*ebiten.Image{}
	switch {
	case cs.rig != nil:
		r := *cs.rig
		r.carry = HoldKnife
		cs.rig = &r
	case cs.Spec != nil:
		sp := *cs.Spec
		sp.Carry = HoldKnife
		cs.Spec = &sp
	}
	return &cs
}

// finishIntro: a real first game goes on to the first party, a replay goes back to where it came from.
func (g *Game) finishIntro() {
	g.intro = nil
	g.snd.StopThump()
	g.snd.StopScene()
	resetThumpBeat()
	g.bedEmpty, g.bedNoTalk = false, false
	if g.sceneReplay {
		g.leaveScene()
		return
	}
	if g.introBoot {
		g.leaveBootIntro()
		return
	}
	g.startLevel(0)
}

func (g *Game) updateIntro() {
	in := g.intro
	if in == nil {
		g.leaveScene()
		return
	}
	const dt = 1.0 / 60
	in.t += dt
	g.time = in.t
	if in.skipArm > 0 {
		in.skipArm -= dt
	}
	if actBack.just() || actConfirm.just() || ebitenClick() {
		if in.skipArm > 0 {
			g.finishIntro()
			return
		}
		in.skipArm = 2.2
	}
	g.bedEmpty = in.t >= inStandUp
	g.bedNoTalk = in.t >= inGlassFall-0.2
	// the sounds of the story
	once := func(key string, at float64, name string) {
		if in.t >= at && !in.did[key] {
			in.did[key] = true
			g.snd.Play(name)
		}
	}
	// the neighbours' bass: one loop, one beat clock for the sound and the picture (see thumpPhase)
	if !in.thump {
		in.thump = true
		in.song = g.introMusicStart()
	}
	if in.song {
		g.snd.SetSceneVolume("base", introThumpVol(in.t))
	} else {
		g.snd.SetThumpVolume(introThumpVol(in.t))
	}
	for k := 0; k < 4; k++ {
		once("sheep"+string(rune('0'+k)), inSheepStart+1.5+float64(k)*2+0.9, "spark")
	}
	once("burst", inBurst, "boom")
	if in.two {
		once("catch", inGlassLand, "spark") // caught in mid air
		once("glass", i2GlassBreak, "glass")
		once("slipper", i2SlipHit, "thud")
		once("slipperBack", i2SlipBack, "thud")
		once("shove", i2Run, "thud")
	} else {
		once("glass", inGlassLand, "glass")
	}
	once("up", inStandUp, "alert")
	once("knife", inKnife, "stab")
	once("door", inWalk1, "thud")
	once("doorClose", inWalk1+2.0, "thud")
	once("card", inCard+0.3, "win")
	if in.t >= introLen {
		g.finishIntro()
	}
}

// beatKick: 1 on the beat, falling off quickly (the picture side of the kick drum of the loop).
func beatKick(t float64) float64 { return thumpKick(t) }

// introThumpVol: how loud the party next door is. It swells while he cannot sleep, drops when the
// cloud pops, jumps when he opens the door onto the stairwell and is cut off when it closes behind him.
func introThumpVol(t float64) float64 {
	switch {
	case t < inBurst:
		return 0.30 + 0.70*math.Pow(t/inBurst, 1.2)
	case t < inBurst+0.6:
		return 0.15
	case t < inWalk1:
		return 0.15 + 0.35*introEase((t-inBurst-0.6)/2)
	case t < inWalk1+1.7: // the door is open: the stairwell, the party at full blast
		return 0.5 + 0.45*introEase((t-inWalk1)/0.5)
	case t < inWalk1+2.4: // and it closes
		return 0.95 - 0.83*introEase((t-inWalk1-1.7)/0.5)
	case t < inCard:
		return 0.12 * (1 - introEase((t-inWalk1-2.4)/(inCard-inWalk1-2.4)))
	}
	return 0
}

// introPump: the kick of the beat times how loud it is: the amount the room shakes this frame.
func introPump(t float64) float64 { return beatKick(t) * introThumpVol(t) }

// beatN: the number of the kick drum beat at scene time t (it changes exactly on the kick).
func beatN(t float64) int {
	n, _ := thumpPhase(t)
	return n
}

func introEase(u float64) float64 {
	u = math.Max(0, math.Min(1, u))
	return u * u * (3 - 2*u)
}

// heroPos: where he is while he walks to the door, and how big he is drawn (the room has depth).
func introHeroPos(t float64) (x, y, s float64) {
	x, y, s = 250, 432, 4
	if t > inWalk0 {
		u := introEase((t - inWalk0) / (inWalk1 - inWalk0))
		x = 250 + (72-250)*u
		y = 432 + (318-432)*u
		s = 4 + (3-4)*u
	}
	return
}

func (g *Game) drawIntro(dst *ebiten.Image) {
	in := g.intro
	if in == nil {
		return
	}
	if in.canvas == nil {
		in.canvas = ebiten.NewImage(screenW, screenH)
	}
	dst.Fill(rgb(0x05060c))
	t := in.t
	if t < inCard {
		g.introRoom(in.canvas, in)
		// camera: the whole room, then a close up on the hero while he gets up and leaves
		hx, hy, _ := introHeroPos(t)
		if in.two {
			hx, hy = introTwoCam(t)
		}
		z, fx, fy := 1.0, 480.0, 270.0
		if t >= inBurst-0.4 {
			k := introEase((t - (inBurst - 0.4)) / 1.8)
			if t > inWalk1+0.5 {
				k *= 1 - introEase((t-(inWalk1+0.5))/1.2)
			}
			z = 1 + 0.75*k
			fx, fy = 480+(hx-480)*k, 270+(hy-90-270)*k
		}
		ox, oy := 480-fx*z, 270-fy*z
		ox = math.Max(screenW-screenW*z, math.Min(0, ox))
		oy = math.Max(screenH-screenH*z, math.Min(0, oy))
		{ // the bass shakes the whole room
			k := introPump(t)
			ox += math.Sin(t*60) * 2.2 * k
			oy += math.Cos(t*47) * 2.6 * k
		}
		in.cam = introCam{z, ox, oy}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(z, z)
		op.GeoM.Translate(ox, oy)
		dst.DrawImage(in.canvas, op)
		g.introOverlay(dst, in)
		if t > inFadeOut {
			fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*introEase((t-inFadeOut)/1.4))))
		}
		if t < 1 {
			fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-t))))
		}
	} else {
		g.introCard(dst, in)
	}
	hint := actionHint(actConfirm) + " / " + actionHint(actBack) + ": saltar (dos veces)"
	col := rgba(0x8a86a0, 170)
	if in.skipArm > 0 {
		hint, col = "Otra vez para saltar la intro", rgb(0xffe060)
	}
	g.drawTextC(dst, hint, 480, 518, col, 1)
}

// introRoom paints the bedroom (the menu's) plus everything that happens in it.
func (g *Game) introRoom(c *ebiten.Image, in *introState) {
	t := in.t
	g.time = t
	c.Fill(rgb(0x08060c))
	bedPumpOn, bedPump, bedBeatN = true, introPump(t), beatN(t) // the speakers swell with the kick
	if in.two {
		bedTwo, bedPoses = true, [2]bedPose{introTwoBed(0, t), introTwoBed(1, t)}
		g.drawBedroom(c)
		bedTwo, bedPoses = false, [2]bedPose{}
		bedPumpOn, bedPump = false, 0
		g.introRoomTwo(c, in)
		return
	}
	g.drawBedroom(c)
	bedPumpOn, bedPump = false, 0
	g.introDoor(c, t)
	g.introGlass(c, t)
	if t >= inStandUp {
		g.introHero(c, t)
	}
	if t >= inSheepStart && t < inBurst+2.6 { // the dream bubble goes over everything: door, furniture, hero
		g.introCloud(c, t-inSheepStart)
	}
	if t >= inBurst && t < inBurst+0.35 { // the flash
		fillRect(c, 0, 0, screenW, screenH, rgba(0xffffff, uint8(230*(1-(t-inBurst)/0.35))))
	}
}

// the glasses: two cups on the window sill only tremble; the one on the bedside table creeps to the
// edge on the beats, falls and breaks. After that, only the bits are left on the floor.
func (g *Game) introGlass(c *ebiten.Image, t float64) {
	k := introPump(t)
	// the cups on the sill
	for i, x := range []float64{478, 536} {
		jx := math.Round(math.Sin(t*55+float64(i)) * 1.2 * k)
		jy := -math.Round(k * 1.2)
		fillRect(c, x+jx, 195+jy, 8, 12, rgba(0xbfe0f8, 140))
		fillRect(c, x+jx+1, 199+jy, 6, 7, rgba(0xe0a050, 200))
		fillRect(c, x+jx, 195+jy, 8, 1, rgb(0xeaf6ff))
	}
	const gx0, gy0 = 282.0, 316.0
	if t < inGlassFall {
		slide := 0.0
		if t > inGlassSlide {
			slide = -3.4 * (t - inGlassSlide) / (inGlassFall - inGlassSlide)
		}
		jump := math.Round(k * 1.6)
		g.drawGlass(c, gx0+slide, gy0-jump, 0, 1)
		if k > 0.45 { // ripples and a "tin"
			word := "tin"
			if int(t/thumpBeat)%2 == 1 {
				word = "tin tin"
			}
			g.drawText(c, word, gx0-16, gy0-16-jump, rgba(0xeaf6ff, uint8(255*k)), 0.9)
		}
		return
	}
	if t < inGlassLand { // the fall
		u := (t - inGlassFall) / (inGlassLand - inGlassFall)
		x := gx0 - 3.4 - 18*u
		y := gy0 + 100*u*u
		g.drawGlass(c, x, y, u*3.2, 1)
		return
	}
	// the bits on the floor: they hop away from the impact and stay there
	u := math.Min(1, (t-inGlassLand)/0.5)
	for i := 0; i < 11; i++ {
		ang := -math.Pi + jpHash(i, 7, 61)*math.Pi
		sp := 14 + jpHash(i, 9, 63)*26
		x := 262 + math.Cos(ang)*sp*u
		y := 414 - math.Abs(math.Sin(ang))*(10+jpHash(i, 4, 65)*12)*math.Sin(math.Pi*u) + jpHash(i, 6, 67)*5
		w := 2 + float64(i%3)
		col := rgba(0xbfe0f8, 210)
		if i%4 == 0 {
			col = rgba(0x5a9ae0, 220)
		}
		fillRect(c, math.Round(x), math.Round(y), w, 2, col)
	}
	if t < inGlassLand+0.8 { // a splash of water and the "CRASH"
		g.drawTextC(c, "¡CRASH!", 262, 366-(t-inGlassLand)*30, rgba(0xff6a8a, uint8(255*(1-(t-inGlassLand)/0.8))), 2)
	}
}

// drawGlass: the tumbler (rot in radians around its middle).
func (g *Game) drawGlass(c *ebiten.Image, x, y, rot, _ float64) {
	im := ebiten.NewImage(9, 14)
	fillRect(im, 0, 0, 9, 14, rgba(0xbfe0f8, 150))
	fillRect(im, 1, 5, 7, 8, rgba(0x5a9ae0, 200))
	fillRect(im, 0, 0, 9, 1, rgb(0xeaf6ff))
	fillRect(im, 1, 13, 7, 1, rgb(0x8ab8d8))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-4.5, -7)
	op.GeoM.Rotate(rot)
	op.GeoM.Translate(x+4.5, y+7)
	c.DrawImage(im, op)
	im.Deallocate()
}

// ---- the cloud of sheep ------------------------------------------------------------------

func (g *Game) introCloud(c *ebiten.Image, u float64) {
	cx, cy := 252.0, 184.0         // clear of the door on the left
	if u >= inBurst-inSheepStart { // popped: confetti
		g.introConfetti(c, cx, cy, u-(inBurst-inSheepStart))
		return
	}
	a := uint8(255 * math.Min(1, u/1.2))
	al := func(col color.RGBA) color.RGBA { col.A = a; return col }
	pulse := 0.0
	if u > 7.2 { // the last sheep makes the cloud throb with the bass
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
	if u > 7.2 { // the sky starts flashing like a disco
		sky = al(rgb(partyColors[int(u*8)%len(partyColors)]))
	}
	fillRect(c, cx-104, cy-44, 208, 92, sky)
	fillRect(c, cx-104, cy+20, 208, 28, al(rgb(0x6ab46a)))
	for x := cx - 36; x <= cx+36; x += 18 {
		fillRect(c, x, cy-4, 3, 30, al(rgb(0x8a6a44)))
	}
	fillRect(c, cx-40, cy+4, 80, 3, al(rgb(0x8a6a44)))
	fillRect(c, cx-40, cy+14, 80, 3, al(rgb(0x8a6a44)))
	// the sheep, one after another; the fourth one is the problem
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
		txt = ""
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

// the burst: confetti flying out of the place where the cloud was
func (g *Game) introConfetti(c *ebiten.Image, cx, cy, s float64) {
	for i := 0; i < 70; i++ {
		ang := jpHash(i, 3, 41) * 2 * math.Pi
		sp := 60 + jpHash(i, 5, 43)*210
		x := cx + math.Cos(ang)*sp*s
		y := cy + math.Sin(ang)*sp*s*0.8 + 160*s*s
		a := uint8(255 * math.Max(0, 1-s/2.4))
		col := rgba(partyColors[i%len(partyColors)], a)
		fillRect(c, x, y, 3, 3, col)
	}
	if s < 1 { // a comic "PAF!"
		g.drawTextC(c, "¡PAF!", cx, cy-20-s*20, rgba(0xff6a8a, uint8(255*(1-s))), 3.4)
	}
}

// ---- the door and the hero ---------------------------------------------------------------

// the door of the room, in the left wall. Its height is about 1.35 times the hero's at that depth. It is
// there from the first frame and rattles with the neighbours' bass; at the end he really opens it,
// walks through and it swings shut behind him.
const doorX, doorBase, doorW, doorH = 34.0, 302.0, 62.0, 132.0

// doorOpen: 0 closed, 1 wide open.
func doorOpen(t float64) float64 {
	o := introEase((t - inWalk1) / 0.7)
	if t > inWalk1+1.7 {
		o = 1 - introEase((t-inWalk1-1.7)/0.5)
	}
	return o
}

func (g *Game) introDoor(c *ebiten.Image, t float64) {
	k := introPump(t)
	jx, jy := math.Sin(t*70)*1.6*k, math.Cos(t*53)*1.2*k
	x, y := math.Round(doorX+jx), math.Round(doorBase-doorH+jy)
	o := doorOpen(t)
	fillRect(c, x-6, y-6, doorW+12, doorH+6, rgb(0x2a1e18)) // frame
	fillRect(c, x-6, y-6, doorW+12, 2, rgb(0x4a3528))
	if o > 0 { // what is behind: the lit stairwell
		fillRect(c, x, y, doorW, doorH, rgb(0xf0c870))
		fillRect(c, x, y, doorW, 10, rgb(0xd8a850))
		g.glowAt(c, Vec{x + doorW/2, y + doorH/2}, 70, color.RGBA{255, 214, 120, uint8(120 * o)})
	}
	leaf := doorW * (1 - 0.86*o) // the leaf swings away from us: it seems to narrow
	if leaf > 1 {
		fillRect(c, x, y, leaf, doorH, rgb(0x4a3322))
		fillRect(c, x+leaf*0.14, y+8, leaf*0.72, 48, rgb(0x3a281a))
		fillRect(c, x+leaf*0.14, y+66, leaf*0.72, 58, rgb(0x3a281a))
		fillRect(c, x+leaf-3, y, 3, doorH, rgb(0x2a1e14)) // the free edge
		kn := math.Round(math.Sin(t*80) * 1.4 * k)        // the handle rattles
		fillRect(c, x+leaf*0.86-3, y+doorH*0.55+kn, 5, 5, rgb(0xe8c060))
	}
	if vol := introThumpVol(t); vol > 0.05 { // dust shaken loose from the frame, one puff per kick
		_, f := thumpPhase(t)
		ph := f * thumpBeat
		if ph < 0.4 {
			for i := 0; i < 7; i++ {
				dx := x + 3 + jpHash(i, 2, 71)*(doorW-6)
				dy := y - 4 + ph*70*(0.6+0.4*jpHash(i, 3, 73))
				fillRect(c, math.Round(dx), math.Round(dy), 1, 2, rgba(0xc8c0b0, uint8(200*vol*(1-ph/0.4))))
			}
		}
	}
}

func (g *Game) introHero(c *ebiten.Image, t float64) {
	cs := g.as.playerCS
	if t >= inKnife && g.intro != nil && g.intro.heroKnife != nil {
		cs = g.intro.heroKnife // the knife is in his hand, drawn with the character
	}
	if cs == nil {
		return
	}
	x, y, s := introHeroPos(t)
	anim, ph := AnimIdle, 0.0
	dir := DirDown
	walking := t > inWalk0 && t < inWalk1+0.5
	if walking {
		anim, ph = AnimWalk, t*5
		dir = DirLeft
		if t > inWalk0+(inWalk1-inWalk0)*0.6 {
			dir = DirUp
		}
	}
	// he jumps up from the bed with a little hop
	hop := 0.0
	if t < inStandUp+0.5 {
		hop = math.Sin((t-inStandUp)/0.5*math.Pi) * 26
	}
	img, flip := cs.img(anim, dir, ph)
	if img == nil {
		return
	}
	alpha := 1.0
	if t > inWalk1+0.5 { // goes through the door
		alpha = 1 - introEase((t-inWalk1-0.5)/1.0)
	}
	fillShadow(c, x, y)
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(charW, 0)
	}
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(math.Round(x-charCX*s), math.Round(y-hop-charBase*s))
	op.ColorScale.ScaleAlpha(float32(alpha))
	c.DrawImage(img, op)
	if t >= inKnife && t < inKnife+0.5 { // a glint on the blade
		g.drawText(c, "*", x+8*s, y-hop-26*s, rgba(0xffffff, uint8(255*(1-(t-inKnife)/0.5))), 2)
	}
}

// ---- captions and speech -----------------------------------------------------------------------

func (g *Game) introOverlay(dst *ebiten.Image, in *introState) {
	if in.two {
		g.introOverlayTwo(dst, in)
		return
	}
	t := in.t
	var line string
	switch {
	case t < 3.5:
		line = "3:47 de la madrugada. Otra noche sin pegar ojo."
	case t < inGlassFall:
		line = "El bajo del vecino hace temblar hasta el vaso de agua."
	case t < inGlassLand+0.5:
		line = "El vaso... se... ¡cae!"
	case t < inSheepStart:
		line = "Genial. Ahora también me toca barrer."
	case t < 12:
		line = "Vale. Ovejitas. Eso siempre funciona."
	case t < 17:
		line = "Una oveja... dos ovejas... tres..."
	case t < inBurst:
		line = "¿Esa oveja lleva gafas de sol y gorro de fiesta?"
	case t < 20.5:
		line = "¡PAF! Hasta las ovejas se han ido de fiesta."
	case t < inKnife+1:
		line = "Se acabó. Hay una sola forma de que esta ciudad duerma."
	case t < inFadeOut-1:
		line = "Cuchillo en mano, a poner fin a las fiestas."
	default:
		line = "Una fiesta menos. Y otra. Y otra..."
	}
	g.endCaption(dst, line)
	if t >= inGlassLand+0.35 && t < inSheepStart-0.2 && !g.bedEmpty { // he looks at the bits, fed up
		z := in.cam
		g.speechBubble(dst, z.ox+g.talkX*z.z, z.oy+(g.talkY-48)*z.z, "-_-  Mi vaso...")
	}
	if t >= inStandUp && t < inKnife { // his first words
		hx, hy, s := introHeroPos(t)
		z := in.cam
		g.speechBubble(dst, z.ox+hx*z.z, z.oy+(hy-44*s)*z.z-6, "¡¡YA ESTÁ BIEN!!")
	}
}

func (g *Game) introCard(dst *ebiten.Image, in *introState) {
	t := in.t - inCard
	dst.Fill(rgb(0x04050c))
	k := math.Min(1, t/1.2)
	g.drawTextC(dst, "NADIE TE DEJÓ DORMIR.", 480, 190, rgba(0xf4f0ff, uint8(255*k)), 3.2)
	if t > 2.2 {
		k2 := math.Min(1, (t-2.2)/1.2)
		g.drawTextC(dst, "AHORA NADIE SALDRÁ DE LA FIESTA.", 480, 250, rgba(0xff6a8a, uint8(255*k2)), 3.2)
	}
	if t > 4.6 {
		g.drawLogo(dst, 480, 320)
	}
}
