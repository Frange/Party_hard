package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// The ending.
//
// Beating the 21st party (the space station) closes the story:
//   - without every trophy: the hero walks home through the quiet city (the
//     camera zooms on him), gets into bed, counts sheep and falls asleep; then the
//     credits, a big "¿ FINAL ?" and "Gracias por jugar".
//   - with every trophy: the night is not over. An invitation arrives and the
//     last party is on the island of the first film (jurassic.go). When that one is
//     over, the same story, but a real "FIN".
// The DEBUG menu can start either ending and the island at any time.
// ---------------------------------------------------------------------------

const (
	endInvite = iota // the card that sends you to the island
	endWalk          // the street, zoomed on the hero
	endDoor
	endBed // counting sheep
	endCredits
	endStinger // only the secret ending: the scene after the credits
	endFinal   // "¿ FINAL ?" / "FIN"
	endArrival // only after the letter: the helicopter ride to the island (arrival.go)
	endPrelude // the action in the last scene before the letter and the two endings (prelude.go)
	endParty   // only the secret ending: the big party on the island (endparty.go)
)

type endState struct {
	stage    int
	t        float64 // time in the stage
	total    float64
	trueEnd  bool
	canvas   *ebiten.Image
	bed      *ebiten.Image
	heroX    float64
	skipLock float64
	music    string   // the tune of this ending that is playing ("" = none yet)
	x        endExtra // what the invitation and the two endings keep (ending2.go, invite.go)
}

const (
	endCW, endCH = 480, 270
	endGround    = 210.0
)

func (g *Game) startEnding(trueEnd bool) {
	g.end = &endState{stage: endWalk, trueEnd: trueEnd, heroX: 980}
	g.mode = ModeEnding
	g.sceneReplay = false
	if trueEnd {
		g.markSeen("seen_end2")
	} else {
		g.markSeen("seen_end1")
	}
	g.snd.StopMusic()
	g.bedQuiet = false
}

// startEndingPre / startInvitePre: the story as it is played: the action of the last scene comes first
// (prelude.go), then the ending or the letter. startEnding / startInvite are the ending itself.
func (g *Game) startEndingPre(trueEnd bool) {
	g.startEnding(trueEnd)
	if trueEnd {
		g.enterPrelude(preIsland)
	} else {
		g.enterPrelude(preBad)
	}
}

func (g *Game) startInvitePre() {
	g.startInvite()
	g.enterPrelude(preLetter)
}

// startInvite shows the card that leads to the last party (every trophy won).
func (g *Game) startInvite() {
	g.end = &endState{stage: endInvite}
	g.mode = ModeEnding
	g.sceneReplay = false
	g.markSeen("seen_invite")
}

func (g *Game) leaveEnding() {
	g.snd.StopThump()
	g.snd.StopScene()
	if g.sceneReplay { // watched again from the scenes list or from DEBUG: back where it came from
		g.leaveScene()
		return
	}
	g.end = nil
	g.bedQuiet = false
	g.loadLevel(0)
	g.mode = ModeMenu
	g.snd.StartMusic()
}

// stage lengths in seconds
func endLen(stage int) float64 { return endLenFor(stage, false) }

// endLenFor: the secret ending lasts longer than the normal one.
func endLenFor(stage int, secret bool) float64 {
	switch stage {
	case endInvite:
		return 1e9 // it waits for the player
	case endPrelude:
		return preBadLen
	case endParty:
		return partyLen
	case endWalk:
		if secret {
			return 21
		}
		return 8.5 // the normal ending is a short epilogue
	case endDoor:
		if secret {
			return 9.5
		}
		return 4.8
	case endBed:
		if secret {
			return 34
		}
		return 26
	case endCredits:
		if secret {
			return partyCreditsLen
		}
		return 30
	case endStinger:
		return 8.5
	}
	return 1e9
}

// endStageLen is the length of the part that is on now (the preludes each have their own).
func endStageLen(e *endState) float64 {
	if e.stage == endPrelude {
		return e.x.pre.length
	}
	return endLenFor(e.stage, e.trueEnd)
}

func (g *Game) updateEnding() {
	e := g.end
	if e == nil {
		g.leaveEnding()
		return
	}
	const dt = 1.0 / 60
	e.t += dt
	e.total += dt
	g.time = e.total
	g.bedQuiet = e.stage == endBed
	// the music of the ending: a lullaby (or the secret one, or the wind of the invitation), no drums
	if want, n := endTune(e); want != e.music {
		if e.music == "" || !g.snd.SceneIsSong("base") { // a song of the intros folder plays on through the stages
			g.sceneMusic(n, want)
		}
		e.music = want
	}
	g.snd.SetSceneVolume("base", endMusicVol(e))
	if e.stage == endInvite {
		g.updateInvite(e) // the letter waits for the player; it never goes away by itself
		return
	}
	if e.stage == endArrival {
		g.updateArrival(e)
		return
	}
	// the quiet night: crickets when the music has gone and he sleeps
	end := endLenFor(endBed, e.trueEnd)
	g.cricket(e, (e.stage == endBed && e.t > end-8) || (e.stage == endFinal && e.t < 9) || e.stage == endStinger && e.t < 2.4)
	if e.skipLock > 0 {
		e.skipLock -= dt
	}
	if actBack.just() {
		if e.stage == endPrelude { // only the action of the last scene is skipped
			g.endLeavePrelude(e)
			return
		}
		if e.stage == endInvite { // not the island after all
			g.leaveEnding()
			return
		}
		if e.stage == endFinal {
			g.leaveEnding()
			return
		}
		e.stage, e.t = endFinal, 0
		return
	}
	if (actConfirm.just() || ebitenClick()) && e.skipLock <= 0 {
		switch {
		case e.stage == endFinal && e.t > 4:
			g.leaveEnding()
			return
		case e.stage != endFinal:
			e.t = endStageLen(e) // skip what is left of this part
			e.skipLock = 0.3
		}
	}
	if e.stage == endPrelude {
		g.preludeSounds(e)
	}
	if e.stage == endParty {
		g.partySounds(e)
	}
	if e.t >= endStageLen(e) {
		switch e.stage {
		case endFinal:
		case endPrelude:
			g.endLeavePrelude(e)
			return
		default:
			e.stage, e.t = nextEndStage(e.stage, e.trueEnd), 0
		}
	}
	if e.stage == endWalk {
		e.heroX = 980 - e.t*870/endLenFor(endWalk, e.trueEnd)
	}
	if e.stage == endFinal && e.t > 11 && e.t < 11.1 {
		g.snd.Play("win")
	}
}

func ebitenClick() bool { return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) }

func (g *Game) drawEnding(dst *ebiten.Image) {
	e := g.end
	if e == nil {
		return
	}
	switch e.stage {
	case endInvite:
		g.drawInvite(dst, e)
	case endArrival:
		g.drawArrival(dst, e)
	case endPrelude:
		g.drawPrelude(dst, e)
	case endParty:
		g.drawParty(dst, e)
	case endWalk, endDoor:
		g.drawWalk(dst, e)
	case endBed:
		g.drawBedEnd(dst, e)
	case endCredits:
		g.drawEndCredits(dst, e)
	case endStinger:
		g.drawStinger(dst, e)
	default:
		g.drawEndFinal(dst, e)
	}
	hint := actionHint(actConfirm) + ": saltar   " + actionHint(actBack) + ": salir"
	if e.stage == endFinal && e.t > 4 {
		hint = actionHint(actConfirm) + ": volver al menú"
	}
	if e.stage == endInvite || e.stage == endArrival {
		return // the letter and the arrival have their own hint
	}
	if e.stage == endPrelude || e.stage == endParty {
		hint = actionHint(actConfirm) + ": saltar"
	}
	g.drawTextC(dst, hint, 480, 518, rgba(0x8a86a0, 190), 1)
}

// ---- the walk home ----------------------------------------------------------------------

func jpH(x int, seed uint32) float64 { return jpHash(x, 7, seed) }

// endWorld draws the street at 1x into the canvas; camX is the world x of its left edge.
func (g *Game) endWorld(c *ebiten.Image, camX float64, t float64, e *endState) {
	e.x.say = e.x.say[:0]
	c.Fill(rgb(0x070a1c))
	for y := 0.0; y < 150; y += 3 { // sky
		fillRect(c, 0, y, endCW, 3, mixc(rgb(0x070a1c), rgb(0x1c2a58), y/150))
	}
	filledEllipse(c, 390, 40, 17, 17, rgb(0xeae6d4)) // the moon
	filledEllipse(c, 384, 36, 16, 16, rgb(0x14244c))
	// far skyline (slow)
	off := math.Mod(camX*0.25, 1000)
	for k := -1; k < 30; k++ {
		bx := float64(k)*26 - math.Mod(off, 26)
		idx := int(math.Floor((off+bx+26)/26)) + 1000
		h := 28 + jpH(idx, 3)*58
		fillRect(c, bx, endGround-18-h, 24, h+18, rgb(0x0c1226))
		for wy := endGround - 14 - h; wy < endGround-24; wy += 7 {
			for wx := bx + 3; wx < bx+21; wx += 6 {
				if jpHash(int(wx)+idx, int(wy), 5) > 0.93 {
					fillRect(c, wx, wy, 2, 3, rgb(0xe8c860))
				}
			}
		}
	}
	recLayer("bg:skyline")
	// the houses along the street
	for k := -2; k < 12; k++ {
		wx := float64(k)*74 - math.Mod(camX*0.6, 74)
		idx := int(math.Floor((camX*0.6+wx+148)/74)) + 500
		hh := 54 + jpH(idx, 9)*18
		col := []uint32{0x2a2438, 0x2a3446, 0x38283a, 0x2a3a32}[idx%4]
		fillRect(c, wx, endGround-hh, 62, hh, rgb(col))
		fillRect(c, wx-3, endGround-hh-8, 68, 9, shade(rgb(col), 0.7))
		fillRect(c, wx-3, endGround-hh-8, 68, 2, shade(rgb(col), 1.4))
		for r := 0; r < 2; r++ {
			for q := 0; q < 3; q++ {
				lit := jpHash(idx*3+q, r, 11) > 0.9
				wc := rgb(0x161a2c)
				if lit {
					wc = rgb(0x4a3a1c)
				}
				fillRect(c, wx+8+float64(q)*17, endGround-hh+10+float64(r)*22, 10, 14, wc)
			}
		}
		fillRect(c, wx+26, endGround-22, 10, 22, shade(rgb(col), 0.55))
	}
	recLayer("bg:houses")
	// pavement and road
	fillRect(c, 0, endGround, endCW, 14, rgb(0x4a4658))
	fillRect(c, 0, endGround, endCW, 1, rgb(0x6a6680))
	fillRect(c, 0, endGround+14, endCW, endCH-endGround-14, rgb(0x1e1e28))
	for x := -math.Mod(camX, 40); x < endCW; x += 40 {
		fillRect(c, x, endGround+40, 22, 2, rgb(0xb8aa60))
	}
	// the mess of the last party, lying about
	for k := 0; k < 40; k++ {
		wx := float64(k)*37 + jpH(k, 21)*20
		sx := wx - camX
		if sx < -10 || sx > endCW+10 {
			continue
		}
		switch k % 4 {
		case 0: // a cup
			fillRect(c, sx, endGround+9, 4, 5, rgb(0xd8503a))
			fillRect(c, sx, endGround+9, 4, 1, rgb(0xf08a70))
		case 1: // confetti
			for q := 0; q < 6; q++ {
				fillRect(c, sx+float64(q*3), endGround+4+float64(q%3)*3, 1, 1, rgb(partyColors[(k+q)%len(partyColors)]))
			}
		case 2: // a sad balloon
			filledEllipse(c, sx, endGround+10, 4, 3, rgb(0xe03a8a))
			fillRect(c, sx+4, endGround+10, 6, 1, rgb(0xc8c8d8))
		default: // a bottle
			fillRect(c, sx, endGround+8, 2, 6, rgb(0x3aa050))
			fillRect(c, sx, endGround+6, 1, 3, rgb(0x3aa050))
		}
	}
	// street lamps
	for k := -1; k < 6; k++ {
		wx := float64(k)*130 - math.Mod(camX, 130) + 40
		fillRect(c, wx, endGround-58, 2, 58, rgb(0x2a2a36))
		fillRect(c, wx-8, endGround-60, 12, 3, rgb(0x2a2a36))
		flick := 1.0
		if int(t*3+float64(k))%11 == 0 {
			flick = 0.4
		}
		g.glowAt(c, Vec{wx - 4, endGround - 56}, 40*flick, color.RGBA{255, 214, 120, 70})
		fillRect(c, wx-7, endGround-57, 8, 2, rgb(0xfff0b0))
	}
	// his house, at the far left of the street: warm porch light and the door
	hx := 120 - camX
	if hx > -140 && hx < endCW+20 {
		fillRect(c, hx-40, endGround-78, 96, 78, rgb(0x3a3048))
		fillRect(c, hx-44, endGround-88, 104, 12, rgb(0x5a2e30))
		fillRect(c, hx-44, endGround-88, 104, 2, rgb(0x8a4a48))
		fillRect(c, hx-30, endGround-60, 18, 18, rgb(0x2a3a58))
		fillRect(c, hx+26, endGround-60, 18, 18, rgb(0x2a3a58))
		open := 0.0
		if e.stage == endDoor {
			open = math.Min(1, e.t/0.8)
			if e.t > 2.6 { // he is in: the door closes behind him
				open = 1 - clampF((e.t-2.6)/0.45, 0, 1)
			}
		}
		fillRect(c, hx-2, endGround-34, 22, 34, rgb(0x241a14))
		if open > 0 { // the light of the hall
			fillRect(c, hx, endGround-32, 18*open, 32, rgb(0xf0c870))
		}
		fillRect(c, hx-2, endGround-36, 22, 3, rgb(0x5a4030))
		g.glowAt(c, Vec{hx + 9, endGround - 40}, 46, color.RGBA{255, 200, 90, 80})
		fillRect(c, hx+22, endGround-30, 3, 5, rgb(0xfff0a0)) // porch lamp
	}
	recLayer("bg:street")
	// the hero (and the gags of each ending around him)
	g.endBehind(c, camX, t, e)
	g.endHero(c, e, camX)
	g.endFront(c, camX, t, e)
	if e.stage == endDoor && e.t > 2.9 && e.t < 3.9 { // the ghosts bump into the shut door
		e.sayAt("¡BUM!", hx+30, endGround-70-(e.t-2.9)*8, 0xf8d030, uint8(255*(1-(e.t-2.9))))
	}
}

func (g *Game) endHero(c *ebiten.Image, e *endState, camX float64) {
	cs := g.as.playerCS
	if g.scene2P() {
		if c0 := g.sceneHero(0, false); c0 != nil {
			cs = c0
		}
	}
	if cs == nil {
		return
	}
	recLayer("actor:hero")
	x := e.heroX - camX
	anim, ph := AnimWalk, e.total*6
	dir := DirLeft
	if e.stage == endDoor {
		x = 120 + 9 - camX
		anim, ph = AnimIdle, 0
		if e.t > 0.8 { // walks in
			x = 129 - camX
			dir = DirUp
			anim, ph = AnimWalk, e.total*5
		}
	}
	if e.stage == endWalk && e.t > endLen(endWalk)-1.5 {
		anim = AnimWalk
	}
	img, flip := cs.img(anim, dir, ph)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(charW, 0)
	}
	op.GeoM.Translate(math.Round(x-charCX), math.Round(endGround+11-charBase))
	if e.stage == endDoor && e.t > 0.8 { // going through the door
		k := math.Min(1, (e.t-0.8)/1.6)
		op.ColorScale.ScaleAlpha(float32(1 - k))
	}
	fillShadow(c, x, endGround+11)
	c.DrawImage(img, op)
	if g.scene2P() {
		g.endHero2(c, e, camX)
	}
}

func (g *Game) drawWalk(dst *ebiten.Image, e *endState) {
	if e.canvas == nil {
		e.canvas = ebiten.NewImage(endCW, endCH)
	}
	camX := math.Max(0, e.heroX-300)
	if e.stage == endDoor {
		camX = math.Max(0, 129-300)
	}
	g.endWorld(e.canvas, camX, e.total, e)
	// the camera zooms in on him: from the whole street to a close up
	var zoom float64
	if e.stage == endWalk {
		zoom = 2 + math.Min(1.6, e.t/endLenFor(endWalk, e.trueEnd)*1.9)
	} else {
		zoom = 2.8 + math.Min(0.4, e.t*0.04) // wide enough to see his door and the neighbour's window
	}
	hx := e.heroX - camX
	if e.stage == endDoor {
		hx = 120 - camX
	}
	fx, fy := hx, endGround-20
	if e.stage == endDoor {
		fx = 178 - camX
	}
	dst.Fill(rgb(0x05060c))
	ox, oy := 480-fx*zoom, 340-fy*zoom
	ox = math.Max(screenW-endCW*zoom, math.Min(0, ox)) // never show past the edge of the street
	oy = math.Max(screenH-endCH*zoom, math.Min(0, oy))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(ox, oy)
	dst.DrawImage(e.canvas, op)
	dst.DrawImage(g.as.vignette, nil)
	e.x.camZ, e.x.camOX, e.x.camOY = zoom, ox, oy
	for _, w := range e.x.say { // the little texts of the street, crisp, with the camera of this frame
		g.drawText(dst, w.s, math.Round(ox+w.x*zoom), math.Round(oy+w.y*zoom), rgba(w.col, w.a), 2)
	}
	// captions
	line := g.walkCaption(e)
	g.endCaption(dst, line)
	if g.scene2P() {
		g.endTwoSay(dst, e)
	}
	if e.stage == endDoor { // fade out at the very end of the stage
		if end := endLenFor(endDoor, e.trueEnd); e.t > end-1.6 {
			fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*math.Min(1, (e.t-(end-1.6))/1.6))))
		}
	}
	if e.stage == endWalk && e.t < 1 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-e.t))))
	}
}

func (g *Game) endCaption(dst *ebiten.Image, line string) {
	w := g.textW(tr(line), 1.5) + 40
	fillRect(dst, 480-w/2, 452, w, 34, rgba(0x000000, 170))
	g.drawTextC(dst, line, 480, 460, rgb(0xf4f0ff), 1.5)
}

// ---- bed: counting sheep ----------------------------------------------------------------------

func (g *Game) drawBedEnd(dst *ebiten.Image, e *endState) {
	if g.scene2P() {
		g.drawBedEnd2(dst, e)
		return
	}
	if e.bed == nil {
		e.bed = ebiten.NewImage(screenW, screenH)
	}
	g.time = e.total
	e.bed.Fill(rgb(0x08060c))
	g.drawBedroom(e.bed)
	// a slow zoom on the bed
	zoom := 1 + math.Min(0.22, e.t/endLen(endBed)*0.3)
	dst.Fill(rgb(0x05060c))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-420, -330)
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(480, 330)
	dst.DrawImage(e.bed, op)
	// the thought bubble, with the sheep
	g.drawSheepCloud(dst, e)
	// the lights go out
	dark := 0.0
	if e.t > endLen(endBed)-5 {
		dark = math.Min(1, (e.t-(endLen(endBed)-5))/4.2)
	}
	fillRect(dst, 0, 0, screenW, screenH, rgba(0x000008, uint8(235*dark)))
	if e.t > endLen(endBed)-4.5 {
		for i := 0; i < 3; i++ {
			f := math.Mod(e.t*0.5+float64(i)/3, 1)
			g.drawText(dst, "Z", 400+f*60+float64(i)*8, 300-f*80, rgba(0xe8f0ff, uint8(240*(1-f))), float64(3+i))
		}
	}
	g.endCaption(dst, g.bedCaption(e))
	g.bedGags(dst, e)
}

func (g *Game) drawSheepCloud(dst *ebiten.Image, e *endState) {
	if e.t < 4 {
		return
	}
	cx, cy := 190.0, 190.0
	a := uint8(255 * math.Min(1, (e.t-4)/1.2))
	al := func(c color.RGBA) color.RGBA { c.A = a; return c }
	filledEllipse(dst, cx, cy, 130, 70, al(rgb(0xf4f6ff)))
	filledEllipse(dst, cx-70, cy-42, 54, 34, al(rgb(0xf4f6ff)))
	filledEllipse(dst, cx+60, cy-44, 58, 34, al(rgb(0xf4f6ff)))
	filledEllipse(dst, cx-74, cy+50, 46, 28, al(rgb(0xf4f6ff)))
	for i, r := range []float64{9, 6, 4} { // the little bubbles towards his head
		filledEllipse(dst, cx+90+float64(i)*26, cy+80+float64(i)*16, r, r, al(rgb(0xf4f6ff)))
	}
	// sky inside the cloud, a meadow and a fence
	fillRect(dst, cx-104, cy-44, 208, 92, al(rgb(0xbfe0f8)))
	fillRect(dst, cx-104, cy+20, 208, 28, al(rgb(0x6ab46a)))
	for x := cx - 36; x <= cx+36; x += 18 {
		fillRect(dst, x, cy-4, 3, 30, al(rgb(0x8a6a44)))
	}
	fillRect(dst, cx-40, cy+4, 80, 3, al(rgb(0x8a6a44)))
	fillRect(dst, cx-40, cy+14, 80, 3, al(rgb(0x8a6a44)))
	// the sheep: one after another; the count goes up with every jump
	const per = 1.9
	t := e.t - 5
	if t < 0 {
		return
	}
	n := int(t / per)
	if n > 9 {
		n = 9
	}
	for k := math.Max(0, float64(n-1)); k <= float64(n); k++ {
		u := (t - k*per) / per
		if u < 0 || u > 1 || k > 9 {
			continue
		}
		x := cx - 110 + u*220
		jump := 0.0
		if u > 0.34 && u < 0.66 {
			jump = math.Sin((u-0.34)/0.32*math.Pi) * 34
		}
		g.drawSheep(dst, x, cy+22-jump, u, a)
		g.sheepExtra(dst, x, cy+22-jump, int(k), e.trueEnd, a)
	}
	count := n + 1
	if count > 10 {
		count = 10
	}
	txt := fmt.Sprintf("%d ovejita...", count)
	if count > 1 {
		txt = fmt.Sprintf("%d ovejitas...", count)
	}
	if t > 9*per+1 {
		txt = "zzz... ovejita... zzz..."
	}
	g.drawTextC(dst, txt, cx, cy-62, al(rgb(0x3a3a5a)), 1.4)
}

func (g *Game) drawSheep(dst *ebiten.Image, x, y, u float64, a uint8) {
	al := func(c color.RGBA) color.RGBA { c.A = a; return c }
	wool, dark := rgb(0xfdfdfd), rgb(0x2a2a34)
	legs := math.Sin(u * 40)
	for _, lx := range []float64{-9, -3, 4, 10} {
		fillRect(dst, x+lx, y+6, 2, 8+legs*lx*0.05, al(dark))
	}
	for _, p := range [][3]float64{{0, 0, 12}, {-9, -3, 8}, {8, -3, 9}, {-4, -6, 8}, {5, -7, 8}, {12, 0, 7}, {-12, 2, 7}} {
		filledEllipse(dst, x+p[0], y+p[1], p[2]*0.8, p[2]*0.7, al(wool))
	}
	filledEllipse(dst, x+17, y-1, 6, 5, al(dark))
	fillRect(dst, x+19, y-3, 2, 2, al(rgb(0xffffff)))
	fillRect(dst, x+14, y-6, 3, 3, al(dark))
}

// ---- credits ----------------------------------------------------------------------------------------

func (g *Game) drawEndCredits(dst *ebiten.Image, e *endState) {
	dst.Fill(rgb(0x04050c))
	if e.trueEnd { // the party goes on behind the names
		g.partyCreditsBG(dst, e)
	}
	rows := endCreditRows(e.trueEnd)
	y := 560 - e.t*endCreditSpeed(e.trueEnd)
	for _, r := range rows {
		if r[0] == "" {
			y += 28
			continue
		}
		if r[1] == "Fiesta Nocturna" {
			g.drawTextC(dst, r[0], 480, y, rgb(0xff6a8a), 3)
			g.drawTextC(dst, r[1], 480, y+44, rgb(0xf4f0ff), 1.8)
			y += 98
			continue
		}
		g.drawText(dst, r[0], 250, y, rgb(0x3fb59a), 1.6)
		g.drawText(dst, r[1], 500, y, rgb(0xffe060), 1.6)
		y += 38
	}
	g.drawWrapped(dst, "Juego hecho como homenaje al Party Hard original, desarrollado por Pinokl Games y distribuido por tinyBuild. Este proyecto es de fans y no tiene ninguna relación oficial con ellos.", 190, y+16, 580, rgb(0xb8b0d0))
	fade := math.Min(1, e.t/1.5)
	fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-fade))))
}

// ---- the last card ------------------------------------------------------------------------------------

func (g *Game) drawEndFinal(dst *ebiten.Image, e *endState) {
	dst.Fill(rgb(0x04050c))
	k := math.Min(1, e.t/1.4)
	if e.trueEnd {
		g.drawTextC(dst, "FIN", 480, 150, rgba(0xf8d030, uint8(255*k)), 8)
		if e.t > 2.2 {
			g.drawTextC(dst, "Todos los trofeos, todas las fiestas, hasta una isla con dinosaurios.", 480, 290, rgb(0xf4f0ff), 1.3)
			g.drawTextC(dst, "Ahora sí: a dormir.", 480, 318, rgb(0x60e090), 1.5)
		}
	} else {
		wob := math.Sin(e.total*2.2) * 2
		g.drawTextC(dst, "¿ FINAL ?", 480, 140+wob, rgba(0xff6a8a, uint8(255*k)), 7)
		if e.t > 2.2 {
			g.drawTextC(dst, "Dicen que quien consiga todos los trofeos recibirá una invitación...", 480, 290, rgb(0xb8b0d0), 1.2)
			g.drawTextC(dst, "Y que la última fiesta tiene dientes.", 480, 316, rgb(0xe8c870), 1.3)
		}
	}
	if e.t > 4.6 {
		g.drawTextC(dst, "Gracias por jugar", 480, 400, rgba(0xf4f0ff, uint8(255*math.Min(1, (e.t-4.6)/1.2))), 2.6)
	}
}

// endTune: the tune for this moment of the ending and the number of its scene (for the intros folder):
// the invitation has its own, the secret ending a fanfare, the normal one a lullaby that goes calm at the end.
func endTune(e *endState) (string, int) {
	switch {
	case e.stage == endInvite || e.stage == endArrival:
		return ambInvite, sceneInviteN
	case e.stage == endPrelude && (e.x.pre.kind == preLetter || e.x.pre.kind == preIsland):
		return ambInvite, sceneInviteN // the letter goes on with the theme it has afterwards; the island with the one of the helicopter
	case e.stage == endPrelude:
		return ambEnd, sceneEndN
	case e.trueEnd:
		return ambSecret, sceneSecretN
	case e.stage == endFinal:
		return ambEndCalm, sceneEndN
	}
	return ambEnd, sceneEndN
}

// endMusicVol: how loud the music of the ending is in each part: it comes up while he walks home,
// is softest in the bed (the sheep), swells in the credits and dies away at the very end.
func endMusicVol(e *endState) float64 {
	switch e.stage {
	case endPrelude:
		return preMusicVol(e)
	case endParty:
		return 0.75 * math.Min(1, e.t/1.5)
	case endInvite:
		if e.x.pre.length > 0 { // it comes from the action of the station: the tune is already playing
			return 0.6
		}
		return 0.6 * math.Min(1, e.t/1.2)
	case endArrival:
		return arrMusicVol(e.t)
	case endWalk:
		return 0.5 * math.Min(1, e.t/3)
	case endDoor:
		return 0.5
	case endBed: // it dies away at the end: the silence
		return 0.55 * (1 - clampF((e.t-(endLenFor(endBed, e.trueEnd)-9))/6, 0, 1))
	case endCredits:
		return 0.7
	case endStinger:
		return 0.12
	}
	if e.trueEnd {
		return 0.7 * (1 - introEase(e.t/12))
	}
	return 0.6 * (1 - introEase(e.t/14))
}
