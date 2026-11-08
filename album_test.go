package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// albumTestGame: the shared headless game with a private copy of the progress, frozen, restored afterwards.
func albumTestGame(t *testing.T) *Game {
	g := auditGame(t)
	prevP, prevF, prevTab, prevTwo := cloneProgress(g.progress), progressFrozen, g.trophyTab, twoPlayerMode
	progressFrozen = true
	g.progress.Album = nil
	t.Cleanup(func() {
		g.progress, progressFrozen, g.trophyTab, twoPlayerMode = prevP, prevF, prevTab, prevTwo
		g.albumSel, g.albumScroll, g.mode = 0, 0, ModeMenu
	})
	return g
}

func TestAlbumListsVIPAndSpecials(t *testing.T) {
	cards := albumCards()
	keys := map[string]bool{}
	for _, c := range cards {
		if keys[c.key] {
			t.Errorf("duplicated card %s", c.key)
		}
		keys[c.key] = true
	}
	if len(vipRegistry()) != 11 {
		t.Fatalf("expected 11 VIPs, got %d", len(vipRegistry()))
	}
	for _, d := range vipRegistry() {
		if !keys["vip:"+d.id] {
			t.Errorf("VIP %s missing", d.id)
		}
	}
	for _, k := range []string{"sam_sam", "cesar", "diego", "yoda", "shrek"} {
		if !keys[k] {
			t.Errorf("special %s missing", k)
		}
	}
	if len(cards) < 22 {
		t.Errorf("only %d cards", len(cards))
	}
}

func TestAlbumUnlockAndSilhouette(t *testing.T) {
	g := albumTestGame(t)
	vip := albumCards()[0]
	if g.albumUnlocked(vip) || g.albumCount() != 0 {
		t.Fatal("a fresh album must be locked")
	}
	g.albumMark("vip:"+vipRegistry()[0].id, albumSeen) // a VIP is not unlocked by sight
	if g.albumUnlocked(vip) {
		t.Error("a VIP must need a kill or help")
	}
	g.albumMark("vip:"+vipRegistry()[0].id, albumKilled)
	g.albumMark("diego", albumSeen)
	g.albumMark("vip:"+vipRegistry()[1].id, albumHelped)
	if !g.albumUnlocked(vip) || g.albumCount() != 3 {
		t.Errorf("count = %d, want 3", g.albumCount())
	}
	if g.progress.Stats["album"] != 3 {
		t.Errorf("album stat = %d", g.progress.Stats["album"])
	}
	g.albumMark("diego", albumSeen) // again: nothing changes
	if g.albumCount() != 3 {
		t.Error("count moved")
	}
	// the locked drawing (silhouette, ???) and the unlocked one do not panic
	dst := ebiten.NewImage(960, 540)
	for _, two := range []bool{false, true} {
		twoPlayerMode = two
		g.mode, g.trophyTab = ModeTrophies, tabAlbum
		g.drawTrophies(dst)
	}
}

func TestAlbumOldProgressLoads(t *testing.T) {
	var p Progress
	old := `{"unlocked":3,"stats":{"kills":4},"trophies":{"t_first":true},"wallet":10,"owned":[],"equip":{}}`
	if err := json.Unmarshal([]byte(old), &p); err != nil || p.Unlocked != 3 {
		t.Fatalf("old save: %v", err)
	}
	if p.Album != nil {
		t.Error("an old save has no album")
	}
	c := cloneProgress(p)
	c.Album = map[string]int{"x": 1}
	if p.Album != nil {
		t.Error("clone shares the map")
	}
	g := albumTestGame(t)
	g.progress = p
	if g.albumCount() != 0 {
		t.Error("old save must start empty")
	}
	g.albumMark("diego", albumKilled) // the nil map is created on demand
	if g.albumCount() != 1 {
		t.Error("mark on an old save failed")
	}
}

func TestAlbumNeverWritesWhenFrozen(t *testing.T) {
	g := albumTestGame(t)
	path := progressPath()
	before, _ := os.ReadFile(path)
	g.albumMark("yoda", albumSeen)
	g.albumMark("vip:king", albumKilled)
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Error("progress.json changed with progressFrozen")
	}
}

func TestAlbumScroll(t *testing.T) {
	g := albumTestGame(t)
	n := len(albumCards())
	if n <= albumCols*albumRows {
		t.Skip("everything fits")
	}
	g.albumSel = n - 1
	g.albumClamp()
	rows := (n + albumCols - 1) / albumCols
	if g.albumScroll != rows-albumRows {
		t.Errorf("scroll = %d, want %d", g.albumScroll, rows-albumRows)
	}
	g.albumSel = 0
	g.albumClamp()
	if g.albumScroll != 0 {
		t.Errorf("scroll = %d, want 0", g.albumScroll)
	}
	g.albumSel = albumCols * albumRows // first card of the 4th row
	g.albumClamp()
	if g.albumScroll != 1 {
		t.Errorf("scroll = %d, want 1", g.albumScroll)
	}
}

func TestSecretTrophiesHiddenUntilWon(t *testing.T) {
	g := albumTestGame(t)
	n := 0
	for _, tr := range trophyList {
		if !secretTrophy[tr.id] {
			continue
		}
		n++
		if !g.secretHidden(tr) {
			t.Errorf("%s must be hidden", tr.id)
		}
	}
	if n < 8 || n > 12 {
		t.Errorf("%d secret trophies", n)
	}
	g.progress.Trophies = map[string]bool{"s_mine": true}
	for _, tr := range trophyList {
		if tr.id == "s_mine" && g.secretHidden(tr) {
			t.Error("a won secret trophy must show")
		}
	}
	// the legend trophy does not wait for the secret ones
	g.progress.Trophies = map[string]bool{}
	for _, tr := range trophyList {
		if tr.id != legendTrophy && !secretTrophy[tr.id] {
			g.progress.Trophies[tr.id] = true
		}
	}
	if !g.othersDone() {
		t.Error("secret trophies must not block the legend trophy")
	}
}

func TestSecretTrophyFromStats(t *testing.T) {
	g := albumTestGame(t)
	g.progress.Trophies = nil
	g.stat("minex", 1)
	g.checkTrophies()
	if !g.progress.Trophies["s_mine"] {
		t.Error("s_mine not won")
	}
	if len(g.toasts) == 0 || !g.toasts[len(g.toasts)-1].secret {
		t.Error("no secret toast")
	}
	g.toasts = nil
}

func TestSecretTrapBurst(t *testing.T) {
	g := albumTestGame(t)
	g.progress.Stats = map[string]int{}
	g.sec = secretState{}
	g.time = 10
	for i := 0; i < 3; i++ {
		g.time += 0.1
		g.secretTrapKill()
	}
	if g.progress.Stats["minex"] != 1 {
		t.Errorf("minex = %d", g.progress.Stats["minex"])
	}
	g.time += 5
	g.secretTrapKill()
	if g.progress.Stats["minex"] != 1 {
		t.Error("a lone kill must not count")
	}
}

// albumNearGame: a play session with a frozen, empty progress and one special guest we control.
func albumNearGame(t *testing.T) (*Game, *Guest, albumCard) {
	g := hudPlay(t, false)
	prevP, prevF := cloneProgress(g.progress), progressFrozen
	progressFrozen = true
	t.Cleanup(func() { g.progress, progressFrozen = prevP, prevF; g.toasts = nil })
	g.progress.Album = nil
	g.progress.Trophies = map[string]bool{}
	g.progress.Stats = map[string]int{}
	var card albumCard
	for _, c := range albumCards() {
		if !c.vip {
			card = c
			break
		}
	}
	var gu *Guest
	for _, x := range g.guests {
		if x != nil && !x.dead && x.cs != nil && x.cs.Spec != nil {
			gu = x
			break
		}
	}
	if gu == nil {
		t.Skip("no guest")
	}
	prevName, prevPos := gu.cs.Spec.Name, gu.pos
	t.Cleanup(func() { gu.cs.Spec.Name, gu.pos = prevName, prevPos })
	gu.cs.Spec.Name = card.slug
	for _, x := range g.guests { // nobody else counts
		if x != gu && x != nil && x.cs != nil && x.cs.Spec != nil && x.cs.Spec.Name != "" {
			n, o := x.cs.Spec.Name, x.cs.Spec
			o.Name = ""
			t.Cleanup(func() { o.Name = n })
		}
	}
	return g, gu, card
}

// A clean progress: five seconds of a level with the whole level on screen unlock no card.
func TestAlbumNotSeenFromFar(t *testing.T) {
	g, gu, _ := albumNearGame(t)
	gu.pos = Vec{g.p.pos.X + 400, g.p.pos.Y}
	for i := 0; i < 300; i++ {
		g.sec.scanT = 0
		g.albumScan(1.0 / 60)
	}
	if n := g.albumCount(); n != 0 {
		t.Fatalf("%d cards after 5 s with nobody near", n)
	}
}

// Near a player and with a clear line: the card is seen. A wall in between: not.
func TestAlbumSeenNearAndInSight(t *testing.T) {
	g, gu, card := albumNearGame(t)
	gu.pos = Vec{g.p.pos.X + 60, g.p.pos.Y}
	if !g.lvl.LOS(g.p.pos, gu.pos) {
		t.Skip("a wall at the start")
	}
	g.sec.scanT = 0
	g.albumScan(1.0 / 60)
	if g.progress.Album[card.key]&albumSeen == 0 {
		t.Fatal("a special guest 60 px away in plain sight must be seen")
	}
	// behind a wall
	g.progress.Album = nil
	wall := Rect{(g.p.pos.X + gu.pos.X) / 2, g.p.pos.Y - 40, 6, 80}
	g.lvl.sight = append(g.lvl.sight, wall)
	t.Cleanup(func() { g.lvl.sight = g.lvl.sight[:len(g.lvl.sight)-1] })
	g.sec.scanT = 0
	g.albumScan(1.0 / 60)
	if g.albumCount() != 0 {
		t.Fatal("a wall between them: not seen")
	}
}

// The secret trophy needs 15 DISTINCT cards in the progress, not a stale counter.
func TestAlbumSecretNeedsFifteenDistinct(t *testing.T) {
	g, _, _ := albumNearGame(t)
	g.progress.Stats["album"] = 99 // stale counter from older saves
	g.trophyDirty = true
	g.checkTrophies()
	if g.progress.Trophies["s_album15"] {
		t.Fatal("won with 0 cards")
	}
	g.progress.Album = map[string]int{}
	for i, c := range albumCards() {
		if i < 14 {
			g.progress.Album[c.key] = albumKilled
		}
	}
	g.trophyDirty = true
	g.checkTrophies()
	if g.progress.Trophies["s_album15"] {
		t.Fatal("won with 14 cards")
	}
	g.progress.Album[albumCards()[14].key] = albumKilled
	g.trophyDirty = true
	g.checkTrophies()
	if !g.progress.Trophies["s_album15"] {
		t.Fatal("not won with 15 cards")
	}
}

// Toasts wait (queued) while the level hint plate or the newspaper is on screen.
func TestToastsWaitForHintAndNewspaper(t *testing.T) {
	hintRestore(t)
	g, _, _ := albumNearGame(t)
	g.lh.reset(g)
	g.mode = ModePlay
	g.sb.t = sbDur + 1
	g.updateLevelHint(0.1)
	if !g.levelHintActive() {
		t.Fatal("the hint plate must be on after the start marker")
	}
	g.toasts = []trophyToast{{name: "x", reward: "y", secret: true}}
	for i := 0; i < 120; i++ {
		g.updateToasts(1.0 / 60)
	}
	if g.toasts[0].t != 0 || !g.toastHeld() {
		t.Fatalf("toast ran during the hint: t=%v", g.toasts[0].t)
	}
	g.lh.closed = true
	g.updateToasts(0.5)
	if g.toasts[0].t != 0.5 {
		t.Fatalf("toast did not start after the hint: t=%v", g.toasts[0].t)
	}
	// newspaper front page
	g.toasts = []trophyToast{{name: "x", reward: "y"}}
	prevMode := g.mode
	t.Cleanup(func() { g.mode = prevMode })
	g.mode = ModeLevelDone
	g.updateToasts(1)
	if g.toasts[0].t != 0 {
		t.Fatal("toast ran over the newspaper")
	}
	g.mode = ModePlay
	g.updateToasts(1)
	if g.toasts[0].t != 1 {
		t.Fatal("toast did not run after the newspaper")
	}
}

// No name or role printed on a card is cut with a dot, in Spanish or in English (real measured width).
func TestAlbumCardNamesFit(t *testing.T) {
	g := auditGame(t)
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	loadLanguages()
	for _, lg := range []string{"es", "en"} {
		uiLang = lg
		for _, c := range albumCards() {
			for _, s := range []string{c.cardName(), c.roleES} {
				if w := g.textW(tr(s), 1); w > albumCardW-8 {
					t.Errorf("%s card %s: %q is %.0f px wide (max %.0f)", lg, c.key, tr(s), w, albumCardW-8)
				}
				if got := g.fitText(s, 1, albumCardW-8); got != tr(s) {
					t.Errorf("%s card %s: %q would be cut to %q", lg, c.key, tr(s), got)
				}
			}
		}
	}
}
