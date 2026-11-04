package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Text recorder used by the UI text audit (uitext_audit_test.go). While textRec is nil (always, in the game)
// the hooks in drawText/bigText/gradText/plainText and panel cost one nil check.

type textRect struct {
	kind  string // drawText | bigText | gradText | plainText
	s     string // as drawn (already translated)
	x, y  float64
	w, h  float64
	scale float64
	dstW  int // size of the image it was drawn on (the screen is screenW x screenH)
	dstH  int
	clip  image.Rectangle // bounds of the image it was drawn on
	sub   bool            // the image is a SubImage of a bigger one (a clip: coordinates are the parent's)
}

type panelRect struct {
	x, y, w, h float64
	opaque     bool // an opaque fill hides what was drawn before it
	seq        int  // number of texts recorded before it
}

type textRecorder struct {
	texts  []textRect
	panels []panelRect
	fills  []panelRect // big filled rectangles (panels drawn with fillRect)
}

var textRec *textRecorder

func (g *Game) recText(dst *ebiten.Image, kind, s string, x, y, scale float64) {
	if textRec == nil || s == "" {
		return
	}
	w, h := textMeasure(g, s)
	textRec.texts = append(textRec.texts, textRect{kind, s, x, y, w * scale, h * scale, scale, dst.Bounds().Dx(), dst.Bounds().Dy(), dst.Bounds(), dst.Bounds().Min != image.Point{}})
}

func (g *Game) recPanel(x, y, w, h float64) {
	if textRec != nil {
		textRec.panels = append(textRec.panels, panelRect{x, y, w, h, false, len(textRec.texts)})
	}
}
