package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The editor's widget kit: icon buttons, segmented choices, headings,
// thumbnail grids and pop-up menus. Same immediate-mode idea as ui.go.
// ---------------------------------------------------------------------------

type segItem struct {
	icon, label string
}

// iconBtn is a button with a small icon on the left and a label.
func (u *UI) iconBtn(x, y, w, h int, icon, label string, on bool, tip string) bool {
	hot := u.in(x, y, w, h) && u.grab == ""
	col := uiBtn
	if on {
		col = uiBtnOn
	}
	if hot {
		col = uiBtnHot
		if on {
			col = uiBtnOnHot
		}
	}
	u.round(x, y, w, h, col)
	if on {
		u.roundFrame(x, y, w, h, rgb(0xa898ff))
	}
	is := 2.0
	if h < 26 {
		is = 1.5
	}
	iw := int(12 * is)
	tx := x + w/2
	ix := x + 6
	if label == "" {
		ix = x + (w-iw)/2
	} else {
		tx = x + 6 + iw + (w-6-iw)/2
		if w < 70 || icon == "" {
			tx = x + w/2
		}
	}
	if icon != "" {
		iy := y + (h-iw)/2
		main := uiText
		if on {
			main = rgb(0xffffff)
		}
		u.add(func(d *ebiten.Image) { drawIcon(d, icon, float64(ix), float64(iy), is, main, uiAccent) })
	}
	if label != "" {
		ty := y + (h-12)/2
		if icon != "" && w >= 70 {
			u.add(func(d *ebiten.Image) { u.g.plainText(d, label, float64(x+6+iw+6), float64(ty), uiText) })
		} else {
			u.textC(tx, ty, label, uiText)
		}
	}
	if tip != "" {
		u.tooltip(x, y, w, h, tip)
	}
	if hot && u.pressed {
		u.focus = ""
		return true
	}
	return false
}

// seg draws a row of exclusive choices and returns the clicked one (or -1).
func (u *UI) seg(x, y, w, h int, items []segItem, sel int) int {
	n := len(items)
	if n == 0 {
		return -1
	}
	cw := w / n
	click := -1
	for i, it := range items {
		bx := x + i*cw
		bw := cw - 2
		if i == n-1 {
			bw = x + w - bx
		}
		if u.iconBtn(bx, y, bw, h, it.icon, it.label, i == sel, "") {
			click = i
		}
	}
	return click
}

// heading is a small section title with a rule.
func (u *UI) heading(x, y, w int, text string) {
	u.text(x, y, text, uiAccent)
	tw := int(u.g.textW(text, 1))
	u.rect(x+tw+8, y+6, w-tw-8, 1, uiLine)
}

// stepper: "label  [-] value [+]". Returns true when the value changed.
func (u *UI) stepper(x, y int, label string, val *int, lo, hi, step int) bool {
	u.text(x, y+4, label, uiDim)
	vx := x + int(u.g.textW(label, 1)) + 10
	changed := false
	if u.iconBtn(vx, y, 22, 20, "minus", "", false, "") && *val-step >= lo {
		*val -= step
		changed = true
	}
	u.textC(vx+22+22, y+4, fmt.Sprint(*val), uiText)
	if u.iconBtn(vx+22+44, y, 22, 20, "plus", "", false, "") && *val+step <= hi {
		*val += step
		changed = true
	}
	return changed
}

// thumb is one cell of a thumbnail grid.
type thumb struct {
	label string
	img   *ebiten.Image // nil while it loads
}

// thumbGrid shows a scrollable grid of thumbnails with their names and
// returns the clicked index (or -1).
func (u *UI) thumbGrid(x, y, w, h int, cols int, items []thumb, sel int, scroll *int) int {
	u.round(x, y, w, h, rgb(0x101018))
	u.roundFrame(x, y, w, h, uiLine)
	cw := (w - 8) / cols
	ch := cw + 14
	rows := (h - 6) / ch
	total := (len(items) + cols - 1) / cols
	if u.in(x, y, w, h) && u.wheel != 0 {
		*scroll -= int(math.Round(u.wheel))
	}
	*scroll = clampI(*scroll, 0, maxI(0, total-rows))
	clicked := -1
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			idx := (r+*scroll)*cols + c
			if idx >= len(items) {
				break
			}
			it := items[idx]
			cx, cy := x+4+c*cw, y+3+r*ch
			hot := u.in(cx, cy, cw-2, ch-2)
			switch {
			case idx == sel:
				u.round(cx, cy, cw-2, ch-2, uiBtnOn)
			case hot:
				u.round(cx, cy, cw-2, ch-2, uiBtnHot)
			default:
				u.round(cx, cy, cw-2, ch-2, uiCard)
			}
			if it.img != nil {
				im := it.img
				iw, ih := im.Bounds().Dx(), im.Bounds().Dy()
				k := math.Min(float64(cw-10)/float64(iw), float64(ch-22)/float64(ih))
				if k >= 1 {
					k = math.Floor(k)
				}
				ox := float64(cx) + (float64(cw-2)-float64(iw)*k)/2
				oy := float64(cy) + 2 + (float64(ch-18)-float64(ih)*k)/2
				u.add(func(d *ebiten.Image) {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Scale(k, k)
					op.GeoM.Translate(math.Round(ox), math.Round(oy))
					d.DrawImage(im, op)
				})
			}
			lab := it.label
			for u.g.textW(lab, 1) > float64(cw-6) && len(lab) > 2 {
				lab = lab[:len(lab)-1]
			}
			u.textC(cx+(cw-2)/2, cy+ch-16, lab, uiText)
			if hot && u.pressed {
				clicked = idx
			}
			if hot {
				u.tip = it.label
			}
		}
	}
	if total > rows {
		// scrollbar
		th := maxI(14, (h-8)*rows/total)
		ty := y + 4 + (h-8-th)**scroll/maxI(1, total-rows)
		u.round(x+w-5, ty, 3, th, uiDim)
	}
	return clicked
}

// ---- pop-up menu ------------------------------------------------------------------

type menuState struct {
	id         string
	x, y, w    int
	by         int // top of the button that opened it (the menu flips upwards when it does not fit)
	items      []segItem
	resultID   string
	result     int
	pressedRaw bool
}

// menuBtn is a button that opens a pop-up list. It returns the chosen item once, else -1.
func (u *UI) menuBtn(x, y, w, h int, icon, label, id string, items []segItem, tip string) int {
	if u.iconBtn(x, y, w, h, icon, label, u.menu.id == id, tip) {
		if u.menu.id == id {
			u.menu.id = ""
		} else {
			u.menu.id, u.menu.x, u.menu.y, u.menu.w, u.menu.items, u.menu.by = id, x, y+h+2, maxI(w, 190), items, y
			if x+u.menu.w > edW-4 {
				u.menu.x = edW - 4 - u.menu.w
			}
		}
	}
	if u.menu.resultID == id {
		r := u.menu.result
		u.menu.resultID = ""
		return r
	}
	return -1
}

func (u *UI) menuRows() int { return len(u.menu.items) }

// finishMenu draws the open pop-up on top of everything and handles its clicks. Call it last.
func (u *UI) finishMenu() {
	m := &u.menu
	if m.id == "" {
		return
	}
	rh := 26
	h := len(m.items)*rh + 8
	if m.y+h > edH-4 {
		m.y = maxI(4, m.by-h-2)
	}
	u.round(m.x-1, m.y-1, m.w+2, h+2, uiLine)
	u.round(m.x, m.y, m.w, h, uiPanel)
	for i, it := range m.items {
		by := m.y + 4 + i*rh
		hot := u.mx >= m.x && u.mx < m.x+m.w && u.my >= by && u.my < by+rh
		if hot {
			u.round(m.x+3, by, m.w-6, rh-2, uiBtnHot)
		}
		ix, iy := m.x+10, by+(rh-2-18)/2
		icon, label := it.icon, it.label
		if icon != "" {
			u.add(func(d *ebiten.Image) { drawIcon(d, icon, float64(ix), float64(iy), 1.5, uiText, uiAccent) })
		}
		u.text(m.x+34, by+7, label, uiText)
		if hot && m.pressedRaw {
			m.resultID, m.result, m.id = m.id, i, ""
			return
		}
	}
	if m.pressedRaw {
		m.id = "" // a click outside closes it
	}
}

// swatch draws a colour chip.
func (u *UI) swatch(x, y, w, h int, c color.RGBA) {
	u.round(x, y, w, h, c)
	u.roundFrame(x, y, w, h, uiLine)
}
