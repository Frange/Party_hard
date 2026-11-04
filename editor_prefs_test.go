package main

import (
	"os"
	"path/filepath"
	"testing"
)

func testEditor() *Editor {
	ed := &Editor{pt: newPainter(), root: "assets"}
	ed.ch = newCharEd(ed)
	ed.ob = newObjEd(ed)
	ed.lv = newLevEd(ed)
	return ed
}

// choices written to the file come back the same in a new editor
func TestEditorPrefsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "editor_prefs.json")
	a := testEditor()
	cvGrid, cvBG = false, 3
	defer func() { cvGrid, cvBG = true, 0 }()
	a.tab = 1
	a.ch.typeIdx, a.ch.subIdx, a.ch.mode, a.ch.prevAnim, a.ch.prevDir = 3, 0, 1, AnimDance2, 1
	a.ob.view, a.ob.typeIdx, a.ob.mode, a.ob.ltool = 0, 2, 1, 2
	a.pt.tool, a.pt.col, a.pt.brush, a.pt.mirror = ToolEraser, rgb(0x112233), 2, true
	a.pt.recent = append(a.pt.recent, rgb(0xaa0000), rgb(0x00bb00))
	a.lv.mode, a.lv.prefZoom, a.lv.showLogic, a.lv.bsize = 2, 1.5, true, 3
	a.lv.objTool, a.lv.ptTool = 2, ptMoveTool
	if err := writePrefs(path, a.currentPrefs()); err != nil {
		t.Fatal(err)
	}
	cvGrid, cvBG = true, 0
	b := testEditor()
	p, ok := readPrefs(path)
	if !ok {
		t.Fatal("not read back")
	}
	b.applyPrefs(p)
	if cvGrid || cvBG != 3 || b.tab != 1 || b.ch.typeIdx != 3 || b.ch.mode != 1 || b.ch.prevAnim != AnimDance2 || b.ch.prevDir != 1 {
		t.Fatalf("grid/background/tab/character not restored: %+v", p)
	}
	if b.ob.typeIdx != 2 || b.ob.mode != 1 || b.ob.ltool != 2 || b.pt.tool != ToolEraser || b.pt.col != rgb(0x112233) || !b.pt.mirror || b.pt.brush != 2 {
		t.Fatalf("object/painter not restored: %+v", p)
	}
	if len(b.pt.recent) != 2 || b.pt.recent[0] != rgb(0xaa0000) || b.lv.mode != 2 || b.lv.prefZoom != 1.5 || !b.lv.showLogic || b.lv.bsize != 3 || b.lv.objTool != 2 || b.lv.ptTool != ptMoveTool {
		t.Fatalf("recent colours/level not restored: %+v", p)
	}
}

// a missing or damaged file means the defaults, and absurd values are ignored
func TestEditorPrefsTolerant(t *testing.T) {
	dir := t.TempDir()
	if p, ok := readPrefs(filepath.Join(dir, "nothing.json")); ok || !p.Grid {
		t.Fatal("missing file must give the defaults")
	}
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{not json"), 0o644)
	if p, ok := readPrefs(bad); ok || !p.Grid {
		t.Fatal("corrupt file must give the defaults")
	}
	wild := filepath.Join(dir, "wild.json")
	os.WriteFile(wild, []byte(`{"tab":99,"background":-4,"charType":500,"charSub":77,"tool":123,"color":"zzz","recent":["nope","FF0000"],"previewAnim":9999}`), 0o644)
	p, _ := readPrefs(wild)
	e := testEditor()
	e.applyPrefs(p)
	defer func() { cvGrid, cvBG = true, 0 }()
	if e.tab > 2 || cvBG != 0 || e.ch.typeIdx >= len(charTypes) || e.pt.tool >= ToolCount || len(e.pt.recent) != 1 {
		t.Fatalf("wild values were not clamped: tab %d bg %d type %d tool %d recent %d", e.tab, cvBG, e.ch.typeIdx, e.pt.tool, len(e.pt.recent))
	}
}
