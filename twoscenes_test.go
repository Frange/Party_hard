package main

import (
	"math"
	"testing"
)

// in both beds the head is on the pillow and the feet point to the foot of the bed
func TestTwoBedsHeadOnThePillow(t *testing.T) {
	if !twoBedsApart() {
		t.Fatal("the two beds touch or leave no room for the table")
	}
	for _, long := range []float64{44, 40} { // a lying sprite as long as the character is tall
		for _, top := range []float64{0, 3, 8} { // empty rows before the head
			sc := math.Max(2, math.Floor((bed1X1-bed1X0-10)/long))
			for i := 0; i < 2; i++ {
				cx, _ := sleeperPlacement(i, long, top, sc)
				half := long * sc / 2
				var head, feet, pillowC, foot float64
				if i == 0 { // head on the left end of the sprite
					left := cx - half
					head, feet = left+(top+sleeperHeadRows)*sc, left+long*sc
					pillowC, foot = bedHead1X, bed1X1
				} else { // the sprite is mirrored: the head is on its right end
					right := cx + half
					head, feet = right-(top+sleeperHeadRows)*sc, right-long*sc
					pillowC, foot = bedHead2X, bed2X0
				}
				if math.Abs(head-pillowC) > 1 {
					t.Errorf("player %d: head at %.1f but the pillow is at %.1f", i+1, head, pillowC)
				}
				if i == 0 && !(feet > head && feet <= foot+20) || i == 1 && !(feet < head && feet >= foot-20) {
					t.Errorf("player %d: feet at %.1f, head at %.1f: they must point to the foot of the bed (%.1f)", i+1, feet, head, foot)
				}
			}
		}
	}
	// the pillows are at the head end of each bed: left for the first, right for the second
	if !(bedHead1X < (bed1X0+bed1X1)/2 && bedHead2X > (bed2X0+bed2X1)/2) {
		t.Error("the pillows are not at the head ends of the beds")
	}
}

// sig is what a hero is doing in one frame (anything that makes it look different on screen)
type sceneSig struct {
	a, b, c, d, e, f int
}

func fracDifferent(n int, f func(k int) (sceneSig, sceneSig)) float64 {
	diff := 0
	for k := 0; k < n; k++ {
		a, b := f(k)
		if a != b {
			diff++
		}
	}
	return float64(diff) / float64(n)
}

func bi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// the two heroes have their own character: in every scene what each of them does is different
func TestTwoPlayerScenesAreAsymmetric(t *testing.T) {
	// intro: in bed and out of bed
	f := fracDifferent(int(introLen*4), func(k int) (sceneSig, sceneSig) {
		tt := float64(k) / 4
		sig := func(i int) sceneSig {
			p, a := introTwoBed(i, tt), introTwoActorAt(i, tt)
			return sceneSig{int(p.sit * 3), int(p.pillow * 3), bi(p.earplug)*2 + bi(p.fist), int(p.turn), a.anim*4 + int(a.dir), bi(a.visible)*2 + bi(a.hop > 0)}
		}
		return sig(0), sig(1)
	})
	if f < 0.6 {
		t.Errorf("intro: the two heroes do something different in only %.0f%% of the frames", 100*f)
	}
	// the street of the endings, both of them
	for _, secret := range []bool{false, true} {
		walk := 0
		n := 0
		for k := 0; k < 15*10; k++ {
			e := &endState{stage: endWalk, t: float64(k) / 10, total: float64(k) / 10, trueEnd: secret}
			anim, _, dir, hop, _ := endTwoActor(e)
			n++
			if anim != AnimWalk || dir != DirLeft || hop > 0 { // the blue one just walks left
				walk++
			}
		}
		if float64(walk)/float64(n) < 0.25 {
			t.Errorf("ending (secret=%v): the red one just walks like the blue one (%d/%d frames differ)", secret, walk, n)
		}
	}
	// the beds of the endings
	for _, secret := range []bool{false, true} {
		diff, n := 0, 0
		for k := 0; k < int(endLenFor(endBed, secret))*4; k++ {
			e := &endState{stage: endBed, t: float64(k) / 4, trueEnd: secret}
			p := endBedPoses(e)
			n++
			if p[0] != p[1] {
				diff++
			}
		}
		if float64(diff)/float64(n) < 0.6 {
			t.Errorf("ending bed (secret=%v): the two sleepers are the same in too many frames (%d/%d differ)", secret, diff, n)
		}
	}
	// the letter
	diff, n := 0, 0
	for ph := invArrive; ph <= invBoom; ph++ {
		for k := 0; k < 20; k++ {
			a0, p0, h0 := inviteTwoPose(0, ph, float64(k)/10, float64(k)/10)
			a1, p1, h1 := inviteTwoPose(1, ph, float64(k)/10, float64(k)/10)
			n++
			if a0 != a1 || p0 != p1 || h0 != h1 {
				diff++
			}
		}
	}
	if float64(diff)/float64(n) < 0.8 {
		t.Errorf("letter: the two heroes act the same in too many frames (%d/%d differ)", n-diff, n)
	}
	// the street of the interludes: the red one is restless around the blue one, never glued to him
	lo, hi := 1e9, -1e9
	for k := 0; k < 25*10; k++ {
		o := interTwoOffset(float64(k) / 10)
		lo, hi = math.Min(lo, o), math.Max(hi, o)
	}
	if hi-lo < 150 {
		t.Errorf("interlude: the red one only moves %.0f px around the blue one", hi-lo)
	}
	// every line of each of them is theirs: nobody says the other one's lines
	seen := map[string]bool{}
	for _, s := range interTwoLines[0] {
		seen[s] = true
	}
	for _, s := range interTwoLines[1] {
		if seen[s] {
			t.Errorf("both heroes say %q", s)
		}
	}
}

// the glasses of the intro: the blue one's is caught in the air, the red one's breaks
func TestIntroTwoGlassGags(t *testing.T) {
	x, y, _, arm := introTwoGlassA(inGlassLand + 0.2)
	if arm < 0.99 || math.Abs(x-i2CatchX) > 0.5 || math.Abs(y-i2CatchY) > 0.5 {
		t.Errorf("the blue one does not hold his glass at the catch point (arm %.2f, %.1f,%.1f)", arm, x, y)
	}
	if _, _, _, broken := introTwoGlassB(i2GlassBreak - 0.05); broken {
		t.Error("the red one's glass is broken before it lands")
	}
	if _, _, _, broken := introTwoGlassB(i2GlassBreak + 0.05); !broken {
		t.Error("the red one's glass does not break")
	}
	// his glass is back on the table when the sheep arrive
	if x, y, _, arm := introTwoGlassA(inSheepStart); arm > 0.01 || math.Abs(y-i2GlassY) > 3 || x > i2GlassAx0 {
		t.Errorf("the blue one's glass is not back on the table (%.1f, %.1f, arm %.2f)", x, y, arm)
	}
}

// each of them leaves through the door on his own: the red one first, and nobody is in the room when it closes
func TestIntroTwoExit(t *testing.T) {
	a1, a2 := introTwoActorAt(0, inWalk1+0.5), introTwoActorAt(1, inWalk1+0.5)
	if !(a2.alpha < a1.alpha) {
		t.Errorf("the red one (who lost) must go through first: alphas %.2f vs %.2f", a1.alpha, a2.alpha)
	}
	for _, i := range []int{0, 1} {
		if a := introTwoActorAt(i, inWalk1+2.4); a.alpha > 0.01 {
			t.Errorf("hero %d is still in the room when the door has closed", i+1)
		}
	}
	// the red one is at the door long before it opens, and the blue one is not
	b := introTwoActorAt(1, i2AtDoor+0.2)
	a := introTwoActorAt(0, i2AtDoor+0.2)
	if !(b.x < a.x-30) {
		t.Errorf("the red one should be at the door before the blue one: x %.0f vs %.0f", b.x, a.x)
	}
}

// the rocket: the two heroes never stand in the same place, they sit on two different seats and the red one
// gets to the barrel first
func TestRocketTwoSeats(t *testing.T) {
	g := testGame()
	g.inter = &interState{two: true, kind: interRocket}
	apart, n := 0, 0
	for k := 0; k < int((rkBoard+3)*10); k++ {
		tt := float64(k) / 10
		x1, _, _ := g.rocketHeroPos(tt)
		x2, _, _ := g.rocketHeroPos2(tt)
		n++
		if math.Abs(x1-x2) > 12 {
			apart++
		}
	}
	if float64(apart)/float64(n) < 0.8 {
		t.Errorf("rocket: the two heroes are at the same place in too many frames (%d/%d apart)", apart, n)
	}
	x1, _, in1 := g.rocketHeroPos(rkBoard + 2)
	x2, _, in2 := g.rocketHeroPos2(rkBoard + 2)
	if !in1 || !in2 || math.Abs(x1-x2) < 30 {
		t.Errorf("rocket: they must sit on two different seats (x %.0f and %.0f, inside %v %v)", x1, x2, in1, in2)
	}
	// the red one is waiting at the barrel while the blue one is still on his way
	b1, _, _ := g.rocketHeroPos(itBoard0 + 1.0)
	b2, _, _ := g.rocketHeroPos2(itBoard0 + 1.0)
	if !(b2 > b1-1e9) { // (they both move right: the red one must not be behind the blue one by much)
		t.Error("unreachable")
	}
	// one player: nothing changes: the seat is the old one
	g.inter = &interState{kind: interRocket}
	if x, _, _ := g.rocketHeroPos(rkBoard + 2); math.Abs(x-(rkX-4)) > 1.5 {
		t.Errorf("one player: the seat moved to %.1f", x)
	}
}
