package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// Editor debug mode: open one of the game's own scenarios, correct it and save
// it back as an override (see level_override.go).
// ---------------------------------------------------------------------------

// originalTitle is the name shown for the i-th campaign scenario.
func originalTitle(i int) string {
	e := templateEntries()
	if i < len(e) {
		return fmt.Sprintf("%02d  %s", i+1, e[i].title)
	}
	return fmt.Sprintf("%02d", i+1)
}

func (l *LevEd) overridePath(key string) string {
	return filepath.Join(backgroundsDir(l.ed.root), key)
}

// loadOriginal opens the i-th campaign scenario: the saved correction if there is one,
// otherwise the scenario as the game generates it.
func (l *LevEd) loadOriginal(i int) {
	key := campaignKey(i)
	if key == "" {
		return
	}
	if dirExists(l.overridePath(key)) {
		if la, err := loadLevelAsset(l.overridePath(key)); err == nil {
			la.Meta.Override = key
			l.setLevel(la)
			l.name = key
			l.sel = -1
			l.ed.say("Con tus correcciones: " + originalTitle(i))
			return
		}
	}
	l.openPristine(i)
}

// openPristine opens the scenario as generated, ignoring any saved correction.
func (l *LevEd) openPristine(i int) {
	key := campaignKey(i)
	def := resolveCampaign(campaignLevels[i])
	if def == nil {
		l.ed.say("No se pudo crear el escenario original")
		return
	}
	pristine := cloneDef(def)
	la, err := exportLevelDef(l.ed.g, def, key)
	if err != nil {
		l.ed.say("No se pudo cargar el original: " + err.Error())
		return
	}
	la.Meta.Override = key
	l.setLevel(la)
	l.mv.origDef = pristine
	l.name = key
	l.sel = -1
	l.ed.say("Original: " + originalTitle(i))
}

// restoreOriginal deletes the saved correction of the open scenario.
func (l *LevEd) restoreOriginal() {
	key := l.overrideKey
	if key == "" {
		return
	}
	idx := -1
	for i := range campaignLevels {
		if campaignKey(i) == key {
			idx = i
		}
	}
	if err := os.RemoveAll(l.overridePath(key)); err != nil {
		l.ed.say("No se pudo borrar la corrección: " + err.Error())
		return
	}
	if idx >= 0 {
		l.openPristine(idx)
	}
	l.ed.say("Original restaurado: se ha borrado tu corrección")
}

// debugBlock draws the debug part of the scenario list (below the usual buttons).
func (l *LevEd) debugBlock(u *UI, x, y, w int) {
	u.heading(x+4, y, w-8, "DEBUG · ORIGINALES")
	y += 22
	var items []segItem
	for i := range campaignLevels {
		t := originalTitle(i)
		if dirExists(l.overridePath(campaignKey(i))) {
			t += "  ★"
		}
		items = append(items, segItem{"world", t})
	}
	if i := u.menuBtn(x, y, w, 34, "world", "Escenarios del juego", "lvorig", items, "Abre uno de los 21 escenarios del juego para corregirlo (★: ya tiene correcciones)"); i >= 0 {
		l.loadOriginal(i)
	}
	y += 40
	if l.overrideKey != "" {
		u.text(x+4, y, "Editando:", uiAccent)
		u.text(x+70, y, l.overrideKey, uiText)
		y += 18
		if u.iconBtn(x, y, w, 30, "undo", "Restaurar original", false, "Borra tus correcciones de este escenario") {
			l.restoreOriginal()
		}
		y += 36
		u.text(x+4, y, "Cuentan: objetos, puntos, muros y zonas.", uiDim)
		u.text(x+4, y+14, "El dibujo del fondo no se aplica.", uiDim)
	}
}

// redrawOriginal paints the background of the open original again from its (moved) ground,
// water and objects, so what was dragged is drawn where it now is.
func (l *LevEd) redrawOriginal() {
	if l.overrideKey == "" || l.ed.g == nil {
		return
	}
	idx := -1
	for i := range campaignLevels {
		if campaignKey(i) == l.overrideKey {
			idx = i
		}
	}
	if idx < 0 {
		return
	}
	if l.mv.origDef == nil {
		def := resolveCampaign(campaignLevels[idx])
		if def == nil {
			return
		}
		l.mv.origDef = cloneDef(def)
	}
	l.syncMeta()
	c := cloneDef(l.mv.origDef)
	applyLevelOverride(c, l.la)
	if c.PaintBG != nil || c.BGImg != nil {
		l.ed.say("Este escenario se dibuja de otra forma: el fondo no se repinta")
		return
	}
	copy(l.la.BG.p, renderDefBG(l.ed.g, c))
	l.cv.dirty = true
}

// demo is a debug helper for screenshots: it moves the first pool or helipad like a drag would.
func (l *LevEd) demo(what string) {
	m := &l.la.Meta
	var it lvItem
	switch what {
	case "zone": // select a rectangle around the first pool and push it 80 px to the right
		if len(m.Pools) > 0 {
			p := m.Pools[0]
			l.objTool = 2
			l.mv.zrect = [4]int{snapDown(p.X - 8), snapDown(p.Y - 8), snapUp(p.X + p.W + 8), snapUp(p.Y + p.H + 8)}
			l.mv.zhas = true
			l.zoneStep(80, 0)
			l.redrawOriginal()
			l.mv.needDraw = false
		}
		return
	case "pool":
		if len(m.Pools) > 0 {
			it = lvItem{kind: itPool}
		}
	case "helipad":
		for i, f := range m.Floors {
			if f.Kind == "helipad" {
				it = lvItem{kind: itFloor, i: i}
				break
			}
		}
	}
	if it.kind == itNone {
		return
	}
	l.mv.sel = it
	x, y, _ := l.itemPos(it)
	before := l.takeSnap(false)
	l.setItemPos(it, x+60, y+40)
	l.finishItemMove(before)
	l.redrawOriginal()
	l.mv.needDraw = false
}
