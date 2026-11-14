package main

// ---------------------------------------------------------------------------
// The music of the ESTACIÓN DE TREN (v1.4.0, tanda H): the user found its rap loop too dull, so the station gets a
// new genre of its own: BREAKBEAT, a fast broken beat (172 BPM) with a sub bass that rolls like a train, piano stabs
// and a minor-key lead, written like the other songs of genres.go. The rap song stays in the game (a user mp3 named
// "rap" still finds its style); no party plays it any more.
// ---------------------------------------------------------------------------

func init() {
	cm, ab, bb := gChord(gN("C4"), gN("Eb4"), gN("G4")), gChord(gN("Ab3"), gN("C4"), gN("Eb4")), gChord(gN("Bb3"), gN("D4"), gN("F4"))
	eb, fm, gm := gChord(gN("Eb4"), gN("G4"), gN("Bb4")), gChord(gN("F3"), gN("Ab3"), gN("C4")), gChord(gN("G3"), gN("Bb3"), gN("D4"))
	prog := []string{cm, cm, ab, bb, eb, fm, gm, cm}
	roots := []int{gN("C2"), gN("C2"), gN("Ab1"), gN("Bb1"), gN("Eb2"), gN("F2"), gN("G1"), gN("C2")}
	genreSongs["breakbeat"] = genreSong{bpm: 172, beats: 32, scale: scCMinor, echo: 0.375, fb: 0.25, wet: 0.12,
		tracks: []gTrack{
			{"saw", noteLine(
				"C5:0.5 Eb5:0.5 G5:0.5 C5:0.5 Bb4:1 G4:1", "Ab4:0.5 C5:0.5 Eb5:1 D5:0.5 C5:0.5 Bb4:1",
				"G4:0.5 Bb4:0.5 D5:0.5 F5:0.5 Eb5:1 D5:1", "C5:1 G5:1 C5:2",
				"Eb5:0.5 F5:0.5 G5:0.5 Bb5:0.5 Ab5:1 G5:1", "F5:0.5 Eb5:0.5 D5:0.5 C5:0.5 D5:1 Eb5:1",
				"G5:0.5 F5:0.5 Eb5:0.5 D5:0.5 C5:0.5 D5:0.5 Eb5:1", "D5:1 Bb4:1 C5:2"), 0.26, true},
			{"piano", perBar(8, func(i int) string { return prog[i] + ":0.5 R:1 " + prog[i] + ":0.5 R:1.5 " + prog[i] + ":0.5" }), 0.22, false},
			{"sub", perBar(8, func(i int) string {
				r := roots[i]
				return midiName(r) + ":0.75 R:0.25 " + midiName(r) + ":0.5 " + midiName(r+12) + ":0.5 R:0.5 " + midiName(r) + ":0.5 " + midiName(r+7) + ":1"
			}), 0.6, false},
		},
		perc: []gPerc{{"k.........k.....", 0.7}, {"....n.......n..n", 0.55}, {"hhhhhhhhhhhhhhhh", 0.12}, {"..g....g..g....g", 0.22}, {"x" + dots(63), 0.3}}}
	genreTable["breakbeat"] = genreInfo{"breakbeat", "Música: Breakbeat", false, 172, 164, 180}
	for _, w := range []string{"breakbeat", "breaks", "dnb", "drumandbass", "jungle"} {
		genreAliases[w] = "breakbeat"
	}
	genreShortName["breakbeat"] = "Breakbeat"
	if _, ok := langDataEN["Música: Breakbeat"]; !ok {
		langDataEN["Música: Breakbeat"] = "Music: Breakbeat"
	}
}
