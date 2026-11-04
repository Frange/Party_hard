package main

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestSceneFileNumber(t *testing.T) {
	for name, want := range map[string]int{
		"1_intro.mp3": 1, "2_interludio_A.mp3": 2, "2_interludio_A_2.mp3": 2, "3_interludio_B.mp3": 3, "4_final.mp3": 4,
		"5_final_secreto.mp3": 5, "6_invitacion.mp3": 6, "6 invitacion.MP3": 6, "4-final.mp3": 4, "7_cohete.mp3": 7,
		"8_nada.mp3": 0, "0_nada.mp3": 0, "intro.mp3": 0, "12_x.mp3": 0, "1_intro.wav": 0, ".1_hidden.mp3": 0, "1intro.mp3": 0, "1.mp3": 1,
	} {
		if got := sceneFileNumber(name); got != want {
			t.Errorf("%q: scene %d, want %d", name, got, want)
		}
	}
}

func TestLoadSceneSongs(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"1_intro.mp3", "2_interludio_A.mp3", "2_interludio_A_2.mp3", "notas.mp3", "8_x.mp3", "3_x.wav", ".4_hidden.mp3"} {
		os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644)
	}
	got := loadSceneSongs(dir)
	if len(got[1]) != 1 || len(got[2]) != 2 || len(got) != 2 {
		t.Fatalf("songs: %v", got)
	}
}

// no file (or one that cannot be played): the synthesised music takes over
func TestSceneSongFallback(t *testing.T) {
	dir := t.TempDir()
	old := introsDirOverride
	introsDirOverride = dir
	defer func() { introsDirOverride = old }()
	for n := 1; n <= sceneMusicCount; n++ {
		if f := sceneSong(n); f != "" {
			t.Errorf("scene %d: found %q in an empty folder", n, f)
		}
	}
	os.WriteFile(filepath.Join(dir, "2_broken.mp3"), []byte("this is not an mp3"), 0o644)
	if f := sceneSong(sceneInterA); f != "" {
		t.Errorf("a file that cannot be decoded was chosen: %q", f)
	}
	// the first intro without a song keeps the "pum pum" and its own beat
	resetThumpBeat()
	if math.Abs(thumpBeat-float64(thumpBeatSamples)/sampleRate) > 1e-12 {
		t.Error("the beat of the synthesised loop changed")
	}
}

func TestIntroBPM(t *testing.T) {
	dir := t.TempDir()
	old := introsDirOverride
	introsDirOverride = dir
	defer func() { introsDirOverride = old; resetThumpBeat() }()
	if introBPM() != defaultIntroBPM {
		t.Error("no file: the default tempo")
	}
	for text, want := range map[string]float64{"128": 128, " 90.5\n": 90.5, "abc": defaultIntroBPM, "500": defaultIntroBPM, "10": defaultIntroBPM} {
		os.WriteFile(filepath.Join(dir, "1_intro.bpm"), []byte(text), 0o644)
		if got := introBPM(); got != want {
			t.Errorf("%q: %.1f BPM, want %.1f", text, got, want)
		}
	}
	// the picture follows the tempo of the song
	setThumpBPM(120)
	if math.Abs(thumpBeat-0.5) > 1e-12 {
		t.Fatalf("120 BPM is %.4f s a beat", thumpBeat)
	}
	if n, _ := thumpPhase(thumpLatency + 3.2); n != 6 {
		t.Errorf("at 3.2 s with 120 BPM it is beat %d, want 6", n)
	}
	if k := thumpKick(thumpLatency + 2.0 + 1e-6); k < 0.99 {
		t.Errorf("no pulse on the beat at 120 BPM (%.2f)", k)
	}
}

func allTunes() map[string][]byte {
	out := map[string][]byte{}
	for _, k := range ambientKinds {
		out[k] = ambientPCM(k)
	}
	out["thump (first intro)"] = buildThump()
	return out
}

// subBassShare: the share of the energy that is below ~80 Hz. A bass drum lives there; the plucked bass of
// the tunes stays above ~90 Hz.
func subBassShare(pcm []byte) float64 {
	l := thumpLeft(pcm)
	low := lowpass(l, 600)
	var eLow, eAll float64
	for i := range l {
		eLow += low[i] * low[i]
		eAll += l[i] * l[i]
	}
	return eLow / eAll
}

// every tune is written with whole bars, the right notes, a lively tempo and regular rhythm
func TestSongsAreWellWritten(t *testing.T) {
	for kind, sd := range songs {
		beats := float64(sd.bars * 4)
		if sd.bpm < 110 || sd.bpm > 140 {
			if kind != ambEndCalm { // the calm end of the lullaby is the only slow one
				t.Errorf("%s: %.0f BPM is not a lively tempo", kind, sd.bpm)
			}
		}
		for _, tr := range sd.tracks {
			notes := parseNotes(tr.notes)
			end := 0.0
			for _, nt := range notes {
				if math.Abs(math.Mod(nt.at*4, 1)) > 1e-9 {
					t.Errorf("%s: a note at beat %.3f is off the sixteenth-note grid", kind, nt.at)
				}
				end = nt.at + nt.dur
				for _, m := range nt.midi {
					if !sd.scale.has(m) {
						t.Errorf("%s: %s is not in %s", kind, midiName(m), sd.scale.name)
					}
					if f, ok := freqMidi(midiFreq(m)); !ok || f != m {
						t.Errorf("%s: %s is not a tempered note", kind, midiName(m))
					}
					if tr.inst == "pizz" && midiFreq(m) < 90 {
						t.Errorf("%s: the plucked bass goes down to %.0f Hz: that is a kick's place", kind, midiFreq(m))
					}
				}
			}
			if math.Abs(end-beats) > 1e-9 {
				t.Errorf("%s: a %s line lasts %.2f beats, the song %.0f", kind, tr.inst, end, beats)
			}
		}
		for _, pc := range sd.perc {
			if len(pc.pattern) != 16 || strings.ContainsAny(strings.ReplaceAll(pc.pattern, ".", ""), "kKbB") || strings.Trim(pc.pattern, ".whsr") != "" {
				t.Errorf("%s: strange percussion %q (only woodblock, hat, shaker, rim)", kind, pc.pattern)
			}
		}
	}
	// the tense memories play over the same bars as the march, so the music can cut between them in time
	for _, pair := range [][2]string{{ambInterA, ambMemA}, {ambInterB, ambMemB}} {
		a, _ := songs[pair[0]].loopSamples()
		b, _ := songs[pair[1]].loopSamples()
		if a != b || songs[pair[0]].bpm != songs[pair[1]].bpm {
			t.Errorf("%s and %s do not share tempo and length", pair[0], pair[1])
		}
	}
}

// the hook of the party next door (first intro) and its bass are in A minor, over whole bars of the 124 BPM beat
func TestPartyHookInScale(t *testing.T) {
	end := 0.0
	for _, nt := range parseNotes(partyHook) {
		end = nt.at + nt.dur
		for _, m := range nt.midi {
			if !scAMinor.has(m) {
				t.Errorf("the hook has %s, not in A minor", midiName(m))
			}
		}
	}
	if end != thumpBeats {
		t.Errorf("the hook lasts %.1f beats, the loop %d", end, thumpBeats)
	}
	for _, m := range partyBassMidi {
		if !scAMinor.has(m) {
			t.Errorf("the bass has %s, not in A minor", midiName(m))
		}
	}
}

// what can be checked without listening: no clipping, no DC offset, no click where a loop comes round, no drum
func TestTunesAreClean(t *testing.T) {
	for name, pcm := range allTunes() {
		l := thumpLeft(pcm)
		if len(l) < sampleRate*6 {
			t.Fatalf("%s: only %d samples", name, len(l))
		}
		peak, mean := 0.0, 0.0
		clipped := 0
		for _, v := range l {
			peak = math.Max(peak, math.Abs(v))
			mean += v
			if math.Abs(v) >= 0.999*32000/32768 {
				clipped++
			}
		}
		mean /= float64(len(l))
		if peak < 0.3 {
			t.Errorf("%s: peak %.2f is nearly silent", name, peak)
		}
		if peak >= 32000.0/32768 { // 0 dBFS is 32768
			t.Errorf("%s: peak %.3f reaches 0 dBFS", name, peak)
		}
		if clipped > 0 {
			t.Errorf("%s: %d samples at the limit", name, clipped)
		}
		if math.Abs(mean) > 0.003 {
			t.Errorf("%s: DC offset %.4f", name, mean)
		}
		// the jump where the loop comes round must be no bigger than the jumps inside the music
		var diffs []float64
		for i := 1; i < len(l); i++ {
			diffs = append(diffs, math.Abs(l[i]-l[i-1]))
		}
		sort.Float64s(diffs)
		p999 := diffs[len(diffs)*999/1000]
		if seam := math.Abs(l[0] - l[len(l)-1]); seam > 2*p999+0.01 {
			t.Errorf("%s: a click of %.3f where the loop comes round (the music jumps at most %.3f)", name, seam, p999)
		}
	}
}

// only the first intro has a bass drum
func TestOnlyTheFirstIntroHasADrum(t *testing.T) {
	for _, k := range ambientKinds {
		if s := subBassShare(ambientPCM(k)); s > 0.04 {
			t.Errorf("%s: %.0f%% of its energy is sub-bass: there is a drum in it", k, s*100)
		}
	}
	if s := subBassShare(buildThump()); s < 0.2 {
		t.Errorf("the measure does not see the kick of the first intro (%.0f%%)", s*100)
	}
}

// the picture of the interludes and the ending no longer jumps with a kick drum
func TestScenesDoNotPulseWithTheKick(t *testing.T) {
	for _, f := range []string{"interlude.go", "ending.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(b)
		for _, bad := range []string{"beatKick(", "beatN(", "thumpKick(", "StartThump(", "SetThumpVolume("} {
			if strings.Contains(src, bad) {
				t.Errorf("%s uses %s: the drum belongs to the first intro only", f, bad)
			}
		}
	}
}

// muted: nothing starts and nothing breaks; stopping clears it
func TestSceneMuteAndStop(t *testing.T) {
	s := &Sounds{muted: true, tracks: map[string][]byte{}, sfx: map[string][]byte{}}
	built := false
	build := func() []byte { built = true; return nil }
	s.StartScene("base", "k", build, "", true)
	s.SetSceneVolume("base", 1)
	if built || len(s.sceneP) != 0 {
		t.Fatal("a muted game started a scene")
	}
	if !s.SceneActive("base") {
		t.Fatal("the request was not kept")
	}
	s.StartScene("base", "k", build, "", true) // the same request again changes nothing
	if len(s.sceneWant) != 1 {
		t.Fatal("repeated request")
	}
	s.StartScene("base", "other", build, "x.mp3", true)
	if !s.SceneIsSong("base") {
		t.Fatal("a song was asked for")
	}
	s.StopScene()
	if s.SceneActive("base") || len(s.sceneWant) != 0 {
		t.Fatal("stop did not clear it")
	}
}

func TestEndMusicVolume(t *testing.T) {
	for st := endInvite; st <= endFinal; st++ {
		for tt := 0.0; tt < 40; tt += 0.25 {
			e := &endState{stage: st, t: tt}
			if v := endMusicVol(e); v < 0 || v > 1 {
				t.Fatalf("stage %d at %.1f: volume %.2f", st, tt, v)
			}
		}
	}
	if endMusicVol(&endState{stage: endFinal, t: 20}) > 0.01 {
		t.Error("the music should have died away at the end")
	}
	if endMusicVol(&endState{stage: endBed, t: 5}) >= endMusicVol(&endState{stage: endCredits, t: 5}) {
		t.Error("the credits should be louder than the sheep")
	}
}
