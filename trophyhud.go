package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// The trophy counter in the top right corner (main menu and levels) while there
// are trophies left to win. Inside a level it turns see-through when the player
// walks under it, like the tall props do. Clicking it pauses the game and opens
// a window with every trophy and its progress.
// ---------------------------------------------------------------------------

const (
	trophyHudW, trophyHudH = 92.0, 24.0
	trophyHudX, trophyHudY = float64(960) - trophyHudW - 8, 6.0
	trophyCloseX           = 900.0 // the close button of the window
	trophyCloseY           = 62.0
)

// trophyHudShown: menu and levels, and only until every trophy is won.
func (g *Game) trophyHudShown() bool {
	if g.allTrophies() || g.iconMode {
		return false
	}
	return g.mode == ModeMenu || g.mode == ModePlay
}

// updateTrophyHud handles the click on the counter and the window. It returns true while the
// window is open: the rest of the frame is skipped, which is what pauses the game.
func (g *Game) updateTrophyHud() bool {
	if g.trophyOv {
		if !g.trophyHudShown() && g.mode != ModeMenu && g.mode != ModePlay {
			g.trophyOv = false
			return false
		}
		g.updateTrophyList()
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			fx, fy := float64(mx), float64(my)
			outside := fx < 230 || fx > 930 || fy < 56 || fy > 496
			onX := fx >= trophyCloseX && fx <= trophyCloseX+22 && fy >= trophyCloseY-4 && fy <= trophyCloseY+18
			if outside || onX {
				g.trophyOv = false
			}
		}
		if actBack.just() || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.trophyOv = false
		}
		return true
	}
	if !g.trophyHudShown() {
		return false
	}
	// see-through while the player is in that corner of the screen
	tgt := 1.0
	if g.mode == ModePlay {
		sx, sy := g.wx(g.p.pos.X), g.wy(g.p.pos.Y)
		if sx > trophyHudX-30 && sx < trophyHudX+trophyHudW+30 && sy > trophyHudY-10 && sy < trophyHudY+trophyHudH+46 {
			tgt = 0.3
		}
	}
	g.trophyHudA += (tgt - g.trophyHudA) * 0.2
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		fx, fy := float64(mx), float64(my)
		if fx >= trophyHudX && fx <= trophyHudX+trophyHudW && fy >= trophyHudY && fy <= trophyHudY+trophyHudH {
			g.trophyOv = true
			g.trophySel, g.trophyScroll = 0, 0
			return true
		}
	}
	return false
}

func (g *Game) drawTrophyHud(dst *ebiten.Image) {
	if g.trophyHudShown() {
		if g.trophyHudA <= 0 {
			g.trophyHudA = 1
		}
		a := g.trophyHudA
		mx, my := ebiten.CursorPosition()
		hot := float64(mx) >= trophyHudX && float64(mx) <= trophyHudX+trophyHudW && float64(my) >= trophyHudY && float64(my) <= trophyHudY+trophyHudH
		bg := rgba(0x05070e, uint8(190*a))
		if hot && a > 0.9 {
			bg = rgba(0x2a1a4a, 220)
		}
		fillRect(dst, trophyHudX, trophyHudY, trophyHudW, trophyHudH, bg)
		fillRect(dst, trophyHudX, trophyHudY, trophyHudW, 1, rgba(0x6a4ab0, uint8(200*a)))
		fillRect(dst, trophyHudX, trophyHudY+trophyHudH-1, trophyHudW, 1, rgba(0x6a4ab0, uint8(200*a)))
		cup := rgba(0xe8c040, uint8(255*a))
		x, y := trophyHudX+8, trophyHudY+5
		filledEllipse(dst, x+7, y+5, 6, 5, cup)
		fillRect(dst, x+1, y, 12, 6, cup)
		fillRect(dst, x+5, y+9, 3, 4, cup)
		fillRect(dst, x+2, y+13, 9, 2, cup)
		fillRect(dst, x-1, y+1, 2, 4, cup)
		fillRect(dst, x+13, y+1, 2, 4, cup)
		txt := fmt.Sprintf("%d/%d", g.trophyCount(), len(trophyList))
		g.drawText(dst, txt, trophyHudX+30, trophyHudY+5, rgba(0xf0e060, uint8(255*a)), 1.3)
	}
	if g.trophyOv {
		fillRect(dst, 0, 0, float64(screenW), float64(screenH), rgba(0x000000, 170))
		g.drawTrophyPanel(dst)
		fillRect(dst, trophyCloseX, trophyCloseY-4, 22, 22, rgba(0x6a1a2a, 230))
		g.drawTextC(dst, "X", trophyCloseX+11, trophyCloseY, rgb(0xffd0d8), 1.3)
		_ = math.Pi
	}
}
