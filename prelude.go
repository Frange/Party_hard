package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The preludes: the action of the LAST scene, before the letter or the ending.
//
// What happens in the last place of each road is different every time:
//   preLetter (space station, every trophy): the heroes float to the end of the party, see a
//     glowing letter on the floor, fight over it (zero gravity: it escapes to the ceiling), catch it
//     and open it. Then the letter of invite.go comes up, exactly as it was.
//   preBad (space station, trophies missing): the party has used up the oxygen. Red alarm, a panic,
//     everybody runs to the escape pod, only the heroes fit. Then the short epilogue of ending.go.
//   preIsland (Isla Nublar, after the last party): the rented helicopter is a wreck, raptors in
//     party hats come out of the palms, the T-Rex arrives... and fires a confetti cannon. Then the
//     party of endparty.go (the real ending).
// Everything is drawn into a 480x270 canvas and shown at 2x; the texts are drawn on the screen.
// ---------------------------------------------------------------------------

const (
	preLetter = iota
	preBad
	preIsland
)

// endStageLetter: the value of -endstage that plays the prelude of the letter (the station) on its own.
const endStageLetter = 10

const (
	preLetterLen = 17.0
	preBadLen    = 12.6
	preIslandLen = 15.5
)

func preludeLen(kind int) float64 {
	switch kind {
	case preLetter:
		return preLetterLen
	case preIsland:
		return preIslandLen
	}
	return preBadLen
}

type preludeState struct {
	kind   int
	length float64
	built  bool
	astro  *CharSet
	alien  *CharSet
	done   [8]bool // one-shot sounds
}

// enterPrelude turns the ending that has just been started into its prelude.
func (g *Game) enterPrelude(kind int) {
	e := g.end
	if e == nil {
		return
	}
	e.stage, e.t = endPrelude, 0
	e.x.pre = preludeState{kind: kind, length: preludeLen(kind)}
}

// endLeavePrelude: on with what the prelude leads to (the letter, the epilogue or the party).
func (g *Game) endLeavePrelude(e *endState) {
	next := nextEndStage(endPrelude, e.trueEnd)
	if e.x.pre.kind == preLetter {
		next = endInvite
	}
	e.stage, e.t = next, 0
	e.skipLock = 0.3
}

func preMusicVol(e *endState) float64 {
	if e.x.pre.kind == preBad { // the alarm is the music: the tune is only a murmur
		return 0.3 * math.Min(1, e.t/1.5)
	}
	return 0.5 * math.Min(1, e.t/1.5) // it goes on into what follows
}

// castSet builds the animated look of a preset by its name (famous or archetype), or nil.
func castSet(name string, seed int64) *CharSet {
	f := presetByName(name)
	if f == nil {
		for _, a := range archetypePresets {
			if a.Name == name {
				f = a.F
				break
			}
		}
	}
	if f == nil {
		return nil
	}
	spec := f(rand.New(rand.NewSource(seed)))
	spec.noOverride = true
	return newCharSet(&spec)
}

// preBuild makes the guest stars of the preludes (once).
func (g *Game) preBuild(e *endState) {
	p := &e.x.pre
	if p.built {
		return
	}
	p.built = true
	p.astro = castSet("astronauta", 5)
	p.alien = castSet("alien_turista", 6)
	g.sfxOnce("rex_roar", jrRoarSound)
}

// ---- who is where ------------------------------------------------------------------------------

// prePose: where an actor of a prelude is and what it does.
type prePose struct {
	x, y, hop, lean float64
	alpha           float64
	anim            int
	dir             Dir
	ph              float64
	hide            bool
}

func newPose(x, y float64, anim int, dir Dir) prePose {
	return prePose{x: x, y: y, alpha: 1, anim: anim, dir: dir}
}

// preLine: a balloon of a prelude. who: 0 the first hero, 1 the second one (only with two players), 2 the
// third actor (astronaut or raptor), 3 the loudspeaker or the T-Rex (see each scene).
type preLine struct {
	who  int
	a, b float64
	s    string
	two  bool // only with two players
}

const preFloorY = 214.0

// preActors gives the three actors of a prelude at time t.
func preActors(kind int, t float64, two bool) (h1, h2, o prePose) {
	switch kind {
	case preLetter:
		return preLetterPoses(t, two)
	case preIsland:
		return preIslandPoses(t, two)
	}
	return preBadPoses(t, two)
}

func hopArc(u, period, height float64) float64 {
	if u < 0 {
		return 0
	}
	return height * math.Abs(math.Sin(u*math.Pi/period))
}

func preLetterPoses(t float64, two bool) (h1, h2, as prePose) {
	h1 = newPose(0, preFloorY, AnimIdle, DirRight)
	switch {
	case t < 3: // floating strides in the low gravity
		h1.x = -24 + 154*t/3
		h1.anim, h1.ph, h1.hop = AnimWalk, t*7, 7*math.Abs(math.Sin(t*3.1))
	case t < 5.6:
		h1.x = 130
		if t > 3.6 {
			h1.anim, h1.ph = AnimPoint, (t-3.6)*3
		}
	case t < 7.4:
		h1.x = 130 + 132*arrSmooth((t-5.6)/1.8)
		h1.anim, h1.ph, h1.hop = AnimWalk, t*7, 5*math.Abs(math.Sin(t*3.1))
	case t < 9.6: // the tug of war
		h1.x = 262
		h1.anim, h1.ph, h1.lean = AnimPanic, t*10, 0.06*math.Sin(t*14)
	case t < 12.4: // after the letter, up to the ceiling
		h1.x = 262
		h1.hop = hopArc(t-9.6, 1.4, 48)
		h1.anim, h1.ph = AnimWave, (t-9.6)*6
	default:
		h1.x = 262
		if t > 15.4 {
			h1.anim, h1.ph = AnimCheer, t*6
		}
	}
	h2 = h1
	h2.y = preFloorY - 2
	h2.x = h1.x - 34 + 8*math.Sin(t*1.3)
	switch { // the red one is the nervous one
	case t < 3:
		h2.x = h1.x - 30
	case t > 3.6 && t < 5.6:
		h2.anim, h2.ph, h2.hop = AnimPanic, t*8, 4*math.Abs(math.Sin(t*8))
	case t >= 7.4 && t < 9.6:
		h2.x = h1.x - 30
		h2.anim, h2.ph, h2.lean = AnimPanic, t*9, -0.05*math.Sin(t*13)
	case t >= 9.6 && t < 12.4:
		h2.hop = hopArc(t-9.75, 1.4, 40)
	case t >= 15.4:
		h2.anim, h2.ph = AnimCheer, t*6+2
	}
	as = newPose(372, preFloorY+1, AnimDance1, DirDown)
	as.ph = t * 6
	switch {
	case t < 2.4:
	case t < 5.6:
		as.anim, as.dir, as.ph = AnimIdle, DirLeft, 0
	case t < 7.4:
		as.x = 372 - 56*arrSmooth((t-5.6)/1.8)
		as.anim, as.dir, as.ph = AnimWalk, DirLeft, t*7
	case t < 9.6:
		as.x = 316
		as.anim, as.dir, as.ph, as.lean = AnimPanic, DirLeft, t*10, -0.06*math.Sin(t*14)
	case t < 12.4:
		as.x = 316
		as.anim, as.dir, as.ph = AnimWave, DirLeft, (t-9.6)*6
		as.hop = hopArc(t-9.9, 1.4, 40)
	default:
		as.x = 316
		as.anim, as.dir, as.ph = AnimTalk, DirLeft, t*5
	}
	if t >= 2.4 && t < 5.6 {
		as.x = 372
	}
	return
}

// preEnvelope: the letter of the station: position, opacity and whether it is in the hands of the first hero.
func preEnvelope(t float64, h1 prePose) (x, y, a float64) {
	switch {
	case t < 3:
		return 300, 196, 0
	case t < 7.4:
		return 300, 196 + 3*math.Sin(t*2), clampF((t-3)/0.6, 0, 1)
	case t < 9.6:
		return 289 + 10*math.Sin(t*18), 188, 1
	case t < 12.4: // it floats to the ceiling
		u := (t - 9.6) / 2.8
		return 289 + 30*math.Sin(u*5), 188 - 58*arrSmooth(u), 1
	}
	u := clampF((t-12.4)/0.5, 0, 1)
	hx, hy := h1.x+12, 184.0
	return lerp(289+30*math.Sin(5), hx, u), lerp(130, hy, u), 1
}

var preLetterLines = []preLine{
	{1, 1.0, 2.6, "¡Qué ligero me siento!", true},
	{0, 3.6, 5.4, "¿Eso brilla... o es la resaca?", false},
	{2, 5.7, 7.3, "Un sobre flotando. Qué raro.", false},
	{0, 7.5, 8.9, "¡Lo vi yo primero!", false},
	{2, 9.0, 10.4, "¡Es de todos! ¡Gravedad cero!", false},
	{2, 10.5, 11.9, "¡Se escapa hacia el techo!", false},
	{0, 12.6, 14.0, "Lo tengo. Parece importante.", false},
	{2, 14.1, 15.5, "Huele a selva y a problemas.", false},
	{1, 15.6, 16.8, "¡ÁBRELO YA!", true},
}

func preLetterCaption(t float64) string {
	switch {
	case t < 3:
		return "Última fiesta ganada. Gravedad cero, resaca máxima."
	case t < 9.6:
		return "Algo brilla en el suelo. Todos lo miran. Todos lo quieren."
	case t < 12.4:
		return "En gravedad cero, hasta una carta se da a la fuga."
	}
	return "El sobre huele a selva. Y a problemas con dientes."
}

// ---- the bad one: the oxygen runs out ------------------------------------------------------------

const preBadPodX = 432.0

func preBadPoses(t float64, two bool) (h1, h2, as prePose) {
	h1 = newPose(150, preFloorY, AnimIdle, DirRight)
	switch {
	case t < 0.8:
	case t < 4.5:
		h1.anim, h1.ph, h1.hop = AnimPanic, t*8, 3*math.Abs(math.Sin(t*8))
	case t < 7.7:
		u := arrSmooth((t - 4.5) / 3.2)
		h1.x = 150 + (preBadPodX-12-150)*u
		h1.anim, h1.ph, h1.hop = AnimRun, t*11, 2*math.Abs(math.Sin(t*11))
	case t < 8.2:
		h1.x = preBadPodX - 12
	default:
		h1.x = preBadPodX - 12
		h1.alpha = 1 - clampF((t-8.2)/0.4, 0, 1)
		h1.hide = h1.alpha <= 0
	}
	// the red one: the wrong way first
	h2 = newPose(110, preFloorY-2, AnimIdle, DirRight)
	switch {
	case t < 4.5:
		h2.anim, h2.ph, h2.hop = AnimPanic, t*9+1, 3*math.Abs(math.Sin(t*9+1))
	case t < 5.7:
		h2.x = 110 - 60*(t-4.5)/1.2
		h2.anim, h2.ph, h2.dir = AnimRun, t*11, DirLeft
	case t < 8.3:
		u := arrSmooth((t - 5.7) / 2.6)
		h2.x = 50 + (preBadPodX-34-50)*u
		h2.anim, h2.ph = AnimRun, t*11
	default:
		h2.x = preBadPodX - 34
		h2.alpha = 1 - clampF((t-8.5)/0.4, 0, 1)
		h2.hide = h2.alpha <= 0
	}
	as = newPose(300, preFloorY+1, AnimIdle, DirLeft)
	switch {
	case t < 2.6:
		as.anim, as.ph = AnimDrink, t*3
	case t < 4.6:
		as.anim, as.ph = AnimTalk, t*5
	case t < 8.9:
		u := arrSmooth((t - 4.6) / 4.3)
		as.x = 300 + 100*u
		as.anim, as.ph, as.dir = AnimRun, t*9, DirRight
	case t < 10:
		as.x = 400
		as.anim, as.ph, as.dir = AnimPunch, t*6, DirRight
	default:
		as.x = 400
		as.anim, as.ph, as.dir = AnimWave, t*5, DirRight
	}
	return
}

// prePodX: where the escape pod is (it leaves at 9.8 s).
func prePodX(t float64) float64 {
	if t < 9.8 {
		return preBadPodX
	}
	u := t - 9.8
	return preBadPodX + u*u*190
}

var preBadLines = []preLine{
	{0, 1.2, 2.8, "¡Me quedo sin aire!", false},
	{2, 3.0, 4.6, "Tranquilo: respiro... ¿ya no respiro?", false},
	{0, 4.7, 6.1, "¡¡A LA CÁPSULA!!", false},
	{1, 5.9, 7.0, "¿Por dónde era?", true},
	{2, 8.9, 10.3, "¡Eh! ¿Y mi asiento?", false},
	{2, 10.6, 12.2, "Yo me quedo a recoger.", false},
}

func preBadCaption(t float64) string {
	switch {
	case t < 4.6:
		return "La fiesta ha gastado todo el oxígeno de la estación."
	case t < 9.8:
		return "Sálvese quien pueda... y quien tenga cápsula."
	}
	return "Despegue de emergencia. Plaza para uno (o dos)."
}

// preAlarm: the loudspeaker of the station (a banner at the top of the screen).
func preAlarm(t float64) string {
	switch {
	case t >= 0.4 && t < 2.9:
		return "ALERTA: OXÍGENO AL 3%"
	case t >= 3.1 && t < 5.2:
		return "PRÓXIMA PARADA: LA TIERRA"
	}
	return ""
}

// ---- the island: the welcome of the dinosaurs --------------------------------------------------------

const (
	preIslandPadX = 236.0
	preRexStopX   = 392.0
)

func preIslandPoses(t float64, two bool) (h1, h2, rp prePose) {
	h1 = newPose(120, preFloorY, AnimIdle, DirRight)
	switch {
	case t < 3:
		h1.x = -20 + 140*t/3
		h1.anim, h1.ph, h1.hop = AnimWalk, t*7, 0
	case t < 7:
		h1.anim, h1.ph, h1.hop = AnimPanic, t*8, 3*math.Abs(math.Sin(t*8))
	case t < 10:
		h1.anim, h1.ph = AnimCrouch, t*3
	case t < 12.5:
	default:
		h1.anim, h1.ph = AnimCheer, t*6
	}
	h2 = h1
	h2.y = preFloorY - 2
	h2.x = h1.x - 32
	switch {
	case t < 3:
	case t < 7:
		h2.x = h1.x - 40 + 12*math.Sin(t*2)
		h2.anim, h2.ph = AnimPanic, t*9+2
	case t >= 7 && t < 10:
		h2.x = h1.x - 70*arrSmooth((t-7)/1.2) - 32 // he hides behind the other one... and runs away
		h2.anim, h2.ph, h2.dir = AnimRun, t*11, DirLeft
		if t > 8.2 {
			h2.x = h1.x - 32 - 70
			h2.anim, h2.dir = AnimCrouch, DirRight
		}
	}
	rp = newPose(330, preFloorY, AnimIdle, DirLeft) // the raptors are drawn by the scene (props)
	return
}

// preRexX: where the T-Rex is (it comes in from the right).
func preRexX(t float64) float64 {
	return 650 - (650-preRexStopX)*arrSmooth((t-5.2)/3.6)
}

var preIslandLines = []preLine{
	{0, 1.2, 2.9, "Y el helicóptero era alquilado...", false},
	{0, 3.4, 4.9, "¡¡RAPTORES!! ¡Corred!", false},
	{1, 5.0, 6.4, "¡Tú primero, yo te sigo!", true},
	{0, 7.0, 8.6, "Adiós, mundo cruel...", false},
	{2, 10.7, 12.2, "¡Sorpresa! Hay tarta.", false},
	{0, 12.3, 13.7, "...¿No me vais a comer?", false},
	{3, 13.8, 15.3, "RAWR (= ¡Felicidades!)", false},
}

func preIslandCaption(t float64) string {
	switch {
	case t < 3:
		return "Última parada: la pista de aterrizaje de la isla."
	case t < 10:
		return "La fauna local ha preparado un recibimiento."
	}
	return "Y el recibimiento viene con confeti."
}

// ---- sounds ---------------------------------------------------------------------------------------

// preSound plays a sound once, the first time the clock gets past `at`.
func (g *Game) preSound(e *endState, slot int, at float64, name string) {
	if e.t >= at && !e.x.pre.done[slot] {
		e.x.pre.done[slot] = true
		g.snd.Play(name)
	}
}

func (g *Game) preludeSounds(e *endState) {
	g.preBuild(e)
	switch e.x.pre.kind {
	case preLetter:
		g.preSound(e, 0, 3.0, "ring")
		g.preSound(e, 1, 13.0, "spark")
	case preBad:
		g.preSound(e, 0, 0.3, "alert")
		g.preSound(e, 1, 1.5, "alert")
		g.preSound(e, 2, 3.2, "alert")
		g.preSound(e, 3, 9.8, "boom")
	case preIsland:
		g.preSound(e, 0, 6.0, "thud")
		g.preSound(e, 1, 7.2, "thud")
		g.preSound(e, 2, 8.6, "rex_roar")
		g.preSound(e, 3, 10.0, "win")
	}
}

// ---- drawing: shared pieces -----------------------------------------------------------------------------

// preProp draws a prop sprite of the game with its anchor (bottom centre) at (x, y) of a canvas.
func (g *Game) preProp(c *ebiten.Image, name string, v int, x, y, s float64, flip bool, alpha float64) {
	if g == nil || g.as == nil {
		return
	}
	imgs := g.as.props[name]
	if len(imgs) == 0 {
		return
	}
	p := imgs[((v%len(imgs))+len(imgs))%len(imgs)]
	if p.img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-p.ox, -p.oy)
	if flip {
		op.GeoM.Scale(-s, s)
	} else {
		op.GeoM.Scale(s, s)
	}
	op.GeoM.Translate(math.Round(x), math.Round(y))
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	c.DrawImage(p.img, op)
}

// preGlow: a soft light (additive), if the game has the glow picture.
func (g *Game) preGlow(c *ebiten.Image, x, y, r float64, col color.RGBA) {
	if g == nil || g.as == nil || g.as.glow == nil || r <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-48, -48)
	op.GeoM.Scale(r/48, r/48)
	op.GeoM.Translate(x, y)
	op.Blend = ebiten.BlendLighter
	op.ColorScale.ScaleWithColor(col)
	c.DrawImage(g.as.glow, op)
}

func (g *Game) prePutPose(c *ebiten.Image, cs *CharSet, p prePose, tint [3]float32) {
	if cs == nil || p.hide {
		return
	}
	recLayer("actor:pre")
	g.sceneChar(c, cs, p.anim, p.dir, p.ph, p.x, p.y, 1, p.alpha, p.hop, p.lean, tint)
}

// preBlit shows the canvas at 2x (a shake moves it by whole pixels).
func (g *Game) preBlit(dst *ebiten.Image, e *endState, shake float64) {
	dst.Fill(rgb(0x05060c))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	if shake > 0 {
		op.GeoM.Translate(math.Round(math.Sin(e.total*97)*shake), math.Round(math.Cos(e.total*83)*shake))
	}
	dst.DrawImage(e.canvas, op)
}

// preBubbles draws the balloons that are on at time t over the actors.
func (g *Game) preBubbles(dst *ebiten.Image, e *endState, lines []preLine, who [4]prePose, rexHead Vec) {
	two := g.scene2P()
	n := 0
	for _, l := range lines {
		if (l.two && !two) || e.t < l.a || e.t >= l.b || n >= 2 {
			continue
		}
		n++
		p := who[l.who]
		x, y := p.x*2, (p.y-p.hop-46)*2
		if l.who == 3 {
			x, y = rexHead.X*2, rexHead.Y*2
		}
		g.speechBubble(dst, x, y, l.s)
	}
}

// preBanner: the loudspeaker of the station.
func (g *Game) preBanner(dst *ebiten.Image, s string, t float64) {
	w := g.textW(s, 3) + 44
	on := int(t*5)%2 == 0
	bg := rgb(0x6a0a0a)
	if on {
		bg = rgb(0xa01414)
	}
	fillRect(dst, 480-w/2, 26, w, 42, rgb(0x14080a))
	fillRect(dst, 480-w/2+3, 29, w-6, 36, bg)
	g.drawTextC(dst, s, 480, 36, rgb(0xfff0c8), 3)
}

// ---- drawing: the station -------------------------------------------------------------------------------

func hashF(a, b int, seed uint32) float64 { return jpHash(a, b, seed) }

// preStation paints the last place of the space station: the view of the Earth, the metal wall, the
// white tiles, the dance floor with its blue lights and the DJ with the speakers.
func (g *Game) preStation(c *ebiten.Image, t float64, alarm float64) {
	c.Fill(rgb(0x04061c))
	// the big window: space (no dots), the Earth and a solar panel
	fillRect(c, 20, 12, 440, 112, rgb(0x2a2e3a))
	for y := 0.0; y < 108; y += 4 {
		fillRect(c, 24, 16+y, 432, 4, mixc(rgb(0x04061c), rgb(0x0c1244), y/108))
	}
	filledEllipse(c, 372, 168, 112, 108, rgba(0x6ab0ff, 60)) // the air
	filledEllipse(c, 372, 168, 106, 102, rgb(0x1c5ca8))
	filledEllipse(c, 340, 118, 40, 17, rgb(0x2f78c8))
	filledEllipse(c, 396, 124, 38, 12, rgb(0x3a9a4a))
	filledEllipse(c, 330, 126, 22, 9, rgb(0x3a9a4a))
	filledEllipse(c, 420, 134, 20, 6, rgb(0x4aaa58))
	filledEllipse(c, 366, 112, 30, 5, rgba(0xe8f4ff, 150)) // clouds
	filledEllipse(c, 316, 134, 24, 4, rgba(0xe8f4ff, 130))
	sx := 52 + math.Mod(t*1.5, 4)
	fillRect(c, sx, 52, 86, 34, rgb(0x1a2f6a))
	for yy := 52.0; yy < 86; yy += 6 {
		fillRect(c, sx, yy, 86, 1, rgb(0x4a6ac0))
	}
	for xx := sx; xx < sx+86; xx += 11 {
		fillRect(c, xx, 52, 1, 34, rgb(0x4a6ac0))
	}
	fillRect(c, sx+84, 66, 14, 3, rgb(0x8890a0))
	fillRect(c, 24, 16, 432, 2, rgb(0x6a7080)) // the frame
	fillRect(c, 156, 16, 5, 108, rgb(0x3a3e4c))
	fillRect(c, 300, 16, 5, 108, rgb(0x3a3e4c))
	// the wall below it
	fillRect(c, 0, 124, endCW, 52, rgb(0x4a5060))
	fillRect(c, 0, 124, endCW, 2, rgb(0x8890a0))
	for x := 0.0; x < endCW; x += 48 {
		fillRect(c, x, 126, 1, 50, rgb(0x3a3e4c))
		fillRect(c, x+2, 130, 44, 1, rgb(0x5a6070))
	}
	for x := -4.0; x < endCW; x += 16 { // the hazard band
		fillRect(c, x, 168, 8, 6, rgb(0xe8c030))
		fillRect(c, x+8, 168, 8, 6, rgb(0x2a2e3a))
	}
	// the tiles
	for y := 174.0; y < endCH; y += 3 {
		fillRect(c, 0, y, endCW, 3, mixc(rgb(0xe4ecf0), rgb(0xa8b4c2), (y-174)/96))
	}
	for y := 190.0; y < endCH; y += 22 {
		fillRect(c, 0, y, endCW, 1, rgba(0x6a7488, 150))
	}
	for x := 0.0; x < endCW; x += 40 {
		fillRect(c, x, 174, 1, 96, rgba(0x6a7488, 110))
	}
	// the dance floor, lit blue
	for r := 0; r < 4; r++ {
		for q := 0; q < 9; q++ {
			on := (r+q+int(t*3))%2 == 0
			col := rgb(0x2a4ac8)
			if on {
				col = rgb(0x1a2a88)
			}
			fillRect(c, 152+float64(q)*20, 196+float64(r)*12, 20, 12, col)
		}
	}
	g.preGlow(c, 240, 216, 110, color.RGBA{50, 80, 220, 90})
	// the DJ, the speakers and the plants
	g.preProp(c, "speaker", 0, 196, 190, 1, false, 1)
	g.preProp(c, "speaker", 0, 284, 190, 1, false, 1)
	g.preProp(c, "dj", 0, 240, 186, 1, false, 1)
	g.preProp(c, "plant", 0, 30, 186, 1, false, 1)
	g.preProp(c, "plant", 0, 450, 186, 1, false, 1)
	g.preProp(c, "lamp", 0, 120, 176, 1, false, 1)
	g.preProp(c, "lamp", 0, 360, 176, 1, false, 1)
	// the mess of the party, floating in the low gravity: confetti and a balloon
	for k := 0; k < 46; k++ {
		x := hashF(k, 1, 31) * endCW
		y := math.Mod(hashF(k, 2, 31)*endCH+endCH-t*(3+hashF(k, 3, 31)*5)+math.Sin(t+float64(k))*3, endCH-60) + 40
		fillRect(c, math.Round(x+math.Sin(t*0.7+float64(k))*4), math.Round(y), 2, 2, rgb(partyColors[k%len(partyColors)]))
	}
	bx, by := 70+14*math.Sin(t*0.6), 100+10*math.Sin(t*0.9)
	filledEllipse(c, bx, by, 7, 9, rgb(0xe03a8a))
	fillRect(c, bx, by+9, 1, 22, rgba(0xc8c8d8, 180))
	if alarm > 0 {
		fillRect(c, 0, 0, endCW, endCH, rgba(0xff0000, uint8(70*alarm*(0.5+0.5*math.Sin(t*7)))))
	}
	recLayer("bg:station")
}

func (g *Game) preEnvelopeDraw(c *ebiten.Image, x, y, a, t float64) {
	if a <= 0 {
		return
	}
	x, y = math.Round(x), math.Round(y)
	al := func(v uint8) uint8 { return uint8(float64(v) * a) }
	g.preGlow(c, x, y, 36+6*math.Sin(t*5), color.RGBA{255, 214, 100, al(110)})
	for i := 0; i < 6; i++ { // the glittering crosses around it
		ph := math.Mod(t*1.3+float64(i)*0.37, 1)
		ax := x + math.Cos(float64(i)*1.1)*(14+ph*10)
		ay := y + math.Sin(float64(i)*1.1)*(9+ph*6)
		sc := rgba(0xfff0a0, al(uint8(255*(1-ph))))
		fillRect(c, ax-1, ay-3, 2, 6, sc)
		fillRect(c, ax-3, ay-1, 6, 2, sc)
	}
	fillRect(c, x-9, y-6, 18, 13, rgb(0x8a6a2a))
	fillRect(c, x-8, y-5, 16, 11, rgb(0xf2e6c2))
	for i := 0; i < 8; i++ { // the flap
		fillRect(c, x-8+float64(i), y-5+float64(i)*0.6, 1, 1, rgb(0xcdb98a))
		fillRect(c, x+7-float64(i), y-5+float64(i)*0.6, 1, 1, rgb(0xcdb98a))
	}
	fillRect(c, x-2, y-1, 4, 4, rgb(0xc82a1a))
}

// preGoldBurst: the seal cracks and a ring of light goes through the room (letter, 13 to 16.4 s): a ring of
// sparks, not a disc, so the people stay in sight.
func (g *Game) preGoldBurst(c *ebiten.Image, x, y, u float64) {
	if u < 0 || u > 1 {
		return
	}
	r := 6 + 250*easeOut(u)
	a := uint8(255 * (1 - u))
	for q := 0; q < 72; q++ {
		ang := float64(q) / 72 * 2 * math.Pi
		fillRect(c, math.Round(x+math.Cos(ang)*r), math.Round(y+math.Sin(ang)*r*0.7), 3, 3, rgba(0xffe080, a))
		if r > 20 {
			fillRect(c, math.Round(x+math.Cos(ang+0.04)*(r-9)), math.Round(y+math.Sin(ang+0.04)*(r-9)*0.7), 2, 2, rgba(0xfff4c0, a/2))
		}
	}
	g.preGlow(c, x, y, 30+30*u, color.RGBA{255, 220, 120, uint8(90 * (1 - u))})
}

func (g *Game) drawPodStation(c *ebiten.Image, t float64) {
	x := prePodX(t)
	y := 208.0
	if t > 9.8 { // the flame
		fl := 18 + 8*math.Abs(math.Sin(t*40))
		fillRect(c, x-26-fl, y-16, fl, 10, rgb(0xf0a030))
		fillRect(c, x-26-fl*0.6, y-14, fl*0.6, 6, rgb(0xfff0a0))
	}
	// hatch and body: a rounded capsule seen from the side
	filledEllipse(c, x, y-18, 28, 20, rgb(0xc8ccd8))
	filledEllipse(c, x, y-22, 24, 12, rgb(0xe8ecf4))
	fillRect(c, x-28, y-14, 56, 5, rgb(0xd04030))
	fillRect(c, x-10, y-32, 20, 14, rgb(0x2a4a6a)) // the window
	fillRect(c, x-10, y-32, 20, 2, rgb(0x6a9ac8))
	if t > 8.4 { // the faces of those who got in
		filledEllipse(c, x-3, y-24, 4, 4, rgb(0xe8b890))
		filledEllipse(c, x+4, y-23, 3, 3, rgb(0xd09878))
	}
	if t < 8.3 { // the open hatch, with the light of the inside
		fillRect(c, x-30, y-26, 5, 22, rgb(0xf0e0a0))
	}
	fillRect(c, x-14, y-3, 28, 3, rgb(0x6a7080))
}

// ---- drawing: the island --------------------------------------------------------------------------------

func (g *Game) preIsland(c *ebiten.Image, t float64) {
	for y := 0.0; y < 150; y += 3 { // the dusk
		k := y / 150
		col := mixc(rgb(0x2a1a56), rgb(0xe0703a), k*k)
		fillRect(c, 0, y, endCW, 3, col)
	}
	filledEllipse(c, 300, 142, 54, 54, rgba(0xffd070, 90)) // the sun, sinking
	filledEllipse(c, 300, 142, 40, 40, rgb(0xf8b040))
	// the volcano and the jungle line
	for y := 0; y < 60; y++ {
		half := 6 + float64(y)*1.7
		fillRect(c, 120-half, 96+float64(y), half*2, 1, rgb(0x2a1a3a))
	}
	fillRect(c, 108, 92, 24, 3, rgb(0xd04820)) // the lava in the mouth
	for x := 0.0; x < endCW; x += 6 {
		h := 18 + 14*math.Sin(x/23) + 8*math.Sin(x/7+1)
		fillRect(c, x, 152-h, 6, h+2, rgb(0x14301c))
	}
	// the lawn and the helipad
	for y := 150.0; y < endCH; y += 3 {
		fillRect(c, 0, y, endCW, 3, mixc(rgb(0x2e6a3a), rgb(0x1c4a2a), (y-150)/120))
	}
	filledEllipse(c, preIslandPadX, 208, 124, 32, rgb(0x5a606a))
	filledEllipse(c, preIslandPadX, 206, 120, 29, rgb(0x8a909a))
	filledEllipse(c, preIslandPadX, 206, 104, 24, rgb(0x7a808a))
	for i := 0; i < 14; i++ { // the lights of the pad
		a := float64(i) / 14 * 2 * math.Pi
		on := (i+int(t*4))%2 == 0
		lc := rgb(0x6a3030)
		if on {
			lc = rgb(0xff5030)
		}
		fillRect(c, preIslandPadX+math.Cos(a)*116, 206+math.Sin(a)*27, 3, 2, lc)
	}
	// the H
	hx, hy := preIslandPadX-86, 197.0
	hc := rgb(0xe8c030)
	fillRect(c, hx, hy, 4, 18, hc)
	fillRect(c, hx+20, hy, 4, 18, hc)
	fillRect(c, hx, hy+7, 24, 4, hc)
	// the cage of the raptors (the door is open)
	fillRect(c, 4, 168, 72, 3, rgb(0x3a3a44))
	fillRect(c, 4, 168, 3, 46, rgb(0x3a3a44))
	fillRect(c, 73, 168, 3, 46, rgb(0x3a3a44))
	for x := 12.0; x < 72; x += 9 {
		fillRect(c, x, 170, 2, 44, rgb(0x5a5a68))
	}
	fillRect(c, 66, 172, 2, 40, rgb(0x2a2a30))
	// palms
	g.preProp(c, "palm", 0, 52, 176, 0.8, false, 1)
	g.preProp(c, "palm", 1, 446, 172, 0.85, false, 1)
	g.preProp(c, "palm", 2, 352, 160, 0.7, false, 1)
	// the wreck of the rented helicopter, on the pad, smoking
	g.preWreck(c, t)
	recLayer("bg:island")
}

func (g *Game) preWreck(c *ebiten.Image, t float64) {
	side, _ := arrHeliImgs()
	if side != nil {
		op := &ebiten.DrawImageOptions{}
		w, h := float64(side.Bounds().Dx()), float64(side.Bounds().Dy())
		op.GeoM.Translate(-w/2, -h)
		op.GeoM.Rotate(-0.12)
		op.GeoM.Translate(preIslandPadX+4, 212)
		op.ColorScale.Scale(0.35, 0.33, 0.33, 1)
		c.DrawImage(side, op)
	}
	for i := 0; i < 7; i++ { // the smoke
		u := math.Mod(t*0.5+float64(i)/7, 1)
		x := preIslandPadX + 6 + math.Sin(u*6+float64(i))*9
		y := 190 - u*90
		r := 6 + u*14
		filledEllipse(c, x, y, r, r*0.8, rgba(0x2a2a2e, uint8(170*(1-u))))
	}
	fillRect(c, preIslandPadX-2, 196, 5, 5, rgb(0xf06020)) // an ember
	g.preGlow(c, preIslandPadX, 198, 22, color.RGBA{255, 120, 40, 80})
}

// preHat draws a party hat (a cone with a bobble) with its base centred on (x, y).
func preHat(c *ebiten.Image, x, y float64, col uint32) {
	x, y = math.Round(x), math.Round(y)
	for i := 0; i < 8; i++ {
		fillRect(c, x-4+float64(i)*0.5, y-float64(i), 8-float64(i), 1, rgb(col))
	}
	fillRect(c, x-4, y-3, 8, 1, rgb(0xf8f0a0))
	fillRect(c, x-1, y-10, 3, 3, rgb(0xff6a8a))
}

func (g *Game) preRaptors(c *ebiten.Image, t float64) {
	recLayer("actor:raptor")
	for i, r := range []struct{ x, at float64 }{{318, 3.0}, {356, 3.5}} {
		if t < r.at {
			continue
		}
		u := math.Min(1, (t-r.at)/0.7)
		x := r.x + 30*(1-u)
		hop := 4 * math.Abs(math.Sin(t*5+float64(i)*2)) * u
		if t > 10 {
			hop = 9 * math.Abs(math.Sin(t*7+float64(i)))
		}
		y := preFloorY - 2 + float64(i)*3 - hop
		g.preProp(c, "jp_raptor", i, x, y, 1, false, 1)
		preHat(c, x-14, y-34, []uint32{0xe03a8a, 0x3ad0a8}[i])
		if t > 9 { // a party horn
			fillRect(c, x-24, y-26, 8, 2, rgb(0xf0c030))
			for q := 0; q < 6; q++ {
				fillRect(c, x-30-float64(q)*2, y-26+math.Sin(t*9+float64(q))*2, 2, 2, rgb(partyColors[(q+i)%len(partyColors)]))
			}
		}
	}
}

// the head of the T-Rex of the prelude (for the balloon and the confetti cannon)
func preRexHead(t float64) Vec { return Vec{preRexX(t) - 46, 156} }

func (g *Game) preRex(c *ebiten.Image, t float64) {
	if t < 5.2 {
		return
	}
	recLayer("actor:rex")
	x := preRexX(t)
	bob := 0.0
	if t < 8.8 {
		bob = 2 * math.Abs(math.Sin((t-5.2)*4.4))
	}
	g.preProp(c, "jp_trex", 0, x, 224-bob, 1, false, 1)
	h := preRexHead(t)
	preHat(c, h.X+4, h.Y-12, 0xf0c030)
	if t >= 10 { // the confetti cannon
		k := t - 10
		for i := 0; i < 90; i++ {
			a := -math.Pi + 0.9*(hashF(i, 1, 5)-0.5) - 0.35
			v := 70 + 90*hashF(i, 2, 5)
			px := h.X - 6 + math.Cos(a)*v*k*0.9
			py := h.Y + 8 + math.Sin(a)*v*k + 70*k*k
			if py > preFloorY+6 {
				py = preFloorY + 6
			}
			fillRect(c, math.Round(px), math.Round(py), 2, 2, rgb(partyColors[i%len(partyColors)]))
		}
		if k < 0.5 {
			g.preGlow(c, h.X-8, h.Y+8, 34, color.RGBA{255, 230, 140, uint8(200 * (1 - k*2))})
		}
	}
}

// ---- the drawing of the preludes ---------------------------------------------------------------------

func (g *Game) drawPrelude(dst *ebiten.Image, e *endState) {
	g.preBuild(e)
	if e.canvas == nil {
		e.canvas = ebiten.NewImage(endCW, endCH)
	}
	c := e.canvas
	t := e.t
	two := g.scene2P()
	p := &e.x.pre
	h1, h2, o := preActors(p.kind, t, two)
	cs0, cs1 := g.sceneHero(0, false), g.sceneHero(1, false)
	tint0, tint1 := [3]float32{1, 1, 1}, sceneTint(1, cs0, cs1)
	shake := 0.0
	var lines []preLine
	var caption string
	var rexHead Vec
	other := p.astro
	switch p.kind {
	case preLetter:
		g.preStation(c, t, 0)
		ex, ey, ea := preEnvelope(t, h1)
		g.prePutPose(c, other, o, tint0)
		g.prePutPose(c, cs0, h1, tint0)
		if two {
			g.prePutPose(c, cs1, h2, tint1)
		}
		recLayer("front:envelope")
		g.preEnvelopeDraw(c, ex, ey, ea, t)
		g.preGoldBurst(c, ex, ey, (t-13)/3.4)
		if f := clampF((t-16.2)/0.8, 0, 1); f > 0 {
			fillRect(c, 0, 0, endCW, endCH, rgba(0xfff4d0, uint8(255*f)))
		}
		lines, caption = preLetterLines, preLetterCaption(t)
	case preBad:
		g.preStation(c, t, clampF(t/0.5, 0, 1))
		g.drawPodStation(c, t)
		g.prePutPose(c, other, o, tint0)
		g.prePutPose(c, cs0, h1, tint0)
		if two {
			g.prePutPose(c, cs1, h2, tint1)
		}
		if t >= 9.8 && t < 11.6 {
			shake = 3
		}
		if f := clampF((t-11.8)/0.8, 0, 1); f > 0 {
			fillRect(c, 0, 0, endCW, endCH, rgba(0x000000, uint8(255*f)))
		}
		lines, caption = preBadLines, preBadCaption(t)
	default:
		g.preIsland(c, t)
		g.preRaptors(c, t)
		g.prePutPose(c, cs0, h1, tint0)
		if two {
			g.prePutPose(c, cs1, h2, tint1)
		}
		g.preRex(c, t)
		rexHead = preRexHead(t)
		rexHead.Y -= 40
		if t >= 8.6 && t < 9.6 || t >= 10 && t < 10.5 {
			shake = 3
		}
		if f := clampF((t-14.7)/0.8, 0, 1); f > 0 {
			fillRect(c, 0, 0, endCW, endCH, rgba(0xfff4d0, uint8(255*f)))
		}
		lines, caption = preIslandLines, preIslandCaption(t)
	}
	if t < 0.8 { // fade in from black
		fillRect(c, 0, 0, endCW, endCH, rgba(0x000000, uint8(255*(1-t/0.8))))
	}
	g.preBlit(dst, e, shake)
	// texts (on the screen, not on the canvas)
	if p.kind == preBad {
		if s := preAlarm(t); s != "" {
			g.preBanner(dst, s, t)
		}
	}
	if p.kind == preLetter && t >= 3 && t < 3.8 {
		g.sceneBang(dst, "!", h1.x*2, (h1.y-h1.hop-52)*2, (t-3)/0.8, 0xffe060, 5)
	}
	if p.kind == preIsland && t >= 8.6 && t < 9.8 {
		g.sceneBang(dst, "RAAAWR", rexHead.X*2, 150, (t-8.6)/1.2, 0xff5050, 6)
	}
	if p.kind == preIsland && t >= 10.4 && t < 13.6 {
		g.drawTextC(dst, "¡SORPRESA!", 480, 100+math.Round(4*math.Sin(t*6)), rgb(partyColors[int(t*5)%len(partyColors)]), 6)
	}
	if p.kind == preBad && t >= 9.9 && t < 11.4 {
		g.sceneBang(dst, "FUUUSH", 480, 150, (t-9.9)/1.5, 0xf0c030, 5)
	}
	if p.kind == preLetter && t >= 13.0 && t < 13.8 {
		g.sceneBang(dst, "¡CRAC!", h1.x*2, (h1.y-70)*2, (t-13)/0.8, 0xffe060, 4)
	}
	who := [4]prePose{h1, h2, o, o}
	g.preBubbles(dst, e, lines, who, rexHead)
	g.endCaption(dst, caption)
}
