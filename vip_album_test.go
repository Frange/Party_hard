package main

import "testing"

func vipAlbumGame(t *testing.T) *Game {
	t.Helper()
	g := vipGame(t, "PARQUE DE ATRACCIONES", false)
	oldF := progressFrozen
	progressFrozen = true
	g.progress.Album = nil
	t.Cleanup(func() { progressFrozen = oldF })
	return g
}

// a VIP counts for the album only when he dies inside the VIP area
func TestVIPCardOnlyInsideTheZone(t *testing.T) {
	g := vipAlbumGame(t)
	boss := vipStaff(g, vipBoss)[0]
	key := "vip:" + boss.vipID
	if !g.vipInZone(boss.pos) {
		t.Fatal("the boss starts inside the area")
	}
	// outside: killed, but no card and no eliminated mark
	boss.pos = Vec{g.lvl.vip.trig.X - 500, g.lvl.vip.trig.Y - 500}
	g.p.pos = Vec{boss.pos.X + 8, boss.pos.Y}
	g.killGuard(boss)
	if !boss.dead {
		t.Fatal("the VIP should be dead")
	}
	if g.progress.Album[key] != 0 || vipEliminated[boss.vipID] {
		t.Fatalf("killed outside the zone must not count: album=%d eliminated=%v", g.progress.Album[key], vipEliminated[boss.vipID])
	}
	// inside: the card is won
	g2 := vipAlbumGame(t)
	b2 := vipStaff(g2, vipBoss)[0]
	g2.p.pos = Vec{b2.pos.X + 8, b2.pos.Y}
	g2.killGuard(b2)
	c := albumCard{key: "vip:" + b2.vipID, vip: true}
	if g2.progress.Album[c.key]&albumKilled == 0 || !g2.albumUnlocked(c) || !vipEliminated[b2.vipID] {
		t.Fatal("killed inside the zone must unlock the card")
	}
}

// saves keep what they had: the card bits of an old save still unlock the VIP
func TestVIPOldSaveKeepsItsCards(t *testing.T) {
	g := albumTestGame(t)
	g.progress.Album = map[string]int{"vip:king": albumKilled, "vip:pope": albumHelped}
	n := 0
	for _, c := range albumCards() {
		if c.vip && g.albumUnlocked(c) {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("old VIP cards = %d, want 2", n)
	}
}

// the three collections have their own tab, apart from the others
func TestVIPSectionIsSeparate(t *testing.T) {
	g := albumTestGame(t)
	want := map[int]int{tabVIPs: len(vipRegistry()), tabKillers: len(killerCatalog) + len(killerTrophy)}
	total := 0
	for tab := tabChars; tab < tabCount; tab++ {
		g.trophyTab = tab
		v := g.albumView()
		total += len(v)
		for _, c := range v {
			if c.kind() != goalTabs[tab].kind {
				t.Fatalf("tab %d shows a card of kind %d", tab, c.kind())
			}
		}
		if n, ok := want[tab]; ok && len(v) != n {
			t.Errorf("tab %d has %d cards, want %d", tab, len(v), n)
		}
	}
	if total != len(albumCards()) {
		t.Errorf("tabs show %d cards, the album has %d", total, len(albumCards()))
	}
	g.trophyTab = tabTrophies
	if len(g.albumView()) != 0 {
		t.Error("the trophy list has no cards")
	}
	// the tabs sit under the toast (it covers y 16-80), inside the panel, in order and apart from each other
	prev := 0.0
	for i, r := range g.goalTabRects() {
		if r[1] < 84 || r[0] < 232 || r[0]+r[2] > 900 || r[0] < prev {
			t.Errorf("tab %d at %v overlaps the toast, the close button or its neighbour", i, r)
		}
		prev = r[0] + r[2]
	}
}
