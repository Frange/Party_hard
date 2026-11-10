package main

import (
	"math"
	"sort"
	"testing"
)

// pcmStats: peak, RMS and the RMS of the 300 Hz..4 kHz band (where the hook lives) of the left channel.
func pcmStats(l []float64) (peak, rms, band float64) {
	a, b := lowpass(l, 12), lowpass(l, 120)
	for i, v := range l {
		peak = math.Max(peak, math.Abs(v))
		rms += v * v
		d := a[i] - b[i]
		band += d * d
	}
	n := float64(len(l))
	return peak, math.Sqrt(rms / n), math.Sqrt(band / n)
}

// checkLoopClean: no clipping, no DC, no click where the loop comes round (as TestTunesAreClean).
func checkLoopClean(t *testing.T, name string, l []float64) {
	t.Helper()
	peak, mean := 0.0, 0.0
	for _, v := range l {
		peak = math.Max(peak, math.Abs(v))
		mean += v
	}
	mean /= float64(len(l))
	if peak >= 0.999*32000/32768 {
		t.Errorf("%s: peak %.3f is at the limit (clipping)", name, peak)
	}
	if math.Abs(mean) > 0.003 {
		t.Errorf("%s: DC offset %.4f", name, mean)
	}
	diffs := make([]float64, 0, len(l))
	for i := 1; i < len(l); i++ {
		diffs = append(diffs, math.Abs(l[i]-l[i-1]))
	}
	sort.Float64s(diffs)
	p999 := diffs[len(diffs)*999/1000]
	seam := math.Abs(l[0] - l[len(l)-1])
	if seam > 2*p999+0.01 {
		t.Errorf("%s: a click of %.3f where the loop comes round (the music jumps at most %.3f)", name, seam, p999)
	}
	t.Logf("%s: peak %.3f, seam %.4f (p99.9 step %.4f)", name, peak, seam, p999)
}

// onsetMax: the biggest rise of the RMS from one 10 ms block to the next (how hard the strongest attack hits).
func onsetMax(l []float64) float64 {
	const blk = sampleRate / 100
	prev, best := 0.0, 0.0
	for i := 0; i+blk <= len(l); i += blk {
		e := 0.0
		for _, v := range l[i : i+blk] {
			e += v * v
		}
		r := math.Sqrt(e / blk)
		best = math.Max(best, r-prev)
		prev = r
	}
	return best
}

// the synthesised party of the first intro: louder than before, growing to a ceiling, never clipping
func TestIntroMusicLouderWithRamp(t *testing.T) {
	oldL, newL := thumpLeft(buildThumpWith(0.42, 22)), thumpLeft(buildThump())
	op, or, ob := pcmStats(oldL)
	np, nr, nb := pcmStats(newL)
	t.Logf("before: peak %.3f rms %.4f hook band %.4f (player gain 0.7)", op, or, ob)
	t.Logf("after:  peak %.3f rms %.4f hook band %.4f (player gain %.2f)", np, nr, nb, thumpGain)
	checkLoopClean(t, "thump", newL)
	if nb < ob*1.05 {
		t.Errorf("the hook did not come through the wall: band rms %.4f vs %.4f", nb, ob)
	}
	prev := -1.0
	for tt := 0.0; tt < introLen; tt += 0.05 {
		v := introMusicVol(tt)
		if v < 0 || v > introMusicCap+1e-9 {
			t.Fatalf("level %.3f at %.2f s", v, tt)
		}
		if tt < inBurst-0.05 {
			if v < prev-1e-9 {
				t.Fatalf("the level falls at %.2f s before the cloud pops", tt)
			}
			prev = v
		}
		if old := introThumpVol(tt) * 0.7; old > 0.05 && v*thumpGain < old*1.2-1e-9 && v < introMusicCap {
			t.Fatalf("at %.2f s the player volume %.3f is not louder than before (%.3f)", tt, v*thumpGain, old)
		}
	}
	// what reaches the speakers: level x gain x the loop, measured on the hook band
	for _, tt := range []float64{0, inBurst / 2, inBurst - 0.1, inWalk1 + 1} {
		before := introThumpVol(tt) * 0.7 * ob
		after := introMusicVol(tt) * thumpGain * nb
		t.Logf("t=%5.2f s: hook band at the speakers %.4f -> %.4f (x%.2f)", tt, before, after, after/before)
		if after < before*1.35 {
			t.Errorf("t=%.2f s: the hook is not clearly louder (x%.2f)", tt, after/before)
		}
	}
	if peakOut := introMusicCap * thumpGain * np; peakOut > 0.85 {
		t.Errorf("at the ceiling the party peaks at %.2f: it would cover the other sounds", peakOut)
	}
	// the story is the same: it drops when the cloud pops and is gone at the title card
	if introMusicVol(inBurst+0.2) > 0.3 || introMusicVol(inCard) != 0 {
		t.Error("the intro lost the drop of the cloud or the closed door")
	}
}

// the anthem of the title and the menu: the chorus of the intro, repeated (TestAnthemIsTheChorusRepeated), long, seamless
func TestAnthemLongAndSeamless(t *testing.T) {
	pcm := ambientPCM(ambAnthem)
	l := thumpLeft(pcm)
	n := anthemBars * 4 * thumpBeatSamples
	if len(l) != n {
		t.Fatalf("anthem length %d, want %d", len(l), n)
	}
	secs := float64(n) / sampleRate
	t.Logf("anthem: %d bars at 124 BPM = %.1f s (the intro loop is %.1f s)", anthemBars, secs, float64(thumpBeats*thumpBeatSamples)/sampleRate)
	if secs < 60 {
		t.Fatalf("the anthem lasts only %.1f s", secs)
	}
	checkLoopClean(t, "anthem", l)
	_, rms, _ := pcmStats(l)
	if rms < 0.08 {
		t.Errorf("the anthem is nearly silent (rms %.3f)", rms)
	}
	// the dip (bar anthemDipBar) drops the kick, the chorus has it
	bar := 4 * thumpBeatSamples
	hookSub := subBassShare(toPCM(l[0 : 4*bar]))
	dipSub := subBassShare(toPCM(l[anthemDipBar*bar+thumpBeatSamples/2 : (anthemDipBar+1)*bar]))
	t.Logf("sub-bass share: chorus %.0f%%, dip %.0f%%", hookSub*100, dipSub*100)
	if dipSub > hookSub/3 {
		t.Error("the dip should drop the kick")
	}
	if parseNotes(partyHook)[0].midi[0] != 69 {
		t.Error("the hook changed")
	}
}

// the gold menu: a slow, quiet lullaby with no drum and no hard attacks
func TestSleepMusicIsCalm(t *testing.T) {
	if sleepSong.bpm < 60 || sleepSong.bpm > 70 {
		t.Fatalf("the lullaby is at %.0f BPM", sleepSong.bpm)
	}
	sl := thumpLeft(ambientPCM(ambSleep))
	al := thumpLeft(ambientPCM(ambAnthem))
	checkLoopClean(t, "sleep", sl)
	_, sr, _ := pcmStats(sl)
	_, ar, _ := pcmStats(al)
	so, ao := onsetMax(sl), onsetMax(al)
	t.Logf("rms: lullaby %.4f, anthem %.4f; strongest attack (10 ms rms rise): lullaby %.4f, anthem %.4f; %.0f vs 124 BPM",
		sr, ar, so, ao, sleepSong.bpm)
	if sr > ar*0.5 {
		t.Errorf("the lullaby is not quiet: rms %.4f vs %.4f", sr, ar)
	}
	if so > ao/3 {
		t.Errorf("the lullaby has hard attacks: %.4f vs %.4f", so, ao)
	}
	if s := subBassShare(ambientPCM(ambSleep)); s > 0.04 {
		t.Errorf("the lullaby has a drum (%.0f%% sub-bass)", s*100)
	}
	if secs := float64(len(sl)) / sampleRate; secs < 20 {
		t.Errorf("the lullaby loop lasts %.1f s", secs)
	}
}

// which track the menu asks for, per tier
func TestMenuMusicForTier(t *testing.T) {
	want := map[int][2]string{
		tierNormal: {menuKeyAnthem, ambAnthem}, tierBronze: {menuKeyCalm, ambCalm},
		tierSilver: {menuKeyCalmer, ambCalmer}, tierGold: {menuKeySleep, ambSleep},
	}
	for tier := tierNormal; tier <= tierGold; tier++ {
		k, kind := menuMusicFor(tier)
		if k != want[tier][0] || kind != want[tier][1] {
			t.Errorf("tier %d plays %q (%s)", tier, k, kind)
		}
	}
}

// the menu calms down step by step: normal > bronze > silver > gold (the lullaby), in loudness and in attacks
func TestMenuMusicCalmsByTier(t *testing.T) {
	var prevR, prevO float64
	for tier := tierNormal; tier <= tierGold; tier++ {
		_, kind := menuMusicFor(tier)
		l := thumpLeft(ambientPCM(kind))
		checkLoopClean(t, kind, l)
		_, r, _ := pcmStats(l)
		o := onsetMax(l)
		t.Logf("tier %d (%s): %.1f s, rms %.4f, strongest attack %.4f", tier, kind, float64(len(l))/sampleRate, r, o)
		if tier > tierNormal && (r >= prevR*0.92 || o >= prevO*0.92) {
			t.Errorf("tier %d is not calmer than tier %d (rms %.4f vs %.4f, attack %.4f vs %.4f)", tier, tier-1, r, prevR, o, prevO)
		}
		prevR, prevO = r, o
	}
	// the calm versions are the same song: same length, the same hook at the start
	if a, b := len(ambientPCM(ambAnthem)), len(ambientPCM(ambCalmer)); a != b {
		t.Errorf("the calm anthem has another length (%d vs %d)", b, a)
	}
}

// the crossfade: the old song fades out, the new one comes in and fades up; a stopped music stays stopped
func TestMenuCrossfade(t *testing.T) {
	ambientPCM(ambAnthem)
	ambientPCM(ambSleep)
	ambientPCM(ambCalmer)
	s := newSounds(false)
	defer s.StopMusic()
	const dt = 1.0 / 60
	s.syncMenuMusic(tierNormal, dt) // the first song of the run: the anthem, fading in
	if s.cur != menuKeyAnthem || s.music == nil || !s.music.IsPlaying() || s.fadeGain() > 0.01 {
		t.Fatalf("the menu starts with %q (gain %.2f)", s.cur, s.fadeGain())
	}
	for i := 0; i < int(menuFadeSecs/dt)+2; i++ {
		s.syncMenuMusic(tierNormal, dt)
	}
	if s.cur != menuKeyAnthem || s.fadeGain() < 0.999 {
		t.Fatalf("the anthem did not reach full volume (%q, %.2f)", s.cur, s.fadeGain())
	}
	frames := 0
	for s.cur != menuKeySleep && frames < 600 {
		s.syncMenuMusic(tierGold, dt)
		frames++
		if s.cur == menuKeyAnthem && frames > 10 && s.fadeGain() > 0.99 {
			t.Fatal("the anthem does not fade out")
		}
	}
	if s.cur != menuKeySleep {
		t.Fatal("never switched to the lullaby")
	}
	if secs := float64(frames) * dt; secs < menuFadeSecs*0.9 {
		t.Fatalf("the switch took %.2f s: it is a cut, not a fade", secs)
	}
	if s.fadeGain() > 0.01 {
		t.Fatal("the lullaby should start silent")
	}
	for i := 0; i < int(menuFadeSecs/dt)+2; i++ {
		s.syncMenuMusic(tierGold, dt)
	}
	if s.fadeGain() < 0.999 || !s.music.IsPlaying() {
		t.Fatalf("the lullaby did not fade in (%.2f)", s.fadeGain())
	}
	// back to silver: the calmer anthem comes in with a fade too
	frames = 0
	for s.cur != menuKeyCalmer && frames < 600 {
		s.syncMenuMusic(tierSilver, dt)
		frames++
	}
	if s.cur != menuKeyCalmer || float64(frames) < menuFadeSecs/dt*0.9 {
		t.Fatalf("the anthem did not come back with a fade (%d frames)", frames)
	}
	// stopped music (e.g. the title or the editor): it switches at once and stays silent
	s.StopMusic()
	s.syncMenuMusic(tierGold, dt)
	if s.cur != menuKeySleep || s.music.IsPlaying() || s.fadeGain() != 1 {
		t.Fatal("a stopped music must switch silently")
	}
	s.ClearFade()
	if s.fadeGain() != 1 {
		t.Fatal("ClearFade")
	}
}
