package main

import (
	"bytes"
	"math"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// ---------------------------------------------------------------------------
// Ambient sound by level (waves, wind, crickets, a far train, a crowd murmur).
//
// OFF by default: it was written without being able to listen to it, so the user must judge it.
// Turn it on with the environment variable PH_AMBIENT_SOUND=1 (see LEEME.txt). When on, a very
// quiet loop (about 12% of the music volume) plays only while playing a level of ambientPlan,
// and it is paused in menus, interludes, help/trophy windows and whenever the sound is muted
// (-mute, key M).
// ---------------------------------------------------------------------------

// ambientSound: the switch. Tests set it directly.
var ambientSound = os.Getenv("PH_AMBIENT_SOUND") == "1"

// ambientVolume: player volume of the loop (the music plays at 0.5).
const ambientVolume = 0.06

const ambLoopSec = 8.0

// ambPlay is the one ambient player of the process (the Sounds it belongs to is kept to detect a change).
var ambPlay struct {
	owner *Sounds
	p     *audio.Player
	key   string
}

// circLow: moving-average low pass that wraps around, so a noise loop has no seam.
func circLow(in []float64, n int) []float64 {
	out := make([]float64, len(in))
	l := len(in)
	var acc float64
	for i := 0; i < n; i++ {
		acc += in[l-1-i]
	}
	for i := range in {
		acc += in[i]
		acc -= in[((i-n)%l+l)%l]
		out[i] = acc / float64(n)
	}
	return out
}

func normalize(m []float64, peak float64) {
	mx := 1e-9
	for _, v := range m {
		if a := math.Abs(v); a > mx {
			mx = a
		}
	}
	for i := range m {
		m[i] *= peak / mx
	}
}

func whiteLoop(r *rand.Rand, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = r.Float64()*2 - 1
	}
	return out
}

// buildAmbientLoop synthesises a seamless stereo loop (PCM bytes) of the named kind; nil if unknown.
func buildAmbientLoop(kind string) []byte {
	n := int(ambLoopSec * sampleRate)
	r := rand.New(rand.NewSource(titleSeed(kind)))
	sec := func(s float64) float64 { return s * sampleRate }
	var m []float64
	switch kind {
	case "waves": // pink-ish noise swelling and fading, twice per loop
		a := circLow(whiteLoop(r, n), 24)
		b := circLow(a, 12)
		m = make([]float64, n)
		for i := range m {
			t := float64(i) / float64(n)
			env := 0.25 + 0.75*math.Pow(0.5+0.5*math.Sin(2*math.Pi*(2*t)-1.2), 2)
			m[i] = (a[i]*0.5 + b[i]*1.4) * env
		}
		normalize(m, 0.8)
	case "wind": // low whoosh with a slow gust
		a := circLow(whiteLoop(r, n), 90)
		m = make([]float64, n)
		for i := range m {
			t := float64(i) / float64(n)
			m[i] = a[i] * (0.35 + 0.65*(0.5+0.5*math.Sin(2*math.Pi*t+0.7)))
		}
		normalize(m, 0.8)
	case "crickets": // two kinds of high chirps in short bursts over a very soft hiss
		m = make([]float64, n)
		hiss := circLow(whiteLoop(r, n), 6)
		for i := range m {
			m[i] = hiss[i] * 0.05
		}
		for c := 0; c < 2; c++ {
			f := 4300.0 + float64(c)*520
			for b := 0; b < 9; b++ {
				start := int(sec(0.2 + r.Float64()*(ambLoopSec-1.2)))
				for p := 0; p < 3+r.Intn(3); p++ {
					p0 := start + p*int(sec(0.045))
					for i := 0; i < int(sec(0.022)) && p0+i < n; i++ {
						e := math.Sin(math.Pi * float64(i) / sec(0.022))
						m[p0+i] += math.Sin(2*math.Pi*f*float64(i)/sampleRate) * e * 0.35
					}
				}
			}
		}
		normalize(m, 0.7)
	case "train": // far rumble that swells as it passes, with wheel clacks
		a := circLow(whiteLoop(r, n), 40)
		m = make([]float64, n)
		for i := range m {
			t := float64(i) / float64(n)
			env := math.Pow(math.Sin(math.Pi*t), 2)
			m[i] = a[i] * env * 1.2
			if k := int(float64(i) / sec(0.26)); k >= 0 {
				ph := float64(i) - float64(k)*sec(0.26)
				if ph < sec(0.05) {
					m[i] += math.Sin(2*math.Pi*140*ph/sampleRate) * math.Exp(-ph/sec(0.012)) * 0.25 * env
				}
			}
		}
		normalize(m, 0.7)
	case "crowd": // a murmur: band-limited noise with random swells
		a := circLow(whiteLoop(r, n), 8)
		b := circLow(a, 40)
		m = make([]float64, n)
		for i := range m {
			t := float64(i) / float64(n)
			env := 0.5 + 0.25*math.Sin(2*math.Pi*3*t) + 0.25*math.Sin(2*math.Pi*5*t+1.1)
			m[i] = (a[i] - b[i]) * env
		}
		normalize(m, 0.7)
	case "hum": // the station: two low tones that beat slowly
		m = make([]float64, n)
		for i := range m {
			t := float64(i) / sampleRate
			m[i] = math.Sin(2*math.Pi*55*t)*0.5 + math.Sin(2*math.Pi*82.5*t)*0.3 + math.Sin(2*math.Pi*110*t)*0.1
		}
		normalize(m, 0.6)
	default:
		return nil
	}
	return toPCM(m)
}

// ambientWanted: the loop that should be heard right now ("" = none).
func (g *Game) ambientWanted() string {
	if !ambientSound || g.snd == nil || g.snd.muted || g.mode != ModePlay || g.lvl == nil || g.lvl.Def == nil || g.helpOv || g.trophyOv {
		return ""
	}
	_, kind := ambientFor(g.lvl.Def.Title)
	return kind
}

// updateAmbientSound starts, switches or pauses the ambient loop (called every Update).
func (g *Game) updateAmbientSound() {
	want := g.ambientWanted()
	a := &ambPlay
	if want == "" {
		if a.p != nil && a.p.IsPlaying() {
			a.p.Pause()
		}
		return
	}
	if a.owner != g.snd || a.key != want || a.p == nil {
		if a.p != nil {
			a.p.Pause()
			a.p.Close()
			a.p = nil
		}
		data, ok := g.snd.tracks["amb:"+want]
		if !ok {
			data = buildAmbientLoop(want)
			g.snd.tracks["amb:"+want] = data
		}
		if data == nil {
			a.owner, a.key = g.snd, want
			return
		}
		p, err := g.snd.ctx.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(data), int64(len(data))))
		if err != nil {
			return
		}
		p.SetVolume(ambientVolume)
		a.owner, a.key, a.p = g.snd, want, p
	}
	if !a.p.IsPlaying() {
		a.p.Play()
	}
}
