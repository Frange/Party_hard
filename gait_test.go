package main

import (
	"math"
	"math/rand"
	"testing"
)

var gaitKinds = []int{AnimMoon, AnimRobot, AnimRobotDance, AnimSwagger, AnimHips, AnimMarch, AnimHop, AnimLumber, AnimStagger, AnimWobble, AnimSneak, AnimGlide, AnimFilm}

func gaitFrame(kind, f int) Pose { return poseFor(kind, f) }

func limbs(p Pose) []Limb { return []Limb{p.LA, p.RA, p.LF, p.RF} }

// every character with a style exists in the project, and the style exists
func TestGaitSlugsExist(t *testing.T) {
	for slug, st := range gaitSlugs {
		if presetByName(slug) == nil {
			t.Errorf("%s has a gait (%s) but is not a character of the project", slug, st)
		}
		if _, ok := gaitDefs[st]; !ok {
			t.Errorf("%s: unknown gait %s", slug, st)
		}
	}
	for _, must := range []string{"michael_jackson", "elvis", "robocop"} {
		if gaitSlugs[must] == "" {
			t.Errorf("%s must have a gait", must)
		}
	}
	styles := map[string]bool{}
	for _, st := range gaitSlugs {
		styles[st] = true
	}
	if len(styles) < 8 || len(gaitSlugs) < 16 {
		t.Errorf("expected at least 8 styles and 16 characters, got %d and %d", len(styles), len(gaitSlugs))
	}
	for _, l := range gaitSummary() {
		t.Log(l)
	}
}

// every gait animation is finite, deterministic, moves the limbs and draws without panic
func TestGaitPosesDeterministicAndDrawn(t *testing.T) {
	spec := presetByName("robocop")(rand.New(rand.NewSource(1)))
	for _, k := range gaitKinds {
		n := animFrameCount(k)
		if n < 2 {
			t.Fatalf("anim %d has %d frames", k, n)
		}
		moved := false
		for f := 0; f < n; f++ {
			a, b := gaitFrame(k, f), gaitFrame(k, f)
			if a != b {
				t.Errorf("anim %d frame %d is not deterministic", k, f)
			}
			for _, l := range limbs(a) {
				for _, v := range []float64{l.X, l.Y, l.Z} {
					if math.IsNaN(v) || math.Abs(v) > 14 {
						t.Errorf("anim %d frame %d: limb out of range %v", k, f, l)
					}
				}
			}
			if a != gaitFrame(k, 0) {
				moved = true
			}
			for vw := 0; vw < 3; vw++ {
				if b := genChar(&spec, vw, a); b == nil {
					t.Errorf("anim %d view %d: no sprite", k, vw)
				}
			}
		}
		if !moved {
			t.Errorf("anim %d never changes between frames", k)
		}
	}
}

func TestGaitExpectedAngles(t *testing.T) {
	// moonwalk: the feet slide in opposite directions, one on tiptoe
	var minZ, maxZ, lift float64
	for f := 0; f < animFrameCount(AnimMoon); f++ {
		p := gaitFrame(AnimMoon, f)
		minZ, maxZ = math.Min(minZ, p.LF.Z), math.Max(maxZ, p.LF.Z)
		lift = math.Max(lift, 10-math.Min(p.LF.Y, p.RF.Y))
		if math.Abs(p.LF.Z+p.RF.Z) > 1e-9 {
			t.Errorf("moonwalk frame %d: feet must slide in opposite directions", f)
		}
		if p.RA.Y > 0 {
			t.Errorf("moonwalk: the hand must reach the hat")
		}
	}
	if maxZ-minZ < 5 || lift < 1 {
		t.Errorf("moonwalk: slide %.1f tiptoe %.1f", maxZ-minZ, lift)
	}
	// robot: steps in pairs of frames (stiff pause), arms always at right angles
	p0, p1, p2 := gaitFrame(AnimRobot, 0), gaitFrame(AnimRobot, 1), gaitFrame(AnimRobot, 2)
	if p0.LF.Z != p1.LF.Z || p0.LF.Z == p2.LF.Z {
		t.Errorf("robot: pause then step expected")
	}
	for f := 0; f < 4; f++ {
		if p := gaitFrame(AnimRobot, f); p.LA.Y != 5 || p.RA.Y != 5 || p.Bob != 0 {
			t.Errorf("robot frame %d: rigid arms and no bob expected, got %+v", f, p)
		}
	}
	jerk := map[float64]bool{}
	for f := 0; f < 4; f++ {
		jerk[gaitFrame(AnimRobotDance, f).HeadZ] = true
	}
	if len(jerk) < 3 {
		t.Errorf("robot dance: the head must jump between positions")
	}
	// Elvis: hips swing both ways and the leg shakes
	var sMin, sMax, tMin, tMax float64 = 9, -9, 9, -9
	for f := 0; f < 6; f++ {
		p := gaitFrame(AnimHips, f)
		sMin, sMax = math.Min(sMin, p.Sway), math.Max(sMax, p.Sway)
		tMin, tMax = math.Min(tMin, p.RF.Z), math.Max(tMax, p.RF.Z)
		if p.Hold != 0 || !p.Open {
			continue
		}
	}
	if sMax-sMin < 4 || tMax-tMin < 0.5 {
		t.Errorf("hips: sway %.1f tremble %.1f", sMax-sMin, tMax-tMin)
	}
	// hop: both feet together and a real jump
	var bMin, bMax float64
	for f := 0; f < 4; f++ {
		p := gaitFrame(AnimHop, f)
		if p.LF != p.RF {
			t.Errorf("hop: feet together expected")
		}
		bMin, bMax = math.Min(bMin, p.Bob), math.Max(bMax, p.Bob)
	}
	if bMax-bMin < 4 {
		t.Errorf("hop too small: %.1f", bMax-bMin)
	}
	// march: high knees
	var hi float64
	for f := 0; f < 6; f++ {
		p := gaitFrame(AnimMarch, f)
		hi = math.Max(hi, 10-math.Min(p.LF.Y, p.RF.Y))
	}
	if hi < 5 {
		t.Errorf("march: knees too low (%.1f)", hi)
	}
}

func TestGaitAnimSwapAndDir(t *testing.T) {
	if a, _ := gaitAnim("moon", AnimWalk, 1); a != AnimMoon {
		t.Errorf("moonwalk replaces the walk")
	}
	if a, _ := gaitAnim("moon", AnimRun, 1); a != AnimRun {
		t.Errorf("running is not replaced")
	}
	if a, _ := gaitAnim("robot", AnimDance2, 1); a != AnimRobotDance {
		t.Errorf("robot dance replaces the dances")
	}
	if a, _ := gaitAnim("", AnimWalk, 1); a != AnimWalk {
		t.Errorf("no gait = no change")
	}
	g := testGame()
	g.p = &Player{pos: Vec{100, 100}}
	mj := &Guest{gait: "moon", moving: true, dir: DirRight}
	if g.gaitDir(mj) != DirLeft {
		t.Errorf("moonwalk looks backwards")
	}
	mj.moving = false
	if g.gaitDir(mj) != DirRight {
		t.Errorf("standing, the face is the real one")
	}
	rb := &Guest{gait: "robot", dir: DirRight, off: 0.3}
	seen := map[Dir]bool{}
	for g.time = 0; g.time < 10; g.time += 0.25 {
		seen[g.gaitDir(rb)] = true
	}
	if len(seen) < 3 {
		t.Errorf("robot head must turn in jumps, saw %v", seen)
	}
	shy := &Guest{gait: "shy", dir: DirDown, pos: Vec{120, 100}}
	if g.gaitDir(shy) != DirUp {
		t.Errorf("shy guest turns his back to the player")
	}
	shy.pos = Vec{400, 100}
	if g.gaitDir(shy) != DirDown {
		t.Errorf("far away he does not turn")
	}
}

// ordinary guests with a trait: a low proportion, never the famous or the named
func TestGaitTraitProportion(t *testing.T) {
	n, hit := 0, map[string]int{}
	for off := 0.0; off < 6; off += 0.0007 {
		n++
		hit[gaitOf("", false, off)]++
	}
	rest := 1 - float64(hit[""])/float64(n)
	if rest < 0.08 || rest > 0.16 {
		t.Errorf("traits for %.0f%% of the guests: %v", rest*100, hit)
	}
	if hit["stagger"] == 0 || hit["shy"] == 0 || hit["filmer"] == 0 {
		t.Errorf("all three traits should appear: %v", hit)
	}
	if gaitOf("", true, 0.001) != "" || gaitOf("someone", false, 0.001) != "" {
		t.Errorf("famous or named guests have no random trait")
	}
}
