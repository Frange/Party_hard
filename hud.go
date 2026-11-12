package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// THE PLAY HUD: one compact panel in the top left corner (it replaces the loose lines POINTS / KILLS /
// GENTE / GUARDIAS / forces that used to float over the map).
//
//	+------------------------------------------------------------+
//	| [icon] 23  GENTE                                    PUNTOS |   big number = people left (pops when it drops,
//	|            GUARDIAS 2                               01240  |   pulses gold when only the last guests remain)
//	| [=========progress bar (kills so far)=======-------------] |   2P: the filled part is split blue / red by who killed
//	| BAJAS 12/35  7 5   (figures at scale 2)                    |
//	| [clock] 1:23                        RETO x3  CADENA x4     |   chips only when they apply (never in STEALTH)
//	| POLICIA:2 FBI:1 ASESINO:1                                  |   forces on the scene (only when there are any)
//	+------------------------------------------------------------+
//
// Stars of the police stay top centre (drawStars); the help "?" and the trophy counter stay top right; the stealth
// phase panel stays on the right (650,34). Everything under the panel (item, stamina) starts at hudState.bottom.
// Points count up (animated), the bar eases; both are updated in updatePlay, so a pause freezes them.
// ---------------------------------------------------------------------------

const (
	hudX, hudY = 16.0, 16.0 // 16 px from the screen edges (as the trophy counter)
	hudW       = 248.0
	hudPad     = 8.0
	hudBarY    = 46.0 // relative to hudY
	hudRowBY   = 48.0 // BAJAS row (text box top of the scale-2 figures; the box is 32 high, the glyphs sit in its middle)
	hudRowTY   = 75.0 // TIME row (clock + M:SS)
	hudRowCY   = 86.0 // forces rows
	hudBaseH   = 91.0 // height without forces
)

type hudState struct {
	pts    float64 // points shown (counts up to g.points)
	bar    float64 // progress shown, 0..1
	people int     // last people count (to pop the number when it drops)
	pop    float64 // seconds left of the pop flash
	bottom float64 // y of the first free line under the panel (set when drawn)
	hit    bool    // a character was drawn over the panel in the last frame (hudhide.go)
	hold   float64 // seconds left before the panel may come back
	hide   float64 // 0 = fully shown, 1 = hidden (zero value: shown)
	armed  bool    // a player is being drawn: hudTouch counts (hudhide.go)
}

func (h *hudState) reset(g *Game) {
	*h = hudState{pts: float64(g.points), bar: g.hudProgress(), people: g.hudCounts().people, bottom: hudY + hudBaseH + 4}
}

// hudCounts: everything the panel shows, computed in one place (the draw and the tests use it).
type hudCounts struct {
	people, guards, killers int
	forces                  []string // "POLICIA:2", "FBI:1"... (police tiers present) + killers
}

func (g *Game) hudCounts() hudCounts {
	var c hudCounts
	for _, gu := range g.guests {
		if !gu.out() {
			c.people++
		}
	}
	for _, gd := range g.guards {
		if !gd.dead {
			c.guards++
		}
	}
	byKind := map[int]int{}
	for _, cp := range g.cops {
		if !cp.medic && cp.state != cLeave {
			byKind[cp.kind]++
		}
	}
	for _, s := range g.allSlashers() {
		if s.state != slGone {
			c.killers++
		}
	}
	for _, k := range []int{unitPolice, unitFBI, unitMilitary, unitSWAT, unitRangers} {
		if n := byKind[k]; n > 0 {
			c.forces = append(c.forces, fmt.Sprintf("%s:%d", starLabel(k), n))
		}
	}
	if c.killers > 0 {
		c.forces = append(c.forces, fmt.Sprintf("ASESINO:%d", c.killers))
	}
	return c
}

// hudProgress: how much of the party is done (0..1), by the people who are already out.
func (g *Game) hudProgress() float64 {
	if g.total <= 0 {
		return 0
	}
	return clampF(float64(g.total-g.aliveCount())/float64(g.total), 0, 1)
}

// updateHUD runs the animations of the panel (called from updatePlay).
func (g *Game) updateHUD(dt float64) {
	h := &g.hud
	if d := float64(g.points) - h.pts; d < 0 || d > 1e7 {
		h.pts = float64(g.points)
	} else if d > 0 {
		h.pts = math.Min(float64(g.points), h.pts+math.Max(1, d*math.Min(1, dt*12)))
	}
	h.bar += (g.hudProgress() - h.bar) * math.Min(1, dt*6)
	if p := g.hudCounts().people; p != h.people {
		if p < h.people {
			h.pop = 0.35
		}
		h.people = p
	}
	if h.pop > 0 {
		h.pop -= dt
	}
	h.stepHide(dt)
}

// hudLerp mixes two colours.
func hudLerp(a, b color.RGBA, t float64) color.RGBA {
	t = clampF(t, 0, 1)
	return color.RGBA{uint8(lerp(float64(a.R), float64(b.R), t)), uint8(lerp(float64(a.G), float64(b.G), t)), uint8(lerp(float64(a.B), float64(b.B), t)), 255}
}

// drawPersonIcon: the little guest of the GENTE counter (9x14).
func drawPersonIcon(dst *ebiten.Image, x, y float64, c color.RGBA) {
	dark := rgb(0x1a0c10)
	fillRect(dst, x+1, y-1, 7, 7, dark)
	fillRect(dst, x-1, y+5, 11, 10, dark)
	fillRect(dst, x+2, y, 5, 5, c)               // head
	fillRect(dst, x, y+6, 9, 5, c)               // shoulders and chest
	fillRect(dst, x+1, y+11, 3, 3, c)            // legs
	fillRect(dst, x+5, y+11, 3, 3, c)            //
	fillRect(dst, x+3, y+2, 1, 1, dark)          // eyes
	fillRect(dst, x+5, y+2, 1, 1, dark)          //
	fillRect(dst, x+4, y+6, 1, 3, shade(c, 0.7)) // jacket line
}

// drawHUDPanel paints the panel and returns nothing; the y under it is left in g.hud.bottom.
func (g *Game) drawHUDPanel(dst *ebiten.Image) {
	cnt := g.hudCounts()
	lines := g.hudForceLines(cnt.forces)
	g.hud.bottom = hudY + hudPanelH(len(lines)) + 3 // the column under the panel keeps its place while the panel is hidden
	g.withHudFade(dst, func(d *ebiten.Image) { g.paintHUDPanel(d, cnt, lines) })
}

// hudPanelH: the height of the panel with n lines of forces.
func hudPanelH(n int) float64 {
	if n > 0 {
		return hudBaseH + 11*float64(n)
	}
	return hudBaseH
}

func (g *Game) paintHUDPanel(dst *ebiten.Image, cnt hudCounts, lines []string) {
	h := &g.hud
	ph := hudPanelH(len(lines))
	px, py := hudX, hudY
	fillRect(dst, px, py, hudW, ph, rgba(0x08060f, 175))
	edge := rgba(0x6a5a9a, 190)
	fillRect(dst, px, py, hudW, 1, edge)
	fillRect(dst, px, py+ph-1, hudW, 1, edge)
	fillRect(dst, px, py, 1, ph, edge)
	fillRect(dst, px+hudW-1, py, 1, ph, edge)

	// ---- people left: icon + big number + labels
	people := cnt.people
	numCol := rgb(0xff8070)
	if h.pop > 0 {
		numCol = hudLerp(rgb(0xff8070), rgb(0xffffff), h.pop/0.35)
	} else if a := g.aliveCount(); a >= 1 && a <= lastGuestsMax && g.mode == ModePlay {
		numCol = hudLerp(rgb(0xff8070), rgb(0xffd860), 0.5+0.5*math.Sin(g.time*4)) // the last guests: the number glows gold
	}
	drawPersonIcon(dst, px+hudPad, py+16, numCol)
	g.drawText(dst, itoa(people), px+hudPad+17, py+2, numCol, 3)
	lx := px + hudPad + 17 + 3*18 + 8 // the label column does not move with the digits
	g.drawText(dst, "GENTE", lx, py+7, rgb(0xd8d0f0), 1)
	g.drawText(dst, fmt.Sprintf("GUARDIAS %d", cnt.guards), lx, py+19, rgb(0xff9a60), 1)

	// ---- points (counts up), right aligned
	pts := g.points
	if g.mode == ModePlay {
		pts = int(math.Round(h.pts))
	}
	ps := fmt.Sprintf("%04d", pts)
	right := px + hudW - hudPad
	g.drawText(dst, "PUNTOS", right-g.textW("PUNTOS", 1), py+7, rgb(0xd8d0f0), 1)
	g.drawText(dst, ps, right-g.textW(ps, 2), py+17, rgb(0xb88cff), 2)

	// ---- progress bar (the thin line): how much of the party is done
	bx, by, bw := px+hudPad, py+hudBarY, hudW-2*hudPad
	fillRect(dst, bx-1, by-1, bw+2, 5, rgb(0x14102a))
	fillRect(dst, bx, by, bw, 3, rgb(0x2a2440))
	fw := math.Round(bw * clampF(h.bar, 0, 1))
	if g.is2P() {
		k0, k1 := g.run.killsOf(0), g.run.killsOf(1)
		w0 := 0.0
		if k0+k1 > 0 {
			w0 = math.Round(fw * float64(k0) / float64(k0+k1))
		}
		c0, c1 := ringColor(g.players()[0]), ringColor(g.players()[1])
		fillRect(dst, bx, by, w0, 3, color.RGBA{c0[0], c0[1], c0[2], 255})
		fillRect(dst, bx+w0, by, fw-w0, 3, color.RGBA{c1[0], c1[1], c1[2], 255})
		if k0+k1 == 0 {
			fillRect(dst, bx, by, fw, 3, rgb(0x9a90c0))
		}
	} else {
		fillRect(dst, bx, by, fw, 3, hudLerp(rgb(0xff6aa8), rgb(0x60ff80), h.bar))
	}
	if fw >= 2 { // a lighter top line gives the fill a little relief
		fillRect(dst, bx, by, fw, 1, rgba(0xffffff, 70))
	}

	// ---- kills
	// The figures are scale 2 (they were 1: too small and tight); the word BAJAS and the chips stay at 1, centred
	// on the same line.
	ry := py + hudRowBY
	ry1 := ry + 8
	kw := "BAJAS"
	g.drawText(dst, kw, px+hudPad, ry1, rgb(0xff6aa8), 1)
	kx := px + hudPad + g.textW(kw+" ", 1)
	ks := fmt.Sprintf("%d/%d", g.total-g.aliveCount(), g.total)
	g.drawText(dst, ks, kx, ry, rgb(0xff6aa8), 2)
	if g.is2P() {
		x := kx + g.textW(ks, 2) + 10
		for i, pl := range g.players() {
			c := ringColor(pl)
			s := itoa(g.run.killsOf(i))
			g.drawText(dst, s, x, ry, color.RGBA{c[0], c[1], c[2], 255}, 2)
			x += g.textW(s, 2) + 8
		}
	}

	// ---- the time of the level (the one the speed bonus counts)
	g.drawHUDClock(dst, px+hudPad, py+hudRowTY)
	g.drawHUDStatus(dst, g.hudIconsX(), py+hudRowTY+hudIconDY) // status icons (hudicons.go): right of the clock, only the active ones
	g.drawChips(dst, right, py+hudRowTY)                       // RETO / CADENA moved here: the bigger BAJAS figures need the whole row above

	// ---- forces on the scene
	for i, l := range lines {
		g.drawText(dst, l, px+hudPad, py+hudRowCY+11*float64(i), rgb(0x90c8ff), 1)
	}
}

// hudForceLines packs the forces ("POLICIA:2") into at most two lines of the panel width.
func (g *Game) hudForceLines(forces []string) []string {
	var lines []string
	cur := ""
	for _, f := range forces {
		try := f
		if cur != "" {
			try = cur + " " + f
		}
		if cur != "" && g.textW(try, 1) > hudW-2*hudPad {
			lines = append(lines, cur)
			cur = f
			continue
		}
		cur = try
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) > 2 {
		lines = lines[:2]
	}
	return lines
}

// drawChips: "RETO x3" and "CADENA xN", right aligned on the TIME row (levelscore.go gives the values).
func (g *Game) drawChips(dst *ebiten.Image, right, y float64) {
	if g.stealth || g.mode != ModePlay {
		return
	}
	if n := maxI(g.chainNow(0), g.chainNow(1)); n >= 2 {
		s := fmt.Sprintf("CADENA x%d", n)
		g.drawText(dst, s, right-g.textW(s, 1), y, rgb(0x60e0ff), 1)
		right -= g.textW(s, 1) + 10
	}
	if g.challenge {
		s := "RETO x3"
		if left, ok := g.dailyTimeLeft(); ok { // RETO DE HOY "Contrarreloj": the time left, red in the last minute
			s = "RETO " + hudClockText(left) // compact: the x3 is in the party list and the score anyway
		}
		w := g.textW(s, 1)
		col := rgb(0xff9030)
		if left, ok := g.dailyTimeLeft(); ok && left < 60 && int(g.time*2)%2 == 0 {
			col = rgb(0xff4a3a)
		}
		g.drawText(dst, s, right-w, y, col, 1)
		drawChallengeFlag(dst, right-w-9, y, col)
	}
}

// drawChallengeFlag: the 6x7 flag icon of the RETO chip (whole pixels, dark edge).
func drawChallengeFlag(dst *ebiten.Image, x, y float64, c color.RGBA) {
	dark := rgb(0x1a0c10)
	fillRect(dst, x, y, 1, 7, rgb(0xd8d0f0))
	fillRect(dst, x+1, y, 5, 1, dark)
	fillRect(dst, x+1, y+1, 5, 3, c)
	fillRect(dst, x+1, y+4, 4, 1, dark)
}

// hudClockColor: the colour of the level time. Green while the full speed bonus is within reach (<3:00), amber while
// there is still some (<5:00), grey after. For 1.5 s after crossing a threshold it flashes to white in hard steps
// (no blur, no fade).
func hudClockColor(t float64) color.RGBA {
	green, amber, grey := rgb(0x60e090), rgb(0xffc040), rgb(0x9a94b0)
	var c color.RGBA
	var since float64
	switch {
	case t < timeFastS:
		c, since = green, -1
	case t < timeMidS:
		c, since = amber, t-timeFastS
	default:
		c, since = grey, t-timeMidS
	}
	if since >= 0 && since < 1.5 && int(since/0.25)%2 == 0 {
		return rgb(0xffffff)
	}
	return c
}

// hudClockText: M:SS (minutes are not capped: 12:05, 104:09).
func hudClockText(t float64) string {
	if t < 0 {
		t = 0
	}
	return fmt.Sprintf("%d:%02d", int(t)/60, int(t)%60)
}

// drawHUDClock: a 7x7 clock icon and the time, on the TIME row of the panel.
func (g *Game) drawHUDClock(dst *ebiten.Image, x, y float64) {
	c := hudClockColor(g.time)
	dark := rgb(0x1a0c10)
	fillRect(dst, x, y, 7, 7, dark)
	fillRect(dst, x+1, y+1, 5, 5, c)
	fillRect(dst, x+2, y+2, 3, 3, dark)
	fillRect(dst, x+3, y+2, 1, 2, c) // minute hand
	fillRect(dst, x+3, y+3, 2, 1, c) // hour hand
	g.drawText(dst, hudClockText(g.time), x+11, y, c, 1)
}

// hudSideGap: the free space between the HUD panel and the first note under it.
const hudSideGap = 12.0

// hudSide: the y of each note stacked in the column under the HUD panel (OBJETO, stamina bar(s), ambulance, jail),
// one under the other and never touching: a block only takes room when it is on screen. x is always hudX.
func (g *Game) hudSide() (obj, stam, rev, jail float64) {
	y := g.hud.bottom - 3 + hudSideGap + g.cdLineH() // the countdown line goes first (countdown.go)
	lines := 1
	if g.is2P() {
		lines = 2
	}
	obj = y
	for _, pl := range g.players() {
		if pl.item != "" && g.mode == ModePlay {
			y += 12*float64(lines) + 2
			break
		}
	}
	stam = y
	for _, pl := range g.players() {
		if g.mode == ModePlay && pl.stamina < 5.9 {
			y += 10 + float64(lines-1)*6 // the bar(s) are 4 high, 6 apart
			y += 6
			break
		}
	}
	rev = y
	if g.is2P() {
		y += g.reviveHudH() // one line, or the box "Tu compañero ha muerto..." (revive.go)
	}
	jail = y
	return
}
