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

// The tricks.
const (
	finTrickNone = iota
	finTrickKickflip
	finTrickOllie
	finTrickSpin
	finTrickWhip
)

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
}

var finRiders = [3]finRider{
	{slug: "diego", ride: 2, deck: 0xf08a20, speed: 150, ax: 420, phase: 0.08, jumpH: 46, jumpDur: 0.62,
		spots: [2]float64{2*math.Pi - 0.36, math.Pi + 0.36}, every: [2]int{1, 1}, push: 1.9, pushLen: 0.45},
	{slug: "cesar", ride: 1, deck: 0x3aa050, speed: 118, ax: 396, rev: true, phase: 0.40, jumpH: 38, jumpDur: 0.84,
		spots: [2]float64{2*math.Pi - 0.30, math.Pi + 0.30}, every: [2]int{1, 2}, push: 2.6, pushLen: 0.5, pushBase: 0.7},
	{slug: "sam_sam", ride: 1, deck: 0xe83a3a, speed: 134, ax: 408, phase: 0.64, jumpH: 42, jumpDur: 0.74,
		spots: [2]float64{2*math.Pi - 0.33, math.Pi + 0.33}, every: [2]int{2, 1}, push: 2.2, pushLen: 0.45, pushBase: 1.3},
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
	span := r.speed * r.jumpDur
	for k := range r.spots {
		start := tb.spotD[k] - span/2
		lap := math.Floor((d - start) / tb.length)
		rel := d - start - lap*tb.length
		if rel < 0 || rel >= span || d-rel < r.phase*tb.length { // not yet, or a jump begun before he rolled in
			continue
		}
		n := int(lap)
		if n%r.every[k] != 0 {
			continue
		}
		switch r.slug {
		case "diego":
			kind = finTrickKickflip
			if (n+k)%2 == 1 {
				kind = finTrickOllie
			}
		case "cesar":
			kind = finTrickSpin
		default:
			kind = finTrickWhip
		}
		return kind, rel / span
	}
	return finTrickNone, 0
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
		s.h = math.Round(r.jumpH * 4 * s.p * (1 - s.p))
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
		// about to jump: a crouch just before the take-off
		if k, _ := finTrickAt(i, s.d+r.speed*0.18); k != finTrickNone {
			s.crouch, s.push = true, false
		}
	}
	return s
}

// finRideBox: the rectangle rider i covers at time t (kid, board and shadow), for the layout tests. Pure.
func finRideBox(i int, t float64) (x0, y0, x1, y1 float64, on bool) {
	s := finRideAt(i, t)
	if !s.on {
		return 0, 0, 0, 0, false
	}
	hw := finRideHW
	if s.trick == finTrickWhip {
		hw = finRideHWWhip
	}
	return s.x - hw, s.g - s.h - finRideH, s.x + hw, s.g + finRideSc, true // the shadow: 1 px under the ground line
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
	finScViews
)

type finRideArt struct {
	kids   [3]*CharSet
	skate  [finSkViews][4]*ebiten.Image
	scoot  [2][finScViews][4]*ebiten.Image // [0] César, [1] Sam-sam
	shadow [3]*ebiten.Image
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
	a := &finRideArt{}
	for i, r := range finRiders {
		if f := presetByName(r.slug); f != nil {
			spec := f(rand.New(rand.NewSource(7)))
			spec.Ride = 0 // he rides the big board drawn here
			a.kids[i] = newCharSet(&spec)
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
	bar(stem-1, 3)
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
)

const finRideKidCX = 12 // the middle column of a 24 px wide character frame

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
	board := finPart{sx: 1}
	var btop float64
	boardFirst := true
	drop := 0.0 // the board falls a little under his feet during a kickflip
	if r.ride == 2 {
		v := finSkFlat
		switch {
		case s.dir == DirUp || s.dir == DirDown:
			v = finSkEnd
		case s.trick == finTrickKickflip:
			switch {
			case s.p < 0.15:
				v = finSkNoseUp
			case s.p < 0.3:
				v, drop = finSkFaceDeck, 2
			case s.p < 0.45:
				v, drop = finSkUpside, 3
			case s.p < 0.6:
				v, drop = finSkFaceGrip, 2
			case s.p > 0.85:
				v = finSkNoseDown
			}
		case s.trick == finTrickOllie:
			switch {
			case s.p < 0.35:
				v = finSkNoseUp
			case s.p > 0.7:
				v = finSkNoseDown
			}
		}
		board = finPart{kind: finPartSkate, view: v, cx: finRideSkCX, ay: finRideSkGY}
		btop = finRideSkTop
	} else {
		v := finScSide
		switch {
		case s.trick == finTrickSpin: // César turns round with his scooter: side, front, other side, back
			switch int(s.p * 4) {
			case 1:
				kidDir, v, sx = DirDown, finScFront, 1
			case 2:
				sx = -sx
			case 3:
				kidDir, v, sx = DirUp, finScFront, 1
			}
		case s.trick == finTrickWhip: // Sam-sam swings the deck round the stem
			switch int(s.p * 5) {
			case 1, 3:
				v = finScEdge
			case 2:
				v = finScWhip
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
	// the kid: his feet on the deck; pushing he puts a foot down (a walking frame); bobbing he bends his knees
	kid := finPart{kind: finPartKid, anim: AnimIdle, dir: kidDir, cx: finRideKidCX, ay: finRideFoot, x: s.x, y: gy - (board.ay-btop)*sc, sx: sx}
	if s.push {
		kid.anim, kid.fr = AnimWalk, float64(s.pushF)
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
