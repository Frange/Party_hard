package main

import (
	"encoding/binary"
	"math"
	"sort"
	"strings"
	"testing"
)

func pcmSamples(b []byte) []float64 {
	out := make([]float64, len(b)/4)
	for i := range out {
		out[i] = float64(int16(binary.LittleEndian.Uint16(b[i*4:]))) / 32768
	}
	return out
}

// every party of the campaign (and of the random pool) has a genre that exists
func TestEveryLevelHasAGenre(t *testing.T) {
	defs := builtinLevels()
	if len(defs) < 21 {
		t.Fatalf("only %d levels", len(defs))
	}
	for _, d := range defs {
		if _, ok := genreOfTitle[strings.ToUpper(d.Title)]; !ok {
			t.Errorf("level %q has no genre", d.Title)
		}
	}
	for title, id := range genreOfTitle {
		gi, ok := genreTable[id]
		if !ok {
			t.Errorf("%q: unknown genre %q", title, id)
			continue
		}
		if !gi.legacy {
			if _, ok := genreSongs[gi.id]; !ok {
				t.Errorf("genre %q has no song", gi.id)
			}
		}
	}
	// the campaign has variety: no more than two parties in a row with the same style, and many styles overall
	var seq []string
	for _, e := range campaignLevels {
		var title string
		if e.level != nil {
			title = e.level().Title
		} else {
			title = e.title
		}
		seq = append(seq, genreForTitle(title).id)
	}
	kinds := map[string]bool{}
	for i, id := range seq {
		kinds[id] = true
		if i >= 2 && seq[i-1] == id && seq[i-2] == id {
			t.Errorf("three parties in a row with %s", id)
		}
		if i >= 1 && seq[i-1] == id {
			t.Errorf("parties %d and %d have the same genre %s", i, i+1, id)
		}
	}
	if len(kinds) < 14 {
		t.Errorf("only %d genres in the campaign: %v", len(kinds), seq)
	}
	for _, need := range []string{"electro", "metal", "reggaeton", "pop", "rap", "hiphop", "clasica", "punk"} {
		if !kinds[need] {
			t.Errorf("the campaign never uses %s", need)
		}
	}
}

// the notes of every tune belong to the key of the song (after moving it), the bars add up and the tempo is right
func TestGenreSongsAreWellWritten(t *testing.T) {
	for id, gs := range genreSongs {
		gi, ok := genreTable[id]
		if !ok {
			gi, ok = afterDJTable[id] // the songs after the DJ (songs_afterdj.go)
		}
		if !ok {
			t.Errorf("%s: a song with no genre", id)
		}
		if gs.bpm < gi.lo || gs.bpm > gi.hi {
			t.Errorf("%s: tempo %v out of %v-%v", id, gs.bpm, gi.lo, gi.hi)
		}
		for _, p := range gs.perc {
			if len(p.pattern)%16 != 0 {
				t.Errorf("%s: drum pattern %q is not a whole number of bars", id, p.pattern)
			}
		}
		for _, shift := range []int{0, 5, 7, -2} {
			sc := gs.scale
			sc.rootPC = ((sc.rootPC+shift)%12 + 12) % 12
			for ti, tr := range gs.tracks {
				text := tr.notes
				for _, retro := range []bool{false, true} {
					x := text
					if retro && tr.mel {
						x = retroBars(x)
					}
					if shift != 0 {
						x = transposeText(x, shift)
					}
					var total float64
					for _, n := range parseNotes(x) {
						total += n.dur
						for _, m := range n.midi {
							if !sc.has(m) {
								t.Errorf("%s track %d (%s): %s is not in %s (shift %d)", id, ti, tr.inst, midiName(m), sc.name, shift)
							}
						}
					}
					if math.Abs(total-gs.beats) > 0.01 {
						t.Errorf("%s track %d (%s): %.3f beats, the loop has %.0f", id, ti, tr.inst, total, gs.beats)
					}
				}
			}
		}
	}
}

// the songs sound: not empty, no clipping, no DC offset, no click where the loop closes, one loudness for all
func TestGenreSongsRenderClean(t *testing.T) {
	ids := make([]string, 0, len(genreSongs))
	for id := range genreSongs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for variant := 0; variant < 3; variant++ {
			sp := songSpec{style: id, bpm: genreSongs[id].bpm, shift: []int{0, 5, 7}[variant], variant: variant, seed: 1}
			x := pcmSamples(renderSong(sp))
			if len(x) < sampleRate*10 {
				t.Fatalf("%s v%d: only %d samples", id, variant, len(x))
			}
			var mean, sq, peak float64
			for _, v := range x {
				mean += v
				sq += v * v
				peak = math.Max(peak, math.Abs(v))
			}
			mean /= float64(len(x))
			rms := math.Sqrt(sq / float64(len(x)))
			if peak >= 0.98 {
				t.Errorf("%s v%d: peak %.3f is too high", id, variant, peak)
			}
			if rms < 0.05 {
				t.Errorf("%s v%d: nearly silent (rms %.3f)", id, variant, rms)
			}
			if rms > 0.2 {
				t.Errorf("%s v%d: too loud (rms %.3f)", id, variant, rms)
			}
			if math.Abs(mean) > 0.005 {
				t.Errorf("%s v%d: DC offset %.4f", id, variant, mean)
			}
			// the seam: the jump from the last sample to the first is like any other
			diffs := make([]float64, 0, len(x)/50)
			for i := 1; i < len(x); i += 50 {
				diffs = append(diffs, math.Abs(x[i]-x[i-1]))
			}
			sort.Float64s(diffs)
			p99 := diffs[len(diffs)*99/100]
			if seam := math.Abs(x[0] - x[len(x)-1]); seam > 4*p99+0.03 {
				t.Errorf("%s v%d: click at the loop seam (%.3f, p99 %.3f)", id, variant, seam, p99)
			}
			if variant == 0 {
				t.Logf("%-10s %5.1fs  rms %.3f  peak %.3f", id, float64(len(x))/sampleRate, rms, peak)
			}
		}
	}
}

// the styles of the old sequencer still work and every playlist has three different songs
func TestGenrePlaylists(t *testing.T) {
	for _, title := range []string{"HOUSE PARTY", "MIAMI BEACH", "SKATE PARK", "DISCOTECA", "ESTADIO", "CASINO", "un nivel del usuario"} {
		pl := playlistFor(title)
		if len(pl) != 3 {
			t.Fatalf("%s: %d songs", title, len(pl))
		}
		gi := genreForTitle(title)
		seeds := map[int64]bool{}
		for _, sp := range pl {
			if sp.style != gi.id {
				t.Errorf("%s: song of style %s, the party is %s", title, sp.style, gi.id)
			}
			if sp.bpm < gi.lo-8 || sp.bpm > gi.hi+8 {
				t.Errorf("%s: tempo %v", title, sp.bpm)
			}
			seeds[sp.seed] = true
		}
		if len(seeds) != 3 {
			t.Errorf("%s: the three songs share a seed", title)
		}
	}
	for _, id := range []string{"house", "electro", "techno", "disco", "reggaeton", "punk"} {
		sp := songSpec{style: id, bpm: genreTable[id].bpm, root: 55, scale: scaleMinor, seed: 3, bars: 2}
		if len(renderSong(sp)) == 0 {
			t.Errorf("%s: empty", id)
		}
	}
	// the bands keep to their style
	if bandGenre("metallica") != "metal" || bandGenre("sum41") != "punk" {
		t.Error("band genres")
	}
	for _, slug := range []string{"metallica", "beatles", "elvis", "lostprophets", "sum41"} {
		if len(bandPlaylist(slug)) == 0 {
			t.Errorf("no playlist for %s", slug)
		}
	}
}

// the title card names the style only while our own music plays
func TestGenreLabel(t *testing.T) {
	g := &Game{lvl: &Level{Def: &LevelDef{Title: "MIAMI BEACH"}}}
	if g.levelGenreLabel() != "Música: Reggaetón" {
		t.Errorf("label: %q", g.levelGenreLabel())
	}
	g.musicMode = "level"
	if g.levelGenreLabel() != "" {
		t.Error("a real mp3 is playing: no label")
	}
	g.musicMode = "level-synth"
	g.djDead = true
	if g.levelGenreLabel() != "" {
		t.Error("with the DJ dead it is punk, not the style of the level")
	}
}

// a band of its own style (Metallica) replaces the music of the party; the punk bands play the style of the party
func TestOwnStyleBand(t *testing.T) {
	g := &Game{lvl: &Level{Def: &LevelDef{Title: "MIAMI BEACH"}}}
	g.band = &Band{slug: "sum41"}
	if g.ownStyleBand() || g.levelGenreLabel() == "" {
		t.Error("a punk band must not change the genre of the party")
	}
	g.band = &Band{slug: "metallica"}
	if !g.ownStyleBand() || g.levelGenreLabel() != "" {
		t.Error("Metallica plays metal")
	}
}
