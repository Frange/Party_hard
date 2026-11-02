package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

var animNames = [AnimCount]string{
	AnimIdle: "idle", AnimWalk: "walk", AnimRun: "run", AnimDance0: "dance0", AnimDance1: "dance1", AnimDance2: "dance2", AnimDance3: "dance3",
	AnimDance4: "dance4", AnimDance5: "dance5", AnimDance6: "dance6", AnimDance7: "dance7",
	AnimDance8: "dance8", AnimDance9: "dance9", AnimDance10: "dance10", AnimDance11: "dance11",
	AnimDrink: "drink", AnimTalk: "talk", AnimWave: "wave", AnimPunch: "punch", AnimStab: "stab", AnimFall: "fall", AnimVomit: "vomit",
	AnimKiss: "kiss", AnimDJ: "dj", AnimDrum: "drum", AnimGuitar: "guitar", AnimSing: "sing", AnimCarry: "carry", AnimTired: "tired",
	AnimLying: "lying", AnimSleep: "sleep", AnimCall: "call", AnimPanic: "panic", AnimCrouch: "crouch",
	AnimPoint: "point", AnimLove: "love", AnimLaugh: "laugh", AnimPirouette: "pirouette", AnimPee: "pee", AnimZombie: "zombie", AnimLean: "lean",
}

var carryNames = map[string]int{
	"knife": HoldKnife, "cup": HoldCup, "mic": HoldMic, "phone": HoldPhone, "baguette": HoldBaguette, "tray": HoldTray,
	"camera": HoldCamera, "bigcup": HoldBigCup, "saber": HoldSaber, "beer": HoldBeer, "flag": HoldFlag, "pizza": HoldPizza, "sign": HoldSign,
	"machete": HoldMachete, "chainsaw": HoldChainsaw, "claw": HoldClaw, "broom": HoldBroom, "vacuum": HoldVacuum, "selfie": HoldSelfie, "plant": HoldPlant,
	"sax": HoldSax, "trophy": HoldTrophy, "balloons": HoldBalloons, "megaphone": HoldMegaphone, "briefcase": HoldBriefcase, "torch": HoldTorch,
}

// ---------------------------------------------------------------------------
// Rig: animates a character painted in its rest pose, part by part.
// Every part is rotated around its joint (shoulder, hip) or moved (head, torso),
// using the same pose functions as the built-in characters.
// ---------------------------------------------------------------------------

type rigLayer struct {
	x0, y0, w, h int
	pix          []color.RGBA
	px, py       float64 // joint in canvas coordinates
	length       float64
	ok           bool
}

type rigView struct {
	L [7]rigLayer
}

type Rig struct {
	W, H  int
	V     [3]*rigView
	feetY float64
	topY  int
	carry int
}

func buildRig(ca *CharAsset) *Rig {
	r := &Rig{W: ca.Meta.Width, H: ca.Meta.Height, carry: carryNames[strings.ToLower(ca.Meta.Carry)], topY: 999}
	maxFeet := 0
	for vi, v := range ca.Views {
		rv := &rigView{}
		r.V[vi] = rv
		for id := 1; id <= 6; id++ {
			x0, y0, x1, y1 := v.w, v.h, -1, -1
			for y := 0; y < v.h; y++ {
				for x := 0; x < v.w; x++ {
					if v.pid[y*v.w+x] == uint8(id) && v.p[(y*v.w+x)*4+3] > 0 {
						x0, y0 = minI(x0, x), minI(y0, y)
						x1, y1 = maxI(x1, x), maxI(y1, y)
					}
				}
			}
			if x1 < 0 {
				continue
			}
			l := &rv.L[id]
			l.ok, l.x0, l.y0, l.w, l.h = true, x0, y0, x1-x0+1, y1-y0+1
			l.pix = make([]color.RGBA, l.w*l.h)
			for y := y0; y <= y1; y++ {
				for x := x0; x <= x1; x++ {
					if v.pid[y*v.w+x] == uint8(id) {
						i := (y*v.w + x) * 4
						if v.p[i+3] > 0 {
							l.pix[(y-y0)*l.w+(x-x0)] = color.RGBA{v.p[i], v.p[i+1], v.p[i+2], 255}
						}
					}
				}
			}
			cxm := float64(x0) + float64(l.w)/2
			switch id {
			case PartHead:
				l.px, l.py = cxm, float64(y1+1)
			case PartTorso:
				l.px, l.py = cxm, float64(y1+1)
			default: // arms and legs hang from the top
				l.px, l.py = cxm, float64(y0)
				l.length = float64(l.h)
			}
			if id >= PartLegL && y1+1 > maxFeet {
				maxFeet = y1 + 1
			}
			if y0 < r.topY {
				r.topY = y0
			}
		}
		// in profile the far arm and leg are usually hidden behind the near ones: borrow them
		if vi == 1 {
			for _, pr := range [][2]int{{PartArmL, PartArmR}, {PartLegL, PartLegR}} {
				if !rv.L[pr[0]].ok && rv.L[pr[1]].ok {
					src := rv.L[pr[1]]
					dup := src
					dup.pix = make([]color.RGBA, len(src.pix))
					for i, c := range src.pix {
						if c.A != 0 {
							dup.pix[i] = shade(c, 0.74)
						}
					}
					rv.L[pr[0]] = dup
				}
			}
		}
		// the torso hides parts of arms and legs in profile: fill the gaps behind them
		t := &rv.L[PartTorso]
		if t.ok {
			for y := 0; y < t.h; y++ {
				for x := 0; x < t.w; x++ {
					if t.pix[y*t.w+x].A != 0 {
						continue
					}
					id := v.pid[(y+t.y0)*v.w+x+t.x0]
					if id < PartArmL {
						continue
					}
					// copy the nearest torso colour of this row
					for d := 1; d < t.w; d++ {
						if x-d >= 0 && t.pix[y*t.w+x-d].A != 0 {
							t.pix[y*t.w+x] = t.pix[y*t.w+x-d]
							break
						}
						if x+d < t.w && t.pix[y*t.w+x+d].A != 0 {
							t.pix[y*t.w+x] = t.pix[y*t.w+x+d]
							break
						}
					}
				}
			}
		}
	}
	if maxFeet == 0 {
		maxFeet = r.H - 1
	}
	r.feetY = float64(maxFeet)
	return r
}

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (l *rigLayer) at(x, y int) color.RGBA {
	if x < 0 || y < 0 || x >= l.w || y >= l.h {
		return color.RGBA{}
	}
	return l.pix[y*l.w+x]
}

// blitRot draws a limb rotated by theta (0 = hanging straight down) and shortened by k.
func (r *Rig) blitRot(dst *pixBuf, l *rigLayer, pdx, pdy, theta, k float64) {
	if !l.ok {
		return
	}
	cs, sn := math.Cos(theta), math.Sin(theta)
	lx, ly := l.px-float64(l.x0), l.py-float64(l.y0)
	// forward splat with sub-samples: a one pixel wide limb stays connected at any angle
	const sub = 4
	for y := 0; y < l.h; y++ {
		for x := 0; x < l.w; x++ {
			c := l.pix[y*l.w+x]
			if c.A == 0 {
				continue
			}
			for j := 0; j < sub; j++ {
				for i := 0; i < 1; i++ {
					sx := float64(x) + 0.5 - lx
					sy := (float64(y) + (float64(j)+0.5)/sub - ly) * k
					dx := sx*cs + sy*sn
					dy := -sx*sn + sy*cs
					dst.set(int(math.Floor(pdx+dx)), int(math.Floor(pdy+dy)), c)
				}
			}
		}
	}
}

func (r *Rig) blitShift(dst *pixBuf, l *rigLayer, dx, dy float64, shear float64) {
	if !l.ok {
		return
	}
	for y := 0; y < l.h; y++ {
		off := shear * (1 - float64(y)/float64(maxI(1, l.h-1)))
		for x := 0; x < l.w; x++ {
			c := l.pix[y*l.w+x]
			if c.A != 0 {
				dst.set(l.x0+x+int(math.Round(dx+off)), l.y0+y+int(math.Round(dy)), c)
			}
		}
	}
}

// render builds one frame of the animation pose for a view (0 front, 1 side, 2 back).
// assetView maps the renderer's view numbering (0 front, 1 back, 2 side) to the asset order (front, side, back).
func assetView(vw int) int {
	switch vw {
	case vSide:
		return 1
	case vBack:
		return 2
	}
	return 0
}

func (r *Rig) render(vw int, p Pose) *pixBuf {
	b := newPix(r.W, r.H)
	v := r.V[assetView(vw)]
	bob := math.Round(p.Bob)
	if p.Sit {
		bob = math.Max(bob, 3)
	}
	shiftX, dip := 0.0, 0.0
	if vw == vSide {
		shiftX = p.Lean * 0.7
	} else {
		dip = math.Abs(p.Lean) * 0.3
	}
	sway := p.Sway * 0.7
	torsoDX, torsoDY := sway+shiftX*0.5, bob+dip*0.5
	headDX, headDY := sway+shiftX, bob+dip+math.Abs(p.HeadZ)*0.25
	if vw == vSide {
		headDX += p.HeadZ * 0.7
	}
	// pose limbs by screen position
	var aL, aR, lL, lR Limb
	switch vw {
	case vBack:
		aL, aR, lL, lR = p.RA, p.LA, p.RF, p.LF
	default:
		aL, aR, lL, lR = p.LA, p.RA, p.LF, p.RF
	}
	armVec := func(l Limb, sign float64) (float64, float64) {
		switch vw {
		case vSide:
			return l.Z, l.Y
		case vBack:
			return sign * l.X * 0.8, l.Y - l.Z*0.1
		}
		return sign * l.X * 0.8, l.Y + l.Z*0.1
	}
	armRestDX := 1.2
	drawArm := func(id int, l Limb, sign float64) (float64, float64) {
		ly := &v.L[id]
		if !ly.ok {
			return 0, 0
		}
		dx, dy := armVec(l, sign)
		rdx := sign * armRestDX
		if vw == vSide {
			rdx = 0
		}
		theta := math.Atan2(dx, dy) - math.Atan2(rdx, 9)
		k := math.Min(1.05, math.Max(0.45, math.Hypot(dx, dy)/9.1))
		pdx, pdy := ly.px+torsoDX, ly.py+torsoDY
		r.blitRot(b, ly, pdx, pdy, theta, k)
		tt := theta
		return pdx + math.Sin(tt)*ly.length*k, pdy + math.Cos(tt)*ly.length*k
	}
	drawLeg := func(id int, l Limb, sign float64) {
		ly := &v.L[id]
		if !ly.ok {
			return
		}
		var dx float64
		if vw == vSide {
			dx = l.Z
		} else {
			dx = sign * l.X * 0.8
			if vw == vBack {
				dx = sign * l.X * 0.8
			}
		}
		rest := sign * 1.2
		if vw == vSide {
			rest = 0
		}
		pdx, pdy := ly.px+sway*0.5, ly.py+bob
		footY := r.feetY - (10 - l.Y)
		if p.Sit {
			footY = pdy + math.Min(l.Y-1, 7)
		}
		dy := footY - pdy
		dxr := dx - rest
		theta := math.Atan2(dxr, math.Max(dy, 1))
		k := math.Min(1, math.Max(0.4, math.Hypot(dxr, dy)/math.Max(1, ly.length)))
		r.blitRot(b, ly, pdx, pdy, theta, k)
	}
	// sign: -1 = screen left, +1 = screen right
	var hx0, hy0 float64
	heldID := 4
	switch vw {
	case vSide:
		drawLeg(PartLegL, lL, -1)
		drawArm(PartArmL, aL, -1)
		drawLeg(PartLegR, lR, 1)
		r.blitShift(b, &v.L[PartTorso], sway, torsoDY, shiftX)
		r.blitShift(b, &v.L[PartHead], headDX, headDY, 0)
		hx0, hy0 = drawArm(PartArmR, aR, 1)
	case vBack:
		drawLeg(PartLegL, lL, -1)
		drawLeg(PartLegR, lR, 1)
		r.blitShift(b, &v.L[PartTorso], sway, torsoDY, 0)
		hx0, hy0 = drawArm(PartArmL, aL, -1)
		heldID = 3
		drawArm(PartArmR, aR, 1)
		r.blitShift(b, &v.L[PartHead], headDX, headDY, 0)
	default:
		drawLeg(PartLegL, lL, -1)
		drawLeg(PartLegR, lR, 1)
		r.blitShift(b, &v.L[PartTorso], sway, torsoDY, 0)
		drawArm(PartArmL, aL, -1)
		hx0, hy0 = drawArm(PartArmR, aR, 1)
		r.blitShift(b, &v.L[PartHead], headDX, headDY, 0)
	}
	_ = heldID
	if p.Hold == HoldPee && vw != vBack {
		t := &v.L[PartTorso]
		if t.ok {
			drawStream(b, vw, t.px+torsoDX, t.py+torsoDY+2, r.feetY, p.Sit)
		}
	}
	kind := p.Hold
	if kind == HoldNone || (kind == HoldKnife && r.carry != HoldNone) {
		kind = r.carry
	}
	if kind != HoldNone && kind != HoldGuitar && v.L[PartArmR].ok {
		drawHeld(b, kind, color.RGBA{}, vw, hx0, hy0)
	}
	return b
}

// ---------------------------------------------------------------------------
// CharSet backed by a rig and optional hand drawn frames
// ---------------------------------------------------------------------------

func newRigCharSet(ca *CharAsset) *CharSet {
	spec := &CharSpec{Name: ca.Meta.Name, DisplayName: ca.Meta.DisplayName, Female: ca.Meta.Gender == "f", noOverride: true}
	rig := buildRig(ca)
	spec.Stature = rig.topY - 10
	cs := newCharSet(spec)
	cs.rig = rig
	cs.over = map[string]*pixBuf{}
	if ca.Dir != "" {
		files, _ := filepath.Glob(filepath.Join(ca.Dir, "anim", "*.png"))
		for _, f := range files {
			b, err := loadPNG(f)
			if err != nil {
				continue
			}
			cs.over[strings.TrimSuffix(filepath.Base(f), ".png")] = b
		}
	}
	return cs
}

func overrideKey(anim, vw, f int) string {
	return fmt.Sprintf("%s_%s_%d", animNames[anim], viewNames[assetView(vw)], f)
}

func (cs *CharSet) rigFrame(anim, vw, f int) *pixBuf {
	if ov := cs.over[overrideKey(anim, vw, f)]; ov != nil {
		return ov
	}
	switch anim {
	case AnimLying:
		if ov := cs.over["lying_front_0"]; ov != nil {
			return ov
		}
		return cs.rig.render(vFront, poseFor(AnimIdle, 0)).rotCW()
	case AnimSleep:
		if ov := cs.over["sleep_front_0"]; ov != nil {
			return ov
		}
		return cs.rig.render(vFront, poseFor(AnimIdle, 1)).rotCCW()
	case AnimPee:
		return cs.rig.render(vw, peePose(f, cs.Spec.Female))
	}
	b := cs.rig.render(vw, poseFor(anim, f))
	if cs.Spec.Ride != 0 {
		drawRide(b, cs.Spec, vw, b.w/2, 24) // the kids keep their scooter or skateboard
	}
	return b
}

var _ = os.Stat
var _ = ebiten.NewImage
