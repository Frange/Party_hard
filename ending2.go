package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The two endings are different pieces of the same night.
//
//   normal ("¿ FINAL ?"): a tired walk home with a stray cat behind him, a yawn
//     now and then, the neighbour turns off his light, sheep in pyjamas and a quiet
//     credit roll.
//   secret ("FIN"): he comes back from the island by helicopter. T-Rex footprints on
//     the street, the ghost of every guest behind him in a conga (Guybrush, Sonic,
//     Shrek, a skeleton... and Murray on his stake), the door slammed in their faces,
//     the neighbour who plays the music of the T-Rex, sheep that are dinosaurs, a
//     very long credit roll and a scene after the credits.
//
// Both end the same way: the lights out, the crickets, not a single bass note.
// ---------------------------------------------------------------------------

// endExtra: what the invitation and the endings keep apart from the main state.
type endExtra struct {
	inv      inviteState
	cameos   []*CharSet
	murray   PropImg
	haveMur  bool
	built    bool
	cricketT float64 // time of the next chirp
	roared   bool
	pums     int
	camZ     float64 // the camera of the walk stage (for the balloons of the two player version)
	camOX    float64
	camOY    float64
	cloudTmp *ebiten.Image
}

var cameoNames = []string{"guybrush", "sonic", "shrek", "esqueleto_bailarin", "homer", "jack_rey_calabaza"}

// endBuild prepares the guests of the conga and Murray (once).
func (g *Game) endBuild(e *endState) {
	if e.x.built {
		return
	}
	e.x.built = true
	if !e.trueEnd {
		return
	}
	for i, n := range cameoNames {
		f := presetByName(n)
		if f == nil {
			continue
		}
		spec := f(rand.New(rand.NewSource(int64(31 + i))))
		spec.noOverride = true
		e.x.cameos = append(e.x.cameos, newCharSet(&spec))
	}
	if imgs := g.as.props["murray"]; len(imgs) > 0 {
		e.x.murray, e.x.haveMur = imgs[0], true
	}
	if _, ok := g.snd.sfx["rex_roar"]; !ok {
		g.snd.sfx["rex_roar"] = jrRoarSound()
	}
}

func init() {
	// the crickets of the quiet night: three quick chirps
	chirp := func(at float64) []float64 {
		return shift(mix2(tone(4300, 0.045, 0.16, 6, 0), shift(tone(4300, 0.045, 0.14, 6, 0), 0.075), shift(tone(4300, 0.045, 0.12, 6, 0), 0.15)), at)
	}
	cricketPCM = func() []byte { return mix(chirp(0), chirp(0.34)) }
}

var cricketPCM func() []byte

// cricket chirps now and then while the night is quiet.
func (g *Game) cricket(e *endState, quiet bool) {
	if !quiet {
		return
	}
	if _, ok := g.snd.sfx["cricket"]; !ok && cricketPCM != nil {
		g.snd.sfx["cricket"] = cricketPCM()
	}
	if e.total >= e.x.cricketT {
		e.x.cricketT = e.total + 0.9 + 0.8*jpHash(int(e.total*10), 3, 91)
		g.snd.Play("cricket")
	}
}

// ---- the street: extra gags (called from endWorld) ---------------------------------------

// endBehind draws what is seen behind the hero (before him).
func (g *Game) endBehind(c *ebiten.Image, camX, t float64, e *endState) {
	g.endBuild(e)
	if e.trueEnd {
		g.endFootprints(c, camX)
		if e.stage == endWalk && e.t < 8 {
			g.endHelicopter(c, t, e.t)
		}
		g.endConga(c, camX, t, e)
		return
	}
	g.endCat(c, camX, t, e)
}

// endFront draws what goes in front of the hero.
func (g *Game) endFront(c *ebiten.Image, camX, t float64, e *endState) {
	if e.stage == endWalk && !e.trueEnd {
		g.endYawn(c, camX, e)
	}
	if e.stage == endDoor {
		g.endNeighbour(c, camX, t, e)
	}
}

// footprints of the T-Rex on the road, the bones and an egg shell
func (g *Game) endFootprints(c *ebiten.Image, camX float64) {
	dark := rgba(0x08080e, 150)
	for k := 0; k < 14; k++ {
		wx := 90 + float64(k)*92
		sx := wx - camX
		if sx < -30 || sx > endCW+30 {
			continue
		}
		y := endGround + 30 + float64(k%2)*16
		filledEllipse(c, sx, y, 9, 5, dark)
		for _, d := range []float64{-7, 0, 7} {
			filledEllipse(c, sx+d, y-6, 2.6, 3.4, dark)
			fillRect(c, sx+d-1, y-9, 2, 2, rgba(0xd8d0b0, 140)) // the claws
		}
	}
	// the ribs of something that was a lamp post
	for _, wx := range []float64{330, 640} {
		sx := wx - camX
		if sx < -30 || sx > endCW+30 {
			continue
		}
		fillRect(c, sx, endGround+6, 2, 8, rgb(0xe8e0c8))
		for i := 0; i < 5; i++ {
			fillRect(c, sx+float64(i)*4, endGround+7-float64(i%2), 2, 7, rgb(0xe8e0c8))
		}
		fillRect(c, sx-2, endGround+13, 24, 2, rgb(0xcfc6aa))
	}
	// an egg that has been eaten
	sx := 520 - camX
	if sx > -20 && sx < endCW+20 {
		filledEllipse(c, sx, endGround+10, 7, 8, rgb(0xc8d8a0))
		fillRect(c, sx-6, endGround+5, 12, 3, rgb(0x070a1c))
		fillRect(c, sx-3, endGround+3, 2, 3, rgb(0xc8d8a0))
		fillRect(c, sx+2, endGround+2, 2, 4, rgb(0xc8d8a0))
	}
}

// the helicopter that dropped him: it crosses the sky and leaves
func (g *Game) endHelicopter(c *ebiten.Image, t, st float64) {
	x := 520 - st*95
	y := 34 + math.Sin(st*2)*3
	body := rgb(0x0a0c18)
	filledEllipse(c, x, y, 16, 8, body)
	fillRect(c, x+12, y-2, 26, 4, body)
	fillRect(c, x+36, y-8, 3, 10, body)
	fillRect(c, x-3, y-12, 2, 5, body)
	// the rotor spins
	w := 30 * math.Abs(math.Cos(t*28))
	fillRect(c, x-w, y-13, 2*w+1, 1, rgba(0x1c2036, 220))
	// the lights
	if int(t*4)%2 == 0 {
		fillRect(c, x-14, y+6, 2, 2, rgb(0xff3030))
	}
	g.glowAt(c, Vec{x, y + 14}, 18, color.RGBA{255, 240, 190, 36})
}

// ghostOf draws a guest as a ghost of the party: pale, a little transparent, floating.
func (g *Game) ghostOf(c *ebiten.Image, cs *CharSet, x float64, dir Dir, ph, alpha float64, anim int) {
	img, flip := cs.img(anim, dir, ph)
	if img == nil {
		return
	}
	bob := math.Sin(ph*1.7) * 2.5
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(charW, 0)
	}
	op.GeoM.Translate(math.Round(x-charCX), math.Round(endGround+11-charBase-5+bob))
	op.ColorScale.Scale(0.62, 0.82, 1, float32(alpha))
	filledEllipse(c, x, endGround+12, 8, 2.2, rgba(0x6a8aff, uint8(60*alpha)))
	c.DrawImage(img, op)
}

// the conga of the ghosts of every party behind him
func (g *Game) endConga(c *ebiten.Image, camX, t float64, e *endState) {
	if len(e.x.cameos) == 0 {
		return
	}
	var base float64
	switch e.stage {
	case endWalk:
		base = e.heroX + 26
	case endDoor:
		base = 129 + 26 + math.Max(0, 70-e.t*26)
		if e.t > 3.4 { // after the slam they melt away
			return
		}
	default:
		return
	}
	alpha := 0.62
	if e.stage == endWalk {
		alpha *= math.Min(1, e.t/2.5) // they appear behind him little by little
	}
	if e.stage == endDoor && e.t > 2.7 {
		alpha *= 1 - (e.t-2.7)/0.7
	}
	for i := len(e.x.cameos) - 1; i >= 0; i-- {
		x := base + float64(i)*25 - camX
		if x < -30 || x > endCW+30 {
			continue
		}
		g.ghostOf(c, e.x.cameos[i], x, DirLeft, t*6+float64(i)*0.9, alpha, AnimDance2)
	}
	// Murray on his stake, carried by the last two
	if e.x.haveMur {
		x := base + float64(len(e.x.cameos))*25 + 6 - camX
		if x > -30 && x < endCW+30 {
			bob := math.Sin(t*6) * 2
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(math.Round(x-e.x.murray.ox), math.Round(endGround+8-e.x.murray.oy+bob-6))
			op.ColorScale.Scale(0.8, 0.95, 1, float32(alpha+0.2))
			c.DrawImage(e.x.murray.img, op)
		}
	}
}

// a stray cat that follows the tired hero
func (g *Game) endCat(c *ebiten.Image, camX, t float64, e *endState) {
	if e.stage != endWalk && e.stage != endDoor {
		return
	}
	key := animalKey(animCat, 0, fWalk0+int(t*8)%4)
	sit := false
	x := e.heroX + 46 + math.Sin(t*1.3)*6 - camX
	if e.stage == endDoor {
		x = 160 - camX
		sit = true
	}
	if sit {
		key = animalKey(animCat, 0, fStand)
	}
	im := g.as.animals[key]
	if im == nil {
		return
	}
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(-1, 1) // the sprite faces right and he walks left
	op.GeoM.Translate(w, 0)
	op.GeoM.Translate(math.Round(x-w/2), math.Round(endGround+12-h+2))
	fillShadow(c, x, endGround+12)
	c.DrawImage(im, op)
}

// yawns of the hero (normal ending): a big "Z" and "AAAH" now and then
func (g *Game) endYawn(c *ebiten.Image, camX float64, e *endState) {
	ph := math.Mod(e.t, 4.2)
	if ph > 1.4 {
		return
	}
	x := e.heroX - camX
	a := uint8(255 * math.Sin(ph/1.4*math.Pi))
	g.drawText(c, "AAAH", x-14, endGround-52-ph*8, rgba(0xf4f0ff, a), 1)
	g.drawText(c, "z", x+8, endGround-42-ph*10, rgba(0xa8b8ff, a), 1)
}

// the neighbour of the door: normal = turns the light off; secret = plays the T-Rex music, until the T-Rex arrives
func (g *Game) endNeighbour(c *ebiten.Image, camX, t float64, e *endState) {
	nx := 232 - camX
	if nx < -90 || nx > endCW+10 {
		return
	}
	fillRect(c, nx, endGround-70, 84, 70, rgb(0x2a2a40))
	fillRect(c, nx-4, endGround-80, 92, 12, rgb(0x4a2a38))
	fillRect(c, nx-4, endGround-80, 92, 2, rgb(0x7a4a58))
	fillRect(c, nx+30, endGround-26, 14, 26, rgb(0x1a1620))
	wx, wy, ww, wh := nx+10, endGround-58, 30.0, 24.0
	lit := e.t < 3.4 // the window of the neighbour
	if e.trueEnd {
		lit = e.t < 8.0
	}
	col := rgb(0x161a2c)
	if lit {
		col = mixc(rgb(0xe8c070), rgb(0xff60c0), math.Abs(math.Sin(t*4))*0.7)
	}
	fillRect(c, wx, wy, ww, wh, col)
	if lit {
		g.glowAt(c, Vec{wx + ww/2, wy + wh/2}, 34, color.RGBA{255, 140, 200, 70})
	}
	fillRect(c, wx+ww/2, wy, 1, wh, rgb(0x2a2438))
	fillRect(c, wx, wy+wh/2, ww, 1, rgb(0x2a2438))
	if !e.trueEnd {
		return
	}
	// the speaker in the window that jumps to the bass; the music of the T-Rex
	if lit && e.t > 3.8 {
		k := math.Abs(math.Sin(t * 15.5))
		filledEllipse(c, wx+ww/2, wy+wh/2, 8+3*k, 8+3*k, rgb(0x14141c))
		filledEllipse(c, wx+ww/2, wy+wh/2, 4+2*k, 4+2*k, rgb(0x3a3a50))
		if int(t*7.75)%2 == 0 && int(t*7.75) != e.x.pums {
			e.x.pums = int(t * 7.75)
			g.snd.Play("thud")
		}
	}
	// the T-Rex head in the window and... silence
	if e.t > 6.3 {
		k := clampF((e.t-6.3)/1.2, 0, 1)
		hx, hy := wx+ww/2+30*(1-k), wy+wh/2+6
		filledEllipse(c, hx, hy, 15, 10, rgb(0x2a3a1c))
		fillRect(c, hx-4, hy+3, 20, 5, rgb(0x2a3a1c))
		fillRect(c, hx-1, hy-3, 3, 3, rgb(0xf0e060))
		for q := 0.0; q < 16; q += 4 {
			fillRect(c, hx-2+q, hy+7, 2, 3, rgb(0xf4f0e0))
		}
	}
	if e.t > 6.0 && !e.x.roared {
		e.x.roared = true
		g.snd.Play("rex_roar")
	}
}

// ---- credits ----------------------------------------------------------------------------------------

func endCreditRows(secret bool) [][2]string {
	rows := [][2]string{
		{"PARTY HARD", "Fiesta Nocturna"},
		{"", ""},
		{"Director", "Frange"},
		{"IA", "Claude"},
		{"Testers", "Diego, César y Samuel"},
		{"Año", "2026"},
		{"", ""},
	}
	if !secret {
		return append(rows,
			[2]string{"Vecinos molestos", "Todos. Ya no."},
			[2]string{"Música del vecino", "Silenciada de por vida"},
			[2]string{"Ovejas contadas", "4.000 (una iba en pijama)"},
			[2]string{"Sueño recuperado", "Al 100%"},
			[2]string{"Fiestas arruinadas", "Las 21. Qué pena, qué pena"},
			[2]string{"", ""},
		)
	}
	return append(rows,
		[2]string{"Cameo no solicitado", "Un T-Rex con muy mala educación"},
		[2]string{"Pirata poderoso (poco)", "Guybrush Threepwood"},
		[2]string{"Calavera parlante", "Murray (sigue hablando)"},
		[2]string{"Erizo sin carnet", "Sonic, multado por exceso de velocidad"},
		[2]string{"Ogro de pantano", "Shrek, se ha comido el catering"},
		[2]string{"Esqueleto bailarín", "Sin contrato ni nómina"},
		[2]string{"Altavoces destruidos", "1.847 (y uno de los dientes)"},
		[2]string{"Dinosaurios heridos", "Ninguno. Los invitados, otra cosa"},
		[2]string{"Invitados fantasma", "Todos. No pagan, pero bailan"},
		[2]string{"Vecino con música jurásica", "En paz descanse (su altavoz)"},
		[2]string{"Sueño recuperado", "Por fin, al 1000%"},
		[2]string{"", ""},
	)
}

// ---- the scene after the credits (secret ending) ------------------------------------------------

func (g *Game) drawStinger(dst *ebiten.Image, e *endState) {
	g.endBuild(e)
	dst.Fill(rgb(0x02030a))
	t := e.t
	if t < 2.4 { // the card
		a := uint8(255 * math.Sin(clampF(t/2.4, 0, 1)*math.Pi))
		g.drawTextC(dst, "Tras los créditos...", 480, 250, rgba(0xf4f0ff, a), 2)
		return
	}
	k := clampF((t-2.4)/1.0, 0, 1)
	// the window of his bedroom, from outside, in the dark
	wx, wy, ww, wh := 330.0, 90.0, 300.0, 250.0
	fillRect(dst, wx-14, wy-14, ww+28, wh+28, rgb(0x2a2030))
	fillRect(dst, wx, wy, ww, wh, mixc(rgb(0x070a1c), rgb(0x1a2a58), 0.4))
	filledEllipse(dst, wx+ww-60, wy+56, 24, 24, rgb(0xeae6d4))
	filledEllipse(dst, wx+ww-68, wy+50, 22, 22, rgb(0x16264c))
	fillRect(dst, wx+ww/2-2, wy, 4, wh, rgb(0x2a2030)) // the frame
	fillRect(dst, wx, wy+wh/2-2, ww, 4, rgb(0x2a2030))
	// he sleeps: a lump under the blanket (seen through the glass in the dark)
	fillRect(dst, wx+20, wy+wh-34, 120, 30, rgb(0x1a1830))
	filledEllipse(dst, wx+56, wy+wh-38, 16, 9, rgb(0x20203a))
	g.drawText(dst, "z", wx+84, wy+wh-70-math.Mod(t*8, 22), rgba(0xa8b8ff, uint8(200-math.Mod(t*8, 22)*8)), 2)
	// Murray's stake outside, bottom right
	if e.x.haveMur {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(3, 3)
		op.GeoM.Translate(math.Round(wx+ww+40-e.x.murray.ox*3), math.Round(wy+wh+10-e.x.murray.oy*3))
		dst.DrawImage(e.x.murray.img, op)
	}
	// the little head of the T-Rex that comes to the window, sniffs and goes
	if t > 3.6 {
		u := clampF((t-3.6)/1.6, 0, 1)
		out := clampF((t-6.4)/1.2, 0, 1)
		hx := wx + ww + 80 - 190*easeOut(u) + 190*out*out
		hy := wy + 110 + math.Sin(t*3)*6
		filledEllipse(dst, hx, hy, 52, 36, rgb(0x2a4a1c))
		fillRect(dst, hx-40, hy+8, 80, 16+4*math.Abs(math.Sin(t*5)), rgb(0x2a4a1c))
		fillRect(dst, hx-10, hy-14, 10, 9, rgb(0xf0e060))
		fillRect(dst, hx-6, hy-12, 3, 5, rgb(0x14140a))
		fillRect(dst, hx-46, hy-6, 4, 3, rgb(0x14140a))
		for q := 0.0; q < 48; q += 8 {
			fillRect(dst, hx-34+q, hy+22, 3, 5, rgb(0xf4f0e0))
		}
		if t > 4.4 && t < 6.2 {
			g.drawTextC(dst, "RAWR?", hx, hy-64, rgb(0x8ae070), 2)
		}
	}
	if t > 5.6 && t < 8.2 {
		g.drawTextC(dst, "Murray: ¿Alguien me baja de aquí?", 480, 440, rgb(0xf0e8d0), 1)
	}
	if t > 4.0 && !e.x.roared {
		e.x.roared = true
		g.snd.Play("rex_roar")
	}
	if e.t < 6.4 {
		g.cricket(e, true)
	}
	fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-k))))
	if t > 7.5 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*clampF((t-7.5)/1.0, 0, 1))))
	}
}

// walkCaption: what the narrator says while he walks home and at the door.
func (g *Game) walkCaption(e *endState) string {
	if g.scene2P() {
		return g.walkCaption2(e)
	}
	if !e.trueEnd {
		switch {
		case e.stage == endDoor && e.t < 3:
			return "Llave, puerta, luz de la entrada... Qué silencio más bonito."
		case e.stage == endDoor && e.t < 4.5:
			return "Hasta el vecino ha apagado la luz."
		case e.stage == endDoor:
			return "Ni un bajo. Ni una voz. Ni un solo 'otra más'."
		case e.t < 4.8:
			return "Después de acabar con todas las fiestas..."
		case e.t < 9.5:
			return "La ciudad, por fin, en silencio. Sólo se oye un globo desinflándose."
		default:
			return "Un gato callejero se ha apuntado a la comitiva. Todos queremos dormir."
		}
	}
	switch {
	case e.stage == endDoor && e.t < 2.6:
		return "Llave, puerta... y detrás de él, la mitad de la fiesta."
	case e.stage == endDoor && e.t < 4.6:
		return "¡Portazo! Los invitados fantasma se quedan fuera."
	case e.stage == endDoor && e.t < 6.2:
		return "El vecino pone la Música Jurásica a todo volumen..."
	case e.stage == endDoor && e.t < 8.2:
		return "...y alguien muy grande ha oído el bajo."
	case e.stage == endDoor:
		return "Silencio. Un altavoz menos en el barrio."
	case e.t < 5:
		return "Un helicóptero le deja en casa y se va corriendo. Con razón."
	case e.t < 10:
		return "Huellas de T-Rex por toda la calle. El barrio está de lo más animado."
	case e.t < 15.5:
		return "Los invitados de todas las fiestas le siguen en forma de fantasma. Hacen la conga."
	default:
		return "Hasta Murray viene en su estaca, hablando sin parar."
	}
}

func endCreditSpeed(secret bool) float64 { return 27 }

// bedCaption: what the narrator says in the bed.
func (g *Game) bedCaption(e *endState) string {
	if g.scene2P() {
		return g.bedCaption2(e)
	}
	end := endLenFor(endBed, e.trueEnd)
	if !e.trueEnd {
		switch {
		case e.t < 5:
			return "Cama. Almohada. Mantita. Ni un solo bajo de graves."
		case e.t < 12:
			return "A ver... voy a contar ovejitas."
		case e.t < 19:
			return "Hasta las ovejas se han puesto el pijama."
		case e.t < end-9:
			return "...tres mil... cuatro mil... ¡qué sueño!"
		default:
			return "Grillos. Sólo grillos. Zzz..."
		}
	}
	switch {
	case e.t < 5:
		return "Cama. Almohada. Mantita. Y un T-Rex lejano que ya no ruge."
	case e.t < 11:
		return "A contar ovejitas... una... dos..."
	case e.t < 17:
		return "¡Esa oveja tiene dientes! Y la siguiente es verde."
	case e.t < 21:
		return "La última oveja es un T-Rex. Por supuesto."
	case e.t < 25:
		return "¿Pum... pum...? Otra vez no. Otra vez no."
	case e.t < 28:
		return "¡RAAAWR! ...y se acabó el altavoz del vecino."
	default:
		return "Silencio. Sólo grillos. Zzz..."
	}
}

// bedGags: the bass that comes back for a moment (secret ending), the roar that ends it, and the crickets.
func (g *Game) bedGags(dst *ebiten.Image, e *endState) {
	if !e.trueEnd {
		return
	}
	g.endBuild(e)
	for i, at := range []float64{21.6, 22.4, 23.2, 24.0} {
		u := e.t - at
		if u >= 0 && u < 0.5 {
			if int(e.x.pums) <= i && u < 0.05 {
				e.x.pums = i + 1
				g.snd.Play("thud")
			}
			g.drawTextC(dst, "PUM", 480+math.Sin(float64(i)*4)*140, 150+float64(i%2)*40-u*30, rgba(0xff8ad0, uint8(255*(1-u/0.5))), 4)
		}
	}
	if e.t >= 25.6 && !e.x.roared {
		e.x.roared = true
		g.snd.Play("rex_roar")
	}
	if e.t >= 25.6 && e.t < 27.8 {
		sh := math.Sin(e.t*70) * 3
		g.drawTextC(dst, "RAAAWR", 480+sh, 190, rgba(0xff5050, uint8(255*(1-clampF((e.t-27)/0.8, 0, 1)))), 7)
	}
}

// sheepExtra decorates the sheep of the ending: pyjamas for the normal one, dinosaur scales for the secret one.
func (g *Game) sheepExtra(dst *ebiten.Image, x, y float64, k int, secret bool, a uint8) {
	al := func(c color.RGBA) color.RGBA { c.A = a; return c }
	if !secret {
		for i := -2; i <= 2; i++ { // pyjama stripes
			fillRect(dst, x+float64(i)*5, y-6, 2, 9, al(rgb(0x6a8ae8)))
		}
		// night cap with a bobble
		fillRect(dst, x+12, y-11, 8, 4, al(rgb(0xd04858)))
		fillRect(dst, x+16, y-14, 4, 4, al(rgb(0xd04858)))
		filledEllipse(dst, x+21, y-14, 2.2, 2.2, al(rgb(0xfff0f0)))
		if k == 5 { // this one yawns
			filledEllipse(dst, x+21, y+2, 3, 3, al(rgb(0x2a1218)))
		}
		return
	}
	green := rgb(0x4ab04a)
	if k%3 == 2 {
		green = rgb(0x8ab030)
	}
	for i := -2; i <= 2; i++ { // a row of spikes along the back
		fillRect(dst, x+float64(i)*5-1, y-11-float64(2-absI(i)), 3, float64(7-absI(i)), al(green))
	}
	fillRect(dst, x-18, y-1, 10, 4, al(green)) // a tail
	if k == 6 {                                // the sheep that is a T-Rex
		filledEllipse(dst, x+19, y-3, 12, 9, al(rgb(0x3a8a30)))
		fillRect(dst, x+17, y+2, 14, 5, al(rgb(0x3a8a30)))
		fillRect(dst, x+21, y-6, 3, 3, al(rgb(0xf0e060)))
		for q := 0; q < 4; q++ {
			fillRect(dst, x+19+float64(q)*3, y+7, 2, 3, al(rgb(0xf4f0e0)))
		}
	}
}

func absI(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// nextEndStage is the order of the pieces: the secret ending has a scene after the credits.
func nextEndStage(stage int, secret bool) int {
	switch stage {
	case endWalk:
		return endDoor
	case endDoor:
		return endBed
	case endBed:
		return endCredits
	case endCredits:
		if secret {
			return endStinger
		}
		return endFinal
	}
	return endFinal
}
