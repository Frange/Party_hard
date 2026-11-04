package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// The editor remembers its last choices between sessions (grid, transparent
// background colour, tab, kind of character or object, tool, colour, view...)
// in assets/editor_prefs.json, next to progress.json and language.json. It is
// written only after something has changed and stayed changed for a moment,
// never every frame. Search texts are deliberately not remembered. A missing
// or damaged file just means the defaults.
// ---------------------------------------------------------------------------

type editorPrefs struct {
	Grid bool `json:"grid"`
	BG   int  `json:"background"`
	Tab  int  `json:"tab"`

	CharType int `json:"charType"`
	CharSub  int `json:"charSub"`
	CharMode int `json:"charMode"` // 0 DIBUJO, 1 PARTES
	PrevAnim int `json:"previewAnim"`
	PrevDir  int `json:"previewDir"`

	ObjView int `json:"objView"` // 0 objects, 1 animals
	ObjType int `json:"objType"`
	ObjMode int `json:"objMode"` // 0 PINTAR, 1 LÓGICA
	ObjTool int `json:"objLogicTool"`

	Tool   int      `json:"tool"`
	Color  string   `json:"color"`
	Recent []string `json:"recent"`
	Brush  int      `json:"brush"`
	Mirror bool     `json:"mirror"`

	LevMode    int     `json:"levelMode"`
	LevZoom    float64 `json:"levelZoom"`
	LevLogic   bool    `json:"levelShowWalls"`
	LevObjs    bool    `json:"levelShowObjects"`
	LevPoints  bool    `json:"levelShowPoints"`
	LevBrush   int     `json:"levelBrush"`
	LevObjTool int     `json:"levelObjectTool"` // 0 place, 1 move, 2 zone
	LevPtTool  int     `json:"levelPointTool"`
}

func prefsPath() string {
	root := assetsRoot(false)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "editor_prefs.json")
}

func (ed *Editor) currentPrefs() editorPrefs {
	p := editorPrefs{
		Grid: cvGrid, BG: cvBG, Tab: ed.tab,
		CharType: ed.ch.typeIdx, CharSub: ed.ch.subIdx, CharMode: ed.ch.mode, PrevAnim: ed.ch.prevAnim, PrevDir: ed.ch.prevDir,
		ObjView: ed.ob.view, ObjType: ed.ob.typeIdx, ObjMode: ed.ob.mode, ObjTool: ed.ob.ltool,
		Tool: ed.pt.tool, Color: hexOf(ed.pt.col), Brush: ed.pt.brush, Mirror: ed.pt.mirror,
		LevMode: ed.lv.mode, LevZoom: ed.lv.prefZoom, LevLogic: ed.lv.showLogic, LevObjs: ed.lv.showObjs, LevPoints: ed.lv.showPts, LevBrush: ed.lv.bsize, LevObjTool: ed.lv.objTool, LevPtTool: ed.lv.ptTool,
	}
	for _, c := range ed.pt.recent {
		p.Recent = append(p.Recent, hexOf(c))
	}
	return p
}

// applyPrefs puts the remembered choices in place. Every value is checked: anything out
// of range (an old file, a hand-edited one) is ignored.
func (ed *Editor) applyPrefs(p editorPrefs) {
	cvGrid = p.Grid
	cvBG = clampI(p.BG, 0, len(canvasBGs)-1)
	ed.tab = clampI(p.Tab, 0, 2)

	ed.ch.typeIdx = clampI(p.CharType, 0, len(charTypes)-1)
	ed.ch.subIdx = clampI(p.CharSub, 0, len(charTypes[ed.ch.typeIdx].subs)-1)
	ed.ch.mode = clampI(p.CharMode, 0, 1)
	if p.PrevAnim >= 0 && p.PrevAnim < AnimCount && p.PrevAnim != AnimLying && p.PrevAnim != AnimSleep {
		ed.ch.prevAnim = p.PrevAnim
	}
	ed.ch.prevDir = clampI(p.PrevDir, 0, len(prevDirs)-1)

	ed.ob.view = clampI(p.ObjView, 0, 1)
	ed.ob.typeIdx = clampI(p.ObjType, 0, len(objTypes)-1)
	ed.ob.mode = clampI(p.ObjMode, 0, 1)
	ed.ob.ltool = clampI(p.ObjTool, 0, 2)

	ed.pt.tool = clampI(p.Tool, 0, ToolCount-1)
	if c, ok := parseHex(p.Color); ok {
		ed.pt.col, ed.pt.hexText = c, p.Color
	}
	ed.pt.recent = nil
	for _, h := range p.Recent {
		if c, ok := parseHex(h); ok && len(ed.pt.recent) < 24 {
			ed.pt.recent = append(ed.pt.recent, c)
		}
	}
	ed.pt.brush = clampI(p.Brush, 1, 8)
	ed.pt.mirror = p.Mirror

	ed.lv.mode = clampI(p.LevMode, 0, 4)
	for _, z := range lvZoomSteps {
		if z == p.LevZoom {
			ed.lv.prefZoom = z
		}
	}
	ed.lv.showLogic, ed.lv.showObjs, ed.lv.showPts = p.LevLogic, p.LevObjs, p.LevPoints
	ed.lv.bsize = clampI(p.LevBrush, 1, 8)
	ed.lv.objTool = clampI(p.LevObjTool, 0, 2)
	ed.lv.ptTool = clampI(p.LevPtTool, 0, ptMoveTool)
}

// defaultPrefs are the values of a fresh editor.
func defaultPrefs() editorPrefs {
	return editorPrefs{Grid: true, CharType: 1, Tool: ToolPencil, Color: "2C3C78", Brush: 1, LevObjs: true, LevPoints: true, LevBrush: 2}
}

func readPrefs(path string) (editorPrefs, bool) {
	p := defaultPrefs()
	data, err := os.ReadFile(path)
	if err != nil {
		return p, false
	}
	if json.Unmarshal(data, &p) != nil {
		return defaultPrefs(), false
	}
	return p, true
}

func writePrefs(path string, p editorPrefs) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// loadPrefs is called once when the editor is first opened.
func (ed *Editor) loadPrefs() {
	if noAssets {
		return
	}
	if path := prefsPath(); path != "" {
		if p, ok := readPrefs(path); ok {
			ed.applyPrefs(p)
			ed.ob.newBlank() // a blank object of the remembered kind
		}
	}
	data, _ := json.Marshal(ed.currentPrefs())
	ed.prefSaved = string(data)
}

// savePrefs writes the choices once they have changed and settled (about a second and a
// half without further changes). Nothing is written in screenshot or benchmark runs.
func (ed *Editor) savePrefs() {
	ed.prefTick++
	if ed.prefTick%15 != 0 {
		return
	}
	data, _ := json.Marshal(ed.currentPrefs())
	cur := string(data)
	switch {
	case cur == ed.prefSaved:
		ed.prefPending = ""
	case cur != ed.prefPending:
		ed.prefPending, ed.prefAt = cur, ed.prefTick
	case ed.prefTick-ed.prefAt >= 90:
		ed.prefSaved, ed.prefPending = cur, ""
		if path := prefsPath(); path != "" && !noAssets && !progressFrozen {
			writePrefs(path, ed.currentPrefs())
		}
	}
}
