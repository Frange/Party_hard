package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// gameWin* remember the game window size to restore it when leaving the editor.
var gameWinW, gameWinH int

var benchStop func()

// roleOf is charRoleOf with a cache (it opens and decodes a meta.json per character).
func (ed *Editor) roleOf(dir string) string {
	if r, ok := ed.roleCache[dir]; ok {
		return r
	}
	if ed.roleCache == nil {
		ed.roleCache = map[string]string{}
	}
	r := charRoleOf(dir)
	ed.roleCache[dir] = r
	return r
}

// spriteDir is spriteDirOf with a cache: the lists ask for it for every thumbnail on every
// frame and each answer costs a disk lookup. reloadList clears the cache.
func (ed *Editor) spriteDir(cat, name string) string {
	k := cat + "/" + name
	if d, ok := ed.dirCache[k]; ok {
		return d
	}
	if ed.dirCache == nil {
		ed.dirCache = map[string]string{}
	}
	d := spriteDirOf(ed.root, cat, name)
	ed.dirCache[k] = d
	return d
}

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
	imp  *ImportDlg // the "adjust picture" window while it is open

	debug          bool // editor debug mode: the game's own scenarios can be corrected
	konami         int
	demoMove       string
	pendingOrig    int // debug flag: open this original scenario on the first frame
	dirCache       map[string]string
	prefSaved      string // what editor_prefs.json holds (as JSON)
	prefPending    string
	prefAt         int
	prefTick       int
	roleCache      map[string]string
	benchN         int // debug: quit after this many frames and print the average cost
	benchU, benchD time.Duration
	benchF         int
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
		ed.loadPrefs()
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
	ebiten.SetWindowTitle(editorWindowTitle())
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
	ebiten.SetWindowTitle(gameWindowTitle())
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
	if ed.benchN > 0 {
		t0 := time.Now()
		defer func() {
			ed.benchU += time.Since(t0)
			ed.benchF++
			if ed.benchF == ed.benchN {
				fmt.Printf("editor bench: %d frames, update %.2f ms, draw %.2f ms per frame\n", ed.benchF, float64(ed.benchU.Microseconds())/1000/float64(ed.benchF), float64(ed.benchD.Microseconds())/1000/float64(ed.benchF))
				if benchStop != nil {
					benchStop()
				}
				os.Exit(0)
			}
		}()
	}
	u := ed.ui
	u.begin()
	if ed.msgT > 0 {
		ed.msgT--
	}
	if ed.imp != nil {
		if drops := droppedPNGs(); len(drops) > 0 {
			ed.imp.queue = append(ed.imp.queue, drops...)
			ed.imp.total += len(drops)
		}
		ed.imp.update(u)
		return nil
	}
	if ed.pendingOrig > 0 {
		ed.lv.loadOriginal(ed.pendingOrig - 1)
		ed.pendingOrig = 0
		ed.lv.demo(ed.demoMove)
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
	if ed.debug {
		u.round(edW-402, 6, 84, 24, rgb(0x8a1f3a))
		u.textC(edW-360, 11, "DEBUG", uiText)
		u.tooltip(edW-402, 6, 84, 24, "Modo debug del editor: puedes corregir los escenarios originales del juego")
	}
	ed.konamiStep(u)
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
		ed.requestImport(drops)
	}
	ed.savePrefs()
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
	if g.ed.benchN > 0 {
		t0 := time.Now()
		defer func() { g.ed.benchD += time.Since(t0) }()
	}
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
				if dirExists(spriteDirOf(ed.root, sb.folder, name)) && sb.matches(charRoleOf(spriteDirOf(ed.root, sb.folder, name))) {
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
	err  string // set when the file could not be read
}

func isImgName(p string) bool {
	p = strings.ToLower(p)
	return strings.HasSuffix(p, ".png") || strings.HasSuffix(p, ".jpg") || strings.HasSuffix(p, ".jpeg") || strings.HasSuffix(p, ".pdf")
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
		if strings.HasSuffix(strings.ToLower(path), ".pdf") {
			// the PDF reader needs a real file
			tmp, err := os.CreateTemp("", "ph-drop*.pdf")
			if err != nil {
				return nil
			}
			io.Copy(tmp, f)
			tmp.Close()
			defer os.Remove(tmp.Name())
			b, err := loadImageAny(tmp.Name())
			if err != nil {
				out = append(out, droppedImage{err: err.Error()})
				return nil
			}
			out = append(out, droppedImage{name: strings.ToLower(path), b: b})
			return nil
		}
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

// konamiStep watches the arrow keys for up, up, down, down, left, right, left, right.
// It only listens while no text field has the focus.
func (ed *Editor) konamiStep(u *UI) {
	if u.focus != "" || ed.debug {
		ed.konami = 0
		return
	}
	seq := []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyArrowLeft, ebiten.KeyArrowRight}
	for _, k := range []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight} {
		if !inpututil.IsKeyJustPressed(k) {
			continue
		}
		switch {
		case k == seq[ed.konami]:
			ed.konami++
		case k == ebiten.KeyArrowUp:
			ed.konami = 1
		default:
			ed.konami = 0
		}
		break
	}
	if ed.konami >= len(seq) {
		ed.konami = 0
		ed.debug = true
		ed.say("MODO DEBUG DESBLOQUEADO")
		ed.g.snd.Play("win")
	}
}
