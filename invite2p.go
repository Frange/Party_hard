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

// invTalk is one balloon of the 2 player letter: who says it (0 blue, 1 red), when (seconds since the phase
// began, [from, to)) and what. Only ONE balloon is on at a time (the turns never overlap), each one goes
// out when its time is over, and they all live in the strip under the paper (see inviteBubbleRect).
type invTalk struct {
	who      int
	from, to float64
	s        string
}

// invTalks: the dialogue of each phase. cycle > 0 repeats the phase's lines every `cycle` seconds (the
// closed envelope and the open letter wait for the players for ever).
var invTalks = map[int]struct {
	cycle float64
	l     []invTalk
}{
	invWait: {10, []invTalk{
		{1, 0.8, 3.8, "¿Quién escribe? ¡Ábrela ya!"},
		{0, 4.2, 7.2, "Con calma. Huele a selva."},
	}},
	invOpen: {0, []invTalk{
		{1, 0.0, 0.9, "¡Rompe, rompe!"},
		{0, 0.95, 1.95, "Con cuidado..."},
	}},
	invRead: {0, []invTalk{
		{0, 0.4, 2.4, "Interesante."},
		{1, 2.6, 5.0, "¿¿Dinosaurios?? ¡Yo voy!"},
		{0, 5.2, 7.6, "Dice que explota."},
		{1, 7.8, 10.8, "¡¡NO!! ¡¡TÍRALA!!"},
	}},
	invCount: {0, []invTalk{
		{0, 0.0, 1.0, "Cinco... cuatro..."},
		{1, 1.05, 2.3, "¡¡CORRED!!"},
	}},
	invBoom: {0, []invTalk{
		{1, 0.8, 1.7, "¡¡PERO QUÉ!!"},
		{0, 1.8, 2.7, "Vaya."},
	}},
}

// inviteTwoSpeaker: who is speaking now and what (-1 and "" if nobody).
func inviteTwoSpeaker(phase int, t float64) (int, string) {
	d, ok := invTalks[phase]
	if !ok {
		return -1, ""
	}
	if d.cycle > 0 {
		t = math.Mod(t, d.cycle)
	}
	for _, l := range d.l {
		if t >= l.from && t < l.to {
			return l.who, l.s
		}
	}
	return -1, ""
}

// Balloons of the letter live in the strip under the paper (it ends at y 480): the blue one's starts next
// to him on the left and the red one's ends next to him on the right, so they never cover the letter, the
// captions, each other or the "continue" hint in the middle.
const (
	invBubY     = 488.0
	invBubH     = 30.0
	invBubLeft  = 98.0
	invBubRight = 826.0
)

// inviteBubbleRect: the box (without its outline) of balloon s said by who.
func (g *Game) inviteBubbleRect(who int, s string) (x, y, w, h float64) {
	w = math.Round(g.textW(s, 2)) + 24
	x = invBubLeft
	if who == 1 {
		x = invBubRight - w
	}
	return x, invBubY, w, invBubH
}

// inviteBubble draws balloon s of player who with its tail towards him.
func (g *Game) inviteBubble(dst *ebiten.Image, who int, s string) {
	x, y, w, h := g.inviteBubbleRect(who, s)
	fillRect(dst, x-2, y-2, w+4, h+4, rgb(0x14121c))
	fillRect(dst, x, y, w, h, rgb(0xfffaf0))
	ty := y + h/2 - 3
	if who == 0 {
		fillRect(dst, x-6, ty, 6, 6, rgb(0x14121c))
		fillRect(dst, x-5, ty+1, 6, 4, rgb(0xfffaf0))
	} else {
		fillRect(dst, x+w, ty, 6, 6, rgb(0x14121c))
		fillRect(dst, x+w-1, ty+1, 6, 4, rgb(0xfffaf0))
	}
	g.drawTextC(dst, s, x+w/2, y-1, rgb(0x14102a), 2)
}

func (g *Game) inviteTwo(dst *ebiten.Image, e *endState, iv *inviteState) {
	defer func() { // the balloon is the last thing drawn: nothing covers it
		if who, s := inviteTwoSpeaker(iv.phase, iv.t); who >= 0 {
			g.inviteBubble(dst, who, s)
		}
	}()
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
	}
}
