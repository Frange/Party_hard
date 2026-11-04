package main

import (
	"testing"
)

// first campaign scenario that has water, and one with a helipad floor
func levelWith(t *testing.T, want func(d *LevelDef) bool) (int, *LevelDef) {
	t.Helper()
	for i := range campaignLevels {
		if d := resolveCampaign(campaignLevels[i]); d != nil && want(d) {
			return i, d
		}
	}
	t.Skip("no scenario with that")
	return -1, nil
}

func hasKind(d *LevelDef, kind string) bool {
	for _, f := range d.Floors {
		if f.Kind == kind {
			return true
		}
	}
	return false
}

// moving a pool of an original with the editor's tools moves the water in the game
// (the drawn background follows because the generator paints from the pool list)
func TestMovePoolOfOriginal(t *testing.T) {
	idx, def := levelWith(t, func(d *LevelDef) bool { return len(d.Pools) > 0 && d.Swim })
	ed := testEditor()
	la := levelAssetFromDef(def, campaignKey(idx))
	la.Meta.Override = campaignKey(idx)
	ed.lv.setLevel(la)
	l := ed.lv
	old := la.Meta.Pools[0]
	l.mv.sel = lvItem{kind: itPool, i: 0}
	before := l.takeSnap(false)
	l.setItemPos(l.mv.sel, old.X+60, old.Y+40)
	l.finishItemMove(before)
	if !l.mv.needDraw {
		t.Fatal("the background of an original must be painted again after moving water")
	}
	l.syncMeta()

	d2 := resolveCampaign(campaignLevels[idx])
	applyLevelOverride(d2, l.la)
	if len(d2.Pools) != len(def.Pools) {
		t.Fatalf("pools: %d, want %d", len(d2.Pools), len(def.Pools))
	}
	if d2.Pools[0].X != float64(old.X+60) || d2.Pools[0].Y != float64(old.Y+40) {
		t.Fatalf("pool did not move: %+v", d2.Pools[0])
	}
	if !d2.Swim {
		t.Fatal("Swim was lost")
	}
	lv := buildLevel(d2)
	inNew := Vec{float64(old.X + 60 + old.W/2), float64(old.Y + 40 + old.H/2)}
	if !lv.InPool(inNew) {
		t.Fatal("the new place is not water")
	}
	if len(d2.Pools) == 1 && lv.InPool(Vec{float64(old.X + 5), float64(old.Y + 5)}) && !pointInRect(d2.Pools[0], float64(old.X+5), float64(old.Y+5)) {
		t.Fatal("the old place is still water")
	}
	// the water cells followed: the logic grid has water at the new place and not at the old corner
	cx, cy := (old.X+60+old.W/2)/logicCell, (old.Y+40+old.H/2)/logicCell
	if l.solid[cy][cx] != '~' {
		t.Fatalf("water cell missing at the new place: %q", l.solid[cy][cx])
	}
	// the same override through builtinLevels
	prev := levelOverrides
	levelOverrides = map[string]*LevelAsset{campaignKey(idx): l.la}
	defer func() { levelOverrides = prev }()
	if got := builtinLevels(); len(got) <= idx || len(got[idx].Pools) == 0 || got[idx].Pools[0].X != float64(old.X+60) {
		t.Fatal("builtinLevels did not apply the override")
	}
}

func pointInRect(r Rect, x, y float64) bool { return r.Has(x, y) }

// the helipad floor and its landing points move together, and survive the file
func TestMoveHelipadOfOriginal(t *testing.T) {
	idx, def := levelWith(t, func(d *LevelDef) bool { return hasKind(d, "helipad") && len(d.Pads) > 0 })
	ed := testEditor()
	la := levelAssetFromDef(def, campaignKey(idx))
	la.Meta.Override = campaignKey(idx)
	ed.lv.setLevel(la)
	l := ed.lv
	fi := -1
	for i, f := range la.Meta.Floors {
		if f.Kind == "helipad" {
			fi = i
			break
		}
	}
	oldF, oldPad := la.Meta.Floors[fi], la.Meta.Pads[0]
	padInside := oldPad.X >= oldF.X && oldPad.X < oldF.X+oldF.W && oldPad.Y >= oldF.Y && oldPad.Y < oldF.Y+oldF.H
	l.mv.sel = lvItem{kind: itFloor, i: fi}
	before := l.takeSnap(false)
	l.setItemPos(l.mv.sel, oldF.X+48, oldF.Y+32)
	l.finishItemMove(before)
	l.syncMeta()

	d2 := resolveCampaign(campaignLevels[idx])
	applyLevelOverride(d2, l.la)
	if got := d2.Floors[fi]; got.R.X != float64(oldF.X+48) || got.R.Y != float64(oldF.Y+32) || got.Kind != "helipad" {
		t.Fatalf("floor did not move: %+v", got)
	}
	if padInside && (d2.Pads[0].X != float64(oldPad.X+48) || d2.Pads[0].Y != float64(oldPad.Y+32)) {
		t.Fatalf("landing point did not follow: %+v", d2.Pads[0])
	}
	if len(d2.Floors) != len(def.Floors) {
		t.Fatal("floors were lost")
	}
	buildLevel(d2) // must still build
	// undo brings everything back
	if !l.moveUndo() {
		t.Fatal("nothing to undo")
	}
	if l.la.Meta.Floors[fi].X != oldF.X || l.la.Meta.Pads[0] != oldPad {
		t.Fatal("undo did not restore the helipad")
	}
}

// the zone tool carries pixels, cells and objects along, and undo gives them back
func TestMoveZone(t *testing.T) {
	ed := testEditor()
	l := ed.lv
	l.setLevel(newLevelAsset("zona"))
	// a red block of picture with a wall cell and an object in it
	for y := 40; y < 60; y++ {
		for x := 40; x < 60; x++ {
			i := (y*screenW + x) * 4
			l.la.BG.p[i], l.la.BG.p[i+3] = 200, 255
		}
	}
	l.solid[11][11] = '#'
	l.zone[11][11] = 'r'
	l.la.Meta.Props = []LevelProp{{Kind: "lamp", X: 50, Y: 50}, {Kind: "lamp", X: 300, Y: 300}}
	l.mv.zrect = [4]int{40, 40, 60, 60}
	l.mv.zhas = true

	s0 := l.takeSnap(true)
	zc := l.captureZone(l.mv.zrect)
	if len(zc.props) != 1 {
		t.Fatalf("zone holds %d objects, want 1", len(zc.props))
	}
	l.applyZone(s0, &zc, 100, 60, false)
	l.pushMoveUndo(s0)
	at := func(x, y int) byte { return l.la.BG.p[(y*screenW+x)*4] }
	if at(150, 100) != 200 || at(50, 50) == 200 {
		t.Fatalf("pixels did not move: new %d old %d", at(150, 100), at(50, 50))
	}
	if l.solid[11+15][11+25] != '#' || l.solid[11][11] != '.' || l.zone[11+15][11+25] != 'r' {
		t.Fatal("cells did not move")
	}
	if l.la.Meta.Props[0].X != 150 || l.la.Meta.Props[0].Y != 110 || l.la.Meta.Props[1].X != 300 {
		t.Fatalf("objects: %+v", l.la.Meta.Props)
	}
	// copy leaves the original in place
	s1 := l.takeSnap(true)
	zc2 := l.captureZone([4]int{140, 90, 160, 120})
	l.applyZone(s1, &zc2, 40, 0, true)
	if at(150, 100) != 200 || at(190, 100) != 200 || len(l.la.Meta.Props) != 3 {
		t.Fatal("copy must keep the original and add a duplicate")
	}
	l.restoreSnap(s1)
	if len(l.la.Meta.Props) != 2 {
		t.Fatal("restore of the copy failed")
	}
	// undo of the move
	if !l.moveUndo() {
		t.Fatal("no undo")
	}
	if at(50, 50) != 200 || at(150, 100) == 200 || l.solid[11][11] != '#' || l.la.Meta.Props[0].X != 50 {
		t.Fatal("undo did not restore the zone")
	}
}

// the arrow-key nudge and delete work on single things, and an untouched grab changes nothing
func TestNudgeAndDelete(t *testing.T) {
	ed := testEditor()
	l := ed.lv
	l.setLevel(newLevelAsset("pts"))
	l.la.Meta.Pads = []LevelPoint{{100, 100}}
	l.la.Meta.Lights = []LevelLight{{X: 10, Y: 10, R: 30}}
	l.mode, l.ptTool = 3, ptMoveTool
	l.selectItem(lvItem{kind: itPad, i: 0})
	l.nudge(4, -4)
	if l.la.Meta.Pads[0] != (LevelPoint{104, 96}) {
		t.Fatalf("pad: %+v", l.la.Meta.Pads[0])
	}
	l.selectItem(lvItem{kind: itLight, i: 0})
	l.deleteSelected()
	if len(l.la.Meta.Lights) != 0 {
		t.Fatal("light not deleted")
	}
	l.moveUndo()
	if len(l.la.Meta.Lights) != 1 {
		t.Fatal("undo did not bring the light back")
	}
	l.selectItem(lvItem{kind: itStart})
	l.deleteSelected() // the start cannot be deleted
	if l.mv.sel.kind != itStart {
		t.Fatal("the start must stay")
	}
}
