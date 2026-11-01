package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Songs: a tiny step sequencer. Every level has its own playlist (house, techno, disco,
// reggaeton...), the menu keeps the original electro theme and when the DJ dies the
// party switches to punk rock.
// ---------------------------------------------------------------------------

type songSpec struct {
	style string
	bpm   float64
	root  float64
	scale []int
	seed  int64
	bars  int
}

var (
	scaleMinor   = []int{0, 2, 3, 5, 7, 8, 10}
	scalePhryg   = []int{0, 1, 3, 5, 7, 8, 10}
	scaleMajor   = []int{0, 2, 4, 5, 7, 9, 11}
	scaleMinPent = []int{0, 3, 5, 7, 10}
)

func semi(root float64, n int) float64 { return root * math.Pow(2, float64(n)/12) }

func degree(root float64, scale []int, d int) float64 {
	oct := d / len(scale)
	i := d % len(scale)
	if i < 0 {
		i += len(scale)
		oct--
	}
	return semi(root, scale[i]+12*oct)
}

type songBuf struct {
	out  []float64
	step float64 // seconds per 16th
	rng  *rand.Rand
}

func (s *songBuf) at(step int) int { return int(float64(step) * s.step * sampleRate) }

func (s *songBuf) add(start int, snip []float64, vol float64) {
	for i, v := range snip {
		j := start + i
		if j >= len(s.out) {
			j -= len(s.out) // wrap so loops are seamless
		}
		if j >= 0 && j < len(s.out) {
			s.out[j] += v * vol
		}
	}
}

func kickSnip(punch float64) []float64 {
	n := int(0.28 * sampleRate)
	out := make([]float64, n)
	ph := 0.0
	for i := range out {
		t := float64(i) / sampleRate
		f := 46 + 130*math.Exp(-t*28)
		ph += 2 * math.Pi * f / sampleRate
		out[i] = math.Sin(ph) * math.Exp(-t*9) * punch
	}
	return out
}

func snareSnip(tone, noiseAmt float64) []float64 {
	n := int(0.22 * sampleRate)
	out := make([]float64, n)
	r := rand.New(rand.NewSource(11))
	for i := range out {
		t := float64(i) / sampleRate
		out[i] = (math.Sin(2*math.Pi*185*t)*tone + (r.Float64()*2-1)*noiseAmt) * math.Exp(-t*22)
	}
	return out
}

func hatSnip(open bool) []float64 {
	d := 0.04
	dec := 70.0
	if open {
		d, dec = 0.22, 16
	}
	n := int(d * sampleRate)
	out := make([]float64, n)
	r := rand.New(rand.NewSource(5))
	prev := 0.0
	for i := range out {
		t := float64(i) / sampleRate
		x := r.Float64()*2 - 1
		out[i] = (x - prev*0.6) * math.Exp(-t*dec) // crude high-pass
		prev = x
	}
	return out
}

func crashSnip() []float64 {
	n := int(0.9 * sampleRate)
	out := make([]float64, n)
	r := rand.New(rand.NewSource(9))
	prev := 0.0
	for i := range out {
		t := float64(i) / sampleRate
		x := r.Float64()*2 - 1
		out[i] = (x - prev*0.5) * math.Exp(-t*4)
		prev = x
	}
	return out
}

// note renders one oscillator note. kind: 0 sine, 1 square, 2 saw, 3 pluck, 4 guitar power chord
func noteSnip(freq, dur float64, kind int, decay float64) []float64 {
	n := int(dur * sampleRate)
	out := make([]float64, n)
	for i := range out {
		t := float64(i) / sampleRate
		env := math.Exp(-decay*t) * math.Min(1, (dur-t)*40)
		var v float64
		switch kind {
		case 0:
			v = math.Sin(2 * math.Pi * freq * t)
		case 1:
			if math.Sin(2*math.Pi*freq*t) >= 0 {
				v = 0.7
			} else {
				v = -0.7
			}
		case 2:
			v = (2*math.Mod(freq*t, 1) - 1) * 0.8
		case 3:
			v = (2*math.Mod(freq*t, 1) - 1) * 0.5
			v += math.Sin(2*math.Pi*freq*2*t) * 0.3
		case 4:
			v = (2*math.Mod(freq*t, 1)-1)*0.6 + (2*math.Mod(freq*1.4983*t, 1)-1)*0.5 + (2*math.Mod(freq*2*t, 1)-1)*0.4
			v = math.Tanh(v * 2.6)
		}
		out[i] = v * env
	}
	return out
}

func renderSong(sp songSpec) []byte {
	rng := rand.New(rand.NewSource(sp.seed))
	beat := 60 / sp.bpm
	bars := sp.bars
	if bars == 0 {
		bars = 8
	}
	total := beat * 4 * float64(bars)
	sb := &songBuf{out: make([]float64, int(total*sampleRate)), step: beat / 4, rng: rng}
	steps := bars * 16

	kick, snare, clap := kickSnip(0.9), snareSnip(0.5, 0.45), snareSnip(0.1, 0.8)
	hatC, hatO := hatSnip(false), hatSnip(true)
	prog := []int{0, 5, 3, 4}
	if sp.style == "punk" {
		prog = [][]int{{0, 5, 3, 4}, {0, 3, 4, 3}, {0, 4, 5, 3}, {0, 6, 5, 6}, {0, 3, 0, 4}}[rng.Intn(5)]
	} else {
		prog = [][]int{{0, 5, 3, 4}, {0, 3, 5, 4}, {0, 6, 5, 4}, {0, 4, 5, 3}, {0, 2, 5, 4}}[rng.Intn(5)]
	}
	// melody: 32 steps with rests, from the scale
	mel := make([]int, 32)
	for i := range mel {
		if rng.Float64() < 0.55 {
			mel[i] = -99
		} else {
			mel[i] = []int{0, 2, 4, 7, 9, 4, 2, 5}[rng.Intn(8)]
		}
	}
	for st := 0; st < steps; st++ {
		bar := st / 16
		s16 := st % 16
		chord := prog[(bar/2)%len(prog)]
		rootF := degree(sp.root, sp.scale, chord)
		pos := sb.at(st)
		switch sp.style {
		case "electro", "techno":
			if s16%4 == 0 {
				sb.add(pos, kick, 0.9)
			}
			if s16 == 4 || s16 == 12 {
				sb.add(pos, snare, 0.45)
			}
			if s16%4 == 2 {
				sb.add(pos, hatO, 0.10)
			}
			if sp.style == "techno" {
				if s16%2 == 1 {
					sb.add(pos, hatC, 0.08)
				}
				if s16%4 == 2 {
					sb.add(pos, noteSnip(rootF/2, sb.step*3, 2, 6), 0.2)
				}
			} else {
				if s16%2 == 0 {
					b := rootF / 2
					if s16%8 == 6 {
						b = rootF * 0.75
					}
					sb.add(pos, noteSnip(b, sb.step*1.8, 1, 5), 0.2)
				}
			}
			if m := mel[st%32]; m != -99 && (bar >= 1) {
				sb.add(pos, noteSnip(degree(sp.root*2, sp.scale, chord+m), sb.step*1.5, 2, 7), 0.09)
			}
		case "house", "disco":
			if s16%4 == 0 {
				sb.add(pos, kick, 0.85)
			}
			if s16 == 4 || s16 == 12 {
				sb.add(pos, clap, 0.35)
			}
			if s16%4 == 2 {
				sb.add(pos, hatO, 0.12)
			}
			if sp.style == "disco" {
				b := rootF / 2
				if s16%2 == 1 {
					b *= 2
				}
				sb.add(pos, noteSnip(b, sb.step*1.6, 3, 8), 0.24)
				if s16%4 == 3 {
					for _, d := range []int{0, 2, 4} {
						sb.add(pos, noteSnip(degree(sp.root*2, sp.scale, chord+d), sb.step*1.5, 3, 9), 0.07)
					}
				}
			} else {
				if s16%4 == 2 {
					sb.add(pos, noteSnip(rootF/2, sb.step*2.5, 1, 6), 0.24)
				}
				if s16 == 3 || s16 == 10 {
					for _, d := range []int{0, 2, 4, 6} {
						sb.add(pos, noteSnip(degree(sp.root*2, sp.scale, chord+d), sb.step*2.2, 2, 8), 0.06)
					}
				}
			}
			if m := mel[st%32]; m != -99 && bar >= 2 && sp.style == "house" {
				sb.add(pos, noteSnip(degree(sp.root*4, sp.scale, chord+m), sb.step*1.2, 0, 9), 0.08)
			}
		case "reggaeton":
			if s16%4 == 0 {
				sb.add(pos, kick, 0.9)
			}
			if s16 == 3 || s16 == 6 || s16 == 11 || s16 == 14 {
				sb.add(pos, snare, 0.42)
			}
			if s16%2 == 0 {
				sb.add(pos, hatC, 0.08)
			}
			if s16 == 0 || s16 == 6 || s16 == 8 || s16 == 14 {
				sb.add(pos, noteSnip(rootF/2, sb.step*3.5, 0, 3), 0.42)
			}
			if m := mel[st%32]; m != -99 && s16%2 == 0 {
				sb.add(pos, noteSnip(degree(sp.root*2, sp.scale, chord+m), sb.step*1.8, 3, 6), 0.12)
			}
		case "punk":
			if s16 == 0 || s16 == 6 || s16 == 8 || s16 == 10 {
				sb.add(pos, kick, 0.85)
			}
			if s16 == 4 || s16 == 12 {
				sb.add(pos, snare, 0.6)
			}
			if s16%2 == 0 {
				sb.add(pos, hatC, 0.1)
			}
			if s16 == 0 && bar%4 == 0 {
				sb.add(pos, crashSnip(), 0.25)
			}
			g := degree(sp.root, sp.scale, chord)
			if s16%2 == 0 {
				muted := (s16%4 != 0) && sp.seed%2 == 0
				d := sb.step * 1.9
				if muted {
					d = sb.step * 0.8
				}
				sb.add(pos, noteSnip(g, d, 4, 5), 0.24)
				sb.add(pos, noteSnip(g/2, d, 1, 5), 0.16)
			}
			if m := mel[st%32]; m != -99 && bar%4 >= 2 && s16%2 == 1 {
				sb.add(pos, noteSnip(degree(sp.root*4, scaleMinPent, m/2), sb.step*1.2, 2, 6), 0.06)
			}
		}
	}
	// soft limiter
	for i, v := range sb.out {
		sb.out[i] = math.Tanh(v*0.9) * 0.9
	}
	return toPCM(sb.out)
}

// playlist returns the track specs for a level title: three songs in different styles.
func playlistFor(title string) []songSpec {
	h := fnv.New32a()
	h.Write([]byte(title))
	r := rand.New(rand.NewSource(int64(h.Sum32())))
	styles := []string{"house", "electro", "techno", "disco", "reggaeton", "house", "electro"}
	r.Shuffle(len(styles), func(i, j int) { styles[i], styles[j] = styles[j], styles[i] })
	roots := []float64{55, 58.27, 61.74, 65.41, 73.42, 49, 51.91}
	scales := [][]int{scaleMinor, scalePhryg, scaleMinor, scaleMajor}
	var out []songSpec
	for i := 0; i < 3; i++ {
		st := styles[i]
		bpm := map[string]float64{"house": 122, "electro": 128, "techno": 138, "disco": 116, "reggaeton": 94}[st] + float64(r.Intn(7)-3)
		sc := scales[r.Intn(len(scales))]
		if st == "disco" || st == "reggaeton" {
			sc = scaleMinor
		}
		out = append(out, songSpec{style: st, bpm: bpm, root: roots[r.Intn(len(roots))], scale: sc, seed: int64(h.Sum32()) + int64(i)*101, bars: 8})
	}
	return out
}

// punkPlaylist: several punk rock songs that rotate while the DJ is dead.
func punkPlaylist() []songSpec {
	var out []songSpec
	roots := []float64{82.41, 73.42, 98.0, 110.0, 87.31}
	for i := 0; i < 5; i++ {
		out = append(out, songSpec{style: "punk", bpm: 168 + float64(i*7%24), root: roots[i%len(roots)], scale: scaleMajor, seed: int64(1000 + i*37), bars: 8})
	}
	return out
}

func trackKey(style string, i int, title string) string {
	return fmt.Sprintf("%s:%s:%d", style, title, i)
}

// syncMusic picks the song that fits the moment: the menu theme, the level playlist or punk after the DJ dies.
func (g *Game) syncMusic() {
	if g.snd == nil {
		return
	}
	if g.mode == ModeMenu || g.mode == ModeEditor || g.lvl == nil {
		g.musicMode = ""
		g.snd.SetTrack("menu", musicLoop)
		return
	}
	skip := g.skipSong
	if g.syncRealMusic() {
		g.skipSong = false
		return
	}
	g.skipSong = false
	if skip {
		g.songIdx++
		g.nextSong = g.time + 70 + g.rng.Float64()*40
	}
	title := g.lvl.Def.Title
	if g.time >= g.nextSong {
		g.songIdx++
		g.nextSong = g.time + 70 + g.rng.Float64()*40
		if g.mode == ModePlay {
			g.say("El DJ cambia de canción")
		}
	}
	var sp songSpec
	if g.djDead || g.band != nil {
		pl := punkPlaylist()
		sp = pl[g.songIdx%len(pl)]
		title = "punk"
	} else {
		pl := playlistFor(title)
		sp = pl[g.songIdx%len(pl)]
	}
	g.snd.SetTrack(trackKey(sp.style, int(sp.seed%1000), title), func() []byte { return renderSong(sp) })
}
