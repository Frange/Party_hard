package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func wardCharGame(t *testing.T) *Game {
	g := wardKindGame(t)
	pp, ps, psc, pw, pt, pm := wardPick, wardPickSel, wardPickScroll, wardP, g.shopTab, g.mode
	pn := wardrobeNumPlayers
	t.Cleanup(func() {
		wardPick, wardPickSel, wardPickScroll, wardP, wardrobeNumPlayers = pp, ps, psc, pw, pn
		g.shopTab, g.mode = pt, pm
	})
	wardPick, wardPickSel, wardPickScroll, wardP = false, 0, 0, 0
	return g
}

func charCount() int {
	n := 0
	seen := map[string]bool{}
	for _, c := range albumCards() {
		if c.kind() == cardChar && !seen[c.slug] {
			seen[c.slug] = true
			n++
		}
	}
	return n
}

// the list: "Por defecto" first, then what is won (or was bought), then the locked ones as "???"
func TestWardrobeCharPickList(t *testing.T) {
	g := wardCharGame(t)
	l := g.wardPickList()
	if l[0].id != "" || l[0].name != "Por defecto" || l[0].locked {
		t.Fatalf("first cell must be the default look: %+v", l[0])
	}
	if len(l) != 1+charCount() {
		t.Fatalf("cells %d, want default + %d characters", len(l), charCount())
	}
	for _, it := range l[1:] {
		if !it.locked {
			t.Fatalf("nothing is won yet but %s is wearable", it.id)
		}
	}
	if o := g.wardBaseOptions(); len(o) != 1 || o[0].id != "" {
		t.Fatalf("row options with nothing won: %v", o)
	}
	// one won by kills, one bought in the old shop; an old skin with no preset any more is skipped (no crash)
	g.progress.Album = map[string]int{"homer": albumSeen | albumElim | albumKilled}
	g.progress.CharKills = map[string]int{"homer": charKillsNeed}
	g.progress.Owned = []string{"skin:lisa", "skin:tiburon_mascota"}
	g.syncShop()
	got := map[string]bool{}
	for _, it := range g.wardPickList() {
		if !it.locked {
			got[it.id] = true
		}
	}
	if len(got) != 3 || !got[""] || !got["homer"] || !got["lisa"] || got["tiburon_mascota"] {
		t.Fatalf("wearable: %v", got)
	}
	if o := optIDs(g.wardOptions("base")); len(o) != 3 || !o["homer"] || !o["lisa"] {
		t.Fatalf("the row Personaje must cycle the same characters: %v", o)
	}
	// DEBUG > Álbum: todo conseguido -> every character is wearable, none locked
	debugAlbum(&g.progress, 3)
	g.syncShop()
	all := g.wardPickList()
	for _, it := range all {
		if it.locked {
			t.Fatalf("todo conseguido but %s is locked", it.id)
		}
	}
	if len(all) != 1+charCount() {
		t.Fatalf("todo conseguido: %d cells", len(all))
	}
	// no killers or police cards are offered (only what was already bought as a skin)
	for _, it := range all {
		if strings.HasPrefix(it.id, "killer:") || strings.HasPrefix(it.id, "cop:") || strings.HasPrefix(it.id, "vip:") {
			t.Fatalf("%s is not a character", it.id)
		}
	}
}

// choosing puts the character on that player only, it is saved in the equip maps and survives a save/load
func TestWardrobeCharChooseSaveLoadTwoPlayers(t *testing.T) {
	g := wardCharGame(t)
	wardrobeNumPlayers = func(*Game) int { return 2 }
	debugAlbum(&g.progress, 3)
	g.syncShop()
	find := func(id string) wardPickItem {
		for _, it := range g.wardPickList() {
			if it.id == id {
				return it
			}
		}
		t.Fatalf("%s not in the list", id)
		return wardPickItem{}
	}
	if !g.wardPickChoose(0, find("homer")) || !g.wardPickChoose(1, find("lisa")) {
		t.Fatal("choose failed")
	}
	if g.progress.Equip["base"] != "homer" || g.progress.EquipP2["base"] != "lisa" {
		t.Fatalf("saved: %v / %v", g.progress.Equip, g.progress.EquipP2)
	}
	if g.playerSpecFor(0, "", "").Name != "homer" || g.playerSpecFor(1, "", "").Name != "lisa" {
		t.Fatal("each player must wear his own character")
	}
	if g.playerCS(0) == g.playerCS(1) || g.playerCS(0).Spec.Name != "homer" {
		t.Fatal("the game sets of the two players")
	}
	if g.as.playerCS == nil || g.as.playerCS.Spec.Name != "homer" {
		t.Fatal("applyShop must dress the protagonist of the party")
	}
	l := g.wardPickList()
	if l[g.wardPickIndex(l, 0)].id != "homer" || l[g.wardPickIndex(l, 1)].id != "lisa" {
		t.Fatal("the picker must mark what each one wears")
	}
	data, err := json.Marshal(g.progress)
	if err != nil {
		t.Fatal(err)
	}
	var p Progress
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if p.Equip["base"] != "homer" || p.EquipP2["base"] != "lisa" {
		t.Fatal("the choice is lost on load")
	}
	// back to the default with the first cell
	g.wardPickChoose(0, l[0])
	if g.equipOf(0, "base") != "" || g.equipOf(1, "base") != "lisa" {
		t.Fatal("taking it off changed the other player")
	}
	// a locked cell does nothing
	debugAlbum(&g.progress, 4)
	g.syncShop()
	if g.wardPickChoose(0, g.wardPickList()[1]) || g.equipOf(0, "base") != "" {
		t.Fatal("a locked character must not be worn")
	}
}

// an old save (no base, or a base that no longer exists or is not owned) gives the default look, no crash
func TestWardrobeCharOldSave(t *testing.T) {
	g := wardCharGame(t)
	for _, js := range []string{
		`{"unlocked":3,"wallet":50,"owned":["hat:caproja"],"equip":{"hat":"hat:caproja"}}`,
		`{"unlocked":3,"equip":{"base":"no_such_slug"},"equipP2":{"base":"homer"}}`,
	} {
		var p Progress
		if err := json.Unmarshal([]byte(js), &p); err != nil {
			t.Fatal(err)
		}
		g.progress = p
		g.syncShop()
		l := g.wardPickList()
		if g.wardPickIndex(l, 0) != 0 || g.wardPickIndex(l, 1) != 0 {
			t.Fatalf("%s: must fall back to the default cell", js)
		}
		def := playerSpec()
		if n := g.playerSpecFor(0, "", "").Name; n != def.Name {
			t.Fatalf("%s: player 1 is %q", js, n)
		}
		if n := g.playerSpecFor(1, "", "").Name; n == "homer" {
			t.Fatal("homer is not won: player 2 must not wear him")
		}
		if g.playerCS(0) == nil {
			t.Fatal("no character")
		}
		var pl Player
		pl.cs = g.playerCS(0)
		if a, _, _ := g.playerGait(&pl, AnimWalk, 0); a != AnimWalk {
			t.Fatal("the default player has no gait")
		}
	}
}

// the gait of the character worn goes with him (moonwalk for Michael Jackson)
func TestWardrobeCharGaitInGame(t *testing.T) {
	g := wardCharGame(t)
	slug := ""
	for s, gt := range gaitSlugs {
		if gt == "moon" && presetByName(s) != nil {
			slug = s
			break
		}
	}
	if slug == "" {
		t.Skip("no moonwalker in this build")
	}
	g.progress.Owned = []string{"skin:" + slug}
	g.syncShop()
	g.setEquipOf(0, "base", slug)
	pl := &Player{cs: g.playerCS(0), dir: DirLeft, moving: true}
	a, _, d := g.playerGait(pl, AnimWalk, 0)
	if a != AnimMoon || d != DirRight {
		t.Fatalf("%s walks with anim %d dir %v", slug, a, d)
	}
}

// the locked cells never show the name of the character
func TestWardrobeCharLockedHideNames(t *testing.T) {
	g := wardCharGame(t)
	g.progress.Album = map[string]int{"homer": albumKilled, "lisa": albumSeen | albumElim}
	g.progress.CharKills = map[string]int{"homer": charKillsNeed, "lisa": 3}
	g.syncShop()
	l := g.wardPickList()
	names := map[string]bool{}
	for _, it := range l {
		if it.locked {
			names[tr(it.card.nameES)] = true
		}
	}
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	for sel := 0; sel < len(l); sel += wpkCols * wpkRows {
		wardPickSel, wardPickScroll = sel, 0
		wardPickKeepInView(len(l))
		textRec = &textRecorder{}
		g.drawWardPick(img)
		for _, r := range textRec.texts {
			if names[r.s] {
				t.Fatalf("locked name %q drawn", r.s)
			}
		}
		textRec = nil
	}
	// lisa: locked, 3 of 5 kills -> the card's own line, with no name
	for i, it := range l {
		if it.id == "lisa" {
			wardPickSel = i
		}
	}
	d := g.wardPickDesc()
	if !strings.HasPrefix(d, "??? - ") || !strings.Contains(d, "2") || strings.Contains(d, tr("Lisa")) {
		t.Fatalf("locked description: %q", d)
	}
	if !strings.Contains(d, strings.TrimPrefix(g.charStatus(*l[wardPickSel].card), "")) {
		t.Fatal("the locked description must be the card status")
	}
}

// geometry: cells apart (>= 12 px), inside the panel, names inside their cell with padding; ES and EN, 1 and 2 players
func TestWardrobeCharPickGeometry(t *testing.T) {
	g := wardCharGame(t)
	pg, pe, pu := gameLang, editorLang, uiLang
	t.Cleanup(func() { gameLang, editorLang, uiLang, textRec = pg, pe, pu, nil })
	debugAlbum(&g.progress, 3)
	g.syncShop()
	l := g.wardPickList()
	// the cell rectangles
	for v := 0; v < wpkCols*wpkRows; v++ {
		x, y := wardPickCell(v)
		if x < shopPanX+12 || x+wpkCellW > 600 || y < shopTabTextY+16+12 || y+wpkCellH > shopPanY+shopPanH-12 || y+wpkCellH > backBtnY-12 {
			t.Fatalf("cell %d out of the area: %v,%v", v, x, y)
		}
		for u := v + 1; u < wpkCols*wpkRows; u++ {
			x2, y2 := wardPickCell(u)
			dx := maxF(x2-(x+wpkCellW), x-(x2+wpkCellW))
			dy := maxF(y2-(y+wpkCellH), y-(y2+wpkCellH))
			if maxF(dx, dy) < 12 {
				t.Fatalf("cells %d and %d closer than 12 px", v, u)
			}
		}
	}
	if wpkBarX < wpkX+3*wpkCellW+2*wpkGap+12 || wpkBarX+4 > 640-12 {
		t.Fatal("the scroll bar touches the grid or the mannequin")
	}
	// the longest names, in both languages and with one and two players
	longest := 0
	for i, it := range l {
		if len(it.name) > len(l[longest].name) {
			longest = i
		}
	}
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	for _, lang := range []string{"es", "en"} {
		gameLang, editorLang, uiLang = lang, lang, lang
		for _, np := range []int{1, 2} {
			wardrobeNumPlayers = func(*Game) int { return np }
			for _, sel := range []int{0, longest, len(l) - 1} {
				g.mode, g.shopTab = ModeShop, shopWardrobe
				wardKind, wardPick, wardPickSel, wardPickScroll = wardKindPlayer, true, sel, 0
				wardPickKeepInView(len(l))
				textRec = &textRecorder{}
				g.drawWardPick(img)
				g.drawWardPickInfo(img)
				texts := textRec.texts
				textRec = nil
				for _, r := range texts {
					if r.x != float64(int(r.x)) || r.y != float64(int(r.y)) || r.scale != float64(int(r.scale)) {
						t.Fatalf("[%s] %q at %v,%v scale %v: not whole", lang, r.s, r.x, r.y, r.scale)
					}
					if r.x >= 640 { // the description under the mannequin
						if r.x+r.w > shopPanX+shopPanW-12 || r.y < 390+6 || r.y+r.h > shopPanY+shopPanH-12 {
							t.Fatalf("[%s] info %q out of its place: %v,%v w %v", lang, r.s, r.x, r.y, r.w)
						}
						continue
					}
					inside := false
					for v := 0; v < wpkCols*wpkRows; v++ {
						x, y := wardPickCell(v)
						if r.x >= x+6 && r.x+r.w <= x+wpkCellW-6 && r.y >= y+4 && r.y+r.h <= y+wpkCellH-4 {
							inside = true
						}
					}
					if !inside {
						t.Fatalf("[%s] %dP: %q (%v,%v w %v h %v) not inside a cell with padding", lang, np, r.s, r.x, r.y, r.w, r.h)
					}
				}
			}
		}
	}
}

// the ATRÁS button keeps working on the row "Personaje": focused + A goes back (it does not open the grid), and with
// the grid open B / Esc / ATRÁS close the grid and stay in the wardrobe
func TestWardrobeCharBackButton(t *testing.T) {
	g := wardCharGame(t)
	pf, pi := backFocus, backInject
	t.Cleanup(func() { backFocus, backInject = pf, pi })
	g.mode, g.shopTab, wardKind, wardRow = ModeShop, shopWardrobe, wardKindPlayer, 0
	backFocus, backInject = true, false
	if g.wardPickTry("base") || wardPick {
		t.Fatal("A on the focused ATRÁS opened the grid")
	}
	backFocus, backInject = false, true
	if g.wardPickTry("base") || wardPick {
		t.Fatal("the pressed ATRÁS opened the grid")
	}
	backInject = false
	g.wardPickOpen()
	backInject = true
	if !g.updateWardPick() || wardPick || g.mode != ModeShop {
		t.Fatal("ATRÁS with the grid open must close the grid and stay in the wardrobe")
	}
}
