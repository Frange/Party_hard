package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func charCard(t *testing.T, slug string) albumCard {
	t.Helper()
	for _, c := range albumCards() {
		if c.kind() == cardChar && c.key == slug {
			return c
		}
	}
	t.Fatalf("no card %s", slug)
	return albumCard{}
}

// A character is ELIMINADO from the first kill and CONSEGUIDO (usable in the wardrobe) at the fifth.
func TestCharFiveKillsWinHim(t *testing.T) {
	g := goalsGame(t)
	c := charCard(t, "homer")
	if g.charKillsLeft(c) != charKillsNeed || !strings.Contains(g.charDetailLine(c), "???") {
		t.Fatal("a fresh character: 5 kills left and not seen")
	}
	for i := 1; i < charKillsNeed; i++ {
		g.charKill("homer")
		if g.albumUnlocked(c) || ownedID("skin:homer") {
			t.Fatalf("won after %d kills", i)
		}
		if !g.albumSeenOnly(c) || g.albumBits(c)&albumElim == 0 || g.albumGroup(c) != 2 {
			t.Fatalf("kill %d: must be ELIMINADO (face shown, middle group)", i)
		}
		if left := g.charKillsLeft(c); left != charKillsNeed-i {
			t.Fatalf("kill %d: %d left", i, left)
		}
	}
	if !strings.Contains(g.charDetailLine(c), "Falta 1 muerte") {
		t.Errorf("line %q", g.charDetailLine(c))
	}
	g.charKill("homer")
	if !g.albumUnlocked(c) || !ownedID("skin:homer") || g.charKillsLeft(c) != 0 || g.albumGroup(c) != 3 {
		t.Fatal("the fifth kill wins him and the wardrobe sees him")
	}
	if !strings.Contains(g.charDetailLine(c), "Conseguido") {
		t.Errorf("line %q", g.charDetailLine(c))
	}
	g.charKill("homer") // keeps counting, nothing breaks
	if g.progress.CharKills["homer"] != charKillsNeed+1 {
		t.Errorf("count %d", g.progress.CharKills["homer"])
	}
	// after a reload the lookup is rebuilt from the album
	g.syncShop()
	if !ownedID("skin:homer") {
		t.Fatal("won character lost on reload")
	}
	g.charKill("not_a_character") // no card: ignored
	if _, ok := g.progress.CharKills["not_a_character"]; ok {
		t.Error("counted a guest with no card")
	}
}

// Old saves: a character killed once before the change (albumKilled, no CharKills) or bought stays won.
func TestCharOldSavesKeepEverything(t *testing.T) {
	g := goalsGame(t)
	var p Progress
	if err := json.Unmarshal([]byte(`{"unlocked":5,"album":{"bart":3,"lisa":1},"owned":["skin:marge"]}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.CharKills != nil {
		t.Fatal("an old save has no kill counters")
	}
	g.progress = p
	g.syncShop()
	for _, s := range []string{"bart", "marge"} {
		if c := charCard(t, s); !g.albumUnlocked(c) || g.charKillsLeft(c) != 0 || !ownedID("skin:"+s) {
			t.Errorf("%s must stay CONSEGUIDO", s)
		}
	}
	if c := charCard(t, "lisa"); g.albumUnlocked(c) || g.charKillsLeft(c) != charKillsNeed {
		t.Error("a seen character still needs 5 kills")
	}
	g.charKill("bart") // a won one just counts
	if !g.albumUnlocked(charCard(t, "bart")) {
		t.Fatal("lost")
	}
	// the counter is optional in the file and cloned
	b, _ := json.Marshal(Progress{Unlocked: 1})
	if strings.Contains(string(b), "charKills") {
		t.Error("charKills must be omitted when empty")
	}
	c := cloneProgress(g.progress)
	c.CharKills["bart"] = 99
	if g.progress.CharKills["bart"] == 99 {
		t.Error("clone shares CharKills")
	}
}

// The card pages list ??? first, then seen / partly killed, then the won ones; stable inside every group.
func TestAlbumViewOrder(t *testing.T) {
	g := goalsGame(t)
	g.trophyTab = tabChars
	var chars []string
	for _, c := range albumCards() {
		if c.kind() == cardChar && c.key != "homer" {
			chars = append(chars, c.key)
		}
	}
	// the ELIMINADO card comes EARLIER in the catalogue than the VISTO one: the view must invert them
	elimK, seenK := chars[0], chars[len(chars)-1]
	g.progress.Album = map[string]int{"homer": albumKilled | albumSeen, seenK: albumSeen, elimK: albumSeen | albumElim}
	g.progress.CharKills = map[string]int{elimK: 2}
	v := g.albumView()
	pos := map[string]int{}
	last := -1
	for i, c := range v {
		pos[c.key] = i
		gr := g.albumGroup(c)
		if gr < last {
			t.Fatalf("card %s (group %d) after group %d", c.key, gr, last)
		}
		last = gr
	}
	if pos["homer"] != len(v)-1 || !(pos[seenK] < pos[elimK] && pos[elimK] < pos["homer"]) {
		t.Errorf("order VISTO %d ELIMINADO %d homer %d of %d", pos[seenK], pos[elimK], pos["homer"], len(v))
	}
	if g.albumGroup(albumCardByKey(t, elimK)) != 2 || g.albumGroup(albumCardByKey(t, seenK)) != 1 {
		t.Error("groups: seen only = 1, eliminated = 2")
	}
	// stable: the ??? cards keep the catalogue order
	var cat []string
	for _, c := range albumCards() {
		if c.kind() == cardChar && g.albumGroup(c) == 0 {
			cat = append(cat, c.key)
		}
	}
	for i, k := range cat {
		if v[i].key != k {
			t.Fatalf("??? group not stable at %d: %s vs %s", i, v[i].key, k)
		}
	}
	// the killers too
	g.trophyTab = tabKillers
	g.progress.Album["killer:"+killerCatalog[0].slug] = albumKilled
	kv := g.albumView()
	if kv[len(kv)-1].slug != killerCatalog[0].slug {
		t.Error("a defeated killer goes to the end")
	}
	// the trophies: the won ones at the bottom
	g.progress.Trophies = map[string]bool{trophyList[0].id: true}
	tv := g.trophyView()
	if len(tv) != len(trophyList) || tv[len(tv)-1].id != trophyList[0].id || tv[0].id != trophyList[1].id {
		t.Error("trophy order")
	}
}

// Every character has his own description, none repeated, in Spanish and English, and the line fits.
func TestCharDescriptionsOwnAndFit(t *testing.T) {
	g := auditGame(t)
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	loadLanguages()
	seen := map[string]string{}
	seenEN := map[string]string{}
	for _, c := range albumCards() {
		if c.kind() != cardChar {
			continue
		}
		d := charDesc(c.slug)
		if d == "" {
			t.Errorf("%s has no description", c.slug)
			continue
		}
		if o, dup := seen[d]; dup {
			t.Errorf("%s and %s share %q", c.slug, o, d)
		}
		seen[d] = c.slug
		uiLang = "en"
		en := tr(d)
		uiLang = "es"
		if en == d {
			t.Errorf("%s: %q has no English", c.slug, d)
		}
		if o, dup := seenEN[en]; dup {
			t.Errorf("%s and %s share %q", c.slug, o, en)
		}
		seenEN[en] = c.slug
		if d == "Invitado famoso" || d == "Peque travieso" {
			t.Errorf("%s: generic description", c.slug)
		}
	}
	// the longest line (seen, 5 kills left) fits the 676 px of the detail line in both languages
	for _, lg := range []string{"es", "en"} {
		uiLang = lg
		for _, c := range albumCards() {
			if c.kind() != cardChar {
				continue
			}
			line := tr(c.nameES) + " - " + tr(charDesc(c.slug)) + " - " + tr("Conseguido: elígelo en el vestuario")
			l2 := tr(c.nameES) + " - " + tr(charDesc(c.slug)) + " - " + strings.Replace(tr("Faltan %d muertes para conseguirlo"), "%d", "5", 1)
			for _, s := range []string{line, l2} {
				if w := g.textW(s, 1); w > 676 {
					t.Errorf("%s: %q is %.0f px (max 676)", lg, s, w)
				}
			}
		}
	}
}

func albumCardByKey(t *testing.T, k string) albumCard {
	for _, c := range albumCards() {
		if c.key == k {
			return c
		}
	}
	t.Fatalf("no card %s", k)
	return albumCard{}
}
