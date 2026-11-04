package main

// ---------------------------------------------------------------------------
// The tune of the rocket interlude (after party 20): a comic, bouncy march in F major with a chiptune
// lead, a marimba ostinato and a jumping plucked bass. No bass drum, like all the story scenes: the
// rhythm is woodblock and shaker. The memories cut to a sneaky F minor version of the angry-walk tune.
// ---------------------------------------------------------------------------

var scFMinor = scaleDef{"F minor", 5, stepsMinor}

const (
	ambRocket = "rocket"
	ambMemC   = "memoryC"
)

var rocketMel = noteLine(
	"F5:0.5 A5:0.5 C6:0.5 A5:0.5 F5:1 R:1",
	"G5:0.5 Bb5:0.5 D6:0.5 Bb5:0.5 G5:1 R:1",
	"A5:0.5 C6:0.5 F6:0.5 C6:0.5 A5:1 G5:0.5 F5:0.5",
	"G5:0.5 E5:0.5 D5:0.5 E5:0.5 F5:2",
	"F5:0.5 A5:0.5 C6:0.5 A5:0.5 F5:1 R:1",
	"D6:0.5 D6:0.5 D6:0.5 Bb5:0.5 F6:1 R:1",
	"E6:0.5 D6:0.5 C6:0.5 Bb5:0.5 A5:0.5 G5:0.5 A5:0.5 C6:0.5",
	"F5:1 C6:0.5 A5:0.5 F5:1 R:1",
)

// the bass hops between the root and the fifth, never below ~130 Hz
var rocketBass = noteLine(
	"F3:1 R:1 C4:1 R:1", "Bb3:1 R:1 F4:1 R:1", "F3:1 R:1 C4:1 R:1", "C3:1 R:1 G3:1 R:1",
	"F3:1 R:1 C4:1 R:1", "Bb3:1 R:1 F4:1 R:1", "C3:1 R:1 G3:1 R:1", "F3:1 R:1 C4:1 R:1",
)

// the off-beat "pa pa" of the marimba
var rocketStab = noteLine(
	"R:1 [A4,C5]:1 R:1 [A4,C5]:1", "R:1 [Bb4,D5]:1 R:1 [Bb4,D5]:1", "R:1 [A4,C5]:1 R:1 [A4,C5]:1", "R:1 [G4,C5]:1 R:1 [G4,C5]:1",
	"R:1 [A4,C5]:1 R:1 [A4,C5]:1", "R:1 [Bb4,D5]:1 R:1 [Bb4,D5]:1", "R:1 [G4,C5]:1 R:1 [G4,C5]:1", "R:1 [A4,C5]:1 R:1 [A4,C5]:1",
)

func init() {
	songs[ambRocket] = songDef{name: "rocket", bpm: 138, bars: 8, scale: scFMajor, echo: 0.75, fb: 0.28, wet: 0.2,
		tracks: []trackDef{{"square", rocketMel, 0.4}, {"pizz", rocketBass, 0.55}, {"marimba", rocketStab, 0.24}},
		perc:   []percDef{{"....w.......w...", 0.33}, {"s.s.s.s.s.s.s.s.", 0.16}}}
	// the memories: the sneaky walk, a fourth higher, at the same tempo and length as the rocket tune so they cross-fade in step
	songs[ambMemC] = songDef{name: "memoryC", bpm: 138, bars: 8, scale: scFMinor, echo: 0.5, fb: 0.28, wet: 0.2,
		tracks: []trackDef{{"marimba", transposeText(sneakMel, 5), 0.5}, {"pizz", transposeText(sneakBass, 5), 0.5}},
		perc:   []percDef{{"w...w...w...w...", 0.22}, {"h.h.h.h.h.h.h.h.", 0.1}}}
	ambientKinds = append(ambientKinds, ambRocket, ambMemC)
}
