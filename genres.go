package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Music genres: every party has its own style (reggaeton on the beach, metal in
// the cemetery, a waltz under the Eiffel Tower...). The old step sequencer in
// music2.go keeps house, electro, techno, disco, reggaeton and punk; the other
// genres are written here as note text (see songs.go) and played with a few
// more instruments and a full drum kit. When the user drops a real mp3 in
// music/ it wins over all of this (musiclib.go).
// ---------------------------------------------------------------------------

// ---- more instruments -------------------------------------------------------------------------

func relEnv(t, dur, rel float64) float64 { return smooth01((dur + rel - t) / rel) }

// sawSum is a band-limited saw: sin(k*ph)/k for k = 1..harmonics. The sines of the multiples come from one
// sin and one cos with the rotation recurrence s(k+1) = 2cos(ph)*s(k) - s(k-1), which is several times faster.
func sawSum(ph float64, harmonics int) float64 {
	s1, c := math.Sincos(ph)
	c2 := 2 * c
	prev, cur := 0.0, s1
	v := s1
	for k := 2; k <= harmonics; k++ {
		prev, cur = cur, c2*cur-prev
		v += cur / float64(k)
	}
	return v * 0.62
}

var hornWeights = func() (w [6]float64) {
	for k := range w {
		w[k] = 1 / math.Pow(float64(k+1), 1.4)
	}
	return
}()

func init() {
	// French horn: a soft brass with a slow attack, a gentle vibrato and few harmonics
	instruments["horn"] = instrument{sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.25) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			vib := 1 + 0.0035*math.Sin(2*math.Pi*5.2*t)*smooth01((t-0.25)/0.4)
			ph := 2 * math.Pi * f * t * vib
			bright := 0.55 + 0.45*smooth01(t/0.3)
			var v float64
			w := 1.0
			for k := 1; k <= 6; k++ {
				v += math.Sin(float64(k)*ph) * w * hornWeights[k-1]
				w *= bright
			}
			out[i] = v * envAR(t, dur+0.04, 0.07, 0.18) * vol * 0.55
		}
		return out
	}}
	// harp: a clean pluck whose upper partials die first and a long ring
	instruments["harp"] = instrument{play: func(f, dur, vol float64) []float64 {
		n := int(math.Min(dur+1.1, 2.4) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			ph := 2 * math.Pi * f * t
			v := math.Sin(ph)*math.Exp(-1.9*t) + 0.42*math.Sin(2*ph)*math.Exp(-3.6*t) + 0.18*math.Sin(3*ph)*math.Exp(-6*t) + 0.07*math.Sin(4*ph)*math.Exp(-9*t)
			out[i] = v * math.Min(1, t/0.002) * smooth01((float64(n-i))/(0.25*sampleRate)) * vol * 0.8
		}
		return out
	}}
	// celesta: a bell-like sparkle, quick to fade
	instruments["celesta"] = instrument{play: func(f, dur, vol float64) []float64 {
		n := int(1.3 * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			ph := 2 * math.Pi * f * t
			v := math.Sin(ph)*math.Exp(-3.2*t) + 0.3*math.Sin(4*ph)*math.Exp(-8*t) + 0.12*math.Sin(2.76*ph)*math.Exp(-11*t)
			out[i] = v * math.Min(1, t/0.002) * smooth01(float64(n-i)/(0.2*sampleRate)) * vol * 0.8
		}
		return out
	}}
	instruments["piano"] = instrument{play: func(f, dur, vol float64) []float64 {
		n := int(math.Min(dur+0.7, 2.2) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			ph := 2 * math.Pi * f * t
			v := math.Sin(ph)*math.Exp(-2.2*t) + 0.5*math.Sin(2*ph)*math.Exp(-3.4*t) + 0.24*math.Sin(3*ph)*math.Exp(-5*t) + 0.1*math.Sin(4*ph)*math.Exp(-7*t)
			out[i] = v * math.Min(1, t/0.003) * relEnv(t, dur, 0.35) * vol * 0.8
		}
		return out
	}}
	instruments["ebass"] = instrument{sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.06) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			ph := 2 * math.Pi * f * t
			v := math.Sin(ph) + 0.5*math.Sin(2*ph)*math.Exp(-4*t) + 0.22*math.Sin(3*ph)*math.Exp(-7*t) + 0.1*math.Sin(4*ph)*math.Exp(-9*t)
			out[i] = v * envAR(t, dur, 0.004, 0.05) * (0.7 + 0.3*math.Exp(-6*t)) * vol * 0.8
		}
		return out
	}}
	// 808: a sine with a quick pitch drop and a long soft tail
	instruments["sub"] = instrument{play: func(f, dur, vol float64) []float64 {
		n := int(math.Min(dur+0.25, 1.4) * sampleRate)
		out := make([]float64, n)
		ph := 0.0
		for i := range out {
			t := float64(i) / sampleRate
			ph += 2 * math.Pi * f * (1 + 0.9*math.Exp(-t*38)) / sampleRate
			v := math.Tanh(1.5*math.Sin(ph)) * 0.8
			out[i] = v * math.Exp(-1.6*t) * math.Min(1, t/0.002) * relEnv(t, dur, 0.18) * vol
		}
		return out
	}}
	// distorted guitar: a saw pushed through a soft clipper; short notes are palm muted
	instruments["dist"] = instrument{sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.08) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			ph := 2 * math.Pi * f * t
			v := math.Tanh(2.6*sawSum(ph, 7)) * 0.62
			env := envAR(t, dur, 0.003, 0.05)
			if dur < 0.4 {
				env *= math.Exp(-9 * t)
			}
			out[i] = v * env * vol
		}
		return out
	}}
	// plucked string (Karplus-Strong): nylon / steel guitar
	instruments["guitar"] = instrument{play: func(f, dur, vol float64) []float64 {
		period := int(sampleRate / f)
		if period < 2 {
			period = 2
		}
		n := int(math.Min(dur+0.5, 1.8) * sampleRate)
		out := make([]float64, n)
		r := rand.New(rand.NewSource(int64(f * 10)))
		buf := make([]float64, period)
		for i := range buf {
			buf[i] = r.Float64()*2 - 1
		}
		for i := range buf { // a gentler start: one pass of smoothing
			buf[i] = 0.5 * (buf[i] + buf[(i+1)%period])
		}
		for i := range out {
			j := i % period
			out[i] = buf[j] * 0.9 * relEnv(float64(i)/sampleRate, dur, 0.4) * vol
			buf[j] = 0.9965 * 0.5 * (buf[j] + buf[(j+1)%period])
		}
		return out
	}}
	instruments["strings"] = instrument{sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.3) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			vib := 1 + 0.003*math.Sin(2*math.Pi*5.1*t)*smooth01(t/0.5)
			var v float64
			for _, d := range []float64{0.996, 1, 1.004} {
				v += sawSum(2*math.Pi*f*d*t*vib, 6)
			}
			out[i] = v * 0.33 * envAR(t, dur+0.05, 0.14, 0.25) * vol
		}
		return out
	}}
	instruments["fiddle"] = instrument{sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.1) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			vib := 1 + 0.006*math.Sin(2*math.Pi*5.6*t)*smooth01((t-0.12)/0.2)
			v := sawSum(2*math.Pi*f*t*vib, 8)
			out[i] = v * 0.5 * envAR(t, dur, 0.04, 0.08) * vol
		}
		return out
	}}
	instruments["pad"] = instrument{sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.4) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			var v float64
			for _, d := range []float64{0.997, 1.003} {
				ph := 2 * math.Pi * f * d * t
				v += math.Sin(ph) + 0.3*math.Sin(2*ph) + 0.12*math.Sin(3*ph)
			}
			out[i] = v * 0.3 * envAR(t, dur+0.1, 0.3, 0.4) * vol
		}
		return out
	}}
	// bright detuned saw lead
	instruments["saw"] = instrument{sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.08) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			v := sawSum(2*math.Pi*f*0.997*t, 9) + sawSum(2*math.Pi*f*1.003*t, 9)
			out[i] = v * 0.3 * envAR(t, dur, 0.008, 0.07) * (0.75 + 0.25*math.Exp(-5*t)) * vol
		}
		return out
	}}
	instruments["synbass"] = instrument{sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.05) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			ph := 2 * math.Pi * f * t
			var v float64
			for k := 1; k <= 5; k++ {
				v += math.Sin(float64(k)*ph) / float64(k) * (0.55 + 0.45*math.Exp(-float64(k)*5*t))
			}
			out[i] = v * 0.62 * envAR(t, dur, 0.004, 0.04) * vol
		}
		return out
	}}
}

// ---- the drum kit -----------------------------------------------------------------------------

var kitCache = map[byte][]float64{}

// kitSample: k kick, K soft kick, n snare, g ghost snare, c clap, o open hat, x crash, t tom, T timpani,
// j ride; w h s r are the light percussion of songs.go.
func kitSample(k byte) []float64 {
	if p, ok := kitCache[k]; ok {
		return p
	}
	var out []float64
	switch k {
	case 'k':
		out = kickSnip(0.9)
	case 'K':
		out = kickSnip(0.55)
	case 'n':
		out = snareSnip(0.5, 0.45)
	case 'g':
		out = snareSnip(0.2, 0.25)
	case 'c':
		out = snareSnip(0.1, 0.8)
	case 'o':
		out = hatSnip(true)
	case 'x':
		out = crashSnip()
	case 't', 'T':
		f0, f1, dec, secs := 150.0, 95.0, 9.0, 0.32
		if k == 'T' {
			f0, f1, dec, secs = 105.0, 70.0, 5.0, 0.6
		}
		out = make([]float64, int(secs*sampleRate))
		ph := 0.0
		for i := range out {
			t := float64(i) / sampleRate
			ph += 2 * math.Pi * (f1 + (f0-f1)*math.Exp(-t*30)) / sampleRate
			out[i] = math.Sin(ph) * math.Exp(-dec*t) * 0.9
		}
	case 'j':
		out = make([]float64, int(0.4*sampleRate))
		r := rand.New(rand.NewSource(21))
		prev := 0.0
		for i := range out {
			t := float64(i) / sampleRate
			x := r.Float64()*2 - 1
			hp := x - prev*0.7
			prev = x
			out[i] = (hp*0.35 + 0.4*math.Sin(2*math.Pi*5200*t) + 0.3*math.Sin(2*math.Pi*7100*t)) * math.Exp(-9*t) * 0.5
		}
	default:
		out = percSample(k)
	}
	kitCache[k] = out
	return out
}

// the kit is built once, at start, so the background threads that render songs only read it
func init() {
	for _, k := range []byte("kKngcoxtTjwhsr") {
		kitSample(k)
	}
}

// ---- song definitions -------------------------------------------------------------------------

type gTrack struct {
	inst  string
	notes string
	vol   float64
	mel   bool // the tune: the other versions of the song read it backwards, bar by bar
}

type gPerc struct {
	pattern string // sixteenth notes; its length is a multiple of 16 and it cycles over the bars
	vol     float64
}

type genreSong struct {
	bpm    float64
	beats  float64 // length of the loop
	scale  scaleDef
	tracks []gTrack
	perc   []gPerc
	swing  int // 0 straight, 1 triplet swing on the off beats, 2 lazy sixteenths
	// noRetro: variant 2 does not read the tune backwards (a lyrical melody must not run in reverse)
	noRetro bool
	// rotVariant: the variant (1 or 2) that starts half a loop later, so the sections come in another order (0 = none)
	rotVariant int
	echo       float64
	fb         float64
	wet        float64
}

func gRep(bar string, n int) string {
	p := make([]string, n)
	for i := range p {
		p[i] = bar
	}
	return strings.Join(p, " | ")
}

func gN(name string) int {
	m, ok := parseNote(name)
	if !ok {
		panic("genres: bad note " + name)
	}
	return m
}

// gChord writes a chord from MIDI numbers.
func gChord(ms ...int) string {
	p := make([]string, len(ms))
	for i, m := range ms {
		p[i] = midiName(m)
	}
	return "[" + strings.Join(p, ",") + "]"
}

func gDur(d float64) string { return ":" + strconv.FormatFloat(d, 'f', -1, 64) }

// retroBars reads every bar backwards: same notes, another contour.
func retroBars(text string) string {
	bars := strings.Split(text, " | ")
	for i, b := range bars {
		toks := strings.Fields(b)
		for l, r := 0, len(toks)-1; l < r; l, r = l+1, r-1 {
			toks[l], toks[r] = toks[r], toks[l]
		}
		bars[i] = strings.Join(toks, " ")
	}
	return strings.Join(bars, " | ")
}

// perBar builds a line of bars: f(i) is the text of bar i.
func perBar(n int, f func(i int) string) string {
	p := make([]string, n)
	for i := range p {
		p[i] = f(i)
	}
	return strings.Join(p, " | ")
}

// arpBar plays a chord as sixteenth notes: tones are MIDI numbers, idx picks one for each of the 16 steps.
func arpBar(tones []int, idx []int, d float64) string {
	p := make([]string, len(idx))
	for i, k := range idx {
		p[i] = midiName(tones[k]) + gDur(d)
	}
	return strings.Join(p, " ")
}

func dots(n int) string { return strings.Repeat(".", n) }

var (
	scDMajor   = scaleDef{"D major", 2, stepsMajor}
	scEMajor   = scaleDef{"E major", 4, stepsMajor}
	scEPhryg   = scaleDef{"E phrygian", 4, []int{0, 1, 3, 5, 7, 8, 10}}
	scASpanish = scaleDef{"A spanish", 9, []int{0, 1, 2, 3, 4, 5, 7, 8, 10}}
)

// genreSongs: the new genres (house, electro, techno, disco, reggaeton and punk live in music2.go).
var genreSongs = map[string]genreSong{}

func init() {
	// ---- pop: bright, a hook that sticks, four chords I V vi IV
	{
		chords := [][]int{{gN("C4"), gN("E4"), gN("G4")}, {gN("B3"), gN("D4"), gN("G4")}, {gN("A3"), gN("C4"), gN("E4")}, {gN("A3"), gN("C4"), gN("F4")}}
		order := []int{0, 1, 2, 3, 0, 1, 3, 1}
		roots := []int{gN("C2"), gN("G2"), gN("A2"), gN("F2")}
		genreSongs["pop"] = genreSong{bpm: 116, beats: 32, scale: scCMajor, echo: 0.5, fb: 0.25, wet: 0.12,
			tracks: []gTrack{
				{"saw", noteLine(
					"E5:0.5 E5:0.5 G5:1 E5:0.5 D5:0.5 C5:1", "D5:0.5 D5:0.5 G5:1 D5:0.5 B4:0.5 G4:1",
					"C5:0.5 C5:0.5 E5:1 C5:0.5 B4:0.5 A4:1", "A4:0.5 C5:0.5 F5:1 E5:1 C5:1",
					"E5:0.5 E5:0.5 G5:1 A5:0.5 G5:0.5 E5:1", "D5:0.5 D5:0.5 G5:1 B5:0.5 A5:0.5 G5:1",
					"A5:1 G5:0.5 F5:0.5 E5:1 C5:1", "D5:1 B4:1 G4:1 R:1"), 0.45, true},
				{"piano", perBar(8, func(i int) string {
					c := gChord(chords[order[i]]...)
					return c + ":1 R:0.5 " + c + ":0.5 " + c + ":1 R:1"
				}), 0.3, false},
				{"ebass", perBar(8, func(i int) string {
					r := roots[[]int{0, 1, 2, 3, 0, 1, 3, 1}[i]]
					return midiName(r) + ":1 " + midiName(r) + ":0.5 " + midiName(r+12) + ":0.5 " + midiName(r) + ":1 R:0.5 " + midiName(r) + ":0.5"
				}), 0.5, false},
			},
			perc: []gPerc{{"k...k...k...k...", 0.6}, {"....c.......c...", 0.45}, {"h.h.h.h.h.h.h.h.", 0.16}, {"..............o.", 0.12}}}
	}
	// ---- metal: chugging power chords, double kick, a screaming lead (E phrygian)
	{
		riff := func(a, b, c int) string {
			x := func(m int, d float64) string { return gChord(m, m+7) + gDur(d) }
			return strings.Join([]string{x(a, 0.25), x(a, 0.25), x(a, 0.5), x(a, 0.25), x(a, 0.25), x(b, 0.5), x(a, 0.25), x(a, 0.25), x(c, 0.5), x(a, 1)}, " ")
		}
		e2, g2, f2, a2, d3, c3 := gN("E2"), gN("G2"), gN("F2"), gN("A2"), gN("D3"), gN("C3")
		genreSongs["metal"] = genreSong{bpm: 160, beats: 32, scale: scEPhryg, echo: 0.375, fb: 0.2, wet: 0.08,
			tracks: []gTrack{
				{"dist", noteLine(riff(e2, g2, f2), riff(e2, g2, f2), riff(e2, a2, g2), riff(e2, d3, c3), riff(e2, g2, f2), riff(e2, g2, f2), riff(e2, a2, g2), riff(e2, f2, e2)), 0.5, false},
				{"saw", noteLine(
					"E5:1 G5:0.5 F5:0.5 E5:1 B4:1", "D5:1 E5:1 G5:1 F5:1", "A5:1 G5:0.5 F5:0.5 E5:2", "C5:1 D5:1 E5:2",
					"E5:1 G5:0.5 F5:0.5 E5:1 B4:1", "D5:1 E5:1 G5:1 F5:1", "B5:1 A5:1 G5:1 E5:1", "F5:1 G5:1 E5:2"), 0.28, true},
				{"ebass", gRep(gRep("E2:0.5", 8), 8), 0.5, false},
			},
			perc: []gPerc{{"k.kk..k.k.kk..k.", 0.7}, {"....n.......n...", 0.55}, {"h.h.h.h.h.h.h.h.", 0.16}, {"x" + dots(63), 0.35}}}
	}
	// ---- rap: boom bap, 808 and a dusty piano loop (C minor)
	{
		cm, ab, bb := gChord(gN("C4"), gN("Eb4"), gN("G4")), gChord(gN("Ab3"), gN("C4"), gN("Eb4")), gChord(gN("Bb3"), gN("F4"))
		prog := []string{cm, cm, ab, bb, cm, cm, ab, bb}
		bassRoot := []int{gN("C2"), gN("C2"), gN("Ab1"), gN("Bb1"), gN("C2"), gN("C2"), gN("Ab1"), gN("Bb1")}
		genreSongs["rap"] = genreSong{bpm: 92, beats: 32, scale: scCMinor, echo: 0.75, fb: 0.3, wet: 0.16,
			tracks: []gTrack{
				{"box", noteLine(
					"G4:1 R:0.5 Eb4:0.5 C4:1 R:1", "G4:0.5 Bb4:0.5 G4:1 Eb4:1 R:1", "Eb5:1 C5:1 Ab4:1 R:1", "F5:1 Eb5:1 Bb4:2",
					"G4:1 R:0.5 Eb4:0.5 C4:1 R:1", "Bb4:0.5 C5:0.5 Bb4:1 G4:1 R:1", "Eb5:1 G5:1 Ab4:1 R:1", "F5:1 Eb5:1 C5:2"), 0.34, true},
				{"piano", perBar(8, func(i int) string { return prog[i] + ":1.5 R:0.5 " + prog[i] + ":1 R:1" }), 0.3, false},
				{"sub", perBar(8, func(i int) string {
					r := bassRoot[i]
					return midiName(r) + ":1.5 R:0.5 " + midiName(r+7) + ":0.5 " + midiName(r) + ":1.5"
				}), 0.6, false},
			},
			perc: []gPerc{{"k.....k...k.....", 0.75}, {"....n.......n...", 0.55}, {"..............g.", 0.3}, {"h.hhh.h.h.hhh.h.", 0.15}}}
	}
	// ---- hip hop: lazy lo-fi, jazzy chords, a music box (C major)
	{
		ch := [][]int{{gN("F3"), gN("A3"), gN("C4"), gN("E4")}, {gN("E3"), gN("G3"), gN("B3"), gN("D4")}, {gN("D3"), gN("F3"), gN("A3"), gN("C4")}, {gN("C3"), gN("E3"), gN("G3"), gN("B3")}}
		roots := []int{gN("F2"), gN("E2"), gN("D2"), gN("C2")}
		genreSongs["hiphop"] = genreSong{bpm: 80, beats: 32, scale: scCMajor, swing: 2, echo: 0.75, fb: 0.4, wet: 0.28,
			tracks: []gTrack{
				{"box", noteLine(
					"A4:1 G4:1 E4:2", "G4:1 E4:1 D4:2", "A4:1.5 G4:0.5 E4:2", "C5:2 R:2",
					"E5:1 D5:1 C5:2", "D5:1 C5:1 A4:2", "G4:1.5 A4:0.5 C5:2", "D5:2 R:2"), 0.34, true},
				{"piano", perBar(8, func(i int) string { c := gChord(ch[i%4]...); return c + ":2.5 R:0.5 " + c + ":1" }), 0.26, false},
				{"sub", perBar(8, func(i int) string { return midiName(roots[i%4]) + ":3 R:1" }), 0.5, false},
			},
			perc: []gPerc{{"k.......k.k.....", 0.6}, {"....n.......n...", 0.3}, {"h.h.h.h.h.h.h.h.", 0.14}}}
	}
	// ---- classical: a waltz, strings over an arpeggio and a plucked bass (C major, 3/4)
	{
		ch := [][]int{{gN("C4"), gN("E4"), gN("G4")}, {gN("F3"), gN("A3"), gN("C4")}, {gN("G3"), gN("B3"), gN("D4")}, {gN("C4"), gN("E4"), gN("G4")},
			{gN("A3"), gN("C4"), gN("E4")}, {gN("D4"), gN("F4"), gN("A4")}, {gN("G3"), gN("B3"), gN("D4")}, {gN("C4"), gN("E4"), gN("G4")}}
		roots := []int{gN("C3"), gN("F2"), gN("G2"), gN("C3"), gN("A2"), gN("D3"), gN("G2"), gN("C3")}
		genreSongs["clasica"] = genreSong{bpm: 132, beats: 24, scale: scCMajor, echo: 0.5, fb: 0.3, wet: 0.2,
			tracks: []gTrack{
				{"strings", noteLine(
					"E5:1.5 D5:0.5 C5:1", "A4:1.5 G4:0.5 F4:1", "B4:1.5 A4:0.5 G4:1", "C5:2 E5:1",
					"A5:1.5 G5:0.5 E5:1", "F5:1.5 E5:0.5 D5:1", "D5:1 B4:1 G4:1", "C5:3"), 0.42, true},
				{"piano", perBar(8, func(i int) string {
					c := ch[i]
					return strings.Join([]string{midiName(c[0]) + ":0.5", midiName(c[1]) + ":0.5", midiName(c[2]) + ":0.5", midiName(c[1]) + ":0.5", midiName(c[2]) + ":0.5", midiName(c[1]) + ":0.5"}, " ")
				}), 0.22, false},
				{"pizz", perBar(8, func(i int) string { return midiName(roots[i]) + ":1 R:2" }), 0.5, false},
			}}
	}
	// ---- jazz: swing, a walking bass, comping piano and a trumpet (C major)
	{
		dm7, g7, cm7, am7 := gChord(gN("D4"), gN("F4"), gN("A4"), gN("C5")), gChord(gN("G3"), gN("B3"), gN("D4"), gN("F4")), gChord(gN("C4"), gN("E4"), gN("G4"), gN("B4")), gChord(gN("A3"), gN("C4"), gN("E4"), gN("G4"))
		prog := []string{dm7, g7, cm7, am7, dm7, g7, cm7, g7}
		walk := []string{"D2:1 F2:1 A2:1 C3:1", "G2:1 B2:1 D3:1 F3:1", "C2:1 E2:1 G2:1 B2:1", "A2:1 C3:1 E3:1 G3:1", "D2:1 F2:1 A2:1 C3:1", "G2:1 B2:1 D3:1 F3:1", "C2:1 E2:1 G2:1 B2:1", "G2:1 F2:1 D2:1 B1:1"}
		genreSongs["jazz"] = genreSong{bpm: 132, beats: 32, scale: scCMajor, swing: 1, echo: 0.5, fb: 0.2, wet: 0.1,
			tracks: []gTrack{
				{"brass", noteLine(
					"F5:1 A5:0.6667 G5:0.3333 F5:1 D5:1", "B4:1 D5:0.6667 F5:0.3333 E5:2", "G5:2 E5:1 C5:1", "A4:1 C5:0.6667 E5:0.3333 G5:1 R:1",
					"A5:1 F5:0.6667 D5:0.3333 F5:1 A5:1", "G5:1 F5:0.6667 D5:0.3333 B4:2", "E5:1 G5:0.6667 B5:0.3333 C6:2", "D5:1 F5:1 G5:1 R:1"), 0.34, true},
				{"piano", perBar(8, func(i int) string { return prog[i] + ":1 R:1 " + prog[i] + ":0.6667 R:0.3333 R:1" }), 0.26, false},
				{"pizz", noteLine(walk...), 0.55, false},
			},
			perc: []gPerc{{"j...j.j.j...j.j.", 0.4}, {"....h.......h...", 0.2}, {"K.......K.......", 0.2}, {".......g........", 0.2}}}
	}
	// ---- country: boom-chick, a guitar and a fiddle (C major)
	{
		cC, cF, cG := gChord(gN("E4"), gN("G4"), gN("C5")), gChord(gN("F4"), gN("A4"), gN("C5")), gChord(gN("D4"), gN("G4"), gN("B4"))
		order := []string{cC, cF, cC, cG, cC, cF, cG, cC}
		root := []int{gN("C3"), gN("F2"), gN("C3"), gN("G2"), gN("C3"), gN("F2"), gN("G2"), gN("C3")}
		genreSongs["country"] = genreSong{bpm: 112, beats: 32, scale: scCMajor, echo: 0.5, fb: 0.2, wet: 0.1,
			tracks: []gTrack{
				{"fiddle", noteLine(
					"E5:0.5 G5:0.5 E5:0.5 C5:0.5 D5:1 E5:1", "F5:0.5 A5:0.5 F5:0.5 C5:0.5 A4:1 C5:1", "E5:0.5 G5:0.5 C6:0.5 G5:0.5 E5:1 C5:1", "D5:1 B4:1 G4:2",
					"G5:0.5 E5:0.5 G5:0.5 E5:0.5 C5:2", "A5:0.5 F5:0.5 A5:0.5 C6:0.5 A5:1 F5:1", "B4:0.5 D5:0.5 G5:0.5 B5:0.5 A5:1 G5:1", "E5:1 D5:1 C5:2"), 0.4, true},
				{"guitar", perBar(8, func(i int) string { return "R:1 " + order[i] + ":1 R:1 " + order[i] + ":1" }), 0.3, false},
				{"pizz", perBar(8, func(i int) string { return midiName(root[i]) + ":1 R:1 " + midiName(root[i]+7) + ":1 R:1" }), 0.55, false},
			},
			perc: []gPerc{{"K.......K.......", 0.4}, {"....r.......r...", 0.3}, {"s.s.s.s.s.s.s.s.", 0.12}}}
	}
	// ---- flamenco / pasodoble: palmas, a Spanish guitar and a brass line (A, phrygian dominant)
	{
		am, g, f, e := gChord(gN("A3"), gN("C4"), gN("E4")), gChord(gN("G3"), gN("B3"), gN("D4")), gChord(gN("F3"), gN("A3"), gN("C4")), gChord(gN("E3"), gN("B3"), gN("E4"))
		order := []string{am, g, f, e, am, g, f, e}
		root := []int{gN("A2"), gN("G2"), gN("F2"), gN("E2"), gN("A2"), gN("G2"), gN("F2"), gN("E2")}
		genreSongs["flamenco"] = genreSong{bpm: 120, beats: 32, scale: scASpanish, echo: 0.5, fb: 0.25, wet: 0.14,
			tracks: []gTrack{
				{"brass", noteLine(
					"A5:0.5 Bb5:0.5 A5:0.5 G5:0.5 F5:1 E5:1", "G5:0.5 A5:0.5 G5:0.5 F5:0.5 E5:1 D5:1", "F5:0.5 G5:0.5 F5:0.5 E5:0.5 D5:1 C5:1", "E5:0.5 F5:0.5 E5:0.5 D5:0.5 C#5:1 E5:1",
					"A5:1 C6:1 B5:1 A5:1", "G5:1 B5:1 A5:1 G5:1", "F5:1 A5:1 G5:1 F5:1", "E5:1 C#5:1 A4:2"), 0.34, true},
				{"guitar", perBar(8, func(i int) string {
					return order[i] + ":0.5 R:0.5 " + order[i] + ":0.5 R:0.5 " + order[i] + ":0.5 " + order[i] + ":0.5 " + order[i] + ":1"
				}), 0.3, false},
				{"pizz", perBar(8, func(i int) string {
					r := midiName(root[i])
					return r + ":1 R:0.5 " + r + ":0.5 " + r + ":1 R:1"
				}), 0.5, false},
			},
			perc: []gPerc{{"K.......K.......", 0.5}, {"..c...c...c...c.", 0.4}, {"h.h.h.h.h.h.h.h.", 0.1}}}
	}
	// ---- rock: power chords, a driving bass and a pentatonic lead (E major)
	{
		pe, pa, pb := gChord(gN("E2"), gN("B2")), gChord(gN("A2"), gN("E3")), gChord(gN("B2"), gN("F#3"))
		order := []string{pe, pa, pb, pa, pe, pa, pb, pe}
		root := []int{gN("E2"), gN("A2"), gN("B2"), gN("A2"), gN("E2"), gN("A2"), gN("B2"), gN("E2")}
		genreSongs["rock"] = genreSong{bpm: 124, beats: 32, scale: scEMajor, echo: 0.375, fb: 0.2, wet: 0.08,
			tracks: []gTrack{
				{"dist", perBar(8, func(i int) string {
					x := order[i]
					return x + ":1 " + x + ":0.5 " + x + ":0.5 " + x + ":1 " + x + ":0.5 " + x + ":0.5"
				}), 0.42, false},
				{"saw", noteLine(
					"G#5:0.5 B5:0.5 G#5:1 E5:1 F#5:1", "A5:1 E5:0.5 F#5:0.5 A5:1 C#6:1", "B5:1 F#5:0.5 G#5:0.5 B5:1 D#6:1", "C#6:1 B5:1 A5:2",
					"G#5:1 B5:1 E6:2", "E6:1 C#6:1 A5:2", "D#6:1 B5:1 F#5:2", "E5:1 G#5:1 E5:2"), 0.3, true},
				{"ebass", perBar(8, func(i int) string { return gRep(midiName(root[i])+":0.5", 8) }), 0.5, false},
			},
			perc: []gPerc{{"k.....k.k.......", 0.7}, {"....n.......n...", 0.55}, {"h.h.h.h.h.h.h.h.", 0.16}, {"x" + dots(63), 0.3}}}
	}
	// ---- synthwave: arpeggios, pads and a gated snare (A minor)
	{
		ch := [][]int{{gN("A3"), gN("C4"), gN("E4"), gN("A4")}, {gN("F3"), gN("A3"), gN("C4"), gN("F4")}, {gN("C4"), gN("E4"), gN("G4"), gN("C5")}, {gN("G3"), gN("B3"), gN("D4"), gN("G4")}}
		roots := []int{gN("A2"), gN("F2"), gN("C3"), gN("G2")}
		idx := []int{0, 1, 2, 3, 2, 1, 2, 3, 0, 1, 2, 3, 2, 1, 2, 1}
		genreSongs["synthwave"] = genreSong{bpm: 100, beats: 32, scale: scAMinor, echo: 0.75, fb: 0.35, wet: 0.2,
			tracks: []gTrack{
				{"square", noteLine(
					"E5:2 C5:1 D5:1", "C5:2 A4:1 C5:1", "E5:1.5 G5:0.5 E5:2", "D5:2 B4:1 G4:1",
					"A5:2 G5:1 E5:1", "A5:1 C6:1 A5:2", "G5:1 E5:1 C5:2", "B4:1 D5:1 G4:2"), 0.3, true},
				{"saw", perBar(8, func(i int) string { return arpBar(ch[i%4], idx, 0.25) }), 0.16, false},
				{"pad", perBar(8, func(i int) string { return gChord(ch[i%4][:3]...) + ":4" }), 0.3, false},
				{"synbass", perBar(8, func(i int) string {
					r := roots[i%4]
					a, b := midiName(r), midiName(r+12)
					return strings.Join([]string{a + ":0.5", a + ":0.5", b + ":0.5", a + ":0.5", a + ":0.5", a + ":0.5", b + ":0.5", a + ":0.5"}, " ")
				}), 0.5, false},
			},
			perc: []gPerc{{"k...k...k...k...", 0.65}, {"....n.......n...", 0.45}, {"....c.......c...", 0.25}, {"..h...h...h...h.", 0.14}}}
	}
	// ---- orchestral: strings, an ostinato, brass and timpani (G major)
	{
		ch := [][]int{{gN("G3"), gN("B3"), gN("D4")}, {gN("E3"), gN("G3"), gN("B3")}, {gN("C4"), gN("E4"), gN("G4")}, {gN("D4"), gN("F#4"), gN("A4")}}
		genreSongs["orquestal"] = genreSong{bpm: 108, beats: 32, scale: scGMajor, echo: 0.75, fb: 0.3, wet: 0.24,
			tracks: []gTrack{
				{"brass", noteLine(
					"D5:1 G5:1 B5:1 A5:1", "G5:1 E5:1 B4:2", "E5:1 G5:1 C6:2", "D5:1 F#5:1 A5:2",
					"B5:1.5 A5:0.5 G5:2", "E5:1 G5:1 B5:2", "C6:1 B5:1 A5:1 G5:1", "F#5:1 A5:1 D6:2"), 0.3, true},
				{"strings", perBar(8, func(i int) string { return gChord(ch[i%4]...) + ":4" }), 0.36, false},
				{"pizz", perBar(8, func(i int) string {
					c := ch[i%4]
					return arpBar(c, []int{0, 1, 2, 1, 0, 1, 2, 1}, 0.5)
				}), 0.4, false},
			},
			perc: []gPerc{{"T.......T.......", 0.5}, {"x" + dots(63), 0.2}, {"..............w.", 0.1}}}
	}
	// ---- epic: tribal toms, a low brass motif and a dark pad (C minor)
	{
		cm, ab, bb := []int{gN("C3"), gN("Eb3"), gN("G3")}, []int{gN("Ab2"), gN("Eb3"), gN("C4")}, []int{gN("Bb2"), gN("F3")}
		order := [][]int{cm, cm, ab, bb, cm, cm, ab, bb}
		genreSongs["epico"] = genreSong{bpm: 100, beats: 32, scale: scCMinor, echo: 0.75, fb: 0.35, wet: 0.25,
			tracks: []gTrack{
				{"brass", noteLine(
					"C4:1 R:0.5 C4:0.5 Eb4:1 G4:1", "F4:1 Eb4:1 C4:2", "G4:1 R:0.5 G4:0.5 Bb4:1 C5:1", "Bb4:1 G4:1 Eb4:2",
					"C5:1 R:0.5 C5:0.5 Eb5:1 G5:1", "F5:1 Eb5:1 C5:2", "G5:1 Bb5:1 C6:1 Bb5:1", "G5:2 C5:2"), 0.34, true},
				{"pad", perBar(8, func(i int) string { return gChord(order[i]...) + ":4" }), 0.34, false},
				{"pizz", perBar(8, func(i int) string {
					r := order[i][0]
					a, b := midiName(r), midiName(r+12)
					return strings.Join([]string{a + ":0.5", a + ":0.5", b + ":0.5", a + ":0.5", a + ":0.5", a + ":0.5", b + ":0.5", a + ":0.5"}, " ")
				}), 0.4, false},
			},
			perc: []gPerc{{"t..t..t...t.t...", 0.55}, {"T.......T.......", 0.5}, {"x" + dots(63), 0.2}}}
	}
	// ---- isla: a wonder-and-danger score for the island, an ORIGINAL piece in homage to the adventure scores of the big
	// orchestral films (it shares their colour: a noble, rising horn theme over strings and harp, timpani and cymbals,
	// and a dark ostinato with stomping timpani for the monster). The melody, the chords and the rhythm are written
	// here from scratch: none of it is taken from John Williams' Jurassic Park themes or from any other piece.
	// 16 bars at 72 BPM: wonder (8) - danger (4) - the theme returns, grander (4). D major; the danger is its relative B minor,
	// so every note belongs to the same key.
	{
		bm, g, d, a := []int{gN("B3"), gN("D4"), gN("F#4")}, []int{gN("G3"), gN("B3"), gN("D4")}, []int{gN("A3"), gN("D4"), gN("F#4")}, []int{gN("A3"), gN("C#4"), gN("E4")}
		chords := [][]int{bm, g, d, a, bm, g, a, d, // wonder
			{gN("B2"), gN("F#3"), gN("B3")}, {gN("B2"), gN("F#3"), gN("D4")}, {gN("G2"), gN("D3"), gN("B3")}, {gN("A2"), gN("E3"), gN("A3")}, // danger
			bm, g, a, d} // the theme returns
		harpTones := map[int][]int{
			0: {gN("B3"), gN("D4"), gN("F#4"), gN("B4"), gN("D5"), gN("F#5")},
			1: {gN("G3"), gN("B3"), gN("D4"), gN("G4"), gN("B4"), gN("D5")},
			2: {gN("D4"), gN("F#4"), gN("A4"), gN("D5"), gN("F#5"), gN("A5")},
			3: {gN("A3"), gN("C#4"), gN("E4"), gN("A4"), gN("C#5"), gN("E5")},
		}
		kind := []int{0, 1, 2, 3, 0, 1, 3, 2, -1, -1, -1, -1, 0, 1, 3, 2} // which harp arpeggio each bar plays (-1: silent)
		rise := []int{0, 1, 2, 3, 4, 3, 2, 1}
		basses := []string{"B2:2 F#3:2", "G2:2 D3:2", "D3:2 A3:2", "A2:2 E3:2", "B2:2 F#3:2", "G2:2 D3:2", "A2:2 E3:2", "D3:2 A3:2",
			"B1:0.5 R:0.5 B1:0.5 R:0.5 F#2:0.5 R:0.5 B1:0.5 C#2:0.5", "B1:0.5 R:0.5 B1:0.5 R:0.5 D2:0.5 R:0.5 C#2:0.5 B1:0.5",
			"G1:0.5 R:0.5 G1:0.5 R:0.5 D2:0.5 R:0.5 G1:0.5 A1:0.5", "A1:0.5 A1:0.5 R:0.5 A1:0.5 E2:0.5 R:0.5 A1:1",
			"B2:2 F#3:2", "G2:2 D3:2", "A2:2 E3:2", "D3:4"}
		timp := func(bars ...string) string { return strings.Join(bars, "") }
		silent := dots(16)
		genreSongs["isla"] = genreSong{bpm: 72, beats: 64, scale: scDMajor, echo: 1.0, fb: 0.35, wet: 0.3, noRetro: true, rotVariant: 1,
			tracks: []gTrack{
				// the horn: the theme (8 bars), a sinister whisper (4 bars), the theme again an octave up in its second half (4 bars)
				{"horn", noteLine(
					"F#4:1 B4:2 A4:1", "G4:1 E5:2 D5:1", "F#5:1.5 E5:0.5 D5:1 B4:1", "A4:3 R:1",
					"B4:1 F#5:2 E5:1", "D5:1 A5:2 G5:1", "F#5:1 E5:0.5 D5:0.5 E5:1 F#5:1", "D5:4",
					"R:4", "B3:1.5 C#4:0.5 D4:2", "G4:0.5 F#4:0.5 D4:1 C#4:2", "R:2 A3:1 C#4:1",
					"B4:1 F#5:2 E5:1", "D5:1 A5:2 B5:1", "A5:1.5 G5:0.5 F#5:1 E5:1", "D5:3 R:1"), 0.34, true},
				{"strings", perBar(16, func(i int) string { return gChord(chords[i]...) + ":4" }), 0.3, false},
				{"harp", perBar(16, func(i int) string {
					if kind[i] < 0 {
						return "R:4"
					}
					return arpBar(harpTones[kind[i]], rise, 0.5)
				}), 0.42, false},
				{"celesta", noteLine(
					"R:4", "R:4", "R:4", "R:2 A5:0.5 D6:0.5 F#6:1", "R:4", "R:4", "R:4", "R:2 F#5:0.5 A5:0.5 D6:1",
					"R:4", "R:4", "R:4", "R:4", "R:4", "R:4", "R:4", "R:1 A5:0.5 D6:0.5 F#6:1 A6:1"), 0.28, false},
				{"ebass", noteLine(basses...), 0.5, false},
				// brass stingers in the danger: a held fifth and a rising chord that pushes back into the theme
				{"brass", noteLine(
					"R:4", "R:4", "R:4", "R:4", "R:4", "R:4", "R:4", "R:4",
					"R:4", "R:2 [B3,F#4]:1.5 R:0.5", "R:3 [G3,D4]:1", "R:2 [A3,C#4,E4]:2",
					"R:4", "R:4", "R:4", "R:4"), 0.26, false},
			},
			perc: []gPerc{
				// timpani: soft in the wonder, a roll before every entry, stomps (footsteps) in the danger
				{timp("T...............", silent, "........T.......", "T.......T.T.T.T.", "T...............", silent, "........T.......", "T.......T.T.TTTT",
					"T.......T.......", "T.......T.......", "T...T...T..T.T..", "T.T.T.T.TtTtTTTT",
					"T.......T.......", "T...T...T...T...", "T...T...T.T.T.T.", "T.......T......."), 0.5},
				{timp("x"+dots(15), silent, silent, silent, "x"+dots(15), silent, silent, silent, "x"+dots(15), silent, silent, "............x...",
					"x"+dots(15), silent, silent, silent), 0.22},
			}}
	}
	// ---- chiptune / funfair: bouncy squares, an oom-pah bass (C major)
	{
		ch := [][]int{{gN("C4"), gN("E4"), gN("G4")}, {gN("F4"), gN("A4"), gN("C5")}, {gN("G3"), gN("B3"), gN("D4")}, {gN("C4"), gN("E4"), gN("G4")},
			{gN("C4"), gN("E4"), gN("G4")}, {gN("F4"), gN("A4"), gN("C5")}, {gN("G3"), gN("B3"), gN("D4")}, {gN("C4"), gN("E4"), gN("G4")}}
		roots := []int{gN("C3"), gN("F2"), gN("G2"), gN("C3"), gN("C3"), gN("F2"), gN("G2"), gN("C3")}
		genreSongs["chiptune"] = genreSong{bpm: 144, beats: 32, scale: scCMajor, echo: 0.375, fb: 0.2, wet: 0.1,
			tracks: []gTrack{
				{"square", noteLine(
					"E5:0.5 E5:0.5 G5:0.5 E5:0.5 C5:0.5 E5:0.5 G5:1", "F5:0.5 F5:0.5 A5:0.5 F5:0.5 C5:0.5 F5:0.5 A5:1",
					"D5:0.5 D5:0.5 G5:0.5 D5:0.5 B4:0.5 D5:0.5 G5:1", "E5:1 G5:1 C6:2",
					"C6:0.5 B5:0.5 A5:0.5 G5:0.5 F5:0.5 E5:0.5 D5:1", "A5:0.5 G5:0.5 F5:0.5 E5:0.5 F5:0.5 G5:0.5 A5:1",
					"B5:0.5 A5:0.5 G5:0.5 F5:0.5 E5:0.5 D5:0.5 G5:1", "E5:0.5 G5:0.5 C6:1 C6:2"), 0.3, true},
				{"square", perBar(8, func(i int) string { return arpBar(ch[i], []int{0, 1, 2, 1, 0, 1, 2, 1, 0, 1, 2, 1, 0, 1, 2, 1}, 0.25) }), 0.1, false},
				{"pizz", perBar(8, func(i int) string {
					a, b := midiName(roots[i]), midiName(roots[i]+12)
					return a + ":0.5 R:0.5 " + b + ":0.5 R:0.5 " + a + ":0.5 R:0.5 " + b + ":0.5 R:0.5"
				}), 0.5, false},
			},
			perc: []gPerc{{"K.......K.......", 0.45}, {"....r.......r...", 0.4}, {"h.h.h.h.h.h.h.h.", 0.14}, {"s.s.s.s.s.s.s.s.", 0.1}}}
	}
}

// ---- rendering --------------------------------------------------------------------------------

// render plays the song once into a seamless loop. shift moves everything by semitones,
// bpmMul speeds it up or down and retro reads the tune backwards bar by bar.
func (gs genreSong) render(shift int, bpmMul float64, retro bool) []float64 {
	bpm := gs.bpm * bpmMul
	spb := 60 / bpm * sampleRate
	n := int(math.Round(gs.beats * spb))
	out := make([]float64, n)
	for _, tr := range gs.tracks {
		inst, ok := instruments[tr.inst]
		if !ok {
			panic("genres: no instrument " + tr.inst)
		}
		text := tr.notes
		if retro && tr.mel {
			text = retroBars(text)
		}
		if shift != 0 {
			text = transposeText(text, shift)
		}
		// the same note of the same length comes back many times (chords, arpeggios, riffs): it is played once
		// (every instrument is deterministic: the same note gives the same samples)
		type noteKey struct {
			m    int
			secs float64
		}
		played := map[noteKey][]float64{}
		for _, nt := range parseNotes(text) {
			at := int(math.Round(nt.at * spb))
			secs := nt.dur * 60 / bpm
			for _, m := range nt.midi {
				snip, ok := played[noteKey{m, secs}]
				if !ok {
					snip = inst.play(midiFreq(m), secs, tr.vol)
					played[noteKey{m, secs}] = snip
				}
				mixWrap(out, at, snip, 1)
			}
		}
	}
	bars := int(math.Ceil(gs.beats / 4))
	for _, pc := range gs.perc {
		for bar := 0; bar < bars; bar++ {
			for step := 0; step < 16; step++ {
				ch := pc.pattern[(bar*16+step)%len(pc.pattern)]
				if ch == '.' {
					continue
				}
				beat := float64(bar*4) + float64(step)/4
				switch gs.swing {
				case 1:
					if step%4 == 2 {
						beat += 1.0 / 6
					}
				case 2:
					if step%2 == 1 {
						beat += 0.07
					}
				}
				if beat >= gs.beats {
					continue
				}
				at := int(math.Round(beat * spb))
				mixWrap(out, at, kitSample(ch), pc.vol)
			}
		}
	}
	if gs.echo > 0 {
		out = echoLoop(out, gs.echo*60/bpm, gs.fb, gs.wet)
	}
	return out
}

// mixWrap adds snip*vol to out from at, going round to the start past the end (the songs are seamless loops).
func mixWrap(out []float64, at int, snip []float64, vol float64) {
	n := len(out)
	at %= n
	for len(snip) > 0 {
		k := min(len(snip), n-at)
		dst := out[at : at+k]
		for i, v := range snip[:k] {
			dst[i] += v * vol
		}
		snip, at = snip[k:], 0
	}
}

// finishGenre: no DC offset, the peaks gently squeezed, one loudness for every genre, and a peak that is always below 0 dBFS.
func finishGenre(x []float64) []byte { return finishGenreRMS(x, 0.17) }

// finishGenreRMS: finishGenre with another loudness (target RMS).
func finishGenreRMS(x []float64, target float64) []byte {
	var mean float64
	for _, v := range x {
		mean += v
	}
	mean /= float64(len(x))
	peak := 0.0
	for i := range x {
		x[i] -= mean
		peak = math.Max(peak, math.Abs(x[i]))
	}
	if peak == 0 {
		return toPCM(x)
	}
	for i := range x {
		x[i] = math.Tanh(1.4*x[i]/peak) / math.Tanh(1.4)
	}
	var sq float64
	peak = 0
	for _, v := range x {
		sq += v * v
		peak = math.Max(peak, math.Abs(v))
	}
	rms := math.Sqrt(sq / float64(len(x)))
	g := math.Min(target/rms, 0.9/peak)
	for i := range x {
		x[i] *= g
	}
	return toPCM(x)
}

// ---- the genre of every party -----------------------------------------------------------------

type genreInfo struct {
	id     string // the id of the style: a genreSongs key or a style of music2.go
	label  string // shown at the start of the party
	legacy bool   // played by the old step sequencer in music2.go
	bpm    float64
	lo, hi float64 // the tempo a song of this genre may have
}

var genreTable = map[string]genreInfo{
	"house":     {"house", "Música: House", true, 122, 116, 128},
	"electro":   {"electro", "Música: Electrónica", true, 128, 124, 142},
	"techno":    {"techno", "Música: Techno", true, 138, 132, 146},
	"disco":     {"disco", "Música: Disco", true, 116, 110, 122},
	"reggaeton": {"reggaeton", "Música: Reggaetón", true, 94, 90, 100},
	"punk":      {"punk", "Música: Punk", true, 172, 164, 184},
	"pop":       {"pop", "Música: Pop", false, 116, 108, 124},
	"metal":     {"metal", "Música: Metal", false, 160, 150, 172},
	"rap":       {"rap", "Música: Rap", false, 92, 86, 98},
	"hiphop":    {"hiphop", "Música: Hip-hop", false, 80, 74, 86},
	"clasica":   {"clasica", "Música: Clásica", false, 132, 124, 144},
	"jazz":      {"jazz", "Música: Jazz", false, 132, 124, 140},
	"country":   {"country", "Música: Country", false, 112, 104, 120},
	"flamenco":  {"flamenco", "Música: Pasodoble", false, 120, 112, 128},
	"rock":      {"rock", "Música: Rock", false, 124, 116, 132},
	"synthwave": {"synthwave", "Música: Synthwave", false, 100, 94, 108},
	"orquestal": {"orquestal", "Música: Orquestal", false, 108, 100, 116},
	"epico":     {"epico", "Música: Épica", false, 100, 94, 108},
	"isla":      {"isla", "Música: Orquestal épica", false, 72, 64, 80},
	"chiptune":  {"chiptune", "Música: Chiptune", false, 144, 136, 152},
}

// genreOfTitle: the style of every party of the campaign (and of the random pool), by the title of the level.
var genreOfTitle = map[string]string{
	"HOUSE PARTY":           "house",
	"MIAMI BEACH":           "reggaeton",
	"GLOW NEON":             "synthwave",
	"PLAZA DE TOROS":        "flamenco",
	"PARÍS":                 "clasica",
	"CRUCERO NOCTURNO":      "disco",
	"PARQUE DE ATRACCIONES": "chiptune",
	"ESTADIO":               "pop",
	"DISCOTECA":             "techno",
	"FESTIVAL":              "electro",
	"SKATE PARK":            "punk",
	"CEMENTERIO":            "metal",
	"CINE":                  "orquestal",
	"CIRCUITO":              "electro",
	"ESTACIÓN DE ESQUÍ":     "pop",
	"ROOFTOP PARTY":         "hiphop",
	"CASINO":                "jazz",
	"AUTOPISTA":             "rock",
	"ESTACIÓN DE TREN":      "breakbeat", // v1.4.0: the rap loop was too dull (trainmusic.go)
	"ESTACIÓN ESPACIAL":     "electro",
	"ISLA NUBLADA":          "isla",
	"PUB":                   "rock",
	"PLAZA EUROPEA":         "flamenco",
	"CENTRO COMERCIAL":      "pop",
}

var genreFallback = []string{"house", "pop", "rock", "electro", "disco", "reggaeton", "hiphop", "synthwave"}

// genreForTitle: the genre of a party; the levels made by the user get one from their name.
func genreForTitle(title string) genreInfo {
	id, ok := genreOfTitle[strings.ToUpper(title)]
	if !ok {
		h := fnv.New32a()
		h.Write([]byte(title))
		id = genreFallback[int(h.Sum32())%len(genreFallback)]
	}
	return genreTable[id]
}

// levelGenreLabel is what the title card says about the music, only while the synthesised music is playing.
func (g *Game) levelGenreLabel() string {
	if g.lvl == nil || g.djDead || g.ownStyleBand() {
		return ""
	}
	gi := genreForTitle(g.lvl.Def.Title)
	if g.musicMode != "" && !strings.HasSuffix(g.musicMode, "-synth") {
		// a real mp3 is playing: its style is only ours to say when its name says it is the style of the party
		if genreFromFileName(g.songFile) == gi.id {
			return gi.label
		}
		return ""
	}
	return gi.label
}

// genrePlaylist: three songs of the genre for a party, each in its own key, speed and reading of the tune.
func genrePlaylist(title string) []songSpec {
	gi := genreForTitle(title)
	h := fnv.New32a()
	h.Write([]byte(title))
	r := rand.New(rand.NewSource(int64(h.Sum32())))
	keys := []int{0, 2, 3, 5, -2}
	s0 := keys[r.Intn(len(keys))]
	var out []songSpec
	for i := 0; i < 3; i++ {
		if gi.legacy {
			roots := []float64{55, 58.27, 61.74, 65.41, 73.42, 49, 51.91}
			sc := [][]int{scaleMinor, scalePhryg, scaleMinor, scaleMajor}
			scale := sc[r.Intn(len(sc))]
			if gi.id == "disco" || gi.id == "reggaeton" {
				scale = scaleMinor
			}
			root := roots[r.Intn(len(roots))]
			if gi.id == "punk" {
				scale = scaleMajor
				root = []float64{82.41, 73.42, 98.0, 110.0, 87.31}[r.Intn(5)]
			}
			bpm := gi.bpm + float64(r.Intn(7)-3)
			out = append(out, songSpec{style: gi.id, bpm: bpm, root: root, scale: scale, seed: int64(h.Sum32()) + int64(i)*101, bars: 8})
			continue
		}
		mul := []float64{1, 1.04, 0.97}[i]
		shift := s0 + []int{0, 5, 7}[i]
		if gi.id == "isla" {
			shift = []int{0, 2, -2}[i] // the horn theme must stay in a noble range: D major, E major, C major
		}
		out = append(out, songSpec{style: gi.id, bpm: genreSongs[gi.id].bpm * mul, shift: shift, variant: i, seed: int64(h.Sum32()) + int64(i)*101, bars: 8})
	}
	return out
}

// genreSiblings: the other songs of the playlist now playing, so they are ready when it is their turn.
func genreSiblings(g *Game, title string) []songSpec {
	if g.djDead {
		return nil
	}
	// the song the DJ's death would start is rendered beforehand too (songs_afterdj.go)
	next := afterDJPlaylist()[afterDJNext(g.rng)]
	if g.ownStyleBand() {
		return append(bandPlaylist(g.band.slug), next)
	}
	return append(playlistFor(title), next)
}

// genreSongPCM renders a song of one of the new genres.
func genreSongPCM(sp songSpec) []byte {
	gs := genreSongs[sp.style]
	mul := sp.bpm / gs.bpm
	x := gs.render(sp.shift, mul, sp.variant == 2 && !gs.noRetro)
	if gs.rotVariant > 0 && sp.variant == gs.rotVariant { // the loop is seamless: starting half way round puts the sections in another order
		h := len(x) / 2
		x = append(append([]float64{}, x[h:]...), x[:h]...)
	}
	if isAfterDJ(sp.style) { // the songs after the DJ (songs_afterdj.go) are a little quieter
		return finishGenreRMS(x, tdjRMS)
	}
	return finishGenre(x)
}

// bandGenre: what a band plays (a famous group of the party keeps to its own style; the others are punk).
func bandGenre(slug string) string {
	switch slug {
	case "metallica":
		return "metal"
	case "beatles":
		return "pop"
	case "lostprophets", "despistaos":
		return "rock"
	case "elvis":
		return "country"
	}
	return "punk"
}

// ownStyleBand: a band is playing and its style is not the punk of the others: then its songs replace the music of the party
// (the punk bands play the style of the party, so every party keeps its own genre).
func (g *Game) ownStyleBand() bool {
	return g.band != nil && bandGenre(g.band.slug) != "punk"
}

// bandPlaylist: the songs a band plays on its stage.
func bandPlaylist(slug string) []songSpec {
	id := bandGenre(slug)
	if id == "punk" {
		return punkPlaylist()
	}
	gs := genreSongs[id]
	var out []songSpec
	for i := 0; i < 3; i++ {
		out = append(out, songSpec{style: id, bpm: gs.bpm * []float64{1, 1.03, 0.98}[i], shift: []int{0, 3, 5}[i], variant: i, seed: int64(7000 + i*53 + len(slug)), bars: 8})
	}
	return out
}

// ---- rendering in the background ---------------------------------------------------------------

var (
	genreMu    sync.Mutex
	genreReady = map[string][]byte{}
	genreBusy  bool

	genreLastWarm string // the track key whose playlist was last sent to the background thread
)

func genreKey(sp songSpec) string {
	return fmt.Sprintf("%s:%d:%d:%.2f", sp.style, sp.shift, sp.variant, sp.bpm)
}

// genreWarm renders the songs of a party in a background thread (a song takes about a second);
// whatever is not in the list any more is forgotten.
func genreWarm(specs []songSpec) {
	genreMu.Lock()
	defer genreMu.Unlock()
	keep := map[string]bool{}
	for _, sp := range specs {
		keep[genreKey(sp)] = true
	}
	for k := range genreReady {
		if !keep[k] {
			delete(genreReady, k)
		}
	}
	if genreBusy {
		return
	}
	var todo []songSpec
	for _, sp := range specs {
		if _, ok := genreSongs[sp.style]; ok {
			if _, done := genreReady[genreKey(sp)]; !done {
				todo = append(todo, sp)
			}
		}
	}
	if len(todo) == 0 {
		return
	}
	genreBusy = true
	go func() {
		for _, sp := range todo {
			pcm := genreSongPCM(sp)
			genreMu.Lock()
			genreReady[genreKey(sp)] = pcm
			genreMu.Unlock()
		}
		genreMu.Lock()
		genreBusy = false
		genreMu.Unlock()
	}()
}

// genreReadyPCM: the song if it has been rendered already.
func genreReadyPCM(sp songSpec) []byte {
	genreMu.Lock()
	defer genreMu.Unlock()
	return genreReady[genreKey(sp)]
}
