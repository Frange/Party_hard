package main

// TestUITextAudit: automatic audit of every text drawn on the main screens, in Spanish and English.
// It needs no window: the game draws on an off-screen image while uitext_rec.go records each text call
// (rectangle, string, scale, image it went to) and every big filled panel. Then, per screen:
//   (a) no two texts overlap more than a little (explicit exceptions in auditAllow, each with a reason)
//   (b) every text lies inside the logical screen 960x540, and does not cross the border of a panel
//   (c) scale and position are whole numbers (a bitmap font at a fractional scale or offset is blurry),
//       and the text goes straight to the screen, not to a buffer that is scaled afterwards
//   (d) the English text, usually longer, obeys all of the above too (every screen is audited in both)
//   (e) nothing lies on the logo (its PARTY / HARD are recorded as texts, so (a) covers it)
// Run alone: go test -count=1 -run TestUITextAudit .    (fast mode: go test -short ... only audits English)
// Read docs/QA/MEMORIA.md before adding an exception.

import (
	"fmt"
	"image"
	"math"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// an overlap that is by design. screen "" = any screen; a and b are substrings of the texts (b "" = any).
type auditExc struct{ screen, a, b, why string }

var auditAllow = []auditExc{
	{"", "PARTY", "HARD", "the two words of the logo are one stacked block"},
	{"", "=Z", "=Z", "the Zs of the sleeper in the bedroom scenery float over each other"},
}

// texts that may leave the screen or a panel on purpose (scrolling credits, shake, etc.)
var auditOutside = []auditExc{
	{"wardrobe players=2", "Con quién sales", "", "the big bed of the 2P bedroom scenery counts as a panel: the text is on the right of the real list"},
	{"wardrobe players=2", "has comprado en", "", "same"},
	{"wardrobe players=2", "Who you go to", "", "same"},
	{"wardrobe players=2", "you bought under", "", "same"},
	{"wardrobe players=2", "Color de pelo", "", "same"},
}

var auditSpanishOK = []auditExc{
	{"", "César", "", "a tester's name in the credits"},
	{"", "Filemón", "", "name of a character of the shop (proper noun)"},
	{"", "Tintín", "", "name of a character of the shop (proper noun)"},
	{"", "Capitán Haddock", "", "name of a character of the shop (proper noun)"},
}

// FAULTS FOUND BY THE AUDIT THAT ARE NOT FIXED YET (owner told). They are printed, not failed, and the test fails when
// one stops happening: then delete the line. Never add a fault here to get a green test without telling the orchestrator.
var auditPending = []auditExc{}

type auditShot struct {
	name   string
	setup  func(g *Game)
	panels bool // check the border of big filled panels (menu-like screens only)
}

var (
	auditOnce sync.Once
	auditG    *Game
)

func auditGame() *Game {
	auditOnce.Do(func() {
		noAssets = true
		progressFrozen = true // nothing here may touch the real progress.json
		loadLanguages()
		auditG = newGame(true)
	})
	return auditG
}

func (e auditExc) match(screen string, a, b string) bool {
	if e.screen != "" && !strings.Contains(screen, e.screen) {
		return false
	}
	m := func(pat, x string) bool { // "=text" is an exact match, anything else a substring
		if strings.HasPrefix(pat, "=") {
			return x == pat[1:]
		}
		return strings.Contains(x, pat)
	}
	hit := func(x, y string) bool { return m(e.a, x) && (e.b == "" || m(e.b, y)) }
	return hit(a, b) || hit(b, a)
}

type auditFail struct{ screen, msg, a, b string } // a, b: the texts involved, to match exceptions

// inkRect is the part of a text that has ink: the font's line is taller than its glyphs.
func (t textRect) ink() (x0, y0, x1, y1 float64) {
	return t.x, t.y + 3*t.scale, t.x + t.w, t.y + t.h - 3*t.scale
}

func overlapFrac(a, b textRect) float64 {
	ax0, ay0, ax1, ay1 := a.ink()
	bx0, by0, bx1, by1 := b.ink()
	w := math.Min(ax1, bx1) - math.Max(ax0, bx0)
	h := math.Min(ay1, by1) - math.Max(ay0, by0)
	if w <= 0 || h <= 0 {
		return 0
	}
	small := math.Min((ax1-ax0)*(ay1-ay0), (bx1-bx0)*(by1-by0))
	if small <= 0 {
		return 0
	}
	return w * h / small
}

const auditMaxOverlap = 0.10 // of the smaller text

// auditCheck applies the rules (a)-(e) to what was recorded for one screen.
func auditCheck(screen string, rec *textRecorder, panels bool, exc []auditExc, outside []auditExc) []auditFail {
	var out []auditFail
	add := func(f string, a ...any) { out = append(out, auditFail{screen: screen, msg: fmt.Sprintf(f, a...)}) }
	desc := func(t textRect) string {
		return fmt.Sprintf("%q [%s x%.0f at %.0f,%.0f %.0fx%.0f]", t.s, t.kind, t.scale, t.x, t.y, t.w, t.h)
	}
	seen := map[string]bool{}
	var ts []textRect
	for _, t := range rec.texts { // the same text twice on the same spot is a drawing trick, not two texts
		k := fmt.Sprint(t.s, t.x, t.y, t.scale)
		if !seen[k] {
			seen[k] = true
			ts = append(ts, t)
		}
	}
	allowed := func(list []auditExc, a, b string) bool {
		for _, e := range list {
			if e.match(screen, a, b) {
				return true
			}
		}
		return false
	}
	var vis []textRect // a text outside the clip it was drawn in (a scrolled list) is not visible
	for _, t := range ts {
		if t.sub && (int(t.x) >= t.clip.Max.X || int(t.x+t.w) <= t.clip.Min.X || int(t.y) >= t.clip.Max.Y || int(t.y+t.h) <= t.clip.Min.Y) {
			continue
		}
		vis = append(vis, t)
	}
	ts = vis
	for i, t := range ts {
		// (c)
		if t.scale != math.Floor(t.scale) || t.scale < 1 || t.x != math.Floor(t.x) || t.y != math.Floor(t.y) {
			add("not whole pixels: %s", desc(t))
		}
		if (t.dstW != screenW || t.dstH != screenH) && !t.sub {
			if !allowed(outside, t.s, "") {
				add("drawn on a %dx%d buffer (blurry if it is scaled later): %s", t.dstW, t.dstH, desc(t))
			}
			continue // coordinates of another image do not say where it ends up
		}
		// (b)
		if (t.x < 0 || t.y < 0 || t.x+t.w > screenW || t.y+t.h > screenH) && !allowed(outside, t.s, "") {
			add("outside the screen 960x540: %s", desc(t))
		}
		if panels && t.kind != "gradText" { // the logo sits behind the panels by design
			for _, p := range append(append([]panelRect{}, rec.panels...), rec.fills...) {
				ix := math.Min(t.x+t.w, p.x+p.w) - math.Max(t.x, p.x)
				iy := math.Min(t.y+t.h, p.y+p.h) - math.Max(t.y, p.y)
				if ix <= 0 || iy <= 0 {
					continue
				}
				inside := t.x >= p.x && t.x+t.w <= p.x+p.w && t.y >= p.y && t.y+t.h <= p.y+p.h
				if !inside && !allowed(outside, t.s, "") {
					add("crosses the border of the panel %.0f,%.0f %.0fx%.0f: %s", p.x, p.y, p.w, p.h, desc(t))
					break
				}
			}
		}
		// (d) English screens must not show Spanish (menu-like screens only: the play screens use made-up phrases)
		if panels && strings.HasSuffix(screen, "[en]") && strings.ContainsAny(t.s, "áéíóúñ¿¡Ñ") && !allowed(auditSpanishOK, t.s, "") {
			add("Spanish text in the English version: %s", desc(t))
		}
		// (a), (e)
		for j := i + 1; j < len(ts); j++ {
			u := ts[j]
			if (u.dstW != screenW || u.dstH != screenH) && !u.sub {
				continue
			}
			if f := overlapFrac(t, u); f > auditMaxOverlap && !allowed(exc, t.s, u.s) {
				out = append(out, auditFail{screen, fmt.Sprintf("overlap %.0f%%: %s  <->  %s", f*100, desc(t), desc(u)), t.s, u.s})
			}
		}
	}
	return out
}

func auditRender(g *Game, sh auditShot, lang string) (rec *textRecorder, err string) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Sprintf("the screen could not be drawn: %v\n%s", r, firstFrames(debug.Stack()))
		}
	}()
	gameLang, editorLang = lang, lang
	sh.setup(g)
	textRec = &textRecorder{}
	defer func() { textRec = nil }()
	img := ebiten.NewImage(screenW, screenH)
	g.Draw(img)
	img.Deallocate()
	return auditWithoutBackdrop(g, auditOcclude(textRec)), ""
}

func firstFrames(b []byte) string {
	var keep []string
	for _, l := range strings.Split(string(b), "\n") {
		if strings.Contains(l, "partyhard.") && !strings.Contains(l, "auditRender") {
			keep = append(keep, strings.TrimSpace(l))
			if len(keep) == 3 {
				break
			}
		}
	}
	return strings.Join(keep, "\n")
}

// ---- the screens ------------------------------------------------------------------------------

func auditReset(g *Game) {
	g.end, g.intro, g.title = nil, nil, nil
	g.showCredits, g.trophyOv, g.debugMenu = false, false, false
	g.msgs = nil
	g.menuMsgT = 0
	g.toasts = nil
	g.progress.Unlocked = 99
	if g.progress.Stats == nil {
		g.progress.Stats = map[string]int{}
	}
	g.progress.Stats["jp_unlocked"] = 1
	g.progress.Stats["played"] = 1
	twoPlayerMode = false
	dbgTwoPlayers = false
	bubbleReqs = nil
	g.lvl.graded = true // gradeBG reads pixels, which cannot be done without a window
}

func auditPlay(g *Game, two bool) {
	auditReset(g)
	twoPlayerMode, dbgTwoPlayers = two, two
	g.loadLevel(0)
	g.lvl.graded = true
	g.titleT = 0 // the title card of the party is audited apart (level start)
	if g.p2 != nil {
		g.p2.pos.X += 90
	}
	g.mode = ModePlay
}

func auditShots() []auditShot {
	var s []auditShot
	add := func(name string, panels bool, f func(g *Game)) { s = append(s, auditShot{name, f, panels}) }

	for _, two := range []bool{false, true} {
		for _, dbg := range []bool{false, true} {
			two, dbg := two, dbg
			for _, sel := range []int{0, 4, 9} { // first, middle and last row (the last is the longest list with DEBUG)
				sel := sel
				add(fmt.Sprintf("main menu players=%d debug=%v row=%d", map[bool]int{false: 1, true: 2}[two], dbg, sel), true, func(g *Game) {
					auditReset(g)
					twoPlayerMode = two
					g.debugMenu = dbg
					g.mode = ModeMenu
					g.mainSel = minI(sel, len(g.menuList())-1)
					if sel == 9 {
						g.menuMsg, g.menuMsgT = "Termina la fiesta actual antes de empezar otra", 2
					}
				})
			}
		}
	}
	for _, two := range []bool{false, true} { // the JUGADORES row selected, with its funny line
		for _, dbg := range []bool{false, true} {
			two, dbg := two, dbg
			add(fmt.Sprintf("main menu JUGADORES selected two=%v debug=%v", two, dbg), true, func(g *Game) {
				auditReset(g)
				twoPlayerMode = two
				g.debugMenu = dbg
				g.mode = ModeMenu
				for i, id := range g.menuList() {
					if id == menuPlayers {
						g.mainSel = i
					}
				}
			})
		}
	}
	add("title", false, func(g *Game) {
		auditReset(g)
		g.startTitle()
		g.title.t = 4
	})
	for _, sel := range []int{0, 7, 14, 20, 21, 22} {
		sel := sel
		add(fmt.Sprintf("level select (Isla Nublar unlocked) sel=%d", sel), true, func(g *Game) {
			auditReset(g)
			g.mode = ModeLevelSelect
			g.menuSel = sel
		})
	}
	for r := 0; r < setRows; r++ {
		r := r
		add(fmt.Sprintf("settings row=%d", r), true, func(g *Game) {
			auditReset(g)
			g.mode = ModeSettings
			g.setSel = r
		})
	}
	add("credits", true, func(g *Game) {
		auditReset(g)
		g.mode = ModeSettings
		g.showCredits = true
	})
	for tab := range shopFlat {
		tab := tab
		for _, sel := range []int{0, 3} {
			sel := sel
			add(fmt.Sprintf("shop list=%d item=%d", tab, sel), true, func(g *Game) {
				auditReset(g)
				g.mode = ModeShop
				g.progress.Wallet = 99999
				g.shopGo(tab)
				g.shopSel = sel
				if n := len(g.shopItems()); n > 0 && sel >= n {
					g.shopSel = n - 1
				}
			})
		}
	}
	for _, two := range []bool{false, true} {
		two := two
		add(fmt.Sprintf("wardrobe players=%d", map[bool]int{false: 1, true: 2}[two]), true, func(g *Game) {
			auditReset(g)
			twoPlayerMode = two
			g.mode = ModeShop
			g.shopTab = shopWardrobe
			g.progress.Wallet = 99999
		})
	}
	for _, sel := range []int{0, 5, 12, 30} {
		sel := sel
		add(fmt.Sprintf("trophies sel=%d", sel), true, func(g *Game) {
			auditReset(g)
			g.mode = ModeTrophies
			g.trophySel = minI(sel, len(trophyList)-1)
		})
	}
	add("trophies window in game + toast", false, func(g *Game) {
		auditPlay(g, false)
		g.trophyOv = true
		g.toasts = []trophyToast{{name: "Trofeo con un nombre bastante largo", reward: "Una recompensa larga"}}
		g.toasts[0].t = 1.5
	})
	for _, two := range []bool{false, true} {
		two := two
		n := map[bool]int{false: 1, true: 2}[two]
		add(fmt.Sprintf("HUD in game players=%d", n), false, func(g *Game) { auditPlay(g, two) })
		add(fmt.Sprintf("HUD + say messages + bubbles players=%d", n), false, func(g *Game) {
			auditPlay(g, two)
			g.msgs = []HudMsg{{"El invitado del sombrero ha caído al suelo muy borracho", 3}, {"Alguien ha llamado a la policía otra vez", 3}}
			cx := g.p.pos
			for i, ph := range []string{"No puedo creer lo que ha pasado aquí esta noche", "¡Socorro! ¡Hay alguien con un cuchillo!", "Qué fiesta más genial, ¿verdad?", "¡Eh! ¿Quién ha apagado la música?"} {
				g.bubble(i, 1, Vec{cx.X + float64(i-2)*60, cx.Y - 20 - float64(i%2)*30}, ph, i == 1, 0)
			}
		})
		add(fmt.Sprintf("level complete players=%d", n), false, func(g *Game) {
			auditPlay(g, two)
			g.mode, g.points, g.endTime = ModeLevelDone, 12345, 187
		})
		add(fmt.Sprintf("GAME OVER players=%d", n), false, func(g *Game) {
			auditPlay(g, two)
			g.mode, g.points, g.endTime = ModeLose, 12345, 187
			g.loseReason = "Te ha pillado la policía y te han llevado a la comisaría"
			g.endAt = g.time - 5
		})
	}
	add("level start: title card + intro message", false, func(g *Game) {
		auditPlay(g, false)
		g.titleT = 4.5
		g.say("Bienvenido a la fiesta. Que no quede nadie vivo.")
	})
	add("final screen", false, func(g *Game) {
		auditPlay(g, false)
		g.mode, g.points = ModeFinal, 999
	})
	for i := 0; i < sceneCount; i++ {
		i := i
		add(fmt.Sprintf("scenes list sel=%d", i), true, func(g *Game) {
			auditReset(g)
			g.progress.Stats = map[string]int{"seen_end1": 1, "seen_end2": 1, "seen_invite": 1, "seen_interlude5": 1}
			g.mode = ModeScenes
			g.sceneSel = i
		})
	}
	add("letter and invitation", false, func(g *Game) {
		auditReset(g)
		g.startInvite()
		g.end.t = 14
	})
	return s
}

// ---- the test ---------------------------------------------------------------------------------

func TestUITextAudit(t *testing.T) {
	g := auditGame()
	defer func(a, b string) { gameLang, editorLang = a, b }(gameLang, editorLang)
	langs := []string{"es", "en"}
	if testing.Short() {
		langs = []string{"en"} // fast mode: English is the longer one
	}
	var fails []auditFail
	screens, texts := 0, 0
	pendingHit := map[int]bool{}
	var pending []auditFail
	for _, sh := range auditShots() {
		for _, lang := range langs {
			rec, err := auditRender(g, sh, lang)
			name := sh.name + " [" + lang + "]"
			if err != "" {
				fails = append(fails, auditFail{screen: name, msg: err})
				continue
			}
			screens++
			texts += len(rec.texts)
			if len(rec.texts) == 0 {
				fails = append(fails, auditFail{screen: name, msg: "no text was recorded (is the screen drawn?)"})
			}
			for _, f := range auditCheck(name, rec, sh.panels, auditAllow, auditOutside) {
				if i := auditPendingIdx(name, f); i >= 0 {
					pendingHit[i] = true
					pending = append(pending, f)
					continue
				}
				fails = append(fails, f)
			}
		}
	}
	t.Logf("UI text audit: %d screens drawn, %d texts checked, %d problems", screens, texts, len(fails))
	if len(pending) > 0 {
		seen := map[string]bool{}
		for _, f := range pending {
			k := f.a + "|" + f.b
			if !seen[k] {
				seen[k] = true
				t.Logf("PENDING (known, not fixed): %s: %s", f.screen, f.msg)
			}
		}
	}
	for i, e := range auditPending {
		if !pendingHit[i] && !testing.Short() {
			t.Errorf("auditPending[%d] (%q / %q) no longer happens: delete it from the list (%s)", i, e.a, e.b, e.why)
		}
	}
	if len(fails) > 0 {
		sort.SliceStable(fails, func(i, j int) bool { return fails[i].screen < fails[j].screen })
		var b strings.Builder
		fmt.Fprintf(&b, "\n===== UI TEXT AUDIT REPORT: %d problems =====\n", len(fails))
		last := ""
		for _, f := range fails {
			if f.screen != last {
				fmt.Fprintf(&b, "\n[%s]\n", f.screen)
				last = f.screen
			}
			fmt.Fprintf(&b, "  - %s\n", f.msg)
		}
		t.Error(b.String())
	}
}

// the audit has to catch real mistakes: overlapping, outside the screen, crossing a panel, fractional position
func TestUITextAuditDetectsFaults(t *testing.T) {
	mk := func(s string, x, y, sc float64, w float64) textRect {
		return textRect{"drawText", s, x, y, w, 16 * sc, sc, screenW, screenH, image.Rect(0, 0, screenW, screenH), false}
	}
	rec := &textRecorder{
		texts: []textRect{
			mk("OVERLAP A", 100, 100, 1, 90), mk("OVERLAP B", 130, 104, 1, 90), // overlapping
			mk("LEFT THE SCREEN", 900, 200, 1, 120),
			mk("CROSSES PANEL", 280, 300, 1, 100), // the panel ends at x=300
			mk("FRACTIONAL", 50.5, 400, 1.5, 80),
			mk("FINE", 600, 450, 1, 40), mk("ALSO FINE", 600, 470, 1, 60),
		},
		panels: []panelRect{{x: 230, y: 280, w: 70, h: 100}},
	}
	got := auditCheck("synthetic", rec, true, nil, nil)
	for _, want := range []string{"OVERLAP", "outside the screen", "crosses the border", "not whole pixels"} {
		found := false
		for _, f := range got {
			if strings.Contains(f.msg, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("the audit did not report %q (got %v)", want, got)
		}
	}
	for _, f := range got {
		if strings.Contains(f.msg, `"FINE"`) || strings.Contains(f.msg, "ALSO FINE") {
			t.Errorf("false alarm: %s", f.msg)
		}
	}
	// an exception silences exactly that pair
	got = auditCheck("synthetic", rec, true, []auditExc{{"", "OVERLAP A", "OVERLAP B", "test"}}, nil)
	for _, f := range got {
		if strings.Contains(f.msg, "overlap") {
			t.Errorf("the exception did not apply: %s", f.msg)
		}
	}
}

// the bedroom behind the menus draws its own texts (the clock, the Zs of the sleeper) and big shapes: they are
// scenery, not interface. They are drawn alone once and removed from what a screen recorded.
func auditWithoutBackdrop(g *Game, rec *textRecorder) *textRecorder {
	textRec = &textRecorder{}
	img := ebiten.NewImage(screenW, screenH)
	if twoPlayerMode {
		bedTwo, bedPoses, bedShiftX = true, menuBedPoses(), menuBedShiftX
	}
	g.drawBedroom(img)
	bedTwo, bedPoses, bedShiftX = false, [2]bedPose{}, 0
	img.Deallocate()
	base := textRec
	textRec = rec
	key := func(t textRect) string { return fmt.Sprint(t.s, t.x, t.y, t.scale) }
	bt := map[string]bool{}
	for _, t := range base.texts {
		bt[key(t)] = true
	}
	bf := map[panelRect]bool{}
	for _, f := range base.fills {
		bf[f] = true
	}
	out := &textRecorder{panels: rec.panels}
	for _, t := range rec.texts {
		if !bt[key(t)] {
			out.texts = append(out.texts, t)
		}
	}
	for _, f := range rec.fills {
		if !bf[f] {
			out.fills = append(out.fills, f)
		}
	}
	return out
}

// auditOcclude drops the texts that an opaque rectangle drawn later covers completely (the credits panel over the settings).
func auditOcclude(rec *textRecorder) *textRecorder {
	out := &textRecorder{panels: rec.panels, fills: rec.fills}
	for k, t := range rec.texts {
		hidden := false
		for _, f := range rec.fills {
			if f.opaque && f.seq > k && t.x >= f.x && t.y >= f.y && t.x+t.w <= f.x+f.w && t.y+t.h <= f.y+f.h {
				hidden = true
			}
		}
		if !hidden {
			out.texts = append(out.texts, t)
		}
	}
	return out
}

func auditPendingIdx(screen string, f auditFail) int {
	for i, e := range auditPending {
		if f.a != "" && e.match(screen, f.a, f.b) {
			return i
		}
	}
	return -1
}
