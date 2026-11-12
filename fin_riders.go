package main

import (
	"image/color"
	"math"
	"math/rand"
	"sort"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The three kids on wheels ride round the bottom of the last card of both endings (the secret "FIN" and the
// normal "¿ FINAL ?") while Murray talks: Diego on his skateboard, César and Sam-sam on their scooters.
//   - Their path is a loop (finRidePos): along the bottom of the card through the middle (a narrow strip
//     between "Gracias por jugar" and the hint), climbing up the screen at both ends, where the card is empty.
//     Diego and Sam-sam go one way, César the other: they cross and overtake each other.
//   - Twice a lap, in the empty ends, they do a trick (finTrickAt): Diego a kickflip or an ollie, César a
//     360 spin, Sam-sam a tailwhip. A jump is an arc with its shadow left on the ground.
//   - Everything is a pure function of the card's time (no shared random state): the tests replay the whole
//     ride and check that they never come near a text, Murray or his bubble (fin_riders_test.go).
//   - The art (the kids without their tiny built-in board, the boards with 4 frames of spinning wheels and the
//     shadows) is built once (finRideArtGet); drawing a frame allocates nothing of its own.
// ---------------------------------------------------------------------------

const (
	finRideSc     = 2.0   // whole scale: crisp pixels
	finRidersFrom = 3.0   // they roll in once the lines under the title are up
	finRidersFade = 0.8   // fade in
	finRideYC     = 489.0 // the middle of the strip they ride on (their wheels)...
	finRideAmp    = 10.0  // ...from finRideYC-amp (back, the far side of the loop) to +amp (front)
	finRideLift   = 110.0 // how far up the screen the loop climbs at both ends (the far side)
	finRideH      = 66.0  // from the ground to the top of the kid's head (crouching never raises it)
	finRideHW     = 22.0  // half the width of rider and board
	finRideHWWhip = 40.0  // during a tailwhip the scooter swings out in front of him
	finRideHWBail = 44.0  // lying on the ground after a fall, his board rolled on in front of him
	finRideArms   = 14.0  // arms up in the air (a no-hander, a barspin, a cheer after landing) reach this much higher
	finBailKidDX  = 8     // lying, the kid is this far behind the middle (scaled px)...
	finBailBoardX = 14    // ...and his board this far in front
	finRotN       = 12    // a backflip: 12 frames of 30 degrees (crisp, no smooth rotation)
	finRotS       = 72    // the square canvas of a rotated frame
	finKidPivY    = 28    // the kid turns round this row of his 24x44 frame (his middle)
	finRideSkW    = 22    // the skateboard's buffer: width,
	finRideSkH    = 14    // height,
	finRideSkCX   = 11    // centre column,
	finRideSkGY   = 12    // ground line (under the wheels)
	finRideSkTop  = 6     // and the row his feet stand on
	finRideScW    = 40    // the scooter's buffer
	finRideScH    = 24
	finRideScCX   = 16 // the rider's centre (the stem is 6 px in front of it)
	finRideScGY   = 24
	finRideScTop  = 19
	finRideFoot   = 41 // the row under the feet in a 24x44 character frame
)

// The tricks (the same names the kids shout in the game's bowls: skater_lines.go).
const (
	finTrickNone     = iota
	finTrickKickflip // skateboard: the board flips round its long axis under him
	finTrickOllie
	finTrickSpin     // a 360 with his board: side, front, other side, back
	finTrickWhip     // scooter: the deck swings round the stem
	finTrickHeelflip // skateboard: a kickflip the other way round
	finTrickShoveit  // skateboard: the board turns flat under him (a 360 shove-it)
	finTrickGrab     // skateboard: crouched in the air, a hand on the board
	finTrickManual   // skateboard: on the back wheels, no jump
	finTrickBackflip // kid and board turn a whole lap backwards (12 crisp frames)
	finTrickBail     // he falls off, lies on the ground a moment and gets back on
	finTrickBunny    // scooter: a little bunny hop
	finTrickBarspin  // scooter: the handlebar spins round
	finTrickNoHand   // scooter: both hands up in the air
	finTrickCombo    // two in one long jump: kickflip + grab, or tailwhip + barspin
	finTrickCount
)

// finTrickLift: how high a trick jumps, in units of the rider's jumpH (the bail only on its first part).
func finTrickLift(k int) float64 {
	switch k {
	case finTrickManual:
		return 0
	case finTrickBunny:
		return 0.6
	case finTrickBackflip:
		return 1.25
	case finTrickCombo:
		return 1.2
	case finTrickBail:
		return 0.7
	}
	return 1
}

// finTrickDur: how long a trick lasts, in units of the rider's jumpDur.
func finTrickDur(k int) float64 {
	switch k {
	case finTrickManual:
		return 1.4
	case finTrickCombo:
		return 1.35
	case finTrickBackflip:
		return 1.15
	case finTrickBail:
		return 2.4
	}
	return 1
}

const finBailAir = 0.3 // the part of a bail that is the jump; then he lies there and gets up

// finTrickHeight: the height of the jump at progress p of a trick (whole pixels). Pure.
func finTrickHeight(k int, jumpH, p float64) float64 {
	if k == finTrickBail {
		if p >= finBailAir {
			return 0
		}
		p /= finBailAir
	}
	return math.Round(finTrickLift(k) * jumpH * 4 * p * (1 - p))
}

// finRider: one kid and how he rides.
type finRider struct {
	slug     string
	ride     int     // 1 scooter, 2 skateboard
	deck     uint32  // the colour of his board (the one he has in the game)
	speed    float64 // px a second along the loop
	ax       float64 // half the width of his loop
	rev      bool    // he goes round the other way
	phase    float64 // where he is on the loop (fraction of a lap) when he rolls in
	jumpH    float64 // height of his jumps
	jumpDur  float64 // and how long they last
	spots    [2]float64
	every    [2]int  // the trick at spot k is done on the laps n with n%every[k]==0
	push     float64 // seconds between two pushes with the foot
	pushLen  float64
	pushBase float64
	tricks   []int // his repertoire, done in this order, one after the other, round and round
}

var finRiders = [3]finRider{
	{slug: "diego", ride: 2, deck: 0xf08a20, speed: 150, ax: 420, phase: 0.08, jumpH: 46, jumpDur: 0.62,
		spots: [2]float64{2*math.Pi - 0.36, math.Pi + 0.36}, every: [2]int{1, 1}, push: 1.9, pushLen: 0.45,
		tricks: []int{finTrickKickflip, finTrickOllie, finTrickGrab, finTrickHeelflip, finTrickSpin, finTrickShoveit,
			finTrickManual, finTrickBackflip, finTrickCombo, finTrickBail}},
	{slug: "cesar", ride: 1, deck: 0x3aa050, speed: 118, ax: 396, rev: true, phase: 0.40, jumpH: 38, jumpDur: 0.84,
		spots: [2]float64{2*math.Pi - 0.30, math.Pi + 0.30}, every: [2]int{1, 2}, push: 2.6, pushLen: 0.5, pushBase: 0.7,
		tricks: []int{finTrickSpin, finTrickBunny, finTrickBarspin, finTrickNoHand, finTrickWhip, finTrickBackflip,
			finTrickCombo, finTrickBail}},
	{slug: "sam_sam", ride: 1, deck: 0xe83a3a, speed: 134, ax: 408, phase: 0.64, jumpH: 42, jumpDur: 0.74,
		spots: [2]float64{2*math.Pi - 0.33, math.Pi + 0.33}, every: [2]int{2, 1}, push: 2.2, pushLen: 0.45, pushBase: 1.3,
		tricks: []int{finTrickWhip, finTrickBarspin, finTrickNoHand, finTrickBunny, finTrickSpin, finTrickBackflip,
			finTrickBail, finTrickCombo}},
}

// finRidePath: the loop of a rider with half-width ax at angle th: x round the middle of the screen; the near
// side (sin > 0) runs along the strip, the far side climbs up the screen at both ends. Pure.
func finRidePath(ax, th float64) (x, y float64) {
	c, s := math.Cos(th), math.Sin(th)
	k := clampF((math.Abs(c)-0.66)/0.34, 0, 1)
	return 480 + ax*c, finRideYC + finRideAmp*s - finRideLift*k*k*(1-s)/2
}

// finRideTable: the length of each loop along the way (a table per rider, built once).
type finRideTable struct {
	th, cum [1025]float64
	length  float64
	spotD   [2]float64 // where the middle of each trick is, along the loop
}

var finRideTabs = func() (t [3]finRideTable) {
	for i, r := range finRiders {
		tb := &t[i]
		px, py := finRidePath(r.ax, 0)
		for j := 0; j <= 1024; j++ {
			th := 2 * math.Pi * float64(j) / 1024
			x, y := finRidePath(r.ax, th)
			tb.th[j] = th
			if j > 0 {
				tb.cum[j] = tb.cum[j-1] + math.Hypot(x-px, y-py)
			}
			px, py = x, y
		}
		tb.length = tb.cum[1024]
		for k, sp := range r.spots {
			j := int(math.Round(sp / (2 * math.Pi) * 1024))
			tb.spotD[k] = tb.cum[j]
		}
	}
	return t
}()

// finRideTheta: the angle of the loop d px along it (u-space: the rider's own sense of travel).
func (tb *finRideTable) theta(d float64) float64 {
	d = math.Mod(d, tb.length)
	if d < 0 {
		d += tb.length
	}
	j := sort.SearchFloat64s(tb.cum[:], d)
	if j <= 0 {
		return 0
	}
	if j > 1024 {
		j = 1024
	}
	f := (d - tb.cum[j-1]) / math.Max(1e-9, tb.cum[j]-tb.cum[j-1])
	return tb.th[j-1] + f*(tb.th[j]-tb.th[j-1])
}

// finRideState: where a rider is and what he is doing at one moment of the card.
type finRideState struct {
	on        bool
	alpha     float64
	x, g      float64 // centre and ground line (whole pixels)
	h         float64 // height of the jump (whole pixels)
	dir       Dir     // DirLeft/DirRight (the way he goes), DirUp/DirDown when he climbs or comes down the loop
	trick     int
	p         float64 // progress of the trick, 0..1
	d         float64 // px travelled (the wheels turn with it)
	push      bool    // pushing with a foot
	pushF     int
	crouch    bool
	react     bool // just landed a trick: an arm up in the air
	wheelF    int
	dirOfMove Dir // the side he is going to (for the tests: the mirror matches it)
}

// finRidePos: the point of rider i's loop d px along it (in his own sense of travel). Pure.
func finRidePos(i int, d float64) (x, y float64) {
	r := &finRiders[i]
	th := finRideTabs[i].theta(d)
	if r.rev { // the same loop (it is symmetric) the other way round
		th = math.Pi - th
	}
	return finRidePath(r.ax, th)
}

// finTrickAt: the trick rider i is doing d px along his ride (finTrickNone if none) and its progress. Pure.
func finTrickAt(i int, d float64) (kind int, p float64) {
	r, tb := &finRiders[i], &finRideTabs[i]
	for k := range r.spots {
		lap := math.Floor((d-tb.spotD[k])/tb.length + 0.5) // the lap whose trick at spot k is nearest
		n := int(lap)
		if n < 0 || n%r.every[k] != 0 {
			continue
		}
		kind = finTrickPick(i, n, k)
		span := r.speed * r.jumpDur * finTrickDur(kind)
		start := tb.spotD[k] + lap*tb.length - span/2
		rel := d - start
		if rel < 0 || rel >= span || start < r.phase*tb.length { // not yet, or a jump begun before he rolled in
			continue
		}
		return kind, rel / span
	}
	return finTrickNone, 0
}

// finTrickPick: the trick rider i does at spot k of lap n: the next one of his repertoire (every trick he has
// done before counts, so the whole list comes round before any repeats). Pure.
func finTrickPick(i, n, k int) int {
	r := &finRiders[i]
	idx := 0
	for kk, e := range r.every {
		idx += (n + e - 1) / e // the laps before n with a trick at spot kk
		if kk < k && n%e == 0 {
			idx++
		}
	}
	return r.tricks[idx%len(r.tricks)]
}

// finRideAt: the state of rider i at time t of the card. Pure (and allocation-free).
func finRideAt(i int, t float64) finRideState {
	var s finRideState
	if t < finRidersFrom {
		return s
	}
	r := &finRiders[i]
	tt := t - finRidersFrom
	s.on, s.alpha = true, math.Min(1, tt/finRidersFade)
	s.d = r.phase*finRideTabs[i].length + r.speed*tt
	x, y := finRidePos(i, s.d)
	s.x, s.g = math.Round(x), math.Round(y)
	s.trick, s.p = finTrickAt(i, s.d)
	if s.trick != finTrickNone {
		s.h = finTrickHeight(s.trick, r.jumpH, s.p)
	}
	// the way he goes: the tangent over a few px either side
	x0, y0 := finRidePos(i, s.d-8)
	x1, y1 := finRidePos(i, s.d+8)
	dx, dy := x1-x0, y1-y0
	s.dirOfMove = DirRight
	if dx < 0 {
		s.dirOfMove = DirLeft
	}
	s.dir = s.dirOfMove
	if math.Abs(dy) > 1.4*math.Abs(dx) && s.trick == finTrickNone {
		s.dir = DirDown
		if dy < 0 {
			s.dir = DirUp
		}
	}
	s.wheelF = int(math.Floor(s.d/7)) % 4
	if s.trick == finTrickNone {
		ph := math.Mod(tt+r.pushBase, r.push)
		if ph < r.pushLen && s.dir != DirUp && s.dir != DirDown {
			s.push, s.pushF = true, int(ph/r.pushLen*float64(animFrameCount(AnimWalk)))
		} else {
			s.crouch = int(math.Floor(tt*2.6+float64(i)))%2 == 0 // a little bob on the board
		}
		// just landed: an arm up for a moment (not after a fall, nor after a manual)
		if k, _ := finTrickAt(i, s.d-r.speed*0.5); k != finTrickNone && k != finTrickBail && k != finTrickManual {
			s.react, s.push, s.crouch = true, false, false
		}
		// about to jump: a crouch just before the take-off
		if k, _ := finTrickAt(i, s.d+r.speed*0.18); k != finTrickNone {
			s.crouch, s.push, s.react = true, false, false
		}
	}
	return s
}

// finRideReach: the half width and the height over the ground of rider i's box during trick k (the backflip and
// the bail need more room: finRotTabs measures their frames).
func finRideReach(i, k int) (hw, h float64) {
	switch k {
	case finTrickWhip, finTrickCombo:
		if finRiders[i].ride == 1 {
			return finRideHWWhip, finRideH
		}
	case finTrickBackflip:
		t := finRotTabs()
		return t.flipHW[i], math.Max(finRideH, t.flipH[i])
	case finTrickBail:
		return finRideHWBail, finRideH
	}
	return finRideHW, finRideH
}

// finRideBox: the rectangle rider i covers at time t (kid, board and shadow), for the layout tests. Pure.
func finRideBox(i int, t float64) (x0, y0, x1, y1 float64, on bool) {
	s := finRideAt(i, t)
	if !s.on {
		return 0, 0, 0, 0, false
	}
	hw, h := finRideReach(i, s.trick)
	if s.react || s.trick == finTrickNoHand || s.trick == finTrickBarspin || s.trick == finTrickCombo {
		h += finRideArms // arms up in the air
	}
	return s.x - hw, s.g - s.h - h, s.x + hw, s.g + finRideSc, true // the shadow: 1 px under the ground line
}

// ---- the art, built once -------------------------------------------------------------------------------

// Skateboard views.
const (
	finSkFlat = iota
	finSkNoseUp
	finSkNoseDown
	finSkUpside
	finSkFaceDeck
	finSkFaceGrip
	finSkEnd
	finSkViews
)

// Scooter views.
const (
	finScSide  = iota
	finScWhip  // the deck swung round in front of the stem
	finScFront // seen from the front, under a kid facing us (or his back)
	finScEdge  // the deck edge-on under the stem, halfway round a tailwhip
	finScBarN  // a barspin: the handlebar seen end-on...
	finScBarW  // ...and seen flat on, its whole width
	finScViews
)

type finRideArt struct {
	kids    [3]*CharSet
	skate   [finSkViews][4]*ebiten.Image
	scoot   [2][finScViews][4]*ebiten.Image // [0] César, [1] Sam-sam
	shadow  [3]*ebiten.Image
	kidRot  [3][4]*ebiten.Image       // falling forward: 0, 30, 60 and 90 degrees (lying)
	combo   [3][finRotN]*ebiten.Image // kid and board together, turned backwards 30 degrees a frame
	kidArms [3]*ebiten.Image          // both arms up in the air
}

// finKidArmsBuf: rider i seen from the side with both arms up (a no-hander, a barspin): his own look, no props.
func finKidArmsBuf(i int) *pixBuf {
	cs := finRideKids()[i]
	pp := poseFor(AnimIdle, 0)
	pp.LA, pp.RA = Limb{4, -8, 0}, Limb{5, -7, 0}
	pp.Open = true
	pp.Anim, pp.F = AnimIdle, 0
	return genChar(cs.Spec, vSide, pp)
}

var (
	finKidsOnce sync.Once
	finKidsCS   [3]*CharSet
)

// finRideKids: the three kids as the card draws them (their own look, without the game's tiny board).
func finRideKids() [3]*CharSet {
	finKidsOnce.Do(func() {
		for i, r := range finRiders {
			if f := presetByName(r.slug); f != nil {
				spec := f(rand.New(rand.NewSource(7)))
				spec.Ride = 0 // he rides the big board drawn here
				finKidsCS[i] = newCharSet(&spec)
			}
		}
	})
	return finKidsCS
}

// finKidFrame: the pixels of a frame of the kid seen from the side, exactly as his CharSet builds it.
func finKidFrame(cs *CharSet, anim, f int) *pixBuf {
	if cs.rig != nil {
		return cs.rigFrame(anim, vSide, f)
	}
	pp := poseFor(anim, f)
	pp.Anim, pp.F = anim, f
	return genChar(cs.Spec, vSide, pp)
}

// finBlit: src copied onto dst at (ox, oy) where it is not transparent.
func finBlit(dst, src *pixBuf, ox, oy int) {
	for y := 0; y < src.h; y++ {
		for x := 0; x < src.w; x++ {
			i := (y*src.w + x) * 4
			if src.p[i+3] == 0 {
				continue
			}
			dx, dy := x+ox, y+oy
			if dx < 0 || dy < 0 || dx >= dst.w || dy >= dst.h {
				continue
			}
			j := (dy*dst.w + dx) * 4
			copy(dst.p[j:j+4], src.p[i:i+4])
		}
	}
}

// finKidRotBuf: rider i alone, turned clockwise (falling forward) by step*30 degrees round his middle.
func finKidRotBuf(i, step int) *pixBuf {
	cs := finRideKids()[i]
	return rotateAroundN(finKidFrame(cs, AnimIdle, 0), float64(step)*math.Pi/6, finRideKidCX, finKidPivY, finRotS)
}

// finComboBuf: rider i standing on his board, the two turned together backwards (anticlockwise facing right)
// by step*30 degrees: a backflip.
func finComboBuf(i, step int) *pixBuf {
	cs := finRideKids()[i]
	kid := finKidFrame(cs, AnimCrouch, 0)
	var b *pixBuf
	var pvx int
	if finRiders[i].ride == 2 { // the deck's top row under his feet (row 41 of his frame)
		b = newPix(2*finRideKidCX, finRideFoot-finRideSkTop+finRideSkH)
		finBlit(b, finDrawSkate(finSkFlat, 0, hx(finRiders[i].deck)), finRideKidCX-finRideSkCX, finRideFoot-finRideSkTop)
		finBlit(b, kid, 0, 0)
		pvx = finRideKidCX
	} else {
		b = newPix(finRideScW, finRideFoot-finRideScTop+finRideScH)
		finBlit(b, finDrawScooter(finScSide, 0, hx(finRiders[i].deck)), 0, finRideFoot-finRideScTop)
		finBlit(b, kid, finRideScCX-finRideKidCX, 0)
		pvx = finRideScCX
	}
	return rotateAroundN(b, -float64(step)*math.Pi/6, float64(pvx), finKidPivY+2, finRotS)
}

// finRotTab: where the rotated frames rest (their lowest row: they stand on it) and how much room a backflip takes.
type finRotTab struct {
	kidAY   [3][4]float64
	comboAY [3][finRotN]float64
	flipHW  [3]float64 // half width of the box of a backflip (screen px)
	flipH   [3]float64 // and its height over the ground
}

var (
	finRotOnce sync.Once
	finRotT    finRotTab
)

func finRotTabs() *finRotTab {
	finRotOnce.Do(func() {
		for i := range finRiders {
			for k := 0; k < 4; k++ {
				_, _, _, y1, _ := finBufBounds(finKidRotBuf(i, k))
				finRotT.kidAY[i][k] = float64(y1 + 1)
			}
			for k := 0; k < finRotN; k++ {
				x0, y0, x1, y1, _ := finBufBounds(finComboBuf(i, k))
				finRotT.comboAY[i][k] = float64(y1 + 1)
				hw := math.Max(finRotS/2-float64(x0), float64(x1+1)-finRotS/2) * finRideSc
				finRotT.flipHW[i] = math.Max(finRotT.flipHW[i], math.Ceil(hw))
				finRotT.flipH[i] = math.Max(finRotT.flipH[i], float64(y1+1-y0)*finRideSc)
			}
		}
	})
	return &finRotT
}

// finBufBounds: the opaque pixels of a buffer (first and last column and row).
func finBufBounds(b *pixBuf) (x0, y0, x1, y1 int, ok bool) {
	x0, y0, x1, y1 = b.w, b.h, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] != 0 {
				x0, y0 = minI(x0, x), minI(y0, y)
				x1, y1 = max(x1, x), max(y1, y)
			}
		}
	}
	return x0, y0, x1, y1, x1 >= 0
}

// finRidersDrawn counts the frames drawn with the riders (the tests check both cards draw them).
var finRidersDrawn int

var (
	finArtMu sync.Mutex
	finArt   *finRideArt
)

// finRideArtGet: the kids (their own sprite, without the little board the game draws under them) and their
// boards, built the first time and kept.
func finRideArtGet() *finRideArt {
	finArtMu.Lock()
	defer finArtMu.Unlock()
	if finArt != nil {
		return finArt
	}
	a := &finRideArt{kids: finRideKids()}
	for i := range finRiders {
		if a.kids[i] == nil {
			continue
		}
		a.kidArms[i] = finKidArmsBuf(i).img()
		for k := 0; k < 4; k++ {
			a.kidRot[i][k] = finKidRotBuf(i, k).img()
		}
		for k := 0; k < finRotN; k++ {
			a.combo[i][k] = finComboBuf(i, k).img()
		}
	}
	for v := 0; v < finSkViews; v++ {
		for w := 0; w < 4; w++ {
			a.skate[v][w] = finDrawSkate(v, w, hx(finRiders[0].deck)).img()
		}
	}
	for k, ri := range []int{1, 2} {
		for v := 0; v < finScViews; v++ {
			for w := 0; w < 4; w++ {
				a.scoot[k][v][w] = finDrawScooter(v, w, hx(finRiders[ri].deck)).img()
			}
		}
	}
	for k := range a.shadow {
		a.shadow[k] = finDrawShadow(k).img()
	}
	finArt = a
	return a
}

// finDrawShadow: the shadow on the ground, k = 0 (on the board) to 2 (high in a jump: smaller).
func finDrawShadow(k int) *pixBuf {
	b := newPix(22, 3)
	b.ellipse(11, 1.5, [3]float64{10, 7, 5}[k], 1.5, rgba(0x34304e, 230)) // the card is nearly black: a dim floor patch reads as his shadow
	return b
}

// finWheel: a 3x3 wheel (a plus) whose dark notch goes round with frame f: it turns as he rolls.
func finWheel(b *pixBuf, cx, cy, f int) {
	tyre, hub, notch := hx(0xf0f0f0), hx(0x8a8a96), hx(0x3a3a46)
	arms := [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} // clockwise: rolling to the right
	b.set(cx, cy, hub)
	for k, d := range arms {
		col := tyre
		if k == f%4 {
			col = notch
		}
		b.set(cx+d[0], cy+d[1], col)
	}
}

// finDrawSkate: Diego's skateboard, side view facing right (nose to the right) unless view says otherwise;
// f is the frame of the wheels.
func finDrawSkate(view, f int, deck color.RGBA) *pixBuf {
	b := newPix(finRideSkW, finRideSkH)
	const ox, oy = finRideSkCX, finRideSkTop + 1 // local (0,0): the middle of the deck's coloured row
	grip, metal := hx(0x2a2a30), hx(0x9a9aa6)
	light := shade(deck, 1.25)
	switch view {
	case finSkFaceDeck, finSkFaceGrip: // halfway through a kickflip: the board seen flat on
		face, edge := deck, shade(deck, 0.7)
		if view == finSkFaceGrip {
			face, edge = grip, hx(0x18181c)
		}
		b.rect(ox-8, oy-3, 17, 5, face)
		for x := ox - 8; x <= ox+8; x++ {
			b.set(x, oy-3, edge)
			b.set(x, oy+1, edge)
		}
		b.rect(ox-9, oy-2, 1, 3, edge)
		b.rect(ox+9, oy-2, 1, 3, edge)
		if view == finSkFaceDeck { // the graphic under the board: a white flash
			b.rect(ox-2, oy-1, 5, 1, hx(0xf4f4f4))
			b.set(ox-3, oy, hx(0xf4f4f4))
			b.set(ox+3, oy-2, hx(0xf4f4f4))
		} else { // the bolts of the trucks
			for _, x := range []int{ox - 6, ox - 4, ox + 4, ox + 6} {
				b.set(x, oy-1, metal)
			}
		}
		return b
	case finSkEnd: // seen from a nose: the deck, the truck and two wheels
		b.rect(ox-4, oy-1, 9, 1, grip)
		b.rect(ox-4, oy, 9, 1, deck)
		b.rect(ox-3, oy+1, 7, 1, metal)
		for _, x := range []int{ox - 4, ox + 3} {
			for y := 0; y < 3; y++ {
				col := hx(0xf0f0f0)
				if y == f%3 {
					col = hx(0xb0b0bc)
				}
				b.rect(x, oy+2+y, 2, 1, col)
			}
		}
		return b
	}
	tilt := 0.0
	switch view {
	case finSkNoseUp:
		tilt = 0.3
	case finSkNoseDown:
		tilt = -0.25
	}
	up := view == finSkUpside
	off := func(lx int) int { return -int(math.Round(float64(lx) * tilt)) }
	lift := -max(off(-5), off(5)) // tilted, it rests on its lowest wheel: that one stays on the ground line
	put := func(lx, ly int, col color.RGBA) {
		if up { // upside down: the wheels on top
			ly = 1 - ly
		}
		b.set(ox+lx, oy+ly+off(lx)+lift, col)
	}
	for lx := -8; lx <= 8; lx++ {
		put(lx, -1, grip)
		put(lx, 0, deck)
	}
	put(-8, 0, light)
	put(8, 0, light)
	for _, sx := range []int{-1, 1} { // the kicked-up nose and tail
		put(9*sx, -1, deck)
		put(10*sx, -2, deck)
		put(9*sx, -2, grip)
	}
	for _, sx := range []int{-1, 1} { // trucks and wheels
		put(5*sx, 1, metal)
		put(4*sx, 1, metal)
		wx := 5 * sx
		cy := oy + 3 + off(wx) + lift
		if up {
			cy = oy - 2 + off(wx) + lift
		}
		finWheel(b, ox+wx, cy, f)
	}
	return b
}

// finDrawScooter: a kid's scooter, side view facing right (the stem in front of him) unless view says otherwise.
func finDrawScooter(view, f int, deck color.RGBA) *pixBuf {
	b := newPix(finRideScW, finRideScH)
	metal, dark, grip := hx(0x9a9aa6), hx(0x3a3a46), hx(0x1c1c22)
	light := shade(deck, 1.25)
	stem := finRideScCX + 6
	bar := func(cx, half int) {
		b.rect(cx-half, 2, 2*half+1, 1, metal)
		b.rect(cx-half, 2, 2, 1, grip)
		b.rect(cx+half-1, 2, 2, 1, grip)
	}
	switch view {
	case finScFront:
		cx := finRideScCX
		b.line(cx, 3, cx, 20, metal)
		bar(cx, 6)
		b.rect(cx-2, finRideScTop, 5, 1, light)
		b.rect(cx-2, finRideScTop+1, 5, 1, deck)
		for y := 0; y < 3; y++ {
			col := hx(0xf0f0f0)
			if y == f%3 {
				col = hx(0xb0b0bc)
			}
			b.rect(cx-1, 21+y, 3, 1, col)
		}
		return b
	case finScEdge:
		b.line(stem, 20, stem-1, 3, metal)
		bar(stem-1, 3)
		b.rect(stem-2, finRideScTop, 5, 2, deck)
		b.rect(stem-2, finRideScTop, 5, 1, light)
		finWheel(b, stem, 22, f)
		return b
	}
	x0, x1, rear := finRideScCX-7, stem, finRideScCX-6
	if view == finScWhip {
		x0, x1, rear = stem, stem+13, stem+12
	}
	b.rect(x0, finRideScTop, x1-x0+1, 1, light)
	b.rect(x0, finRideScTop+1, x1-x0+1, 1, deck)
	b.set(rear, finRideScTop+2, dark) // the brake over the back wheel
	b.line(stem, 20, stem-1, 3, metal)
	b.set(stem+1, 21, metal) // the fork
	switch view {
	case finScBarN: // the bar end-on: just its two grips, one behind the other
		b.rect(stem-2, 1, 2, 3, grip)
		b.set(stem-1, 2, metal)
	case finScBarW: // the bar flat on, a long line across the stem
		bar(stem-1, 5)
	default:
		bar(stem-1, 3)
	}
	finWheel(b, rear, 22, f)
	finWheel(b, stem, 22, f)
	return b
}

// ---- drawing -------------------------------------------------------------------------------------------

// drawFinRiders: the three kids, the farthest first (the ones on the near side of the loop pass in front).
func (g *Game) drawFinRiders(dst *ebiten.Image, e *endState) {
	if e.t < finRidersFrom {
		return
	}
	finRidersDrawn++
	a := finRideArtGet()
	var st [3]finRideState
	order := [3]int{0, 1, 2}
	for i := range st {
		st[i] = finRideAt(i, e.t)
	}
	for i := 1; i < 3; i++ { // by ground line: three of them, an insertion sort
		for j := i; j > 0 && st[order[j]].g < st[order[j-1]].g; j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}
	for _, i := range order {
		finDrawRider(dst, a, i, &st[i])
	}
}

// finPart: one image of a rider as drawn: what it is and where its anchor (cx, ay of the image) goes on the
// screen, mirrored when sx is -1.
type finPart struct {
	kind    int // finPartShadow, finPartSkate, finPartScooter or finPartKid
	view, w int // board view and wheel frame, or the shadow's size
	anim    int // the kid's animation, frame and facing
	fr      float64
	dir     Dir
	cx, ay  float64
	x, y    float64
	sx      float64
}

const (
	finPartShadow = iota
	finPartSkate
	finPartScooter
	finPartKid
	finPartKidRot // the kid alone, turned (view: the step of finKidRotBuf)
	finPartCombo  // kid and board turned together (view: the step of finComboBuf)
	finPartKidArms
)

const finKidArmsUp = -1 // in finRidePlan: the kid's frame with both arms up (finKidArmsBuf)

const finRideKidCX = 12 // the middle column of a 24 px wide character frame

// finSkFlipViews: the board's views through a kickflip (round its long axis one way) and a heelflip (the other).
var (
	finSkKickViews = [...]int{finSkNoseUp, finSkFaceDeck, finSkUpside, finSkFaceGrip, finSkFlat, finSkFlat, finSkNoseDown}
	finSkHeelViews = [...]int{finSkNoseUp, finSkFaceGrip, finSkUpside, finSkFaceDeck, finSkFlat, finSkFlat, finSkNoseDown}
)

// finRidePlan: the images of rider i in state s, in drawing order (shadow, then board and kid). Pure.
func finRidePlan(i int, s *finRideState) (parts [3]finPart, n int) {
	if !s.on {
		return parts, 0
	}
	r := &finRiders[i]
	const sc = finRideSc
	// the shadow stays on the ground, smaller the higher he jumps
	sh := 0
	if s.h > 24 {
		sh = 2
	} else if s.h > 8 {
		sh = 1
	}
	parts[0] = finPart{kind: finPartShadow, w: sh, cx: 11, ay: 2, x: s.x, y: s.g, sx: 1}

	// which way the kid and his board face
	kidDir, sx := s.dir, 1.0
	if s.dir == DirLeft {
		sx = -1
	}
	if s.dir == DirLeft || s.dir == DirRight {
		kidDir = DirRight // the side frame, mirrored by sx
	}
	gy := s.g - s.h
	trick, p := s.trick, s.p
	if trick == finTrickBackflip { // kid and board turn round together: one image, resting on its lowest row
		step := int(math.Round(p*finRotN)) % finRotN
		parts[1] = finPart{kind: finPartCombo, view: step, cx: finRotS / 2, ay: finRotTabs().comboAY[i][step], x: s.x, y: gy, sx: sx}
		return parts, 2
	}
	if trick == finTrickCombo { // the first half one trick, the second another
		if p < 0.5 {
			trick, p = finTrickKickflip, p*2
			if r.ride == 1 {
				trick = finTrickWhip
			}
		} else {
			trick, p = finTrickGrab, p*2-1
			if r.ride == 1 {
				trick = finTrickBarspin
			}
		}
	}
	board := finPart{sx: 1}
	var btop float64
	boardFirst := true
	drop := 0.0 // the board falls a little under his feet during a flip
	kidAnim, kidFr := AnimIdle, 0.0
	if s.push {
		kidAnim, kidFr = AnimWalk, float64(s.pushF)
	}
	if s.react {
		kidAnim, kidFr = AnimWave, float64(int(s.d/12)%4)
	}
	if r.ride == 2 {
		v := finSkFlat
		switch {
		case s.dir == DirUp || s.dir == DirDown:
			v = finSkEnd
		case trick == finTrickKickflip || trick == finTrickHeelflip:
			views := finSkKickViews
			if trick == finTrickHeelflip {
				views = finSkHeelViews
			}
			k := minI(int(p*float64(len(views))), len(views)-1)
			v = views[k]
			if v != finSkNoseUp && v != finSkNoseDown && v != finSkFlat {
				drop = 2
				if v == finSkUpside {
					drop = 3
				}
			}
		case trick == finTrickOllie:
			switch {
			case p < 0.35:
				v = finSkNoseUp
			case p > 0.7:
				v = finSkNoseDown
			}
		case trick == finTrickShoveit: // the board turns flat under him: side, end, side, end
			if int(p*4)%2 == 1 {
				v = finSkEnd
			}
			drop = 1
			kidAnim, kidFr = AnimCrouch, 0
		case trick == finTrickGrab:
			kidAnim, kidFr = AnimCrouch, 0
			if p > 0.2 && p < 0.8 {
				v = finSkNoseUp
			}
		case trick == finTrickManual:
			v = finSkNoseUp
		case trick == finTrickSpin: // Diego turns round with his board under him
			switch int(p * 4) {
			case 1:
				kidDir, v, sx = DirDown, finSkEnd, 1
			case 2:
				sx = -sx
			case 3:
				kidDir, v, sx = DirUp, finSkEnd, 1
			}
		}
		board = finPart{kind: finPartSkate, view: v, cx: finRideSkCX, ay: finRideSkGY}
		btop = finRideSkTop
	} else {
		v := finScSide
		switch {
		case trick == finTrickSpin: // César turns round with his scooter: side, front, other side, back
			switch int(p * 4) {
			case 1:
				kidDir, v, sx = DirDown, finScFront, 1
			case 2:
				sx = -sx
			case 3:
				kidDir, v, sx = DirUp, finScFront, 1
			}
		case trick == finTrickWhip: // Sam-sam swings the deck round the stem
			switch int(p * 5) {
			case 1, 3:
				v = finScEdge
			case 2:
				v = finScWhip
			}
		case trick == finTrickBarspin: // the bar spins: end-on, flat on, end-on; his hands off it
			switch int(p * 5) {
			case 1, 3:
				v = finScBarN
				kidAnim, kidFr = finKidArmsUp, 0
			case 2:
				v = finScBarW
				kidAnim, kidFr = finKidArmsUp, 0
			}
		case trick == finTrickNoHand:
			if p > 0.2 && p < 0.8 {
				kidAnim, kidFr, drop = finKidArmsUp, 0, 2
			}
		case trick == finTrickBunny:
			if p > 0.3 && p < 0.7 {
				kidAnim, kidFr = AnimCrouch, 0
			}
		case s.dir == DirUp || s.dir == DirDown:
			v = finScFront
		}
		if kidDir == DirDown {
			boardFirst = false // facing us, the stem is in front of him
		}
		board = finPart{kind: finPartScooter, view: v, cx: finRideScCX, ay: finRideScGY}
		btop = finRideScTop
	}
	board.w, board.x, board.y, board.sx = s.wheelF, s.x, gy+drop*sc, sx
	if trick == finTrickBail { // he flies off forward, lies there, gets up; the board rolls on a little and waits
		step := 3
		switch {
		case p < finBailAir:
			step = minI(3, int(p/finBailAir*4))
		case p > 0.9:
			step = 1
		case p > 0.82:
			step = 2
		}
		off := 0.0 // how far apart they are, 0..1
		switch {
		case p < finBailAir:
			off = p / finBailAir
		case p > 0.82:
			off = math.Max(0, (1-p)/0.18)
		default:
			off = 1
		}
		board.x = s.x + sx*math.Round(off*finBailBoardX)
		board.y = s.g
		board.view = finSkFlat
		if r.ride == 1 {
			board.view = finScSide
		}
		kid := finPart{kind: finPartKidRot, view: step, cx: finRotS / 2, ay: finRotTabs().kidAY[i][step],
			x: s.x - sx*math.Round(off*finBailKidDX), y: gy, sx: sx}
		parts[1], parts[2] = board, kid
		return parts, 3
	}
	// the kid: his feet on the deck; pushing he puts a foot down (a walking frame); bobbing he bends his knees
	kid := finPart{kind: finPartKid, anim: kidAnim, fr: kidFr, dir: kidDir, cx: finRideKidCX, ay: finRideFoot, x: s.x, y: gy - (board.ay-btop)*sc, sx: sx}
	if kidAnim == finKidArmsUp { // both hands off the bar, up in the air (his own frame, seen from the side)
		kid.kind, kid.anim, kid.dir = finPartKidArms, AnimIdle, DirRight
	}
	if s.crouch {
		kid.y += sc
	}
	if boardFirst {
		parts[1], parts[2] = board, kid
	} else {
		parts[1], parts[2] = kid, board
	}
	return parts, 3
}

// finDrawRider: rider i in state s.
func finDrawRider(dst *ebiten.Image, a *finRideArt, i int, s *finRideState) {
	if !s.on || a.kids[i] == nil {
		return
	}
	k := 0 // César's scooter, or Sam-sam's
	if finRiders[i].slug == "sam_sam" {
		k = 1
	}
	parts, n := finRidePlan(i, s)
	var op ebiten.DrawImageOptions
	for _, p := range parts[:n] {
		var im *ebiten.Image
		switch p.kind {
		case finPartShadow:
			im = a.shadow[p.w]
		case finPartSkate:
			im = a.skate[p.view][p.w]
		case finPartScooter:
			im = a.scoot[k][p.view][p.w]
		case finPartKidRot:
			im = a.kidRot[i][p.view]
		case finPartCombo:
			im = a.combo[i][p.view]
		case finPartKidArms:
			im = a.kidArms[i]
		default:
			im, _ = a.kids[i].img(p.anim, p.dir, p.fr)
		}
		op.GeoM.Reset()
		op.GeoM.Translate(-p.cx, -p.ay)
		op.GeoM.Scale(p.sx*finRideSc, finRideSc)
		op.GeoM.Translate(p.x, p.y)
		op.ColorScale.Reset()
		op.ColorScale.ScaleAlpha(float32(s.alpha))
		dst.DrawImage(im, &op)
	}
}
