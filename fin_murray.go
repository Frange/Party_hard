package main

import (
	"image/color"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ---------------------------------------------------------------------------
// Murray on the last card of both endings: the talking skull on his stake, saying a
// new line every finSayEvery seconds in a speech bubble typed letter by letter.
//   - the secret "FIN": top left (clear of "FIN", the two lines under it and
//     "Gracias por jugar"), with finMurrayLines; the skull is mirrored so he faces right;
//   - the normal "¿ FINAL ?": top right (bubble on his left), with its own
//     set finMurrayLinesNormal (never one of the secret ones); the skull as drawn faces left.
// On both cards he looks at the middle of the screen: his face turned to it and his pupils
// (the glints in his sockets) down and forward (finMurrayLook).
// The lines come in a random order without repeating until all have been said (finMurrayLine).
// ---------------------------------------------------------------------------

const (
	finSayEvery  = 7.0  // a new line every 7 seconds
	finSayFrom   = 2.6  // the first one, just after the lines under "FIN"
	finMenuAfter = 20.0 // the secret FIN shows (and allows) the way back to the menu only after this many seconds
	finMenuNorm  = 8.0  // the normal "¿ FINAL ?": a short wait, enough for Murray's first line
	finTypeCPS   = 30.0 // the bubble is typed at this many letters a second
	finPauseSp   = 0.03 // a tiny extra pause after a space...
	finPausePun  = 0.22 // ...and a longer one after punctuation (the jaw rests)
	finJawHz     = 7.0  // open/closed cycles a second while he talks
)

// finMenuOpen: whether the end card offers (and accepts) going back to the menu at time t. The secret FIN waits
// finMenuAfter seconds so Murray can say a few lines; the normal "¿ FINAL ?" waits finMenuNorm seconds (his first
// line is fully typed by then). Pure.
func finMenuOpen(trueEnd bool, t float64) bool {
	if !trueEnd {
		return t >= finMenuNorm
	}
	return t >= finMenuAfter
}

// finTypeAt: the typewriter of Murray's bubble. t is the time since the line started, s the (translated) line.
// It returns how many runes are visible and whether the jaw is open: it moves at finJawHz while letters come out,
// rests during the pauses after punctuation and stays shut once the line is complete. Pure.
func finTypeAt(t float64, s string) (vis int, open bool) {
	rs := []rune(s)
	if t < 0 || len(rs) == 0 {
		return 0, false
	}
	at := 0.0   // when the next rune appears
	last := 0.0 // when the last visible rune appeared
	for _, r := range rs {
		if at > t {
			break
		}
		vis++
		last = at
		at += 1 / finTypeCPS
		switch r {
		case ' ':
			at += finPauseSp
		case '.', ',', ';', ':', '!', '?', '…', '»':
			at += finPausePun
		}
	}
	if vis >= len(rs) {
		return len(rs), false
	}
	if t-last > 1/finTypeCPS+finPauseSp+0.005 { // a pause after punctuation: mouth shut
		return vis, false
	}
	return vis, int(math.Floor(t*finJawHz*2))%2 == 0
}

// finTypeLen: how long the whole line takes to type.
func finTypeLen(s string) float64 {
	for t := 0.0; t < 30; t += 0.01 {
		if v, _ := finTypeAt(t, s); v >= len([]rune(s)) {
			return t
		}
	}
	return 30
}

// finMurrayLines: ES -> EN. Short (<= 48 runes) so they fit the bubble.
var finMurrayLines = [][2]string{
	{"¿Juego acabado? Hala, ahora a jugar al GTA 6.", "Game beaten? Off you go to play GTA 6, then."},
	{"Créditos vistos. Ya podéis apagar. En serio.", "Credits seen. You can switch off. Really."},
	{"«Un bloque más», dijo. Hace tres años.", "\"One more block,\" he said. Three years ago."},
	{"Ese ruidito sssss detrás de ti... no es nada.", "That little hisssss behind you... is nothing."},
	{"Nunca caves hacia abajo. Consejo de calavera.", "Never dig straight down. Skull's advice."},
	{"Un creeper en una fiesta: el mejor petardo.", "A creeper at a party: the best firecracker."},
	{"Si ves un Enderman, no le mires. A mí sí.", "See an Enderman? Don't stare. Stare at me."},
	{"Cama en el Nether: mala idea. Como esta fiesta.", "A bed in the Nether: bad idea. Like this party."},
	{"Mi bioma favorito: la estaca.", "My favourite biome: the stake."},
	{"El One Piece existe. Tú has elegido una fiesta.", "The One Piece is real. You chose a party."},
	{"Luffy estira el brazo. Yo no tengo ni brazo.", "Luffy stretches his arm. I don't even have one."},
	{"¡Seré el Rey de los Piratas! ...desde mi estaca.", "I'll be King of the Pirates! ...from my stake."},
	{"Mi Fruta del Diablo: no poder bajarme de aquí.", "My Devil Fruit: never getting off this stake."},
	{"Que alguien diga «nakama». Necesito amigos.", "Someone say \"nakama\". I need friends."},
	{"Mi poder pasa de 9000. Mi altura, no.", "My power level is over 9000. My height isn't."},
	{"Y esta ni siquiera es mi forma final.", "And this isn't even my final form."},
	{"Hasta un slime de anime tiene más cuerpo que yo.", "Even an anime slime has more body than me."},
	{"Si te vas a un isekai, llévame. Ocupo poco.", "If you get isekai'd, take me. I'm compact."},
	{"El vecino ha llamado a la policía. Otra vez.", "The neighbour called the police. Again."},
	{"La poli aún busca al culpable. Yo no he sido.", "The cops still want the culprit. Wasn't me."},
	{"El T-Rex se comió la tarta. Y yo sin estómago.", "The T-Rex ate the cake. And I have no stomach."},
	{"La tarta no era mentira: se la comió el T-Rex.", "The cake wasn't a lie: the T-Rex ate it."},
	{"El T-Rex rugió. Yo también, pero nadie me oyó.", "The T-Rex roared. So did I. Nobody heard."},
	{"¿Y el alquiler de este mes? Ni lo menciones.", "This month's rent? Don't even mention it."},
	{"¿El vecino? Se ha mudado. A otro continente.", "The neighbour? He moved. To another continent."},
	{"Consejo: no invites a un dinosaurio a tu fiesta.", "Tip: never invite a dinosaur to your party."},
	{"Consejo: bebe agua. Yo no puedo, se me sale.", "Tip: drink water. I can't, it leaks out."},
	{"Consejo: si el bajo retumba, el vecino también.", "Tip: if the bass booms, so does the neighbour."},
	{"Consejo: guarda partida. Yo no lo hice. Mírame.", "Tip: save your game. I didn't. Look at me."},
	{"Soy una calavera demoníaca y temible. ¡Temed!", "I'm a fearsome demonic skull. Tremble!"},
	{"¿Nadie aplaude? Llevo aquí todo el juego.", "No applause? I've been here the whole game."},
	{"Mi nombre en los créditos: más grande, ¿eh?", "My name in the credits: bigger, OK?"},
	{"Récord de speedrun: saltarse los créditos. Feo.", "Speedrun record: skipping the credits. Rude."},
	{"Like y suscríbete. Ah, que esto no es un vídeo.", "Like and subscribe. Oh wait, not a video."},
	{"Sal a tocar hierba. Yo no puedo, toco tierra.", "Go touch grass. I can't, I touch dirt."},
	{"Todo bien. Todo perfectamente bien.", "This is fine. Everything is fine."},
	{"¿Secuela? Party Hardy 2: la calavera se baja.", "Sequel? Party Hardy 2: the skull gets down."},
	{"¿Aún aquí? Qué constancia. Yo no tengo opción.", "Still here? Impressive. I have no choice."},
	{"Una frase cada 7 segundos. Me cronometré.", "One line every 7 seconds. I timed it."},
	{"¿Todos los trofeos? Ahora, la vida real. Suerte.", "All trophies? Now try real life. Good luck."},
	{"Ni una fiesta más. Bueno... una. La última.", "No more parties. Well... one. The last one."},
	{"Mañana, resaca. Pasado, fiesta. Es el ciclo.", "Hangover tomorrow. Party after. The cycle."},
}

// finMurrayLinesNormal: what he says on the normal "¿ FINAL ?" (the night is not over: trophies are missing,
// a letter will come, the last party bites). ES -> EN, <= 48 runes, none shared with finMurrayLines.
var finMurrayLinesNormal = [][2]string{
	{"¿Ya está? Yo esperaba un final con más dientes.", "That's it? I expected an ending with more teeth."},
	{"Te faltan trofeos. Lo sé. Lo veo desde aquí.", "You're missing trophies. I can see from here."},
	{"Esto no es el final. Bueno, casi. Más o menos.", "This isn't the end. Well, almost. Sort of."},
	{"Llegará una carta. No la tires con el buzoneo.", "A letter will come. Don't bin it as junk."},
	{"Dicen que hay una isla. No sé nadar. Ni andar.", "An island, they say. I can't swim. Or walk."},
	{"¿Interrogación en el título? Mala señal.", "A question mark in the title? Bad sign."},
	{"El final bueno es para quien lo tiene todo.", "The good ending is for those who have it all."},
	{"Hoy duermes. Mañana, a por los trofeos.", "Sleep tonight. Trophies tomorrow."},
	{"He contado ovejas contigo. Me salieron nueve.", "I counted sheep with you. I got to nine."},
	{"Las ovejas también tienen resaca, ¿sabes?", "Sheep get hangovers too, you know."},
	{"No apagues aún. Bueno, apaga. Yo me quedo.", "Don't switch off yet. Fine, do. I'll stay."},
	{"Soy la mascota del final malo. Qué honor.", "I'm the bad ending's mascot. What an honour."},
	{"Final regular. Como mi dentadura.", "A so-so ending. Like my teeth."},
	{"¿Ves la estaca? Así me siento en los finales.", "See the stake? That's how endings feel to me."},
	{"Tranquilo, ni el vecino ha visto el final real.", "Relax, not even the neighbour saw the real end."},
	{"Trofeos que te faltan: los que no tienes.", "Trophies missing: the ones you haven't got."},
	{"Una pista: el último invitado muerde.", "A hint: the last guest bites."},
	{"Otra pista: allí se llega en helicóptero.", "Another hint: you get there by chopper."},
	{"Vi el final real. Bueno, lo oí. No tengo ojos.", "I saw the real end. Well, heard it. No eyes."},
	{"¿Pulsar para seguir? No hay seguir. Aún.", "Press to continue? There is no continue. Yet."},
	{"Si esto fuera un examen: un siete. Y medio.", "If this were an exam: a B. B minus."},
	{"La fiesta ha terminado. La limpieza, no.", "The party's over. The cleaning isn't."},
	{"Alguien tiene que fregar. Yo no tengo manos.", "Someone has to do the dishes. I have no hands."},
	{"Ni una fiesta más esta noche. Mañana, hablamos.", "No more parties tonight. Tomorrow, we'll talk."},
	{"¿Quieres el final bueno? Vuelve a fiestear.", "Want the good ending? Go party again."},
	{"Salgo en los dos finales. Cobro doble.", "I'm in both endings. Double pay."},
	{"Tu sofá te echa de menos. Yo, un poco.", "Your sofa misses you. Me, a little."},
	{"Bostezo. Contagioso, ¿eh? Y sin pulmones.", "Yawn. Contagious, huh? And I've no lungs."},
	{"El signo de interrogación lo puse yo. De nada.", "I put the question mark there. You're welcome."},
	{"Fiesta acabada, vecino dormido. Por fin, paz.", "Party over, neighbour asleep. Peace at last."},
}

// finMurraySet: the lines of the card of this ending.
func finMurraySet(trueEnd bool) [][2]string {
	if trueEnd {
		return finMurrayLines
	}
	return finMurrayLinesNormal
}

func init() {
	for _, p := range append(append([][2]string{}, finMurrayLines...), finMurrayLinesNormal...) {
		if _, ok := langDataEN[p[0]]; !ok {
			langDataEN[p[0]] = p[1]
		}
	}
}

// finMurraySeed: a new order every time the game runs.
var finMurraySeed = time.Now().UnixNano()

func init() { // debug: PH_FINSEED=n fixes the order (to capture a given line)
	if v, err := strconv.ParseInt(os.Getenv("PH_FINSEED"), 10, 64); err == nil {
		finMurraySeed = v
	}
}

// finMurraySlot: which line is being said at time t of the card (-1: none yet). Pure.
func finMurraySlot(t float64) int {
	if t < finSayFrom {
		return -1
	}
	return int(math.Floor((t-finSayFrom)/finSayEvery + 1e-9)) // the epsilon: 8 s after a slot is the next one, not float noise
}

// finMurrayPerm: the raw order of round c (a shuffle of the n lines).
func finMurrayPerm(seed int64, c, n int) []int {
	return rand.New(rand.NewSource(seed + int64(c)*7919)).Perm(n)
}

// finMurrayLine: the index into finMurrayLines said in a slot (finMurrayLineN for any set).
func finMurrayLine(seed int64, slot int) int { return finMurrayLineN(seed, slot, len(finMurrayLines)) }

// finMurrayLineN: the index into a set of n lines said in a slot. Every round of n slots says each
// line once; the first line of a round is never the last one of the round before. Pure.
func finMurrayLineN(seed int64, slot, n int) int {
	if slot < 0 || n <= 0 {
		return -1
	}
	c, i := slot/n, slot%n
	p := finMurrayPerm(seed, c, n)
	if c > 0 && n >= 3 && p[0] == finMurrayPerm(seed, c-1, n)[n-1] {
		p[0], p[1] = p[1], p[0]
	}
	return p[i]
}

// The bubble's text: a whole scale (crisp bitmap font), its line pitch, padding and wrap width.
const (
	finBubScale = 2.0   // 12 px a letter, 32 px a cell
	finBubLH    = 28.0  // line pitch
	finBubPad   = 7.0   // paper around the ink
	finBubWrap  = 372.0 // widest line (31 letters): every line fits in 3 lines
	finBubTail  = 10.0  // the tail hangs this far under the bubble's paper
)

// finMurrayGeom: where Murray stands (his feet, at scale 4) and the bubble's anchor on each card. The secret
// FIN has him top left with the bubble to his right; the normal "¿ FINAL ?" (wider title) has him top right,
// higher up, with the bubble to his left: ax is then the bubble's RIGHT edge. mirror: the skull is drawn
// mirrored (the sprite faces left, so on the left card it is flipped to face the middle). Pure.
func finMurrayGeom(trueEnd bool) (footX, footY, ax, by float64, mirror bool) {
	if trueEnd {
		return 100, 238, 170, 28, true
	}
	return screenW - 100, 216, screenW - 170, 18, false
}

// finBubbleLeft: whether the bubble hangs to the left of Murray (normal card): ax is its right edge.
func finBubbleLeft(trueEnd bool) bool { return !trueEnd }

// finBubbleRect: the bubble (without its 2 px ink border) for text of width w and n lines. Pure.
func finBubbleRect(trueEnd bool, w float64, n int) (bx, by, bw, bh float64) {
	_, _, ax, by, _ := finMurrayGeom(trueEnd)
	// the cell of a line is 32 px with 4 empty px on top; the last line's descenders end 28 px under it
	bw, bh = math.Ceil(w)+2*finBubPad, float64(n-1)*finBubLH+2*finBubPad+24
	bx = ax
	if finBubbleLeft(trueEnd) {
		bx = ax - bw
	}
	return bx, by, bw, bh
}

// finBubbleLines: a line of Murray wrapped for the bubble (translated, at most 3 lines).
func (g *Game) finBubbleLines(line string) []string {
	return wrapBubble(tr(line), finBubWrap, func(s string) float64 { return g.textW(s, finBubScale) }, 3)
}

// finMurrayLook: the skull looking down and forward (towards the jaw's side, the side he faces): the white
// glint of each socket, drawn in the middle by drawMurray, moves one pixel forward and one down.
func finMurrayLook(b *pixBuf) {
	ink, glint := c(0x0a0a0c), c(0xffffff)
	b.rect(6, 11, 2, 2, ink)
	b.rect(14, 11, 2, 2, ink)
	b.rect(5, 12, 2, 2, glint)
	b.rect(13, 12, 2, 2, glint)
}

var (
	finLookMu   sync.Mutex
	finLookImgs = map[int]PropImg{}
)

// finMurrayFrame: frame v of Murray (0 shut, 2 open) with the eyes of finMurrayLook, built once like the
// prop's own frames (renderPropSprite + stylizeSprite); fallback is the prop frame as it is.
func finMurrayFrame(fallback PropImg, v int) PropImg {
	k := propKinds["murray"]
	if k == nil || k.Draw == nil {
		return fallback
	}
	finLookMu.Lock()
	defer finLookMu.Unlock()
	if im, ok := finLookImgs[v]; ok {
		return im
	}
	k2 := *k
	k2.Draw = func(b *pixBuf, v int) { k.Draw(b, v); finMurrayLook(b) }
	im := PropImg{img: stylizeSprite(renderPropSprite("murray", &k2, v)).img(), ox: fallback.ox, oy: fallback.oy}
	finLookImgs[v] = im
	return im
}

// drawFinMurray: Murray and his bubble on the last card of both endings. The bubble is typed letter by letter
// (finTypeAt) and his jaw moves while the letters come out.
func (g *Game) drawFinMurray(dst *ebiten.Image, e *endState) {
	if g.as == nil {
		return
	}
	imgs := g.as.props["murray"]
	if len(imgs) == 0 || e.t < 2.2 {
		return
	}
	const sc = 4.0
	footX, footY, _, _, mirror := finMurrayGeom(e.trueEnd)
	set := finMurraySet(e.trueEnd)
	slot := finMurraySlot(e.t)
	var lines []string
	vis, open := 0, false
	if slot >= 0 {
		lines = g.finBubbleLines(set[finMurrayLineN(finMurraySeed, slot, len(set))][0])
		vis, open = finTypeAt(e.t-finSayFrom-float64(slot)*finSayEvery, strings.Join(lines, " "))
	}
	fr := finMurrayFrame(imgs[0], 0)
	if open && len(imgs) >= 3 {
		fr = finMurrayFrame(imgs[2], 2)
	}
	op := &ebiten.DrawImageOptions{}
	if mirror { // the sprite faces left: flipped, he faces right (the middle of the secret card)
		op.GeoM.Scale(-sc, sc)
		op.GeoM.Translate(math.Round(footX+fr.ox*sc), math.Round(footY-fr.oy*sc))
	} else {
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(math.Round(footX-fr.ox*sc), math.Round(footY-fr.oy*sc))
	}
	dst.DrawImage(fr.img, op)
	if slot < 0 {
		return
	}
	w := 0.0
	for _, l := range lines { // the bubble has its final size from the first letter: it does not grow while typed
		w = math.Max(w, g.textW(l, finBubScale))
	}
	bx, by, bw, bh := finBubbleRect(e.trueEnd, w, len(lines))
	left := finBubbleLeft(e.trueEnd)
	ink, paper := rgb(0x0a0a0c), rgb(0xf6f2e6)
	fillRect(dst, bx-2, by-2, bw+4, bh+4, ink)
	fillRect(dst, bx, by, bw, bh, paper)
	// the tail: out of the bottom corner of the bubble on his side, down and towards his head
	// (bottom-left on the secret card; mirrored, bottom-right, on the normal one)
	mx := func(x, wd float64) float64 { // the left edge of a run of width wd, mirrored about the bubble's centre
		if left {
			return 2*bx + bw - x - wd
		}
		return x
	}
	for r := -2.0; r < finBubTail-1; r++ { // finBubTail rows of ink under the paper (the last one is the tip)
		x0 := bx + 12 - math.Floor((r+2)*1.5)
		wd := finBubTail - 1 - r
		if r >= 0 {
			fillRect(dst, mx(x0-2, wd+4), by+bh+r, wd+4, 1, ink)
		}
		fillRect(dst, mx(x0, wd), by+bh+r, wd, 1, paper)
	}
	tip := bx + 12 - math.Floor((finBubTail+1)*1.5) - 2
	fillRect(dst, mx(tip, 4), by+bh+finBubTail-1, 4, 1, ink)
	rest := vis
	for i, l := range lines {
		rs := []rune(l)
		n := minI(rest, len(rs))
		rest -= n + 1 // the space that joined this line to the next
		if n <= 0 {
			break
		}
		g.drawInkText(dst, string(rs[:n]), bx+finBubPad, by+finBubPad-4+float64(i)*finBubLH, ink, finBubScale)
	}
}

// drawInkText: text already translated, in one flat colour on a light paper, at a whole scale and on whole
// pixels, with no drop shadow (drawText's black shadow under dark ink only thickens and blurs the letters).
func (g *Game) drawInkText(dst *ebiten.Image, s string, x, y float64, c color.RGBA, scale float64) {
	scale = pxScale(scale)
	x, y = math.Round(x), math.Round(y)
	g.recText(dst, "drawText", s, x, y, scale)
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, g.face, op)
}

// finBubbleBox: the whole bubble as drawn (ink border and tail included) for text of width w and n lines. Pure.
func finBubbleBox(trueEnd bool, w float64, n int) (x0, y0, x1, y1 float64) {
	bx, by, bw, bh := finBubbleRect(trueEnd, w, n)
	x0, x1 = bx-2, bx+bw+2
	tip := 12 - math.Floor((finBubTail+1)*1.5) - 2 // the tip's left edge, from bx (negative: left of the bubble)
	if finBubbleLeft(trueEnd) {
		x1 = math.Max(x1, bx+bw-tip)
	} else {
		x0 = math.Min(x0, bx+tip)
	}
	return x0, by - 2, x1, by + bh + finBubTail
}
