package main

import (
	"fmt"
	"math"
	"time"
)

// ---------------------------------------------------------------------------
// THE MUSIC OF THE FIRST SCREEN AND THE MAIN MENU.
//
//   - The anthem of the party next door (ambAnthem): the hook of the first intro (partyHook, its kick and
//     its bass, at the same 124 BPM) and NOTHING invented around it: the song is made longer by repeating
//     that chorus in the arrangements it already had (alone, with the octave below, with the marimba
//     counter-melody and the pad, and once "through the wall" as in the intro). One bar after the first
//     chorus is the dip (anthemDipBar: no drums, no bass, the pad holds G and a snare roll brings the drop):
//     the title fades to black on it and the menu goes on with the same player, without a cut. 33 bars
//     (about 64 s), one seamless loop (every tail wraps round to the start).
//   - The lullaby (ambSleep): when the menu is gold (everything done, menuTier() == tierGold) the
//     party is over and he sleeps in peace: a slow pad and a soft music box, 66 BPM, no drum, quiet.
//
// The title starts the menu's own track (StartMenuMusicNow) so the menu never switches songs when it
// opens; the menu crossfades (Sounds.FadeToTrack) only when its tier changes. Songs of the intros/ folder
// never play here: the menu music is all synthesised.
// ---------------------------------------------------------------------------

const (
	ambAnthem = "anthem"
	ambSleep  = "sleep"

	anthemPeak = 0.85 // the peak of the loop (below 0 dBFS)
	sleepPeak  = 0.28 // the lullaby is quiet on purpose

	menuKeyAnthem = "menu"        // the track key of the anthem in the menu
	menuKeySleep  = "menu:sleep"  // and of the lullaby
	menuKeyCalm   = "menu:calm"   // the calm anthem (bronze)
	menuKeyCalmer = "menu:calmer" // the calmer anthem (silver)
	ambCalm       = "anthemCalm"
	ambCalmer     = "anthemCalmer"
	menuFadeSecs  = 1.2 // how long each half of a crossfade lasts

	anthemBarSamples = 4 * thumpBeatSamples // one bar of the anthem (1.9375 s)
	anthemDipBar     = 4                    // the dip comes right after the first chorus (bars 0-3)
	// anthemDipAt: when the dip starts (s from the first sample): the title fades to black on it (title.go)
	anthemDipAt = float64(anthemDipBar*anthemBarSamples) / float64(sampleRate)
	// the level of the parts in the dip (against the mix of the chorus)
	dipPadGain  = 1.1
	dipHatGain  = 0.7
	dipRollFrom = 0.04
	dipRollTo   = 0.16
)

// the sections of the anthem: every one but the dip is the 4-bar chorus of the intro (partyHook over A A C G)
const (
	secHook       = iota // the chorus as it opens the song (and as in the intro): kick, bass, hats, square lead
	secHookOct           // the same with the lead doubled an octave below (the "second time")
	secHookCount         // the same with the marimba counter-melody and the pad
	secHookCountO        // counter-melody, pad and octave: the fullest one
	secHookWall          // the chorus heard through the wall, as in the first intro (the lead low-passed): a breather
	secDip               // ONE bar: no drums, no bass, the pad on G and a snare roll up to the drop
)

// anthemForm: the order of the sections (the loop goes back to the first one).
var anthemForm = []int{
	secHook, secDip, // 0-3, 4: the title (its fade to black falls on the dip)
	secHookOct, secHookCount, secHookCountO, // 5-16: the menu takes over with the drop
	secHookWall,                             // 17-20
	secHookOct, secHookCount, secHookCountO, // 21-32
}

func anthemSecBars(kind int) int {
	if kind == secDip {
		return 1
	}
	return 4
}

// anthemBars: how many bars the loop has (33).
var anthemBars = func() int {
	n := 0
	for _, k := range anthemForm {
		n += anthemSecBars(k)
	}
	return n
}()

// anthemLayout: the first bar of every section of anthemForm.
func anthemLayout() []int {
	out := make([]int, len(anthemForm))
	b := 0
	for i, k := range anthemForm {
		out[i] = b
		b += anthemSecBars(k)
	}
	return out
}

// the parts written for the chorus (the same as before: only the order of the sections is new)
var (
	partyRoots   = []int{33, 33, 36, 31} // the bass of the chorus: A A C G (partyBassMidi an octave up is the same line)
	anthemPadA   = noteLine("[A3,C4,E4]:4", "[A3,C4,E4]:4", "[G3,C4,E4]:4", "[G3,B3,D4]:4")
	anthemDipPad = "[G3,B3,D4]:4" // the dip holds the last chord of the chorus: the drop resolves it on A
	// the marimba counter-melody under the chorus
	anthemCounter = noteLine(ostinato("A3", "C4", "E4"), ostinato("A3", "C4", "E4"), ostinato("C4", "E4", "G4"), ostinato("G3", "B3", "D4"))
)

// partyKick: the bass drum of the party (the same one as the wall loop of the first intro).
func partyKick() []float64 {
	kick := samples(0.34)
	for i := range kick {
		t := float64(i) / sampleRate
		kick[i] = math.Sin(2*math.Pi*(46+120*math.Exp(-t*40))*t) * math.Exp(-t*14) * 0.95
	}
	return kick
}

// renderAnthem plays the anthem into float samples (one loop of anthemBars bars, the tails wrap round).
func renderAnthem() []float64 { return renderAnthemMix(anthemFull) }

// anthemMix: how hard each part of the anthem plays. The menu plays it calmer as the progress grows
// (menuTier): the same song and the same hook, with a softer kick, less snare and counter-melody,
// more pad and a lower peak; gold has the lullaby instead.
type anthemMix struct {
	kick, snare, hat, lead, counter, pad, peak float64
	roll                                       bool // the snare roll of the dip
}

var (
	anthemFull   = anthemMix{kick: 0.8, snare: 0.32, hat: 0.5, lead: 1, counter: 0.2, pad: 0.1, peak: anthemPeak, roll: true}
	anthemCalm   = anthemMix{kick: 0.45, snare: 0.14, hat: 0.3, lead: 0.85, counter: 0.12, pad: 0.14, peak: 0.62} // bronze: the game is finished
	anthemCalmer = anthemMix{kick: 0.18, snare: 0, hat: 0.14, lead: 0.7, counter: 0.06, pad: 0.18, peak: 0.5}     // silver: and every trophy
)

// renderAnthemMix renders the whole anthem with the given mix.
func renderAnthemMix(mx anthemMix) []float64 {
	spb := thumpBeatSamples // the very beat of the first intro: whole samples per beat, so the loop has no seam
	barS := anthemBarSamples
	n := anthemBars * barS
	low := make([]float64, n)  // kick and bass
	lead := make([]float64, n) // the melodies (with echo)
	wall := make([]float64, n) // the chorus through the wall (low-passed, then echo with the lead)
	bed := make([]float64, n)  // pad, counter-melody, hats, snare
	add := func(dst []float64, at int, p []float64, gain float64) {
		if at >= n || at < 0 {
			return
		}
		for len(p) > 0 { // the tails wrap round to the start
			k := min(len(p), n-at)
			d := dst[at : at+k]
			for i, v := range p[:k] {
				d[i] += v * gain
			}
			p, at = p[k:], 0
		}
	}
	type noteKey struct {
		inst      string
		m         int
		secs, vol float64
	}
	cache := map[noteKey][]float64{} // the same note again (the chorus eight times, the pads): rendered once
	play := func(dst []float64, inst, text string, bar int, vol float64) {
		in := instruments[inst]
		for _, nt := range parseNotes(text) {
			at := bar*barS + int(math.Round(nt.at*float64(spb)))
			if at >= n {
				break
			}
			secs := nt.dur * float64(spb) / sampleRate
			for _, m := range nt.midi {
				k := noteKey{inst, m, secs, vol}
				p, ok := cache[k]
				if !ok {
					p = in.play(midiFreq(m), secs, vol)
					cache[k] = p
				}
				add(dst, at, p, 1)
			}
		}
	}
	kick := partyKick()
	hat := noise(0.05, 0.5, 70)
	hatLo := lowpass(hat, 6)
	for i := range hat { // only the hiss: take the low part away
		hat[i] -= hatLo[i]
	}
	snare := percSample('r')
	bassCache := map[int][]float64{}
	bassOf := func(root int) []float64 {
		b, ok := bassCache[root]
		if !ok {
			f := midiFreq(root)
			b = mix2(tone(f, 0.24, 0.9, 8, 0), tone(f*2, 0.2, 0.25, 10, 0))
			bassCache[root] = b
		}
		return b
	}
	// the drums and the bass of one chorus bar (the same as the loop of the intro)
	groove := func(bar, root int, hatGain float64) {
		for beat := 0; beat < 4; beat++ {
			at := bar*barS + beat*spb
			off := at + spb/2
			add(low, at, kick, mx.kick)
			add(bed, off, hat, hatGain)
			if beat%2 == 1 {
				add(bed, at, snare, mx.snare)
			}
			add(low, off, bassOf(root), 0.42)
		}
	}
	for si, b0 := range anthemLayout() {
		kind := anthemForm[si]
		if kind == secDip {
			// the dip: after one last hit the kick, the snare and the bass stop; the hats keep the pulse, the pad holds the last
			// chord of the chorus and its echo rings out; a soft snare roll over the last two beats brings the
			// drop of the next chorus
			add(low, b0*barS, kick, mx.kick*0.6) // the last hit on the downbeat, then the drums are gone
			add(low, b0*barS, bassOf(partyRoots[3]), 0.42)
			for _, m := range parseNotes(anthemDipPad)[0].midi { // it swells in under the last beat of the chorus (G there too)
				add(bed, b0*barS-spb, instruments["softpad"].play(midiFreq(m), 5*float64(spb)/sampleRate, mx.pad*dipPadGain), 1)
			}
			for beat := 0; beat < 4; beat++ {
				add(bed, b0*barS+beat*spb+spb/2, hat, mx.hat*dipHatGain)
			}
			if mx.roll {
				for s := 0; s < 8; s++ {
					add(bed, b0*barS+2*spb+s*spb/4, snare, dipRollFrom+(dipRollTo-dipRollFrom)*float64(s)/7)
				}
			}
			continue
		}
		hatGain := mx.hat
		if kind == secHookWall {
			hatGain *= 0.5
		}
		for k := 0; k < 4; k++ {
			groove(b0+k, partyRoots[k], hatGain)
		}
		switch kind {
		case secHookWall:
			play(wall, "square", partyHook, b0, 0.6*mx.lead) // the gain of the hook in the intro's wall loop
		default:
			play(lead, "square", partyHook, b0, 0.42*mx.lead)
		}
		if kind == secHookOct || kind == secHookCountO {
			play(lead, "square", transposeText(partyHook, -12), b0, 0.16*mx.lead)
		}
		if kind == secHookCount || kind == secHookCountO {
			play(bed, "marimba", anthemCounter, b0, mx.counter)
			play(bed, "softpad", anthemPadA, b0, mx.pad)
		}
	}
	wall = lowpass(wall, 22) // the wall of the original intro loop (buildThump before it was brightened)
	for i := range lead {
		lead[i] += wall[i]
	}
	lead = echoLoop(lead, 0.75*float64(spb)/sampleRate, 0.28, 0.25)
	low = lowpass(low, 3)
	out := make([]float64, n)
	for i := range out {
		out[i] = low[i] + lead[i] + bed[i]
	}
	return out
}

// buildAnthem: the anthem as a loop of PCM (no DC offset, peak at anthemPeak).
func buildAnthem() []byte { return finishSong(renderAnthem(), anthemPeak) }

// ---- the lullaby -------------------------------------------------------------------------------------

func init() {
	// a soft pad ("softpad": its own name, "pad" belongs to genres.go and the level songs): two slightly detuned sines, a slow swell in and a long release
	instruments["softpad"] = instrument{sustain: true, play: func(f, dur, vol float64) []float64 {
		n := int((dur + 0.9) * sampleRate)
		out := make([]float64, n)
		a, b, c := newOsc(f*1.002), newOsc(f*0.998), newOsc(f*2)
		for i := range out {
			t := float64(i) / sampleRate
			v := 0.5*a.next() + 0.5*b.next() + 0.1*c.next()
			out[i] = v * envAR(t, dur+0.8, 0.7, 0.9) * vol
		}
		return out
	}}
	// a music box with a soft touch (no click): a slower attack and fewer overtones than "box"
	instruments["softbox"] = instrument{play: func(f, dur, vol float64) []float64 {
		n := int(2.2 * sampleRate)
		out := make([]float64, n)
		a, b := newOsc(f), newOsc(f*2)
		ea, eb := 1.0, 1.0
		da, db := math.Exp(-1.6/sampleRate), math.Exp(-4.0/sampleRate)
		for i := range out {
			t := float64(i) / sampleRate
			out[i] = (a.next()*ea + 0.15*b.next()*eb) * smooth01(t/0.025) * vol
			ea, eb = ea*da, eb*db
		}
		return out
	}}
}

// osc: a sine by recurrence (two multiplies a sample instead of a math.Sin): the long notes of the pads
// would otherwise take seconds to render.
type osc struct{ k, y1, y2 float64 }

func newOsc(f float64) *osc {
	w := 2 * math.Pi * f / sampleRate
	return &osc{k: 2 * math.Cos(w), y1: 0, y2: -math.Sin(w)} // y(0) = sin 0, y(-1) = sin(-w)
}

// next returns sin(w*i) for i = 0, 1, 2...
func (o *osc) next() float64 {
	y := o.y1
	o.y1, o.y2 = o.k*o.y1-o.y2, y
	return y
}

var sleepMel = noteLine(
	"A5:2 C6:1 A5:1", "G5:3 R:1", "F5:1 G5:1 A5:1 C6:1", "A5:4",
	"Bb5:2 A5:1 G5:1", "A5:2 F5:2", "G5:1 A5:1 G5:1 E5:1", "F5:4",
)

var sleepPadChords = noteLine(
	"[F3,A3,C4]:4", "[E3,G3,C4]:4", "[F3,A3,D4]:4", "[F3,A3,C4]:4",
	"[F3,Bb3,D4]:4", "[F3,A3,C4]:4", "[E3,G3,Bb3]:4", "[F3,A3,C4]:4",
)

var sleepLow = noteLine("F2:4", "C3:4", "D3:4", "F2:4", "Bb2:4", "F2:4", "C3:4", "F2:4")

// sleepSong: the lullaby of the gold menu (kept out of `songs`: those are the lively scene tunes).
var sleepSong = songDef{name: "sleep", bpm: 66, bars: 8, scale: scFMajor, echo: 1, fb: 0.4, wet: 0.35,
	tracks: []trackDef{{"softbox", sleepMel, 0.32}, {"softpad", sleepPadChords, 0.16}, {"softpad", sleepLow, 0.2}}}

func buildSleep() []byte { return finishSong(composeSong(sleepSong), sleepPeak) }

// ---- which one the menu plays -----------------------------------------------------------------------

// menuMusicFor: the track of the main menu for a menu tier (menutier.go) and the ambient kind that holds
// it: the anthem (normal), calm (bronze), calmer (silver) and the lullaby (gold).
func menuMusicFor(tier int) (key, kind string) {
	switch tier {
	case tierGold:
		return menuKeySleep, ambSleep
	case tierSilver:
		return menuKeyCalmer, ambCalmer
	case tierBronze:
		return menuKeyCalm, ambCalm
	}
	return menuKeyAnthem, ambAnthem
}

// syncMenuMusic: the menu asks for its track every frame; it switches (with a crossfade) only once the
// track has been built by the background thread, so the menu never waits for it.
func (s *Sounds) syncMenuMusic(tier int, dt float64) {
	key, kind := menuMusicFor(tier)
	if key == s.cur && s.music != nil {
		s.FadeToTrack(key, nil, dt)
		return
	}
	if pcm := ambientReady(kind); pcm != nil {
		s.FadeToTrack(key, func() []byte { return pcm }, dt)
		return
	}
	ambientWarmOne(kind) // not built yet (the calm versions are built only when a tier asks for them)
}

// StartMenuMusicNow starts the menu's own track for a tier from its first bar, fading in over menuFadeSecs
// (syncMenuMusic, every frame, brings it up). The first intro calls it when it leaves you in the menu (the
// anthem from its start). It waits for the track if the background thread is still building it, so the title
// (the very first frames of the program) uses StartMenuMusicWhenReady instead.
func (s *Sounds) StartMenuMusicNow(tier int) {
	_, kind := menuMusicFor(tier)
	s.startMenuMusicAt(tier, ambientPCM(kind), 0)
}

// StartMenuMusicWhenReady is what the title does every frame until it reports true: it never waits for the
// background thread. While the track is not built it only asks for it; once it is there it starts it as
// StartMenuMusicNow does, but `at` seconds into the song (the time the title has been on screen), so the
// dip of the anthem still falls on the fade to black (tiEnd) and the menu goes on with the same player.
func (s *Sounds) StartMenuMusicWhenReady(tier int, at float64) bool {
	_, kind := menuMusicFor(tier)
	pcm := ambientReady(kind)
	if pcm == nil {
		ambientWarmOne(kind)
		return false
	}
	s.startMenuMusicAt(tier, pcm, at)
	return true
}

// warmMenuMusic builds the menu's track of a tier in a thread of its own right now (the title asks for it on
// its first frame: it must be there, or nearly).
func warmMenuMusic(tier int) {
	_, kind := menuMusicFor(tier)
	ambientWarmOne(kind)
}

// warmMenuMusicEarly: newGame calls it the moment the save is read, long before the title asks for the
// track (the levels, the album and the shop that menuTier needs are not built yet: the save alone gives a
// good guess). The anthem is the first thing the warm-up thread of newSounds builds; a finished game builds
// its calm version in a thread of its own, and with the legend trophy the silver and gold ones too.
func warmMenuMusicEarly(p Progress) {
	n, won := 0, 0
	for i := range campaignLevels {
		if campaignLevels[i].retired { // the retired parties no longer count (campaign_retired.go)
			continue
		}
		n++
		if p.Stats[fmt.Sprintf("won_%d", i)] > 0 {
			won++
		}
	}
	if n == 0 || won < n {
		return
	}
	if !p.Trophies[legendTrophy] {
		warmMenuMusic(tierBronze)
		return
	}
	warmMenuMusic(tierSilver)
	warmMenuMusic(tierGold)
}

func (s *Sounds) startMenuMusicAt(tier int, pcm []byte, at float64) {
	key, _ := menuMusicFor(tier)
	s.ClearFade()
	s.SetTrack(key, func() []byte { return pcm })
	if s.music == nil {
		return
	}
	pos := time.Duration(0)
	if loop := float64(len(pcm)) / (4 * sampleRate); at > 0 && loop > 0 {
		pos = time.Duration(math.Mod(at, loop) * float64(time.Second))
	}
	s.music.SetPosition(pos)
	s.paused = s.muted
	if !s.muted {
		s.music.Play()
	}
	s.fadeDip = 1
	s.music.SetVolume(0.5 * s.fadeGain())
}

// FadeToTrack switches the background song with a crossfade: the one playing fades out over menuFadeSecs,
// the new one starts and fades in. Call it every frame (dt = the frame time). When the music is stopped or
// muted it switches at once and silently (and stays stopped).
func (s *Sounds) FadeToTrack(key string, build func() []byte, dt float64) {
	if key == s.cur && s.music != nil {
		s.fadePend, s.fadeBuild = "", nil
		s.fadeDip = math.Max(0, s.fadeDip-dt/menuFadeSecs)
		return
	}
	stopped := s.paused || s.muted || s.music == nil || !s.music.IsPlaying()
	if !stopped && s.fadeDip < 1 {
		s.fadePend, s.fadeBuild = key, build
		s.fadeDip = math.Min(1, s.fadeDip+dt/menuFadeSecs)
		return
	}
	wasPaused := s.paused || s.muted
	first := s.music == nil
	s.fadePend, s.fadeBuild = "", nil
	s.SetTrack(key, build)
	switch {
	case wasPaused:
		s.StopMusic() // SetTrack starts the song: a stopped music stays stopped
		s.fadeDip = 0
	case first:
		s.fadeDip = 1 // the very first song of the run fades in
	}
}

// ClearFade forgets any crossfade (the music of a party plays at its normal volume).
func (s *Sounds) ClearFade() { s.fadeDip, s.fadePend, s.fadeBuild = 0, "", nil }

// fadeGain: what the crossfade leaves of the volume (1 = none).
func (s *Sounds) fadeGain() float64 { return 1 - smooth01(s.fadeDip) }
