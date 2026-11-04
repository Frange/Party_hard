package main

import (
	"math/rand"
	"testing"
)

func TestRocketInterludeAfterParty20(t *testing.T) {
	g := testGame()
	if k, ok := g.interludeAfter(19); !ok || k != interRocket {
		t.Fatal("after the train station (party 20) there must be the rocket interlude")
	}
	for _, i := range []int{18, 20, 21} {
		if _, ok := g.interludeAfter(i); ok {
			t.Errorf("level %d should not have an interlude", i)
		}
	}
	g.markSeen("seen_interlude20")
	if _, ok := g.interludeAfter(19); ok {
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
	if len(pl) != 5 || pl[0] != 15 || pl[4] != 19 {
		t.Errorf("memories: %v", pl)
	}
}

func TestRocketTimeline(t *testing.T) {
	if !(itStop < rkGuard && rkGuard < rkAsk && rkAsk < rkTicket && rkTicket < rkBoard && rkBoard < rkCount0 &&
		rkCount0+5*rkTick0 <= rkFizzle && rkFizzle < rkKick && rkKick < rkCount1 && rkCount1+3*rkTick1 <= rkLift &&
		rkLift < rkSpace && rkSpace < rkFlash && rkFlash < rocketLen) {
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
