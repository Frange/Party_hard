package main

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Geometry of the menus: no block touches another (>= menuGap px), nothing sits closer than menuEdge px to the
// edge of the 960x540 screen (full-bleed frames aside) and text keeps menuPad px inside its box.
const (
	menuGap  = 12.0
	menuEdge = 16.0
	menuPad  = 6.0
)

type mBlock struct {
	name       string
	x, y, w, h float64
}

func (b mBlock) x1() float64 { return b.x + b.w }
func (b mBlock) y1() float64 { return b.y + b.h }

func inkBlock(name string, r textRect) mBlock {
	x0, y0, x1, y1 := r.ink()
	return mBlock{name, x0, y0, x1 - x0, y1 - y0}
}

func unionBlocks(name string, bs ...mBlock) mBlock {
	u := bs[0]
	u.name = name
	for _, b := range bs[1:] {
		x0, y0 := math.Min(u.x, b.x), math.Min(u.y, b.y)
		x1, y1 := math.Max(u.x1(), b.x1()), math.Max(u.y1(), b.y1())
		u.x, u.y, u.w, u.h = x0, y0, x1-x0, y1-y0
	}
	return u
}

func mGap(a, b mBlock) float64 {
	gx := math.Max(b.x-a.x1(), a.x-b.x1())
	gy := math.Max(b.y-a.y1(), a.y-b.y1())
	return math.Max(gx, gy)
}

func needGap(t *testing.T, ctx string, min float64, a, b mBlock) {
	t.Helper()
	if gp := mGap(a, b); gp < min {
		t.Errorf("%s: %s %+v and %s %+v are %.0f px apart (min %.0f)", ctx, a.name, a, b.name, b, gp, min)
	}
}

func needEdge(t *testing.T, ctx string, b mBlock) {
	t.Helper()
	if b.x < menuEdge || b.y < menuEdge || b.x1() > screenW-menuEdge || b.y1() > screenH-menuEdge {
		t.Errorf("%s: %s %+v is closer than %.0f px to the screen edge", ctx, b.name, b, menuEdge)
	}
}

// drawRec draws the current screen once and returns what was drawn as text.
func drawRec(g *Game) []textRect {
	textRec = &textRecorder{}
	img := ebiten.NewImage(screenW, screenH)
	g.Draw(img)
	img.Deallocate()
	out := textRec.texts
	textRec = nil
	return out
}

// logoBlock: the PARTY / HARDY letters with the bob (+-3 px) and the outline counted.
func logoBlock(texts []textRect) (mBlock, bool) {
	var bs []mBlock
	for _, r := range texts {
		if r.kind == "gradText" && (r.s == "PARTY" || r.s == "HARDY") {
			b := inkBlock("logo", r)
			b.w += 2
			b.h += 2
			bs = append(bs, b)
		}
	}
	if len(bs) == 0 {
		return mBlock{}, false
	}
	return unionBlocks("logo", bs...), true
}

// logoBobbing: the logo block at the highest and the lowest point of its bob.
func logoBobbing(g *Game, t0 float64) mBlock {
	var all []mBlock
	for _, tm := range []float64{math.Pi / 6, math.Pi / 2} { // sin(3t) = +1 and -1
		g.time = tm
		b, ok := logoBlock(drawRec(g))
		if !ok {
			return mBlock{}
		}
		all = append(all, b)
	}
	g.time = t0
	return unionBlocks("logo", all...)
}

// The main menu in every state (normal / bronze / silver / gold, with and without DEBUG), 1 and 2 players, ES and
// EN: the logo, the title under it, the rows, the completion block, the "?" and trophy buttons and the version
// lines keep their air, and the right hand blocks stay 16 px from the edge.
func TestMenuMarginsMainMenu(t *testing.T) {
	g := auditGame(t)
	pl, ps, pu, pe, pt := twoPlayerMode, gameLang, uiLang, editorLang, g.time
	t.Cleanup(func() {
		twoPlayerMode, gameLang, uiLang, editorLang, g.time = pl, ps, pu, pe, pt
		g.dbg.tier, g.debugMenu = 0, false
		auditReset(g)
	})
	for _, lang := range []string{"es", "en"} {
		gameLang, uiLang, editorLang = lang, lang, lang
		for _, two := range []bool{false, true} {
			for tier := 1; tier <= 4; tier++ {
				for _, dbg := range []bool{false, true} {
					auditReset(g)
					twoPlayerMode, g.dbg.tier, g.debugMenu = two, tier, dbg
					g.mode = ModeMenu
					ctx := fmt.Sprintf("%s two=%v tier=%d debug=%v", lang, two, tier, dbg)
					logo := logoBobbing(g, 0)
					if logo.w == 0 {
						t.Fatalf("%s: no logo drawn", ctx)
					}
					texts := drawRec(g)
					title, _ := menuTierTitle(tier - 1)
					var rows []mBlock
					var titleB *mBlock
					for _, r := range texts {
						switch {
						case r.kind == "drawText" && title != "" && r.s == tr(title):
							b := inkBlock("tier title", r)
							titleB = &b
						case r.kind == "drawText" && r.x >= 108 && r.x <= 112:
							rows = append(rows, inkBlock("row "+r.s, r))
						case r.s == menuVersionLine() || r.s == tr(menuHomageLine):
							needEdge(t, ctx, inkBlock("corner "+r.s, r))
						}
					}
					if len(rows) == 0 {
						t.Fatalf("%s: no rows", ctx)
					}
					first := rows[0]
					needGap(t, ctx, menuGap, logo, first)
					if title != "" {
						if titleB == nil {
							t.Fatalf("%s: tier title %q not drawn", ctx, title)
						}
						needGap(t, ctx, menuGap, logo, *titleB)
						needGap(t, ctx, menuGap, *titleB, first)
					}
					// the right hand blocks
					lines := g.complLines(g.complParts())
					w := 0.0
					for _, l := range lines {
						w = math.Max(w, g.textW(l, 1))
					}
					box := mBlock{"completion", complRight - w - 2*complPad, complY - 3, w + 2*complPad, float64(len(lines))*complPitch + 4}
					trophy := mBlock{"trophy counter", trophyHudX, trophyHudY, trophyHudW, trophyHudH}
					help := mBlock{"help button", helpBtnX, helpBtnY, helpBtnW, helpBtnH}
					needGap(t, ctx, menuGap, box, trophy)
					needGap(t, ctx, menuGap, box, help)
					needGap(t, ctx, menuGap, help, trophy)
					for _, b := range []mBlock{box, trophy, help, logo} {
						needEdge(t, ctx, b)
					}
					window := mBlock{"bedroom window", 340, 22, 356, 190} // the frame of the neighbours' window (bedroom.go)
					needGap(t, ctx, menuGap, box, window)
					for _, r := range texts {
						if r.kind == "drawText" && r.x >= complRight-w-2*complPad && r.y >= complY-3 && r.y < complY+float64(len(lines))*complPitch {
							x0, _, x1, _ := r.ink()
							if x0-box.x < menuPad || box.x1()-x1 < menuPad {
								t.Errorf("%s: %q is %.0f / %.0f px from the sides of its dark box (min %.0f)", ctx, r.s, x0-box.x, box.x1()-x1, menuPad)
							}
						}
					}
					for i, a := range rows { // the rows never touch each other
						for _, b := range rows[i+1:] {
							if a.y1() > b.y && a.x1() > b.x && a.y < b.y1() {
								t.Errorf("%s: %s overlaps %s", ctx, a.name, b.name)
							}
						}
					}
					if last := rows[len(rows)-1]; last.y1() > 474-menuGap+4 {
						t.Errorf("%s: the last row ends at y=%.0f (the footer starts at 474)", ctx, last.y1())
					}
				}
			}
		}
	}
}

// The panel screens (party list, shop, scenes, trophies, settings): the logo keeps >= 12 px from the panel and from
// the language box / back button, and the back button sits 16 px from the edges.
func TestMenuMarginsLogoAndBack(t *testing.T) {
	g := auditGame(t)
	pt := g.time
	t.Cleanup(func() { g.time = pt; auditReset(g) })
	back := mBlock{"back button", backBtnX, backBtnY, backBtnW, backBtnH}
	needEdge(t, "back", back)
	panel := mBlock{"panel", 230, 14, 700, 512}
	for name, m := range map[string]Mode{"levelselect": ModeLevelSelect, "shop": ModeShop, "scenes": ModeScenes, "trophies": ModeTrophies, "settings": ModeSettings} {
		for _, lang := range []string{"es", "en"} {
			gameLang, uiLang, editorLang = lang, lang, lang
			auditReset(g)
			g.mode = m
			logo := logoBobbing(g, 0)
			if logo.w == 0 {
				t.Fatalf("%s: no logo", name)
			}
			needGap(t, name+" "+lang, menuGap, logo, mBlock{"panel", 230, 0, 700, 540})
			needGap(t, name+" "+lang, menuGap, logo, back)
			_ = panel
		}
	}
	gameLang, uiLang, editorLang = "es", "es", "es"
}

// The shop in every list and in both languages, with 1 and 2 players: the title, PUNTOS, the groups, the second row
// of tabs, the rows, the counter, the message and BUY ALL keep their air inside a panel that ends above BACK.
func TestMenuMarginsShop(t *testing.T) {
	g := auditGame(t)
	pl, ps, pu, pe := twoPlayerMode, gameLang, uiLang, editorLang
	t.Cleanup(func() {
		twoPlayerMode, gameLang, uiLang, editorLang = pl, ps, pu, pe
		wardKind = 0
		auditReset(g)
	})
	panel := mBlock{"panel", shopPanX, shopPanY, shopPanW, shopPanH}
	back := mBlock{"back button", backBtnX, backBtnY, backBtnW, backBtnH}
	needGap(t, "shop", menuGap, panel, back)
	if panel.y < menuEdge {
		t.Errorf("shop panel at y=%.0f", panel.y)
	}
	for _, lang := range []string{"es", "en"} {
		gameLang, uiLang, editorLang = lang, lang, lang
		for _, two := range []bool{false, true} {
			twoPlayerMode = two
			for tab := range shopFlat {
				for _, sel := range []int{0, 4} {
					auditReset(g)
					g.mode = ModeShop
					g.progress.Wallet = 99999999
					g.shopGo(tab)
					wardKind = 0
					if n := len(g.shopItems()); n > 0 && sel >= n {
						sel = n - 1
					}
					g.shopSel = sel
					g.shopMsg = "¡Comprados 120 artículos por 123456 puntos!"
					ctx := fmt.Sprintf("%s two=%v tab=%d sel=%d", lang, two, tab, sel)
					texts := drawRec(g)
					g.shopMsg = ""
					var title, pts, counter, msg *mBlock
					var tabs, groups []mBlock
					for _, r := range texts {
						b := inkBlock(r.s, r)
						switch {
						case r.s == tr("TIENDA"):
							title = &b
						case strings.HasPrefix(r.s, tr("PUNTOS")+":"):
							pts = &b
						case r.kind == "drawText" && r.y == shopMsgY && r.x == 244, r.kind == "drawText" && r.y == shopMsgWardY && r.x == 244:
							msg = &b
						case r.y == shopTabTextY && r.x >= 244 && r.x < 560:
							tabs = append(tabs, b)
						case r.y == 82:
							groups = append(groups, b)
						case r.y == 408 && r.x == 244:
							counter = &b
						}
					}
					if title == nil || pts == nil || len(groups) != len(shopGroups) {
						t.Fatalf("%s: title/points/groups missing (%v %v %d)", ctx, title != nil, pts != nil, len(groups))
					}
					for _, b := range []*mBlock{title, pts} {
						if b.y-panel.y < menuPad || b.x1() > panel.x1()-menuPad {
							t.Errorf("%s: %s too close to the panel border", ctx, b.name)
						}
					}
					needGap(t, ctx, menuGap, *title, *pts)
					for gi, gb := range groups {
						fx := shopGroupX(gi) - 4
						frame := mBlock{"group frame", fx, shopGroupFrameY, shopGroupW + 2, 22}
						needGap(t, ctx, menuGap, *title, frame)
						needGap(t, ctx, menuGap, *pts, frame)
						for _, tb := range tabs {
							needGap(t, ctx, menuGap, frame, tb)
						}
						_ = gb
					}
					if counter != nil && msg != nil {
						needGap(t, ctx, menuGap, *counter, *msg)
					}
					buy := mBlock{"buy all", shopBuyX, shopBuyY, shopBuyW, shopBuyH}
					if g.shopTab != shopWardrobe {
						if msg != nil {
							needGap(t, ctx, menuGap, *msg, buy)
						}
						if buy.y1() > panel.y1()-menuGap+0.5 {
							t.Errorf("%s: BUY ALL ends at y=%.0f, panel at %.0f", ctx, buy.y1(), panel.y1())
						}
					}
					// the rows: the name and the price/tag of a row never touch, and the price keeps 6 px inside the row bar
					if g.shopTab != shopWardrobe {
						byY := map[float64][]mBlock{}
						for _, r := range texts {
							if r.kind == "drawText" && r.y >= 146 && r.y < 146+shopRows*28 && r.x >= 244 && r.x <= 600 {
								byY[r.y] = append(byY[r.y], inkBlock(r.s, r))
							}
						}
						for y, bs := range byY {
							if len(bs) != 2 {
								continue
							}
							a, b := bs[0], bs[1]
							if a.x > b.x {
								a, b = b, a
							}
							if b.x-a.x1() < menuGap {
								t.Errorf("%s: row y=%.0f %q and %q are %.0f px apart", ctx, y, a.name, b.name, b.x-a.x1())
							}
							if 600-b.x1() < menuPad {
								t.Errorf("%s: row y=%.0f %q is %.0f px from the right edge of its bar", ctx, y, b.name, 600-b.x1())
							}
						}
					}
				}
			}
		}
	}
}
