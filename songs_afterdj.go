package main

import (
	"math/rand"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// The music after the DJ dies. It used to be punk from the old step sequencer:
// a kick on every beat and an 8-bar loop ("chunta chunta"). Now it is six songs
// written here: lounge, funk, a ballad, soft disco, ambient and bossa. Each one
// has 32 bars in four sections (intro A, A answered a third higher, B with other
// chords, a break and the theme again), a tune made from a motif that follows the
// chords, and a drum part that changes by section: rests, syncopations, fills and
// whole bars without a kick. One plays at random when the DJ dies, never the same
// as the last one; when the song changes it is another one.
// ---------------------------------------------------------------------------

const (
	tdjBars = 32   // 4 sections of 8 bars
	tdjRest = -999 // a rest in a motif
	tdjRMS  = 0.14 // a bit quieter than the music of the parties (finishGenre uses 0.17)
)

// tdjCtx: where a bar is and which chord it plays.
type tdjCtx struct {
	sec, bar, abs int   // section (0-3), bar inside it (0-7) and bar of the song (0-31)
	deg           int   // scale degree of the chord (0 = the tonic)
	chord         []int // root, third, fifth and seventh from G3 up
	root          int   // the bass note, from E2 up
	sc            scaleDef
}

type tdjLayer struct {
	inst string
	vol  float64
	mel  bool
	bar  func(c tdjCtx) string // the note text of one bar (4 beats)
}

type tdjDrum struct {
	vol float64
	bar func(c tdjCtx) string // 16 sixteenths: a kit letter or '.'
}

type tdjSong struct {
	id       string
	bpm      float64
	sc       scaleDef
	swing    int
	prog     [4][8]int // the chords, as scale degrees, of the four sections
	layers   []tdjLayer
	drums    []tdjDrum
	echo, fb float64
	wet      float64
}

// tdjM: one note of a motif, in scale steps over the root of the chord (tdjRest = a rest).
type tdjM struct {
	d float64
	s int
}

// tdjTune: the motifs of a tune; plan says which one each bar plays.
type tdjTune struct{ a, a2, b, b2, cad, fill []tdjM }

func (t tdjTune) plan(c tdjCtx) []tdjM {
	last := c.bar == 7
	alt := func(x, y []tdjM) []tdjM {
		if c.bar%2 == 0 {
			return x
		}
		return y
	}
	switch c.sec {
	case 0: // intro: two bars without the tune
		if c.bar < 2 {
			return nil
		}
		if last {
			return t.cad
		}
		return alt(t.a, t.a2)
	case 1: // the theme, answered a third higher
		if last {
			return t.cad
		}
		if c.bar == 3 {
			return t.fill
		}
		if c.bar >= 4 {
			return tdjUp(alt(t.a, t.a2), 2)
		}
		return alt(t.a, t.a2)
	case 2: // the B part, other chords and another motif
		if last {
			return t.fill
		}
		return alt(t.b, t.b2)
	}
	// the break (only a reply now and then) and the theme once more
	if c.bar < 4 {
		if c.bar%2 == 1 {
			return t.b2
		}
		return nil
	}
	if last {
		return t.cad
	}
	return alt(t.a, t.a2)
}

func tdjUp(m []tdjM, n int) []tdjM {
	out := make([]tdjM, len(m))
	for i, x := range m {
		out[i] = x
		if x.s != tdjRest {
			out[i].s += n
		}
	}
	return out
}

// tdjDegNote: the note of degree deg of the scale, at low or the first one above it.
func tdjDegNote(sc scaleDef, deg, low int) int {
	d := ((deg % len(sc.steps)) + len(sc.steps)) % len(sc.steps)
	pc := (sc.rootPC + sc.steps[d]) % 12
	m := low
	for ((m%12)+12)%12 != pc {
		m++
	}
	return m
}

// tdjStep moves a note of the scale n steps along the scale.
func tdjStep(sc scaleDef, m, n int) int {
	for ; n > 0; n-- {
		m++
		for !sc.has(m) {
			m++
		}
	}
	for ; n < 0; n++ {
		m--
		for !sc.has(m) {
			m--
		}
	}
	return m
}

func tdjChord(sc scaleDef, deg, low int) []int {
	r := tdjDegNote(sc, deg, low)
	return []int{r, tdjStep(sc, r, 2), tdjStep(sc, r, 4), tdjStep(sc, r, 6)}
}

// tdjMelody plays the motif of each bar from the root of its chord, at low or above.
func tdjMelody(low int, t tdjTune) func(c tdjCtx) string {
	return func(c tdjCtx) string {
		ms := t.plan(c)
		if ms == nil {
			return "R:4"
		}
		a := tdjDegNote(c.sc, c.deg, low)
		p := make([]string, len(ms))
		for i, m := range ms {
			if m.s == tdjRest {
				p[i] = "R" + gDur(m.d)
			} else {
				p[i] = midiName(tdjStep(c.sc, a, m.s)) + gDur(m.d)
			}
		}
		return strings.Join(p, " ")
	}
}

// tdjRhythm writes a chord with a rhythm: X is the chord ("X:1.5 R:0.5 X:2").
func tdjRhythm(ch []int, rhythm string) string {
	return strings.ReplaceAll(rhythm, "X", gChord(ch...))
}

// tdjBass writes a bass line: r root, f fifth, o octave, u the step above the root, d the step below.
func tdjBass(c tdjCtx, line string) string {
	r := c.root
	names := map[string]string{"r": midiName(r), "f": midiName(tdjStep(c.sc, r, 4)), "o": midiName(r + 12),
		"u": midiName(tdjStep(c.sc, r, 1)), "d": midiName(tdjStep(c.sc, r, -1)), "t": midiName(tdjStep(c.sc, r, 2))}
	toks := strings.Fields(line)
	for i, tok := range toks {
		k := tok[:strings.Index(tok, ":")]
		if n, ok := names[k]; ok {
			toks[i] = n + tok[len(k):]
		}
	}
	return strings.Join(toks, " ")
}

// tdjPick: one of the patterns by bar (they cycle).
func tdjPick(i int, p ...string) string { return p[i%len(p)] }

var tdjSilent = dots(16)

// tdjSongs: the songs after the DJ, in the order of afterDJPlaylist.
var tdjSongs []tdjSong

func init() {
	// ---- lounge: a soft horn over a jazzy piano, brushes and a rim (F major)
	tdjSongs = append(tdjSongs, tdjSong{id: "tdj_lounge", bpm: 92, sc: scFMajor, swing: 2, echo: 0.75, fb: 0.3, wet: 0.18,
		prog: [4][8]int{{0, 5, 1, 4, 0, 5, 1, 4}, {0, 5, 1, 4, 3, 2, 1, 4}, {3, 2, 1, 5, 3, 2, 1, 4}, {1, 4, 0, 5, 1, 4, 0, 0}},
		layers: []tdjLayer{
			{"horn", 0.3, true, tdjMelody(65, tdjTune{
				a:    []tdjM{{1.5, 4}, {0.5, 2}, {1, 0}, {1, tdjRest}},
				a2:   []tdjM{{0.5, 2}, {0.5, 4}, {1, 7}, {1, 6}, {1, 4}},
				b:    []tdjM{{2, 4}, {1, 5}, {1, 4}},
				b2:   []tdjM{{1, 2}, {1, 1}, {2, 0}},
				cad:  []tdjM{{3, 0}, {1, tdjRest}},
				fill: []tdjM{{0.5, 4}, {0.5, 5}, {0.5, 4}, {0.5, 2}, {2, 0}}})},
			{"piano", 0.22, false, func(c tdjCtx) string {
				if c.sec == 0 && c.bar < 2 {
					return tdjRhythm(c.chord, "X:4")
				}
				return tdjRhythm(c.chord, tdjPick(c.abs, "X:1.5 X:0.5 R:0.5 X:1 R:0.5", "R:0.5 X:1 R:0.5 X:1.5 R:0.5"))
			}},
			{"ebass", 0.42, false, func(c tdjCtx) string {
				switch {
				case c.sec == 0 && c.bar < 2:
					return "R:4"
				case c.sec == 0 || (c.sec == 3 && c.bar < 4):
					return tdjBass(c, "r:2 f:2")
				}
				return tdjBass(c, tdjPick(c.bar, "r:1 R:0.5 r:0.5 f:1 o:1", "r:1.5 t:0.5 f:1 d:1"))
			}},
			{"pad", 0.16, false, func(c tdjCtx) string {
				if c.sec >= 2 || (c.sec == 0 && c.bar < 4) {
					return tdjRhythm(c.chord[:3], "X:4")
				}
				return "R:4"
			}},
		},
		drums: []tdjDrum{
			{0.45, func(c tdjCtx) string { // soft kick
				switch {
				case c.sec == 0 && c.bar < 2, c.sec == 3 && c.bar < 4:
					return tdjSilent
				case c.bar == 7:
					return "K.....K...K.K..."
				}
				return tdjPick(c.bar, "K.........K.....", "K......K..K.....", "K.........K.....", "K.....K.........")
			}},
			{0.3, func(c tdjCtx) string { // rim
				if c.sec == 0 {
					return tdjSilent
				}
				if c.sec == 2 {
					return "....r.....r.r..."
				}
				return "....r.......r..."
			}},
			{0.1, func(c tdjCtx) string { // hats
				if c.sec == 0 && c.bar < 4 {
					return tdjSilent
				}
				return "h.h.h.h.h.h.h.h."
			}},
			{0.08, func(c tdjCtx) string { // shaker in the B part
				if c.sec == 2 {
					return "s.ss.ss.s.ss.ss."
				}
				return tdjSilent
			}},
		}})

	// ---- funk: a syncopated bass, guitar chops, a brass hook and ghost notes (E dorian, written in D major)
	tdjSongs = append(tdjSongs, tdjSong{id: "tdj_funk", bpm: 104, sc: scDMajor, swing: 2, echo: 0.375, fb: 0.2, wet: 0.1,
		prog: [4][8]int{{1, 1, 4, 4, 1, 1, 4, 4}, {1, 1, 4, 4, 1, 1, 3, 4}, {3, 2, 5, 4, 3, 2, 5, 4}, {1, 4, 1, 4, 3, 2, 1, 4}},
		layers: []tdjLayer{
			{"brass", 0.24, true, tdjMelody(64, tdjTune{
				a:    []tdjM{{0.5, 0}, {0.25, tdjRest}, {0.25, 2}, {0.5, 4}, {0.5, tdjRest}, {0.25, 5}, {0.25, 4}, {0.5, 2}, {1, tdjRest}},
				a2:   []tdjM{{0.75, 4}, {0.25, 6}, {0.5, 7}, {0.5, tdjRest}, {0.5, 6}, {0.5, 4}, {1, tdjRest}},
				b:    []tdjM{{1, 7}, {0.5, 6}, {0.5, 4}, {1, 5}, {1, tdjRest}},
				b2:   []tdjM{{0.5, 4}, {0.5, 2}, {0.5, 4}, {0.5, 5}, {2, 4}},
				cad:  []tdjM{{0.5, 2}, {0.5, 1}, {1, 0}, {2, tdjRest}},
				fill: []tdjM{{0.25, 0}, {0.25, 2}, {0.25, 4}, {0.25, 6}, {0.5, 7}, {0.5, tdjRest}, {2, tdjRest}}})},
			{"guitar", 0.2, false, func(c tdjCtx) string {
				if c.sec == 0 && c.bar < 2 {
					return "R:4"
				}
				return tdjRhythm(c.chord[1:], tdjPick(c.abs,
					"R:0.5 X:0.25 R:0.25 R:0.5 X:0.25 X:0.25 R:0.5 X:0.25 R:0.25 R:0.5 X:0.5",
					"X:0.25 R:0.75 X:0.25 R:0.25 X:0.25 R:0.25 R:0.5 X:0.25 X:0.25 R:1"))
			}},
			{"ebass", 0.5, false, func(c tdjCtx) string {
				if c.sec == 2 || (c.sec == 3 && c.bar < 4) {
					return tdjBass(c, tdjPick(c.bar, "r:1 R:0.5 r:0.5 f:1 o:1", "r:1.5 u:0.5 t:1 f:1"))
				}
				return tdjBass(c, tdjPick(c.bar, "r:0.75 r:0.25 R:0.5 o:0.25 R:0.25 r:0.5 R:0.25 f:0.25 t:0.5 r:0.5",
					"r:0.5 R:0.25 r:0.25 o:0.5 R:0.5 r:0.25 r:0.25 R:0.5 f:0.5 d:0.5"))
			}},
			{"pad", 0.12, false, func(c tdjCtx) string {
				if c.sec == 2 {
					return tdjRhythm(c.chord[:3], "X:4")
				}
				return "R:4"
			}},
		},
		drums: []tdjDrum{
			{0.55, func(c tdjCtx) string {
				switch {
				case c.sec == 3 && c.bar < 4:
					return tdjPick(c.bar, "k...............", tdjSilent)
				case c.bar == 7:
					return "k.k.......k.k.k."
				}
				return tdjPick(c.bar, "k.........k..k..", "k..k......k.....", "k.........k..k..", "k..k....k.k.....")
			}},
			{0.42, func(c tdjCtx) string {
				switch {
				case c.sec == 0 && c.bar < 2, c.sec == 3 && c.bar < 4:
					return tdjSilent
				case c.bar == 7:
					return "....n.....nn.nnn"
				}
				return "....n.......n..."
			}},
			{0.16, func(c tdjCtx) string { return tdjPick(c.bar, ".g.....g.g....g.", "...g...g.g......") }},
			{0.08, func(c tdjCtx) string {
				if c.sec == 1 || c.sec == 2 {
					return "hhhhhhhhhhhhhhhh"
				}
				return "h.h.h.h.h.h.h.h."
			}},
			{0.1, func(c tdjCtx) string {
				if c.sec == 2 && c.bar%2 == 1 {
					return "..............o."
				}
				return tdjSilent
			}},
		}})

	// ---- ballad: a piano tune over an arpeggio, a pad and very few drums (C major)
	tdjSongs = append(tdjSongs, tdjSong{id: "tdj_balada", bpm: 76, sc: scCMajor, echo: 0.75, fb: 0.35, wet: 0.22,
		prog: [4][8]int{{0, 4, 5, 3, 0, 4, 3, 4}, {0, 4, 5, 3, 0, 4, 3, 0}, {5, 3, 0, 4, 5, 3, 1, 4}, {3, 4, 2, 5, 3, 4, 0, 0}},
		layers: []tdjLayer{
			{"piano", 0.32, true, tdjMelody(67, tdjTune{
				a:    []tdjM{{1, 4}, {1, 2}, {1.5, 4}, {0.5, 5}},
				a2:   []tdjM{{2, 4}, {1, 2}, {1, 1}},
				b:    []tdjM{{1, 7}, {1, 6}, {1, 4}, {1, 5}},
				b2:   []tdjM{{3, 4}, {1, tdjRest}},
				cad:  []tdjM{{4, 0}},
				fill: []tdjM{{1, 2}, {1, 4}, {2, 7}}})},
			{"piano", 0.15, false, func(c tdjCtx) string {
				ch := c.chord
				idx := [][]int{{0, 2, 4, 5, 2, 4, 5, 4}, {0, 2, 4, 2, 5, 4, 2, 4}}[c.abs%2]
				tones := []int{ch[0], ch[1], ch[2], ch[0] + 12, ch[1] + 12, ch[2] + 12}
				return arpBar(tones, idx, 0.5)
			}},
			{"pad", 0.16, false, func(c tdjCtx) string {
				if c.sec == 0 {
					return "R:4"
				}
				return tdjRhythm(c.chord[:3], "X:4")
			}},
			{"ebass", 0.34, false, func(c tdjCtx) string {
				switch {
				case c.sec == 0 && c.bar < 4:
					return "R:4"
				case c.sec == 2:
					return tdjBass(c, "r:2 f:2")
				}
				return tdjBass(c, "r:4")
			}},
		},
		drums: []tdjDrum{
			{0.36, func(c tdjCtx) string {
				switch {
				case c.sec == 0, c.sec == 3 && c.bar < 4:
					return tdjSilent
				case c.sec == 2:
					return tdjPick(c.bar, "K.....K.K.......", "K.........K.....")
				}
				return "K.......K......."
			}},
			{0.24, func(c tdjCtx) string {
				if c.sec == 0 || (c.sec == 3 && c.bar < 4) {
					return tdjSilent
				}
				return "........r......."
			}},
			{0.14, func(c tdjCtx) string {
				if c.sec == 2 {
					return "j...j...j...j..."
				}
				return tdjSilent
			}},
			{0.3, func(c tdjCtx) string {
				if c.sec == 2 && c.bar == 7 {
					return "........t.t.tttt"
				}
				return tdjSilent
			}},
		}})

	// ---- soft disco: octave bass, strings, offbeat hats; the kick only goes four to the floor in a few bars (A minor)
	tdjSongs = append(tdjSongs, tdjSong{id: "tdj_disco", bpm: 112, sc: scAMinor, echo: 0.5, fb: 0.25, wet: 0.14,
		prog: [4][8]int{{0, 3, 6, 2, 0, 3, 6, 2}, {0, 3, 6, 2, 0, 3, 6, 4}, {5, 4, 3, 2, 5, 4, 1, 4}, {0, 3, 6, 2, 5, 3, 4, 4}},
		layers: []tdjLayer{
			{"square", 0.16, true, tdjMelody(69, tdjTune{
				a:    []tdjM{{0.5, 4}, {0.5, tdjRest}, {0.5, 4}, {0.5, 5}, {1, 4}, {1, 2}},
				a2:   []tdjM{{0.5, 7}, {0.5, tdjRest}, {0.5, 7}, {0.5, 6}, {1, 4}, {1, tdjRest}},
				b:    []tdjM{{1.5, 2}, {0.5, 4}, {1, 7}, {1, 4}},
				b2:   []tdjM{{0.5, 5}, {0.5, 4}, {0.5, 2}, {0.5, 0}, {2, tdjRest}},
				cad:  []tdjM{{1, 4}, {1, 2}, {2, 0}},
				fill: []tdjM{{0.5, 0}, {0.5, 2}, {0.5, 4}, {0.5, 7}, {1, 9}, {1, tdjRest}}})},
			{"strings", 0.16, false, func(c tdjCtx) string {
				if c.sec == 0 {
					return "R:4"
				}
				return tdjRhythm(c.chord[:3], "X:4")
			}},
			{"synbass", 0.42, false, func(c tdjCtx) string {
				if c.sec == 0 || (c.sec == 3 && c.bar < 4) {
					return tdjBass(c, tdjPick(c.bar, "r:1 R:0.5 r:0.5 o:1 R:1", "r:1.5 f:0.5 R:1 o:1"))
				}
				return tdjBass(c, "r:0.5 o:0.5 r:0.5 o:0.5 r:0.5 o:0.5 f:0.5 o:0.5")
			}},
			{"pizz", 0.14, false, func(c tdjCtx) string {
				if c.sec == 0 && c.bar < 4 {
					return tdjRhythm(c.chord[1:], "X:1 R:3")
				}
				return tdjRhythm(c.chord[1:], "R:0.5 X:0.5 R:0.5 X:0.5 R:0.5 X:0.5 R:0.5 X:0.5")
			}},
		},
		drums: []tdjDrum{
			{0.5, func(c tdjCtx) string {
				switch {
				case c.sec == 0 && c.bar < 4, c.sec == 3 && c.bar < 4:
					return tdjSilent
				case (c.sec == 1 && c.bar >= 4) || (c.sec == 3 && c.bar >= 4 && c.bar < 7):
					return "k...k...k...k..." // the only bars four to the floor
				case c.bar == 7:
					return "k.....k.k.k.k..."
				}
				return tdjPick(c.bar, "k.......k.......", "k.....k.k.......", "k.......k.....k.")
			}},
			{0.3, func(c tdjCtx) string {
				if c.sec == 0 {
					return tdjSilent
				}
				return "....c.......c..."
			}},
			{0.07, func(c tdjCtx) string {
				if c.sec == 0 && c.bar < 4 {
					return tdjSilent
				}
				return "..o...o...o...o."
			}},
			{0.07, func(c tdjCtx) string {
				if c.sec == 2 {
					return "h.hhh.hhh.hhh.hh"
				}
				return tdjSilent
			}},
		}})

	// ---- ambient: harp arpeggios, a celesta, a pad; the drums are a heartbeat and a shaker (G major)
	tdjSongs = append(tdjSongs, tdjSong{id: "tdj_ambient", bpm: 84, sc: scGMajor, echo: 1.0, fb: 0.4, wet: 0.28,
		prog: [4][8]int{{0, 2, 5, 3, 0, 2, 3, 4}, {0, 2, 5, 3, 0, 2, 3, 4}, {5, 3, 0, 4, 5, 3, 1, 4}, {3, 4, 5, 2, 3, 4, 0, 0}},
		layers: []tdjLayer{
			{"celesta", 0.26, true, tdjMelody(74, tdjTune{
				a:    []tdjM{{2, 4}, {2, 7}},
				a2:   []tdjM{{1, 9}, {1, 7}, {2, 4}},
				b:    []tdjM{{3, 2}, {1, 4}},
				b2:   []tdjM{{2, tdjRest}, {1, 7}, {1, 9}},
				cad:  []tdjM{{4, 7}},
				fill: []tdjM{{1, 4}, {1, 7}, {1, 9}, {1, 11}}})},
			{"harp", 0.17, false, func(c tdjCtx) string {
				ch := c.chord
				tones := []int{ch[0], ch[1], ch[2], ch[3], ch[0] + 12, ch[1] + 12}
				if c.sec == 0 && c.bar < 4 || c.sec == 3 && c.bar < 4 {
					return arpBar(tones, []int{0, 2, 4, 2, 1, 3, 5, 3}, 0.5)
				}
				return arpBar(tones, [][]int{{0, 1, 2, 3, 4, 5, 4, 3, 2, 1, 2, 3, 4, 3, 2, 1}, {0, 2, 4, 5, 4, 2, 1, 3, 5, 3, 1, 2, 4, 2, 1, 0}}[c.abs%2], 0.25)
			}},
			{"pad", 0.22, false, func(c tdjCtx) string { return tdjRhythm(c.chord[:3], "X:4") }},
			{"ebass", 0.3, false, func(c tdjCtx) string {
				if c.sec == 0 {
					return "R:4"
				}
				return tdjBass(c, tdjPick(c.bar, "r:4", "r:3 f:1"))
			}},
		},
		drums: []tdjDrum{
			{0.32, func(c tdjCtx) string {
				switch {
				case c.sec == 0, c.sec == 3 && c.bar < 4:
					return tdjSilent
				case c.sec == 2:
					return tdjPick(c.bar, "K.......K.....K.", "K.........K.....")
				}
				return tdjPick(c.bar, "K...............", "K.......K.......")
			}},
			{0.08, func(c tdjCtx) string {
				if c.sec == 2 || (c.sec == 3 && c.bar >= 4) {
					return "s.s.s.s.s.s.s.s."
				}
				return tdjSilent
			}},
			{0.1, func(c tdjCtx) string {
				if c.sec == 1 {
					return "......w.......w."
				}
				return tdjSilent
			}},
		}})

	// ---- bossa: a nylon guitar, a flute, a plucked bass and the clave on the rim (G major)
	tdjSongs = append(tdjSongs, tdjSong{id: "tdj_bossa", bpm: 100, sc: scGMajor, echo: 0.5, fb: 0.25, wet: 0.16,
		prog: [4][8]int{{0, 0, 1, 4, 0, 0, 1, 4}, {0, 5, 1, 4, 0, 5, 1, 4}, {3, 3, 2, 5, 1, 1, 4, 4}, {3, 4, 2, 5, 1, 4, 0, 0}},
		layers: []tdjLayer{
			{"flute", 0.26, true, tdjMelody(67, tdjTune{
				a:    []tdjM{{1.5, 4}, {0.5, 4}, {1, 5}, {1, 4}},
				a2:   []tdjM{{0.5, 2}, {1, 4}, {0.5, 2}, {2, 0}},
				b:    []tdjM{{1, 7}, {0.5, 6}, {1, 4}, {1.5, 2}},
				b2:   []tdjM{{1, tdjRest}, {0.5, 1}, {0.5, 2}, {2, 4}},
				cad:  []tdjM{{3, 0}, {1, tdjRest}},
				fill: []tdjM{{0.5, 0}, {0.5, 1}, {0.5, 2}, {0.5, 4}, {2, 5}}})},
			{"guitar", 0.2, false, func(c tdjCtx) string {
				return tdjRhythm(c.chord, tdjPick(c.abs, "X:0.75 X:0.75 R:0.5 X:0.75 X:0.75 R:0.5", "R:0.5 X:0.75 X:0.75 R:0.5 X:1 R:0.5"))
			}},
			{"pizz", 0.45, false, func(c tdjCtx) string {
				if c.sec == 0 && c.bar < 2 {
					return "R:4"
				}
				return tdjBass(c, tdjPick(c.bar, "r:1.5 f:0.5 f:1.5 r:0.5", "r:1.5 f:0.5 t:1 f:1"))
			}},
			{"pad", 0.1, false, func(c tdjCtx) string {
				if c.sec == 2 {
					return tdjRhythm(c.chord[:3], "X:4")
				}
				return "R:4"
			}},
		},
		drums: []tdjDrum{
			{0.3, func(c tdjCtx) string {
				switch {
				case c.sec == 0 && c.bar < 2, c.sec == 3 && c.bar < 4:
					return tdjSilent
				case c.bar == 7:
					return "K.....K.K..K.K.."
				}
				return tdjPick(c.bar, "K.....K.K.....K.", "K.....K.K.......")
			}},
			{0.26, func(c tdjCtx) string { // 3-2 clave
				if c.sec == 0 && c.bar < 2 {
					return tdjSilent
				}
				return tdjPick(c.bar, "r..r..r.........", "....r...r.......")
			}},
			{0.07, func(c tdjCtx) string {
				if c.sec == 0 || (c.sec == 3 && c.bar < 4) {
					return tdjSilent
				}
				return "s.ss.ss.s.ss.ss."
			}},
		}})

	for _, s := range tdjSongs {
		genreSongs[s.id] = s.genre()
		afterDJTable[s.id] = genreInfo{s.id, "", false, s.bpm, s.bpm - 8, s.bpm + 8}
	}
}

// afterDJTable: tempo limits of the songs after the DJ (they are not the genre of any party, so not in genreTable).
var afterDJTable = map[string]genreInfo{}

// isAfterDJ: a song of the music after the DJ.
func isAfterDJ(style string) bool { _, ok := afterDJTable[style]; return ok }

// genre turns the song into the note text of genres.go, bar by bar.
func (s tdjSong) genre() genreSong {
	ctx := func(abs int) tdjCtx {
		sec, bar := abs/8, abs%8
		deg := s.prog[sec][bar]
		return tdjCtx{sec: sec, bar: bar, abs: abs, deg: deg, chord: tdjChord(s.sc, deg, 55), root: tdjDegNote(s.sc, deg, 40), sc: s.sc}
	}
	gs := genreSong{bpm: s.bpm, beats: 4 * tdjBars, scale: s.sc, swing: s.swing, noRetro: true, echo: s.echo, fb: s.fb, wet: s.wet}
	for _, l := range s.layers {
		text := perBar(tdjBars, func(i int) string {
			b := l.bar(ctx(i))
			var sum float64
			for _, n := range parseNotes(b) {
				sum += n.dur
			}
			if sum < 3.999 || sum > 4.001 {
				panic("songs_afterdj: " + s.id + " layer " + l.inst + " bar does not last 4 beats: " + b)
			}
			return b
		})
		gs.tracks = append(gs.tracks, gTrack{l.inst, text, l.vol, l.mel})
	}
	for _, d := range s.drums {
		var sb strings.Builder
		for i := 0; i < tdjBars; i++ {
			p := d.bar(ctx(i))
			if len(p) != 16 {
				panic("songs_afterdj: " + s.id + " drum bar is not 16 steps: " + p)
			}
			sb.WriteString(p)
		}
		gs.perc = append(gs.perc, gPerc{sb.String(), d.vol})
	}
	return gs
}

// afterDJPlaylist: the songs that play once the DJ is dead.
func afterDJPlaylist() []songSpec {
	out := make([]songSpec, len(tdjSongs))
	for i, s := range tdjSongs {
		out[i] = songSpec{style: s.id, bpm: s.bpm, seed: int64(9100 + i), bars: tdjBars}
	}
	return out
}

// afterDJ remembers which song after the DJ is playing (cur), the one that was chosen for the next time (next, -1 =
// not chosen yet) and the songIdx it was chosen for. cur survives the end of a party, so the next dead DJ never
// starts with the song the last one had.
var afterDJ = struct {
	cur, next, idx int
	on             bool
}{cur: -1, next: -1}

// afterDJNext: the next song after the DJ, chosen once (so it can be rendered beforehand), never the one playing.
func afterDJNext(rng *rand.Rand) int {
	n := len(tdjSongs)
	if afterDJ.next < 0 || afterDJ.next == afterDJ.cur {
		if rng == nil {
			rng = rand.New(rand.NewSource(int64(afterDJ.cur + 7)))
		}
		k := rng.Intn(n)
		if k == afterDJ.cur {
			k = (k + 1 + rng.Intn(n-1)) % n
		}
		afterDJ.next = k
	}
	return afterDJ.next
}

// afterDJSong: the song for the dead DJ now. A new one (not the last) when he dies and every time the song changes.
func (g *Game) afterDJSong() songSpec {
	if !afterDJ.on || afterDJ.idx != g.songIdx {
		afterDJ.cur = afterDJNext(g.rng)
		afterDJ.next = -1
		afterDJ.on, afterDJ.idx = true, g.songIdx
	}
	return afterDJPlaylist()[afterDJ.cur]
}

// afterDJIdle: the DJ is alive (a new party): the next death chooses again.
func afterDJIdle() { afterDJ.on = false }

// legacyDJFile: the punk songs that older versions wrote to DJ/ (punk_1.mp3 ... punk_6.mp3, the "chunta chunta" ones):
// they are skipped so the songs above play; any other mp3 the user drops in DJ/ still wins.
var legacyDJFile = regexp.MustCompile(`(?i)^punk_\d+\.mp3$`)
