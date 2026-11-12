package main

import (
	"math"
	"strings"
	"testing"
)

// The island has its own piece (an original homage to the adventure scores of the big orchestral films). It is not
// a copy of anything: this checks its character, not any tune of anybody.
func TestIslaScore(t *testing.T) {
	gs, ok := genreSongs["isla"]
	if !ok {
		t.Fatal("no score for the island")
	}
	if genreForTitle("ISLA NUBLADA").id != "isla" || genreForTitle("ISLA NUBLADA").label != "Música: Orquestal épica" {
		t.Errorf("the island is %+v", genreForTitle("ISLA NUBLADA"))
	}
	if gs.bpm < 60 || gs.bpm > 80 || gs.beats != 64 {
		t.Errorf("slow and long: %v bpm, %v beats", gs.bpm, gs.beats)
	}
	// the horn line: 16 bars
	var horn []songNote
	for _, tr := range gs.tracks {
		if tr.inst == "horn" {
			horn = parseNotes(tr.notes)
		}
	}
	if len(horn) == 0 {
		t.Fatal("no horn")
	}
	barOf := func(n songNote) int { return int(n.at / 4) }
	var theme, danger, back []int // midi notes of the three sections
	for _, n := range horn {
		if len(n.midi) == 0 {
			continue
		}
		switch b := barOf(n); {
		case b < 8:
			theme = append(theme, n.midi[0])
		case b < 12:
			danger = append(danger, n.midi[0])
		default:
			back = append(back, n.midi[0])
		}
	}
	leaps, run, maxRun := 0, 0, 0
	for i := 1; i < len(theme); i++ {
		d := theme[i] - theme[i-1]
		if d >= 5 || d <= -5 {
			leaps++
		}
		if d > 0 && d <= 2 { // an ascending stepwise run: the most generic thing a tune can do
			run++
		} else {
			run = 0
		}
		if run > maxRun {
			maxRun = run
		}
	}
	if leaps < 4 {
		t.Errorf("the wonder theme must have wide leaps, it has %d", leaps)
	}
	if maxRun >= 3 {
		t.Errorf("the theme climbs by steps %d times in a row", maxRun+1)
	}
	hi := func(x []int) (m int) {
		for _, v := range x {
			if v > m {
				m = v
			}
		}
		return
	}
	if hi(back) <= hi(theme) {
		t.Errorf("the return must go higher than the first time (%d vs %d)", hi(back), hi(theme))
	}
	if hi(danger) >= gN("A4") {
		t.Errorf("the danger whispers low, its top note is %s", midiName(hi(danger)))
	}
	if theme[len(theme)-1]%12 != 2 || back[len(back)-1]%12 != 2 { // both end on D
		t.Error("the theme rests on the tonic")
	}
	// the danger has stomping timpani and a dark ostinato, the wonder has harp and celesta
	var stomps int
	for _, p := range gs.perc {
		if len(p.pattern) == 256 {
			for _, c := range p.pattern[8*16 : 12*16] { // bars 9-12
				if c == 'T' || c == 't' {
					stomps++
				}
			}
		}
	}
	if stomps < 10 {
		t.Errorf("only %d stomps in the danger", stomps)
	}
	var low bool
	for _, tr := range gs.tracks {
		if tr.inst == "ebass" {
			for _, n := range parseNotes(tr.notes) {
				if n.at >= 32 && n.at < 48 && len(n.midi) > 0 && n.midi[0] <= gN("B1")+7 {
					low = true
				}
			}
		}
	}
	if !low {
		t.Error("no low ostinato in the danger")
	}
	for _, need := range []string{"harp", "celesta", "strings", "brass", "horn"} {
		found := false
		for _, tr := range gs.tracks {
			found = found || tr.inst == need
		}
		if !found {
			t.Errorf("no %s", need)
		}
	}
	// the three variants are different songs (key, speed and order of the sections)
	pl := genrePlaylist("ISLA NUBLADA")
	if len(pl) != 3 || pl[0].shift != 0 || pl[1].shift == pl[0].shift || pl[2].shift == pl[1].shift {
		t.Errorf("playlist %+v", pl)
	}
	a := genreSongPCM(pl[0])
	b := genreSongPCM(pl[1])
	if len(a) < 40*sampleRate*4 || strings.EqualFold(string(a[:4000]), string(b[:4000])) {
		t.Error("the variants must differ")
	}
	x := pcmSamples(a)
	var peak float64
	for _, v := range x {
		peak = math.Max(peak, math.Abs(v))
	}
	if peak >= 0.98 {
		t.Errorf("peak %.3f", peak)
	}
}
