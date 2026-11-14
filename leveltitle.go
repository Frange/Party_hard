package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// The big title of the party at its start (drawn by drawHUD while g.titleT > 0), centred at the top of the screen.
// Rule of the margins: it keeps >= levelTitleGap px from the HUD panel on the left (and from its mirror place on the
// right), in Spanish and in English. A name too long for the big size (scale 4) is drawn one size smaller.

const levelTitleGap = 12.0

// levelTitleRoom: the width the title (with its outline) may take, centred, between the HUD panel and its mirror.
func levelTitleRoom() float64 { return screenW - 2*(hudX+hudW+levelTitleGap) }

// levelTitleSpan: the width the title takes at that scale, outline included (it reaches pxOff left, 2*pxOff right:
// counted on both sides, so the centred text is clear of the panel whatever side is wider).
func (g *Game) levelTitleSpan(s string, scale float64) float64 {
	return g.textW(s, scale) + 4*pxOff(pxScale(scale))
}

// levelTitleScale: 4, or the biggest smaller size that fits in levelTitleRoom (never below 2).
func (g *Game) levelTitleScale(s string) float64 {
	for sc := 4.0; sc > 2; sc-- {
		if g.levelTitleSpan(s, sc) <= levelTitleRoom() {
			return sc
		}
	}
	return 2
}

// levelCardGap: px between the bottom of the big title (outline included) and the small line under it.
const levelCardGap = 6.0

// levelCardLines: the small line of the party card (the kind of party and the style of its music), split in two at
// " · " when it does not fit in levelTitleRoom in one line.
func (g *Game) levelCardLines() []string {
	line := g.partyCardLine()
	if line == "" {
		return nil
	}
	if g.textW(line, 1) <= levelTitleRoom() {
		return []string{line}
	}
	if i := strings.Index(line, " · "); i > 0 {
		return []string{line[:i], line[i+len(" · "):]}
	}
	return []string{line}
}

// drawLevelTitle: the title (orange with a dark outline, fading out) and, small UNDER it, the kind of party and the
// style of its music (v1.4.0: it was above the title, 2 px from the top edge; now it keeps the margins: clear of the
// edge, of the title and, like the title, of the HUD panel).
func (g *Game) drawLevelTitle(dst *ebiten.Image) {
	a := math.Min(1, g.titleT)
	c := color.RGBA{uint8(255 * a), uint8(150 * a), uint8(30 * a), uint8(255 * a)}
	title := g.lvl.Def.Title
	g.bigText(dst, title, screenW/2, levelTitleY, c, color.RGBA{uint8(90 * a), uint8(30 * a), 0, uint8(255 * a)}, g.levelTitleScale(title))
	y := g.levelCardY()
	back := levelCardBacked[title]
	for _, line := range g.levelCardLines() {
		if back { // light facades under the line: a dark see-through strip so it reads (QA v1.4.0)
			w := math.Ceil(g.textW(line, 1))
			fillRect(dst, math.Round(screenW/2-w/2)-4, y-1, w+8, g.levelCardLineH(), color.RGBA{0, 0, 0, uint8(150 * a)})
		}
		g.drawTextC(dst, line, screenW/2, y, color.RGBA{uint8(200 * a), uint8(190 * a), uint8(230 * a), uint8(255 * a)}, 1)
		y += g.levelCardLineH()
	}
}

// levelCardBacked: the parties whose small card line sits over light cream facades (lilac on cream did not read): the
// line gets a dark see-through strip behind it.
var levelCardBacked = map[string]bool{alcalaTitle: true, guadalajaraTitle: true}

// levelTitleY: the top of the big title.
const levelTitleY = 12.0

// levelCardLineH: the step between two small lines of the card.
func (g *Game) levelCardLineH() float64 {
	_, h := textMeasure(g, "Ág")
	return math.Ceil(h) + 2
}

// levelCardY: the top of the small line(s) under the title: under the title's line box and its outline.
func (g *Game) levelCardY() float64 {
	title := g.lvl.Def.Title
	sc := pxScale(g.levelTitleScale(title))
	_, th := textMeasure(g, tr(title))
	return levelTitleY + math.Ceil(th*sc) + 2*pxOff(sc) + levelCardGap
}

// levelTitleBottom: the lowest y the party card (title and small lines) reaches; the messages under it start lower
// while it is up (draw.go).
func (g *Game) levelTitleBottom() float64 {
	return g.levelCardY() + float64(len(g.levelCardLines()))*g.levelCardLineH()
}

// msgBox: one message box of the HUD (the description of the party at its start, the hints...), as drawHUD paints it.
type msgBox struct {
	x, y, w, h float64
	lines      []string // already translated when wrapped (one line: the text as given)
}

// msgBoxes: the message boxes, centred, from y 62 (or under the party card while the title is up). Margins rule
// (v1.4.0, QA): a box is never wider than levelTitleRoom, so it keeps >= levelTitleGap px from the HUD panel and its
// column (whatever its height: forces lines, item, stamina) and from the mirror place on the right; a text too long is
// split in lines; boxes that would come within 16 px of the bottom edge are not shown.
func (g *Game) msgBoxes() []msgBox {
	y := 62.0 // below the level title, which it used to overlap
	if g.titleT > 0 && g.lvl != nil && g.lvl.Def != nil {
		// ... and below the small line(s) of the party card while the title is up
		y = math.Max(y, g.levelTitleBottom()+levelTitleGap)
	}
	room := levelTitleRoom()
	lh := g.levelCardLineH()
	var out []msgBox
	for _, m := range g.msgs {
		lines := []string{m.text}
		if g.textW(m.text, 1)+12 > room {
			lines = g.wrapText(m.text, 1, room-12)
		}
		mw := 0.0
		for _, s := range lines {
			mw = math.Max(mw, g.textW(s, 1)+12)
		}
		mw = math.Min(math.Ceil(mw), room)
		h := 17 + float64(len(lines)-1)*lh
		if y+h > screenH-16 {
			break
		}
		out = append(out, msgBox{math.Round(screenW/2 - mw/2), y, mw, h, lines})
		y += h + 2
	}
	return out
}

// drawMsgBoxes: the message boxes of msgBoxes.
func (g *Game) drawMsgBoxes(dst *ebiten.Image) {
	for _, b := range g.msgBoxes() {
		g.panel(dst, b.x, b.y, b.w, b.h)
		for i, s := range b.lines {
			g.drawTextC(dst, s, screenW/2, b.y+1+float64(i)*g.levelCardLineH(), rgb(0xffffff), 1)
		}
	}
}
