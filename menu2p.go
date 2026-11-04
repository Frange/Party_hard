package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The main menu with two players: the room has two beds (bedroom2p.go, slid to the right by
// menuBedShiftX) and each sleeper keeps the character he has in the intro of two players (intro2p.go):
// the blue one is the patient one (lies still, counts sheep and dinosaurs, paper earplugs, pillow over his
// head) and the red one the nervous one (tosses and turns, sits up fuming, bangs on the wall, throws a
// slipper, makes the "turn it down" sign, stares at the 3:47 on the clock). They do not do the same thing at
// the same time: each one has his own cycle. What they say goes through the bubble manager (bubbles.go).
// ---------------------------------------------------------------------------

const (
	menuCycle1 = 15.0 // seconds of the blue one's routine
	menuCycle2 = 18.0 // and of the red one's
	menuSlot1  = 6.3  // seconds he keeps a sentence
	menuSlot2  = 4.6
)

// menuLines1: what the patient one mutters (not one of them is in menuLines2).
var menuLines1 = []string{
	"son las 3:47 de la mañana...",
	"cuento ovejas y están bailando",
	"respira... solo es música... ¿verdad?",
	"contando dinosaurios: voy por el rex",
	"paz interior... paz interior... ¡PAZ!",
	"los tapones de papel no hacen magia",
	"mañana tengo que madrugar...",
	"si no los oigo, no existen",
}

// menuLines2: what the nervous one shouts at the party next door.
var menuLines2 = []string{
	"¡BAJA ESE VOLUMEN, PEDAZO DE ANIMAL!",
	"¡que algunos curramos mañana!",
	"¿eso es un bajo o un terremoto?",
	"voy a ir en pijama a cantarles",
	"¿3:47 y con reguetón? ¡sinvergüenzas!",
	"¡a ver si les revienta el altavoz!",
	"mi almohada ya pide la baja",
	"¡DJ de pacotilla, pon una nana!",
	"¡cierra la ventana, Paco!",
	"otra canción más y me hago vecino",
}

// menuLine is the sentence player i says at time t: a new one every slot, never one of the other player's.
func menuLine(i int, t float64) string {
	if i == 0 {
		return menuLines1[(int(t/menuSlot1)*3)%len(menuLines1)]
	}
	return menuLines2[(int(t/menuSlot2)*3+1)%len(menuLines2)]
}

// the shape of a "up" window: 0 outside [a, b), 1 inside, with soft edges of half a second
func menuUp(t, a, b float64) float64 {
	return introEase((t-a)/0.5) * (1 - introEase((t-b)/0.5))
}

// menuTwoPoses: what each of them does in bed at time t (drawTwoBeds paints it).
func menuTwoPoses(t float64) [2]bedPose {
	t1 := math.Mod(t, menuCycle1)
	t2 := math.Mod(t+3, menuCycle2)
	// the patient one: he lies still on his back, with earplugs and then the pillow over his head
	p1 := bedPose{turn: 0, earplug: t1 >= 7.5 && t1 < 13, pillow: menuUp(t1, 10.2, 12.8)}
	// the nervous one: tosses and turns, pillow, sits up fuming and banging, lies back, and starts again
	sit := menuUp(t2, 4.2, 9.6)
	p2 := bedPose{
		turn:   3.2,
		pillow: menuUp(t2, 2.2, 4.0),
		sit:    sit,
		puff:   sit > 0.6 && t2 < 6.2,
		fist:   t2 >= 6.2 && t2 < 8.2,
	}
	return [2]bedPose{p1, p2}
}

// menuBedPoses is the room's poses at the start of the routines (the first frame).
func menuBedPoses() [2]bedPose { return menuTwoPoses(0) }

// menuTwoOverlay paints the gags of the two beds and their sentences over the night tint.
func (g *Game) menuTwoOverlay(dst *ebiten.Image, t float64) {
	dy := bedShiftY
	if dy != 0 { // the room slid down: the gags go with it
		if menuGagLayer == nil {
			menuGagLayer = ebiten.NewImage(screenW, screenH)
		}
		menuGagLayer.Clear()
		g.menuTwoGags(menuGagLayer, t)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, dy)
		dst.DrawImage(menuGagLayer, op)
	} else {
		g.menuTwoGags(dst, t)
	}
	g.menuTwoSpeech(dst, t, dy)
}

var menuGagLayer *ebiten.Image

// menuTwoGags paints the gags of the two beds (the Zs, the sheep, the steam, the slipper...).
func (g *Game) menuTwoGags(dst *ebiten.Image, t float64) {
	dx := bedShiftX
	t1 := math.Mod(t, menuCycle1)
	t2 := math.Mod(t+3, menuCycle2)
	ps := menuTwoPoses(t)
	h1x, h1y := bedHead1X+dx, twoBedCenterY
	// ---- the blue one
	if t1 < 3.5 || t1 >= 13 { // Zs
		for i := 0; i < 3; i++ {
			f := math.Mod(t*0.55+float64(i)/3, 1)
			g.drawText(dst, "Z", math.Round(h1x+6+f*34+float64(i)*3), math.Round(h1y-44-f*40), rgba(0xe8f0ff, uint8(240*(1-f))), float64(2+i))
		}
	}
	if t1 >= 3.5 && t1 < 7.5 { // counting sheep: one jumps the fence over his head, every 1.3 s
		k := math.Mod(t1-3.5, 1.3) / 1.3
		n := int((t1-3.5)/1.3) + 1
		fx := h1x + 60
		fillRect(dst, fx-1, h1y-84, 2, 22, rgb(0xc8b088)) // the fence
		fillRect(dst, fx+8, h1y-84, 2, 22, rgb(0xc8b088))
		fillRect(dst, fx-3, h1y-76, 16, 2, rgb(0xc8b088))
		sx, sy := h1x+14+k*110, h1y-66-math.Sin(math.Pi*k)*26
		g.drawSheep(dst, math.Round(sx), math.Round(sy), t1*0.1, 255) // the sheep of the ending
		g.drawText(dst, string(rune('0'+n%10)), h1x-2, h1y-96, rgba(0xe8f0ff, 220), 2)
	}
	if ps[0].earplug {
		drawEarplugs(dst, dx)
		if t1 < 8.7 {
			g.sceneBang(dst, "pok", h1x+30, h1y-34, sceneWindow(t1, 7.5, 8.7), 0xe8e8f4, 1.1)
		}
	}
	// ---- the red one
	sitX, sitY := bed2X0+60+dx, 250.0
	if ps[1].puff {
		drawSteam(dst, t, sitX, sitY)
		if int(t*1.2)%2 == 0 {
			g.sceneBang(dst, "¡PUF!", sitX, sitY-48, math.Mod(t*1.2, 1), 0xff8a6a, 1.5)
		}
	}
	if ps[1].fist { // bangs on the wall behind the headboard, on the beat
		k := math.Abs(math.Sin(t * 5))
		x := bed2X1 + 16 + k*6 + dx // only 26 px of room past the headboard
		fillRect(dst, x, 338, 14, 11, rgb(0xe0b088))
		fillRect(dst, x, 338, 14, 3, rgb(0xf0c8a0))
		if k > 0.7 {
			g.sceneBang(dst, "¡PUM!", bed2X1-30+dx, 312, math.Min(1, (1-k)*3), 0xffe060, 1.8)
		}
	}
	if t2 >= 8.2 && t2 < 9.8 { // the slipper at the window and down onto the floor
		u := (t2 - 8.2) / 1.6
		var x, y float64
		if u < 0.5 {
			v := u * 2
			x, y = sitX-40+(560-(sitX-40))*v, 300+(212-300)*v-math.Sin(math.Pi*v)*40
		} else {
			v := (u - 0.5) * 2
			x, y = 560+(600-560)*v, 212+(420-212)*v*v
		}
		rot := (t2 - 8.2) * 14
		for k := 0; k < 8; k++ {
			ox, oy := math.Cos(rot)*float64(k-4)*1.6, math.Sin(rot)*float64(k-4)*1.6
			fillRect(dst, math.Round(x+ox)-1, math.Round(y+oy)-1, 3, 3, rgb(0x8a5a3a))
		}
		fillRect(dst, math.Round(x)-2, math.Round(y)-2, 4, 4, rgb(0xf4a0c0))
		if u >= 0.5 && u < 0.9 {
			g.sceneBang(dst, "¡CLONC!", 560, 228, (u-0.5)/0.4, 0xff6a8a, 1.6)
		}
	}
	if t2 >= 10.6 && t2 < 13 { // the "turn it down" sign: a hand pushing the volume down
		f := math.Mod(t2*1.6, 1)
		hx, hy := 668.0, 286+f*26
		fillRect(dst, hx, hy, 16, 10, rgb(0xe0b088))
		fillRect(dst, hx, hy, 16, 3, rgb(0xf0c8a0))
		for j := 0; j < 3; j++ { // little arrows pointing down
			ay := hy + 18 + float64(j)*7
			for r := 0; r < 4; r++ {
				fillRect(dst, hx+4+float64(r), ay+float64(r), 8-2*float64(r), 1, rgba(0xff8a6a, uint8(220-j*60)))
			}
		}
		g.drawTextC(dst, "¡BAJA ESO!", hx+8, hy-22, rgba(0xff8a6a, 235), 1.3)
	}
	if t2 >= 13 && t2 < 15.2 { // he stares at the clock: it blinks and he cannot believe it
		if int(t2*4)%2 == 0 {
			fillRect(dst, bedSideTableX0+dx+2, bedSideTableTop-28, 52, 22, rgba(0xff3030, 90))
		}
		g.sceneBang(dst, "¡¿3:47?!", bedSideTableX0+dx+28, bedSideTableTop-44, sceneWindow(t2, 13, 15.2), 0xff6a6a, 1.5)
	}
}

// menuTwoSpeech: what they say (the bubble manager: at most 4, never the same sentence twice at once).
func (g *Game) menuTwoSpeech(dst *ebiten.Image, t, dy float64) {
	dx := bedShiftX
	ps := menuTwoPoses(t)
	h1x, h1y := bedHead1X+dx, twoBedCenterY+dy
	h2x, h2y := bedHead2X+dx, twoBedCenterY+dy
	sitX, sitY := bed2X0+60+dx, 250.0+dy
	bubbleReqs = bubbleReqs[:0]
	g.menuSay(0, h1x, h1y-6, menuListRight)
	sy := h2y - 6
	if ps[1].sit > 0.4 {
		h2x, sy = sitX+10, sitY-8
	}
	g.menuSay(1, h2x, sy, 700) // and off the window of the neighbours (it ends at x=696)
	g.flushMenuBubbles(dst, t)
}

// menuSay asks for player i's current sentence over (x, y); minX keeps the balloon off the list.
func (g *Game) menuSay(i int, x, y, minX float64) {
	g.bubble(i+7001, bubGuest, Vec{x, y}, menuLine(i, g.time), i == 1, 0)
	r := &bubbleReqs[len(bubbleReqs)-1]
	r.cx = x
	if minX > 0 {
		r.cx = math.Max(x, minX+r.w/2+2)
	}
	r.y = y - 44 - r.h
}

// flushMenuBubbles lets the manager choose among the two requests (no players to dodge in the menu).
func (g *Game) flushMenuBubbles(dst *ebiten.Image, t float64) {
	reqs := bubbleReqs
	bubbleReqs = bubbleReqs[:0]
	bubbleMgr.frame++
	bubbleMgr.extra = bubbleMgr.extra[:0]
	for _, r := range bubbleMgr.pick(reqs, float64(bubbleMgr.frame)/60, nil, screenW) {
		g.drawBubbleBox(dst, r)
	}
}
