package main

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// A tiny composer: the tunes of the story scenes are written as note text
// ("C5:0.5 E5:0.5 G5:1 R:1", durations in beats, [C4 E4 G4]:2 is a chord) over a
// scale, and rendered with clean instruments (marimba, pizzicato, chiptune
// square, flute, music box, brass). There is no bass drum in any of them: the
// rhythm comes from woodblock, shaker and the plucked bass, which stays above
// ~90 Hz. Only the first intro has a kick (audio.go) and it borrows the hook
// of this file.
// ---------------------------------------------------------------------------

// scaleDef is a key: the pitch class of its root and the semitones of its scale.
type scaleDef struct {
	name   string
	rootPC int // 0 = C ... 11 = B
	steps  []int
}

var (
	stepsMajor = []int{0, 2, 4, 5, 7, 9, 11}
	stepsMinor = []int{0, 2, 3, 5, 7, 8, 10}

	scCMajor = scaleDef{"C major", 0, stepsMajor}
	scCMinor = scaleDef{"C minor", 0, stepsMinor}
	scGMajor = scaleDef{"G major", 7, stepsMajor}
	scGMinor = scaleDef{"G minor", 7, stepsMinor}
	scFMajor = scaleDef{"F major", 5, stepsMajor}
	scAMinor = scaleDef{"A minor", 9, stepsMinor}
)

// has: does the note (MIDI number) belong to the scale?
func (sc scaleDef) has(midi int) bool {
	pc := ((midi-sc.rootPC)%12 + 12) % 12
	for _, s := range sc.steps {
		if s == pc {
			return true
		}
	}
	return false
}

func midiFreq(m int) float64 { return 440 * math.Pow(2, float64(m-69)/12) }

// freqMidi: the MIDI note of a frequency (ok = false if it is not (nearly) a note of the tempered scale).
func freqMidi(f float64) (int, bool) {
	x := 12*math.Log2(f/440) + 69
	m := int(math.Round(x))
	return m, math.Abs(x-float64(m)) < 0.02
}

// songNote: a note or a chord (several MIDI numbers) starting `at` beats from the start, lasting `dur` beats.
type songNote struct {
	at, dur float64
	midi    []int // empty for a rest
}

var pitchClass = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}

// parseNote reads "C5", "F#4", "Bb3".
func parseNote(s string) (int, bool) {
	if len(s) < 2 {
		return 0, false
	}
	pc, ok := pitchClass[s[0]]
	if !ok {
		return 0, false
	}
	i := 1
	switch s[i] {
	case '#':
		pc++
		i++
	case 'b':
		pc--
		i++
	}
	oct, err := strconv.Atoi(s[i:])
	if err != nil {
		return 0, false
	}
	return 12*(oct+1) + pc, true
}

// parseNotes reads a line of note text: "C5:0.5 [E4 G4]:1 R:1 | ..." ('|' only separates bars for the eyes).
func parseNotes(text string) []songNote {
	var out []songNote
	at := 0.0
	for _, tok := range strings.Fields(strings.ReplaceAll(text, "|", " ")) {
		colon := strings.LastIndex(tok, ":")
		if colon < 0 {
			panic("song: note without a duration: " + tok)
		}
		d, err := strconv.ParseFloat(tok[colon+1:], 64)
		if err != nil {
			panic("song: bad duration in " + tok)
		}
		name := tok[:colon]
		n := songNote{at: at, dur: d}
		switch {
		case name == "R":
		case strings.HasPrefix(name, "["):
			for _, p := range strings.Split(strings.Trim(name, "[]"), ",") {
				m, ok := parseNote(p)
				if !ok {
					panic("song: bad note " + p)
				}
				n.midi = append(n.midi, m)
			}
		default:
			m, ok := parseNote(name)
			if !ok {
				panic("song: bad note " + name)
			}
			n.midi = []int{m}
		}
		out = append(out, n)
		at += d
	}
	return out
}

// noteLine builds a line of note text from bars; a chord is written [C4,E4,G4] (no spaces).
func noteLine(bars ...string) string { return strings.Join(bars, " | ") }

// transposeText moves note text by some semitones (to make the tense version of a tune in another key).
func transposeText(text string, semis int) string {
	toks := strings.Fields(text)
	for i, tok := range toks {
		colon := strings.LastIndex(tok, ":")
		if colon < 0 || tok == "|" {
			continue
		}
		name, dur := tok[:colon], tok[colon:]
		if name == "R" {
			continue
		}
		var parts []string
		for _, p := range strings.Split(strings.Trim(name, "[]"), ",") {
			m, ok := parseNote(p)
			if !ok {
				panic("transpose: bad note " + p)
			}
			parts = append(parts, midiName(m+semis))
		}
		if strings.HasPrefix(name, "[") {
			toks[i] = "[" + strings.Join(parts, ",") + "]" + dur
		} else {
			toks[i] = parts[0] + dur
		}
	}
	return strings.Join(toks, " ")
}

var pcNames = []string{"C", "C#", "D", "Eb", "E", "F", "F#", "G", "Ab", "A", "Bb", "B"}

func midiName(m int) string { return pcNames[((m%12)+12)%12] + strconv.Itoa(m/12-1) }

// ---- instruments: freq, duration (seconds), volume -> samples -----------------------------

type instrument struct {
	play    func(f, dur, vol float64) []float64
	sustain bool // lasts as long as the note (flute, brass); plucked ones ring their own time
}

func envAR(t, dur, atk, rel float64) float64 { return smooth01(t/atk) * smooth01((dur-t)/rel) }

var instruments = map[string]instrument{
	// marimba / xylophone: a round fundamental and a bright knock
	"marimba": {play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.35) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			v := math.Sin(2*math.Pi*f*t)*math.Exp(-7*t) + 0.28*math.Sin(2*math.Pi*f*3.9*t)*math.Exp(-24*t)
			out[i] = v * math.Min(1, t/0.002) * vol
		}
		return out
	}},
	// plucked double bass / cello: clean, short, always above ~90 Hz in the tunes
	"pizz": {play: func(f, dur, vol float64) []float64 {
		n := int(0.42 * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			v := math.Sin(2*math.Pi*f*t) + 0.35*math.Sin(2*math.Pi*f*2*t)*math.Exp(-14*t) + 0.12*math.Sin(2*math.Pi*f*3*t)*math.Exp(-22*t)
			out[i] = v * math.Exp(-9*t) * math.Min(1, t/0.003) * vol
		}
		return out
	}},
	// chiptune square lead, clean (a few harmonics only), with a short pluck
	"square": {sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.05) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			ph := 2 * math.Pi * f * t
			v := math.Sin(ph) + math.Sin(3*ph)/3 + math.Sin(5*ph)/5 + math.Sin(7*ph)/7
			out[i] = v * 0.55 * envAR(t, dur, 0.006, 0.05) * (0.7 + 0.3*math.Exp(-6*t)) * vol
		}
		return out
	}},
	"flute": {sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.2) * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			vib := 1 + 0.004*math.Sin(2*math.Pi*5.2*t)*smooth01(t/0.4)
			ph := 2 * math.Pi * f * t * vib
			v := math.Sin(ph) + 0.12*math.Sin(2*ph) + 0.04*math.Sin(3*ph)
			out[i] = v * envAR(t, dur+0.1, 0.07, 0.14) * vol
		}
		return out
	}},
	"box": {play: func(f, dur, vol float64) []float64 {
		n := int(1.6 * sampleRate)
		out := make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			v := math.Sin(2*math.Pi*f*t)*math.Exp(-2.6*t) + 0.3*math.Sin(2*math.Pi*f*3*t)*math.Exp(-6*t) + 0.12*math.Sin(2*math.Pi*f*5*t)*math.Exp(-9*t)
			out[i] = v * math.Min(1, t/0.003) * vol
		}
		return out
	}},
	"brass": {sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.12) * sampleRate)
		out := make([]float64, n)
		w := []float64{1, 0.7, 0.5, 0.32, 0.18, 0.08}
		for i := range out {
			t := float64(i) / sampleRate
			vib := 1 + 0.003*math.Sin(2*math.Pi*5.5*t)*smooth01(t/0.3)
			var v float64
			for k, a := range w {
				v += a * math.Sin(2*math.Pi*f*float64(k+1)*t*vib)
			}
			out[i] = v * 0.42 * envAR(t, dur, 0.045, 0.09) * vol
		}
		return out
	}},
}

// percussion: one-shot clean sounds, no kick
var (
	percCache = map[byte][]float64{}
	percMu    sync.Mutex // the menu's track and the scene songs are built at the same time (scenemusic.go)
)

func percSample(k byte) []float64 {
	percMu.Lock()
	defer percMu.Unlock()
	if p, ok := percCache[k]; ok {
		return p
	}
	r := rand.New(rand.NewSource(7 + int64(k)))
	var out []float64
	switch k {
	case 'w': // woodblock
		out = make([]float64, int(0.09*sampleRate))
		for i := range out {
			t := float64(i) / sampleRate
			out[i] = (math.Sin(2*math.Pi*1750*t)*math.Exp(-62*t) + 0.4*math.Sin(2*math.Pi*2650*t)*math.Exp(-90*t)) * 0.9
		}
	case 'h': // closed hat: a short tick of high noise
		n := int(0.045 * sampleRate)
		raw := make([]float64, n)
		for i := range raw {
			raw[i] = r.Float64()*2 - 1
		}
		lo := lowpass(raw, 8)
		out = make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			out[i] = (raw[i] - lo[i]) * math.Exp(-85*t) * 0.5
		}
	case 's': // shaker
		n := int(0.08 * sampleRate)
		raw := make([]float64, n)
		for i := range raw {
			raw[i] = r.Float64()*2 - 1
		}
		lo := lowpass(raw, 5)
		out = make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			out[i] = (raw[i] - lo[i]) * smooth01(t/0.012) * math.Exp(-38*t) * 0.5
		}
	case 'r': // rim click / light snare
		n := int(0.1 * sampleRate)
		raw := make([]float64, n)
		for i := range raw {
			raw[i] = r.Float64()*2 - 1
		}
		lo := lowpass(raw, 4)
		out = make([]float64, n)
		for i := range out {
			t := float64(i) / sampleRate
			out[i] = ((raw[i]-lo[i])*0.45 + math.Sin(2*math.Pi*1150*t)*0.5) * math.Exp(-48*t)
		}
	}
	percCache[k] = out
	return out
}

// ---- songs -----------------------------------------------------------------------------------------

type trackDef struct {
	inst  string
	notes string
	vol   float64
}

type percDef struct {
	pattern string // 16 steps (sixteenth notes) for every bar: 'w' woodblock, 'h' hat, 's' shaker, 'r' rim
	vol     float64
}

type songDef struct {
	name   string
	bpm    float64
	bars   int
	scale  scaleDef
	tracks []trackDef
	perc   []percDef
	echo   float64 // delay of the echo in beats (0 = none)
	fb     float64
	wet    float64
}

// loopSamples: how long the loop is, in samples, and how long a beat is.
func (sd songDef) loopSamples() (n int, spb float64) {
	spb = 60 / sd.bpm * sampleRate
	return int(math.Round(float64(sd.bars*4) * spb)), spb
}

// composeSong plays the song into one seamless loop (the tails wrap round to the start).
func composeSong(sd songDef) []float64 {
	n, spb := sd.loopSamples()
	out := make([]float64, n)
	for _, tr := range sd.tracks {
		inst, ok := instruments[tr.inst]
		if !ok {
			panic("song: no instrument " + tr.inst)
		}
		for _, nt := range parseNotes(tr.notes) {
			at := int(math.Round(nt.at * spb))
			secs := nt.dur * 60 / sd.bpm
			for _, m := range nt.midi {
				for i, v := range inst.play(midiFreq(m), secs, tr.vol) {
					out[(at+i)%n] += v
				}
			}
		}
	}
	for _, pc := range sd.perc {
		for bar := 0; bar < sd.bars; bar++ {
			for step, ch := range []byte(pc.pattern) {
				if ch == '.' {
					continue
				}
				at := int(math.Round((float64(bar*4) + float64(step)/4) * spb))
				for i, v := range percSample(ch) {
					out[(at+i)%n] += v * pc.vol
				}
			}
		}
	}
	if sd.echo > 0 {
		out = echoLoop(out, sd.echo*60/sd.bpm, sd.fb, sd.wet)
	}
	return out
}

// finishSong takes out any DC offset and brings the peak to `peak` (always below 0 dBFS).
func finishSong(x []float64, peak float64) []byte {
	var mean float64
	for _, v := range x {
		mean += v
	}
	mean /= float64(len(x))
	m := 0.0
	for i := range x {
		x[i] -= mean
		m = math.Max(m, math.Abs(x[i]))
	}
	if m > 0 {
		for i := range x {
			x[i] *= peak / m
		}
	}
	return toPCM(x)
}

// ---- the tunes -------------------------------------------------------------------------------------

const (
	barRest = "R:4"
)

// the bars of the angry march in C major (also the base of the tense version in C minor)
var marchMel = noteLine(
	"C5:0.5 C5:0.5 C5:0.5 E5:0.5 G5:1 R:1",
	"A5:0.5 A5:0.5 A5:0.5 F5:0.5 G5:1 R:1",
	"E5:0.5 E5:0.5 E5:0.5 G5:0.5 C6:1 R:0.5 B5:0.5",
	"A5:0.5 G5:0.5 F5:0.5 E5:0.5 D5:1 R:1",
	"C5:0.5 C5:0.5 C5:0.5 E5:0.5 G5:1 R:1",
	"F5:0.5 F5:0.5 F5:0.5 A5:0.5 C6:1 R:1",
	"B5:0.5 A5:0.5 G5:0.5 F5:0.5 E5:0.5 D5:0.5 E5:0.5 G5:0.5",
	"C5:1 G5:0.5 E5:0.5 C5:1 R:1",
)

var marchBass = noteLine(
	"C3:1 R:1 G3:1 R:1", "F3:1 R:1 C4:1 R:1", "C3:1 R:1 G3:1 R:1", "G3:1 R:1 D4:1 R:1",
	"C3:1 R:1 G3:1 R:1", "F3:1 R:1 C4:1 R:1", "G3:1 R:1 D4:1 R:1", "C3:1 R:1 G3:1 R:1",
)

var marchStab = noteLine(
	"R:1 [E4,G4]:1 R:1 [E4,G4]:1", "R:1 [A4,C5]:1 R:1 [A4,C5]:1", "R:1 [E4,G4]:1 R:1 [E4,G4]:1", "R:1 [B4,D5]:1 R:1 [B4,D5]:1",
	"R:1 [E4,G4]:1 R:1 [E4,G4]:1", "R:1 [A4,C5]:1 R:1 [A4,C5]:1", "R:1 [B4,D5]:1 R:1 [B4,D5]:1", "R:1 [E4,G4]:1 R:1 [E4,G4]:1",
)

// the sneaky version for the memories: C minor, walking pizzicato and a staccato knock
var sneakMel = noteLine(
	"G5:0.5 R:0.5 G5:0.5 R:0.5 Eb5:0.5 F5:0.5 G5:1",
	"R:0.5 Bb5:0.5 R:0.5 Bb5:0.5 G5:1 R:1",
	"F5:0.5 R:0.5 F5:0.5 R:0.5 Ab5:0.5 G5:0.5 F5:1",
	"D5:0.5 D5:0.5 G5:0.5 D5:0.5 Bb5:1 R:1",
	"G5:0.5 R:0.5 G5:0.5 R:0.5 Eb5:0.5 F5:0.5 G5:1",
	"Ab5:0.5 R:0.5 Eb5:0.5 R:0.5 C6:1 R:1",
	"Bb5:0.5 Ab5:0.5 G5:0.5 F5:0.5 Eb5:0.5 D5:0.5 C5:1",
	"C5:0.5 R:0.5 C5:0.5 R:0.5 G4:1 R:1",
)

func walking(root, fifth string) string {
	return root + ":0.5 " + fifth + ":0.5 " + root + ":0.5 " + fifth + ":0.5 " + root + ":0.5 " + fifth + ":0.5 " + root + ":0.5 " + fifth + ":0.5"
}

var sneakBass = noteLine(
	walking("C3", "G3"), walking("C3", "G3"), walking("F3", "C4"), walking("G3", "D4"),
	walking("C3", "G3"), walking("Ab3", "Eb4"), walking("G3", "D4"), walking("C3", "G3"),
)

// the boat: a jaunty sea march in G major with a square lead
var seaLead = noteLine(
	"G4:0.5 B4:0.5 D5:0.5 B4:0.5 G4:1 R:1",
	"A4:0.5 C5:0.5 E5:0.5 C5:0.5 A4:1 R:1",
	"B4:0.5 D5:0.5 G5:0.5 D5:0.5 B4:1 A4:0.5 G4:0.5",
	"A4:0.5 F#4:0.5 D4:0.5 F#4:0.5 G4:2",
	"G4:0.5 B4:0.5 D5:0.5 B4:0.5 G4:1 R:1",
	"E5:0.5 E5:0.5 E5:0.5 C5:0.5 G5:1 R:1",
	"F#5:0.5 E5:0.5 D5:0.5 C5:0.5 B4:0.5 A4:0.5 B4:0.5 D5:0.5",
	"G4:1 D5:0.5 B4:0.5 G4:1 R:1",
)

var seaBass = noteLine(
	"G3:1 R:1 D4:1 R:1", "C3:1 R:1 G3:1 R:1", "G3:1 R:1 D4:1 R:1", "D3:1 R:1 A3:1 R:1",
	"G3:1 R:1 D4:1 R:1", "C3:1 R:1 G3:1 R:1", "D3:1 R:1 A3:1 R:1", "G3:1 R:1 D4:1 R:1",
)

var seaStab = noteLine(
	"R:1 [B3,D4]:1 R:1 [B3,D4]:1", "R:1 [E4,G4]:1 R:1 [E4,G4]:1", "R:1 [B3,D4]:1 R:1 [B3,D4]:1", "R:1 [F#4,A4]:1 R:1 [F#4,A4]:1",
	"R:1 [B3,D4]:1 R:1 [B3,D4]:1", "R:1 [E4,G4]:1 R:1 [E4,G4]:1", "R:1 [F#4,A4]:1 R:1 [F#4,A4]:1", "R:1 [B3,D4]:1 R:1 [B3,D4]:1",
)

// the lullaby: F major, a music box and a flute
var lullMel = noteLine(
	"A5:1 C6:1 A5:1 F5:1", "G5:1 Bb5:1 G5:1 E5:1", "F5:1 A5:1 C6:1 A5:1", "G5:1 E5:1 F5:2",
	"C6:1 A5:1 F5:1 A5:1", "D6:1 C6:1 Bb5:1 G5:1", "A5:1 G5:1 E5:1 G5:1", "F5:2 R:2",
)

var lullFlute = noteLine(
	barRest, barRest, barRest, barRest,
	"F5:2 A5:2", "G5:2 Bb5:2", "C6:2 A5:2", "F5:4",
)

var lullBass = noteLine(
	"F3:2 C4:2", "G3:2 D4:2", "F3:2 C4:2", "C3:2 C4:2", "F3:2 C4:2", "Bb2:2 F3:2", "C3:2 G3:2", "F3:4",
)

// the calm end of the ending: slow, sparse, the same lullaby fading away
var calmBox = noteLine("F5:2 A5:2", "C6:2 A5:2", "G5:2 F5:2", "F5:4")
var calmFlute = noteLine("[F4,A4,C5]:8", "[F4,A4,C5]:8")

// the secret ending: a festive fanfare in C major
var fanfareMel = noteLine(
	"C5:0.5 C5:0.5 C5:0.5 C5:0.5 E5:1 G5:1",
	"C6:1.5 G5:0.5 G5:2",
	"A5:0.5 A5:0.5 A5:0.5 A5:0.5 F5:1 A5:1",
	"G5:2 E5:1 D5:1",
	"C5:0.5 E5:0.5 G5:0.5 C6:0.5 G5:1 E5:1",
	"F5:0.5 A5:0.5 C6:0.5 A5:0.5 F5:1 D5:1",
	"G5:1 B5:1 D6:1 B5:1",
	"C6:3 R:1",
)

var fanfareChords = noteLine(
	"[E4,G4]:2 [E4,G4]:2", "[E4,G4]:2 [E4,G4]:2", "[A4,C5]:2 [A4,C5]:2", "[B4,D5]:2 [B4,D5]:2",
	"[E4,G4]:2 [E4,G4]:2", "[A4,C5]:2 [A4,C5]:2", "[B4,D5]:2 [B4,D5]:2", "[E4,G4]:4",
)

var fanfareBells = noteLine(
	barRest, barRest, barRest, barRest,
	"R:0.5 G5:0.5 R:0.5 C6:0.5 R:0.5 E5:0.5 R:0.5 G5:0.5",
	"R:0.5 A5:0.5 R:0.5 C6:0.5 R:0.5 F5:0.5 R:0.5 A5:0.5",
	"R:0.5 B5:0.5 R:0.5 D6:0.5 R:0.5 G5:0.5 R:0.5 B5:0.5",
	"R:0.5 G5:0.5 R:0.5 C6:0.5 R:0.5 E5:0.5 R:0.5 G5:0.5",
)

var fanfareBass = noteLine(
	"C3:1 R:1 G3:1 R:1", "C3:1 R:1 G3:1 R:1", "F3:1 R:1 C4:1 R:1", "G3:1 R:1 D4:1 R:1",
	"C3:1 R:1 G3:1 R:1", "F3:1 R:1 C4:1 R:1", "G3:1 R:1 D4:1 R:1", "C3:1 R:1 G3:1 R:1",
)

// the invitation: adventure and a little mystery in A minor, a marimba ostinato under a flute
func ostinato(a, b, c string) string {
	return a + ":0.5 " + b + ":0.5 " + c + ":0.5 " + b + ":0.5 " + a + ":0.5 " + b + ":0.5 " + c + ":0.5 " + b + ":0.5"
}

var inviteOst = noteLine(
	ostinato("A3", "C4", "E4"), ostinato("A3", "C4", "E4"), ostinato("F3", "A3", "C4"), ostinato("G3", "B3", "D4"),
	ostinato("A3", "C4", "E4"), ostinato("F3", "A3", "C4"), ostinato("G3", "B3", "D4"), ostinato("E3", "G3", "B3"),
)

var inviteFlute = noteLine(
	"E5:2 A5:1 G5:1", "E5:1 D5:1 C5:2", "A5:2 G5:1 F5:1", "G5:2 D5:1 B4:1",
	"E5:2 A5:1 C6:1", "A5:1.5 G5:0.5 F5:1 E5:1", "D5:1 G5:1 B5:2", "A5:3 R:1",
)

var inviteBass = noteLine(
	"A2:1 R:1 E3:1 R:1", "A2:1 R:1 E3:1 R:1", "F3:1 R:1 C4:1 R:1", "G2:1 R:1 D3:1 R:1",
	"A2:1 R:1 E3:1 R:1", "F3:1 R:1 C4:1 R:1", "G2:1 R:1 D3:1 R:1", "E3:1 R:1 B3:1 R:1",
)

// the hook of the party next door (first intro): clean eurodance in A minor, 16 beats, over the kick
var partyHook = noteLine(
	"A4:0.5 A4:0.5 C5:0.5 A4:0.5 E5:1 D5:0.5 C5:0.5",
	"A4:0.5 A4:0.5 C5:0.5 A4:0.5 G5:1 E5:0.5 D5:0.5",
	"A4:0.5 A4:0.5 C5:0.5 A4:0.5 E5:1 D5:0.5 C5:0.5",
	"D5:0.5 C5:0.5 A4:0.5 G4:0.5 A4:2",
)

// the bass of the party next door, one note per four beats (A1 A1 C2 G1: below the 90 Hz of the other tunes: it is the kick's friend)
var partyBassMidi = []int{33, 33, 36, 31}

const (
	ambInterA  = "interA"
	ambInterB  = "interB"
	ambMemA    = "memoryA"
	ambMemB    = "memoryB"
	ambEnd     = "endNormal"
	ambEndCalm = "endCalm"
	ambSecret  = "endSecret"
	ambInvite  = "invite"
	ambTitle   = "title"
)

// songs: every tune of the story scenes (the first intro's hook is partyHook)
var songs = map[string]songDef{
	ambInterA: {name: "interA", bpm: 120, bars: 8, scale: scCMajor, echo: 0.75, fb: 0.3, wet: 0.22,
		tracks: []trackDef{{"marimba", marchMel, 0.5}, {"pizz", marchBass, 0.55}, {"marimba", marchStab, 0.22}},
		perc:   []percDef{{"....w.......w...", 0.35}, {"..h...h...h...h.", 0.14}}},
	ambMemA: {name: "memoryA", bpm: 120, bars: 8, scale: scCMinor, echo: 0.5, fb: 0.28, wet: 0.2,
		tracks: []trackDef{{"marimba", sneakMel, 0.5}, {"pizz", sneakBass, 0.5}},
		perc:   []percDef{{"w...w...w...w...", 0.22}, {"h.h.h.h.h.h.h.h.", 0.1}}},
	ambInterB: {name: "interB", bpm: 126, bars: 8, scale: scGMajor, echo: 0.75, fb: 0.3, wet: 0.2,
		tracks: []trackDef{{"square", seaLead, 0.4}, {"pizz", seaBass, 0.55}, {"marimba", seaStab, 0.22}},
		perc:   []percDef{{"....w.......w...", 0.33}, {"s.s.s.s.s.s.s.s.", 0.16}}},
	ambMemB: {name: "memoryB", bpm: 126, bars: 8, scale: scGMinor, echo: 0.5, fb: 0.28, wet: 0.2,
		tracks: []trackDef{{"marimba", transposeText(sneakMel, 7), 0.5}, {"pizz", transposeText(sneakBass, 7), 0.5}},
		perc:   []percDef{{"w...w...w...w...", 0.22}, {"h.h.h.h.h.h.h.h.", 0.1}}},
	ambEnd: {name: "endNormal", bpm: 112, bars: 8, scale: scFMajor, echo: 0.5, fb: 0.35, wet: 0.3,
		tracks: []trackDef{{"box", lullMel, 0.5}, {"flute", lullFlute, 0.3}, {"pizz", lullBass, 0.4}},
		perc:   []percDef{{"s...s...s...s...", 0.1}}},
	ambEndCalm: {name: "endCalm", bpm: 72, bars: 4, scale: scFMajor, echo: 1, fb: 0.45, wet: 0.4,
		tracks: []trackDef{{"box", calmBox, 0.45}, {"flute", calmFlute, 0.2}}},
	ambSecret: {name: "endSecret", bpm: 132, bars: 8, scale: scCMajor, echo: 0.5, fb: 0.25, wet: 0.18,
		tracks: []trackDef{{"brass", fanfareMel, 0.5}, {"brass", fanfareChords, 0.22}, {"box", fanfareBells, 0.28}, {"pizz", fanfareBass, 0.5}},
		perc:   []percDef{{"....r.......r...", 0.3}, {"h.h.h.h.h.h.h.h.", 0.14}, {"s.s.s.s.s.s.s.s.", 0.08}}},
	ambTitle: {name: "title", bpm: 132, bars: 4, scale: scCMajor, echo: 0.5, fb: 0.3, wet: 0.25,
		tracks: []trackDef{{"brass", titleMel, 0.5}, {"brass", titleChords, 0.2}, {"marimba", titleArp, 0.28}, {"pizz", titleBass, 0.5}},
		perc:   []percDef{{"....r.......r...", 0.28}, {"h.h.h.h.h.h.h.h.", 0.12}}},
	ambInvite: {name: "invite", bpm: 116, bars: 8, scale: scAMinor, echo: 0.75, fb: 0.35, wet: 0.25,
		tracks: []trackDef{{"marimba", inviteOst, 0.42}, {"flute", inviteFlute, 0.4}, {"pizz", inviteBass, 0.5}},
		perc:   []percDef{{"..w...w...w...w.", 0.25}, {"s.s.s.s.s.s.s.s.", 0.12}}},
}

// ambientKinds lists the tunes in the order they are built in the background (the likeliest first).
var ambientKinds = []string{ambTitle, ambInterA, ambMemA, ambEnd, ambInterB, ambMemB, ambEndCalm, ambSecret, ambInvite}

func buildAmbient(kind string) []byte {
	switch kind { // the music of the first screen and the menu (menumusic.go)
	case ambAnthem:
		return buildAnthem()
	case ambSleep:
		return buildSleep()
	case ambCalm:
		return finishSong(renderAnthemMix(anthemCalm), anthemCalm.peak)
	case ambCalmer:
		return finishSong(renderAnthemMix(anthemCalmer), anthemCalmer.peak)
	}
	sd, ok := songs[kind]
	if !ok {
		return nil
	}
	return finishSong(composeSong(sd), 0.85)
}

// the title screen: a short, bright fanfare in C major (no bass drum) with a marimba sparkle under it
var titleMel = noteLine(
	"C5:0.5 E5:0.5 G5:0.5 C6:1.5 G5:1",
	"A5:0.5 A5:0.5 A5:0.5 F5:0.5 G5:2",
	"E5:0.5 G5:0.5 C6:0.5 E6:0.5 D6:1 C6:1",
	"G5:1 E5:1 C6:2",
)

var titleChords = noteLine(
	"[E4,G4]:2 [E4,G4]:2", "[F4,A4]:2 [G4,B4]:2", "[E4,G4]:2 [G4,B4]:2", "[C4,E4,G4]:4",
)

var titleArp = noteLine(
	"C4:0.5 E4:0.5 G4:0.5 E4:0.5 C4:0.5 E4:0.5 G4:0.5 E4:0.5",
	"F4:0.5 A4:0.5 C5:0.5 A4:0.5 G4:0.5 B4:0.5 D5:0.5 B4:0.5",
	"E4:0.5 G4:0.5 C5:0.5 G4:0.5 G4:0.5 B4:0.5 D5:0.5 B4:0.5",
	"C4:0.5 E4:0.5 G4:0.5 C5:0.5 E5:2",
)

var titleBass = noteLine(
	"C3:1 R:1 G3:1 R:1", "F3:1 R:1 G3:1 R:1", "C3:1 R:1 G3:1 R:1", "C3:2 G3:1 C3:1",
)
