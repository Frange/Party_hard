package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The bedroom with two beds (the two player version of the intro and of the endings): the blue one
// sleeps (or tries to) on the left, the red one on the right, with a table with the alarm clock
// between them. Same room, same window onto the neighbours' party.
// ---------------------------------------------------------------------------

const (
	bedTopY, bedBotY     = 268.0, 440.0
	bed1X0, bed1X1       = 154.0, 384.0 // the left mattress (head on the left)
	bed2X0, bed2X1       = 576.0, 806.0 // the right mattress (head on the right)
	bedSideTableX0       = 452.0
	bedSideTableX1       = 508.0
	bedSideTableTop      = 330.0
	twoBedSleeperScale   = 5.0
	twoBedCenterY        = (bedTopY + bedBotY) / 2
	bedHeadW             = 58.0
	bedHead1X, bedHead2X = bed1X0 + 8 + bedHeadW/2, bed2X1 - 8 - bedHeadW/2 // the middle of each pillow: where each head rests (bubbles, gags)
)

// twoBedsApart: the two mattresses never touch and the table fits between them.
func twoBedsApart() bool {
	return bed2X0-bed1X1 > bedSideTableX1-bedSideTableX0+40
}

func (g *Game) drawTwoBeds(dst *ebiten.Image, t float64, quiet, gold bool) {
	// ---- the table in the middle: the alarm clock and (in the intro) the two glasses
	fillRect(dst, bedSideTableX0, bedSideTableTop, bedSideTableX1-bedSideTableX0, 70, rgb(0x2e2119))
	fillRect(dst, bedSideTableX0, bedSideTableTop, bedSideTableX1-bedSideTableX0, 4, rgb(0x4a3528))
	fillRect(dst, bedSideTableX0+8, bedSideTableTop+22, 40, 16, rgb(0x14100c))
	fillRect(dst, bedSideTableX0+4, bedSideTableTop-26, 48, 18, rgb(0x0a0a0e)) // alarm clock
	clock := "3:47"
	if quiet {
		clock = "6:30"
	}
	g.drawText(dst, clock, bedSideTableX0+9, bedSideTableTop-22, rgb(0xff3030), 0.9)
	fillRect(dst, bedSideTableX0+4, bedSideTableTop+66, 6, 22, rgb(0x2a1e18))
	fillRect(dst, bedSideTableX1-10, bedSideTableTop+66, 6, 22, rgb(0x2a1e18))
	// ---- the beds
	g.drawOneBed(dst, 0)
	g.drawOneBed(dst, 1)
	// ---- the sleepers (if they are in bed) with their blankets over their legs
	for i := 0; i < 2; i++ {
		ps := bedPoses[i]
		if !g.bedEmpty && !ps.away {
			if ps.sit > 0.4 {
				g.sitterTwo(dst, i, ps.sit)
			} else {
				g.sleeperTwo(dst, i, g.time, ps.turn)
			}
		}
		g.drawBlanket(dst, i, g.time)
		if p := ps.pillow; p > 0.01 && !g.bedEmpty && !ps.away { // the pillow over the head
			hx := bedHead1X + 4
			if i == 1 {
				hx = bedHead2X - 4
			}
			w, h := 58.0, 70.0
			py := twoBedCenterY - h/2 - 14*(1-p)
			fillRect(dst, hx-w/2, py, w, h, rgb(0xd8dee8))
			fillRect(dst, hx-w/2, py, w, 4, rgb(0xf0f4fa))
			fillRect(dst, hx-w/2, py+h-5, w, 5, rgb(0xa8b0c2))
			fillRect(dst, hx-w/2, py, 4, h, rgb(0xb8c0d0))
		}
	}
	// the colours of the party on the sheets
	if !quiet {
		pc := partyColors[(int(t*2.4)+1)%len(partyColors)]
		if bedPumpOn {
			pc = partyColors[bedBeatN%len(partyColors)]
		}
		fillRect(dst, bed1X0, bedTopY, bed1X1-bed1X0, bedBotY-bedTopY, rgba(pc, 14))
		fillRect(dst, bed2X0, bedTopY, bed2X1-bed2X0, bedBotY-bedTopY, rgba(pc, 14))
	}
}

// drawOneBed paints bed i (0 = left, with the headboard on its left; 1 = right, mirrored).
func (g *Game) drawOneBed(dst *ebiten.Image, i int) {
	x0, x1 := bed1X0, bed1X1
	hb, fb := x0-14, x1 // headboard and footboard x
	pill := x0 + 8
	if i == 1 {
		x0, x1 = bed2X0, bed2X1
		hb, fb = x1, x0-12
		pill = x1 - 8 - bedHeadW
	}
	fillRect(dst, hb, 250, 14, 190, rgb(0x3a2a1e)) // headboard
	fillRect(dst, hb, 250, 14, 3, rgb(0x5a4030))
	fillRect(dst, x0, bedTopY, x1-x0, bedBotY-bedTopY, rgb(0xaab4cc)) // mattress
	fillRect(dst, x0, bedTopY, x1-x0, 6, rgb(0xc4cce0))
	fillRect(dst, x0, bedBotY-6, x1-x0, 6, rgb(0x6a7690))
	fillRect(dst, fb, 300, 12, 150, rgb(0x3a2a1e)) // footboard
	fillRect(dst, hb, bedBotY+4, 8, 18, rgb(0x2a1e18))
	fillRect(dst, fb, bedBotY+4, 8, 18, rgb(0x2a1e18))
	fillRect(dst, pill, bedTopY+10, bedHeadW, bedBotY-bedTopY-20, rgb(0xa2aabb)) // pillow
	fillRect(dst, pill, bedTopY+10, bedHeadW, 4, rgb(0xc4ccdc))
	fillRect(dst, pill, bedBotY-14, bedHeadW, 4, rgb(0x8a92a4))
}

// the blanket over the legs: blue for the first player, red for the second
func (g *Game) drawBlanket(dst *ebiten.Image, i int, t float64) {
	wob := math.Sin(t*2+float64(i)*1.3) * 1.5
	x0, x1 := bed1X0+130, bed1X1
	main, light, dark := rgb(0x3a6aa0), rgb(0x6a9ac8), rgb(0x2c5688)
	if i == 1 {
		x0, x1 = bed2X0, bed2X1-130
		main, light, dark = rgb(0xa03a3a), rgb(0xc86a6a), rgb(0x782c2c)
	}
	fillRect(dst, x0, 320+wob, x1-x0, 110, main)
	fillRect(dst, x0, 320+wob, x1-x0, 4, light)
	for x := x0 + 24; x < x1-10; x += 44 {
		fillRect(dst, x, 326+wob, 2, 98, dark)
	}
	fillRect(dst, x0, 424, x1-x0, 6, shade(dark, 0.85))
}

// sleeperTwo draws player i lying in his bed. Both of them toss and turn, never in step.
func (g *Game) sleeperTwo(dst *ebiten.Image, i int, t, period float64) {
	cs := g.sceneHeroBed(i)
	if cs == nil {
		return
	}
	const turn = 0.9
	if period <= 0 { // the patient one: he does not turn, he lies on his back
		period = 1e9
	}
	tt := t + float64(i)*2.7 // out of step with each other
	n := int(tt / period)
	ph := math.Mod(tt, period)
	calm := g.allTrophies() || g.bedQuiet
	if calm {
		n, ph = 0, 0
	}
	flip, lift := 1.0, 0.0
	view := vFront
	if n%2 == 1 {
		view = vBack
	}
	if ph > period-turn {
		u := (ph - (period - turn)) / turn
		flip = math.Cos(math.Pi * u)
		lift = math.Sin(math.Pi*u) * 14
		if u > 0.5 {
			if n%2 == 0 {
				view = vBack
			} else {
				view = vFront
			}
		}
	}
	im := cs.frameImg(AnimSleep, view, 0)
	if im == nil {
		return
	}
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	long := math.Max(w, h)
	sc := math.Max(2, math.Floor((bed1X1-bed1X0-10)/long)) // nearly as long as the bed
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(sc, sc)
	cx, rot := sleeperPlacement(i, long, float64(spriteHeadMargin(im)), sc)
	if h > w { // a standing sprite: lay it down with the head to the pillow
		op.GeoM.Scale(flip, 1)
		op.GeoM.Rotate(rot)
	} else { // already lying, head on the left: the second player is mirrored so his head is on the right
		mir := 1.0
		if i == 1 {
			mir = -1
		}
		op.GeoM.Scale(mir, flip)
	}
	op.GeoM.Translate(cx, twoBedCenterY-lift+math.Sin(t*1.6+float64(i))*1.5)
	dst.DrawImage(im, op)
}

// sleeperPlacement: where the middle of the lying sprite goes and how much it is turned, so that the HEAD
// is on the pillow and the feet point to the foot of the bed: for the first player (head on the left) the
// sprite is turned a quarter counter-clockwise (its top goes left), for the second (head on the right)
// a quarter clockwise. long is the sprite's length (its height when standing), top the empty rows over
// the head, sc the scale.
// sleeperHeadRows: how far from the first painted row (or column) the middle of the head is, in sprite pixels.
const sleeperHeadRows = 3.3

func sleeperPlacement(i int, long, top, sc float64) (cx, rot float64) {
	headC := (top + sleeperHeadRows) * sc // from the head end of the sprite to the middle of the head
	if i == 0 {
		return bedHead1X - headC + long*sc/2, -math.Pi / 2
	}
	return bedHead2X + headC - long*sc/2, math.Pi / 2
}

// spriteHeadMargin: how many empty rows (standing sprite: from the top) or columns (lying sprite: from the
// left, where the head is) there are before the character starts (cached).
var headMarginCache = map[*ebiten.Image]int{}

func spriteHeadMargin(im *ebiten.Image) int {
	if v, ok := headMarginCache[im]; ok {
		return v
	}
	w, h := im.Bounds().Dx(), im.Bounds().Dy()
	pix := make([]byte, 4*w*h)
	m := -1
	func() {
		defer func() { recover() }() // no pixels to read (not in a running game): assume a usual margin
		im.ReadPixels(pix)
		if h > w {
			for y := 0; y < h && m < 0; y++ {
				for x := 0; x < w; x++ {
					if pix[(y*w+x)*4+3] > 0 {
						m = y
						break
					}
				}
			}
			return
		}
		for x := 0; x < w && m < 0; x++ {
			for y := 0; y < h; y++ {
				if pix[(y*w+x)*4+3] > 0 {
					m = x
					break
				}
			}
		}
	}()
	if m < 0 {
		m = 8
	}
	headMarginCache[im] = m
	return m
}

// sceneHeroBed is the look used lying in bed: the player's own, or a red wash over the first one's for the second.
func (g *Game) sceneHeroBed(i int) *CharSet {
	return g.sceneHero(i, false)
}

// sitterTwo draws player i sitting up in bed (his blanket, drawn after him, hides his legs).
func (g *Game) sitterTwo(dst *ebiten.Image, i int, sit float64) {
	cs := g.sceneHero(i, false)
	if cs == nil {
		return
	}
	x := bed1X1 - 60.0
	if i == 1 {
		x = bed2X0 + 60
	}
	y := 372 + (1-sit)*40
	g.sceneChar(dst, cs, AnimIdle, DirDown, 0, x, y, 3.6, 1, 0, 0, [3]float32{1, 1, 1})
}

// ---------------------------------------------------------------------------
// The main menu with two players: the same two beds, slid to the right so that the first headboard stays
// clear of the list of options (the list takes the left 250 px of the room).
// ---------------------------------------------------------------------------

const (
	menuBedShiftX = 114.0
	menuListRight = 252.0 // the options of the list end before this x
)

// menuBedsX returns the x range of every mattress of the room of the menu: one for one player, two (the
// beds of bedroom2p.go slid by menuBedShiftX) for two.
func menuBedsX(two bool) [][2]float64 {
	if !two {
		return [][2]float64{{334, 712}}
	}
	return [][2]float64{{bed1X0 + menuBedShiftX, bed1X1 + menuBedShiftX}, {bed2X0 + menuBedShiftX, bed2X1 + menuBedShiftX}}
}

var bedLayer *ebiten.Image // the two beds are painted here and slid onto the room

// drawTwoBedsShifted draws the beds of the two player room slid by (dx, dy).
func (g *Game) drawTwoBedsShifted(dst *ebiten.Image, t float64, quiet, gold bool, dx, dy float64) {
	if dx == 0 && dy == 0 {
		g.drawTwoBeds(dst, t, quiet, gold)
		return
	}
	if bedLayer == nil {
		bedLayer = ebiten.NewImage(screenW, screenH)
	}
	bedLayer.Clear()
	g.drawTwoBeds(bedLayer, t, quiet, gold)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(dx, dy)
	dst.DrawImage(bedLayer, op)
}

// sleeperTalkState: which of the three things the first sleeper does at time t (1 Zs, 2 muttering, 3 growl)
// and how many times he has turned in bed.
func sleeperTalkState(t float64, calm bool) (n, kind int) {
	const period, turn = 6.8, 0.9
	n = int(t / period)
	ph := math.Mod(t, period)
	switch {
	case calm, ph < 2.3:
		kind = 1
	case ph < period-turn:
		kind = 2
	default:
		kind = 3
	}
	if calm {
		n = 0
	}
	return n, kind
}

// menuBedTop is the top of the headboards of the beds in the room (y=250 before any shift).
const menuBedTop = 250.0

// menuBedShiftYFor: how far down the two beds slide so that the JUGADORES row (the longest of the list, it
// has the colour squares), at y, ends above them when it is level with the beds (DEBUG adds a row above).
func menuBedShiftYFor(rowY float64) float64 {
	return math.Max(0, rowY+30-menuBedTop)
}
