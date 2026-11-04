package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The cinematics with two players (intro, interludes, endings, the invitation).
//
// With the 1 / 2 switch of the main menu on 2 (or -twop) every story scene has a second hero: the
// first player is the blue one and the second the red one (the look each of them wears in the
// wardrobe, see wardrobe.go). They share the bedroom (two separate beds), the street, the car, the
// boat, the rocket, the letter... and they get in each other's way, which is the joke. With one
// player nothing here runs and every scene is exactly as it was.
// ---------------------------------------------------------------------------

// scene2P: do the cinematics play their two player version?
func (g *Game) scene2P() bool { return g.wantTwoPlayers() }

// sceneHero is the animated look of player i (0 or 1) for the cinematics, optionally carrying a knife.
func (g *Game) sceneHero(i int, knife bool) *CharSet {
	var cs *CharSet
	if g.as.playerCS != nil || g.shopCSReady() {
		cs = g.playerCS(i)
	}
	if cs == nil {
		cs = g.as.playerCS
	}
	if cs == nil {
		return nil
	}
	if knife {
		return knifeHeroSet(cs)
	}
	return cs
}

// shopCSReady: the wardrobe can build the characters (a game in the middle of tests may not have a store).
func (g *Game) shopCSReady() bool { return g.as != nil && g.as.playerCS != nil }

// sceneTint is the colour a player is drawn with when both of them would look the same: nothing for
// the first (blue by default) and a red wash for the second if his set is the same object as the first one's.
func sceneTint(i int, a, b *CharSet) [3]float32 {
	if i == 1 && a != nil && a == b {
		return [3]float32{1, 0.55, 0.55}
	}
	return [3]float32{1, 1, 1}
}

// sceneChar draws a character with its feet at (x, y) at scale s: alpha, a hop (pixels up), a lean (radians)
// and a tint.
func (g *Game) sceneChar(c *ebiten.Image, cs *CharSet, anim int, dir Dir, ph, x, y, s, alpha, hop, lean float64, tint [3]float32) {
	if cs == nil || alpha <= 0 {
		return
	}
	img, flip := cs.img(anim, dir, ph)
	if img == nil {
		return
	}
	fillShadow(c, x, y, s/4)
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(charW, 0)
	}
	op.GeoM.Translate(-charCX, -charBase)
	op.GeoM.Scale(s, s)
	if lean != 0 {
		op.GeoM.Rotate(lean)
	}
	op.GeoM.Translate(math.Round(x), math.Round(y-hop))
	op.ColorScale.Scale(tint[0], tint[1], tint[2], 1)
	op.ColorScale.ScaleAlpha(float32(alpha))
	c.DrawImage(img, op)
}

// ---- small effects used by the gags ----------------------------------------------------------

// sceneStars: three little stars circling a head (the same ones the players see when they are punched).
func (g *Game) sceneStars(c *ebiten.Image, x, y, t float64, a float64) {
	for i := 0; i < 3; i++ {
		ang := t*7 + float64(i)*2.1
		sx, sy := x+math.Cos(ang)*14, y+math.Sin(ang)*5
		col := rgba(0xffe060, uint8(255*a))
		fillRect(c, math.Round(sx)-1, math.Round(sy)-3, 2, 6, col)
		fillRect(c, math.Round(sx)-3, math.Round(sy)-1, 6, 2, col)
	}
}

// sceneBang: a comic word that pops and fades.
func (g *Game) sceneBang(c *ebiten.Image, s string, x, y, u float64, col uint32, size float64) {
	if u < 0 || u > 1 {
		return
	}
	k := 1 + 0.5*math.Exp(-u*9)
	g.drawTextC(c, s, x, y-u*18, rgba(col, uint8(255*math.Min(1, 2*(1-u)+0.1))), size*k)
}

// sceneWindow: how much of 0..1 the time t is between a and b.
func sceneWindow(t, a, b float64) float64 {
	if b <= a {
		return 0
	}
	return math.Max(0, math.Min(1, (t-a)/(b-a)))
}

// sceneIn: is t in [a, b)?
func sceneIn(t, a, b float64) bool { return t >= a && t < b }

var _ = color.RGBA{}
