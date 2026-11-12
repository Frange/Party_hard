package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Skate bowls: a sunken concrete pool seen from above (rim, sloping walls and a
// flat floor), in three shapes. They are walkable (the slope is only paint), so
// nobody is ever trapped in one. Only the player, the police, the killers and
// the three kids (Sam-sam, César and Diego) may step into them: the guests'
// navigation grid has the bowls closed (see PathG). Now and then a kid goes to
// a bowl, rides its walls and does tricks (ollies, spins, grabs, kickflips,
// flips, manuals, the odd fall) and leaves when he has had enough. While he
// rides he pays no attention to anybody (he does not run from the player), but
// he can be killed like always.
// ---------------------------------------------------------------------------

const (
	bowlW = 112
	bowlH = 64
)

func init() {
	propKinds["skatebowl"] = &PropKind{
		W: bowlW, H: bowlH, FW: 96, FH: 54, Vars: 3, NoSolid: true, SortOff: -300,
		FootDX: -48, FootDY: -58, hasFoot: true,
		// everything is walkable (the player has to be able to go down and reach the skater)
		Draw: drawSkateBowl,
	}
}

// bowlInside tells whether a normalised point (nx, ny in -1..1 of the bowl's box) is inside the shape.
func bowlInside(v int, nx, ny float64) bool {
	switch v % 3 {
	case 1: // kidney: two lobes with a notch on the far side
		a := (nx+0.36)*(nx+0.36)/0.5776 + ny*ny/0.9
		b := (nx-0.36)*(nx-0.36)/0.5776 + ny*ny/0.9
		if nx > -0.2 && nx < 0.2 && ny < -0.62+nx*nx*4 {
			return false
		}
		return a <= 1 || b <= 1
	case 2: // rounded square
		return math.Pow(math.Abs(nx), 4)+math.Pow(math.Abs(ny), 4) <= 1
	}
	return nx*nx+ny*ny <= 1
}

var bowlRings = []float64{1.0, 0.95, 0.88, 0.80, 0.72, 0.64, 0.56}

func drawSkateBowl(b *pixBuf, v int) {
	cx, cy := float64(bowlW)/2, float64(bowlH)/2
	rx, ry := float64(bowlW)/2-1, float64(bowlH)/2-1
	far := []uint32{0xe6e8ec, 0xc4c8d0, 0xa8acb6, 0x949aa4, 0x808690, 0x6e747e}
	rng := rand.New(rand.NewSource(int64(v)*131 + 7))
	level := make([]int8, bowlW*bowlH)
	for y := 0; y < bowlH; y++ {
		for x := 0; x < bowlW; x++ {
			lv := int8(-1)
			for i, s := range bowlRings {
				ccy := cy + (1-s)*9 // the floor sinks towards the near side
				if bowlInside(v, (float64(x)-cx)/(rx*s), (float64(y)-ccy)/(ry*s)) {
					lv = int8(i)
				}
			}
			level[y*bowlW+x] = lv
		}
	}
	for y := 0; y < bowlH; y++ {
		for x := 0; x < bowlW; x++ {
			lv := int(level[y*bowlW+x])
			if lv < 0 {
				continue
			}
			var col uint32
			switch {
			case lv == 0: // coping: the light rim, brighter on the far side
				col = shadeHex(0xeceef2, litK(float64(y), cy, 0.74))
			case lv == len(bowlRings)-1: // flat floor
				col = 0x464b53
				if (x*7+y*13)%23 == 0 {
					col = 0x555b64
				}
			default: // sloping wall: lit on the far side, in shadow on the near one
				col = shadeHex(far[lv-1], litK(float64(y), cy, 0.62))
			}
			b.set(x, y, c(col))
		}
	}
	// a dark line under the rim and the shadow of the far wall on the floor
	for y := 0; y < bowlH; y++ {
		for x := 0; x < bowlW; x++ {
			lv := int(level[y*bowlW+x])
			if lv == len(bowlRings)-1 && y > 0 && int(level[(y-1)*bowlW+x]) == len(bowlRings)-2 {
				b.set(x, y, c(0x2e3238))
			}
			if lv == 1 && y > 0 && level[(y-1)*bowlW+x] == 0 {
				b.set(x, y, c(0x9a9ea6))
			}
		}
	}
	// the seam of the plates on the walls
	for ang := 0.0; ang < 2*math.Pi; ang += math.Pi / 6 {
		for s := 0.9; s > 0.58; s -= 0.02 {
			x, y := int(cx+math.Cos(ang)*rx*s), int(cy+(1-s)*9+math.Sin(ang)*ry*s)
			if x >= 0 && y >= 0 && x < bowlW && y < bowlH && level[y*bowlW+x] > 1 && level[y*bowlW+x] < int8(len(bowlRings)-1) {
				b.set(x, y, shade(b.get(x, y), 0.9))
			}
		}
	}
	// graffiti on the far wall
	paints := []uint32{0xe83a8a, 0x30d0e0, 0xf0d030, 0x7a3ae0, 0x50e070}
	for n := 0; n < 5; n++ {
		x := int(cx) - 34 + rng.Intn(68)
		y := int(cy) - 17 + rng.Intn(6)
		col := paints[rng.Intn(len(paints))]
		for k := 0; k < 4+rng.Intn(5); k++ {
			px, py := x+k, y+(k%3)/2
			if px >= 0 && py >= 0 && px < bowlW && py < bowlH {
				if lv := level[py*bowlW+px]; lv >= 2 && lv < int8(len(bowlRings)-1) {
					b.set(px, py, c(col))
				}
			}
		}
	}
	// a dropped board and a puddle of paint on the floor
	b.rect(int(cx)+10, int(cy)+11, 8, 1, c(0xc04030))
	b.rect(int(cx)-22, int(cy)+9, 3, 2, c(0xe8e8f0))
	// a crack across the floor
	for k := 0; k < 14; k++ {
		x, y := int(cx)-12+k, int(cy)+13-(k*3)%5
		if x >= 0 && y >= 0 && x < bowlW && y < bowlH && level[y*bowlW+x] == int8(len(bowlRings)-1) {
			b.set(x, y, c(0x383c44))
		}
	}
}

// litK is how lit the bowl is at row y: full on the far side, falling smoothly to min on the near side.
func litK(y, cy, min float64) float64 {
	t := (y - (cy - 10)) / 26
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return 1 - (1-min)*t
}

// shadeHex darkens (or lightens) a 0xRRGGBB colour.
func shadeHex(col uint32, k float64) uint32 {
	cl := func(v float64) uint32 {
		if v > 255 {
			return 255
		}
		return uint32(v)
	}
	r, g, bl := float64(col>>16&0xff)*k, float64(col>>8&0xff)*k, float64(col&0xff)*k
	return cl(r)<<16 | cl(g)<<8 | cl(bl)
}

// ---- the skaters: kids riding the bowl -------------------------------------------------------------

const (
	bowlSteps = 32
	kidCanvas = 48
)

// the tricks he does at the top of a wall
const (
	trNone = iota
	trOllie
	trSpin180
	trSpin360
	trGrab
	trKickflip
	trFlip
	trWipeout
	trHandplant // upside down on one hand on the coping, then back in
	trSpin540
	trHeelflip
	trTailwhip // scooters only: the deck swings round the stem
	trBarspin  // scooters only: the handlebar spins
	trNoHander // scooters only: both hands up in the air
)

type skaterSt struct {
	pr        *Prop
	stage     int // 0 on his way to the bowl, 1 stepping in, 2 riding, 3 stepping out
	t         float64
	from, to  Vec
	until     float64 // when the routine ends
	cx, cy    float64 // where the feet are when he is in the middle of the bowl
	ph        float64
	speed     float64
	reach     float64
	swing     float64
	curN      int
	trick     int
	spin      float64 // direction of the spins
	fallT     float64 // seconds left of a wipeout
	fallX     float64
	rng       *rand.Rand
	spec      CharSpec
	deck      color.RGBA
	x, y, z   float64
	ang, bang float64
	face      float64
	pose      int
	lyingSide float64
	boardX    float64
	imgs      map[int]*ebiten.Image
	boardImgs map[int]*ebiten.Image
	scooter   bool    // César and Sam-sam ride their scooters in the bowl too
	view      int     // the scooter's view this frame (finScSide, finScWhip...)
	walkT     float64 // seconds on his way to the bowl (a safety timeout: he never gets stuck)
	sched     bool    // sent by the skate park's turn (skateParkTurn): a short visit
	tricks    int     // tricks done in this visit
	sayN      int     // lines said in this visit
	shout     string  // a line to put in his bubble (set by ride, said by kidBowl)
}

// the kids that are going to, riding or leaving a bowl
var (
	kidSessions = map[*Guest]*skaterSt{}
	kidBowlNext = map[*Guest]float64{}
)

// bowlRateDiv: with -bowlkids (0.5) every free kid goes to a bowl on his own about every half second; otherwise
// the kids only go by turns (skateTurnFor, skater_park.go) and this is not used
var bowlRateDiv = 14.0

const bowlFeetDY = -34.0 // the middle of the floor of the bowl, above the anchor of the sprite

// kidRiding: the kid is doing his tricks in a bowl right now.
func kidRiding(gu *Guest) bool {
	st := kidSessions[gu]
	return st != nil && st.stage == 2
}

// bowlGuest runs for every guest of a level with bowls. It keeps the ordinary guests out of them and
// drives the kids that have gone to ride one. It returns true when it has taken over the guest.
func (g *Game) bowlGuest(gu *Guest, dt float64) bool {
	if isKid(gu) {
		return g.kidBowl(gu, dt)
	}
	if g.lvl.InBowl(gu.pos) { // they cannot get in (their paths go round), but nothing may leave one in
		gu.pos, gu.path = g.lvl.outOfBowls(gu.pos), nil
	}
	return false
}

func newSkaterSt(gu *Guest, pr *Prop, rng *rand.Rand) *skaterSt {
	var spec CharSpec
	if gu.cs != nil && gu.cs.Spec != nil {
		spec = *gu.cs.Spec
		if gu.cs.rig != nil { // an edited (rig) kid has an almost empty spec: take the kid's own look
			if f := presetByName(spec.Name); f != nil {
				spec = f(rng)
			}
		}
	}
	deck := spec.RideCol
	if deck.A == 0 {
		deck = hx(0x3a8ae0)
	}
	scooter := spec.Ride == 1
	spec.Ride = 0 // the board is drawn on its own
	return &skaterSt{
		pr: pr, cx: pr.X, cy: pr.Y + bowlFeetDY, speed: 1.3 + rng.Float64()*0.8, reach: []float64{40, 44, 38}[pr.Var%3],
		swing: 4 + rng.Float64()*4, curN: -9999, face: 1, spin: []float64{1, -1}[rng.Intn(2)],
		rng: rand.New(rand.NewSource(rng.Int63())), spec: spec, deck: deck, scooter: scooter,
		imgs: map[int]*ebiten.Image{}, boardImgs: map[int]*ebiten.Image{},
	}
}

func (g *Game) endKidBowl(gu *Guest, wait float64) {
	if st := kidSessions[gu]; st != nil && st.sched {
		g.skateTurnDone()
	}
	delete(kidSessions, gu)
	kidBowlNext[gu] = g.time + wait + g.rng.Float64()*20
}

// kidTurnState: the states a kid may be taken from when it is his turn to go to a bowl.
func kidTurnState(s gState) bool {
	switch s {
	case gIdle, gGoto, gDance, gDanceWith, gChatGo, gWaitP, gChat:
		return true
	}
	return false
}

// kidBowlWalkMax: a kid who has not reached his bowl after this many seconds gives up (and rolls on as always).
const kidBowlWalkMax = 25.0

// rimReach: whether a kid at p may glide straight to the rim point q (the last bit of his way, over ground the
// guests' grid leaves closed round the bowl): close enough and with nothing solid in between.
func (l *Level) rimReach(p, q Vec) bool {
	if dist(p, q) > 64 {
		return false
	}
	for _, r := range l.solidsP {
		if segHitsRect(p, q, r) {
			return false
		}
	}
	return true
}

// kidBowl is the whole life of a kid in a bowl: he decides to go, walks there, steps in, rides
// for a random time and steps out. If something scares him on the way, he drops it and runs as always.
func (g *Game) kidBowl(gu *Guest, dt float64) bool {
	st := kidSessions[gu]
	if st == nil {
		if g.mode != ModePlay || g.lvl.InBowl(gu.pos) {
			return false
		}
		// the skate park sends one of the three every few seconds (skater_park.go): standing, rolling about, dancing
		// or chatting (he leaves his partner there: a kid's attention span)
		turn := kidTurnState(gu.state) && g.skateTurnFor(gu)
		// besides the turns, a kid only goes on his own with -bowlkids (debug: all of them, all the time)
		if !turn && (bowlRateDiv >= 1 || gu.partner != nil || gu.state != gIdle || g.time < kidBowlNext[gu] || g.rng.Float64() > dt/bowlRateDiv) {
			return false
		}
		var bowls []*Prop
		for _, p := range g.lvl.Props {
			if p.Name == "skatebowl" {
				bowls = append(bowls, p)
			}
		}
		if len(bowls) == 0 {
			return false
		}
		// the nearest bowl he can ride to (a turn); a random one otherwise
		pr := bowls[g.rng.Intn(len(bowls))]
		if turn { // the nearest one nobody is riding
			used := map[*Prop]bool{}
			for _, o := range kidSessions {
				used[o.pr] = true
			}
			best := -1.0
			for _, b := range bowls {
				if d := dist(Vec{b.X, b.Y}, gu.pos); !used[b] && (best < 0 || d < best) {
					pr, best = b, d
				}
			}
			if best < 0 {
				return false // all of them taken: wait
			}
		}
		z := bowlZone(pr)
		q := Vec{math.Max(z.X, math.Min(z.X+z.W, gu.pos.X)), math.Max(z.Y, math.Min(z.Y+z.H, gu.pos.Y))}
		out := Vec{gu.pos.X - q.X, gu.pos.Y - q.Y}
		if n := math.Hypot(out.X, out.Y); n > 0 {
			q = Vec{q.X + out.X/n*9, q.Y + out.Y/n*9}
		}
		path := g.lvl.PathG(gu.pos, q)
		if len(path) == 0 && !g.lvl.rimReach(gu.pos, q) { // no way there: not now
			if turn {
				g.skateTurnDone()
			}
			kidBowlNext[gu] = g.time + 5
			return false
		}
		g.detachPartner(gu)
		st = newSkaterSt(gu, pr, g.rng)
		st.to, st.sched = q, turn
		if turn {
			g.skateTurnStarted(gu)
		}
		gu.path = path
		gu.state, gu.timer = gIdle, 60
		kidSessions[gu] = st
		return false
	}
	// something else has taken hold of him (frightened, abducted, turned into a zombie...): he gives up
	// the bowl. Only the tricks themselves are protected.
	if gu.state != gIdle && !(st.stage == 2 && gu.state != gLift && gu.state != gZombie && gu.state != gKiller && gu.state != gPunch) {
		g.endKidBowl(gu, 10)
		return false
	}
	switch st.stage {
	case 0: // walking to the edge of the bowl
		gu.timer = 60
		gu.walk(guestSpeed, dt)
		st.walkT += dt
		// a safety net: never stuck on the way, never sliding over anything to the rim
		if st.walkT > kidBowlWalkMax || (len(gu.path) == 0 && !g.lvl.rimReach(gu.pos, st.to)) {
			gu.path = nil
			g.endKidBowl(gu, 6)
			return false
		}
		if len(gu.path) == 0 || dist(gu.pos, st.to) < 6 {
			st.stage, st.t, st.from = 1, 0, gu.pos
			if gu.pos.X < st.cx {
				st.ph = math.Asin(math.Max(-0.9, math.Min(0.9, (gu.pos.X-st.cx)/st.reach)))
			} else {
				st.ph = math.Pi - math.Asin(math.Max(-0.9, math.Min(0.9, (gu.pos.X-st.cx)/st.reach)))
			}
			st.x = st.reach * math.Sin(st.ph)
			st.y = st.swing * math.Sin(2*st.ph+1.3)
			st.to = Vec{st.cx + st.x, st.cy + st.y}
		}
	case 1, 3: // stepping in / out over the rim
		st.t += dt
		u := math.Min(1, st.t/0.7)
		gu.pos = Vec{st.from.X + (st.to.X-st.from.X)*u, st.from.Y + (st.to.Y-st.from.Y)*u}
		gu.moving, gu.curSpd = true, guestSpeed*kidSpeedK
		gu.anim += dt * stridePhase(gu.curSpd)
		if st.to.X > st.from.X {
			gu.dir = DirRight
		} else {
			gu.dir = DirLeft
		}
		if u < 1 {
			break
		}
		if st.stage == 1 {
			st.stage = 2
			st.until = g.time + 18 + g.rng.Float64()*27
			if st.sched { // the park's turn: in, up the wall, a big trick or two, and out again
				st.until = g.time + 10.5 + g.rng.Float64()*3.5
			}
		} else {
			gu.state, gu.timer, gu.path = gIdle, 0.3, nil
			g.endKidBowl(gu, 12)
		}
	case 2:
		say, sayT := gu.icon, gu.iconT
		gu.state, gu.icon, gu.iconT, gu.path, gu.moving, gu.timer = gIdle, "", 0, nil, false, 60
		if st.sayN > 0 && sayT > 0 { // only his own shouts stay up while he rides
			gu.icon, gu.iconT = say, sayT-dt
		}
		st.z, st.ang, st.bang, st.pose, st.view = 0, 0, 0, 0, finScSide
		if st.fallT > 0 {
			st.fallT -= dt
			g.skaterFall(st, dt)
		} else {
			st.ph += dt * st.speed
			st.ride(g)
		}
		if st.shout != "" {
			gu.icon, gu.iconT = st.shout, 1.8
			st.sayN++
			st.shout = ""
		}
		gu.pos = Vec{st.cx + st.x, st.cy + st.y}
		if st.face < 0 {
			gu.dir = DirLeft
		} else {
			gu.dir = DirRight
		}
		gu.anim += dt * 6
		if g.time >= st.until && st.fallT <= 0 && st.pose == 0 && st.z < 1 {
			st.stage, st.t, st.from = 3, 0, gu.pos
			side := 1.0
			if gu.pos.X < st.cx {
				side = -1
			}
			st.to = Vec{st.cx + side*(bowlW/2+9), st.cy + 14}
		}
	}
	return true
}

// pickTrick draws the trick for the next wall: some ollies and grabs, a few flips, the odd fall.
func (st *skaterSt) pickTrick() int {
	if st.sched && st.tricks == 0 { // a visit of the park's turn starts with a big one
		return []int{trHandplant, trSpin540, trFlip}[st.rng.Intn(3)]
	}
	if st.scooter { // César and Sam-sam: scooter tricks
		return []int{trNone, trOllie, trOllie, trSpin180, trSpin360, trTailwhip, trTailwhip, trBarspin, trBarspin,
			trNoHander, trNoHander, trHandplant, trSpin540, trFlip, trWipeout}[st.rng.Intn(15)]
	}
	switch r := st.rng.Intn(20); {
	case r < 2:
		return trHandplant
	case r < 4:
		return trHeelflip
	case r < 5:
		return trSpin540
	}
	r := st.rng.Intn(16)
	switch {
	case r < 2:
		return trNone
	case r < 5:
		return trOllie
	case r < 7:
		return trSpin180
	case r < 9:
		return trSpin360
	case r < 11:
		return trGrab
	case r < 13:
		return trKickflip
	case r < 15:
		return trFlip
	}
	return trWipeout
}

// ride: back and forth across the bowl, up the walls at both ends, a trick at the top of each one.
func (st *skaterSt) ride(g *Game) {
	ph := st.ph
	s := math.Sin(ph)
	st.x = st.reach * s
	st.y = st.swing * math.Sin(2*ph+1.3)
	u := math.Abs(s)
	wall := 0.0
	if u > 0.55 {
		wall = math.Pow((u-0.55)/0.45, 1.5)
	}
	st.z = 17 * wall
	st.ang = -math.Copysign(0.5*wall, s) // the board follows the slope of the wall
	st.face = 1
	if math.Cos(ph) < 0 {
		st.face = -1
	}
	st.bang = st.ang
	// a manual while he crosses the middle: nose up on two wheels
	if u < 0.18 && st.trick != trWipeout && int(math.Abs(math.Round((ph-math.Pi/2)/math.Pi)))%3 == 1 {
		st.ang = -0.32 * st.face
		st.bang = st.ang
		st.z = 0
	}
	n := int(math.Round((ph - math.Pi/2) / math.Pi))
	d := ph - (float64(n)*math.Pi + math.Pi/2)
	if math.Abs(d) >= 0.3 {
		return
	}
	if st.curN != n {
		st.curN = n
		st.trick = st.pickTrick()
		if st.trick != trNone {
			st.tricks++
			// now and then he shouts the trick's name, or one of his jokes (kidBowl puts it in his bubble)
			if st.rng.Float64() < 0.4 {
				st.shout = skaterTrickShout[st.trick][0]
				if st.rng.Float64() < 0.4 {
					l := skaterLines(st.spec.Name)
					st.shout = l[st.rng.Intn(len(l))][0]
				}
			}
		}
	}
	t := (d + 0.3) / 0.6
	air := math.Sin(math.Pi * t)
	switch st.trick {
	case trHandplant: // up over the coping, upside down on one hand, and back in
		st.z += 30 * air
		st.pose = 3 // arms over his head: upside down, they reach for the rim
		k := math.Min(1, math.Min(t/0.3, (1-t)/0.3))
		st.ang += st.face * math.Pi * k
		st.bang = st.ang
	case trSpin540:
		st.z += 26 * air
		st.pose = 1
		if (t > 0.25 && t < 0.5) || t > 0.75 {
			st.face = -st.face
		}
	case trHeelflip:
		st.z += 22 * air
		st.pose = 1
		st.bang = -st.spin * 2 * math.Pi * t
	case trTailwhip:
		st.z += 22 * air
		st.pose = 1
		switch int(t * 5) {
		case 1, 3:
			st.view = finScEdge
		case 2:
			st.view = finScWhip
		}
	case trBarspin:
		st.z += 22 * air
		switch int(t * 5) {
		case 1, 3:
			st.view, st.pose = finScBarN, 3
		case 2:
			st.view, st.pose = finScBarW, 3
		default:
			st.pose = 1
		}
	case trNoHander:
		st.z += 24 * air
		st.pose = 1
		if t > 0.2 && t < 0.8 {
			st.pose = 3
		}
	case trOllie:
		st.z += 22 * air
		st.pose = 1
	case trSpin180:
		st.z += 22 * air
		st.pose = 1
		if t > 0.5 {
			st.face = -st.face
		}
	case trSpin360:
		st.z += 22 * air
		st.pose = 1
		if t > 0.33 && t < 0.66 {
			st.face = -st.face
		}
	case trGrab:
		st.z += 22 * air
		st.pose = 2
		st.ang += st.spin * 0.35 * air
		st.bang = st.ang + st.spin*0.5*air
	case trKickflip:
		st.z += 22 * air
		st.pose = 1
		st.bang = st.spin * 2 * math.Pi * t
	case trFlip:
		st.z += 26 * air
		st.pose = 1
		st.ang += st.spin * 2 * math.Pi * t
		st.bang = st.ang
	case trWipeout:
		st.z += 20 * air
		st.pose = 1
		st.ang += st.spin * 0.8 * t
		st.bang = st.ang
		if t >= 0.5 {
			st.fallT = 2.4
			st.fallX = st.x
			st.lyingSide = st.spin
			st.boardX = st.x
		}
	}
}

// skaterFall: he lands badly, slides, lies there a moment and gets up again.
func (g *Game) skaterFall(st *skaterSt, dt float64) {
	const total = 2.4
	el := total - st.fallT
	slide := -math.Copysign(1, st.fallX)
	switch {
	case el < 0.35: // hits the wall and drops
		st.x = st.fallX + slide*4*el
		st.z = math.Max(0, 14*(1-el/0.35))
		st.ang = st.lyingSide * (math.Pi / 2) * (el / 0.35)
		st.pose = 1
	case el < 1.7: // lying on the floor, the board rolls away
		st.x += slide * 6 * dt
		st.ang = st.lyingSide * math.Pi / 2
		st.pose = 4
		st.boardX += slide * 22 * dt
	default: // getting up
		st.ang = st.lyingSide * (math.Pi / 2) * (1 - (el-1.7)/0.7)
		st.pose = 4
	}
	st.x = math.Max(-st.reach, math.Min(st.reach, st.x))
	st.boardX = math.Max(-st.reach, math.Min(st.reach, st.boardX))
	st.bang = st.ang
	st.face = -slide
	if st.fallT <= 0 { // back on the board: carry on towards the middle
		ratio := math.Max(-1, math.Min(1, st.x/st.reach))
		if st.x > 0 {
			st.ph = math.Pi - math.Asin(ratio)
		} else {
			st.ph = -math.Pi - math.Asin(ratio)
		}
		st.curN = int(math.Round((st.ph - math.Pi/2) / math.Pi))
		st.trick = trNone
	}
}

func (st *skaterSt) kidImage(pose, step int) *ebiten.Image {
	key := pose*100 + step
	if im, ok := st.imgs[key]; ok {
		return im
	}
	p := poseFor(AnimIdle, 0)
	switch pose {
	case 1: // in the air: arms out, knees a bit bent
		p.LA, p.RA = Limb{7, 2, 0}, Limb{7, 2, 0}
		p.LF, p.RF = Limb{2, 9, 0}, Limb{2, 9, 0}
	case 2: // grabbing the board
		p.LA, p.RA = Limb{3, 10, 2}, Limb{3, 10, 2}
		p.LF, p.RF = Limb{2.5, 7, 0}, Limb{2.5, 7, 0}
		p.Bob = 2
	case 3: // both arms up: no hands (or, upside down in a handplant, reaching for the rim)
		p.LA, p.RA = Limb{4, -8, 0}, Limb{5, -7, 0}
		p.LF, p.RF = Limb{2, 9, 0}, Limb{2, 9, 0}
		p.Open = true
	case 4: // standing
	default: // riding: knees bent, arms a bit apart
		p.LA, p.RA = Limb{4, 6, 0}, Limb{4, 6, 0}
		p.Bob = 1
	}
	spec := st.spec
	src := genChar(&spec, vSide, p).outlinedSel(0.42)
	im := rotateAround(src, float64(step)/bowlSteps*2*math.Pi, float64(src.w)/2, 30).img()
	st.imgs[key] = im
	return im
}

// boardImage: the skateboard on its own (so it can flip or roll away from the kid).
func (st *skaterSt) boardImage(step int) *ebiten.Image {
	key := step
	if st.scooter {
		key += 100 * (st.view + 1)
	}
	if im, ok := st.boardImgs[key]; ok {
		return im
	}
	if st.scooter { // the same scooter as on the last card, its deck where a skateboard's would be
		sc := finDrawScooter(st.view, 0, st.deck)
		im := rotateAround(sc.outlinedSel(0.42), float64(step)/bowlSteps*2*math.Pi, finRideScCX, finRideScTop+1).img()
		st.boardImgs[key] = im
		return im
	}
	b := newPix(24, 24)
	deck := st.deck
	b.rect(5, 11, 14, 2, deck)
	b.rect(5, 11, 14, 1, shade(deck, 1.25))
	b.set(4, 10, deck)
	b.set(19, 10, deck)
	b.rect(7, 13, 2, 1, hx(0xf0f0f0))
	b.rect(15, 13, 2, 1, hx(0xf0f0f0))
	im := rotateAround(b.outlinedSel(0.42), float64(step)/bowlSteps*2*math.Pi, 12, 12).img()
	st.boardImgs[key] = im
	return im
}

// rotateAround turns a sprite (nearest neighbour) around a pivot into a square canvas, clockwise on screen.
func rotateAround(src *pixBuf, th, pvx, pvy float64) *pixBuf {
	return rotateAroundN(src, th, pvx, pvy, kidCanvas)
}

// rotateAroundN: rotateAround into a square canvas of side n (the pivot lands in its middle).
func rotateAroundN(src *pixBuf, th, pvx, pvy float64, n int) *pixBuf {
	cs, sn := math.Cos(th), math.Sin(th)
	out := newPix(n, n)
	half := float64(n) / 2
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := float64(x)-half+0.5, float64(y)-half+0.5
			sx := int(math.Floor(pvx + dx*cs + dy*sn))
			sy := int(math.Floor(pvy - dx*sn + dy*cs))
			if sx < 0 || sy < 0 || sx >= src.w || sy >= src.h {
				continue
			}
			i := (sy*src.w + sx) * 4
			if src.p[i+3] > 0 {
				j := (y*n + x) * 4
				copy(out.p[j:j+4], src.p[i:i+4])
			}
		}
	}
	return out
}

func angleStep(a float64) int {
	t := math.Mod(math.Mod(a, 2*math.Pi)+2*math.Pi, 2*math.Pi)
	return int(math.Round(t/(2*math.Pi)*bowlSteps)) % bowlSteps
}

// drawSkater: the kid and his board, with a shadow on the floor of the bowl.
func (g *Game) drawSkater(dst *ebiten.Image, gu *Guest) {
	st := kidSessions[gu]
	if st == nil {
		return
	}
	px, py := g.wx(gu.pos.X), g.wy(gu.pos.Y)
	r := math.Max(3, 8-st.z*0.18)
	filledEllipse(dst, px, py+1, r, r*0.4, color.RGBA{0, 0, 0, uint8(80 - math.Min(45, st.z))})
	face := st.face
	// kid: the picture is flipped when he faces left, so its own angle mirrors
	kstep := angleStep(st.ang * face)
	im := st.kidImage(st.pose, kstep)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-kidCanvas/2, -kidCanvas/2)
	op.GeoM.Scale(face, 1)
	op.GeoM.Translate(math.Round(px), math.Round(py-13-st.z))
	// board under the feet: it turns with him (a flip, a handplant), so it stays under his feet, 12 px from his
	// middle along his body; or it lies where it rolled to after a fall
	aq := float64(kstep) / bowlSteps * 2 * math.Pi
	bx, by := px-face*12*math.Sin(aq), py-13-st.z+12*math.Cos(aq)+1
	if st.fallT > 0 && st.fallT < 2.05 {
		bx = g.wx(st.cx + st.boardX)
		by = py
	}
	bi := st.boardImage(angleStep(st.bang * face))
	bop := &ebiten.DrawImageOptions{}
	bop.GeoM.Translate(-kidCanvas/2, -kidCanvas/2)
	bop.GeoM.Scale(face, 1)
	bop.GeoM.Translate(math.Round(bx), math.Round(by-1))
	if st.scooter { // the scooter behind him: his feet on its deck, his hands on its bar
		dst.DrawImage(bi, bop)
		dst.DrawImage(im, op)
		return
	}
	dst.DrawImage(im, op)
	dst.DrawImage(bi, bop)
}

// ---- who may step into a bowl ------------------------------------------------------------------

// bowlZone is the ground a bowl covers.
func bowlZone(pr *Prop) Rect { return Rect{pr.X - bowlW/2, pr.Y - bowlH, bowlW, bowlH} }

// initBowlNav finds the bowls of the level and builds the navigation grid of the guests: the same as
// everybody's, but with the bowls closed. The player, the kids, the police and the killers use the
// normal grid, so they can walk in and out; guests are kept out of the bowls on purpose.
func (l *Level) initBowlNav() {
	l.bowls, l.blockedG = nil, nil
	for _, p := range l.Props {
		if p.Name == "skatebowl" {
			l.bowls = append(l.bowls, bowlZone(p))
		}
	}
	if len(l.bowls) == 0 {
		return
	}
	l.blockedG = append([]bool(nil), l.blocked...)
	for cy := 0; cy < l.ch; cy++ {
		for cx := 0; cx < l.cw; cx++ {
			cell := Rect{float64(cx) * navCell, float64(cy) * navCell, navCell, navCell}
			for _, r := range l.bowls {
				if ox, oy := rectOv(cell, r.Grow(2)); ox > 0 && oy > 0 { // any cell that touches the bowl
					l.blockedG[cy*l.cw+cx] = true
				}
			}
		}
	}
}

// InBowl: is the point inside a bowl?
func (l *Level) InBowl(p Vec) bool {
	for _, r := range l.bowls {
		if r.Has(p.X, p.Y) {
			return true
		}
	}
	return false
}

// outOfBowls pushes a point that is inside a bowl out through the nearest edge.
func (l *Level) outOfBowls(p Vec) Vec {
	for _, r := range l.bowls {
		if !r.Has(p.X, p.Y) {
			continue
		}
		l2, r2, u, d := p.X-r.X, r.X+r.W-p.X, p.Y-r.Y, r.Y+r.H-p.Y
		m := math.Min(math.Min(l2, r2), math.Min(u, d))
		const gap = 8
		switch m {
		case d:
			p.Y = r.Y + r.H + gap
		case u:
			p.Y = r.Y - gap
		case l2:
			p.X = r.X - gap
		default:
			p.X = r.X + r.W + gap
		}
	}
	return p
}

// PathG is Path for the guests: it never goes through a bowl, and a goal inside one is moved to its edge.
func (l *Level) PathG(from, to Vec) []Vec {
	if len(l.bowls) == 0 || l.blockedG == nil {
		return l.Path(from, to)
	}
	save := l.blocked
	l.blocked = l.blockedG
	p := l.Path(l.outOfBowls(from), l.outOfBowls(to))
	l.blocked = save
	return p
}
