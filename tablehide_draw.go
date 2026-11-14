package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The guests hiding under a table (pe_tablehide.go) used to be drawn as a dark hole with two feet in it: from a little
// way off it looked like a pair of shoes without a body. Now the cloth is lifted INSIDE its front (never over the top
// of the table nor under the hem) and the guest is seen crouched in the dark: head, shoulders and hands peeking out.
// hideOpening and hideFigure are pure geometry (tested in tablehide_draw_test.go): the opening lies inside the cloth
// of the sprite, the figure inside the opening, and the top of the figure is the top of the head.
// ---------------------------------------------------------------------------

// hideCloth: the rows of the cloth front of each hiding table, from the table's anchor (world px above tbl.Y):
// top = first row under the table top, bot = last row above the hem. (clothtable: design rows 9..28 of 30;
// seatlong: rows 9..19 of 22; design row r is at tbl.Y - H + r.)
var hideCloth = map[string]struct{ top, bot, halfW float64 }{
	"clothtable": {19, 4, 16},
	"seatlong":   {12, 3, 26},
}

const hideOpenHalfW = 7.0

// hideOpening: the lifted part of the cloth for a guest hiding at slotX (world rectangle).
func hideOpening(tbl *Prop, slotX float64) Rect {
	c, ok := hideCloth[tbl.Name]
	if !ok {
		c = hideCloth["clothtable"]
	}
	x := math.Max(tbl.X-c.halfW+hideOpenHalfW, math.Min(tbl.X+c.halfW-hideOpenHalfW, slotX))
	return Rect{x - hideOpenHalfW, tbl.Y - c.top, 2 * hideOpenHalfW, c.top - c.bot}
}

// hideFigure: the part of the crouched guest's picture shown in the opening: source rows [r0, r1) and columns
// [c0, c1) of the picture (anchor charCX, charBase), and where its top-left corner goes in the world.
func hideFigure(open Rect, headTop int) (src image.Rectangle, at Vec) {
	r0 := headTop
	r1 := minI(charBase, r0+int(open.H)-1)
	cx := open.X + open.W/2
	c0, c1 := charCX-int(hideOpenHalfW)+1, charCX+int(hideOpenHalfW)-1
	return image.Rect(c0, r0, c1, r1), Vec{cx - float64(charCX) + float64(c0), open.Y + 1}
}

// hideHeadTops: first painted row (the outline over the hair or hat) of each guest's crouched picture.
var hideHeadTops = map[*CharSet]int{}

func hideHeadTop(cs *CharSet) int {
	if v, ok := hideHeadTops[cs]; ok {
		return v
	}
	top := charBase - 28
	var b *pixBuf
	switch {
	case cs.rig != nil:
		b = cs.rigFrame(AnimSitRest, int(vFront), 0)
	case cs.Spec != nil:
		pp := poseFor(AnimSitRest, 0)
		pp.Anim, pp.F = AnimSitRest, 0
		b = genChar(cs.Spec, int(vFront), pp)
	}
	if b != nil {
	scan:
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				if b.p[(y*b.w+x)*4+3] > 0 {
					top = maxI(0, y-1) // the outline goes one row over it
					break scan
				}
			}
		}
	}
	if len(hideHeadTops) > 600 {
		hideHeadTops = map[*CharSet]int{}
	}
	hideHeadTops[cs] = top
	return top
}

// drawHider: the lifted cloth with the guest crouched under it.
func (g *Game) drawHider(dst *ebiten.Image, tbl *Prop, slotX float64, cs *CharSet) {
	open := hideOpening(tbl, slotX)
	x, y := g.wx(open.X), g.wy(open.Y)
	fillRect(dst, x, y, open.W, open.H, color.RGBA{0x24, 0x1a, 0x2a, 255})
	fillRect(dst, x, y, open.W, 1, color.RGBA{0x14, 0x0e, 0x18, 255}) // the shadow of the folded cloth
	if cs == nil {
		return
	}
	src, at := hideFigure(open, hideHeadTop(cs))
	if src.Empty() {
		return
	}
	img, _ := cs.img(AnimSitRest, DirDown, 0)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.wx(at.X), g.wy(at.Y))
	dst.DrawImage(img.SubImage(src.Add(img.Bounds().Min)).(*ebiten.Image), op)
}
