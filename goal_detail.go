package main

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// OBJETIVOS / detail window. Pressing a card (click, Enter / Interactuar, A on the pad) or a trophy of the list
// opens a window over the panel with that element: the character walking in place at a big whole scale (his own
// gait), the trophy as an animated cup, the name, the description the card already shows and its state. The arrows
// at the sides (or left / right) go to the previous / next element of the SAME page, cyclically, in the order of the
// page; closing (X, a click outside, Esc / B / the ATRÁS button) leaves the page selected on the element seen.
// A locked element shows exactly what its card shows (silhouette and ???, never its real name) and does not move.
//
// The window covers exactly the OBJETIVOS panel (230,56 700x440) with an opaque fill: the X sits where the in-game
// window has its own, a click outside is the same rule the in-game window uses, and nothing of the grid shows
// through (the text audit sees no half-covered text). While it is open the list under it gets no input at all (no
// wheel, no hover scroll, no clicks). The state lives here (goalDet), not in Game, and nothing is written in Progress.
// The animation clock only runs while it is open.
// ---------------------------------------------------------------------------

type goalDetailState struct {
	open bool
	t    float64 // seconds since the element in view was shown (drives its animation)
}

var goalDet goalDetailState

// the window: the same rectangle as the OBJETIVOS panel (drawTrophyPanel)
const (
	gdX, gdY, gdW, gdH = 230.0, 56.0, 700.0, 440.0
	gdCloseX, gdCloseY = 902.0, 62.0 // the X, where the in-game window (trophyhud.go) has its own
	gdCloseS           = 20.0
	gdArrowW, gdArrowH = 36.0, 72.0
	gdArrowY           = gdY + gdH/2 - gdArrowH/2
	gdPrevX            = gdX + 12
	gdNextX            = gdX + gdW - 12 - gdArrowW
	gdPicX, gdPicY     = 296.0, 102.0 // the portrait box
	gdPicW, gdPicH     = 240.0, 330.0
	gdTextX            = 556.0                  // the text column
	gdTextW            = gdNextX - 16 - gdTextX // up to the right arrow
	gdTextTop          = 102.0
	gdTextBottom       = gdY + gdH - 14 // nothing below this line
)

// what a point of the screen is in the window
const (
	gdHitInside = iota
	gdHitOutside
	gdHitClose
	gdHitPrev
	gdHitNext
)

func gdIn(mx, my, x, y, w, h float64) bool { return mx >= x && mx <= x+w && my >= y && my <= y+h }

// goalDetailHit: what the pointer is on.
func goalDetailHit(mx, my float64) int {
	switch {
	case !gdIn(mx, my, gdX, gdY, gdW, gdH):
		return gdHitOutside
	case gdIn(mx, my, gdCloseX, gdCloseY, gdCloseS, gdCloseS):
		return gdHitClose
	case gdIn(mx, my, gdPrevX, gdArrowY, gdArrowW, gdArrowH):
		return gdHitPrev
	case gdIn(mx, my, gdNextX, gdArrowY, gdArrowW, gdArrowH):
		return gdHitNext
	}
	return gdHitInside
}

// goalDetailStep: the index d places away from i in a page of n elements, going round at both ends.
func goalDetailStep(i, n, d int) int {
	if n <= 0 {
		return 0
	}
	return ((i+d)%n + n) % n
}

// goalDetailCount: the elements of the page in view (0 on CARRERA: no window there).
func (g *Game) goalDetailCount() int {
	switch {
	case g.trophyTab == tabTrophies:
		return len(trophyList)
	case g.trophyTab == tabCareer:
		return 0
	}
	return len(g.albumView())
}

func (g *Game) goalDetailSel() int {
	if g.trophyTab == tabTrophies {
		return g.trophySel
	}
	return g.albumSel
}

// goalDetailSet selects element i of the page and scrolls the list under the window to it.
func (g *Game) goalDetailSet(i int) {
	if g.trophyTab == tabTrophies {
		g.trophySel = clampI(i, 0, len(trophyList)-1)
		if g.trophySel < g.trophyScroll {
			g.trophyScroll = g.trophySel
		}
		if g.trophySel >= g.trophyScroll+trophyRows {
			g.trophyScroll = g.trophySel - trophyRows + 1
		}
		return
	}
	g.albumSel = i
	g.albumClamp()
}

// openGoalDetail opens the window on the selected element (nothing on CARRERA or an empty page).
func (g *Game) openGoalDetail() {
	if g.goalDetailCount() == 0 {
		return
	}
	g.goalDetailSet(g.goalDetailSel())
	goalDet = goalDetailState{open: true}
}

// closeGoalDetail: back to the page, selected on the element seen. The pointer where it is now does not count as a
// move (it would select the card under it) and the hover scroll waits for the mouse to move again.
func (g *Game) closeGoalDetail() {
	goalDet = goalDetailState{}
	g.lastMX, g.lastMY = ebiten.CursorPosition()
	for _, id := range []string{"goals", "trophies"} {
		if e := edgeStates[id]; e != nil {
			e.mouse, e.acc = false, 0
		}
	}
}

// goalDetailMove: the previous (-1) or next (+1) element of the page.
func (g *Game) goalDetailMove(d int) {
	g.goalDetailSet(goalDetailStep(g.goalDetailSel(), g.goalDetailCount(), d))
	goalDet.t = 0
}

// goalDetailAsk: the input that opens the window on the selected element of a page (the click is the caller's).
func goalDetailAsk() bool { return actConfirm.just() || menuInteractJust() } // ○ goes back on a pad (input.go)

// updateGoalDetail runs the window; it returns true while it is open (the page under it must not read input
// this frame, also on the frame it closes, so that the same Esc does not leave the screen too).
func (g *Game) updateGoalDetail() bool {
	if !goalDet.open {
		return false
	}
	if (g.mode != ModeTrophies && !g.trophyOv) || g.goalDetailCount() == 0 {
		g.closeGoalDetail()
		return false
	}
	goalDet.t += 1.0 / 60
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		switch goalDetailHit(float64(mx), float64(my)) {
		case gdHitOutside, gdHitClose:
			g.closeGoalDetail()
			return true
		case gdHitPrev:
			g.goalDetailMove(-1)
		case gdHitNext:
			g.goalDetailMove(1)
		}
	}
	switch {
	case actBack.just() || escJust():
		g.closeGoalDetail()
	case actLeft.just():
		g.goalDetailMove(-1)
	case actRight.just():
		g.goalDetailMove(1)
	}
	return true
}

// ---- what the window says ------------------------------------------------------------------------------------

type gdPara struct {
	s   string // already translated
	col color.RGBA
}

// gdContent: the texts of the element in view (translated) and how to draw its picture.
type gdContent struct {
	head    string // the page and the position: "PERSONAJES  12 / 282"
	name    string
	sub     string
	seal    string
	sealCol color.RGBA
	paras   []gdPara
	trophy  bool
	tr      Trophy
	card    albumCard
	ok      bool // won: the picture moves
	seen    bool // a character seen / killed but not won: his face, still
}

// albumSeal: the stamp of a card (VISTO, ELIMINADO n/5, CONSEGUIDO, AMIGO, TROFEO), "" on a card never met.
func (g *Game) albumSeal(c albumCard) (string, color.RGBA) {
	ok, seen, bits := g.albumUnlocked(c), g.albumSeenOnly(c), g.albumBits(c)
	if !ok && !seen {
		return "", color.RGBA{}
	}
	seal, scol := "VISTO", rgb(0x8ac0f0)
	switch {
	case bits&albumHelped != 0 && c.vip:
		seal, scol = "AMIGO", rgb(0x70e090)
	case ok && c.kind() == cardChar && c.trophyID == "":
		seal, scol = "CONSEGUIDO", rgb(0x70e090) // killed charKillsNeed times (or bought before the redesign)
	case !ok && bits&albumElim != 0: // killed, not enough times yet
		seal, scol = fmt.Sprintf("ELIMINADO %d/%d", minI(g.progress.CharKills[c.key], charKillsNeed-1), charKillsNeed), rgb(0xff6a6a)
	case ok && bits&albumKilled != 0:
		seal, scol = "ELIMINADO", rgb(0xff6a6a)
	case ok && c.trophyID != "":
		seal, scol = "TROFEO", rgb(0xe8c040)
	case ok:
		seal, scol = "CONSEGUIDO", rgb(0x70e090) // bought in the shop before the redesign
	}
	return seal, scol
}

// charStatus: what is left to win a character (Spanish already translated), the end of the line under the grid.
func (g *Game) charStatus(c albumCard) string {
	if !g.albumUnlocked(c) && g.albumBits(c)&(albumSeen|albumElim) == 0 {
		if c.trophyID != "" {
			return tr(c.lockedHint())
		}
		return tr("Aún no lo has visto")
	}
	switch n := g.charKillsLeft(c); {
	case n == 0:
		return tr("Conseguido: elígelo en el vestuario")
	case c.trophyID != "":
		return tr(c.lockedHint())
	case n == 1:
		return tr("Falta 1 muerte para conseguirlo")
	default:
		return fmt.Sprintf(tr("Faltan %d muertes para conseguirlo"), n)
	}
}

// goalDetailContent: what the window shows for the selected element of the page.
func (g *Game) goalDetailContent() gdContent {
	n := g.goalDetailCount()
	i := clampI(g.goalDetailSel(), 0, maxI(n-1, 0))
	out := gdContent{head: tr(goalTabs[clampI(g.trophyTab, 0, tabCount-1)].name) + fmt.Sprintf("  %d / %d", i+1, n)}
	if g.trophyTab == tabTrophies {
		view := g.trophyView()
		t := view[clampI(i, 0, len(view)-1)]
		out.trophy, out.tr, out.ok = true, t, g.trophyDone(t)
		if g.secretHidden(t) {
			out.name = "???"
			out.paras = []gdPara{{tr("Trofeo secreto: se descubre jugando."), rgb(0xb8b0d0)}, {tr("Premio: ") + "???", rgb(0xf0e060)}}
			return out
		}
		out.name = tr(t.name)
		switch {
		case out.ok:
			out.seal, out.sealCol = "CONSEGUIDO", rgb(0x70e090)
		case t.need > 1 && !secretTrophy[t.id]:
			out.seal, out.sealCol = fmt.Sprintf("%d/%d", minI(progressStat(&g.progress, t.stat), t.need), t.need), rgb(0x8a86a0)
		}
		status, scol := tr("Premio (aún bloqueado): "), rgb(0xf0e060)
		if out.ok {
			status, scol = tr("Premio desbloqueado: "), rgb(0x70e090)
		}
		out.paras = []gdPara{{tr("Cómo conseguirlo: ") + tr(howHint(t.how)), rgb(0xb8b0d0)}, {status + tr(t.reward.name) + ". " + tr(t.reward.desc), scol}}
		return out
	}
	view := g.albumView()
	c := view[clampI(i, 0, len(view)-1)]
	out.card, out.ok, out.seen = c, g.albumUnlocked(c), g.albumSeenOnly(c)
	out.seal, out.sealCol = g.albumSeal(c)
	if !out.ok && !out.seen { // what the card shows: nothing of who he is
		out.name, out.sub = "???", "???"
		hint := tr(c.lockedHint())
		if c.kind() == cardChar {
			hint = g.charStatus(c)
		}
		out.paras = []gdPara{{hint, rgb(0xb8b0d0)}}
		return out
	}
	out.name = tr(c.nameES)
	out.sub = c.roleES
	if c.roleCard != "" {
		out.sub = c.roleCard
	}
	out.sub = tr(out.sub)
	if m, ok := bandRealNames[c.slug]; ok && c.kind() == cardChar && c.roleCard != "" { // a musician: band and what he plays
		out.sub += " · " + tr(m.part)
	}
	if footballTeam(c.slug) != "" && c.kind() == cardChar && c.roleCard != "" { // a footballer: team and trade
		out.sub += " · " + tr("Futbolista")
	}
	switch {
	case c.kind() == cardChar:
		if d := charDesc(c.slug); d != "" {
			out.paras = append(out.paras, gdPara{tr(d), rgb(0xe0dcf0)})
		}
		out.paras = append(out.paras, gdPara{g.charStatus(c), rgb(0xf0e060)})
	case out.seen:
		out.paras = append(out.paras, gdPara{tr("Visto: mátalo para conseguirlo"), rgb(0xf0e060)})
	default:
		if d := goalRewardDesc(c); d != "" { // the killers and police forces that a trophy brings have their own line
			out.paras = append(out.paras, gdPara{tr(d), rgb(0xe0dcf0)})
		}
	}
	return out
}

// goalRewardDesc: the description of the trophy reward that is this killer or police force ("" if none).
func goalRewardDesc(c albumCard) string {
	for _, t := range trophyList {
		switch {
		case c.killer && t.reward.tab == shopKillers && t.reward.arg == c.slug:
			return t.reward.desc
		case c.cop && c.roleKind > unitPolice && t.reward.id == fmt.Sprintf("police:%d", c.roleKind):
			return t.reward.desc
		}
	}
	return ""
}

// gdText: one text of the window, placed.
type gdText struct {
	s     string
	x, y  float64
	scale float64
	col   color.RGBA
}

// gdLineH: the height of a line of text at a whole scale, with its gap.
func (g *Game) gdLineH(scale float64) float64 {
	_, h := textMeasure(g, "ÁgÑ")
	return math.Ceil(h*scale) + 3
}

// goalDetailTexts lays the window's texts out: the head, then the name (big, broken into up to two lines, or small
// when it is longer), the role, the stamp and the paragraphs, all in the text column.
func (g *Game) goalDetailTexts(c gdContent) []gdText {
	out := []gdText{{c.head, gdX + 16, gdY + 10, 1, rgb(0xff6a8a)}}
	y := gdTextTop
	scale := 2.0
	lines := g.wrapText(c.name, scale, gdTextW)
	if len(lines) > 2 {
		scale = 1
		lines = g.wrapText(c.name, scale, gdTextW)
	}
	ncol := rgb(0xf4f0ff)
	if !c.ok && !c.seen {
		ncol = rgb(0x8a86a0)
	}
	for _, l := range lines {
		out = append(out, gdText{l, gdTextX, y, scale, ncol})
		y += g.gdLineH(scale)
	}
	y += 4
	if c.sub != "" {
		for _, l := range g.wrapText(c.sub, 1, gdTextW) {
			out = append(out, gdText{l, gdTextX, y, 1, rgb(0x9a96b0)})
			y += g.gdLineH(1)
		}
	}
	if c.seal != "" {
		out = append(out, gdText{tr(c.seal), gdTextX, y, 1, c.sealCol})
		y += g.gdLineH(1)
	}
	for _, p := range c.paras {
		y += 10
		for _, l := range g.wrapText(p.s, 1, gdTextW) {
			out = append(out, gdText{l, gdTextX, y, 1, p.col})
			y += g.gdLineH(1)
		}
	}
	for i := range out { // whole pixels
		out[i].x, out[i].y = math.Round(out[i].x), math.Round(out[i].y)
	}
	return out
}

// ---- drawing -----------------------------------------------------------------------------------------------------

// drawGoalDetail paints the window over the OBJETIVOS panel (screen and in-game window) when it is open.
func (g *Game) drawGoalDetail(dst *ebiten.Image) {
	if !goalDet.open || (g.mode != ModeTrophies && !g.trophyOv) || g.goalDetailCount() == 0 {
		return
	}
	c := g.goalDetailContent()
	g.recPanel(gdX, gdY, gdW, gdH)
	fillRect(dst, gdX-2, gdY-2, gdW+4, gdH+4, rgb(0x6a4ab0))
	fillRect(dst, gdX, gdY, gdW, gdH, rgb(0x0a0814)) // opaque: the grid does not show through
	fillRect(dst, gdX, gdY+32, gdW, 1, rgba(0x6a4ab0, 160))
	mx, my := ebiten.CursorPosition()
	hot := goalDetailHit(float64(mx), float64(my))
	// the X
	xbg := rgb(0x6a1a2a)
	if hot == gdHitClose {
		xbg = rgb(0x9a2a3a)
	}
	fillRect(dst, gdCloseX, gdCloseY, gdCloseS, gdCloseS, xbg)
	g.drawTextC(dst, "X", gdCloseX+gdCloseS/2, gdCloseY+4, rgb(0xffd0d8), 1)
	// the arrows
	for i, ax := range []float64{gdPrevX, gdNextX} {
		bg := rgb(0x1e1636)
		if (i == 0 && hot == gdHitPrev) || (i == 1 && hot == gdHitNext) {
			bg = rgb(0x3a2a6a)
		}
		fillRect(dst, ax, gdArrowY, gdArrowW, gdArrowH, bg)
		fillRect(dst, ax, gdArrowY, gdArrowW, 1, rgb(0x6a4ab0))
		fillRect(dst, ax, gdArrowY+gdArrowH-1, gdArrowW, 1, rgb(0x6a4ab0))
		gdTriangle(dst, ax+gdArrowW/2, gdArrowY+gdArrowH/2, 10, i == 0, rgb(0xf4f0ff))
	}
	// the picture
	if c.trophy {
		g.drawGoalDetailCup(dst, c)
	} else {
		g.drawGoalDetailChar(dst, c)
	}
	for _, t := range g.goalDetailTexts(c) {
		g.drawText(dst, t.s, t.x, t.y, t.col, t.scale)
	}
}

// gdTriangle: a solid arrow head of whole pixels pointing left (or right) with its middle at cx, cy.
func gdTriangle(dst *ebiten.Image, cx, cy float64, size int, left bool, col color.RGBA) {
	cx, cy = math.Round(cx), math.Round(cy)
	for k := 0; k < size; k++ {
		x := cx - float64(size)/2 + float64(k)
		if !left {
			x = cx + float64(size)/2 - float64(k) - 1
		}
		fillRect(dst, math.Round(x), cy-float64(k), 1, float64(2*k+1), col)
	}
}

// drawGoalDetailChar: the portrait box with the character. Won: he walks in place facing us (his own gait) and
// stands for a moment now and then. Seen only: his face, still. Never met: the dark silhouette of his card.
func (g *Game) drawGoalDetailChar(dst *ebiten.Image, c gdContent) {
	card := c.card
	for r := 0.0; r < gdPicH; r += 2 {
		f := r / gdPicH
		col := color.RGBA{uint8(24 + 10*f), uint8(22 + 8*f), uint8(34 + 12*f), 255}
		if c.ok {
			switch {
			case card.vip:
				col = color.RGBA{uint8(70 + 90*f), uint8(40 + 60*f), uint8(90 - 40*f), 255}
			case card.cop:
				col = color.RGBA{uint8(30 + 20*f), uint8(50 + 50*f), uint8(100 + 70*f), 255}
			case card.killer:
				col = color.RGBA{uint8(70 + 60*f), uint8(26 + 20*f), uint8(34 + 10*f), 255}
			default:
				col = color.RGBA{uint8(40 + 30*f), uint8(60 + 70*f), uint8(110 + 40*f), 255}
			}
		}
		fillRect(dst, gdPicX, gdPicY+r, gdPicW, 2, col)
	}
	cs := g.albumCS(card)
	base, _ := cs.img(AnimIdle, DirDown, 0)
	if base == nil {
		return
	}
	bw, bh := float64(base.Bounds().Dx()), float64(base.Bounds().Dy())
	s := math.Floor(math.Min((gdPicH-60)/bh, (gdPicW-30)/bw))
	s = math.Max(1, math.Min(s, 8))
	feet := gdPicY + gdPicH - 26
	filledEllipse(dst, gdPicX+gdPicW/2, feet, math.Round(bw*s*0.4), math.Round(s*3), rgba(0x000000, 90))
	anim, ph := AnimIdle, 0.0
	if c.ok {
		anim, ph = goalDetailPose(card.slug, goalDet.t)
	}
	im, flip := cs.img(anim, DirDown, ph)
	if im == nil {
		im, flip = base, false
	}
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(w, 0)
	}
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(math.Round(gdPicX+gdPicW/2-w*s/2), math.Round(feet+2-h*s))
	if !c.ok && !c.seen {
		op.ColorScale.Scale(0.02, 0.018, 0.035, 1) // dark silhouette (darker than the card's: at this size it must not show who he is)
	}
	dst.DrawImage(im, op)
}

// goalDetailPose: the animation of a won character t seconds after he is shown: 4 s walking (his gait, if he has
// one), 1.5 s standing.
func goalDetailPose(slug string, t float64) (int, float64) {
	const walkS, cycle = 4.0, 5.5
	k := math.Mod(t, cycle)
	if k < walkS {
		return gaitAnim(gaitSlugs[slug], AnimWalk, k*9)
	}
	return AnimIdle, (k - walkS) * 1.3
}

// drawGoalDetailCup: the trophy as a big cup. Won: gold, it floats and sparkles. Not won: grey and still.
func (g *Game) drawGoalDetailCup(dst *ebiten.Image, c gdContent) {
	fillRect(dst, gdPicX, gdPicY, gdPicW, gdPicH, rgb(0x120c24))
	cx, cy := gdPicX+gdPicW/2, gdPicY+130.0
	cup, dark := rgb(0x5a5668), rgb(0x3a3648)
	if c.ok {
		cup, dark = rgb(0xe8c040), rgb(0xa07a18)
		cy += math.Round(math.Sin(goalDet.t*2.5) * 6)
	}
	filledEllipse(dst, cx, gdPicY+gdPicH-40, 60, 8, rgba(0x000000, 110))
	filledEllipse(dst, cx, cy, 52, 44, cup)
	fillRect(dst, cx-52, cy-48, 104, 48, cup)
	fillRect(dst, cx-10, cy+36, 20, 32, dark)
	fillRect(dst, cx-38, cy+68, 76, 16, cup)
	fillRect(dst, cx-70, cy-36, 16, 28, cup)
	fillRect(dst, cx+54, cy-36, 16, 28, cup)
	if !c.ok {
		return
	}
	// a shine sliding over the bowl and four twinkles around it
	sx := cx - 44 + math.Round(math.Mod(goalDet.t*40, 88))
	fillRect(dst, sx, cy-44, 6, 40, rgba(0xffffff, 120))
	for i, p := range [][2]float64{{-84, -70}, {80, -56}, {-76, 40}, {88, 30}} {
		on := int(goalDet.t*4+float64(i)*1.7)%4 < 2
		if !on {
			continue
		}
		x, y := cx+p[0], cy+p[1]
		fillRect(dst, x-1, y-6, 2, 12, rgb(0xfff4c0))
		fillRect(dst, x-6, y-1, 12, 2, rgb(0xfff4c0))
	}
}

// ---- debug ---------------------------------------------------------------------------------------------------

// goalDetailDebug opens the window from the command line (-goaldetail tab:index; a negative index counts from the
// end of the page: 1:-1 is the last character, the won ones are at the end). Debug only.
func (g *Game) goalDetailDebug(spec string) {
	parts := strings.SplitN(spec, ":", 2)
	tab, _ := strconv.Atoi(parts[0])
	idx := 0
	if len(parts) == 2 {
		idx, _ = strconv.Atoi(parts[1])
	}
	g.mode = ModeTrophies
	g.trophyTab = clampI(tab, 0, tabCount-1)
	n := g.goalDetailCount()
	if idx < 0 {
		idx += n
	}
	if g.trophyTab == tabTrophies {
		g.trophySel = clampI(idx, 0, maxI(n-1, 0))
	} else {
		g.albumSel = clampI(idx, 0, maxI(n-1, 0))
	}
	g.openGoalDetail()
}
