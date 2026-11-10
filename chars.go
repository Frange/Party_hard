package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Skeleton based pixel-art characters: every animation is a pose function
// evaluated per frame, drawn with 2-bone limbs, so movement looks like people.
// ---------------------------------------------------------------------------

const (
	charW    = 24
	charH    = 44
	charCX   = 12
	charFeet = 42 // row of the soles
)

type Limb struct{ X, Y, Z float64 } // target relative to shoulder/hip: X outward, Y down, Z forward

type Pose struct {
	LA, RA, LF, RF Limb
	Lean           float64
	Bob            float64
	Sway           float64
	HeadZ          float64
	Hold           int
	Sit            bool
	Open           bool
	Anim, F        int // animation and frame this pose was made for (vader.go reads them)
}

// Hold items
const (
	HoldNone = iota
	HoldKnife
	HoldCup
	HoldMic
	HoldGuitar
	HoldPhone
	HoldBaguette
	HoldTray
	HoldCamera
	HoldBigCup
	HoldSaber
	HoldBeer
	HoldFlag
	HoldPizza
	HoldSign
	HoldMachete
	HoldChainsaw
	HoldClaw
	HoldBroom
	HoldVacuum
	HoldSelfie
	HoldPlant
	HoldTrophy
	HoldBalloons
	HoldMegaphone
	HoldBriefcase
	HoldTorch
	HoldSax
	HoldPee
	HoldFloat // lifeguard's ring buoy
	HoldBoard // lifeguard's rescue board
	HoldGun
	HoldExt    // fire extinguisher
	HoldPompom // a pompom in each hand (cheerleaders)
	HoldStick  // hockey stick
	HoldRifle  // the endoskeleton's big rifle (ktm_art.go)
)

const (
	AnimIdle = iota
	AnimWalk
	AnimRun
	AnimDance0
	AnimDance1
	AnimDance2
	AnimDance3
	AnimDance4
	AnimDance5
	AnimDance6
	AnimDance7
	AnimDance8
	AnimDance9
	AnimDance10
	AnimDance11
	AnimDrink
	AnimTalk
	AnimWave
	AnimPunch
	AnimStab
	AnimFall
	AnimVomit
	AnimKiss
	AnimDJ
	AnimDrum
	AnimGuitar
	AnimSing
	AnimCarry
	AnimTired
	AnimLying
	AnimSleep
	AnimCall
	AnimPanic
	AnimCrouch
	AnimPoint
	AnimLove
	AnimLaugh
	AnimPirouette
	AnimPee
	AnimZombie
	AnimLean
	AnimCheer
	AnimCount
)

var animFrames = [AnimCount]int{
	AnimIdle: 2, AnimWalk: 6, AnimRun: 6,
	AnimDance0: 6, AnimDance1: 6, AnimDance2: 6, AnimDance3: 6,
	AnimDance4: 6, AnimDance5: 6, AnimDance6: 6, AnimDance7: 6, AnimDance8: 6, AnimDance9: 6, AnimDance10: 6, AnimDance11: 6,
	AnimDrink: 6, AnimTalk: 6, AnimWave: 4, AnimPunch: 4, AnimStab: 3, AnimFall: 3, AnimVomit: 4, AnimKiss: 2,
	AnimDJ: 4, AnimDrum: 4, AnimGuitar: 4, AnimSing: 4, AnimCarry: 6, AnimTired: 2, AnimLying: 1, AnimSleep: 1,
	AnimCall: 2, AnimPanic: 6, AnimCrouch: 2,
	AnimPoint: 4, AnimLove: 6, AnimLaugh: 4, AnimPirouette: 6, AnimPee: 4, AnimZombie: 6, AnimLean: 2, AnimCheer: 4,
}

func restArm() Limb  { return Limb{0.7, 9, 0} }
func restFoot() Limb { return Limb{1.5, 10, 0} }

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func walkLegs(p *Pose, phi, amp, lift float64) {
	s, c := math.Sin(2*math.Pi*phi), math.Cos(2*math.Pi*phi)
	s2, c2 := -s, -c
	p.LF = Limb{1.5, 10 - lift*math.Max(0, c), amp * s}
	p.RF = Limb{1.5, 10 - lift*math.Max(0, c2), amp * s2}
}

func poseFor(kind, f int) Pose {
	if kind >= AnimCount { // the hidden gait animations (gait.go)
		return gaitPoseFor(kind, f)
	}
	n := float64(animFrames[kind])
	phi := float64(f) / n
	t := 2 * math.Pi * phi
	s, c := math.Sin(t), math.Cos(t)
	p := Pose{LA: restArm(), RA: restArm(), LF: restFoot(), RF: restFoot()}
	switch kind {
	case AnimIdle:
		p.Bob = float64(f)
		p.LA.Y, p.RA.Y = 9-float64(f)*0.4, 9-float64(f)*0.4
	case AnimWalk:
		walkLegs(&p, phi, 5.2, 3.0)
		p.LA = Limb{1.5, 9 - 0.6*math.Abs(s), -3.8 * s}
		p.RA = Limb{1.5, 9 - 0.6*math.Abs(s), 3.8 * s}
		p.Bob = -0.7 * math.Abs(s)
	case AnimRun:
		walkLegs(&p, phi, 8, 5.5)
		p.LA = Limb{2, 5.5, -6 * s}
		p.RA = Limb{2, 5.5, 6 * s}
		p.Lean = 2
		p.Bob = -1.2 * math.Abs(s)
	case AnimLean: // Michael Jackson's impossible lean: the body tilts, the feet stay
		p.Lean = 9
		p.LA = Limb{1, 9, 0}
		p.RA = Limb{1, 9, 0}
		p.Bob = 0
	case AnimZombie: // arms stretched straight out in front, shuffling legs
		walkLegs(&p, phi, 3.4, 1.8)
		p.LA = Limb{2.2, -1, 10}
		p.RA = Limb{2.2, -1, 10}
		p.Lean = 1
		p.Bob = -0.5 * math.Abs(s)
	case AnimCarry:
		walkLegs(&p, phi, 4, 2.6)
		p.LA = Limb{3.5, -8, 0}
		p.RA = Limb{3.5, -8, 0}
		p.Bob = -0.6 * math.Abs(s)
	case AnimPanic:
		walkLegs(&p, phi, 8, 5.5)
		p.LA = Limb{5, -7 + 2*s, 0}
		p.RA = Limb{5, -7 - 2*s, 0}
		p.Lean = 1.5
		p.Open = true
		p.Bob = -1.2 * math.Abs(s)
	case AnimDance0: // fist pump
		p.LA = Limb{3, 4 - 10*math.Max(0, s), 2}
		p.RA = Limb{3, 4 - 10*math.Max(0, -s), 2}
		p.LF = Limb{2, 10 - 1.8*math.Max(0, s), 0}
		p.RF = Limb{2, 10 - 1.8*math.Max(0, -s), 0}
		p.Bob = 1.2 * math.Abs(c)
	case AnimDance1: // disco point
		p.RA = Limb{5.5 + 1.5*s, -8 + 2*c, 0}
		p.LA = Limb{4, 5, 1}
		p.Sway = 1.6 * s
		p.LF = Limb{2.5 + 1.5*s, 10, 0}
		p.RF = Limb{0.5 - 1.5*s, 10, 0}
		p.Bob = math.Abs(c)
	case AnimDance2: // twist
		p.LA = Limb{6 * math.Abs(c), 3 + 2*s, 3 * s}
		p.RA = Limb{6 * math.Abs(c), 3 - 2*s, -3 * s}
		p.Sway = 2 * s
		p.LF = Limb{2, 10, 3 * s}
		p.RF = Limb{2, 10, -3 * s}
	case AnimDance3: // headbang / bounce
		p.LA = Limb{4, 1 + 3*s, 2}
		p.RA = Limb{4, 1 - 3*s, 2}
		p.Bob = 2 * math.Abs(s)
		p.Lean = 2.5 * s
		p.LF = Limb{2.5, 10, 0}
		p.RF = Limb{2.5, 10, 0}
	case AnimDrink:
		u := 1 - math.Abs(2*phi-1) // 0..1..0
		u = math.Min(1, u*1.8)
		p.RA = Limb{0.5, lerp(8, -4.5, u), lerp(0, 3.5, u)}
		p.Lean = -1.2 * u
		p.Hold = HoldCup
	case AnimTalk:
		p.RA = Limb{5 + 1.5*s, 2 - 3*math.Abs(s), 3}
		p.LA = Limb{2, 8 - 1.2*math.Max(0, c), 1}
		p.Bob = 0.6 * math.Abs(c)
		p.Open = s > 0
	case AnimWave:
		p.RA = Limb{6 + 2*s, -7, 0}
		p.Open = true
	case AnimPunch:
		switch f {
		case 0:
			p.RA = Limb{1, 4, -5}
			p.Lean = -1
		case 1:
			p.RA = Limb{1, 1, 10}
			p.Lean = 2.5
		case 2:
			p.RA = Limb{1, 1, 9}
			p.Lean = 2
		default:
			p.RA = Limb{2, 6, 2}
		}
		p.LA = Limb{3, 6, 2}
	case AnimStab:
		p.Hold = HoldKnife
		switch f {
		case 0:
			p.RA = Limb{3, -2, 2}
			p.Lean = -1
		case 1:
			p.RA = Limb{1, 3, 10}
			p.Lean = 2.5
		default:
			p.RA = Limb{2, 5, 5}
			p.Lean = 1
		}
	case AnimFall:
		switch f {
		case 0:
			p.LA, p.RA = Limb{6, 1, 0}, Limb{6, 1, 0}
			p.Lean = -2
		case 1:
			p.LA, p.RA = Limb{6, -4, -2}, Limb{6, -4, -2}
			p.Lean = -5
			p.LF, p.RF = Limb{3, 10, 3}, Limb{3, 10, 3}
		default:
			p.LA, p.RA = Limb{6, 2, -4}, Limb{6, 2, -4}
			p.Lean = -8
			p.Bob = 6
			p.LF, p.RF = Limb{3, 7, 7}, Limb{3, 7, 7}
		}
	case AnimVomit:
		p.Lean = 7
		p.HeadZ = 3
		p.Bob = 2
		p.LA, p.RA = Limb{2, 8, 5}, Limb{2, 8, 5}
		p.LF, p.RF = Limb{3, 10, 2}, Limb{3, 10, 2}
		p.Open = true
		if f%2 == 1 {
			p.Lean = 8
			p.Bob = 3
		}
	case AnimTired:
		p.Lean = 4
		p.HeadZ = 1
		p.Bob = 1 + float64(f)
		p.LA, p.RA = Limb{2, 8, 4}, Limb{2, 8, 4}
		p.LF, p.RF = Limb{3, 10.5, 1}, Limb{3, 10.5, 1}
		p.Open = true
	case AnimKiss:
		p.Lean = 2
		p.HeadZ = 2 + float64(f)*0.4
		p.LA, p.RA = Limb{3, 3, 5}, Limb{3, 3, 5}
	case AnimDJ:
		p.LA = Limb{4, 7 + 1.5*s, 5 + 1.5*c}
		p.RA = Limb{4, 7 - 1.5*s, 5 - 1.5*c}
		p.Bob = math.Abs(s) * 1
		p.Lean = 1 + c
	case AnimDrum:
		p.Sit = true
		p.LA = Limb{5, 5 + 4*math.Max(0, s), 6}
		p.RA = Limb{5, 5 + 4*math.Max(0, -s), 6}
		p.LF = Limb{3, 8, 5}
		p.RF = Limb{3, 8 - 1.5*math.Max(0, c), 5}
		p.Bob = 4
	case AnimGuitar:
		p.Hold = HoldGuitar
		p.RA = Limb{2, 7 + 2*s, 5}
		p.LA = Limb{8, 4, 5}
		p.Sway = 1.2 * c
		p.LF = Limb{3, 10, 1}
		p.RF = Limb{2, 10, -1}
	case AnimSing:
		p.Hold = HoldMic
		p.RA = Limb{1, -4.5, 3}
		p.LA = Limb{5 + 2*s, -3 + 5*math.Abs(c), 1}
		p.Open = s > -0.3
		p.Sway = 0.8 * s
	case AnimCall:
		p.Hold = HoldPhone
		p.RA = Limb{1.5, -4.5, 0}
		p.Bob = float64(f) * 0.5
	case AnimCheer: // both pompoms up, a little jump on every beat
		p.Hold = HoldPompom
		p.Open = true
		switch f % 4 {
		case 0:
			p.LA, p.RA = Limb{6.5, -7.5, 0}, Limb{6.5, -7.5, 0}
			p.Bob = -1.6
			p.LF, p.RF = Limb{3, 9, 0}, Limb{3, 9, 0}
		case 1:
			p.LA, p.RA = Limb{7, -3, 0}, Limb{5, -8, 0}
			p.Sway = 1
		case 2:
			p.LA, p.RA = Limb{7, -8, 0}, Limb{7, -8, 0}
			p.Bob = -1.6
			p.LF, p.RF = Limb{3, 9, 0}, Limb{3, 9, 0}
		default:
			p.LA, p.RA = Limb{5, -8, 0}, Limb{7, -3, 0}
			p.Sway = -1
		}
	case AnimCrouch:
		p.Sit = true
		p.Lean = 3
		p.Bob = 5
		p.LA, p.RA = Limb{3, 7, 7 - float64(f)}, Limb{3, 7, 7 - float64(f)}
		p.LF, p.RF = Limb{3, 7, 5}, Limb{3, 7, 5}
	case AnimPoint:
		p.LA = Limb{3, 7, 1}
		switch f {
		case 0:
			p.RA = Limb{2, 6, 2}
		case 1:
			p.RA = Limb{1, 0, 9}
			p.Lean = 1
		default:
			p.RA = Limb{0.5, -1, 10}
			p.Lean = 1.5
		}
	case AnimLove:
		p.LA = Limb{2, 2, 5}
		p.RA = Limb{2, 2, 5}
		p.HeadZ = 1.5 + 0.5*math.Abs(s)
		p.Bob = 1.2 * math.Abs(s)
		p.Open = c > 0
	case AnimLaugh:
		p.Bob = 2 * math.Abs(s)
		p.Lean = -3 * math.Abs(s)
		p.LA = Limb{4, 3, 1}
		p.RA = Limb{4, 3, 1}
		p.Open = true
	case AnimPirouette:
		p.LA = Limb{7, 2, 0}
		p.RA = Limb{7, 2, 0}
		p.Lean = 3 * s
		p.Bob = -4 * math.Max(0, math.Sin(2*t))
		p.LF = Limb{1, 9, 2 * c}
		p.RF = Limb{3, 10, -2 * c}
	case AnimDance4: // breakdance
		p.Sit = true
		p.Bob = 6
		p.LF = Limb{5 * math.Abs(s), 6, 4 * s}
		p.RF = Limb{5 * math.Abs(c), 6, -4 * c}
		p.LA = Limb{3, 4, 6}
		p.RA = Limb{3, 4, 6}
		p.Lean = 4 * s
	case AnimDance5: // robot
		switch f {
		case 0:
			p.LA, p.RA = Limb{7, -6, 0}, Limb{2, 8, 0}
		case 1:
			p.LA, p.RA = Limb{2, 8, 0}, Limb{7, -6, 0}
		case 2:
			p.LA, p.RA = Limb{7, 2, 6}, Limb{2, 7, 0}
		case 3:
			p.LA, p.RA = Limb{2, 7, 0}, Limb{7, 2, 6}
		case 4:
			p.LA, p.RA = Limb{0, -8, 0}, Limb{0, -8, 0}
		default:
			p.LA, p.RA = Limb{4, 6, 0}, Limb{4, 6, 0}
		}
	case AnimDance6: // salsa
		p.Sway = 3 * s
		p.LA = Limb{3, 6, -2 * s}
		p.RA = Limb{3, 6, 2 * s}
		p.LF = Limb{2 + 1.5*s, 10, 0}
		p.RF = Limb{2 - 1.5*s, 10, 0}
	case AnimDance7: // sprinkler
		p.LA = Limb{2, 3, 4}
		p.RA = Limb{6 * s, -2, 3}
		p.Bob = 0.6 * math.Abs(c)
	case AnimDance8: // shopping cart
		p.LA = Limb{3, 2 + 3*s, 4}
		p.RA = Limb{3, 2 + 3*s, 4}
		p.Lean = -1.5
		walkLegs(&p, phi, 2.5, 1.5)
	case AnimDance9: // floss
		p.LA = Limb{2, 4, 6 * s}
		p.RA = Limb{2, 4, -6 * s}
		p.Sway = -2 * s
	case AnimDance10: // moonwalk
		walkLegs(&p, phi, 4, 1)
		p.Lean = 2
		p.LA = Limb{1.5, 9, 2 * s}
		p.RA = Limb{1.5, 9, -2 * s}
	case AnimDance11: // jazz hands
		p.LA = Limb{6, -6, 1}
		p.RA = Limb{6, -6, 1}
		p.Open = f%2 == 0
		p.Bob = 1.5 * math.Abs(s)
	}
	return p
}

// peePose stands in for poseFor on AnimPee: standing for a man, sitting
// (with a visible stream) for a woman. Gender is not part of Pose/poseFor's
// signature, so this is called directly from the few places that already
// special-case an animation by gender/pose (same pattern as AnimLying/AnimSleep).
func peePose(f int, female bool) Pose {
	p := Pose{LA: restArm(), RA: restArm(), LF: restFoot(), RF: restFoot(), Hold: HoldPee}
	if female {
		p.Sit = true
		p.Lean = 2
		p.LA = Limb{3, 7, 3}
		p.RA = Limb{3, 7, 3}
		p.LF = Limb{2.5, 8, 2}
		p.RF = Limb{2.5, 8, 2}
	} else {
		p.Lean = 1
		p.LA = Limb{2, 7, 1}
		p.RA = Limb{0.5, 2, 6}
	}
	p.Bob = 0.4 * float64(f%2)
	return p
}

// drawStream paints the pee effect: a short dribble for the sitting pose,
// an arcing stream down to the ground for the standing one. Not drawn from
// behind.
func drawStream(b *pixBuf, vw int, x, yTop, yBot float64, sit bool) {
	if vw == vBack {
		return
	}
	col := color.RGBA{0xe8, 0xd8, 0x50, 170}
	cI := func(f float64) int { return int(math.Floor(f)) }
	if sit {
		for y := cI(yTop); y <= cI(yTop)+3 && float64(y) <= yBot; y++ {
			b.set(cI(x), y, col)
		}
		return
	}
	steps := maxI(2, int(yBot-yTop))
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		yy := lerp(yTop, yBot, t)
		xx := x + 3*math.Sin(t*math.Pi/2)
		b.set(cI(xx), cI(yy), col)
	}
}

// ---------------------------------------------------------------------------
// Character description
// ---------------------------------------------------------------------------

const (
	TopTee = iota
	TopLong
	TopTank
	TopHoodie
	TopSuit
	TopDress
	TopSwim
	TopRobe
	TopHawaiian
	TopVest
	TopUniform
	TopBare
	TopArmor
)

const (
	BotPants = iota
	BotShorts
	BotTrunks
	BotNone
	BotSkirt
	BotFlare // bell-bottom trousers: the hem flares out over the shoe (70s disco)
)

const (
	HairShort = iota
	HairLong
	HairBald
	HairMohawk
	HairCurly
	HairPompadour
	HairSpiky
	HairHomer
	HairBun
	HairBob
	HairPony
	HairFry
	HairDreads
	HairBeehive
	HairStar
	HairQuiff
	HairBowl
	HairBuns
	HairMessy
	HairComb
	HairSide // short hair with a fringe swept to one side
	HairAfro // a big round afro
)

const (
	HatNone = iota
	HatCap
	HatPolice
	HatBeanie
	HatCowboy
	HatBowler
	HatTurban
	HatAntenna
	HatHelmet
	HatSombrero
	HatMedic
	HatFedora
	HatBeret
	HatCone
	HatMontera
	HatTyrol
	HatUshanka
	HatChef
	HatBucket
	HatVader
	HatTrooper
	HatWig
	HatCrown
	HatViking
	HatNinja
	HatPirate
	HatTopHat
	HatHardhat
	HatPartyCone
	HatHorns
	HatBunny
	HatTV
	HatPumpkin
	HatMoto
	HatFez
	HatWizard
	HatDino
	HatShark
	HatChicken
	HatUnicorn
	HatSamurai
	HatPig
	HatCat
	HatDog
	HatBird
)

const (
	GlassNone = iota
	GlassRound
	GlassShades
	GlassBig
	GlassWhite
	GlassNeon
)

const (
	BeardNone = iota
	BeardStubble
	BeardFull
	BeardMustache
	BeardSideburns
	BeardGoatee
)

const (
	BodyNormal = iota
	BodyBig
	BodySlim
	BodyRobot
	BodyFat
	BodyGiant
	BodyObese
)

const (
	HeadNormal = iota
	HeadBig
	HeadHuge
)

const (
	MaskPlain = iota
	MaskHockey
	MaskGhost
	MaskPig
	MaskClown
	MaskSkull
	MaskHood // a drawn-tight parka hood that sticks out in front of the face in profile
)

const (
	EyesNormal = iota
	EyesAlien
	EyesCyclops
)

type CharSpec struct {
	Name, DisplayName                       string
	Skin, Hair, Shirt, Shirt2, Pants, Shoes color.RGBA
	HatCol, Accent, Mask, Cape, Susp        color.RGBA
	Top, Bottom, Pattern                    int
	Hair2, Hat, Glasses, Beard, Body, Head  int
	Eyes, Ears, Stature, Carry              int
	CarryCol                                color.RGBA
	MaskKind, Weapon                        int
	Female                                  bool
	Bust, Hips                              int  // 0 none, 1 medium, 2 large, 3 exaggerated
	Makeup, Censor                          bool // red lips and eye shadow / mosaic censor bar
	Ride, Face                              int  // 1 scooter, 2 skateboard / 1 smile, 2 sly grin
	RideCol                                 color.RGBA
	noOverride                              bool // built from an asset: do not look for an edited drawing again
}

func hx(h uint32) color.RGBA { return rgb(h) }

func thickLine(b *pixBuf, x0, y0, x1, y1 float64, w int, col color.RGBA) {
	steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))*2) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Floor(x0 + (x1-x0)*t))
		y := int(math.Floor(y0 + (y1-y0)*t))
		for dx := 0; dx < w; dx++ {
			b.set(x+dx, y, col)
		}
	}
}

// ik2 returns the middle joint of a two-bone chain from a to target (bone length l each),
// bending toward the side given by (bx,by).
func ik2(ax, ay, tx, ty, l, bx, by float64) (mx, my, ex, ey float64) {
	dx, dy := tx-ax, ty-ay
	d := math.Hypot(dx, dy)
	if d > 2*l-0.05 {
		k := (2*l - 0.05) / d
		dx, dy = dx*k, dy*k
		d = 2*l - 0.05
	}
	ex, ey = ax+dx, ay+dy
	h := math.Sqrt(math.Max(0, l*l-d*d/4))
	px, py := -dy/math.Max(d, 0.001), dx/math.Max(d, 0.001)
	if px*bx+py*by < 0 {
		px, py = -px, -py
	}
	mx, my = ax+dx/2+px*h, ay+dy/2+py*h
	return
}

const (
	vFront = 0
	vBack  = 1
	vSide  = 2
)

func bodyWidth(b int) int {
	switch b {
	case BodySlim:
		return 3
	case BodyBig:
		return 7
	case BodyFat:
		return 9
	case BodyGiant:
		return 11
	case BodyObese:
		return 13
	}
	return 5
}

// genChar renders one frame at the original game's scale: about 10x25 pixels, no hard outline.
// trackParts makes genChar record which body part painted each pixel (for asset export).
var trackParts bool

func genChar(s *CharSpec, vw int, p Pose) *pixBuf {
	if isSponge(s) {
		return genSponge(s, vw, p)
	}
	if isJohnny5(s) { // Cortocircuito: robot on tracks (johnny5_art.go)
		return genJohnny5(s, vw, p)
	}
	if isCrash(s) { // Crash: cartoon bandicoot (crash_art.go)
		return genCrash(s, vw, p)
	}
	if isBean(s) { // the Fall Guys beans (fallguys.go)
		return genFallBean(s, vw, p)
	}
	if isTaz(s) { // the Tasmanian Devil (taz_art.go)
		return genTaz(s, vw, p)
	}
	if isTurtle(s) { // the four Ninja Turtles (turtles_art.go)
		return genTurtle(s, vw, p)
	}
	if isSimpson(s) { // the yellow family (simpsons_art.go)
		return genSimpson(s, vw, p)
	}
	if isTia(s) { // Mortadelo and Filemon (tia_art.go)
		return genTia(s, vw, p)
	}
	if isBenderArt(s) { // Bender (bender_art.go)
		return genBenderArt(s, vw, p)
	}
	if isHeroArt(s) { // Superman, Batman, Hulk, Captain America, Wolverine, Deadpool (hero_art.go)
		return genHero(s, vw, p)
	}
	if isRangerArt(s) { // the Power Rangers of the 5-star squad (rangers_art.go)
		return genRanger(s, vw, p)
	}
	if isZbp(s) { // Zipi, Zape, the Pink Panther and Mr Bean (zbp_art.go)
		return genZbp(s, vw, p)
	}
	if isDnh(s) { // Doraemon, Nobita, Harry, Hermione and Ron (dnh_art.go)
		return genDnh(s, vw, p)
	}
	if isNjg(s) { // Neo and the Joker (njg_art.go)
		return genNjg(s, vw, p)
	}
	if isCsm(s) { // Chewbacca, Steve, the plumbers, Gandalf, Frodo, Knuckles (csm_art.go)
		return genCsm(s, vw, p)
	}
	if isLmdy(s) { // Lara Croft, Marty McFly, Doc Brown and Yoda (lmdy_art.go)
		return genLmdy(s, vw, p)
	}
	p = opPose(s, p)
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	cx := charCX
	c0 := float64(cx) + 0.5
	robot := s.Body == BodyRobot
	robo := isRobocop(s)
	gaul := gaulKind(s)
	wally := isWally(s)
	vd := isVader(s)
	fk := foxKind(s)
	if fk == foxTwin && foxDancing(p) {
		p.Bob -= 2 + float64(p.F%2) // hovers while dancing
	}
	tw := bodyWidth(s.Body)
	if robot {
		tw = 5
	}
	if vw == vSide {
		tw = maxI(3, tw-1)
	}
	d := float64(s.Stature)
	bob := math.Round(p.Bob)
	if p.Sit {
		bob = math.Max(bob, 3)
	}
	hipY := 26 + d + bob
	torsoTop := hipY - 10
	shiftX := 0.0
	dip := 0.0
	if vw == vSide {
		shiftX = p.Lean * 0.7
	} else {
		dip = math.Abs(p.Lean) * 0.3
	}
	torsoTop += dip * 0.5
	sway := p.Sway * 0.7
	skinD := shade(s.Skin, 0.82)
	pantsD := shade(s.Pants, 0.72)
	cI := func(f float64) int { return int(math.Floor(f)) }

	legW := 2
	armW := 2
	switch s.Body {
	case BodySlim:
		legW, armW = 1, 1
	case BodyBig:
		legW, armW = 3, 2
	case BodyFat:
		legW, armW = 3, 3
	case BodyGiant:
		legW, armW = 4, 3
	case BodyObese:
		legW, armW = 5, 4
	}
	footRow := 41.0
	if s.Ride != 0 {
		footRow = 39 // the board takes the last rows
	}
	boneLeg := (footRow - hipY) / 2
	if p.Sit {
		boneLeg = 6.5
	}
	shY := torsoTop + 1.5

	longSleeve := s.Top == TopLong || s.Top == TopHoodie || s.Top == TopSuit || s.Top == TopRobe || s.Top == TopUniform || s.Top == TopArmor
	armUpper, armFore := s.Shirt, s.Skin
	switch s.Top {
	case TopTank, TopSwim, TopBare:
		armUpper = s.Skin
	}
	if longSleeve {
		armFore = s.Shirt
	}
	if s.Top == TopArmor {
		armFore = shade(s.Shirt, 0.9)
	}
	if robot {
		armUpper, armFore = hx(0x9aa2ae), hx(0x8a929e)
	}
	proj := func(l Limb, sign float64) (float64, float64) {
		switch vw {
		case vSide:
			return l.Z, l.Y
		case vBack:
			return -sign * l.X * 0.8, l.Y - l.Z*0.1
		}
		return sign * l.X * 0.8, l.Y + l.Z*0.1
	}
	drawArm := func(sx, sy float64, l Limb, sign float64, dark bool, id uint8) (float64, float64) {
		b.cur = id
		tx, ty := proj(l, sign)
		out := sign
		if vw == vSide {
			out = 0
		}
		// a hand close to the shoulder (arm folded or pointing at the viewer) shortens the
		// bones instead of folding them into a zigzag that reads as pieces of arm
		bl := math.Min(4.7, math.Max(2.6, math.Hypot(tx, ty)*0.62))
		mx, my, ex, ey := ik2(sx, sy, sx+tx, sy+ty, bl, out*0.25, 0.9)
		up, fo := armUpper, armFore
		if dark {
			up, fo = shade(up, 0.78), shade(fo, 0.78)
		}
		thickLine(b, sx, sy, mx, my, armW, up)
		thickLine(b, mx, my, ex, ey, armW, fo)
		hand := s.Skin
		if fk != foxNone {
			hand = hx(0xf4f4f4)
		}
		if robot {
			hand = hx(0x6a7280)
		}
		if s.Top == TopArmor {
			hand = shade(s.Shirt, 0.7)
		}
		if dark {
			hand = shade(hand, 0.8)
		}
		b.rect(cI(ex), cI(ey), armW, 2, hand)
		qzHand(b, s, vw, ex, ey, armW, dark, id)                // gloves Q-Z (charimp_q.go)
		dbArm(b, s, vw, sx, sy, mx, my, ex, ey, armW, dark, id) // Dragon Ball (dragonball_art.go)
		// ptf_art.go
		ptfHand(b, s, vw, ex, ey, armW, dark, id)
		ktmArm(b, s, vw, sx, sy, mx, my, ex, ey, armW, dark, id) // Kratos, Snake, endoskeleton (ktm_art.go)
		pnkArm(b, s, vw, sx, sy, mx, my, ex, ey, armW, dark, id) // punk bands (pnk_art.go)
		plagueArm(b, s, vw, sx, sy, mx, my, ex, ey, armW, dark, id)
		planetHand(b, s, vw, ex, ey, armW, dark, id) // planet_art.go
		// tattoos and wristbands of the bands (bandsb_art.go)
		bsbArm(b, s, vw, sx, sy, mx, my, ex, ey, armW, dark, id)
		if robo {
			roboArm(b, vw, sx, sy, mx, my, ex, ey, armW, dark, id)
		}
		return ex, ey
	}
	legCol, shinCol := s.Pants, s.Pants
	if s.Bottom == BotShorts || s.Bottom == BotTrunks || (s.Top == TopSwim && s.Female) {
		shinCol = s.Skin
	}
	if s.Top == TopDress || s.Bottom == BotSkirt {
		shinCol = s.Skin
		legCol = s.Skin
	}
	if s.Top == TopRobe {
		legCol, shinCol = s.Shirt, s.Shirt
	}
	if robot {
		legCol, shinCol = hx(0x8a929e), hx(0x7a828e)
	}
	if s.Bottom == BotNone && s.Top != TopRobe {
		legCol, shinCol = s.Skin, s.Skin
	}
	drawLeg := func(hxp float64, l Limb, sign float64, dark bool, id uint8) {
		b.cur = id
		tx, _ := proj(l, sign)
		if vw == vSide {
			tx = l.Z
		}
		fx := hxp + tx
		fy := footRow + (l.Y - 10)
		if p.Sit {
			fy = hipY + math.Min(l.Y-1, 10)
		}
		var mx, my, ex, ey float64
		if vw == vSide {
			mx, my, ex, ey = ik2(hxp, hipY, fx, fy, boneLeg, 1, 0)
		} else {
			mx, my, ex, ey = ik2(hxp, hipY, fx, fy, boneLeg, 0.05*sign, 1)
		}
		if p.Sit && vw != vSide {
			mx, my = hxp, hipY+3
		}
		lc, sc := legCol, shinCol
		if dark {
			lc, sc = shade(lc, 0.76), shade(sc, 0.76)
		}
		thickLine(b, hxp, hipY, mx, my, legW, lc)
		thickLine(b, mx, my, ex, ey, legW, sc)
		shoe := s.Shoes
		if dark {
			shoe = shade(shoe, 0.8)
		}
		sxx, syy := cI(ex), cI(ey)
		if s.Bottom == BotFlare && s.Top != TopDress && s.Top != TopRobe { // bell-bottoms
			if vw == vSide {
				b.rect(sxx-2, syy-3, legW+4, 3, sc)
			} else {
				b.rect(sxx-1, syy-3, legW+2, 3, sc)
			}
		}
		if vw == vSide {
			b.rect(sxx-1, syy, legW+2, 2, shoe)
		} else {
			b.rect(sxx, syy, legW, 2, shoe)
		}
		if vd {
			vaderLeg(b, vw, mx, my, sxx, syy, legW, shoe)
		}
		miLeg(b, s, vw, sign, sxx, syy, legW, dark)
		dbLeg(b, s, vw, mx, my, ex, ey, sxx, syy, legW, dark) // Dragon Ball boots (dragonball_art.go)
		// ptf_art.go
		ptfLeg(b, s, vw, sxx, syy, legW, dark)
		ktmLeg(b, s, vw, hxp, hipY, mx, my, ex, ey, sxx, syy, legW, dark) // ktm_art.go
		plagueLeg(b, s, vw, hxp, hipY, mx, my, ex, ey, sxx, syy, legW, dark)
		planetLeg(b, s, vw, sxx, syy, legW, dark)   // planet_art.go
		fbLeg(b, s, vw, mx, my, ex, ey, legW, dark) // football socks (futbol_art.go)
		if fk != foxNone {
			foxFoot(b, fk, vw, sxx, syy, legW, p)
		}
		if robo {
			roboLeg(b, vw, hxp, hipY, mx, my, ex, ey, legW, dark, sign)
		}
	}
	torsoRows := func() (int, int, int, int) {
		x0 := cI(c0 - float64(tw)/2 + sway + shiftX)
		return x0, x0 + tw, cI(torsoTop), cI(hipY)
	}
	var curves func()
	torso := func() {
		b.cur = 2
		x0, x1, y0, y1 := torsoRows()
		for y := y0; y < y1+1; y++ {
			xs := float64(y1-y) / float64(y1-y0) * shiftX * 0.5
			for x := x0 - int(xs); x < x1-int(xs); x++ {
				col := s.Shirt
				switch s.Top {
				case TopBare:
					col = s.Skin
				case TopSwim:
					col = s.Skin
					if s.Female && y >= y0+1 && y <= y0+2 {
						col = s.Shirt
					}
				case TopHawaiian:
					if ((x*3+y*5)%7 == 0) || ((x+y*2)%9 == 0) {
						col = s.Shirt2
					}
				case TopTee, TopLong, TopHoodie, TopTank, TopDress:
					switch s.Pattern {
					case 1:
						if (x*3+y*5)%8 == 0 {
							col = s.Shirt2
						}
					case 2:
						if y%3 == 0 {
							col = s.Shirt2
						}
					case 3:
						if (x+y)%4 == 0 && x%2 == 0 {
							col = s.Shirt2
						}
					}
				case TopVest:
					if y == y0+4 {
						col = hx(0xd8d8e0)
					}
				case TopArmor:
					if vd {
						col = vaderTorsoCol(x, y, x0, x1, y0, y1, vw)
					} else if y == y0+5 || (x == x0+tw/2) {
						col = shade(s.Shirt, 0.8)
					}
				}
				if robot {
					col = hx(0x9aa2ae)
					if (x+y)%5 == 0 {
						col = hx(0x7a828e)
					}
				}
				col = shade(col, 1.1-0.24*float64(y-y0)/float64(maxI(1, y1-y0)))
				if x == x1-1-int(xs) {
					col = shade(col, 0.82)
				}
				if y == y1 {
					col = shade(col, 0.78)
				}
				if y == y0 {
					col = shade(col, 1.14)
				}
				b.set(x, y, col)
			}
		}
		mid := cI(c0 + sway + shiftX*0.6)
		switch s.Top {
		case TopSuit:
			if vw != vBack {
				for i := 0; i < 4; i++ {
					b.set(mid, y0+i, s.Shirt2)
					if i < 2 {
						b.set(mid-1-i/2, y0+i, s.Shirt2)
						b.set(mid+1+i/2, y0+i, s.Shirt2)
					}
				}
				b.rect(mid, y0+1, 1, 5, s.Accent)
			}
			b.rect(x0, y1-1, tw, 1, shade(s.Shirt, 0.6))
		case TopUniform:
			if vw != vBack {
				b.rect(mid, y0+1, 1, 5, s.Accent)
				b.set(mid+2, y0+3, hx(0xe8c840))
			}
			b.rect(x0, y1-1, tw, 1, hx(0x14141a))
			b.set(mid, y1-1, hx(0xc8c8d0))
		case TopVest:
			if vw != vBack {
				b.rect(x0+1, y0, 1, y1-y0, hx(0xe8d020))
				b.rect(x1-2, y0, 1, y1-y0, hx(0xe8d020))
			}
		case TopHawaiian:
			if vw != vBack {
				b.set(mid, y0, s.Skin)
				b.set(mid, y0+1, s.Skin)
			}
		case TopHoodie:
			if vw == vBack {
				b.rect(x0+1, y0, tw-2, 2, shade(s.Shirt, 0.86))
			} else {
				b.rect(x0+1, y1-3, tw-2, 1, shade(s.Shirt, 0.8))
				b.set(mid-1, y0+1, hx(0xd8d8d8))
				b.set(mid+1, y0+1, hx(0xd8d8d8))
			}
		case TopTank:
			b.rect(x0, y0, 1, 2, s.Skin)
			b.rect(x1-1, y0, 1, 2, s.Skin)
		case TopArmor:
			if vd {
				vaderChest(b, vw, x0, x1, y0, y1, mid, p)
			} else if vw != vBack {
				b.rect(mid-1, y0+2, 3, 2, hx(0x2a2a30))
				b.set(mid, y0+2, hx(0xe03030))
			}
		}
		if fk != foxNone {
			foxBody(b, fk, vw, x0, x1, y0, y1, mid)
		}
		if !dbOwn(s) { // Dragon Ball guests are drawn in dragonball_art.go
			ciBody(b, s, vw, x0, x1, y0, y1, mid, p) // famous guests I-P (charimp_i.go)
		}
		if s.Susp.A != 0 && vw != vBack {
			b.rect(x0+1, y0, 1, y1-y0, s.Susp)
			b.rect(x1-2, y0, 1, y1-y0, s.Susp)
		}
		switch s.Bottom {
		case BotNone:
		default:
			if s.Top != TopDress && s.Top != TopRobe {
				b.rect(x0, y1-1, tw, 2, pantsD)
				if s.Bottom == BotTrunks {
					b.rect(x0, y1-1, tw, 3, s.Pants)
				}
			}
		}
		if s.Bottom == BotSkirt && s.Top != TopDress {
			for y := y1; y < y1+6; y++ {
				ex := (y - y1) / 2
				col := s.Pants
				if y%2 == 0 {
					col = shade(s.Pants, 1.15)
				}
				b.rect(x0-ex, y, tw+2*ex, 1, col)
			}
		}
		if s.Top == TopDress {
			hem := 9
			if s.Pattern == 3 {
				hem = 13
			}
			for y := y1; y < y1+hem; y++ {
				ex := (y - y1) / 2
				col := shade(s.Shirt, 1-0.02*float64(y-y1))
				if s.Pattern == 3 && (y+x0)%4 == 0 {
					col = s.Shirt2
				}
				b.rect(x0-ex, y, tw+2*ex, 1, col)
			}
			if s.Pattern == 3 {
				b.rect(x0-hem/2, y1+hem-1, tw+hem, 1, s.Shirt2)
			}
		}
		if s.Top == TopRobe {
			for y := y1; y < y1+14; y++ {
				b.rect(x0-1, y, tw+2, 1, shade(s.Shirt, 1-0.02*float64(y-y1)))
			}
		}
		if robo {
			roboTorso(b, s, vw, x0, x1, y0, y1, p)
		}
		if wally {
			wallyTorso(b, x0, x1, y0, y1, vw)
		}
		if gaul != gaulNone {
			gaulTorso(b, s, gaul, vw, x0, x1, y0, y1)
		}
		miTorso(b, s, vw, x0, x1, y0, y1, mid, p)
		if !dbOwn(s) {
			caTorso(b, s, vw, x0, x1, y0, y1, mid, p) // famous A-H details (charimp_a.go)
			qzTorso(b, s, vw, x0, x1, y0, y1, mid, p) // famous Q-Z details (charimp_q.go)
		}
		dbTorso(b, s, vw, x0, x1, y0, y1, mid, p) // Dragon Ball (dragonball_art.go)
		// Pope, Michael, Quijote, Sancho (ptf_art.go)
		ptfTorso(b, s, vw, x0, x1, y0, y1, mid, p)
		ktmTorso(b, s, vw, x0, x1, y0, y1, mid, p) // ktm_art.go
		pnkTorso(b, s, vw, x0, x1, y0, y1, mid, p) // punk bands (pnk_art.go)
		plagueTorso(b, s, vw, x0, x1, y0, y1, mid, p)
		planetTorso(b, s, vw, x0, x1, y0, y1, mid, p) // planet_art.go
		// band prints, belts, jackets (bandsb_art.go)
		bsbTorso(b, s, vw, x0, x1, y0, y1, mid, p)
		fbTorso(b, s, vw, x0, x1, y0, y1, mid, p) // national shirts and numbers (futbol_art.go)
		if s.Female && !robot {
			// curves: bust on the chest, wide hips under the waist. From the
			// front they are drawn after the arms so they stay visible.
			draw := func() {
				bc := s.Shirt
				if s.Top == TopBare {
					bc = s.Skin
				}
				hc := s.Pants
				if s.Top == TopDress {
					hc = s.Shirt
				}
				if s.Bottom == BotNone && s.Top != TopDress {
					hc = s.Skin
				}
				n := float64(s.Bust)
				if s.Bust > 0 && vw != vBack {
					if vw == vSide {
						b.ellipse(float64(x1)+n*0.5, float64(y0)+1.5+n*0.5, 1.2+n*0.9, 1.4+n*0.8, bc)
						b.set(x1+int(n), y0+1, shade(bc, 1.2))
					} else {
						for _, sx := range []float64{c0 - 0.5 - n*0.9 - 0.6, c0 + 0.5 + n*0.9 + 0.6} {
							b.ellipse(sx, float64(y0)+1.8+n*0.6, 1.5+n*0.8, 1.7+n*0.8, bc)
							b.ellipse(sx-0.4, float64(y0)+1.2+n*0.5, 0.7+n*0.3, 0.6+n*0.3, shade(bc, 1.2))
							b.rect(int(sx-1), y0+3+int(n*1.2), 3, 1, shade(bc, 0.75))
						}
						b.rect(int(c0)-1, y0+2, 1, 2+int(n), shade(bc, 0.7)) // cleavage
					}
				}
				if s.Hips > 0 {
					h := float64(s.Hips)
					if vw == vSide {
						b.ellipse(float64(x0)-0.5, float64(y1)+0.5, 1.2+h*1.1, 2.4+h*0.6, hc)
					} else {
						b.ellipse(c0, float64(y1)+1.3, float64(tw)/2+1.2+h*1.7, 2.3+h*0.7, hc)
						b.ellipse(c0, float64(y1)+0.2, float64(tw)/2+0.6+h*1.4, 1.2, shade(hc, 1.12))
					}
				}
			}
			if vw == vFront {
				curves = draw
			} else {
				draw()
			}
		}
		if s.Censor {
			mos := func(x, y, w, h int) {
				for yy := y; yy < y+h; yy++ {
					for xx := x; xx < x+w; xx++ {
						col := hx(0xb8b8c0)
						if (xx/2+yy/2)%2 == 0 {
							col = hx(0x6a6a74)
						}
						b.set(xx, yy, col)
					}
				}
			}
			mos(x0-1, y1-2, tw+2, 4)
			if s.Female {
				mos(x0, y0+1, tw, 3)
			}
		}
	}
	cape := func(front bool) {
		b.cur = 2
		if fk == foxTwin {
			foxTails(b, vw, c0+sway, hipY, p)
		}
		if s.Cape.A == 0 {
			return
		}
		x0, x1, y0, y1 := torsoRows()
		if vd {
			vaderCape(b, s, vw, front, x0, x1, y0, y1, p)
			return
		}
		for y := y0; y < y1+14; y++ {
			grow := (y - y0) / 4
			if vw == vSide {
				b.rect(x0-2-grow, y, 2+grow, 1, s.Cape)
			} else if front {
				b.set(x0-1-grow, y, s.Cape)
				b.set(x1+grow, y, s.Cape)
			} else {
				b.rect(x0-1-grow, y, tw+2+2*grow, 1, shade(s.Cape, 1-0.02*float64(y-y0)))
			}
		}
	}
	head := func() {
		b.cur = 1
		rx, ry := 2.6, 3.0
		switch s.Head {
		case HeadBig:
			rx, ry = 3.8, 4.0
		case HeadHuge:
			rx, ry = 4.8, 4.8
		}
		if robot {
			rx, ry = 2.8, 3.0
		}
		if s.Hat == HatVader || s.Hat == HatTrooper {
			rx, ry = 2.6, 3.0
		}
		hxp := c0 + sway + shiftX
		if vw == vSide {
			hxp += p.HeadZ * 0.7
		}
		hyp := torsoTop + 0.8 - ry + math.Abs(p.HeadZ)*0.25
		if vw == vSide {
			rx *= 0.92
		}
		skin := s.Skin
		hairC := s.Hair
		ihx := cI(hxp)
		top := cI(hyp - ry)
		topf := float64(top)
		// neck
		b.rect(ihx, cI(hyp+ry-1), 1, 2, skinD)
		b.rect(ihx-1, cI(hyp+ry-1), 3, 1, skinD)
		// volumes behind the head
		switch s.Hair2 {
		case HairLong:
			if vw == vFront { // seen from the front the hair falls down both sides over the shoulders; the middle (neck, torso, clothes) stays free
				x0, x1 := cI(hxp-rx-0.5), cI(hxp-rx-0.5)+cI(2*rx+1)
				b.rect(x0, cI(hyp), x1-x0, cI(ry), hairC)
				b.rect(x0, cI(hyp), 2, 11, hairC)
				b.rect(x1-2, cI(hyp), 2, 11, hairC)
			} else {
				b.rect(cI(hxp-rx-0.5), cI(hyp), cI(2*rx+1), 11, hairC)
			}
		case HairCurly:
			b.ellipse(hxp, hyp-0.4, rx+1.6, ry+1.2, hairC)
		case HairAfro:
			b.ellipse(hxp, hyp-1.3, rx+3.4, ry+2.9, hairC)
		case HairPony:
			if vw != vFront {
				b.rect(cI(hxp-rx-1), cI(hyp+1), 2, 8, hairC)
			}
		case HairDreads:
			for i := -2; i <= 2; i++ {
				b.rect(ihx+i*2-1+i/2, cI(hyp), 1, 8+(i*i)%3, hairC)
			}
		case HairBeehive:
			b.ellipse(hxp, hyp-ry-1.5, rx+0.4, 4.5, hairC)
		case HairStar:
			for _, o := range [][2]float64{{-4, -2}, {4, -2}, {-3, -4}, {3, -4}, {0, -5}} {
				b.line(ihx, cI(hyp-1), ihx+int(o[0]), cI(hyp+o[1]*1.1)-1, hairC)
				b.line(ihx+1, cI(hyp-1), ihx+1+int(o[0]), cI(hyp+o[1]*1.1)-1, hairC)
			}
		case HairMessy:
			b.ellipse(hxp, hyp-0.8, rx+2, ry+1, hairC)
		case HairBuns:
			b.ellipse(hxp-rx-1.2, hyp-0.6, 1.6, 1.6, hairC)
			b.ellipse(hxp+rx+1.2, hyp-0.6, 1.6, 1.6, hairC)
		}
		if s.Ears > 0 && vw != vSide {
			ey := cI(hyp)
			for k := 1; k <= s.Ears; k++ {
				b.set(cI(hxp-rx)-k, ey-k/2, skin)
				b.set(cI(hxp+rx)+k-1, ey-k/2, skin)
			}
		}
		// face
		switch {
		case robot:
			b.rect(cI(hxp-rx), cI(hyp-ry), cI(2*rx), cI(2*ry), hx(0x9aa2ae))
			b.rect(cI(hxp-rx), cI(hyp-ry), cI(2*rx), 1, hx(0xc0c8d4))
		case s.Mask.A != 0:
			b.ellipse(hxp, hyp, rx, ry, s.Mask)
		default:
			b.ellipse(hxp, hyp, rx, ry, skin)
		}
		hair := func(cond func(x, y int, dx, dy float64) bool, col color.RGBA) {
			fc := skin
			if s.Mask.A != 0 {
				fc = s.Mask
			}
			for y := cI(hyp - ry - 1); y <= cI(hyp+ry+1); y++ {
				for x := cI(hxp - rx - 1); x <= cI(hxp+rx+1); x++ {
					if b.get(x, y) == fc {
						if cond(x, y, float64(x)+0.5-hxp, float64(y)+0.5-hyp) {
							b.set(x, y, col)
						}
					}
				}
			}
		}
		if !robot && s.Mask.A == 0 {
			capCond := func(x, y int, dx, dy float64) bool {
				if vw == vBack {
					return dy < ry*0.6
				}
				if vw == vSide {
					return dy < -ry*0.2 || (dx < -rx*0.1 && dy < ry*0.45)
				}
				return dy < -ry*0.25 || (math.Abs(dx) > rx*0.8 && dy < ry*0.2)
			}
			switch s.Hair2 {
			case HairShort, HairLong, HairBob, HairBun, HairPony, HairPompadour, HairCurly, HairFry, HairSpiky, HairQuiff, HairBowl, HairBuns, HairMessy, HairDreads, HairComb, HairStar, HairBeehive, HairSide, HairAfro:
				hair(capCond, hairC)
				if s.Hair2 != HairBald {
					b.set(ihx-1, top+1, shade(hairC, 1.45))
					b.set(ihx, top, shade(hairC, 1.3))
				}
				switch s.Hair2 {
				case HairPompadour:
					b.rect(ihx-2, top-2, 5, 2, hairC)
					b.rect(ihx-1, top-3, 3, 1, shade(hairC, 1.4))
				case HairSide:
					if vw != vBack {
						// the fringe falls over the forehead, lower on the left than on the right
						hair(func(x, y int, dx, dy float64) bool { return dy < ry*(0.05-dx/rx*0.55) && dy < ry*0.2 }, hairC)
					}
				case HairQuiff:
					b.rect(ihx-3, top-1, 7, 2, hairC)
					b.rect(ihx+1, top-2, 4, 1, hairC)
				case HairBun:
					b.ellipse(hxp, hyp-ry-1, 1.6, 1.5, hairC)
				case HairSpiky:
					for i, h := range []int{2, 3, 2, 3, 2} {
						b.rect(ihx-4+i*2, top-h+1, 1, h, hairC)
					}
				case HairFry:
					for i, h := range []int{1, 3, 2, 4, 2} {
						b.rect(ihx-4+i*2, top-h+1, 2, h, hairC)
					}
				case HairBowl:
					hair(func(x, y int, dx, dy float64) bool { return dy < 0 }, hairC)
				case HairComb:
					b.rect(ihx-2, top, 5, 1, hairC)
				}
				if s.Hair2 == HairLong || s.Hair2 == HairBob {
					n := 11
					if s.Hair2 == HairBob {
						n = 5
					}
					switch vw {
					case vFront:
						b.rect(cI(hxp-rx), cI(hyp), 1, n, hairC)
						b.rect(cI(hxp+rx)-1, cI(hyp), 1, n, hairC)
					case vBack:
						b.rect(cI(hxp-rx), cI(hyp), cI(2*rx), n, hairC)
					default:
						b.rect(cI(hxp-rx), cI(hyp), 3, n, hairC)
					}
				}
			case HairMohawk:
				b.rect(ihx, top-3, 2, 6, hairC)
			case HairHomer:
				b.rect(ihx-1, top-2, 1, 2, hairC)
				b.rect(ihx+1, top-2, 1, 2, hairC)
				if vw != vBack {
					hair(func(x, y int, dx, dy float64) bool { return dy > ry*0.2 }, hx(0xd2b48c))
				}
			}
		}
		eye := hx(0x1a1420)
		ey := cI(hyp + 0.4)
		if vw == vFront {
			switch {
			case s.Eyes == EyesAlien:
				b.ellipse(hxp-1.6, hyp+0.4, 1.4, 1.2, hx(0x0a0a10))
				b.ellipse(hxp+1.6, hyp+0.4, 1.4, 1.2, hx(0x0a0a10))
			case s.Eyes == EyesCyclops:
				b.rect(ihx-1, ey-1, 3, 3, hx(0xffffff))
				b.set(ihx, ey, eye)
			case robot:
				b.rect(ihx-2, ey, 2, 2, hx(0x40e0ff))
				b.rect(ihx+1, ey, 2, 2, hx(0x40e0ff))
			case s.Mask.A != 0:
				maskFront(b, s, hxp, hyp, rx, ry, ihx, ey, skin)
			case s.Head >= HeadBig:
				b.rect(ihx-2, ey, 2, 2, hx(0xf4f4f4))
				b.rect(ihx+1, ey, 2, 2, hx(0xf4f4f4))
				b.set(ihx-1, ey+1, eye)
				b.set(ihx+1, ey+1, eye)
			default:
				b.set(ihx-1, ey, eye)
				b.set(ihx+1, ey, eye)
				if s.Female {
					b.set(ihx-1, ey-1, shade(hairC, 0.6))
					b.set(ihx+1, ey-1, shade(hairC, 0.6))
					b.set(ihx, ey+2, hx(0xc03050))
					if s.Makeup {
						b.rect(ihx-1, ey+2, 3, 1, hx(0xe8082a))
						b.set(ihx, ey+3, hx(0xc8002a))
						b.set(ihx-2, ey, hx(0x7a3ab0))
						b.set(ihx+2, ey, hx(0x7a3ab0))
						b.set(ihx-1, ey-1, hx(0x7a3ab0))
						b.set(ihx+1, ey-1, hx(0x7a3ab0))
					}
				}
			}
			if p.Open && s.Eyes != EyesAlien {
				b.set(ihx, ey+2, hx(0x5a1a1a))
			}
			if s.Face != 0 && s.Mask.A == 0 && !p.Open {
				my := ey + 2
				if s.Head >= HeadBig {
					my = ey + 3
				}
				mc := shade(s.Skin, 0.55)
				if s.Face == 1 {
					b.set(ihx-2, my-1, mc)
					b.rect(ihx-1, my, 3, 1, hx(0xf8f0f0)) // a toothy smile
					b.set(ihx+2, my-1, mc)
				} else { // a sly little smirk, lifted on one side
					b.rect(ihx-1, my, 3, 1, hx(0xf8f0f0))
					b.set(ihx+2, my-1, mc)
				}
			}
		} else if vw == vSide {
			ex := cI(hxp + rx*0.45)
			switch {
			case s.Eyes == EyesAlien:
				b.ellipse(hxp+1.4, hyp+0.4, 1.4, 1.2, hx(0x0a0a10))
			case robot:
				b.rect(ex, ey, 2, 2, hx(0x40e0ff))
			case s.Mask.A != 0:
				maskSide(b, s, hxp, hyp, rx, ry, ex, ey)
			default:
				b.set(ex, ey, eye)
			}
			if s.Mask.A == 0 {
				b.set(cI(hxp+rx), ey+1, skinD)
				if s.Makeup && s.Female {
					b.set(cI(hxp+rx), ey+2, hx(0xe8082a))
				}
			}
			if p.Open {
				b.set(ex, ey+2, hx(0x5a1a1a))
			}
		}
		if vw != vBack && !robot && s.Mask.A == 0 {
			low := func(x, y int, dx, dy float64) bool { return dy > ry*0.2 }
			switch s.Beard {
			case BeardStubble:
				hair(func(x, y int, dx, dy float64) bool { return dy > ry*0.25 && (x+y)%2 == 0 }, shade(mixc(skin, hairC, 0.55), 0.9))
			case BeardFull:
				hair(low, hairC)
				if vw == vFront {
					b.set(ihx, ey+2, skinD)
				}
			case BeardMustache:
				b.rect(ihx-1, ey+1, 3, 1, hairC)
			case BeardSideburns:
				b.rect(cI(hxp-rx), ey, 1, 3, hairC)
				b.rect(cI(hxp+rx)-1, ey, 1, 3, hairC)
			case BeardGoatee:
				b.rect(ihx, ey+2, 1, 2, hairC)
				b.rect(ihx-1, ey+1, 3, 1, hairC)
			}
		}
		if vw != vBack {
			switch s.Glasses {
			case GlassShades:
				b.rect(cI(hxp-rx), ey, cI(2*rx), 2, hx(0x0c0c10))
			case GlassRound:
				b.rect(ihx-2, ey-1, 2, 3, hx(0x303040))
				b.rect(ihx+1, ey-1, 2, 3, hx(0x303040))
				b.set(ihx-2, ey, hx(0xf0f0f8))
				b.set(ihx+1, ey, hx(0xf0f0f8))
			case GlassWhite:
				b.rect(cI(hxp-rx), ey, cI(2*rx), 2, hx(0xf2f2f2))
				b.set(ihx-1, ey, hx(0x303038))
				b.set(ihx+1, ey, hx(0x303038))
			case GlassNeon:
				b.rect(cI(hxp-rx), ey, cI(2*rx), 2, hx(0x30f0e0))
				b.rect(cI(hxp-rx), ey+1, cI(2*rx), 1, hx(0xf040c0))
			case GlassBig:
				b.rect(ihx-3, ey-1, 3, 3, hx(0x20202a))
				b.rect(ihx+1, ey-1, 3, 3, hx(0x20202a))
				b.set(ihx-2, ey, hx(0xf4f4f8))
				b.set(ihx+2, ey, hx(0xf4f4f8))
			}
		}
		hc := s.HatCol
		rxi := cI(rx)
		switch s.Hat {
		case HatCap:
			b.ellipse(hxp, hyp-0.9, rx+0.6, ry-1.2, hc)
			if vw == vFront {
				b.rect(ihx-rxi-1, cI(hyp-0.6), 2*rxi+3, 1, shade(hc, 0.6))
			} else if vw == vSide {
				b.rect(ihx, cI(hyp-0.6), rxi+3, 1, shade(hc, 0.6))
			}
		case HatBeanie:
			b.ellipse(hxp, hyp-0.9, rx+0.5, ry-1, hc)
			b.rect(ihx-rxi, cI(hyp-0.8), 2*rxi+1, 1, shade(hc, 1.25))
			b.set(ihx, top-1, shade(hc, 1.3))
		case HatPolice, HatMedic:
			b.rect(ihx-rxi-1, top-1, 2*rxi+3, 3, hc)
			b.rect(ihx-rxi-1, top+1, 2*rxi+3, 1, shade(hc, 0.65))
			if s.Hat == HatPolice {
				b.rect(ihx-rxi-1, top+2, 2*rxi+3, 1, hx(0x0c0c10))
				if vw == vFront {
					b.set(ihx, top, hx(0xe8c840))
				}
			} else {
				b.rect(ihx, top-1, 1, 2, hx(0xe03030))
				b.rect(ihx-1, top, 3, 1, hx(0xe03030))
			}
		case HatCowboy:
			b.ellipse(hxp, topf+2.2, rx+3.6, 1.2, hc)
			b.rect(ihx-rxi, top-2, 2*rxi+1, 4, hc)
			b.rect(ihx-rxi, top+1, 2*rxi+1, 1, shade(hc, 0.6))
		case HatBowler:
			b.ellipse(hxp, topf+2.2, rx+1.6, 0.9, hx(0x14141a))
			b.ellipse(hxp, topf+1, rx-0.2, 2.2, hx(0x14141a))
			b.rect(ihx-rxi+1, top+1, 2*rxi-1, 1, hx(0x3a3a46))
		case HatTurban:
			b.ellipse(hxp, hyp-1.2, rx+1, ry-0.2, hc)
			b.rect(ihx-rxi-1, cI(hyp-1), 2*rxi+3, 1, shade(hc, 0.82))
			b.line(ihx-rxi, top+1, ihx+rxi, cI(hyp-1), shade(hc, 0.8))
			b.rect(ihx, top-2, 2, 2, hc)
		case HatAntenna:
			b.rect(ihx, top-4, 1, 5, hx(0x6a7280))
			b.rect(ihx-1, top-6, 3, 2, hx(0xe03030))
		case HatHelmet:
			b.ellipse(hxp, hyp-0.8, rx+0.8, ry-0.5, hc)
			b.rect(ihx-rxi-1, top+3, 2*rxi+3, 1, shade(hc, 0.7))
		case HatSombrero:
			b.ellipse(hxp, topf+2.4, rx+5.6, 1.5, hc)
			b.ellipse(hxp, topf+0.4, rx-0.4, 2.6, hc)
			b.rect(ihx-rxi, top+1, 2*rxi+1, 1, hx(0xd04040))
		case HatFedora:
			b.ellipse(hxp, topf+2.2, rx+2.4, 1.1, hc)
			b.rect(ihx-rxi, top-1, 2*rxi+1, 3, hc)
			b.rect(ihx-rxi, top+1, 2*rxi+1, 1, shade(hc, 0.55))
			b.rect(ihx, top-1, 1, 1, shade(hc, 0.7))
		case HatBeret:
			b.ellipse(hxp+0.6, topf+1, rx+1.4, 1.8, hc)
			b.set(ihx+1, top-1, hc)
		case HatCone:
			for i := 0; i < 7; i++ {
				w := 1 + i
				b.rect(ihx-w+1, top-5+i, 2*w, 1, hc)
			}
			b.rect(ihx-rxi-3, top+2, 2*rxi+7, 1, shade(hc, 0.8))
		case HatMontera:
			b.ellipse(hxp-1.6, topf+0.4, 2, 1.8, hx(0x14141a))
			b.ellipse(hxp+1.6, topf+0.4, 2, 1.8, hx(0x14141a))
			b.rect(ihx-rxi, top+1, 2*rxi+1, 2, hx(0x14141a))
		case HatTyrol:
			b.ellipse(hxp, topf+1.6, rx+1.2, 1, hc)
			b.rect(ihx-rxi, top-2, 2*rxi+1, 3, hc)
			b.line(ihx+rxi, top-1, ihx+rxi+2, top-4, hx(0xd8e0f0))
		case HatUshanka:
			b.ellipse(hxp, hyp-0.6, rx+1, ry-0.2, hc)
			b.rect(ihx-rxi-1, cI(hyp), 2, 4, hc)
			b.rect(ihx+rxi, cI(hyp), 2, 4, hc)
		case HatChef:
			b.rect(ihx-rxi, top-1, 2*rxi+1, 2, hx(0xf4f4f8))
			b.ellipse(hxp, topf-2, rx+0.8, 2.6, hx(0xf4f4f8))
		case HatBucket:
			b.ellipse(hxp, topf+2, rx+2, 1, hc)
			b.ellipse(hxp, topf+0.6, rx, 2, hc)
		case HatWig:
			b.ellipse(hxp, hyp-1, rx+1.1, ry-0.2, hc)
		case HatCrown:
			b.rect(ihx-rxi, top-1, 2*rxi+1, 3, hx(0xf0c030))
			b.rect(ihx-rxi, top-1, 2*rxi+1, 1, hx(0xffe070))
			for _, dx := range []int{-rxi, 0, rxi} {
				b.rect(ihx+dx, top-3, 1, 2, hx(0xf0c030))
			}
			b.set(ihx, top, hx(0xd02030))
		case HatViking:
			b.ellipse(hxp, hyp-1.2, rx+0.6, ry-1.2, hx(0x9aa0a8))
			b.rect(ihx-rxi, top+2, 2*rxi+1, 1, hx(0x6a7078))
			b.line(ihx-rxi-1, top+2, ihx-rxi-4, top-4, hx(0xe8e0c8))
			b.line(ihx+rxi+1, top+2, ihx+rxi+4, top-4, hx(0xe8e0c8))
			b.set(ihx-rxi-4, top-5, hx(0xe8e0c8))
			b.set(ihx+rxi+4, top-5, hx(0xe8e0c8))
		case HatNinja:
			b.ellipse(hxp, hyp, rx+0.7, ry+0.5, hc)
			b.rect(ihx-rxi, ey-1, 2*rxi+1, 3, skin)
			b.set(ihx-1, ey, hx(0x1a1420))
			b.set(ihx+1, ey, hx(0x1a1420))
			b.rect(ihx-rxi-1, top+2, 2*rxi+3, 1, hx(0xc82a2a))
		case HatPirate:
			b.ellipse(hxp, topf+1.4, rx+3.4, 2, hx(0x14141a))
			b.rect(ihx-rxi, top-2, 2*rxi+1, 3, hx(0x14141a))
			b.set(ihx, top-1, hx(0xf0f0e8))
			b.set(ihx, top, hx(0xf0f0e8))
			b.rect(ihx-rxi-3, top+1, 2, 2, hx(0x14141a))
			b.rect(ihx+rxi+2, top+1, 2, 2, hx(0x14141a))
		case HatTopHat:
			b.rect(ihx-rxi, top-6, 2*rxi+1, 7, hx(0x16161c))
			b.rect(ihx-rxi, top-2, 2*rxi+1, 1, hc)
			b.ellipse(hxp, topf+1.2, rx+2.2, 1, hx(0x16161c))
		case HatHardhat:
			b.ellipse(hxp, hyp-1.2, rx+1.2, ry-0.8, hx(0xf0c820))
			b.rect(ihx-rxi-1, top+2, 2*rxi+3, 1, hx(0xd0a810))
			b.rect(ihx, top-1, 1, 3, hx(0xf8e060))
		case HatPartyCone:
			for i := 0; i < 8; i++ {
				w := 4 - i/2
				col := hc
				if i%2 == 0 {
					col = shade(hc, 1.3)
				}
				b.rect(ihx-w/2, top-6+i, maxI(1, w), 1, col)
			}
			b.set(ihx, top-7, hx(0xf0e040))
		case HatHorns:
			b.line(ihx-rxi, top+1, ihx-rxi-2, top-4, hx(0xc82020))
			b.line(ihx+rxi, top+1, ihx+rxi+2, top-4, hx(0xc82020))
			b.line(ihx-rxi+1, top+1, ihx-rxi-1, top-4, hx(0xe84040))
			b.line(ihx+rxi-1, top+1, ihx+rxi+1, top-4, hx(0xe84040))
		case HatBunny:
			b.rect(ihx-2, top-8, 2, 8, hc)
			b.rect(ihx+1, top-8, 2, 8, hc)
			b.rect(ihx-2, top-6, 1, 5, hx(0xf0a0b0))
			b.rect(ihx+1, top-6, 1, 5, hx(0xf0a0b0))
		case HatTV:
			b.rect(ihx-rxi-1, top-2, 2*rxi+3, cI(2*ry)+3, hx(0x30303a))
			b.rect(ihx-rxi, top-1, 2*rxi+1, cI(2*ry)+1, hx(0x7ab0c0))
			for i := 0; i < 6; i++ {
				b.set(ihx-rxi+(i*5)%(2*rxi+1), top-1+(i*3)%cI(2*ry), hx(0xf0f8ff))
			}
			b.rect(ihx-1, top-5, 1, 3, hx(0x30303a))
			b.rect(ihx+2, top-5, 1, 3, hx(0x30303a))
		case HatPumpkin:
			b.ellipse(hxp, hyp, rx+1.2, ry+0.6, hx(0xe8802a))
			b.rect(ihx, top-2, 1, 3, hx(0x4a7a2a))
			b.set(ihx-1, ey, hx(0x2a1408))
			b.set(ihx+1, ey, hx(0x2a1408))
			b.rect(ihx-1, ey+2, 3, 1, hx(0x2a1408))
		case HatMoto:
			b.ellipse(hxp, hyp-0.4, rx+1.2, ry+0.4, hc)
			b.rect(ihx-rxi, ey-1, 2*rxi+1, 3, hx(0x1a1a24))
			b.rect(ihx-rxi, ey-1, 2*rxi+1, 1, hx(0x5a6a8a))
		case HatFez:
			b.rect(ihx-rxi+1, top-2, 2*rxi-1, 4, hx(0xc82a2a))
			b.rect(ihx-rxi+1, top-2, 2*rxi-1, 1, hx(0xe85a5a))
			b.line(ihx+rxi-1, top-2, ihx+rxi+1, top+3, hx(0xf0c030))
		case HatWizard:
			for i := 0; i < 11; i++ {
				w := 1 + i/2
				b.rect(ihx-w+1, top-9+i, 2*w-1, 1, hc)
			}
			b.rect(ihx-rxi-2, top+1, 2*rxi+5, 1, shade(hc, 0.8))
			b.set(ihx, top-6, hx(0xf0e040))
		case HatDino:
			b.ellipse(hxp, hyp-0.6, rx+2.2, ry+0.6, hx(0x4aa840))
			b.ellipse(hxp+0.5, hyp+2.4, rx+1.4, 1.8, hx(0x6ac860))
			b.rect(ihx-rxi, ey-2, 2, 2, hx(0xf4f4f0))
			b.rect(ihx+rxi-1, ey-2, 2, 2, hx(0xf4f4f0))
			b.set(ihx-rxi, ey-1, hx(0x14141a))
			b.set(ihx+rxi, ey-1, hx(0x14141a))
			for i := -rxi; i <= rxi; i += 2 {
				b.set(ihx+i, ey+3, hx(0xf4f4f0))
			}
			for i := 0; i < 3; i++ {
				b.rect(ihx-2+i*2, top-2, 1, 2, hx(0x2a8a2a))
			}
		case HatShark:
			b.ellipse(hxp, hyp-0.6, rx+2.4, ry+0.4, hx(0x7a8aa0))
			b.ellipse(hxp, hyp+1.6, rx+1.6, ry-1, hx(0xe8eef4))
			b.rect(ihx-rxi, ey-1, 2, 2, hx(0xf4f4f0))
			b.rect(ihx+rxi-1, ey-1, 2, 2, hx(0xf4f4f0))
			b.set(ihx-rxi, ey, hx(0x14141a))
			b.set(ihx+rxi, ey, hx(0x14141a))
			for i := -rxi; i <= rxi; i++ {
				if i%2 == 0 {
					b.set(ihx+i, ey+3, hx(0xf4f4f0))
				}
			}
			b.line(ihx, top-1, ihx+1, top-5, hx(0x5a6a80))
			b.line(ihx+1, top-1, ihx+2, top-5, hx(0x5a6a80))
		case HatChicken:
			b.ellipse(hxp, hyp-0.4, rx+2, ry+0.6, hx(0xf6f4ee))
			b.rect(ihx-1, top-2, 3, 2, hx(0xd83030))
			b.rect(ihx-1, ey+1, 3, 2, hx(0xf0b020))
			b.set(ihx-rxi, ey-1, hx(0x14141a))
			b.set(ihx+rxi, ey-1, hx(0x14141a))
		case HatPig:
			b.ellipse(hxp, hyp-0.6, rx+0.8, ry-0.6, hc)
			b.rect(ihx-rxi, top-1, 2, 3, shade(hc, 0.8)) // floppy ears
			b.rect(ihx+rxi-1, top-1, 2, 3, shade(hc, 0.8))
			if vw == vFront {
				b.rect(ihx-1, ey+1, 3, 2, shade(hc, 1.12)) // snout
				b.set(ihx-1, ey+2, hx(0x6a2a3a))
				b.set(ihx+1, ey+2, hx(0x6a2a3a))
			}
		case HatCat:
			b.ellipse(hxp, hyp-1, rx+0.6, ry-1, hc)
			b.rect(ihx-rxi, top-3, 2, 3, hc) // pointed ears
			b.rect(ihx+rxi-1, top-3, 2, 3, hc)
			b.set(ihx-rxi, top-4, hc)
			b.set(ihx+rxi, top-4, hc)
			b.set(ihx-rxi, top-2, hx(0xf0a0b0))
			b.set(ihx+rxi, top-2, hx(0xf0a0b0))
			if vw == vFront {
				b.set(ihx, ey+1, hx(0xf0a0b0)) // nose
			}
		case HatDog:
			b.ellipse(hxp, hyp-0.8, rx+0.6, ry-0.8, hc)
			b.rect(ihx-rxi-1, top, 2, 5, shade(hc, 0.7)) // long floppy ears
			b.rect(ihx+rxi, top, 2, 5, shade(hc, 0.7))
			if vw == vFront {
				b.set(ihx, ey+1, hx(0x14141a)) // black nose
				b.rect(ihx-1, ey+2, 3, 1, hx(0xe8d0b0))
			}
		case HatBird:
			b.ellipse(hxp, hyp-1, rx+0.4, ry-1, hc)
			b.rect(ihx, top-4, 1, 4, hx(0xf0c030)) // crest feathers
			b.rect(ihx+1, top-5, 1, 4, hx(0xe85a30))
			b.rect(ihx-1, top-3, 1, 3, hx(0x40b0e0))
			if vw == vFront {
				b.rect(ihx-1, ey+1, 3, 1, hx(0xf0a020)) // little beak
				b.set(ihx, ey+2, hx(0xf0a020))
			}
		case HatUnicorn:
			b.ellipse(hxp, hyp-1.2, rx+0.6, ry-1.2, hx(0xf4f0f8))
			b.line(ihx, top, ihx, top-6, hx(0xf0d040))
			b.line(ihx+1, top, ihx+1, top-5, hx(0xf8e880))
			b.rect(ihx-rxi, top, 1, 3, hx(0xf060b0))
			b.rect(ihx+rxi, top, 1, 3, hx(0x60b0f0))
		case HatSamurai:
			b.ellipse(hxp, hyp-1.2, rx+1.6, ry-1, hx(0x2a2a34))
			b.rect(ihx-rxi-2, top+2, 2*rxi+5, 1, hx(0x8a2a2a))
			b.line(ihx-rxi, top, ihx-rxi-3, top-3, hx(0xf0c030))
			b.line(ihx+rxi, top, ihx+rxi+3, top-3, hx(0xf0c030))
		case HatVader, HatTrooper:
			if s.Hat == HatVader {
				vaderHelmet(b, vw, hxp, hyp, rx, ry, hc)
				break
			}
			b.ellipse(hxp, hyp, rx+0.9, ry+0.6, hc)
			if vw != vBack {
				b.rect(ihx-1, cI(hyp+0.2), 3, 1, hx(0x0a0a0e))
				if s.Hat == HatVader {
					b.rect(ihx-1, cI(hyp+2), 3, 1, hx(0xa0a0a8))
				} else {
					b.rect(ihx-1, cI(hyp+2), 3, 1, hx(0x30303a))
				}
			}
		}
		if wally {
			wallyHead(b, s, vw, ihx, top, ey, cI(rx), p)
		}
		if gaul != gaulNone {
			gaulHead(b, s, gaul, vw, ihx, top, ey, cI(rx), p)
		}
		if !noCharExtras {
			charExtras(b, s, vw, p, hxp, hyp, rx, ry, ey)
		}
		opHead(b, s, vw, p, hxp, hyp, rx, ry, ey)
		miHead(b, s, vw, p, hxp, hyp, rx, ry, ey)
		if !dbOwn(s) {
			caHead(b, s, vw, p, hxp, hyp, rx, ry, ey) // famous A-H details (charimp_a.go)
			ciHead(b, s, vw, p, hxp, hyp, rx, ry, ey) // famous I-P details (charimp_i.go)
			qzHead(b, s, vw, p, hxp, hyp, rx, ry, ey) // famous Q-Z details (charimp_q.go)
		}
		dbHead(b, s, vw, p, hxp, hyp, rx, ry, ey) // Dragon Ball (dragonball_art.go)
		// ptf_art.go
		ptfHead(b, s, vw, p, hxp, hyp, rx, ry, ey)
		ktmHead(b, s, vw, p, hxp, hyp, rx, ry, ey) // ktm_art.go
		bsbHead(b, s, vw, p, hxp, hyp, rx, ry, ey) // band fringes, crest, cap (bandsb_art.go)
		fbHead(b, s, vw, p, hxp, hyp, rx, ry, ey)  // footballers' hair (futbol_art.go)
		pnkHead(b, s, vw, p, hxp, hyp, rx, ry, ey) // punk bands (pnk_art.go)
		plagueHead(b, s, vw, p, hxp, hyp, rx, ry, ey)
		if robo {
			roboHead(b, s, vw, ihx, top, p)
			if roboFiring(p) {
				_, rx1, _, ry1 := torsoRows()
				roboFlash(b, vw, rx1, ry1)
			}
		}
		if fk != foxNone {
			foxHead(b, fk, vw, hxp, hyp, rx, ry, p)
		}
	}
	held := func(hx0, hy0 float64, id uint8) {
		b.cur = id
		kind := p.Hold
		if wally && kind == HoldNone {
			wallyCane(b, hx0, hy0, p)
		}
		if gaul == gaulObelix && kind == HoldKnife {
			kind = HoldNone // a bare fist, with stars
		}
		if robo && kind == HoldKnife {
			kind = HoldGun // his shot instead of a knife
			if vw == vSide {
				hx0 -= 3 // keeps the pistol inside the sprite box
			}
		}
		if kind == HoldNone {
			kind = s.Carry
		}
		if s.Weapon != 0 && (kind == HoldKnife || kind == HoldNone) {
			kind = s.Weapon
		}
		if opHeld(b, s, kind, vw, hx0, hy0, p) {
			return
		}
		if ptfHeld(b, s, kind, vw, hx0, hy0) { // the Pope's cross staff (ptf_art.go)
			return
		}
		if qzHeld(b, s, kind, vw, hx0, hy0, p) { // Thor's hammer, Zorro's rapier (charimp_q.go)
			return
		}
		if ktmHeld(b, s, kind, vw, hx0, hy0) { // Kratos's axe (ktm_art.go)
			return
		}
		if plagueHeld(b, s, kind, vw, hx0, hy0) { // the beak doctor's staff (plague_killer.go)
			return
		}
		if vd && (kind == HoldSaber || kind == HoldKnife) {
			vaderSaber(b, vw, hx0, hy0, p)
			return
		}
		drawHeld(b, kind, s.CarryCol, vw, hx0, hy0)
	}
	// assemble in draw order
	shL := Vec{float64(cx - (tw-1)/2 - armW), shY}
	shR := Vec{float64(cx + (tw-1)/2 + 1), shY}
	shL.X += sway
	shR.X += sway
	hpL := float64(cx - legW)
	hpR := float64(cx + 1)
	switch vw {
	case vSide:
		sx := float64(cx-1) + sway + shiftX
		hp := float64(cx - 1)
		cape(false)
		drawLeg(hp, p.LF, -1, true, 5)
		lsx, lsy := drawArm(sx, shY, p.LA, -1, true, 3)
		if p.Hold == HoldPompom {
			held(lsx, lsy, 3)
		}
		drawLeg(hp, p.RF, 1, false, 6)
		torso()
		if p.Hold == HoldGuitar {
			b.cur = 2
			b.ellipse(c0+1+shiftX, hipY-2, 3.4, 3, hx(0xa8602c))
			b.line(cI(c0+3+shiftX), cI(hipY-2), cI(c0+10), cI(hipY-8), hx(0x5a3418))
		}
		head()
		hx0, hy0 := drawArm(sx, shY, p.RA, 1, false, 4)
		held(hx0, hy0, 4)
	case vBack:
		drawLeg(hpL, p.RF, 1, false, 5)
		drawLeg(hpR, p.LF, -1, false, 6)
		torso()
		cape(false)
		hxL, hyL := drawArm(shL.X, shL.Y, p.RA, 1, false, 3)
		held(hxL, hyL, 3)
		rx0, ry0 := drawArm(shR.X, shR.Y, p.LA, -1, false, 4)
		if p.Hold == HoldPompom {
			held(rx0, ry0, 4)
		}
		head()
	default:
		cape(true)
		drawLeg(hpL, p.LF, -1, false, 5)
		drawLeg(hpR, p.RF, 1, false, 6)
		torso()
		if p.Hold == HoldGuitar {
			b.cur = 2
			b.ellipse(c0+sway, hipY-2, 3.6, 3, hx(0xa8602c))
			b.set(cI(c0+sway), cI(hipY-2), hx(0x2a1a10))
			b.line(cI(c0+sway), cI(hipY-2), cI(c0+8), cI(hipY-9), hx(0x5a3418))
		}
		lx0, ly0 := drawArm(shL.X, shL.Y, p.LA, -1, false, 3)
		if p.Hold == HoldPompom {
			held(lx0, ly0, 3)
		}
		hx0, hy0 := drawArm(shR.X, shR.Y, p.RA, 1, false, 4)
		if curves != nil {
			curves()
		}
		head()
		held(hx0, hy0, 4)
	}
	if p.Hold == HoldPee {
		drawStream(b, vw, c0+sway+shiftX, hipY+2, footRow, p.Sit)
	}
	if s.Ride != 0 {
		drawRide(b, s, vw, charCX, int(hipY)-2)
	}
	opBody(b, s, vw, p, c0, hipY, shY, tw, sway)
	dbOver(b, s, vw, p, c0+sway+shiftX, hipY, torsoTop, tw) // Dragon Ball pads and capes (dragonball_art.go)
	return b
}

// drawRide paints the scooter or skateboard under the feet (the rider stands still on it).
func drawRide(b *pixBuf, s *CharSpec, vw, cx, handY int) {
	deck := s.RideCol
	if deck.A == 0 {
		deck = hx(0x3a8ae0)
	}
	wheel, metal := hx(0xf0f0f0), hx(0x9a9aa6)
	if vw == vSide {
		b.rect(cx-6, 41, 13, 2, deck)
		b.rect(cx-6, 41, 13, 1, shade(deck, 1.25))
		b.rect(cx-5, 43, 2, 1, wheel)
		b.rect(cx+4, 43, 2, 1, wheel)
		if s.Ride == 1 { // scooter: stem at the front with a handlebar
			for y := handY; y < 41; y++ {
				b.set(cx+6, y, metal)
			}
			b.rect(cx+5, handY, 3, 1, hx(0x3a3a46))
		} else { // skateboard: upturned ends
			b.set(cx-7, 40, deck)
			b.set(cx+7, 40, deck)
		}
		return
	}
	b.rect(cx-4, 41, 9, 2, deck)
	b.rect(cx-4, 41, 9, 1, shade(deck, 1.25))
	b.set(cx-4, 43, wheel)
	b.set(cx+4, 43, wheel)
	if s.Ride == 1 && vw == vFront {
		b.rect(cx-4, handY, 9, 1, hx(0x3a3a46))
		b.set(cx, handY+1, metal)
	}
}

// outlinedSel keeps the silhouette exactly as painted, like the original sprites (no extra outline pixels),
// and gives the edges depth instead: the sides facing away from the light darken, the lit ones brighten a bit.
// The factor only controls how strong the darkening is.
func (b *pixBuf) outlinedSel(f float64) *pixBuf {
	o := &pixBuf{w: b.w, h: b.h, p: append([]byte(nil), b.p...)}
	if b.pid != nil {
		o.pid = append([]uint8(nil), b.pid...)
	}
	k := 0.55 + f*0.5
	alpha := func(x, y int) uint8 {
		if x < 0 || y < 0 || x >= b.w || y >= b.h {
			return 0
		}
		return b.p[(y*b.w+x)*4+3]
	}
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			i := (y*b.w + x) * 4
			if b.p[i+3] == 0 {
				continue
			}
			m := 1.0
			if alpha(x, y+1) == 0 {
				m *= k
			}
			if alpha(x+1, y) == 0 {
				m *= (k + 1) / 2
			}
			if alpha(x, y-1) == 0 {
				m *= 1.07
			}
			if alpha(x-1, y) == 0 {
				m *= 1.04
			}
			if m == 1.0 {
				continue
			}
			cl := func(v float64) uint8 {
				if v > 255 {
					return 255
				}
				return uint8(v)
			}
			o.p[i], o.p[i+1], o.p[i+2] = cl(float64(b.p[i])*m), cl(float64(b.p[i+1])*m), cl(float64(b.p[i+2])*m)
		}
	}
	return o
}

func renderChar(s *CharSpec, vw int, p Pose) *pixBuf {
	return genChar(s, vw, p).outlinedSel(0.42)
}

func (b *pixBuf) rotCCW() *pixBuf {
	o := newPix(b.h, b.w)
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			si := (y*b.w + x) * 4
			di := ((b.w-1-x)*o.w + y) * 4
			copy(o.p[di:di+4], b.p[si:si+4])
		}
	}
	return o
}

// CharSet lazily renders and caches every animation frame of one character.
type CharSet struct {
	Spec  *CharSpec
	cache map[int]*ebiten.Image
	rig   *Rig
	over  map[string]*pixBuf
}

func newCharSet(s *CharSpec) *CharSet {
	if !s.noOverride && s.Skin.A == 0 {
		// a nearly empty spec (copied from an asset-drawn character, whose spec only carries the name)
		// would paint an invisible body: build the character's own look instead
		var sp CharSpec
		if f := presetByName(s.Name); s.Name != "" && f != nil {
			sp = f(rand.New(rand.NewSource(7)))
		} else {
			sp = genericSpec(rand.New(rand.NewSource(7)))
			sp.Name = s.Name
		}
		sp.Carry, sp.CarryCol = s.Carry, s.CarryCol
		s = &sp
	}
	if !s.noOverride && s.Name != "" {
		if cs := overrideCharSet(s); cs != nil {
			return cs
		}
	}
	return &CharSet{Spec: s, cache: map[int]*ebiten.Image{}}
}

func (cs *CharSet) frameImg(anim, vw, f int) *ebiten.Image {
	key := anim*1000 + vw*100 + f
	if im, ok := cs.cache[key]; ok {
		return im
	}
	if cs.rig != nil {
		im := cs.rigFrame(anim, vw, f).img()
		cs.cache[key] = im
		return im
	}
	var b *pixBuf
	switch anim {
	case AnimLying:
		sp := *cs.Spec
		sp.Ride = 0
		b = genChar(&sp, 0, poseFor(AnimIdle, 0)).rotCW()
	case AnimSleep:
		sp := *cs.Spec
		sp.Ride = 0
		b = genChar(&sp, 0, poseFor(AnimIdle, 1)).rotCCW()
	case AnimPee:
		b = genChar(cs.Spec, vw, peePose(f, cs.Spec.Female))
	default:
		pp := poseFor(anim, f)
		pp.Anim, pp.F = anim, f
		b = genChar(cs.Spec, vw, pp)
	}
	im := b.outlinedSel(0.42).img()
	cs.cache[key] = im
	return im
}

// image for an animation at a given phase and facing; flip = draw mirrored.
func (cs *CharSet) img(anim int, d Dir, phase float64) (*ebiten.Image, bool) {
	if cs.Spec != nil && cs.Spec.Ride != 0 && (anim == AnimWalk || anim == AnimRun) {
		anim, phase = AnimIdle, 0 // riders glide: legs stay still on the board
	}
	n := animFrameCount(anim)
	f := int(math.Floor(phase)) % n
	if f < 0 {
		f += n
	}
	switch d {
	case DirUp:
		return cs.frameImg(anim, int(vBack), f), false
	case DirLeft:
		return cs.frameImg(anim, int(vSide), f), true
	case DirRight:
		return cs.frameImg(anim, int(vSide), f), false
	}
	return cs.frameImg(anim, int(vFront), f), false
}

func (cs *CharSet) lying(sleep bool) *ebiten.Image {
	if sleep {
		return cs.frameImg(AnimSleep, 0, 0)
	}
	return cs.frameImg(AnimLying, 0, 0)
}

// drawHeld paints a carried item at the hand position.
func drawHeld(b *pixBuf, kind int, col color.RGBA, vw int, hx0, hy0 float64) {
	cI := func(f float64) int { return int(math.Floor(f)) }
	x, y := cI(hx0), cI(hy0)
	fwd := 1
	switch kind {
	case HoldKnife:
		dx, dy := 5, 0
		if vw != vSide {
			dx, dy = 0, 4
			if vw == vBack {
				dy = -4
			}
		}
		b.line(x+1, y+1, x+1+dx, y+1+dy, hx(0xe8eef8))
		b.set(x+1, y+1, hx(0x3a3a46))
	case HoldCup:
		b.rect(x, y-2, 2, 3, hx(0xe03a3a))
		b.set(x, y-2, hx(0xf0c0c0))
	case HoldMic:
		b.rect(x, y-2, 2, 2, hx(0x20202a))
		b.rect(x-1, y-4, 3, 2, hx(0x707080))
	case HoldPhone:
		b.rect(x, y-1, 2, 3, hx(0x101018))
	case HoldBaguette:
		b.line(x, y+1, x+fwd*0+3, y-8, hx(0xd8a850))
		b.line(x+1, y+1, x+4, y-8, hx(0xb88838))
	case HoldTray:
		b.rect(x-3, y-2, 8, 1, hx(0xc8c8d0))
		b.rect(x-1, y-4, 2, 2, hx(0xf0f0f0))
		b.rect(x+2, y-3, 1, 1, hx(0xe04040))
	case HoldCamera:
		b.rect(x-1, y-2, 4, 3, hx(0x2a2a30))
		b.set(x, y-1, hx(0xa8c8e8))
	case HoldBigCup:
		b.rect(x-1, y-4, 4, 5, hx(0xe03a3a))
		b.rect(x-1, y-4, 4, 1, hx(0xf4f4f4))
		b.set(x+1, y-6, hx(0x4a4a50))
	case HoldSaber:
		if col.A == 0 {
			col = hx(0x40a0ff)
		}
		b.line(x+1, y, x+1, y-11, col)
		b.line(x+2, y, x+2, y-11, shade(col, 1.3))
		b.rect(x, y, 3, 2, hx(0x707080))
	case HoldBeer:
		b.rect(x, y-3, 2, 4, hx(0xe8b830))
		b.rect(x, y-3, 2, 1, hx(0xf4f4f4))
	case HoldFlag:
		b.rect(x+1, y-12, 1, 12, hx(0x6a4a2a))
		b.rect(x+2, y-12, 6, 4, hx(0xd02828))
		b.rect(x+2, y-10, 6, 1, hx(0xf0c030))
	case HoldPizza:
		b.rect(x-2, y-3, 6, 4, hx(0xc8a060))
		b.set(x, y-2, hx(0xd03030))
	case HoldSign:
		b.rect(x+1, y-10, 1, 10, hx(0x6a4a2a))
		b.rect(x-2, y-14, 8, 5, hx(0xf4f4f0))
		b.rect(x-1, y-12, 6, 1, hx(0xd03030))
	case HoldMachete:
		b.rect(x, y-1, 2, 3, hx(0x4a2c14))
		b.line(x+1, y+1, x+1+sideDX(vw, 8), y+1+frontDY(vw, 8), hx(0xd0d6e0))
		b.line(x+2, y+1, x+2+sideDX(vw, 8), y+1+frontDY(vw, 8), hx(0x9aa2b0))
	case HoldChainsaw:
		b.rect(x-2, y-2, 6, 4, hx(0xd86a14))
		b.rect(x-2, y-2, 6, 1, hx(0xf0a040))
		b.rect(x+3, y-3, 1, 2, hx(0x2a2a30))
		b.line(x+1, y+2, x+1+sideDX(vw, 11), y+2+frontDY(vw, 11), hx(0x9aa2ae))
		b.line(x+2, y+2, x+2+sideDX(vw, 11), y+2+frontDY(vw, 11), hx(0x4a4a54))
	case HoldClaw:
		for k := 0; k < 4; k++ {
			b.line(x+k-1, y+1, x+k-1+sideDX(vw, 6), y+1+frontDY(vw, 6), hx(0xd8dee8))
		}
		b.rect(x-1, y, 4, 2, hx(0x6a4a2a))
	case HoldBroom:
		b.line(x+1, y-12, x+1, y+3, hx(0x8a5a2e))
		b.rect(x-1, y+3, 5, 4, hx(0xd8b050))
	case HoldStick:
		b.line(x+1, y-13, x+1, y+4, hx(0xc8a060))
		b.rect(x+1, y+4, 5, 2, hx(0x6a4a2a))
		b.rect(x+1, y-13, 1, 3, hx(0xe8e8f0))
	case HoldVacuum:
		b.rect(x-2, y-1, 6, 4, hx(0x7a8a9a))
		b.set(x, y, hx(0xe03030))
		b.line(x+3, y-1, x+2, y-8, hx(0x4a4a50))
	case HoldSelfie:
		b.line(x+1, y, x+3, y-15, hx(0x4a4a50))
		b.rect(x+2, y-19, 3, 4, hx(0x101018))
	case HoldPompom:
		if col.A == 0 {
			col = hx(0xf0d030)
		}
		b.ellipse(float64(x)+0.5, float64(y)-3, 3.3, 3.6, col)
		b.ellipse(float64(x), float64(y)-4, 1.6, 1.8, shade(col, 1.35))
		b.set(x-3, y-6, shade(col, 1.2))
		b.set(x+3, y-5, shade(col, 0.8))
		b.set(x-2, y, shade(col, 0.8))
		b.set(x+2, y+1, col)
	case HoldPlant:
		b.rect(x-1, y-1, 4, 3, hx(0xb4572e))
		b.rect(x-2, y-5, 6, 4, hx(0x3a9a44))
		b.rect(x, y-7, 2, 2, hx(0x5ac060))
	case HoldTrophy:
		b.rect(x-1, y-5, 4, 4, hx(0xf0c030))
		b.rect(x, y-1, 2, 2, hx(0xd8a820))
		b.rect(x-1, y+1, 4, 1, hx(0x6a4a2a))
	case HoldBalloons:
		b.line(x+1, y, x+1, y-12, hx(0xd8d8d8))
		for k, col := range []uint32{0xe03030, 0x3060e0, 0xf0c030} {
			b.ellipse(float64(x-2+k*3)+1, float64(y-15-k%2), 2, 2.4, hx(col))
		}
	case HoldMegaphone:
		b.rect(x-1, y-2, 3, 3, hx(0xc8c8d0))
		b.rect(x+2, y-3, 4, 5, hx(0xd83030))
	case HoldBriefcase:
		b.rect(x-2, y+1, 7, 5, hx(0x6a4a2a))
		b.rect(x-2, y+1, 7, 1, hx(0x8a6a44))
		b.rect(x, y, 3, 1, hx(0x3a2a1a))
	case HoldTorch:
		b.line(x+1, y, x+1, y-8, hx(0x6a4a2a))
		b.rect(x, y-12, 3, 4, hx(0xff8a20))
		b.rect(x+1, y-14, 1, 2, hx(0xffd860))
	case HoldFloat:
		if col.A == 0 {
			col = hx(0xe03030)
		}
		for k := 0; k < 12; k++ {
			a := float64(k) / 12 * 2 * math.Pi
			px, py := x+1+int(math.Round(math.Cos(a)*4)), y-4+int(math.Round(math.Sin(a)*4))
			cc := col
			if k%3 == 0 {
				cc = hx(0xf4f4f4)
			}
			b.rect(px, py, 2, 2, cc)
		}
	case HoldBoard:
		if col.A == 0 {
			col = hx(0xe03030)
		}
		b.rect(x, y-16, 4, 18, col)
		b.rect(x, y-16, 4, 2, shade(col, 1.2))
		b.rect(x+1, y-9, 2, 3, hx(0xf4f4f4))
		b.rect(x, y-3, 4, 1, hx(0xf4f4f4))
	case HoldExt:
		b.rect(x, y-4, 3, 6, hx(0xd02a2a))
		b.rect(x, y-4, 3, 1, hx(0x2a2a30))
		b.rect(x+2, y-6, 3, 1, hx(0x2a2a30))
		b.set(x+1, y-1, hx(0xf4f4f4))
	case HoldGun:
		b.rect(x, y-3, 5, 2, hx(0x2a2a32))
		b.rect(x, y-1, 2, 3, hx(0x3a3a46))
		b.set(x+4, y-3, hx(0x8a8a96))
	case HoldRifle:
		drawRifle(b, vw, x, y)
	case HoldSax:
		b.line(x+1, y-8, x+1, y+1, hx(0xe8b830))
		b.line(x+1, y+1, x+4, y+3, hx(0xe8b830))
		b.rect(x+4, y+2, 2, 3, hx(0xf4d860))
		b.rect(x, y-9, 3, 1, hx(0x2a2a30))
	}
}

func sideDX(vw, n int) int {
	if vw == vSide {
		return n
	}
	return 0
}

func frontDY(vw, n int) int {
	switch vw {
	case vSide:
		return 0
	case vBack:
		return -n
	}
	return n
}

// maskFront paints the face of a mask (wrestler, hockey, ghost, pig, clown, skull) in the front view.
func maskFront(b *pixBuf, s *CharSpec, hxp, hyp, rx, ry float64, ihx, ey int, skin color.RGBA) {
	white := hx(0xf4f4f0)
	black := hx(0x101014)
	switch s.MaskKind {
	case MaskHockey:
		b.rect(ihx-2, ey, 2, 1, black)
		b.rect(ihx+1, ey, 2, 1, black)
		b.set(ihx-2, ey+2, black)
		b.set(ihx+2, ey+2, black)
		b.set(ihx, ey+3, black)
		b.set(ihx-1, ey-1, hx(0xc82020))
		b.set(ihx+1, ey-1, hx(0xc82020))
		b.rect(ihx, ey-2, 1, 3, hx(0xc82020))
	case MaskGhost:
		b.ellipse(hxp, hyp+ry*0.55+1, rx-0.2, 2.6, s.Mask)
		b.rect(ihx-2, ey, 2, 3, black)
		b.rect(ihx+1, ey, 2, 3, black)
		b.rect(ihx-1, ey+4, 3, 3, black)
	case MaskPig:
		b.rect(ihx-1, ey+1, 3, 2, shade(s.Mask, 0.75))
		b.set(ihx-1, ey+1, black)
		b.set(ihx+1, ey+1, black)
		b.set(ihx-2, ey-1, black)
		b.set(ihx+2, ey-1, black)
		b.rect(ihx-3, int(hyp-ry)-1, 2, 2, shade(s.Mask, 0.85))
		b.rect(ihx+2, int(hyp-ry)-1, 2, 2, shade(s.Mask, 0.85))
	case MaskClown:
		b.set(ihx-2, ey-1, hx(0x3060e0))
		b.set(ihx+2, ey-1, hx(0x3060e0))
		b.rect(ihx-1, ey, 3, 2, hx(0xe02020))
		b.rect(ihx-2, ey+3, 5, 1, hx(0xe02020))
		b.set(ihx-3, ey+2, hx(0xe02020))
		b.set(ihx+3, ey+2, hx(0xe02020))
	case MaskSkull:
		b.rect(ihx-2, ey, 2, 2, black)
		b.rect(ihx+1, ey, 2, 2, black)
		b.set(ihx, ey+2, black)
		for i := -2; i <= 2; i++ {
			if i%2 == 0 {
				b.set(ihx+i, ey+4, black)
			}
		}
	default:
		b.set(ihx-1, ey, white)
		b.set(ihx+1, ey, white)
		if s.Hat != HatTrooper && s.Hat != HatVader {
			b.set(ihx, ey+2, skin)
		}
	}
}

func maskSide(b *pixBuf, s *CharSpec, hxp, hyp, rx, ry float64, ex, ey int) {
	black := hx(0x101014)
	switch s.MaskKind {
	case MaskGhost:
		b.ellipse(hxp, hyp+ry*0.55+1, rx-0.3, 2.6, s.Mask)
		b.rect(ex, ey, 1, 3, black)
		b.rect(ex, ey+4, 2, 3, black)
	case MaskHood:
		// the hood puffs out in front of the face, like the cartoon's
		b.ellipse(hxp+rx+0.6, hyp+0.4, 2.6, ry*0.62, s.Mask)
		b.ellipse(hxp+rx+1.2, hyp+0.4, 1.4, ry*0.38, shade(s.Mask, 0.7))
		b.set(ex, ey, hx(0xf4f4f4))
		b.set(ex+1, ey, black)
	case MaskPig:
		b.rect(ex+1, ey+1, 2, 2, shade(s.Mask, 0.75))
		b.set(ex, ey, black)
	case MaskHockey, MaskSkull:
		b.set(ex, ey, black)
		b.set(ex, ey+2, black)
	case MaskClown:
		b.set(ex+1, ey+1, hx(0xe02020))
		b.set(ex, ey, hx(0x3060e0))
	default:
		b.set(ex, ey, hx(0xf4f4f4))
	}
}
