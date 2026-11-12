package main

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ---------------------------------------------------------------------------
// What Diego, César and Sam-sam shout while they ride: a few short skater jokes (ES -> EN), the same lines in
// the game (the skate park's bowls and their hops round any party, through the bubble manager) and on the last
// card of both endings (a small bubble over the head of the one who talks, finRiderSay).
// On the card a line is only said where its bubble fits: in the empty sides of the card (never over the
// texts in the middle), clear of Murray and his bubble and of the other kids' bubbles.
// Everything is a pure function of the card's time (a memo of who spoke when, computed in order).
// ---------------------------------------------------------------------------

// skaterLinesAll: anyone of the three. <= 28 runes (the card's little bubble).
var skaterLinesAll = [][2]string{
	{"¡Mira, sin manos!", "Look, no hands!"},
	{"¡Me he caído de cabeza!", "I landed on my head!"},
	{"Mi madre no sabe que estoy aquí", "My mum doesn't know I'm here"},
	{"¡Eso cuenta como truco!", "That counts as a trick!"},
	{"¡Otra vez! ¡Otra vez!", "Again! Again!"},
	{"¿Casco? ¿Qué casco?", "Helmet? What helmet?"},
	{"¡Grábame, grábame!", "Film me, film me!"},
	{"Mañana hay cole. Me da igual", "School tomorrow. Don't care"},
	{"Me duele todo. ¡Otra!", "Everything hurts. Again!"},
	{"¡Soy el rey del bowl!", "I'm the king of the bowl!"},
	{"¡Cinco minutos más!", "Five more minutes!"},
	{"¡Sin rodilleras, como los pros!", "No knee pads, like the pros!"},
	{"¿Quién ha puesto ese bordillo?", "Who put that kerb there?"},
	{"¡Ahora un 540! ...o un 180", "Now a 540! ...or a 180"},
	{"¡La calavera me anima!", "The skull is cheering me on!"},
	{"¡Nadie me para!", "Nobody can stop me!"},
	{"¡Eso no ha sido una caída!", "That was not a fall!"},
}

// skaterLinesOwn: each one's own (Diego on his skateboard, the other two on scooters).
var skaterLinesOwn = map[string][][2]string{
	"diego": {
		{"¡Kickflip de los buenos!", "A proper kickflip!"},
		{"Mi skate no frena. Nunca", "My board has no brakes. Ever"},
		{"¡Heelflip! Creo", "Heelflip! I think"},
		{"¡Manual hasta la cocina!", "Manual all the way home!"},
	},
	"cesar": {
		{"¡Mi patinete vuela!", "My scooter can fly!"},
		{"¡Barspin! ¿Lo has visto?", "Barspin! Did you see?"},
		{"¡Bunny hop de conejo!", "A bunny hop, like a bunny!"},
		{"He roto otro jarrón. Sin querer", "Broke another vase. Oops"},
	},
	"sam_sam": {
		{"¡Tailwhip doble!", "Double tailwhip!"},
		{"¡El patinete más rápido!", "The fastest scooter ever!"},
		{"¡Sin manos y sin miedo!", "No hands and no fear!"},
		{"¡Vuelta y vuelta y vuelta!", "Round and round and round!"},
	},
}

// skaterLines: the lines of one of the kids (his own first, then everybody's).
func skaterLines(slug string) [][2]string {
	return append(append([][2]string{}, skaterLinesOwn[slug]...), skaterLinesAll...)
}

// skaterTrickShout: what a kid shouts as he lands a trick in a bowl (game) — short, the trick's name.
var skaterTrickShout = map[int][2]string{
	trOllie:     {"¡Ollie!", "Ollie!"},
	trSpin180:   {"¡Un 180!", "A 180!"},
	trSpin360:   {"¡Un 360!", "A 360!"},
	trGrab:      {"¡Grab!", "Grab it!"},
	trKickflip:  {"¡Kickflip!", "Kickflip, yeah!"},
	trFlip:      {"¡Mortal atrás!", "Backflip!"},
	trWipeout:   {"¡Ay, mi cabeza!", "Ow, my head!"},
	trHandplant: {"¡Pino en el borde!", "Handplant!"},
	trSpin540:   {"¡Un 540!", "A 540!"},
	trHeelflip:  {"¡Heelflip!", "Heelflip, baby!"},
	trTailwhip:  {"¡Tailwhip!", "Tailwhip, yeah!"},
	trBarspin:   {"¡Barspin!", "Barspin, yeah!"},
	trNoHander:  {"¡Sin manos!", "No hands!"},
}

func init() {
	add := func(p [2]string) {
		if _, ok := langDataEN[p[0]]; !ok {
			langDataEN[p[0]] = p[1]
		}
	}
	for _, p := range skaterLinesAll {
		add(p)
	}
	for _, l := range skaterLinesOwn {
		for _, p := range l {
			add(p)
		}
	}
	for _, p := range skaterTrickShout {
		add(p)
	}
}

// ---- on the last card -------------------------------------------------------------------------------

const (
	finSayDur   = 2.2  // a line stays this long
	finSayStep  = 0.5  // a kid may start talking every this many seconds...
	finSayRest  = 5.0  // ...but after a line he keeps quiet at least this long
	finSayPad   = 6.0  // paper round the letters
	finSayTxtH  = 12.0 // the letters at scale 1, from the top of the paper's text row
	finSayMaxW  = 200.0
	finSayTail  = 5.0 // the tail under the paper, down to his head
	finSayGapHd = 4.0 // from the tip of the tail to the top of his box
	finSayZoneL = 252.0
	finSayZoneR = 708.0 // the texts of the card are all between these two: bubbles only outside them
)

// finSayEv: one line said by a rider on the card.
type finSayEv struct {
	rider int
	start float64
	n     int // the how-many-eth line he says (his lines in a random order without repeats)
}

type finSayMemo struct {
	upTo    int // candidates already decided
	evs     []finSayEv
	lastEnd [3]float64
	count   [3]int
}

var (
	finSayMu   sync.Mutex
	finSayMems [2]finSayMemo // [0] normal card, [1] secret one
)

// finSayBoxAt: the biggest bubble (finSayMaxW) of rider i at time t of the card, or ok=false when it does not
// fit there (over the middle of the card, off the screen). The bubble hangs over his head, as wide as it may be,
// slid sideways inside its side of the card if needed, but always over him. Pure.
func finSayBoxAt(i int, t float64) (b finRect, ok bool) {
	x0, y0, x1, _, on := finRideBox(i, t)
	if !on {
		return b, false
	}
	cx := math.Round((x0 + x1) / 2)
	w, h := finSayMaxW+4, finSayTxtH+2*finSayPad+4 // with the 2 px ink border
	bot := y0 - finSayGapHd - finSayTail
	b = finRect{cx - w/2, bot - h, cx + w/2, bot + finSayTail}
	lo, hi := 16.0, finSayZoneL
	if cx > screenW/2 {
		lo, hi = finSayZoneR, screenW-16
	}
	if b.x0 < lo {
		b.x0, b.x1 = lo, lo+w
	}
	if b.x1 > hi {
		b.x0, b.x1 = hi-w, hi
	}
	if b.x0 < lo || b.y0 < 16 || cx < b.x0+10 || cx > b.x1-10 {
		return b, false
	}
	return b, true
}

// finRect: a rectangle by its corners.
type finRect struct{ x0, y0, x1, y1 float64 }

func (a finRect) gap(b finRect) float64 {
	return math.Max(math.Max(a.x0-b.x1, b.x0-a.x1), math.Max(a.y0-b.y1, b.y0-a.y1))
}

// finMurrayZone: Murray (his sprite at scale 4 with a margin) and his bubble at its biggest, on each card.
func finMurrayZone(trueEnd bool) (spr, bub finRect) {
	footX, footY, _, _, _ := finMurrayGeom(trueEnd)
	spr = finRect{footX - 60, footY - 190, footX + 60, footY + 6}
	x0, y0, x1, y1 := finBubbleBox(trueEnd, finBubWrap, 3)
	return spr, finRect{x0, y0, x1, y1}
}

// finSayFits: whether rider i may talk from s for finSayDur: his bubble fits all along, >= 12 px from Murray and
// his bubble and from the bubbles already on (it may pass in front of another kid: it is drawn over them).
func finSayFits(m *finSayMemo, trueEnd bool, i int, s float64) bool {
	spr, bub := finMurrayZone(trueEnd)
	for tt := s; tt <= s+finSayDur+1e-9; tt += 1.0 / 15 {
		b, ok := finSayBoxAt(i, tt)
		if !ok || b.gap(spr) < 12 || b.gap(bub) < 12 {
			return false
		}
		for k := len(m.evs) - 1; k >= 0; k-- {
			e := m.evs[k]
			if e.start+finSayDur < s-finSayDur {
				break
			}
			if e.rider == i || tt < e.start || tt > e.start+finSayDur {
				continue
			}
			if ob, ok := finSayBoxAt(e.rider, tt); ok && b.gap(ob) < 12 {
				return false
			}
		}
	}
	return true
}

// finSayAt: what rider i is saying at time t of the card (ok=false: nothing), and since when. Pure.
func finSayAt(trueEnd bool, i int, t float64) (ev finSayEv, ok bool) {
	finSayMu.Lock()
	defer finSayMu.Unlock()
	mi := 0
	if trueEnd {
		mi = 1
	}
	m := &finSayMems[mi]
	first := finRidersFrom + 1
	for {
		s := first + float64(m.upTo)*finSayStep
		if s > t {
			break
		}
		for k := 0; k < 3; k++ { // who goes first turns round, so nobody hogs the card
			r := (m.upTo + k) % 3
			if s < m.lastEnd[r]+finSayRest || !finSayFits(m, trueEnd, r, s) {
				continue
			}
			m.evs = append(m.evs, finSayEv{rider: r, start: s, n: m.count[r]})
			m.count[r]++
			m.lastEnd[r] = s + finSayDur
		}
		m.upTo++
	}
	for k := len(m.evs) - 1; k >= 0; k-- {
		e := m.evs[k]
		if e.start+finSayDur < t-finSayDur {
			break
		}
		if e.rider == i && t >= e.start && t < e.start+finSayDur {
			return e, true
		}
	}
	return ev, false
}

// finSayLine: the n-th line rider i says (all of his, in a random order, before any repeats).
func finSayLine(i, n int) string {
	set := skaterLines(finRiders[i].slug)
	return set[finMurrayLineN(finMurraySeed+int64(i+1)*104729, n, len(set))][0]
}

// finSayRect: the bubble of a line of width w (paper and border, without the tail) for rider i at time t.
func finSayRect(i int, t, w float64) (bx, by, bw, bh, tipX float64, ok bool) {
	b, ok := finSayBoxAt(i, t)
	if !ok {
		return
	}
	x0, _, x1, _, _ := finRideBox(i, t)
	cx := math.Round((x0 + x1) / 2)
	bw, bh = math.Ceil(w)+2*finSayPad, finSayTxtH+2*finSayPad
	bx = math.Round(math.Max(b.x0+2, math.Min(b.x1-2-bw, cx-bw/2)))
	by = b.y0 + 2
	return bx, by, bw, bh, cx, true
}

// drawFinRiderSays: the little bubbles of the kids on the card (after the kids: over them).
func (g *Game) drawFinRiderSays(dst *ebiten.Image, e *endState) {
	if e.t < finRidersFrom {
		return
	}
	ink, paper := rgb(0x0a0a0c), rgb(0xf6f2e6)
	for i := range finRiders {
		ev, ok := finSayAt(e.trueEnd, i, e.t)
		if !ok {
			continue
		}
		line := tr(finSayLine(i, ev.n))
		bx, by, bw, bh, tip, ok := finSayRect(i, e.t, g.textW(line, 1))
		if !ok {
			continue
		}
		fillRect(dst, bx-2, by-2, bw+4, bh+4, ink)
		fillRect(dst, bx, by, bw, bh, paper)
		tip = math.Max(bx+4, math.Min(bx+bw-4, tip))
		tip = math.Round(tip)
		for r := 0.0; r < finSayTail; r++ { // a little tail down to his head, on whole pixels
			wd := finSayTail - r
			x0 := tip - math.Floor(wd/2)
			fillRect(dst, x0-1, by+bh+r, wd+2, 1, ink)
			if r < finSayTail-1 {
				fillRect(dst, x0, by+bh+r, wd, 1, paper)
			}
		}
		x, y := math.Round(bx+finSayPad), math.Round(by+finSayPad-2)
		g.recText(dst, "riderSay", line, x, y, 1)
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, y)
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(dst, line, g.face, op)
	}
}
