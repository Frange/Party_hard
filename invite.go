package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ---------------------------------------------------------------------------
// The invitation to the island (every trophy won and the space station beaten).
//
// A letter arrives, it opens and it is read (the text stays: it never goes away by
// itself). When the player presses confirm or back (Enter, Esc, or the pad's A / B)
// the little device stuck to the paper counts down 5-4-3-2-1 and the letter blows up.
// Only then the next level (the island) begins.
// ---------------------------------------------------------------------------

const (
	invArrive = iota // the envelope drops in
	invWait          // the closed envelope waits for the player
	invOpen          // the seal cracks, the letter comes out
	invRead          // the letter, waiting for the player
	invCount         // the device counts down
	invBoom          // the explosion, and then the island
)

const (
	invArriveLen = 2.0
	invOpenLen   = 1.9
	invStepLen   = 0.45 // seconds per number of the countdown
	invSteps     = 5
	invBoomLen   = 2.6
)

// paper of the letter on the screen
const (
	invPX, invPY = 110.0, 30.0
	invPW, invPH = 740.0, 450.0
)

type inviteState struct {
	phase  int
	t      float64
	beeped int
	lock   float64 // after a press the next one is ignored for a moment (no accidental double press)
	boomed bool
	paper  *ebiten.Image // the letter, painted once with its text
	scale  float64       // text scale of the body (integer)
}

// invPara is one paragraph of the letter.
type invPara struct {
	text  string
	scale float64 // 0 = the body scale
	col   color.RGBA
	align int // 0 centre, 1 left, 2 right
	gap   float64
	maxW  float64 // 0 = the whole width of the paper
}

var (
	invInk   = rgb(0x2a2230)
	invRed   = rgb(0x8a1a14)
	invGreen = rgb(0x1c6a3a)
	invGold  = rgb(0x6a4a14)
)

// the text of the letter, in the order of the paper
func invParas() []invPara {
	return []invPara{
		{"IN GEN — INVITACIÓN EXCLUSIVA", 2, invRed, 0, 16, 0},
		{"Estimado/a invitado/a:", 0, invInk, 1, 8, 0},
		{"Prepárese para asistir a", 0, invInk, 1, 4, 0},
		{"LA FIESTA MÁS GRANDE QUE JAMÁS HAYA IMAGINADO", 0, invRed, 0, 10, 0},
		{"Un evento irrepetible en Isla Nublar, donde la diversión está garantizada... la supervivencia, no tanto.", 0, invInk, 1, 8, 0},
		{"Habrá música, comida, bebida y alguna que otra sorpresa prehistórica.", 0, invInk, 1, 10, 0},
		{"¡Le esperamos en Jurassic Park!", 0, invGreen, 0, 8, 0},
		{"— John Hammond", 0, invGold, 2, 12, 0},
		{"AVISO: Esta carta se autodestruirá en 5 segundos.", 0, invRed, 1, 0, 520},
	}
}

// invLine is a wrapped line ready to draw.
type invLine struct {
	s     string
	x, y  float64
	scale float64
	col   color.RGBA
}

// wrapText breaks s into lines of at most w pixels at the given (integer) scale.
func (g *Game) wrapText(s string, scale, w float64) []string {
	s = tr(s)
	var out []string
	line, cur := "", ""
	flush := func(word string) {
		t := word
		if line != "" {
			t = line + " " + word
		}
		if g.textW(t, scale) > w && line != "" {
			out = append(out, line)
			line = word
		} else {
			line = t
		}
	}
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				flush(cur)
			}
			cur = ""
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		flush(cur)
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}

// inviteLayout lays the letter out at the given body scale: the lines and the height they use.
func (g *Game) inviteLayout(body float64) ([]invLine, float64) {
	const margin, top = 46.0, 38.0
	w := invPW - 2*margin
	var lines []invLine
	y := top
	for _, p := range invParas() {
		sc := p.scale
		if sc == 0 {
			sc = body
		}
		lh := 9*sc + 5
		pw := w
		if p.maxW > 0 {
			pw = p.maxW // keeps the line clear of the device stuck to the paper
		}
		for _, l := range g.wrapText(p.text, sc, pw) {
			tw := g.textW(l, sc)
			x := margin
			switch p.align {
			case 0:
				x = (invPW - tw) / 2
			case 2:
				x = invPW - margin - tw
			}
			lines = append(lines, invLine{l, math.Round(x), math.Round(y), sc, p.col})
			y += lh
		}
		y += p.gap
	}
	return lines, y
}

// inviteScale picks the biggest whole text scale (2, else 1) at which the letter fits above the device.
func (g *Game) inviteScale() float64 {
	for _, sc := range []float64{2, 1} {
		if _, h := g.inviteLayout(sc); h <= invPH-30 {
			return sc
		}
	}
	return 1
}

// paintPaper draws the letter (paper and text) into its own image, once.
func (g *Game) paintPaper(iv *inviteState) {
	iv.scale = g.inviteScale()
	im := ebiten.NewImage(int(invPW), int(invPH))
	paper, edge := rgb(0xf2e6c2), rgb(0xcdb98a)
	im.Fill(edge)
	fillRect(im, 3, 3, invPW-6, invPH-6, paper)
	for y := 12.0; y < invPH-8; y += 18 { // the grain of the paper
		fillRect(im, 8, y, invPW-16, 1, rgba(0xd8c89c, 90))
	}
	// the letterhead: a little amber drop with a mosquito, as in the film
	filledEllipse(im, 40, 36, 14, 17, rgb(0xd88a10))
	filledEllipse(im, 40, 36, 10, 13, rgb(0xf0b030))
	fillRect(im, 39, 28, 2, 16, rgb(0x5a3a08))
	fillRect(im, 34, 33, 12, 2, rgb(0x5a3a08))
	lines, _ := g.inviteLayout(iv.scale)
	for _, l := range lines {
		g.drawInk(im, l.s, l.x, l.y, l.col, l.scale)
	}
	iv.paper = im
}

// drawInk is text without the drop shadow: crisp strokes of the same width on the paper.
func (g *Game) drawInk(dst *ebiten.Image, s string, x, y float64, c color.RGBA, scale float64) {
	scale = pxScale(scale)
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(math.Round(x), math.Round(y))
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, tr(s), g.face, op)
}

// ---- update -----------------------------------------------------------------------------------

// confirmed: Enter, Esc, or the pad's A / B (the actions of input.go).
func invitePressed() bool { return actConfirm.just() || actBack.just() }

// updateInvite runs the letter. It returns true while it has the control of the ending.
func (g *Game) updateInvite(e *endState) bool {
	iv := &e.x.inv
	press := invitePressed()
	if !press && invAutoAt > 0 { // debug: the invitation presses by itself
		switch iv.phase {
		case invWait:
			press = e.total > invAutoAt
		case invRead:
			press = iv.t > 1.0
		}
	}
	sounds, done := iv.step(press, 1.0/60)
	for _, s := range sounds {
		switch s {
		case "boom":
			g.snd.StopScene()
			g.snd.Play("boom")
			g.snd.Play("spark")
		default:
			g.snd.Play(s)
		}
	}
	if done {
		g.finishInvite(e)
	}
	return true
}

// step advances the letter by dt. Nothing moves on by itself except the animations: the closed
// envelope and the open letter wait for a press, and a press right after another one is ignored.
// It returns the sounds to play and whether the letter is over.
func (iv *inviteState) step(press bool, dt float64) (sounds []string, done bool) {
	iv.t += dt
	if iv.lock > 0 {
		iv.lock -= dt
	}
	pressed := false
	if press && iv.lock <= 0 {
		pressed = true
		iv.lock = 0.3
	}
	switch iv.phase {
	case invArrive:
		if iv.t > invArriveLen {
			iv.phase, iv.t = invWait, 0
		} else if pressed {
			iv.t = invArriveLen // hurry the envelope to its place
		}
	case invWait:
		if pressed {
			iv.phase, iv.t = invOpen, 0
			sounds = append(sounds, "ring")
		}
	case invOpen:
		if iv.t > invOpenLen {
			iv.phase, iv.t = invRead, 0
		} else if pressed {
			iv.t = invOpenLen // hurry the letter out
		}
	case invRead:
		if pressed && iv.t > 0.25 {
			iv.phase, iv.t, iv.beeped = invCount, 0, -1
		}
	case invCount:
		step := int(iv.t / invStepLen)
		if step != iv.beeped && step < invSteps {
			iv.beeped = step
			sounds = append(sounds, "alert")
		}
		if iv.t >= invStepLen*invSteps {
			iv.phase, iv.t = invBoom, 0
			iv.boomed = false
		}
	case invBoom:
		if !iv.boomed {
			iv.boomed = true
			sounds = append(sounds, "boom")
		}
		if iv.t > invBoomLen {
			done = true
		}
	}
	return sounds, done
}

// finishInvite: the letter is ashes; on with the island (or back to the list of scenes).
func (g *Game) finishInvite(e *endState) {
	if g.sceneReplay {
		g.leaveEnding()
		return
	}
	g.unlockJP()
	g.end = nil
	g.snd.StopScene()
	g.startLevel(specialJurassic)
}

// unlockJP makes the island a permanent, playable party.
func (g *Game) unlockJP() {
	if g.progress.Stats == nil {
		g.progress.Stats = map[string]int{}
	}
	if g.progress.Stats["jp_unlocked"] > 0 {
		return
	}
	g.progress.Stats["jp_unlocked"] = 1
	saveProgress(g.progress)
}

// jpListed: the island shows in the list of parties once it has been unlocked (or in DEBUG).
func (g *Game) jpListed() bool { return g.progress.Stats["jp_unlocked"] > 0 || debugSnap != nil }

// ---- drawing ----------------------------------------------------------------------------------

func easeOut(u float64) float64 { u = clampF(u, 0, 1); return 1 - (1-u)*(1-u) }

// a pseudo random number in 0..1 from two integers
func inviteRnd(a, b int) float64 { return jpHash(a, b, 77) }

func (g *Game) drawInvite(dst *ebiten.Image, e *endState) {
	iv := &e.x.inv
	if iv.paper == nil {
		g.paintPaper(iv)
	}
	dst.Fill(rgb(0x02060e))
	// a shake of the whole picture while it counts down and when it blows up
	var sx, sy float64
	switch iv.phase {
	case invCount:
		amp := 0.5 + 2.2*(iv.t/(invStepLen*invSteps))
		sx, sy = math.Round(math.Sin(e.total*61)*amp), math.Round(math.Cos(e.total*53)*amp)
	case invBoom:
		amp := 9 * math.Max(0, 1-iv.t/1.0)
		sx, sy = math.Round(math.Sin(e.total*71)*amp), math.Round(math.Cos(e.total*67)*amp)
	}
	switch iv.phase {
	case invArrive, invWait:
		g.drawEnvelope(dst, e, iv, 0, 0)
		a := 1.0
		if iv.phase == invArrive {
			a = math.Min(1, iv.t/0.8)
		}
		g.drawTextC(dst, "ESA MISMA NOCHE, EN EL BUZÓN...", 480, 70, rgba(0xf4f0ff, uint8(255*a)), 2)
		if iv.phase == invWait || iv.t > 1.2 {
			g.drawTextC(dst, "Una invitación sin remitente. Huele a selva.", 480, 372, rgb(0xe8c870), 2)
		}
	case invOpen:
		g.drawEnvelope(dst, e, iv, 1, easeOut(iv.t/invOpenLen))
	case invRead, invCount:
		g.drawLetter(dst, e, iv, sx, sy)
	case invBoom:
		g.drawBoom(dst, e, iv, sx, sy)
	}
	keys := actionHint(actConfirm) + "/" + actionHint(actBack)
	switch {
	case iv.phase == invWait:
		g.drawTextC(dst, keys+": abrir", 480, 470, rgba(0xe8e0f8, uint8(160+90*math.Abs(math.Sin(e.total*3)))), 2)
	case iv.phase == invRead && iv.t > 0.6:
		g.drawTextC(dst, keys+": continuar", 480, 500, rgba(0xe8e0f8, uint8(160+90*math.Abs(math.Sin(e.total*3)))), 1)
	}
	if g.scene2P() {
		g.inviteTwo(dst, e, iv)
	}
}

// drawEnvelope: mode 0 the envelope drops in, mode 1 it opens and the letter comes out.
func (g *Game) drawEnvelope(dst *ebiten.Image, e *endState, iv *inviteState, mode int, k float64) {
	cx, cy := 480.0, 250.0
	if mode == 0 {
		u := iv.t / invArriveLen
		if iv.phase == invWait {
			u = 1
		}
		drop := easeOut(u * 1.3)
		cy = -120 + 370*drop
		if u > 0.7 { // a little bounce when it lands
			cy -= math.Abs(math.Sin((u-0.7)*14)) * 14 * (1 - u)
		}
	}
	w, h := 260.0, 160.0
	fillRect(dst, cx-w/2+6, cy-h/2+8, w, h, rgba(0x000000, 90))
	fillRect(dst, cx-w/2, cy-h/2, w, h, rgb(0xe8dcb8))
	fillRect(dst, cx-w/2, cy-h/2, w, 3, rgb(0xfff6d8))
	fillRect(dst, cx-w/2, cy+h/2-3, w, 3, rgb(0xb8a878))
	if mode == 1 {
		// the letter slides out and grows into the paper
		px := mixf(cx-110, invPX, k)
		py := mixf(cy-40, invPY, k)
		pw := mixf(220, invPW, k)
		ph := mixf(120, invPH, k)
		dst.DrawImage(iv.paper, scaledTo(iv.paper, px, py, pw, ph))
		// the front of the envelope covers the bottom of the letter while it is small
		a := uint8(255 * (1 - k))
		fillRect(dst, cx-w/2, cy-h/2+h*0.45, w, h*0.55, rgba(0xdccfa4, a))
		g.drawFlap(dst, cx, cy-h/2, w, h*0.62, 1-math.Min(1, k*2.2)*2, 255-uint8(255*k))
		return
	}
	g.drawFlap(dst, cx, cy-h/2, w, h*0.62, 1, 255)
	g.drawSeal(dst, cx, cy-h/2+h*0.58, 1)
}

func mixf(a, b, t float64) float64 { return a + (b-a)*t }

// scaledTo is the draw option that puts an image into a rectangle.
func scaledTo(im *ebiten.Image, x, y, w, h float64) *ebiten.DrawImageOptions {
	b := im.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w/float64(b.Dx()), h/float64(b.Dy()))
	op.GeoM.Translate(x, y)
	return op
}

// drawFlap: the triangular flap of the envelope; open runs from 1 (shut, pointing down) to -1 (open, pointing up).
func (g *Game) drawFlap(dst *ebiten.Image, cx, top, w, h, open float64, alpha uint8) {
	col := rgba(0xc8b888, alpha)
	for i := 0.0; i < w/2; i += 1 {
		f := i / (w / 2)
		hh := h * (1 - f) * open
		x1, x2 := cx-w/2+i, cx+w/2-i
		if hh >= 0 {
			fillRect(dst, x1, top, 1, hh+1, col)
			fillRect(dst, x2-1, top, 1, hh+1, col)
		} else {
			fillRect(dst, x1, top+hh, 1, -hh+1, col)
			fillRect(dst, x2-1, top+hh, 1, -hh+1, col)
		}
	}
}

// drawSeal: the red wax seal with the amber drop.
func (g *Game) drawSeal(dst *ebiten.Image, cx, cy, a float64) {
	filledEllipse(dst, cx, cy, 17, 17, rgb(0x8a1a14))
	filledEllipse(dst, cx, cy, 12, 12, rgb(0xc03a2c))
	filledEllipse(dst, cx, cy, 5, 6, rgb(0xf0b030))
}

// drawLetter shows the paper with the device stuck to it.
func (g *Game) drawLetter(dst *ebiten.Image, e *endState, iv *inviteState, sx, sy float64) {
	fillRect(dst, invPX+sx+8, invPY+sy+10, invPW, invPH, rgba(0x000000, 110))
	dst.DrawImage(iv.paper, &ebiten.DrawImageOptions{GeoM: geoMTranslate(invPX+sx, invPY+sy)})
	count := -1 // not armed yet: a dead display and a slow light
	fast := 0.6
	if iv.phase == invCount {
		count = invSteps - int(iv.t/invStepLen)
		if count < 1 {
			count = 1
		}
		fast = 7
		// the paper chars from its edges while the device counts
		burn := iv.t / (invStepLen * invSteps)
		g.drawBurn(dst, invPX+sx, invPY+sy, burn)
	}
	led := math.Sin(e.total*fast*math.Pi) > 0
	g.drawBomb(dst, invPX+invPW-118+sx, invPY+invPH-78+sy, count, led, iv.phase == invCount)
	if iv.phase == invCount { // wisps of smoke from the device
		for i := 0; i < 6; i++ {
			f := math.Mod(e.total*1.2+float64(i)/6, 1)
			x := invPX + invPW - 90 + sx + math.Sin(f*6+float64(i))*10
			y := invPY + invPH - 92 + sy - f*60
			filledEllipse(dst, x, y, 3+f*7, 3+f*7, rgba(0x8a8a96, uint8(120*(1-f))))
		}
	}
}

// drawBurn: charred edges that creep in from the borders of the paper.
func (g *Game) drawBurn(dst *ebiten.Image, x, y, k float64) {
	d := 4 + 70*k
	col := rgba(0x2a1608, uint8(60+160*k))
	for i := 0.0; i < invPW; i += 6 {
		jit := inviteRnd(int(i), 1) * d * 0.6
		fillRect(dst, x+i, y, 6, d*0.5+jit, col)
		fillRect(dst, x+i, y+invPH-d*0.5-jit, 6, d*0.5+jit, col)
	}
	for j := 0.0; j < invPH; j += 6 {
		jit := inviteRnd(2, int(j)) * d * 0.6
		fillRect(dst, x, y+j, d*0.5+jit, 6, col)
		fillRect(dst, x+invPW-d*0.5-jit, y+j, d*0.5+jit, 6, col)
	}
}

// seven-segment digits for the countdown: a b c d e f g
var sevenSeg = [10][7]bool{
	{true, true, true, true, true, true, false},
	{false, true, true, false, false, false, false},
	{true, true, false, true, true, false, true},
	{true, true, true, true, false, false, true},
	{false, true, true, false, false, true, true},
	{true, false, true, true, false, true, true},
	{true, false, true, true, true, true, true},
	{true, true, true, false, false, false, false},
	{true, true, true, true, true, true, true},
	{true, true, true, true, false, true, true},
}

func drawDigit(dst *ebiten.Image, x, y float64, d int, on, off color.RGBA) {
	seg := [7][4]float64{{2, 0, 8, 2}, {10, 2, 2, 8}, {10, 11, 2, 8}, {2, 19, 8, 2}, {0, 11, 2, 8}, {0, 2, 2, 8}, {2, 9.5, 8, 2}}
	for i, s := range seg {
		c := off
		if d >= 0 && d < 10 && sevenSeg[d][i] {
			c = on
		}
		fillRect(dst, x+s[0], y+s[1], s[2], s[3], c)
	}
}

// drawBomb is the little self-destruct device: a battery pack with a red light, a seven segment display
// and wires that run into the paper.
func (g *Game) drawBomb(dst *ebiten.Image, x, y float64, count int, led, armed bool) {
	// the wires into the paper
	wires := []color.RGBA{rgb(0xd03030), rgb(0x3060d0), rgb(0xe0c030)}
	for i, c := range wires {
		wx := x + 6 + float64(i)*6
		for k := 0.0; k < 40; k += 2 {
			fillRect(dst, wx-math.Sin(k/6+float64(i))*6-k*0.4, y+30-k*0.9+float64(i)*2, 2, 2, c)
		}
	}
	fillRect(dst, x+4, y+4, 100, 62, rgb(0x24242e))
	fillRect(dst, x, y, 100, 62, rgb(0x4a4a58))
	fillRect(dst, x, y, 100, 3, rgb(0x7a7a8c))
	fillRect(dst, x, y+59, 100, 3, rgb(0x2a2a34))
	fillRect(dst, x+6, y+8, 48, 26, rgb(0x0a0a10)) // the display
	on, off := rgb(0xff3030), rgb(0x3a0c0c)
	if !armed {
		on = rgb(0xd02020)
	}
	drawDigit(dst, x+22, y+10, count, on, off)
	fillRect(dst, x+62, y+8, 32, 12, rgb(0xe08a20)) // the orange label
	fillRect(dst, x+64, y+11, 24, 2, rgb(0x2a1a08))
	fillRect(dst, x+64, y+15, 18, 2, rgb(0x2a1a08))
	lc := rgb(0x4a1010)
	if led {
		lc = rgb(0xff4040)
		filledEllipse(dst, x+82, y+40, 12, 12, rgba(0xff3030, 70))
	}
	filledEllipse(dst, x+82, y+40, 6, 6, rgb(0x2a0808))
	filledEllipse(dst, x+82, y+40, 4.5, 4.5, lc)
	fillRect(dst, x+8, y+42, 40, 3, rgb(0x2a2a34)) // a little speaker grille
	fillRect(dst, x+8, y+48, 40, 3, rgb(0x2a2a34))
	fillRect(dst, x+50, y-10, 2, 12, rgb(0x9a9aac)) // antenna
	filledEllipse(dst, x+51, y-11, 2.5, 2.5, rgb(0xff4040))
}

func geoMTranslate(x, y float64) ebiten.GeoM {
	var m ebiten.GeoM
	m.Translate(x, y)
	return m
}

// ---- the explosion -----------------------------------------------------------------------------

const boomCols, boomRows = 10, 6

// drawBoom: the flash, the shock wave, flying scraps of paper, sparks and smoke.
func (g *Game) drawBoom(dst *ebiten.Image, e *endState, iv *inviteState, sx, sy float64) {
	t := iv.t
	bx, by := invPX+invPW-68, invPY+invPH-48 // the device
	// the letter breaks into scraps that fly away from the device
	if t < 1.6 {
		cw, ch := invPW/boomCols, invPH/boomRows
		for r := 0; r < boomRows; r++ {
			for c := 0; c < boomCols; c++ {
				cx0, cy0 := invPX+float64(c)*cw+cw/2, invPY+float64(r)*ch+ch/2
				dx, dy := cx0-bx, cy0-by
				d := math.Hypot(dx, dy) + 1
				sp := (260 + 380*inviteRnd(c, r)) / d * 60
				x := cx0 + dx/d*sp*t*1.0 + math.Sin(float64(c*3+r))*10*t
				y := cy0 + dy/d*sp*t + 200*t*t*(0.4+inviteRnd(r, c))
				rot := t * (3 + 8*inviteRnd(c+9, r)) * float64((c+r)%2*2-1)
				a := 1 - math.Max(0, (t-0.9)/0.7)
				sub := iv.paper.SubImage(image.Rect(int(float64(c)*cw), int(float64(r)*ch), int(float64(c+1)*cw), int(float64(r+1)*ch))).(*ebiten.Image)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(-cw/2, -ch/2)
				op.GeoM.Rotate(rot)
				op.GeoM.Translate(x+sx, y+sy)
				burn := math.Min(1, t*1.4)
				op.ColorScale.Scale(float32(1-0.65*burn), float32(1-0.72*burn), float32(1-0.8*burn), float32(a))
				dst.DrawImage(sub, op)
			}
		}
	}
	// smoke
	for i := 0; i < 16; i++ {
		f := clampF((t-0.05*float64(i%8))/1.9, 0, 1)
		if f <= 0 {
			continue
		}
		ang := inviteRnd(i, 3) * 2 * math.Pi
		r := 20 + 90*f
		x := bx + math.Cos(ang)*r*0.9 + sx
		y := by + math.Sin(ang)*r*0.7 - 40*f + sy
		filledEllipse(dst, x, y, 16+36*f, 14+30*f, rgba(0x3a3a46, uint8(200*(1-f))))
	}
	// the fire ball
	if t < 0.55 {
		f := t / 0.55
		filledEllipse(dst, bx+sx, by+sy, 20+150*f, 16+120*f, rgba(0xfff0a0, uint8(255*(1-f))))
		filledEllipse(dst, bx+sx, by+sy, 10+100*f, 8+80*f, rgba(0xff8a20, uint8(255*(1-f*0.8))))
	}
	// the shock wave
	if t < 0.9 {
		ringAt(dst, bx+sx, by+sy, t*900, 6*(1-t/0.9)+1, rgba(0xfff6d8, uint8(220*(1-t/0.9))))
	}
	// sparks
	for i := 0; i < 70; i++ {
		ang := inviteRnd(i, 5) * 2 * math.Pi
		sp := 120 + 520*inviteRnd(i, 6)
		x := bx + math.Cos(ang)*sp*t + sx
		y := by + math.Sin(ang)*sp*t + 260*t*t + sy
		a := clampF(1-t/(0.5+inviteRnd(i, 8)), 0, 1)
		if a > 0 {
			fillRect(dst, x, y, 2, 2, rgba([]uint32{0xfff0a0, 0xffb030, 0xff6a20}[i%3], uint8(255*a)))
		}
	}
	// the flash and the fade to black
	if t < 0.35 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0xffffff, uint8(255*(1-t/0.35))))
	}
	if t > 1.1 {
		g.drawTextC(dst, "¡BUM! Y ahora, a la isla...", 480, 250, rgba(0xf8d030, uint8(255*clampF((t-1.1)/0.5, 0, 1))), 2)
	}
	if t > 1.9 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*clampF((t-1.9)/0.7, 0, 1))))
	}
}

// levelDoneHint: the line under "NIVEL COMPLETADO".
func levelDoneHint(g *Game) string {
	if g.levelIdx == specialJurassic {
		return "ENTER: volver a la lista de fiestas"
	}
	return "ENTER: siguiente fiesta"
}

// invAutoAt (debug): the second of the invitation at which "continue" is pressed by itself.
var invAutoAt float64
