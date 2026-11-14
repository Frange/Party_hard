package main

import (
	"fmt"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Songs: a tiny step sequencer. Every level has its own playlist (house, techno, disco,
// reggaeton...), the menu keeps the original electro theme and when the DJ dies the
// party switches to the songs after the DJ (songs_afterdj.go: funk, big beat, ska, salsa...).
// ---------------------------------------------------------------------------

type songSpec struct {
	style string
	bpm   float64
	root  float64
	scale []int
	seed  int64
	bars  int
	// the new genres (genres.go): the key (semitones over the written one) and which reading of the tune
	shift, variant int
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
	if _, ok := genreSongs[sp.style]; ok {
		return genreSongPCM(sp)
	}
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

// playlistFor returns the track specs for a level title: three songs of the genre of that party (genres.go).
func playlistFor(title string) []songSpec { return genrePlaylist(title) }

// punkPlaylist: the punk songs of the style "punk" (the punk bands); the dead DJ has songs_afterdj.go now.
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

// menuMusicScreen: the screens that hang from the main menu play the menu's own track (the same player:
// opening the shop, the trophies, the settings, the party list, the scenes, a confirmation or DEBUG never
// cuts or restarts it). g.lvl is never nil (the first party is loaded behind the menu), so without this
// those screens would start its song. The settings opened from the pause window are not one of them: the
// party is still there and its song goes on.
func (g *Game) menuMusicScreen() bool {
	switch g.mode {
	case ModeMenu, ModeTitle, ModeEditor, ModeLevelSelect, ModeShop, ModeConfirm, ModeTrophies, ModeDebug, ModeScenes:
		return true
	case ModeSettings:
		return !g.pause.inSet
	}
	return g.lvl == nil
}

// syncMusic picks the song that fits the moment: the menu theme, the level playlist or the songs after the DJ dies.
func (g *Game) syncMusic() {
	if g.snd == nil {
		return
	}
	if g.mode == ModeIntro || (g.mode == ModeTitle && (g.title == nil || !g.title.musicOn)) {
		// the intro plays its own party through the wall (intro.go) and keeps the menu music paused: the
		// party of the level loaded behind it (g.lvl is never nil) must not start here, or the menu that
		// follows would open with that song and switch to its own a moment later. The title starts the
		// menu's music on its first frame (updateTitle): nothing before that either.
		return
	}
	if g.mode == ModeInterlude || g.mode == ModeEnding {
		// the cinematics play their own music (scene layers): the song of the level loaded behind them must
		// never start here. Watched again from the scenes list / DEBUG, the menu's song waits paused
		// (leaveScene resumes it where it was).
		if g.sceneReplay {
			g.snd.StopMusic()
		}
		return
	}
	if g.pause.keepSong {
		if g.mode == ModeLevelSelect {
			// SALIR A ELEGIR FIESTA from the pause window: the song of the abandoned party goes on in the list
			// (loadLevel(0) put the first party behind it: its song must not start)
			// NOTE: if that mp3 has ended, SetFile of the same file starts it again from the top (a loop, by
			// design: the list never goes silent). It is not resumed where it was, and nothing else changes.
			if g.snd.FileEnded() && g.songFile != "" {
				g.snd.SetFile(g.songFile, false)
			}
			return
		}
		g.pause.keepSong = false
	}
	if g.menuMusicScreen() {
		g.musicMode = ""
		g.snd.syncMenuMusic(g.menuTier(), 1.0/60) // the anthem, or the lullaby on the gold menu (menumusic.go)
		return
	}
	g.snd.ClearFade()
	if !g.djDead {
		afterDJIdle() // the next dead DJ chooses a song again (songs_afterdj.go)
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
		g.lifeOopsOnChange(true)
	}
	title := g.lvl.Def.Title
	if g.time >= g.nextSong {
		g.songIdx++
		g.nextSong = g.time + 70 + g.rng.Float64()*40
		g.lifeOopsOnChange(false)
		g.lifeSongChanged() // some guests go dancing (partylife.go)
		if g.mode == ModePlay {
			g.say("El DJ cambia de canción")
		}
	}
	var sp songSpec
	if g.djDead {
		// the DJ is dead: one of the songs after the DJ (songs_afterdj.go), never the last one again
		sp, title = g.afterDJSong(), "afterdj"
	} else if g.ownStyleBand() {
		pl := bandPlaylist(g.band.slug) // a famous band that is not punk keeps to its own style
		title = "band-" + g.band.slug
		sp = pl[g.songIdx%len(pl)]
	} else {
		pl := playlistFor(title)
		sp = pl[g.songIdx%len(pl)]
		if g.life.oops { // the DJ got the record wrong: a waltz (partylife.go)
			sp, title = lifeOopsSpec(), "oops"
		}
	}
	if _, isGenre := genreSongs[sp.style]; isGenre {
		// the new genres take about a second to render: a background thread does it and the
		// music starts as soon as the song is there (the previous track plays meanwhile)
		key := trackKey(sp.style, int(sp.seed%1000)*10+sp.variant, title)
		pcm := genreReadyPCM(sp)
		if key != genreLastWarm || pcm == nil { // (while it is not there the thread is asked again: it may have been busy)
			genreLastWarm = key
			genreWarm(append([]songSpec{sp}, genreSiblings(g, title)...))
		}
		if pcm != nil {
			g.snd.SetTrack(key, func() []byte { return pcm })
		}
		return
	}
	g.snd.SetTrack(trackKey(sp.style, int(sp.seed%1000), title), func() []byte { return renderSong(sp) })
}
