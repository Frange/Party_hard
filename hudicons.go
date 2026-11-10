package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// STATUS ICONS of the HUD: tiny 9x9 pixel-art icons on the TIME row of the panel, to the LEFT, right after the clock
// (so they sit next to the counter and never touch the BAJAS figures nor the RETO / CADENA chips on the right).
// They only exist while the state is active:
//
//	hudCuffs     the police are after this player (BUSCADO / cuffs)
//	hudAlert     a witness saw him and is on his way to warn the police (the "!" of the alert)
//	hudPoison    he carries poison (hand or second slot)
//	hudSuit      the gold VIP suit is on (blinks in its last seconds)
//	hudDisguise  he carries a disguise or already wears it
//
// In 2 players each icon has a 1 px line under it in the colour of its player. Whole-pixel positions and sizes only.
// ---------------------------------------------------------------------------

const (
	hudCuffs = iota
	hudAlert
	hudPoison
	hudSuit
	hudDisguise
)

const (
	hudIconW   = 9.0
	hudIconGap = 2.0
	hudIconDY  = -1.0 // relative to the TIME row (the 9 px icon is centred on the 7 px clock text)
)

// hudIconAdv: width of one icon in the strip (the cuffs are wider than the 9 px badges).
func hudIconAdv(kind int) float64 {
	if kind == hudCuffs {
		w, _ := cuffsSize(cuffsTiny)
		return float64(w)
	}
	return hudIconW
}

// hudIconH: height of one icon (the cuffs are 2 px taller than the badges).
func hudIconH(kind int) float64 {
	if kind == hudCuffs {
		_, h := cuffsSize(cuffsTiny)
		return float64(h)
	}
	return hudIconW
}

// hudStripW: total width of a list of icons.
func hudStripW(icons []hudIcon) float64 {
	w := 0.0
	for i, ic := range icons {
		if i > 0 {
			w += hudIconGap
		}
		w += hudIconAdv(ic.kind)
	}
	return w
}

type hudIcon struct {
	kind   int
	player int // index of the player (2P tint)
}

// hudStatus: the icons p shows right now, in a fixed order.
func (g *Game) hudStatus(p *Player) []int {
	if p == nil || p.dead {
		return nil
	}
	var out []int
	if g.isSuspectP(p) {
		out = append(out, hudCuffs)
	} else if g.playerAlert(p) {
		out = append(out, hudAlert)
	}
	if p.item == "poison" || p.spare == "poison" {
		out = append(out, hudPoison)
	}
	if p.suitT > 0 {
		out = append(out, hudSuit)
	}
	if p.item == "disguise" || p.spare == "disguise" || p.disguised {
		out = append(out, hudDisguise)
	}
	return out
}

// hudIcons: every icon of the players, in order.
func (g *Game) hudIcons() []hudIcon {
	var out []hudIcon
	for i, p := range g.players() {
		for _, k := range g.hudStatus(p) {
			out = append(out, hudIcon{k, i})
		}
	}
	return out
}

// hudIconsX: left edge of the strip: after the clock icon and the time text.
func (g *Game) hudIconsX() float64 {
	return hudX + hudPad + 11 + g.textW(hudClockText(g.time), 1) + 8
}

func (g *Game) drawHUDStatus(dst *ebiten.Image, x, y float64) {
	if g.mode != ModePlay {
		return
	}
	icons := g.hudIcons()
	for _, ic := range icons {
		blink := ic.kind == hudSuit && g.playerAt(ic.player).suitT <= suitWarn && int(g.time*8)%2 == 0
		if !blink {
			g.drawHUDIcon(dst, ic.kind, x, y)
		}
		if g.is2P() {
			c := ringColor(g.playerAt(ic.player))
			fillRect(dst, x, y+hudIconW+1+hudIconH(ic.kind)-hudIconW, hudIconAdv(ic.kind), 1, color.RGBA{c[0], c[1], c[2], 255})
		}
		x += hudIconAdv(ic.kind) + hudIconGap
	}
}

func (g *Game) playerAt(i int) *Player {
	ps := g.players()
	if i < 0 || i >= len(ps) {
		return g.p
	}
	return ps[i]
}

// drawHUDIcon paints one 9x9 icon with a dark edge.
var hudIconDrawn int // tests: icons painted

func (g *Game) drawHUDIcon(dst *ebiten.Image, kind int, x, y float64) {
	hudIconDrawn++
	dark := rgb(0x1a0c10)
	switch kind {
	case hudCuffs: // the shared handcuffs symbol (cuffs_art.go), tiny tier
		drawCuffsIcon(dst, cuffsTiny, x, y, 1)
	case hudAlert: // the "!" badge, it pulses like the one over the head
		body := rgb(0xffd23a)
		ink := rgb(0x3a0a04)
		if math.Sin(g.time*2*math.Pi*4) > 0 {
			body, ink = rgb(0xff4a2a), rgb(0xfff2c0)
		}
		fillRect(dst, x, y+1, 9, 7, dark)
		fillRect(dst, x+1, y, 7, 9, dark)
		fillRect(dst, x+1, y+1, 7, 7, body)
		fillRect(dst, x+4, y+2, 1, 3, ink)
		fillRect(dst, x+4, y+6, 1, 1, ink)
	case hudPoison: // green bottle with a white skull dot
		green := rgb(0x50e070)
		fillRect(dst, x+2, y, 5, 9, dark)
		fillRect(dst, x+3, y, 3, 1, rgb(0xb89060))
		fillRect(dst, x+3, y+1, 3, 2, green)
		fillRect(dst, x+2, y+3, 5, 5, green)
		fillRect(dst, x+3, y+4, 1, 1, dark)
		fillRect(dst, x+5, y+4, 1, 1, dark)
		fillRect(dst, x+3, y+6, 3, 1, rgb(0xf4f4f4))
	case hudSuit: // gold jacket with a dark V
		gold := rgb(0xf0c030)
		fillRect(dst, x, y, 9, 9, dark)
		fillRect(dst, x+1, y+1, 7, 7, gold)
		fillRect(dst, x+3, y+1, 3, 1, dark)
		fillRect(dst, x+3, y+2, 1, 1, dark)
		fillRect(dst, x+5, y+2, 1, 1, dark)
		fillRect(dst, x+4, y+3, 1, 1, dark)
		fillRect(dst, x+4, y+4, 1, 3, rgb(0xfff0a0))
	case hudDisguise: // a purple mask with two eye holes
		pur := rgb(0xb070f0)
		fillRect(dst, x, y+1, 9, 7, dark)
		fillRect(dst, x+1, y, 7, 9, dark)
		fillRect(dst, x+1, y+1, 7, 7, pur)
		fillRect(dst, x+2, y+3, 2, 2, dark)
		fillRect(dst, x+5, y+3, 2, 2, dark)
		fillRect(dst, x+3, y+6, 3, 1, shade(pur, 0.6))
	}
}
