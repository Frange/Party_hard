package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The invitation for two players: one letter ("Estimados invitados") for the two of them, who watch it from
// the margins of the screen, each as he is: the blue one stands still, reads and counts calmly; the
// red one hops about, tears at the seal, asks who is writing, wants to throw the letter away when it
// says it will blow up. Either of them can press to go on. When it explodes they both come out black
// with soot: the blue one says "vaya", the red one has a lot to say.
// ---------------------------------------------------------------------------

// inviteTwoPose: what each of the two does in a phase: animation, phase of the animation and a hop in pixels.
func inviteTwoPose(i int, phase int, t, total float64) (anim int, ph, hop float64) {
	anim, ph = AnimIdle, 0
	if i == 0 { // the patient one: he hardly moves
		switch phase {
		case invOpen:
			anim, ph = AnimIdle, 0
		case invRead:
			anim, ph = AnimTalk, total*3 // he reads aloud, murmuring
		case invCount:
			anim, ph = AnimIdle, 0
		case invBoom:
			anim, ph = AnimTired, total*2
		}
		return
	}
	switch phase { // the nervous one
	case invArrive:
		anim, ph, hop = AnimWave, total*7, math.Abs(math.Sin(total*6))*6
	case invWait:
		anim, ph, hop = AnimPanic, total*9, math.Abs(math.Sin(total*8))*8 // shaking with impatience
	case invOpen:
		anim, ph = AnimPunch, total*9 // tearing at the seal
	case invRead:
		anim, ph, hop = AnimWave, total*8, math.Abs(math.Sin(total*5))*5
	case invCount:
		anim, ph, hop = AnimPanic, total*12, math.Abs(math.Sin(total*14))*10 // frantic
	case invBoom:
		anim, ph, hop = AnimPanic, total*10, math.Abs(math.Sin(total*12))*9
	}
	return
}

// inviteTwoLines: what each of them says in each phase of the letter (the first column is the blue one's).
func inviteTwoLine(i int, iv *inviteState) string {
	t := iv.t
	switch iv.phase {
	case invWait:
		if i == 1 && t > 0.8 {
			return "¿Quién escribe? ¡Ábrela ya!"
		}
		if i == 0 && t > 2.2 {
			return "Con calma. Huele a selva."
		}
	case invOpen:
		if i == 1 {
			return "¡Rompe, rompe!"
		}
		return "Con cuidado..."
	case invRead:
		switch {
		case i == 0 && t < 3.6:
			return "Interesante."
		case i == 1 && t > 1.6 && t < 4.2:
			return "¿¿Dinosaurios?? ¡Yo voy!"
		case i == 0 && t > 4.4 && t < 7:
			return "Dice que explota."
		case i == 1 && t > 7:
			return "¡¡NO!! ¡¡TÍRALA!!"
		}
	case invCount:
		if i == 0 && t < 1.2 {
			return "Cinco... cuatro..."
		}
		if i == 1 {
			return "¡¡CORRED!!"
		}
	case invBoom:
		if i == 0 && t > 1.0 {
			return "Vaya."
		}
		if i == 1 && t > 0.9 {
			return "¡¡PERO QUÉ!!"
		}
	}
	return ""
}

func (g *Game) inviteTwo(dst *ebiten.Image, e *endState, iv *inviteState) {
	cs := [2]*CharSet{g.sceneHero(0, false), g.sceneHero(1, false)}
	if cs[0] == nil || cs[1] == nil {
		return
	}
	xs := [2]float64{56, 904}
	const y = 506.0
	sc := 2.0
	// the blast throws them back a little, and the soot turns them black
	var tint [3]float32
	for i := 0; i < 2; i++ {
		anim, ph, hop := inviteTwoPose(i, iv.phase, iv.t, e.total)
		tint = sceneTint(i, cs[0], cs[1])
		x := xs[i]
		lean := 0.0
		if iv.phase == invBoom {
			k := clampF(iv.t/0.5, 0, 1)
			tint = [3]float32{1 - 0.7*float32(k), 1 - 0.72*float32(k), 1 - 0.75*float32(k)}
			if i == 0 {
				x -= 8 * k
			} else {
				x += 8 * k
			}
			lean = 0.16 * k * float64(1-2*i)
		}
		dir := DirRight
		if i == 1 {
			dir = DirLeft
		}
		if iv.phase == invCount && i == 1 { // he runs at the letter and back, never arriving
			x -= 34 * math.Abs(math.Sin(iv.t*2.4))
			anim, ph = AnimRun, e.total*12
		}
		g.sceneChar(dst, cs[i], anim, dir, ph, x, y, sc, 1, hop, lean, tint)
		if iv.phase == invBoom && iv.t > 0.4 { // smoke from the hair
			for k := 0; k < 3; k++ {
				age := math.Mod(e.total*0.9+float64(k)/3, 1)
				filledEllipse(dst, x+math.Sin(age*6+float64(k))*6, y-46*sc-hop-age*34, 3+age*7, 3+age*6, rgba(0x3a3a46, uint8(180*(1-age))))
			}
		}
		if s := inviteTwoLine(i, iv); s != "" {
			g.speechBubble(dst, x, y-46*sc-hop-4, s)
		}
	}
}
