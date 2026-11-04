package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// Moving things in the scenario editor.
//
//   Mover  (Objetos tab, and the "Mover" tool of Puntos): drag objects, the
//          start, the police points, helipads, guard route points, musicians,
//          lights and animals. In the game's own scenarios (debug mode) also
//          the ground (helipads, dance floors, rugs...) and the water.
//   Zona   (Objetos tab): select a rectangle and move or copy it with
//          everything inside: the background pixels, the walls / water / zones
//          cells, the objects and the points. Delete empties it.
//
// Everything is undoable (Ctrl+Z / Ctrl+Y). Arrow keys nudge by one logic cell
// (Shift: one pixel for single things).
// ---------------------------------------------------------------------------

type lvKind int

const (
	itNone lvKind = iota
	itProp
	itStart
	itCarFrom
	itCarTo
	itPad
	itGuard
	itPerf
	itAnimal
	itLight
	itFloor
	itPool
)

// lvItem names one movable thing: kind, index, and (for guard routes) the point inside it.
type lvItem struct {
	kind lvKind
	i, j int
}

const (
	maskProps = 1 << iota
	maskPoints
	maskGround
)

// lvSnap is a copy of everything the move tools can change.
type lvSnap struct {
	meta        LevelMeta
	solid, zone [][]byte
	bg          []byte // nil: the background is not part of this snapshot
}

type zoneContent struct {
	x0, y0, x1, y1 int // pixels, aligned to logic cells, end exclusive
	solid, zone    [][]byte
	bg             []byte // w*h*4
	props          []int
	pts            []lvItem
	grounds        []lvItem
}

type lvMove struct {
	hover, sel lvItem
	drag       bool
	grab       [2]int
	snap       *lvSnap
	moved      bool
	undo, redo []*lvSnap

	// the "Zona" tool
	zrect    [4]int
	zhas     bool
	zmode    int // 0 idle, 1 selecting, 2 moving
	zstart   [2]int
	zgrab    [2]int
	zd       [2]int
	zcopy    bool // the "Copiar" button: copy instead of move
	zcopyNow bool // copying during the current drag (button or Alt)
	zsnap    *lvSnap
	zc       zoneContent
	needDraw bool      // originals: paint the background again from the moved lists
	origDef  *LevelDef // the generated scenario of the open original
	elScroll int
}

func copyGrid(g [][]byte) [][]byte {
	o := make([][]byte, len(g))
	for i := range g {
		o[i] = append([]byte(nil), g[i]...)
	}
	return o
}

func (l *LevEd) takeSnap(withBG bool) *lvSnap {
	m := &l.la.Meta
	s := &lvSnap{solid: copyGrid(l.solid), zone: copyGrid(l.zone)}
	s.meta.Props = append([]LevelProp(nil), m.Props...)
	s.meta.Pads = append([]LevelPoint(nil), m.Pads...)
	for _, r := range m.Guards {
		s.meta.Guards = append(s.meta.Guards, append([]LevelPoint(nil), r...))
	}
	s.meta.Performers = append([]LevelPerf(nil), m.Performers...)
	s.meta.Animals = append([]LevelAnimal(nil), m.Animals...)
	s.meta.Lights = append([]LevelLight(nil), m.Lights...)
	s.meta.Floors = append([]LevelFloor(nil), m.Floors...)
	s.meta.Pools = append([]LevelRect(nil), m.Pools...)
	s.meta.Start, s.meta.CarFrom, s.meta.CarTo = m.Start, m.CarFrom, m.CarTo
	if withBG {
		s.bg = append([]byte(nil), l.la.BG.p...)
	}
	return s
}

func (l *LevEd) restoreSnap(s *lvSnap) {
	m := &l.la.Meta
	l.solid, l.zone = copyGrid(s.solid), copyGrid(s.zone)
	m.Props = append([]LevelProp(nil), s.meta.Props...)
	m.Pads = append([]LevelPoint(nil), s.meta.Pads...)
	m.Guards = nil
	for _, r := range s.meta.Guards {
		m.Guards = append(m.Guards, append([]LevelPoint(nil), r...))
	}
	m.Performers = append([]LevelPerf(nil), s.meta.Performers...)
	m.Animals = append([]LevelAnimal(nil), s.meta.Animals...)
	m.Lights = append([]LevelLight(nil), s.meta.Lights...)
	m.Floors = append([]LevelFloor(nil), s.meta.Floors...)
	m.Pools = append([]LevelRect(nil), s.meta.Pools...)
	m.Start, m.CarFrom, m.CarTo = s.meta.Start, s.meta.CarFrom, s.meta.CarTo
	if s.bg != nil {
		copy(l.la.BG.p, s.bg)
		l.cv.dirty = true
	}
}

func (l *LevEd) pushMoveUndo(s *lvSnap) {
	l.mv.undo = append(l.mv.undo, s)
	if len(l.mv.undo) > 24 {
		l.mv.undo = l.mv.undo[1:]
	}
	l.mv.redo = nil
}

func (l *LevEd) moveUndo() bool {
	n := len(l.mv.undo)
	if n == 0 {
		return false
	}
	s := l.mv.undo[n-1]
	l.mv.undo = l.mv.undo[:n-1]
	l.mv.redo = append(l.mv.redo, l.takeSnap(s.bg != nil))
	l.restoreSnap(s)
	l.afterMoveRestore()
	return true
}

func (l *LevEd) moveRedo() bool {
	n := len(l.mv.redo)
	if n == 0 {
		return false
	}
	s := l.mv.redo[n-1]
	l.mv.redo = l.mv.redo[:n-1]
	l.mv.undo = append(l.mv.undo, l.takeSnap(s.bg != nil))
	l.restoreSnap(s)
	l.afterMoveRestore()
	return true
}

func (l *LevEd) afterMoveRestore() {
	l.mv.sel, l.selInst, l.mv.zhas = lvItem{}, -1, false
	if l.overrideKey != "" {
		l.mv.needDraw = true
	}
}

// ---- the things -----------------------------------------------------------------

func (l *LevEd) itemPos(it lvItem) (x, y int, ok bool) {
	m := &l.la.Meta
	switch it.kind {
	case itProp:
		if it.i >= 0 && it.i < len(m.Props) {
			return m.Props[it.i].X, m.Props[it.i].Y, true
		}
	case itStart:
		return m.Start.X, m.Start.Y, true
	case itCarFrom:
		return m.CarFrom.X, m.CarFrom.Y, true
	case itCarTo:
		return m.CarTo.X, m.CarTo.Y, true
	case itPad:
		if it.i >= 0 && it.i < len(m.Pads) {
			return m.Pads[it.i].X, m.Pads[it.i].Y, true
		}
	case itGuard:
		if it.i >= 0 && it.i < len(m.Guards) && it.j >= 0 && it.j < len(m.Guards[it.i]) {
			p := m.Guards[it.i][it.j]
			return p.X, p.Y, true
		}
	case itPerf:
		if it.i >= 0 && it.i < len(m.Performers) {
			return m.Performers[it.i].X, m.Performers[it.i].Y, true
		}
	case itAnimal:
		if it.i >= 0 && it.i < len(m.Animals) {
			return m.Animals[it.i].X, m.Animals[it.i].Y, true
		}
	case itLight:
		if it.i >= 0 && it.i < len(m.Lights) {
			return m.Lights[it.i].X, m.Lights[it.i].Y, true
		}
	case itFloor:
		if it.i >= 0 && it.i < len(m.Floors) {
			return m.Floors[it.i].X, m.Floors[it.i].Y, true
		}
	case itPool:
		if it.i >= 0 && it.i < len(m.Pools) {
			return m.Pools[it.i].X, m.Pools[it.i].Y, true
		}
	}
	return 0, 0, false
}

func (l *LevEd) setItemPos(it lvItem, x, y int) {
	if _, _, ok := l.itemPos(it); !ok {
		return
	}
	m := &l.la.Meta
	switch it.kind {
	case itProp:
		m.Props[it.i].X, m.Props[it.i].Y = x, y
	case itStart:
		m.Start = LevelPoint{x, y}
	case itCarFrom:
		m.CarFrom = LevelPoint{x, y}
	case itCarTo:
		m.CarTo = LevelPoint{x, y}
	case itPad:
		m.Pads[it.i] = LevelPoint{x, y}
	case itGuard:
		m.Guards[it.i][it.j] = LevelPoint{x, y}
	case itPerf:
		m.Performers[it.i].X, m.Performers[it.i].Y = x, y
	case itAnimal:
		m.Animals[it.i].X, m.Animals[it.i].Y = x, y
	case itLight:
		m.Lights[it.i].X, m.Lights[it.i].Y = x, y
	case itFloor:
		m.Floors[it.i].X, m.Floors[it.i].Y = x, y
	case itPool:
		m.Pools[it.i].X, m.Pools[it.i].Y = x, y
	}
}

// delItem removes a thing (the start and the police points cannot be removed).
func (l *LevEd) delItem(it lvItem) bool {
	if _, _, ok := l.itemPos(it); !ok {
		return false
	}
	m := &l.la.Meta
	switch it.kind {
	case itProp:
		m.Props = append(m.Props[:it.i], m.Props[it.i+1:]...)
	case itPad:
		m.Pads = append(m.Pads[:it.i], m.Pads[it.i+1:]...)
	case itGuard:
		m.Guards[it.i] = append(m.Guards[it.i][:it.j], m.Guards[it.i][it.j+1:]...)
	case itPerf:
		m.Performers = append(m.Performers[:it.i], m.Performers[it.i+1:]...)
	case itAnimal:
		m.Animals = append(m.Animals[:it.i], m.Animals[it.i+1:]...)
	case itLight:
		m.Lights = append(m.Lights[:it.i], m.Lights[it.i+1:]...)
	case itFloor:
		m.Floors = append(m.Floors[:it.i], m.Floors[it.i+1:]...)
	case itPool:
		m.Pools = append(m.Pools[:it.i], m.Pools[it.i+1:]...)
	default:
		return false
	}
	return true
}

func (it lvItem) label(m *LevelMeta) string {
	switch it.kind {
	case itProp:
		if it.i < len(m.Props) {
			return "Objeto: " + m.Props[it.i].Kind
		}
	case itStart:
		return "Inicio del jugador"
	case itCarFrom:
		return "Policía: llega"
	case itCarTo:
		return "Policía: aparca"
	case itPad:
		return fmt.Sprintf("Helipuerto %d", it.i+1)
	case itGuard:
		return fmt.Sprintf("Guardia: ruta %d, punto %d", it.i+1, it.j+1)
	case itPerf:
		if it.i < len(m.Performers) {
			return "Músico: " + perfLabels[m.Performers[it.i].Role]
		}
	case itAnimal:
		if it.i < len(m.Animals) {
			return "Animal: " + m.Animals[it.i].Kind
		}
	case itLight:
		return fmt.Sprintf("Luz %d", it.i+1)
	case itFloor:
		if it.i < len(m.Floors) {
			f := m.Floors[it.i]
			return fmt.Sprintf("Suelo: %s %dx%d", f.Kind, f.W, f.H)
		}
	case itPool:
		if it.i < len(m.Pools) {
			return fmt.Sprintf("Agua %dx%d", m.Pools[it.i].W, m.Pools[it.i].H)
		}
	}
	return ""
}

// bigGround: floors that cover most of the scenario (the base ground) are not worth grabbing.
func bigGround(w, h int) bool { return w*h > screenW*screenH*35/100 }

// itemRect is the box drawn around a thing (pixels).
func (l *LevEd) itemRect(it lvItem) (x, y, w, h int, ok bool) {
	px, py, ok := l.itemPos(it)
	if !ok {
		return
	}
	m := &l.la.Meta
	switch it.kind {
	case itProp:
		p := m.Props[it.i]
		if imgs := l.ed.g.as.props[p.Kind]; len(imgs) > 0 {
			pi := imgs[p.Var%len(imgs)]
			return px - int(pi.ox), py - int(pi.oy), pi.img.Bounds().Dx(), pi.img.Bounds().Dy(), true
		}
		return px - 6, py - 12, 12, 12, true
	case itFloor:
		f := m.Floors[it.i]
		return f.X, f.Y, f.W, f.H, true
	case itPool:
		p := m.Pools[it.i]
		return p.X, p.Y, p.W, p.H, true
	}
	return px - 7, py - 7, 14, 14, true
}

// hitItem finds the thing under (x, y).
func (l *LevEd) hitItem(x, y int, mask int) lvItem {
	m := &l.la.Meta
	tol := math.Max(10, 12/l.cv.zf())
	d := func(px, py int) float64 { return math.Hypot(float64(px-x), float64(py-y)) }
	if mask&maskPoints != 0 {
		best, bd := lvItem{}, tol
		try := func(it lvItem, px, py int) {
			if dd := d(px, py); dd <= bd {
				best, bd = it, dd
			}
		}
		try(lvItem{kind: itStart}, m.Start.X, m.Start.Y)
		if m.Police == "car" {
			try(lvItem{kind: itCarFrom}, m.CarFrom.X, m.CarFrom.Y)
			try(lvItem{kind: itCarTo}, m.CarTo.X, m.CarTo.Y)
		}
		for i, p := range m.Pads {
			try(lvItem{kind: itPad, i: i}, p.X, p.Y)
		}
		for i, r := range m.Guards {
			for j, p := range r {
				try(lvItem{kind: itGuard, i: i, j: j}, p.X, p.Y)
			}
		}
		for i, p := range m.Performers {
			try(lvItem{kind: itPerf, i: i}, p.X, p.Y)
		}
		for i, p := range m.Animals {
			try(lvItem{kind: itAnimal, i: i}, p.X, p.Y)
		}
		for i, p := range m.Lights {
			try(lvItem{kind: itLight, i: i}, p.X, p.Y)
		}
		if best.kind != itNone {
			return best
		}
	}
	if mask&maskProps != 0 {
		best, bestY := lvItem{}, -1
		for i := range m.Props {
			bx, by, bw, bh, ok := l.itemRect(lvItem{kind: itProp, i: i})
			if ok && x >= bx && x < bx+bw && y >= by && y < by+bh && m.Props[i].Y > bestY {
				best, bestY = lvItem{kind: itProp, i: i}, m.Props[i].Y
			}
		}
		if best.kind != itNone {
			return best
		}
		nb, nd := -1, 20.0
		for i, p := range m.Props {
			if dd := d(p.X, p.Y); dd < nd {
				nb, nd = i, dd
			}
		}
		if nb >= 0 {
			return lvItem{kind: itProp, i: nb}
		}
	}
	if mask&maskGround != 0 {
		best, ba := lvItem{}, 1<<30
		for i, p := range m.Pools {
			if x >= p.X && x < p.X+p.W && y >= p.Y && y < p.Y+p.H && p.W*p.H < ba {
				best, ba = lvItem{kind: itPool, i: i}, p.W*p.H
			}
		}
		if best.kind != itNone {
			return best
		}
		for i, f := range m.Floors {
			if bigGround(f.W, f.H) {
				continue
			}
			if x >= f.X && x < f.X+f.W && y >= f.Y && y < f.Y+f.H && f.W*f.H < ba {
				best, ba = lvItem{kind: itFloor, i: i}, f.W*f.H
			}
		}
		return best
	}
	return lvItem{}
}

// listItems is every thing that can be moved with the Mover tool, for the list on the right.
func (l *LevEd) listItems(mask int) []lvItem {
	m := &l.la.Meta
	var out []lvItem
	if mask&maskGround != 0 {
		for i, p := range m.Pools {
			_ = p
			out = append(out, lvItem{kind: itPool, i: i})
		}
		for i, f := range m.Floors {
			if !bigGround(f.W, f.H) {
				out = append(out, lvItem{kind: itFloor, i: i})
			}
		}
	}
	if mask&maskPoints != 0 {
		out = append(out, lvItem{kind: itStart})
		if m.Police == "car" {
			out = append(out, lvItem{kind: itCarFrom}, lvItem{kind: itCarTo})
		}
		for i := range m.Pads {
			out = append(out, lvItem{kind: itPad, i: i})
		}
		for i, r := range m.Guards {
			for j := range r {
				out = append(out, lvItem{kind: itGuard, i: i, j: j})
			}
		}
		for i := range m.Performers {
			out = append(out, lvItem{kind: itPerf, i: i})
		}
		for i := range m.Animals {
			out = append(out, lvItem{kind: itAnimal, i: i})
		}
		for i := range m.Lights {
			out = append(out, lvItem{kind: itLight, i: i})
		}
	}
	if mask&maskProps != 0 {
		for i := range m.Props {
			out = append(out, lvItem{kind: itProp, i: i})
		}
	}
	return out
}

func (l *LevEd) selectItem(it lvItem) {
	l.mv.sel = it
	l.selInst = -1
	if it.kind == itProp {
		l.selInst = it.i
		if it.i < len(l.la.Meta.Props) {
			l.curVar = l.la.Meta.Props[it.i].Var
		}
	}
}

// ---- Mover ----------------------------------------------------------------------------

// updateMoveTool: hover, grab, drag and drop of single things.
func (l *LevEd) updateMoveTool(u *UI, x, y int, inside bool, mask int) {
	mv := &l.mv
	mv.hover = lvItem{}
	if inside && !mv.drag {
		mv.hover = l.hitItem(x, y, mask)
	}
	if inside && u.pressed && !mv.drag {
		if it := mv.hover; it.kind != itNone {
			l.selectItem(it)
			px, py, _ := l.itemPos(it)
			mv.drag, mv.moved = true, false
			mv.grab = [2]int{px - x, py - y}
			mv.snap = l.takeSnap(false)
		} else {
			l.selectItem(lvItem{})
		}
	}
	if mv.drag {
		if u.down {
			nx, ny := clampI(x+mv.grab[0], 0, screenW-1), clampI(y+mv.grab[1], 0, screenH-1)
			if ox, oy, ok := l.itemPos(mv.sel); ok && (ox != nx || oy != ny) {
				l.setItemPos(mv.sel, nx, ny)
				mv.moved = true
			}
		} else {
			mv.drag = false
			if mv.moved {
				l.finishItemMove(mv.snap)
			}
		}
	}
	if !u.down {
		mv.drag = false
	}
}

// finishItemMove commits a move: it is undoable, and moving ground or water of an original
// drags the cells that belong to it along and asks for the background to be painted again.
func (l *LevEd) finishItemMove(before *lvSnap) {
	l.pushMoveUndo(before)
	it := l.mv.sel
	if it.kind != itFloor && it.kind != itPool {
		return
	}
	var old LevelRect
	switch it.kind {
	case itFloor:
		if it.i >= len(before.meta.Floors) {
			return
		}
		f := before.meta.Floors[it.i]
		old = LevelRect{f.X, f.Y, f.W, f.H}
	case itPool:
		if it.i >= len(before.meta.Pools) {
			return
		}
		old = before.meta.Pools[it.i]
	}
	nx, ny, _ := l.itemPos(it)
	dx, dy := nx-old.X, ny-old.Y
	l.moveGroundCells(it, old, dx, dy)
	if it.kind == itFloor {
		kind := l.la.Meta.Floors[it.i].Kind
		if kind == "helipad" { // the landing points on it go with it
			for i, p := range l.la.Meta.Pads {
				if p.X >= old.X && p.X < old.X+old.W && p.Y >= old.Y && p.Y < old.Y+old.H {
					l.la.Meta.Pads[i] = LevelPoint{p.X + dx, p.Y + dy}
				}
			}
		}
	}
	if l.overrideKey != "" {
		l.mv.needDraw = true
	}
}

// moveGroundCells moves the water cells of a pool, or the dance cells of a dance floor, with it.
func (l *LevEd) moveGroundCells(it lvItem, old LevelRect, dx, dy int) {
	var grid [][]byte
	var ch, back byte
	switch {
	case it.kind == itPool:
		grid, ch, back = l.solid, '~', '.'
	case it.kind == itFloor && l.la.Meta.Floors[it.i].Kind == "dance":
		grid, ch, back = l.zone, 'd', 'r'
	default:
		return
	}
	if len(grid) == 0 {
		return
	}
	x0, y0, x1, y1 := old.X/logicCell, old.Y/logicCell, (old.X+old.W+logicCell-1)/logicCell, (old.Y+old.H+logicCell-1)/logicCell
	var cells [][2]int
	for cy := maxI(0, y0); cy < minI(len(grid), y1); cy++ {
		for cx := maxI(0, x0); cx < minI(len(grid[cy]), x1); cx++ {
			if grid[cy][cx] == ch {
				cells = append(cells, [2]int{cx, cy})
				grid[cy][cx] = back
			}
		}
	}
	for _, c := range cells {
		cx, cy := c[0]+dx/logicCell, c[1]+dy/logicCell
		if cy < 0 || cy >= len(grid) || cx < 0 || cx >= len(grid[cy]) {
			continue
		}
		if it.kind == itPool && l.solid[cy][cx] != '.' {
			continue // water does not replace a wall
		}
		if it.kind == itFloor && l.solid[cy][cx] != '.' && l.solid[cy][cx] != '~' {
			continue
		}
		grid[cy][cx] = ch
	}
}

// nudge moves the selected thing (or the selected zone) with the arrow keys.
func (l *LevEd) nudge(dx, dy int) {
	mv := &l.mv
	if l.mode == 2 && l.objTool == 2 && mv.zhas {
		l.zoneStep(dx, dy)
		return
	}
	if mv.sel.kind == itNone {
		return
	}
	px, py, ok := l.itemPos(mv.sel)
	if !ok {
		return
	}
	before := l.takeSnap(false)
	l.setItemPos(mv.sel, clampI(px+dx, 0, screenW-1), clampI(py+dy, 0, screenH-1))
	l.finishItemMove(before)
}

// deleteSelected removes the selected thing (or empties the selected zone).
func (l *LevEd) deleteSelected() {
	mv := &l.mv
	if l.mode == 2 && l.objTool == 2 && mv.zhas {
		l.zoneDelete()
		return
	}
	if mv.sel.kind == itNone {
		return
	}
	before := l.takeSnap(false)
	if l.delItem(mv.sel) {
		l.pushMoveUndo(before)
		if mv.sel.kind == itFloor || mv.sel.kind == itPool {
			l.mv.needDraw = l.overrideKey != ""
		}
		l.selectItem(lvItem{})
	}
}

// arrowRepeat is true on the press and then repeatedly while the key stays down.
func arrowRepeat(k ebiten.Key) bool {
	if inpututil.IsKeyJustPressed(k) {
		return true
	}
	d := inpututil.KeyPressDuration(k)
	return d > 24 && d%4 == 0
}

// moveKeys handles the keys of the move tools. Called when no text field has the focus.
func (l *LevEd) moveKeys(u *UI) {
	if l.mode != 2 && l.mode != 3 {
		return
	}
	if l.mode == 3 && l.ptTool != ptMoveTool {
		return
	}
	if l.mode == 2 && l.objTool == 0 {
		return
	}
	step := logicCell
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		step = 1
	}
	if (l.mode == 2 && l.objTool == 2) && l.mv.zhas {
		step = logicCell // zones stay on the logic grid
	}
	for _, k := range []struct {
		key    ebiten.Key
		dx, dy int
	}{{ebiten.KeyArrowLeft, -1, 0}, {ebiten.KeyArrowRight, 1, 0}, {ebiten.KeyArrowUp, 0, -1}, {ebiten.KeyArrowDown, 0, 1}} {
		if arrowRepeat(k.key) {
			l.nudge(k.dx*step, k.dy*step)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		l.deleteSelected()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if l.mv.drag && l.mv.snap != nil {
			l.restoreSnap(l.mv.snap)
			l.mv.drag = false
		}
		l.mv.zhas, l.mv.zmode = false, 0
		l.selectItem(lvItem{})
	}
}

// ---- Zona ---------------------------------------------------------------------------------

func (l *LevEd) cells() (cw, ch int) {
	if len(l.solid) == 0 {
		return 0, 0
	}
	return len(l.solid[0]), len(l.solid)
}

func snapDown(v int) int { return v / logicCell * logicCell }
func snapUp(v int) int   { return (v + logicCell - 1) / logicCell * logicCell }

func (l *LevEd) captureZone(r [4]int) zoneContent {
	m := &l.la.Meta
	zc := zoneContent{x0: r[0], y0: r[1], x1: r[2], y1: r[3]}
	cx0, cy0, cx1, cy1 := r[0]/logicCell, r[1]/logicCell, r[2]/logicCell, r[3]/logicCell
	for cy := cy0; cy < cy1 && cy < len(l.solid); cy++ {
		zc.solid = append(zc.solid, append([]byte(nil), l.solid[cy][cx0:minI(cx1, len(l.solid[cy]))]...))
		zc.zone = append(zc.zone, append([]byte(nil), l.zone[cy][cx0:minI(cx1, len(l.zone[cy]))]...))
	}
	if l.overrideKey == "" { // the picture of an original is not part of the correction
		w := r[2] - r[0]
		for y := r[1]; y < r[3] && y < screenH; y++ {
			zc.bg = append(zc.bg, l.la.BG.p[(y*screenW+r[0])*4:(y*screenW+r[0]+minI(w, screenW-r[0]))*4]...)
		}
	}
	in := func(x, y int) bool { return x >= r[0] && x < r[2] && y >= r[1] && y < r[3] }
	for i, p := range m.Props {
		if in(p.X, p.Y) {
			zc.props = append(zc.props, i)
		}
	}
	add := func(it lvItem) {
		if x, y, ok := l.itemPos(it); ok && in(x, y) {
			zc.pts = append(zc.pts, it)
		}
	}
	add(lvItem{kind: itStart})
	if m.Police == "car" {
		add(lvItem{kind: itCarFrom})
		add(lvItem{kind: itCarTo})
	}
	for i := range m.Pads {
		add(lvItem{kind: itPad, i: i})
	}
	for i, rt := range m.Guards {
		for j := range rt {
			add(lvItem{kind: itGuard, i: i, j: j})
		}
	}
	for i := range m.Performers {
		add(lvItem{kind: itPerf, i: i})
	}
	for i := range m.Animals {
		add(lvItem{kind: itAnimal, i: i})
	}
	for i := range m.Lights {
		add(lvItem{kind: itLight, i: i})
	}
	if l.overrideKey != "" {
		for i, f := range m.Floors {
			if !bigGround(f.W, f.H) && f.X >= r[0] && f.Y >= r[1] && f.X+f.W <= r[2] && f.Y+f.H <= r[3] {
				zc.grounds = append(zc.grounds, lvItem{kind: itFloor, i: i})
			}
		}
		for i, p := range m.Pools {
			if p.X >= r[0] && p.Y >= r[1] && p.X+p.W <= r[2] && p.Y+p.H <= r[3] {
				zc.grounds = append(zc.grounds, lvItem{kind: itPool, i: i})
			}
		}
	}
	return zc
}

// ringColour is the colour most common around a rectangle of the picture: what an emptied
// zone is filled with.
func ringColour(bg []byte, r [4]int) (uint8, uint8, uint8) {
	type acc struct{ r, g, b, n int }
	buckets := map[int]*acc{}
	add := func(x, y int) {
		if x < 0 || y < 0 || x >= screenW || y >= screenH {
			return
		}
		i := (y*screenW + x) * 4
		k := int(bg[i]>>4)<<8 | int(bg[i+1]>>4)<<4 | int(bg[i+2]>>4)
		a := buckets[k]
		if a == nil {
			a = &acc{}
			buckets[k] = a
		}
		a.r += int(bg[i])
		a.g += int(bg[i+1])
		a.b += int(bg[i+2])
		a.n++
	}
	for x := r[0] - 1; x <= r[2]; x++ {
		add(x, r[1]-1)
		add(x, r[3])
	}
	for y := r[1]; y < r[3]; y++ {
		add(r[0]-1, y)
		add(r[2], y)
	}
	var best *acc
	for _, a := range buckets {
		if best == nil || a.n > best.n {
			best = a
		}
	}
	if best == nil {
		return 10, 9, 18
	}
	return uint8(best.r / best.n), uint8(best.g / best.n), uint8(best.b / best.n)
}

// emptyZone clears the zone r: background to the surrounding colour, no cells, nothing in it.
func (l *LevEd) clearRegion(r [4]int, bg []byte) {
	if bg != nil {
		cr, cg, cb := ringColour(bg, r)
		for y := maxI(0, r[1]); y < minI(screenH, r[3]); y++ {
			for x := maxI(0, r[0]); x < minI(screenW, r[2]); x++ {
				i := (y*screenW + x) * 4
				l.la.BG.p[i], l.la.BG.p[i+1], l.la.BG.p[i+2], l.la.BG.p[i+3] = cr, cg, cb, 255
			}
		}
	}
	for cy := r[1] / logicCell; cy < r[3]/logicCell && cy < len(l.solid); cy++ {
		for cx := r[0] / logicCell; cx < r[2]/logicCell && cx < len(l.solid[cy]); cx++ {
			l.solid[cy][cx], l.zone[cy][cx] = '.', '.'
		}
	}
}

// applyZone rebuilds the scenario from the snapshot s0 with the zone moved (or copied) by (dx, dy).
func (l *LevEd) applyZone(s0 *lvSnap, zc *zoneContent, dx, dy int, cp bool) {
	l.restoreSnap(s0)
	m := &l.la.Meta
	r := [4]int{zc.x0, zc.y0, zc.x1, zc.y1}
	custom := l.overrideKey == "" && s0.bg != nil
	if !cp {
		l.clearRegion(r, s0.bg)
	}
	// cells (the block replaces what is under it)
	for j, row := range zc.solid {
		cy := zc.y0/logicCell + j + dy/logicCell
		if cy < 0 || cy >= len(l.solid) {
			continue
		}
		for i := range row {
			cx := zc.x0/logicCell + i + dx/logicCell
			if cx < 0 || cx >= len(l.solid[cy]) {
				continue
			}
			l.solid[cy][cx], l.zone[cy][cx] = zc.solid[j][i], zc.zone[j][i]
		}
	}
	// background pixels
	if custom && zc.bg != nil {
		w := zc.x1 - zc.x0
		for y := 0; y < zc.y1-zc.y0; y++ {
			ty := zc.y0 + y + dy
			if ty < 0 || ty >= screenH {
				continue
			}
			for x := 0; x < w; x++ {
				tx := zc.x0 + x + dx
				if tx < 0 || tx >= screenW {
					continue
				}
				si, di := (y*w+x)*4, (ty*screenW+tx)*4
				copy(l.la.BG.p[di:di+4], zc.bg[si:si+4])
			}
		}
	}
	// objects, points, ground
	for _, i := range zc.props {
		p := m.Props[i]
		np := LevelProp{Kind: p.Kind, X: clampI(p.X+dx, 0, screenW-1), Y: clampI(p.Y+dy, 0, screenH-1), Var: p.Var}
		if cp {
			m.Props = append(m.Props, np)
		} else {
			m.Props[i] = np
		}
	}
	if !cp {
		for _, it := range zc.pts {
			x, y, _ := l.itemPos(it)
			l.setItemPos(it, clampI(x+dx, 0, screenW-1), clampI(y+dy, 0, screenH-1))
		}
	} else { // copies of the free points: lights, animals, musicians
		for _, it := range zc.pts {
			x, y, _ := l.itemPos(it)
			nx, ny := clampI(x+dx, 0, screenW-1), clampI(y+dy, 0, screenH-1)
			switch it.kind {
			case itLight:
				c := m.Lights[it.i]
				c.X, c.Y = nx, ny
				m.Lights = append(m.Lights, c)
			case itAnimal:
				c := m.Animals[it.i]
				c.X, c.Y = nx, ny
				m.Animals = append(m.Animals, c)
			case itPerf:
				c := m.Performers[it.i]
				c.X, c.Y = nx, ny
				m.Performers = append(m.Performers, c)
			}
		}
	}
	for _, it := range zc.grounds {
		switch it.kind {
		case itFloor:
			f := m.Floors[it.i]
			f.X, f.Y = f.X+dx, f.Y+dy
			if cp {
				m.Floors = append(m.Floors, f)
			} else {
				m.Floors[it.i] = f
			}
		case itPool:
			p := m.Pools[it.i]
			p.X, p.Y = p.X+dx, p.Y+dy
			if cp {
				m.Pools = append(m.Pools, p)
			} else {
				m.Pools[it.i] = p
			}
		}
	}
	l.cv.dirty = true
}

// updateZone: select a rectangle, then drag it to move it (Alt: copy).
func (l *LevEd) updateZone(u *UI, x, y int, inside bool) {
	mv := &l.mv
	x, y = clampI(x, 0, screenW), clampI(y, 0, screenH)
	inRect := func() bool {
		return mv.zhas && x >= mv.zrect[0] && x < mv.zrect[2] && y >= mv.zrect[1] && y < mv.zrect[3]
	}
	if inside && u.pressed && mv.zmode == 0 {
		if inRect() {
			mv.zmode = 2
			mv.zgrab, mv.zd = [2]int{x, y}, [2]int{0, 0}
			mv.zcopyNow = mv.zcopy || ebiten.IsKeyPressed(ebiten.KeyAlt)
			mv.zsnap = l.takeSnap(l.overrideKey == "")
			mv.zc = l.captureZone(mv.zrect)
		} else {
			mv.zmode, mv.zhas = 1, false
			mv.zstart = [2]int{x, y}
		}
	}
	switch mv.zmode {
	case 1:
		a := mv.zstart
		mv.zrect = [4]int{snapDown(minI(a[0], x)), snapDown(minI(a[1], y)), snapUp(maxI(a[0], x)), snapUp(maxI(a[1], y))}
		mv.zrect[2], mv.zrect[3] = minI(mv.zrect[2], screenW/logicCell*logicCell), minI(mv.zrect[3], screenH/logicCell*logicCell)
		if !u.down {
			mv.zmode = 0
			mv.zhas = mv.zrect[2]-mv.zrect[0] >= logicCell && mv.zrect[3]-mv.zrect[1] >= logicCell
		}
	case 2:
		if u.down {
			dx, dy := snapDown(x-mv.zgrab[0]+logicCell/2*sgn(x-mv.zgrab[0])), snapDown(y-mv.zgrab[1]+logicCell/2*sgn(y-mv.zgrab[1]))
			dx, dy = clampZone(mv.zrect, dx, dy)
			if dx != mv.zd[0] || dy != mv.zd[1] {
				mv.zd = [2]int{dx, dy}
				l.applyZone(mv.zsnap, &mv.zc, dx, dy, mv.zcopyNow)
			}
		} else {
			mv.zmode = 0
			if mv.zd != [2]int{} {
				l.pushMoveUndo(mv.zsnap)
				mv.zrect = [4]int{mv.zrect[0] + mv.zd[0], mv.zrect[1] + mv.zd[1], mv.zrect[2] + mv.zd[0], mv.zrect[3] + mv.zd[1]}
				if l.overrideKey != "" {
					mv.needDraw = true
				}
			}
		}
	}
}

func sgn(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// clampZone keeps the zone inside the picture.
func clampZone(r [4]int, dx, dy int) (int, int) {
	dx = clampI(dx, -r[0], screenW/logicCell*logicCell-r[2])
	dy = clampI(dy, -r[1], screenH/logicCell*logicCell-r[3])
	return dx, dy
}

// zoneStep moves the selected zone by one step of the arrow keys.
func (l *LevEd) zoneStep(dx, dy int) {
	mv := &l.mv
	dx, dy = clampZone(mv.zrect, dx, dy)
	if dx == 0 && dy == 0 {
		return
	}
	s0 := l.takeSnap(l.overrideKey == "")
	zc := l.captureZone(mv.zrect)
	l.applyZone(s0, &zc, dx, dy, false)
	l.pushMoveUndo(s0)
	mv.zrect = [4]int{mv.zrect[0] + dx, mv.zrect[1] + dy, mv.zrect[2] + dx, mv.zrect[3] + dy}
	if l.overrideKey != "" {
		mv.needDraw = true
	}
}

// zoneDelete empties the selected zone, taking everything in it away.
func (l *LevEd) zoneDelete() {
	mv := &l.mv
	s0 := l.takeSnap(l.overrideKey == "")
	zc := l.captureZone(mv.zrect)
	l.clearRegion(mv.zrect, s0.bg)
	m := &l.la.Meta
	// remove from the back so the indices stay valid
	for k := len(zc.props) - 1; k >= 0; k-- {
		m.Props = append(m.Props[:zc.props[k]], m.Props[zc.props[k]+1:]...)
	}
	for k := len(zc.pts) - 1; k >= 0; k-- {
		l.delItem(zc.pts[k])
	}
	for k := len(zc.grounds) - 1; k >= 0; k-- {
		l.delItem(zc.grounds[k])
	}
	l.cv.dirty = true
	l.pushMoveUndo(s0)
	mv.zhas = false
	if l.overrideKey != "" {
		mv.needDraw = true
	}
}

// ---- feedback -------------------------------------------------------------------------------

type lvHL struct {
	x, y, w, h int
	col        color.RGBA
	label      string
}

// highlights are the boxes drawn over the picture: the thing under the mouse, the selected
// one and the zone.
func (l *LevEd) highlights() []lvHL {
	var out []lvHL
	if l.mode != 2 && l.mode != 3 {
		return nil
	}
	m := &l.la.Meta
	if l.mode == 3 && l.ptTool != ptMoveTool || l.mode == 2 && l.objTool == 0 {
		return nil
	}
	add := func(it lvItem, c color.RGBA) {
		if x, y, w, h, ok := l.itemRect(it); ok {
			out = append(out, lvHL{x, y, w, h, c, it.label(m)})
		}
	}
	if l.mv.hover.kind != itNone && l.mv.hover != l.mv.sel {
		add(l.mv.hover, color.RGBA{255, 255, 255, 230})
	}
	if l.mv.sel.kind != itNone {
		add(l.mv.sel, color.RGBA{255, 179, 71, 255})
	}
	if l.mode == 2 && l.objTool == 2 && (l.mv.zhas || l.mv.zmode == 1) {
		r := l.mv.zrect
		lab := fmt.Sprintf("%d x %d", r[2]-r[0], r[3]-r[1])
		if l.mv.zcopy {
			lab += " · copiar"
		}
		out = append(out, lvHL{r[0], r[1], r[2] - r[0], r[3] - r[1], color.RGBA{255, 179, 71, 255}, lab})
	}
	return out
}

// ptMoveTool is the index of the "Mover" tool in the Puntos tab.
const ptMoveTool = 9
