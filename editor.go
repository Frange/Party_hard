package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// gameWin* remember the game window size to restore it when leaving the editor.
var gameWinW, gameWinH int

type Editor struct {
	g    *Game
	root string
	ui   *UI
	tab  int
	msg  string
	msgT int
	pt   *Painter
	ch   *CharEd
	ob   *ObjEd
	lv   *LevEd
}

func (g *Game) openEditor() {
	root := assetsRoot(true)
	if g.ed == nil {
		ed := &Editor{g: g, root: root, pt: newPainter()}
		ed.ui = &UI{g: g}
		ed.ch = newCharEd(ed)
		ed.ob = newObjEd(ed)
		ed.lv = newLevEd(ed)
		g.ed = ed
		ed.ch.reloadList()
		if len(ed.ch.list) > 0 {
			ed.ch.load(ed.ch.list[0])
			ed.ch.listSel = 0
		}
	}
	g.ed.root = root
	if n := exportMissingCharacters(root); n > 0 {
		g.ed.say(fmt.Sprintf("%d personajes añadidos para poder editarlos", n))
	}
	g.ed.ch.reloadList()
	g.ed.ob.reloadList()
	g.ed.lv.reloadList()
	g.mode = ModeEditor
	g.snd.StopMusic()
	mw, mh := ebiten.ScreenSizeInFullscreen()
	w, h := edW, edH
	if float64(w) > float64(mw)*0.96 || float64(h) > float64(mh)*0.9 {
		k := math_min(float64(mw)*0.96/float64(w), float64(mh)*0.9/float64(h))
		w, h = int(float64(w)*k), int(float64(h)*k)
	}
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowTitle("Party Hard - Editor de sprites y escenarios")
}

func math_min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func (g *Game) closeEditor() {
	g.mode = ModeMenu
	ebiten.SetWindowSize(gameWinW, gameWinH)
	ebiten.SetWindowTitle("Party Hard - Fiesta Nocturna")
	// pick up whatever was edited
	g.store = loadAssetStore()
	if g.store != nil {
		applyObjects(g.store.Objs)
	}
	g.as.props = buildPropImages()
	g.rigCache = map[*CharAsset]*CharSet{}
	g.setupCast()
	g.rebuildLevels()
	if g.levelIdx >= len(levelDefs) {
		g.levelIdx = 0
	}
	g.menuSel = 0
	g.loadLevel(0)
	g.snd.StartMusic()
}

func (ed *Editor) say(s string) { ed.msg, ed.msgT = s, 240 }

func (g *Game) updateEditor() error {
	ed := g.ed
	u := ed.ui
	u.begin()
	if ed.msgT > 0 {
		ed.msgT--
	}
	// top bar
	u.rect(0, 0, edW, edTop, rgb(0x0e0e14))
	u.rect(0, edTop-1, edW, 1, uiLine)
	u.add(func(d *ebiten.Image) { drawIcon(d, "star", 12, 8, 1.5, uiAccent, uiAccent) })
	u.text(36, 11, "EDITOR", uiAccent)
	tabs := []segItem{{"person", "Personajes"}, {"sofa", "Objetos y animales"}, {"world", "Escenarios"}}
	if i := u.seg(110, 4, 560, 28, tabs, ed.tab); i >= 0 {
		ed.tab = i
	}
	langs := 0
	if editorLang == "en" {
		langs = 1
	}
	if i := u.seg(edW-310, 4, 100, 28, []segItem{{"", "ES"}, {"", "EN"}}, langs); i >= 0 {
		editorLang = []string{"es", "en"}[i]
		saveLanguages()
	}
	if u.iconBtn(edW-200, 4, 192, 28, "back", "Volver al juego", false, "F2") || inpututil.IsKeyJustPressed(ebiten.KeyF2) && u.focus == "" {
		g.closeEditor()
		return nil
	}
	if ed.msgT > 0 {
		u.text(690, 11, ed.msg, uiAccent)
	}
	// global hotkeys (when no text field is focused)
	if u.focus == "" {
		ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyS) {
			ed.save()
		}
		if !ctrl {
			for k, tool := range map[ebiten.Key]int{ebiten.KeyP: ToolPencil, ebiten.KeyE: ToolEraser, ebiten.KeyI: ToolPicker} {
				if inpututil.IsKeyJustPressed(k) {
					ed.pt.tool = tool
				}
			}
		}
	}
	if drops := droppedPNGs(); len(drops) > 0 {
		ed.handleDrops(drops)
	}
	switch ed.tab {
	case 0:
		ed.ch.update(u)
	case 1:
		ed.ob.update(u)
	default:
		ed.lv.update(u)
	}
	u.finishMenu()
	if u.tip != "" && u.menu.id == "" {
		t := u.tip
		tx := minI(u.mx+12, edW-int(g.textW(t, 1))-16)
		u.round(tx, u.my+18, int(g.textW(t, 1))+12, 20, rgb(0x08080c))
		u.roundFrame(tx, u.my+18, int(g.textW(t, 1))+12, 20, uiLine)
		u.text(tx+6, u.my+22, t, uiText)
	}
	return nil
}

func (ed *Editor) save() {
	switch ed.tab {
	case 0:
		ed.ch.save()
	case 1:
		ed.ob.save()
	case 2:
		ed.lv.save()
	}
}

func (g *Game) drawEditor(screen *ebiten.Image) {
	screen.Fill(uiBG)
	for _, f := range g.ed.ui.cmds {
		f(screen)
	}
}

// ---------------------------------------------------------------------------
// helpers shared by the editors
// ---------------------------------------------------------------------------

func listAssetNames(dir string) []string {
	names := listDirs(dir)
	sort.Strings(names)
	return names
}

func cycleStr(cur string, opts []string) string {
	for i, o := range opts {
		if o == cur {
			return opts[(i+1)%len(opts)]
		}
	}
	return opts[0]
}

func ensureDir(p string) { os.MkdirAll(p, 0o755) }

var _ = filepath.Join
var _ = fmt.Sprintf
var _ = rand.Int

// debugLoad opens an asset by name in the current tab (used for screenshots and tests).
func (ed *Editor) debugLoad(name string) {
	switch ed.tab {
	case 0:
		for ti, t := range charTypes {
			for si, sb := range t.subs {
				if _, err := os.Stat(filepath.Join(spritesDir(ed.root, sb.folder), name)); err == nil && sb.matches(charRoleOf(filepath.Join(spritesDir(ed.root, sb.folder), name))) {
					ed.ch.typeIdx, ed.ch.subIdx = ti, si
					ed.ch.reloadList()
					ed.ch.load(name)
					return
				}
			}
		}
	case 1:
		ed.ob.loadByName(name)
	case 2:
		ed.lv.loadByName(name)
	}
}

type droppedImage struct {
	name string
	b    *pixBuf
}

func isImgName(p string) bool {
	p = strings.ToLower(p)
	return strings.HasSuffix(p, ".png") || strings.HasSuffix(p, ".jpg") || strings.HasSuffix(p, ".jpeg")
}

// droppedPNGs returns the PNG files dropped onto the window during the last frame.
func droppedPNGs() []droppedImage {
	fsys := ebiten.DroppedFiles()
	if fsys == nil {
		return nil
	}
	var out []droppedImage
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !isImgName(path) {
			return nil
		}
		f, err := fsys.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		im, _, err := image.Decode(f)
		if err != nil {
			return nil
		}
		out = append(out, droppedImage{name: strings.ToLower(path), b: pixBufFromImage(im)})
		return nil
	})
	return out
}

func (ed *Editor) handleDrops(drops []droppedImage) {
	switch ed.tab {
	case 0:
		ed.ch.importDrops(drops)
	case 1:
		ed.ob.importDrops(drops)
	case 2:
		ed.lv.importDrops(drops)
	}
}
