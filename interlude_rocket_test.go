package main

import (
	"image/color"
	"math"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestRocketInterludeAfterParty23(t *testing.T) {
	g := testGame()
	builtinLevels()
	disco := campaignIdx("pirate") // the last party before the space station (it was the 70s disco before the pirate ship)
	if k, ok := g.interludeAfter(disco); !ok || k != interRocket || disco != len(campaignLevels)-2 {
		t.Fatal("after the pirate ship (party 25, the last before the space station) there must be the rocket interlude")
	}
	for _, i := range []int{18, 19, 20, 21, 22, 23, disco + 1} {
		if _, ok := g.interludeAfter(i); ok {
			t.Errorf("level %d should not have an interlude", i)
		}
	}
	g.markSeen("seen_interlude20")
	if _, ok := g.interludeAfter(disco); ok {
		t.Error("the rocket interlude shows again after being seen")
	}
	// the other two are not affected
	if _, ok := g.interludeAfter(4); !ok {
		t.Error("seeing the rocket marked the car interlude as seen")
	}
}

func TestRocketInterludeIsListed(t *testing.T) {
	g := testGame()
	debugSnap = nil
	if sceneList[sceneInter3].key != "seen_interlude20" {
		t.Fatal("the scene is not keyed by seen_interlude20")
	}
	if g.sceneUnlocked(sceneInter3) {
		t.Error("locked scene open without being seen")
	}
	if g.sceneName(sceneInter3) != "¿ESTO ES UN BARRIL?" || g.sceneText(sceneInter3) == "" {
		t.Error("the locked label and its hint are missing")
	}
	g.markSeen("seen_interlude20")
	if !g.sceneUnlocked(sceneInter3) || g.sceneName(sceneInter3) != "INTERLUDIO: CORREOS ESPACIALES" {
		t.Error("the scene stays locked after being seen")
	}
	// the memories are the last five parties of the campaign, all of them beaten before the interlude
	pl := interludePlaces(interRocket)
	if len(pl) != 5 || pl[0] != campaignIdx("train") || pl[4] != campaignIdx("pirate") {
		t.Errorf("memories: %v", pl)
	}
}

func TestRocketTimeline(t *testing.T) {
	if !(itStop < rkGuard && rkGuard < rkAsk && rkAsk < rkTicket && rkTicket < rkBoard && rkBoard < rkCount0 &&
		rkCount0+5*rkTick0 <= rkFizzle && rkFizzle < rkKick && rkKick < rkCount1 && rkCount1+3*rkTick1 <= rkLift &&
		rkLift < rkSpace && rkSpace < rkFlash && rkFlash < rocketLen && rocketLen <= 20) {
		t.Fatal("the rocket timeline is out of order")
	}
	if rocketHeight(rkLift-1) != 0 || rocketCam(rkLift+0.5) != 0 {
		t.Error("it moves before the lift off")
	}
	prev := -1.0
	for x := 0.0; x <= rocketLen; x += 0.05 {
		h := rocketHeight(x)
		if h < prev {
			t.Fatalf("the rocket goes down at %.2f", x)
		}
		prev = h
	}
	if rocketHeight(rkSpace) < 600 {
		t.Error("it is not high enough to be in space when the stars come")
	}
}

// the music of the rocket and its memories fit together: same tempo and the same length, so the cross-fade is in step
func TestRocketMusicIsInStep(t *testing.T) {
	a, b := songs[ambRocket], songs[ambMemC]
	if a.bpm != b.bpm || a.bars != b.bars || a.bpm == 0 {
		t.Fatalf("rocket %v bpm x %d bars, memories %v x %d", a.bpm, a.bars, b.bpm, b.bars)
	}
	if sceneMusicCount != sceneRocketN || sceneRocketN != 7 {
		t.Errorf("the rocket song must be file number 7 (it is %d, count %d)", sceneRocketN, sceneMusicCount)
	}
	found := 0
	for _, k := range ambientKinds {
		if k == ambRocket || k == ambMemC {
			found++
		}
	}
	if found != 2 {
		t.Error("the tunes are not in the list that is built in the background")
	}
}

// the guard of the roof is drawn solid
func TestRocketGuardIsSolid(t *testing.T) {
	sp := guardSpec(rand.New(rand.NewSource(77)))
	for _, vw := range []int{vFront, vSide, vBack} {
		b := genChar(&sp, vw, poseFor(AnimIdle, 0))
		op, soft := opaqueCount(b)
		if op < 100 {
			t.Errorf("view %d: only %d opaque pixels", vw, op)
		}
		_ = soft
	}
}

// the redesigned rocket: it is drawn at every moment of the sequence without panics, in 1P and 2P, and its
// footprint stays inside the 960x540 canvas where it sits on the roof. (Not checked here: how it looks.)
func TestRocketBodyDrawsInsideCanvas(t *testing.T) {
	g := auditGame(t) // fonts and px, no window
	oldPx := px
	if px == nil {
		px = ebiten.NewImage(1, 1)
		px.Fill(color.White)
	}
	t.Cleanup(func() { px = oldPx })
	x0, y0, x1, y1 := rocketBox()
	if x0 != math.Round(x0) || y0 != math.Round(y0) || x1 <= x0 || y1 <= y0 {
		t.Fatalf("bad box %v %v %v %v", x0, y0, x1, y1)
	}
	if rkX+x0 < 0 || rkX+x1 > 960 || rkBase+y0 < 0 {
		t.Errorf("the rocket on its pad leaves the canvas: x %.0f..%.0f top %.0f", rkX+x0, rkX+x1, rkBase+y0)
	}
	if rkBase+y1 > 540 {
		t.Errorf("the rocket base is below the canvas")
	}
	for _, two := range []bool{false, true} {
		c := ebiten.NewImage(960, 540)
		for _, tt := range []float64{rkGuard, rkBoard + 2, rkCount0 + 1, rkFizzle + 0.2, rkKick + 0.6, rkCount1 + 2, rkLift + 0.3, rkLift + 3} {
			in := &interState{kind: interRocket, t: tt, two: two}
			g.drawRocketBody(c, in, rkX, rkBase-rocketHeight(tt), func() {})
			g.drawRocketBody(c, in, rkX, rkBase, nil)
		}
		c.Deallocate()
	}
	// the shading tones go from light to dark and the colour mix is exact at its ends
	tones := [4]uint32{1, 2, 3, 4}
	if rkShade(0.15, tones) != 1 || rkShade(0.95, tones) != 4 || rkLerp(0x000000, 0xffffff, 1) != 0xffffff || rkLerp(0x102030, 0x102030, 0.5) != 0x102030 {
		t.Error("shade/lerp helpers are wrong")
	}
}
