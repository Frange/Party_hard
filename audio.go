package main

import (
	"bytes"
	"math"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sampleRate = 48000

// Sounds holds every procedurally generated sound effect.
type Sounds struct {
	ctx      *audio.Context
	music    *audio.Player
	cur      string
	tracks   map[string][]byte
	sfx      map[string][]byte
	muted    bool
	file     *os.File
	isFile   bool
	loopFile bool
	paused   bool
	volume   float64
}

func newSounds(muted bool) *Sounds {
	s := &Sounds{ctx: audio.NewContext(sampleRate), sfx: map[string][]byte{}, tracks: map[string][]byte{}, muted: muted, volume: 1}
	s.sfx["stab"] = mix(
		noise(0.18, 0.9, 18),
		tone(90, 0.18, 0.8, 14, 0),
	)
	s.sfx["alert"] = mix(tone(880, 0.08, 0.5, 20, 0), shift(tone(1180, 0.10, 0.5, 20, 0), 0.08))
	s.sfx["ring"] = mix(tone(1200, 0.07, 0.35, 30, 0), shift(tone(1200, 0.07, 0.35, 30, 0), 0.12))
	s.sfx["splash"] = toPCM(lowpass(noise(0.35, 0.7, 7), 6))
	s.sfx["thud"] = mix(tone(70, 0.2, 0.7, 12, 0), lowpass(noise(0.1, 0.3, 25), 4))
	s.sfx["boom"] = mix(lowpass(noise(0.7, 1.0, 4), 3), tone(50, 0.6, 0.9, 4, 0))
	s.sfx["siren"] = siren(2.4)
	s.sfx["win"] = mix(tone(523, 0.15, 0.4, 6, 0), shift(tone(659, 0.15, 0.4, 6, 0), 0.15), shift(tone(784, 0.15, 0.4, 6, 0), 0.30), shift(tone(1046, 0.4, 0.4, 4, 0), 0.45))
	s.sfx["lose"] = mix(tone(300, 0.25, 0.5, 4, 0), shift(tone(250, 0.25, 0.5, 4, 0), 0.25), shift(tone(180, 0.6, 0.5, 3, 0), 0.5))
	s.sfx["spark"] = toPCM(noise(0.05, 0.25, 40))
	s.sfx["zap"] = mix(noise(0.6, 0.9, 5), tone(120, 0.6, 0.5, 4, 2), tone(1900, 0.2, 0.3, 12, 1))
	s.sfx["glug"] = mix(tone(200, 0.08, 0.4, 20, 0), shift(tone(260, 0.08, 0.4, 20, 0), 0.09), shift(tone(320, 0.1, 0.4, 20, 0), 0.18))

	s.SetTrack("menu", musicLoop)
	return s
}

func (s *Sounds) Play(name string) {
	if s.muted {
		return
	}
	b, ok := s.sfx[name]
	if !ok {
		return
	}
	p := s.ctx.NewPlayerFromBytes(b)
	p.SetVolume(s.volume)
	p.Play()
}

// SetTrack switches the background song (building it the first time it is needed).
func (s *Sounds) SetTrack(key string, build func() []byte) {
	if key == s.cur && s.music != nil {
		return
	}
	data, ok := s.tracks[key]
	if !ok {
		data = build()
		s.tracks[key] = data
	}
	if s.music != nil {
		s.music.Pause()
		s.music.Close()
		s.music = nil
	}
	if s.file != nil {
		s.file.Close()
		s.file = nil
	}
	s.isFile = false
	s.cur = key
	loop := audio.NewInfiniteLoop(bytes.NewReader(data), int64(len(data)))
	p, err := s.ctx.NewPlayer(loop)
	if err != nil {
		return
	}
	p.SetVolume(0.5)
	s.music = p
	s.paused = s.muted
	if !s.muted {
		p.Play()
	}
}

func (s *Sounds) StartMusic() {
	if s.muted || s.music == nil || s.music.IsPlaying() {
		return
	}
	s.paused = false
	s.music.Play()
}

func (s *Sounds) StopMusic() {
	s.paused = true
	if s.music != nil && s.music.IsPlaying() {
		s.music.Pause()
	}
}

// --- synthesis helpers (mono float samples -> stereo int16 bytes) --------

func samples(dur float64) []float64 { return make([]float64, int(dur*sampleRate)) }

func tone(freq, dur, vol, decay float64, kind int) []float64 {
	out := samples(dur)
	for i := range out {
		t := float64(i) / sampleRate
		env := math.Exp(-decay * t)
		var v float64
		switch kind {
		case 1: // square
			if math.Sin(2*math.Pi*freq*t) >= 0 {
				v = 1
			} else {
				v = -1
			}
			v *= 0.5
		case 2: // saw-ish
			v = 2*math.Mod(freq*t, 1) - 1
			v *= 0.6
		default:
			v = math.Sin(2 * math.Pi * freq * t)
		}
		out[i] = v * env * vol
	}
	return out
}

func noise(dur, vol, decay float64) []float64 {
	out := samples(dur)
	r := rand.New(rand.NewSource(3))
	for i := range out {
		t := float64(i) / sampleRate
		out[i] = (r.Float64()*2 - 1) * math.Exp(-decay*t) * vol
	}
	return out
}

func lowpass(in []float64, n int) []float64 {
	out := make([]float64, len(in))
	var acc float64
	for i := range in {
		acc += in[i]
		if i >= n {
			acc -= in[i-n]
		}
		out[i] = acc / float64(n)
	}
	return out
}

func shift(in []float64, by float64) []float64 {
	pad := make([]float64, int(by*sampleRate))
	return append(pad, in...)
}

func mix(parts ...[]float64) []byte {
	n := 0
	for _, p := range parts {
		if len(p) > n {
			n = len(p)
		}
	}
	m := make([]float64, n)
	for _, p := range parts {
		for i, v := range p {
			m[i] += v
		}
	}
	return toPCM(m)
}

func toPCM(m []float64) []byte {
	out := make([]byte, len(m)*4)
	for i, v := range m {
		if v > 1 {
			v = 1
		}
		if v < -1 {
			v = -1
		}
		s := int16(v * 32000)
		out[i*4] = byte(s)
		out[i*4+1] = byte(s >> 8)
		out[i*4+2] = byte(s)
		out[i*4+3] = byte(s >> 8)
	}
	return out
}

func siren(dur float64) []byte {
	out := samples(dur)
	for i := range out {
		t := float64(i) / sampleRate
		f := 620.0
		if int(t/0.35)%2 == 1 {
			f = 820
		}
		fade := 1.0
		if t > dur-0.5 {
			fade = (dur - t) / 0.5
		}
		out[i] = math.Sin(2*math.Pi*f*t) * 0.35 * fade
	}
	return toPCM(out)
}

// musicLoop builds an 8-bar electro loop at 128 BPM: kick, hats, bass and a lead.
func musicLoop() []byte {
	bpm := 128.0
	beat := 60 / bpm
	bars := 8
	total := beat * 4 * float64(bars)
	out := samples(total)
	// bass notes per half bar (Hz)
	bassNotes := []float64{55, 55, 65.4, 73.4, 55, 55, 49, 58.3, 55, 55, 65.4, 73.4, 55, 82.4, 73.4, 65.4}
	leadNotes := []float64{0, 440, 0, 523, 0, 392, 440, 0, 0, 440, 0, 587, 0, 523, 440, 0, 0, 440, 0, 523, 0, 392, 349, 0, 0, 440, 494, 523, 587, 523, 494, 440}
	for i := range out {
		t := float64(i) / sampleRate
		beatPos := t / beat
		inBeat := math.Mod(beatPos, 1)
		var v float64
		// kick on every beat
		kt := inBeat * beat
		v += math.Sin(2*math.Pi*(50+120*math.Exp(-kt*30))*kt) * math.Exp(-kt*10) * 0.9
		// hi-hat on off beats
		ht := math.Mod(beatPos+0.5, 1) * beat
		v += (math.Mod(float64(i)*7919.0/13.0, 2) - 1) * math.Exp(-ht*60) * 0.12
		// snare on 2 and 4
		if int(beatPos)%2 == 1 {
			v += (math.Mod(float64(i)*104729.0/17.0, 2) - 1) * math.Exp(-kt*25) * 0.25
		}
		// bass
		bi := int(beatPos/2) % len(bassNotes)
		bt := math.Mod(beatPos, 2) * beat
		bf := bassNotes[bi]
		sq := 0.0
		if math.Sin(2*math.Pi*bf*t) > 0 {
			sq = 1
		} else {
			sq = -1
		}
		v += (sq*0.5 + math.Sin(2*math.Pi*bf*t)*0.5) * 0.28 * math.Min(1, 1.2-bt/(2*beat)*0.6)
		// lead (every half beat)
		li := int(beatPos*2) % len(leadNotes)
		lf := leadNotes[li]
		if lf > 0 {
			lt := math.Mod(beatPos*2, 1) * beat / 2
			v += (2*math.Mod(lf*t, 1) - 1) * 0.10 * math.Exp(-lt*6)
		}
		out[i] = v * 0.8
	}
	return toPCM(out)
}
