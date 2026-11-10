package main

import (
	"math"
	"sort"
)

// ---------------------------------------------------------------------------
// Movement styles ("gaits") of the special guests: every famous character walks and dances like
// himself (moonwalk, robot steps, hips and a shaking leg, military march, hops...), plus a few
// ordinary guests with a trait (drunk, shy, filming with the phone) in a low proportion.
//
// It is only a different drawing: positions, speed, hitboxes, collisions and the one-per-party
// rule do not change. The styles are extra animation kinds after AnimCount (they are hidden from
// the editor); guestPose() swaps the plain walk / dance of a guest for the one of his style and
// gaitDir() may turn the drawn face (moonwalk looks the other way, the robot turns its head in
// 90 degree jumps, the shy one turns his back). Characters of the table below need their slug in
// famousPresets (the test checks it). Real people are the ones the project already has.
// ---------------------------------------------------------------------------

const (
	AnimMoon       = AnimCount + iota // moonwalk: slides back on tiptoe, hand on the hat
	AnimRobot                         // walking robot: stiff, short steps with pauses, right-angle arms
	AnimRobotDance                    // robot dance: arms in right angles, head in jerks
	AnimSwagger                       // strut with the mic
	AnimHips                          // hip dance with a shaking leg and the mic
	AnimMarch                         // military march
	AnimHop                           // little jumps with both feet together
	AnimLumber                        // heavy lumbering of the big ones
	AnimStagger                       // staggering walk of the drunk
	AnimWobble                        // the drunk standing still, swaying
	AnimSneak                         // sneaking, crouched
	AnimGlide                         // slow, upright, regal
	AnimFilm                          // arm up filming with the phone
	AnimHeavy                         // heavy armored steps: short, stiff, little swing, a metallic rebound
	animGaitEnd
)

var gaitFrames = map[int]int{
	AnimMoon: 6, AnimRobot: 4, AnimRobotDance: 4, AnimSwagger: 6, AnimHips: 6, AnimMarch: 6, AnimHop: 4,
	AnimLumber: 6, AnimStagger: 6, AnimWobble: 6, AnimSneak: 6, AnimGlide: 6, AnimFilm: 4, AnimHeavy: 4,
}

// animFrameCount: frames of any animation, the hidden gait ones included.
func animFrameCount(a int) int {
	if a >= 0 && a < AnimCount {
		return animFrames[a]
	}
	if n := gaitFrames[a]; n > 0 {
		return n
	}
	return 1
}

// gaitDef: which animation replaces the walk, the dance and the standing pose (0 = keep the usual).
type gaitDef struct{ walk, dance, idle int }

var gaitDefs = map[string]gaitDef{
	"moon":    {walk: AnimMoon, dance: AnimMoon}, // he also dances sliding backwards
	"robot":   {walk: AnimRobot, dance: AnimRobotDance},
	"hips":    {walk: AnimSwagger, dance: AnimHips, idle: AnimHips},
	"march":   {walk: AnimMarch},
	"hop":     {walk: AnimHop, dance: AnimHop, idle: AnimHop},
	"lumber":  {walk: AnimLumber},
	"stagger": {walk: AnimStagger, dance: AnimWobble, idle: AnimWobble},
	"sneak":   {walk: AnimSneak},
	"glide":   {walk: AnimGlide},
	"shy":     {walk: AnimSneak},
	"filmer":  {idle: AnimFilm},
	"heavy":   {walk: AnimHeavy, dance: AnimRobotDance},
	"spin":    {dance: AnimHop},                         // bouncy dance; gaitDir spins him like a top now and then
	"twister": {walk: AnimLumber, dance: AnimPirouette}, // the Tasmanian Devil dances as a brown whirlwind (taz_art.go)
}

// gaitSlugs: who walks and dances in which style (slugs of the special pool).
var gaitSlugs = map[string]string{
	"michael_jackson": "moon",
	"robocop":         "heavy", "terminator": "robot", "johnny5": "robot", "bender": "robot",
	"elvis":           "hips",
	"capitan_america": "march", "picard": "march",
	"mario": "hop", "pitufo": "hop", "papa_pitufo": "hop", "pitufina": "hop", "minion": "hop", "bob_esponja": "hop",
	"hulk": "lumber", "conan": "lumber", "he_man": "lumber", "taz": "twister", "pedro_picapiedra": "lumber",
	"jack_sparrow": "stagger", "homer": "stagger",
	"batman": "sneak", "spiderman": "sneak", "obelix": "lumber",
	"vader": "glide", "spock": "glide",
	"chopper": "hop",
	"crash":   "spin",
}

// gaitOf: the style of a character; ordinary guests get a trait with a low proportion
// (about 12%: 5% drunk, 4% shy, 3% filming), decided by their own random offset, not by the game's rng.
func gaitOf(slug string, famous bool, off float64) string {
	if s, ok := gaitSlugs[slug]; ok {
		return s
	}
	if famous || slug != "" {
		return ""
	}
	switch k := int(off*1000) % 100; {
	case k < 5:
		return "stagger"
	case k < 9:
		return "shy"
	case k < 12:
		return "filmer"
	}
	return ""
}

// gaitInit gives the guest his style the first time (called from updateGuest).
func (g *Game) gaitInit(gu *Guest) {
	if gu.gaitInit {
		return
	}
	gu.gaitInit = true
	gu.gait = ""
	if gu.cs == nil || gu.cs.Spec == nil || gu.quirk == "kid" || gu.perf != "" {
		return
	}
	name := gu.cs.Spec.Name
	gu.gait = gaitOf(name, name != "" && g.isFamous(name), gu.off)
}

// gaitAnim swaps the usual animation of the guest for the one of his style.
func gaitAnim(gait string, anim int, phase float64) (int, float64) {
	d, ok := gaitDefs[gait]
	if !ok {
		return anim, phase
	}
	switch {
	case anim == AnimWalk && d.walk != 0:
		return d.walk, phase
	case anim >= AnimDance0 && anim <= AnimDance3 && d.dance != 0:
		return d.dance, phase
	case anim == AnimIdle && d.idle != 0:
		return d.idle, phase * 3
	}
	return anim, phase
}

// gaitDir: the way the guest is DRAWN to face (the game keeps using gu.dir).
func (g *Game) gaitDir(gu *Guest) Dir {
	d := gu.dir
	switch gu.gait {
	case "moon": // the moonwalk: he slides away while looking the other way
		if gu.moving {
			switch d {
			case DirLeft:
				return DirRight
			case DirRight:
				return DirLeft
			}
		}
	case "robot": // the head turns by 90 degrees in jumps
		if !gu.moving && gu.state != gDance && gu.state != gDanceWith {
			seq := [4]Dir{d, DirDown, d, DirUp}
			if d == DirDown || d == DirUp {
				seq = [4]Dir{d, DirLeft, d, DirRight}
			}
			return seq[int(g.time/1.1+gu.off)%4]
		}
	case "spin": // dancing, every few seconds he spins round like a top for one second
		if gu.state == gDance || gu.state == gDanceWith {
			if math.Mod(g.time+gu.off*3, 3) < 1 {
				return [4]Dir{DirDown, DirRight, DirUp, DirLeft}[int(g.time*12)%4]
			}
		}
	case "shy": // turns his back when somebody comes close
		if !gu.moving && g.p != nil && dist(g.p.pos, gu.pos) < 55 {
			return DirUp
		}
	}
	return d
}

// gaitPoseFor: the pose of the hidden animation kinds.
func gaitPoseFor(kind, f int) Pose {
	n := float64(animFrameCount(kind))
	phi := float64(f) / n
	t := 2 * math.Pi * phi
	s, c := math.Sin(t), math.Cos(t)
	odd := float64(f % 2)
	p := Pose{LA: restArm(), RA: restArm(), LF: restFoot(), RF: restFoot()}
	switch kind {
	case AnimMoon:
		// the front foot on tiptoe slides back while the other stays flat, then they swap
		p.LF = Limb{1.5, 10 - 1.4*math.Max(0, s), 4 * c}
		p.RF = Limb{1.5, 10 - 1.4*math.Max(0, -s), -4 * c}
		p.RA = Limb{3.2, -3.2, 1.6} // hand to the brim of the hat
		p.LA = Limb{1.5, 8, -1}
		p.Lean = -1
		p.Bob = -0.4 * math.Abs(s)
	case AnimRobot:
		sg := [4]float64{1, 1, -1, -1}[f]
		lift := [4]float64{1.6, 0, 1.6, 0}[f]
		p.LF = Limb{1.5, 10 - lift*b2f(f == 0), 3.4 * sg}
		p.RF = Limb{1.5, 10 - lift*b2f(f == 2), -3.4 * sg}
		p.LA = Limb{2.4, 5, 3.4 * sg} // forearms at right angles
		p.RA = Limb{2.4, 5, -3.4 * sg}
	case AnimRobotDance:
		switch f {
		case 0:
			p.RA, p.LA = Limb{6.5, -1, 0}, Limb{1.5, 9, 0}
		case 1:
			p.RA, p.LA = Limb{2.2, 2, 8}, Limb{2.2, 2, 8}
			p.HeadZ = 1.5
		case 2:
			p.LA, p.RA = Limb{6.5, -1, 0}, Limb{1.5, 9, 0}
		default:
			p.RA, p.LA = Limb{8, 3, 0}, Limb{8, 3, 0}
			p.HeadZ = -1.5
		}
	case AnimSwagger:
		walkLegs(&p, phi, 4.2, 2.5)
		p.Sway = 2 * s
		p.RA = Limb{3, 0.5, 3} // the mic at the chest
		p.LA = Limb{1.5, 9, -2 * s}
		p.Bob = -0.6 * math.Abs(s)
	case AnimHips:
		p.Sway = 2.6 * s
		p.Lean = 0.8 * c
		p.LF = Limb{1.5 + 0.7*odd, 10 - 0.8*odd, 0}
		p.RF = Limb{2.6, 10 - 1.3, 3 + 0.9*odd} // the knee shakes
		p.RA = Limb{4, -2.5, 2.5}
		p.LA = Limb{6.5 + s, 2, 0}
		p.Open = true
		p.Bob = 0.6 * math.Abs(c)
	case AnimMarch:
		walkLegs(&p, phi, 4.5, 6.5)
		p.LA = Limb{1.5, 8.5, -6 * s}
		p.RA = Limb{1.5, 8.5, 6 * s}
		p.Bob = -0.3 * math.Abs(s)
	case AnimHop:
		lift := [4]float64{0, 4, 5, 1}[f]
		p.LF, p.RF = Limb{1.2, 10 - lift, 0}, Limb{1.2, 10 - lift, 0}
		p.Bob = [4]float64{1.2, -3, -4, 0.5}[f]
		up := [4]float64{4, 1, 0, 3}[f]
		p.LA, p.RA = Limb{4, up + 1, 1}, Limb{4, up + 1, 1}
	case AnimLumber:
		walkLegs(&p, phi, 3.8, 1.5)
		p.LA = Limb{5.5, 6 + c, 1 - 2*s}
		p.RA = Limb{5.5, 6 - c, 1 + 2*s}
		p.Sway = 1.8 * s
		p.Lean = 1.5
		p.Bob = 0.9 * math.Abs(c)
	case AnimStagger:
		walkLegs(&p, phi, 4, 2)
		p.Sway = 3*s + 1.2*math.Sin(2*t+1)
		p.Lean = 1.5 * math.Sin(t+1)
		p.LA = Limb{3, 6 + 2*s, 2 * c}
		p.RA = Limb{3, 6 - 2*c, -2 * s}
	case AnimWobble:
		p.Sway = 3 * s
		p.Lean = 1.5 * c
		p.LA = Limb{5.5, 5, 0}
		p.RA = Limb{2.5, 8, 0}
		p.LF = Limb{2.3 + 0.8*s, 10, 0}
		p.RF = Limb{0.7 - 0.8*s, 10, 0}
	case AnimSneak:
		walkLegs(&p, phi, 2.6, 1.2)
		p.Bob = 2.2
		p.Lean = 3
		p.LA = Limb{1, 6, 3}
		p.RA = Limb{1, 6, 3}
	case AnimGlide:
		walkLegs(&p, phi, 1.4, 0.3)
		p.LA = Limb{0.5, 6, 3} // arms folded
		p.RA = Limb{0.5, 6, 3}
	case AnimHeavy:
		// one heavy foot at a time: short steps, arms almost still, the body drops a little
		// when the foot lands (metallic rebound)
		sg := [4]float64{1, 0.3, -1, -0.3}[f]
		p.LF = Limb{1.5, 10 - 1.1*b2f(f == 0), 2.6 * sg}
		p.RF = Limb{1.5, 10 - 1.1*b2f(f == 2), -2.6 * sg}
		p.LA = Limb{1.1, 8.6, -0.8 * sg}
		p.RA = Limb{1.1, 8.6, 0.8 * sg}
		p.Bob = [4]float64{1, 0, 1, 0}[f]
		p.Sway = 0.3 * sg
	case AnimFilm:
		p.RA = Limb{3, -3.5, 6}
		p.Hold = HoldPhone
		p.HeadZ = 0.5
		p.Bob = 0.3 * math.Abs(s)
	}
	return p
}

// gaitSummary: one line per styled character, for the tests and the debug listing.
func gaitSummary() []string {
	var out []string
	for slug, s := range gaitSlugs {
		out = append(out, slug+" = "+s)
	}
	sort.Strings(out)
	return out
}
