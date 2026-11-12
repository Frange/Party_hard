package main

import (
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func resetAfterDJ(t *testing.T) {
	saved := afterDJ
	t.Cleanup(func() { afterDJ = saved })
	afterDJ.cur, afterDJ.next, afterDJ.idx, afterDJ.on = -1, -1, 0, false
}

// the dead DJ has several songs of different styles, each long enough not to sound like a short loop
func TestAfterDJSongsAreVariedAndLong(t *testing.T) {
	if len(tdjSongs) < 5 {
		t.Fatalf("only %d songs after the DJ", len(tdjSongs))
	}
	insts := map[string]bool{}
	for _, s := range tdjSongs {
		gs, ok := genreSongs[s.id]
		if !ok {
			t.Fatalf("%s is not registered", s.id)
		}
		if secs := gs.beats * 60 / gs.bpm; secs < 60 {
			t.Errorf("%s: %.0f s before it repeats (want >= 60)", s.id, secs)
		}
		mel := ""
		var lead []string
		for _, tr := range gs.tracks {
			if tr.mel {
				mel = tr.notes
				lead = append(lead, tr.inst)
			}
		}
		if mel == "" || len(lead) != 1 {
			t.Fatalf("%s: one tune, not %v", s.id, lead)
		}
		insts[lead[0]] = true
		// the tune evolves: the four sections do not play the same 8 bars
		bars := strings.Split(mel, " | ")
		secs := map[string]bool{}
		distinct := map[string]bool{}
		for i := 0; i < len(bars); i += 8 {
			secs[strings.Join(bars[i:i+8], "|")] = true
		}
		for _, b := range bars {
			distinct[b] = true
		}
		if len(secs) < 4 || len(distinct) < 12 {
			t.Errorf("%s: the tune does not evolve (%d different sections, %d different bars)", s.id, len(secs), len(distinct))
		}
		// a real progression: at least 4 different chords
		chords := map[int]bool{}
		for _, sec := range s.prog {
			for _, d := range sec {
				chords[d] = true
			}
		}
		if len(chords) < 4 {
			t.Errorf("%s: only %d chords", s.id, len(chords))
		}
	}
	if len(insts) < 5 {
		t.Errorf("the tunes use only %d different instruments: %v", len(insts), insts)
	}
}

// the end of the "chunta chunta": the kick changes from bar to bar, rests for whole bars and is seldom four to the floor
func TestAfterDJDrumsAreNotChuntaChunta(t *testing.T) {
	for _, s := range tdjSongs {
		gs := genreSongs[s.id]
		kicks := make([][]byte, tdjBars)
		kits := make([]string, tdjBars)
		for b := range kicks {
			kicks[b] = []byte(dots(16))
		}
		for _, p := range gs.perc {
			for b := 0; b < tdjBars; b++ {
				bar := p.pattern[b*16 : b*16+16]
				kits[b] += bar
				for i := 0; i < 16; i++ {
					if bar[i] == 'k' || bar[i] == 'K' {
						kicks[b][i] = 'k'
					}
				}
			}
		}
		patterns := map[string]bool{}
		allKits := map[string]bool{}
		empty, floor := 0, 0
		for b, k := range kicks {
			patterns[string(k)] = true
			allKits[kits[b]] = true
			if string(k) == dots(16) {
				empty++
			}
			if k[0] == 'k' && k[4] == 'k' && k[8] == 'k' && k[12] == 'k' {
				floor++
			}
		}
		if len(patterns) < 3 {
			t.Errorf("%s: only %d kick patterns", s.id, len(patterns))
		}
		if float64(floor) > 0.25*tdjBars {
			t.Errorf("%s: %d of %d bars four to the floor", s.id, floor, tdjBars)
		}
		if empty < 2 {
			t.Errorf("%s: the kick never rests (%d bars without it)", s.id, empty)
		}
		if len(allKits) < 5 {
			t.Errorf("%s: only %d different drum bars", s.id, len(allKits))
		}
		t.Logf("%-12s kick patterns %d, bars without kick %d, four to the floor %d, drum bars %d", s.id, len(patterns), empty, floor, len(allKits))
	}
}

// every song renders: no long silence, no clipping, a contained volume
func TestAfterDJSongsRenderClean(t *testing.T) {
	for _, sp := range afterDJPlaylist() {
		t0 := time.Now()
		x := pcmSamples(genreSongPCM(sp))
		took := time.Since(t0)
		secs := float64(len(x)) / sampleRate
		if secs < 60 {
			t.Errorf("%s: %.1f s", sp.style, secs)
		}
		var sq, peak float64
		clip := 0
		for _, v := range x {
			sq += v * v
			peak = math.Max(peak, math.Abs(v))
			if math.Abs(v) > 0.97 {
				clip++
			}
		}
		rms := math.Sqrt(sq / float64(len(x)))
		if clip > 0 || peak >= 0.95 {
			t.Errorf("%s: peak %.3f, %d clipped samples", sp.style, peak, clip)
		}
		if rms < 0.06 || rms > 0.16 {
			t.Errorf("%s: rms %.3f (want a contained 0.06-0.16, the parties are 0.17)", sp.style, rms)
		}
		// the longest run of near silence (windows of 1/4 s), at every alignment of the windows (steps of 1/40 s): a
		// silence that straddles two windows is not missed. The render is pure (same samples every run, any load): a
		// silence seen once is always there.
		win, hop := sampleRate/4, sampleRate/40
		ps := make([]float64, len(x)+1) // prefix sums of the squares
		for i, v := range x {
			ps[i+1] = ps[i] + v*v
		}
		worst := 0
		for off := 0; off < win; off += hop {
			run := 0
			for i := off; i+win <= len(x); i += win {
				if math.Sqrt((ps[i+win]-ps[i])/float64(win)) < 0.01 {
					run++
					if run > worst {
						worst = run
					}
				} else {
					run = 0
				}
			}
		}
		if worst >= 4 {
			t.Errorf("%s: %.2f s of silence", sp.style, float64(worst)/4)
		}
		t.Logf("%-12s %5.1fs  rms %.3f  peak %.3f  rendered in %v", sp.style, secs, rms, peak, took.Round(time.Millisecond))
	}
}

// two dead DJs in a row never play the same song, the song changes to another one, and all of them come up
func TestAfterDJNeverRepeats(t *testing.T) {
	resetAfterDJ(t)
	g := &Game{rng: rand.New(rand.NewSource(3))}
	seen := map[string]bool{}
	last := ""
	for death := 0; death < 200; death++ {
		g.djDead = false
		afterDJIdle() // what syncMusic does while the DJ lives
		// a new party: the next song is rendered beforehand (genreSiblings) and is the one the death plays
		sib := genreSiblings(g, "CEMENTERIO")
		pre := sib[len(sib)-1].style
		g.djDead = true
		g.songIdx = g.rng.Intn(5) // as the DJ's death does (game.go)
		sp := g.afterDJSong()
		if sp.style != pre {
			t.Fatalf("death %d: %s plays but %s was rendered beforehand", death, sp.style, pre)
		}
		if sp.style == last {
			t.Fatalf("death %d: %s twice in a row", death, sp.style)
		}
		if again := g.afterDJSong(); again.style != sp.style {
			t.Fatalf("death %d: the song must stay while songIdx does not change", death)
		}
		g.songIdx++ // the song changes (the timer or the booth)
		next := g.afterDJSong()
		if next.style == sp.style {
			t.Fatalf("death %d: the change of song kept %s", death, sp.style)
		}
		seen[sp.style] = true
		last = next.style
	}
	if len(seen) != len(tdjSongs) {
		t.Errorf("only %d of %d songs ever played", len(seen), len(tdjSongs))
	}
}

// the punk that older versions wrote to DJ/ is skipped; any other song of the user plays
func TestAfterDJLegacyFilesSkipped(t *testing.T) {
	for name, legacy := range map[string]bool{"punk_1.mp3": true, "Punk_12.MP3": true, "mi_punk_1.mp3": false, "tras_dj_1_lounge.mp3": false, "punk.mp3": false, "The Clash - London.mp3": false} {
		if legacyDJFile.MatchString(name) != legacy {
			t.Errorf("%s: legacy = %v", name, !legacy)
		}
	}
	p := planMusic(fakeLib(map[int]string{}, nil, nil), "HOUSE PARTY", 0, true, "")
	if p.source != "synth" || p.genre != "afterdj" {
		t.Errorf("DJ dead with an empty DJ/: %+v", p)
	}
}
