package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The endings for two players.
//
// Same two endings, with the two heroes and their characters (twoscenes.go): the blue one is patient,
// the red one nervous. On the way home the blue one walks at his own pace and the red one yawns, jumps
// at a balloon and runs round and round the blue one because of a cat (normal ending) or walks
// backwards shrieking at the ghosts of the conga and slips on a T-Rex footprint (secret ending). At the
// door the red one has lost the keys and gets stuck in the doorway with the blue one. In the bedroom there
// are two beds: the blue one says good night and snores at once; the red one sits up to count sheep (or
// dinosaurs), throws his pillow at the snorer and finally, as everything goes quiet, falls asleep too.
// ---------------------------------------------------------------------------

// endTwoOff is where the red one walks relative to the blue one on the street (pixels; positive = behind).
func endTwoOff(t float64, secret bool) float64 {
	off := 34 + 14*math.Sin(t*1.3)
	if !secret {
		off += 40 * (introEase((t-1.4)/0.8) - introEase((t-3.8)/0.8)) // yawning, he drops behind
		if t > 6.4 && t < 8.3 {                                       // the ghosts: he backs away
			off = 34 + 36*math.Sin((t-6.4)*3.3)
		}
	} else if t > 11 && t < 13 { // slipping on the footprint
		off = 34 + 20*math.Sin((t-11)*3)
	}
	return off
}

// endTwoActor: what the red one does at time t on the street: animation, phase, direction, hop.
func endTwoActor(e *endState) (anim int, ph float64, dir Dir, hop, lean float64) {
	t := e.t
	anim, ph, dir = AnimWalk, e.total*6.8, DirLeft
	if e.stage != endWalk {
		return
	}
	switch {
	case !e.trueEnd && t > 1.6 && t < 3.8: // a huge yawn, dragging his feet
		anim, ph = AnimTired, e.total*3
		lean = 0.05
	case !e.trueEnd && t > 4.0 && t < 5.4: // the balloon scares him
		hop = math.Abs(math.Sin(t*9)) * 8
		anim, ph = AnimPanic, t*8
	case !e.trueEnd && t > 6.4 && t < 8.3: // the ghosts: backwards, looking at them
		anim, ph = AnimPanic, e.total*10
		dir = DirRight
		hop = math.Abs(math.Sin(t*9)) * 5
	case e.trueEnd && t > 6 && t < 14.5: // backwards, looking at the ghosts
		dir = DirRight
		anim = AnimWalk
		if t > 6 && t < 7.4 {
			hop = math.Abs(math.Sin(t*9)) * 6
			anim = AnimPanic
		}
		if t > 11 && t < 12.4 { // slips on the footprint
			anim, ph = AnimFall, 0
		}
	}
	return
}

func (g *Game) endHero2(c *ebiten.Image, e *endState, camX float64) {
	cs := g.sceneHero(1, false)
	if cs == nil {
		return
	}
	recLayer("actor:hero")
	var x, alpha float64
	alpha = 1
	switch e.stage {
	case endWalk:
		x = e.heroX + endTwoOff(e.t, e.trueEnd) - camX
	case endDoor:
		// the keys: he searches his pockets, then rushes at the door and gets stuck with the blue one
		x = 150 - camX
		if e.t > 1.5 {
			u := introEase((e.t - 1.5) / 0.9)
			x = 150 + (129-150)*u - camX
		}
		if e.t > 2.4 {
			alpha = 1 - clampF((e.t-2.4)/0.2, 0, 1)
		}
	default:
		return
	}
	anim, ph, dir, hop, lean := endTwoActor(e)
	if e.stage == endDoor {
		anim, ph, dir = AnimPanic, e.total*9, DirLeft // pockets, pockets...
		if e.t > 1.5 {
			anim, ph, dir = AnimWalk, e.total*7, DirUp
		}
		hop = 0
	}
	tint := sceneTint(1, g.sceneHero(0, false), cs)
	img, flip := cs.img(anim, dir, ph)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(charW, 0)
	}
	op.GeoM.Translate(-charCX, -charBase)
	if lean != 0 {
		op.GeoM.Rotate(lean)
	}
	op.GeoM.Translate(math.Round(x), math.Round(endGround+11-hop))
	op.ColorScale.Scale(tint[0], tint[1], tint[2], 1)
	op.ColorScale.ScaleAlpha(float32(alpha))
	fillShadow(c, x, endGround+11)
	c.DrawImage(img, op)
	if e.stage == endWalk && !e.trueEnd && e.t > 3.8 && e.t < 6.0 { // the balloon, going down with a "pffft"
		u := (e.t - 3.8) / 2.2
		bx := e.heroX - camX + 140 - u*170
		by := 70 + u*60 + math.Sin(u*18)*18
		filledEllipse(c, bx, by, 6-3*u, 7-3*u, rgb(0xe03a8a))
		fillRect(c, bx, by+6, 1, 12, rgba(0xc8c8d8, 180))
		if int(e.t*8)%3 == 0 {
			e.sayAt("pffft", bx+6, by-8, 0xf4f0ff, 200)
		}
	}
}

// endSayWorld: a comic balloon over a point of the street (world coordinates of the endings' canvas).
func (g *Game) endSayWorld(dst *ebiten.Image, e *endState, wx, wy float64, s string) {
	z := e.x.camZ
	if z <= 0 {
		return
	}
	g.speechBubble(dst, e.x.camOX+wx*z, e.x.camOY+wy*z, s)
}

func (g *Game) endTwoSay(dst *ebiten.Image, e *endState) {
	t := e.t
	camX := math.Max(0, e.heroX-300)
	if e.stage == endDoor {
		camX = math.Max(0, 129-300)
	}
	p1x := e.heroX - camX
	p2x := e.heroX + endTwoOff(t, e.trueEnd) - camX
	top := endGround - 46
	if e.stage == endDoor {
		p1x, p2x = 129-camX, 150-camX
		if t > 1.5 {
			p2x = 129 - camX
		}
	}
	b1 := func(s string) { g.endSayWorld(dst, e, p1x, top, s) }
	b2 := func(s string) { g.endSayWorld(dst, e, p2x, top, s) }
	switch {
	case e.stage == endWalk && !e.trueEnd:
		switch {
		case sceneIn(t, 0.8, 2.4):
			b1("Por fin, silencio.")
		case sceneIn(t, 2.0, 3.6):
			b2("¡¡AAAAAAH!! (bostezo)")
		case sceneIn(t, 4.0, 5.4):
			b2("¡¿QUÉ ES ESO?!")
		case sceneIn(t, 5.5, 6.4):
			b1("Un globo. Respira.")
		case sceneIn(t, 6.5, 8.0):
			b2("¡¡FANTASMAS!!")
		}
	case e.stage == endWalk:
		switch {
		case sceneIn(t, 1.0, 2.6):
			b1("Qué aire más limpio.")
		case sceneIn(t, 2.8, 4.4):
			b2("¿Eso era un T-Rex?")
		case sceneIn(t, 6.0, 7.4):
			b2("¡¡FANTASMAS!!")
		case sceneIn(t, 7.6, 9.2):
			b1("Son los de las fiestas.")
		case sceneIn(t, 11.0, 12.4):
			b2("¡AY! ¡Una huella!")
		case sceneIn(t, 12.6, 14.2):
			b1("Te lo dije.")
		case sceneIn(t, 16.0, 17.6):
			b2("¡Murray me mira!")
		case sceneIn(t, 17.8, 19.4):
			b1("Murray mira a todos.")
		}
	case e.stage == endDoor:
		switch {
		case sceneIn(t, 0.2, 1.4):
			b2("¿Dónde están las llaves?")
		case sceneIn(t, 1.5, 2.4):
			b1("Después de ti.")
		case sceneIn(t, 2.2, 3.0):
			b2("¡¡ESPERADME!!")
		}
	}
}

func (g *Game) walkCaption2(e *endState) string {
	if !e.trueEnd {
		switch {
		case e.stage == endDoor && e.t < 2.4:
			return "El azul tiene la llave. El rojo, una lista de excusas."
		case e.stage == endDoor && e.t < 4.5:
			return "Se atascan los dos en la puerta. Hasta el vecino apaga la luz."
		case e.stage == endDoor:
			return "Ni un bajo. Ni una voz. Ni un solo 'otra más'."
		case e.t < 4.8:
			return "Después de acabar con todas las fiestas... a casa. El rojo, bostezando."
		case e.t < 6:
			return "Un globo desinflándose. El rojo ya estaba a punto de pegar un salto."
		default:
			return "Los fantasmas de las fiestas les siguen. El azul saluda; el rojo huye."
		}
	}
	switch {
	case e.stage == endDoor && e.t < 2.4:
		return "El azul con su llave. El rojo con sus bolsillos. Y los fantasmas, detrás."
	case e.stage == endDoor && e.t < 4.6:
		return "¡Portazo! Los invitados fantasma se quedan fuera."
	case e.stage == endDoor && e.t < 6.2:
		return "El vecino pone la Música Jurásica a todo volumen..."
	case e.stage == endDoor && e.t < 8.2:
		return "...y alguien muy grande ha oído el bajo."
	case e.stage == endDoor:
		return "Silencio. Un altavoz menos en el barrio."
	case e.t < 5:
		return "Un helicóptero les deja en casa. El piloto se va corriendo."
	case e.t < 10:
		return "Huellas de T-Rex. El azul las esquiva. El rojo, no tanto."
	case e.t < 15.5:
		return "Los fantasmas de todas las fiestas hacen la conga. El rojo grita."
	default:
		return "Hasta Murray viene en su estaca, hablando sin parar."
	}
}

// ---- the bedroom with two beds ---------------------------------------------------------------------

// endBedPoses: what each of them does in bed at time e.t.
func endBedPoses(e *endState) [2]bedPose {
	end := endLenFor(endBed, e.trueEnd)
	up := func(a, b float64) float64 { return introEase((e.t-a)/0.5) * (1 - introEase((e.t-b)/0.5)) }
	var p [2]bedPose
	p[0].turn = 0 // the blue one falls asleep at once and does not move
	p[1].turn = 2.6
	if e.t > end-9 {
		p[1].turn = 0
	}
	p[1].sit = up(5.6, 9.0) // counting out loud
	pil := 0.0
	if e.trueEnd {
		for _, at := range []float64{21.6, 22.4, 23.2, 24.0} {
			pil = math.Max(pil, up(at-0.1, at+0.5))
		}
	}
	p[1].pillow = math.Max(pil, up(11, 12.5))
	return p
}

// bedCaption2: what the narrator says in the two beds.
func (g *Game) bedCaption2(e *endState) string {
	end := endLenFor(endBed, e.trueEnd)
	if !e.trueEnd {
		switch {
		case e.t < 5:
			return "Dos camas. Buenas noches. Uno ronca ya; el otro, aún no."
		case e.t < 12:
			return "El rojo se sienta a contar ovejitas. En voz alta."
		case e.t < 19:
			return "Hasta las ovejas se han puesto el pijama. Y roncan menos que el azul."
		case e.t < end-9:
			return "Una almohada voladora y... silencio. Por fin."
		default:
			return "Grillos. Sólo grillos. Zzz... zzz..."
		}
	}
	switch {
	case e.t < 5:
		return "Dos camas. Un T-Rex lejano que ya no ruge. Buenas noches."
	case e.t < 11:
		return "El rojo cuenta ovejitas. El azul ya sueña con ellas."
	case e.t < 17:
		return "¡Esa oveja tiene dientes! Y la siguiente es verde."
	case e.t < 21:
		return "La última oveja es un T-Rex. Por supuesto."
	case e.t < 25:
		return "¿Pum... pum...? Otra vez no. Otra vez no."
	case e.t < 28:
		return "¡RAAAWR! ...y se acabó el altavoz del vecino."
	default:
		return "Silencio. Sólo grillos. Zzz... zzz..."
	}
}

func (g *Game) drawBedEnd2(dst *ebiten.Image, e *endState) {
	if e.bed == nil {
		e.bed = ebiten.NewImage(screenW, screenH)
	}
	g.time = e.total
	e.bed.Fill(rgb(0x08060c))
	bedTwo, bedPoses = true, endBedPoses(e)
	g.drawBedroom(e.bed)
	bedTwo, bedPoses = false, [2]bedPose{}
	g.endBedGagsRoom(e.bed, e)
	// a slow zoom on the two beds
	zoom := 1 + math.Min(0.14, e.t/endLen(endBed)*0.2)
	dst.Fill(rgb(0x05060c))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-480, -330)
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(480, 330)
	dst.DrawImage(e.bed, op)
	// the red one's thought cloud with his sheep (or dinosaurs): the one-player cloud, moved over his bed
	if e.t >= 4 {
		if e.x.cloudTmp == nil {
			e.x.cloudTmp = ebiten.NewImage(screenW, screenH)
		}
		e.x.cloudTmp.Clear()
		g.drawSheepCloud(e.x.cloudTmp, e)
		cop := &ebiten.DrawImageOptions{}
		cop.GeoM.Translate(570, 0)
		dst.DrawImage(e.x.cloudTmp, cop)
	}
	// the lights go out
	dark := 0.0
	if e.t > endLen(endBed)-5 {
		dark = math.Min(1, (e.t-(endLen(endBed)-5))/4.2)
	}
	fillRect(dst, 0, 0, screenW, screenH, rgba(0x000008, uint8(235*dark)))
	if e.t > endLen(endBed)-4.5 { // both asleep: one Z over each head
		for i := 0; i < 3; i++ {
			f := math.Mod(e.t*0.5+float64(i)/3, 1)
			g.drawText(dst, "Z", 200+f*50+float64(i)*8, 300-f*80, rgba(0xe8f0ff, uint8(240*(1-f))), float64(3+i))
			g.drawText(dst, "z", 700+f*40+float64(i)*8, 310-f*70, rgba(0xffe8e8, uint8(240*(1-f))), float64(2+i))
		}
	}
	g.endCaption(dst, g.bedCaption(e))
	g.bedGags(dst, e)
	g.endBedSay(dst, e, zoom)
}

// the room side of the jokes: the blue one's snores and the pillow of the red one
func (g *Game) endBedGagsRoom(c *ebiten.Image, e *endState) {
	t := e.t
	if t > 3.5 && t < endLenFor(endBed, e.trueEnd)-5 { // the blue one's snores: a saw
		for i := 0; i < 3; i++ {
			f := math.Mod(t*0.9+float64(i)/3, 1)
			g.drawText(c, "ZZZ", bedHead1X+10+f*40, 290-f*50, rgba(0xa8c0ff, uint8(240*(1-f))), 2)
		}
	}
	if t >= 9.0 && t < 9.9 { // the pillow flies from the red one to the snorer
		u := (t - 9.0) / 0.9
		x := 640 + (bedHead1X+20-640)*u
		y := 330 - math.Sin(math.Pi*u)*90
		fillRect(c, x-14, y-9, 28, 18, rgb(0xd8dee8))
		fillRect(c, x-14, y-9, 28, 3, rgb(0xf0f4fa))
		fillRect(c, x-14, y+6, 28, 3, rgb(0xa8b0c2))
		if u > 0.9 {
			g.sceneBang(c, "¡FOF!", bedHead1X+20, 300, (u-0.9)/0.1, 0xffe060, 1.6)
		}
	}
}

// balloons over the beds: the zoom of the frame is applied by hand (the room is the middle of the screen)
func (g *Game) endBedSay(dst *ebiten.Image, e *endState, zoom float64) {
	t := e.t
	sx := func(x float64) float64 { return 480 + (x-480)*zoom }
	sy := func(y float64) float64 { return 330 + (y-330)*zoom }
	b1 := func(s string) { g.speechBubble(dst, sx(bedHead1X), sy(twoBedCenterY-70), s) }
	b2 := func(s string) { g.speechBubble(dst, sx(bedHead2X), sy(twoBedCenterY-70), s) }
	switch {
	case sceneIn(t, 0.6, 2.2):
		b1("Buenas noches.")
	case sceneIn(t, 2.4, 4.0):
		b2("¡Y no ronques!")
	case sceneIn(t, 9.0, 10.4):
		b2("¡¡QUE RONCAS!!")
	case sceneIn(t, 10.5, 11.8):
		b1("...zzz... ¿mmm?")
	}
}
