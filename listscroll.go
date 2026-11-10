package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// Hover scroll for the lists with a scroll bar (OBJETIVOS cards and trophies, the shop lists): when the mouse rests
// on the bottom strip of a list (edgeZone px) the list scrolls down slowly, on the top strip it scrolls up. Only
// when there is more to see that way, and only while the mouse is the device in use: any key or pad button stops it
// until the mouse moves again. One edgeScroll per list (by name), kept here so no screen needs a new field.
// ---------------------------------------------------------------------------

const edgeZone = 60.0 // px of the list, from its top or its bottom edge, that scroll it

type edgeScroll struct {
	lastX, lastY int
	seen         bool    // lastX/lastY are valid
	mouse        bool    // the mouse moved after the last key / pad press
	acc          float64 // rows accumulated, signed
}

var edgeStates = map[string]*edgeScroll{}

// tick: one frame. mx, my = pointer; keys = a key or pad button was pressed this frame; r = x, y, w, h of the list;
// canUp / canDown = there is more content that way; rate = rows per second at the very edge. It returns the rows to
// scroll now (negative = up).
func (e *edgeScroll) tick(mx, my int, keys bool, r [4]float64, canUp, canDown bool, dt, rate float64) int {
	if e.seen && (mx != e.lastX || my != e.lastY) {
		e.mouse = true
	}
	e.lastX, e.lastY, e.seen = mx, my, true
	if keys {
		e.mouse = false
	}
	x, y := float64(mx), float64(my)
	if !e.mouse || x < r[0] || x > r[0]+r[2] || y < r[1] || y > r[1]+r[3] {
		e.acc = 0
		return 0
	}
	zone := math.Min(edgeZone, r[3]/3)
	dir := 0.0
	depth := 0.0 // 0 at the inner border of the strip, 1 at the edge of the list
	switch {
	case y >= r[1]+r[3]-zone && canDown:
		dir, depth = 1, (y-(r[1]+r[3]-zone))/zone
	case y <= r[1]+zone && canUp:
		dir, depth = -1, (r[1]+zone-y)/zone
	}
	if dir == 0 {
		e.acc = 0
		return 0
	}
	if e.acc*dir < 0 {
		e.acc = 0
	}
	e.acc += dir * dt * rate * (0.35 + 0.65*depth) // gentle at the border of the strip, faster at the edge
	n := int(e.acc)
	e.acc -= float64(n)
	return n
}

// edgeInputKeys: was any key or gamepad button pressed this frame (the mouse is not the device in use then)?
func edgeInputKeys() bool {
	if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
		return true
	}
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if len(inpututil.AppendJustPressedGamepadButtons(id, nil)) > 0 {
			return true
		}
		for a := 0; a < ebiten.GamepadAxisCount(id); a++ {
			if v := ebiten.GamepadAxisValue(id, a); v > 0.5 || v < -0.5 {
				return true
			}
		}
	}
	return false
}

// edgeListScroll: the hover scroll of a list of n items shown rows at a time, cols per row, inside r. sel and scroll
// (in rows) are the list's own; it returns them moved so that the scroll advances and the selection stays in view
// (the caller's usual "keep the selection visible" clamp then leaves the scroll where it is).
func edgeListScroll(id string, sel, scroll, rows, cols, n int, r [4]float64, rate float64) (int, int) {
	e := edgeStates[id]
	if e == nil {
		e = &edgeScroll{}
		edgeStates[id] = e
	}
	mx, my := ebiten.CursorPosition()
	return edgeApply(e.tick(mx, my, edgeInputKeys(), r, scroll > 0, scroll+rows < (n+cols-1)/cols, 1.0/60, rate), sel, scroll, rows, cols, n)
}

// edgeApply moves scroll by d rows (within the list) and brings sel inside the new window, same column.
func edgeApply(d, sel, scroll, rows, cols, n int) (int, int) {
	if d == 0 || n <= 0 || cols <= 0 {
		return sel, scroll
	}
	total := (n + cols - 1) / cols
	scroll = clampI(scroll+d, 0, maxI(0, total-rows))
	row := clampI(sel/cols, scroll, scroll+rows-1)
	return clampI(row*cols+sel%cols, 0, n-1), scroll
}
