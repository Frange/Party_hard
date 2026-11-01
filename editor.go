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
	u.rect(0, 0, edW, 28, rgb(0x14141c))
	tabs := []string{"Personajes", "Objetos / animales", "Escenarios", "Ayuda"}
	for i, t := range tabs {
		if u.button(6+i*150, 4, 146, 20, t, ed.tab == i) {
			ed.tab = i
		}
	}
	if u.button(edW-170, 4, 164, 20, "Volver al juego (F2)", false) || inpututil.IsKeyJustPressed(ebiten.KeyF2) && u.focus == "" {
		g.closeEditor()
		return nil
	}
	if ed.msgT > 0 {
		u.text(640, 8, ed.msg, uiAccent)
	}
	// global hotkeys (when no text field is focused)
	if u.focus == "" {
		ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyS) {
			ed.save()
		}
		if !ctrl {
			for k, tool := range map[ebiten.Key]int{ebiten.KeyP: ToolPencil, ebiten.KeyE: ToolEraser, ebiten.KeyL: ToolLine, ebiten.KeyR: ToolRect, ebiten.KeyF: ToolFill, ebiten.KeyI: ToolPicker, ebiten.KeyO: ToolEllipse} {
				if inpututil.IsKeyJustPressed(k) {
					ed.pt.tool = tool
				}
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyM) {
				ed.pt.mirror = !ed.pt.mirror
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
	case 2:
		ed.lv.update(u)
	default:
		ed.helpTab(u)
	}
	if u.tip != "" {
		t := u.tip
		u.rect(u.mx+12, u.my+16, int(g.textW(t, 1))+10, 18, rgb(0x08080c))
		u.text(u.mx+17, u.my+19, t, uiText)
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

func (ed *Editor) helpTab(u *UI) {
	lines := []string{
		"EDITOR DE SPRITES Y ESCENARIOS",
		"",
		"Todo el contenido del juego vive en la carpeta 'assets' (junto al ejecutable). El juego la lee al arrancar,",
		"así que puedes cambiar personajes, objetos y escenarios sin tocar código. La primera vez se crean con el contenido por defecto.",
		"",
		"  assets/sprites/main_character/  protagonista        assets/sprites/people/  invitados aleatorios",
		"  assets/sprites/special_people/  famosos y personajes  assets/sprites/enemies/ guardias de seguridad",
		"  assets/sprites/special_enemies/ guardias especiales   assets/sprites/police/  policías (role cop) y sanitarios (role medic)",
		"  assets/sprites/objetos/         muebles, vehículos, animales, trampas      assets/backgrounds/  escenarios",
		"",
		"PERSONAJES: pinta el muñeco de pie (frente, perfil, espalda) y marca con 'Parte del cuerpo' qué es cabeza, torso, brazos y piernas.",
		"  El programa genera solo todas las animaciones (andar, correr, bailar, beber, apuñalar...). Si un fotograma no te gusta,",
		"  usa 'Editar este fotograma' para pintarlo a mano; se guarda en la carpeta anim/ del personaje y sustituye al generado.",
		"  Los píxeles sin parte se asignan solos a la parte más cercana al guardar.",
		"OBJETOS: pinta el sprite, marca el ancla (pies), la huella de colisión y las propiedades: sólido, esconderse, teléfono,",
		"  ponche envenenable, trampa (explota / electrocuta), farola, puntos de uso (beber, baño, sofá) y fotogramas animados.",
		"ESCENARIOS: pinta el fondo píxel a píxel (640x360), dibuja la lógica (paredes, vallas, agua, zonas de paseo y de baile),",
		"  coloca objetos, el inicio del jugador, la ruta de los guardias, DJ y músicos, luces y la llegada de la policía. 'Probar' lo lanza.",
		"",
		"ATAJOS: P lápiz  E goma  L línea  R rectángulo  O elipse  F cubo  I cuentagotas  M simetría  Ctrl+Z / Ctrl+Y deshacer  Ctrl+S guardar",
		"  Rueda del ratón: zoom en el lienzo.  Botón central o espacio+arrastrar: mover el lienzo.",
	}
	for i, l := range lines {
		c := uiText
		if i == 0 {
			c = uiAccent
		}
		u.text(30, 50+i*18, l, c)
	}
	u.text(30, 50+len(lines)*18+10, "Carpeta de assets: "+ed.root, uiDim)
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
		for i, c := range charCategories {
			if _, err := os.Stat(filepath.Join(spritesDir(ed.root, c), name)); err == nil {
				ed.ch.catIdx = i
				ed.ch.reloadList()
				ed.ch.load(name)
				return
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
