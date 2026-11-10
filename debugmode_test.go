package main

import "testing"

// every scene constant (scenes.go: up to sceneCount) has a line in debugScenes: a new scene cannot be forgotten.
func TestDebugScenesCoverAll(t *testing.T) {
	seen := map[int]bool{}
	for _, d := range debugScenes {
		if d.label == "" || d.scene < 0 || d.scene >= sceneCount {
			t.Errorf("bad entry %+v", d)
		}
		if seen[d.scene] {
			t.Errorf("scene %d listed twice", d.scene)
		}
		seen[d.scene] = true
	}
	for i := 0; i < sceneCount; i++ {
		if !seen[i] {
			t.Errorf("scene %d (%s) is missing from debugScenes (debugmode.go)", i, sceneList[i].title)
		}
	}
}

func TestDebugRowsFit(t *testing.T) {
	g := layerGame(t, false)
	rows := g.debugRows()
	if last := debugRow0 + float64(len(rows))*debugRowH; last > 498-12 {
		t.Errorf("%d rows end at y=%.0f (must be 12 px above the status line at 498)", len(rows), last)
	}
	for _, lang := range []string{"es", "en"} {
		old := uiLang
		uiLang = lang
		for _, r := range rows {
			if w := g.textW(r.label, 1.3); 156+w > 690 {
				t.Errorf("[%s] label %q is %.0f px wide (the value starts at x=700)", lang, r.label, w)
			}
			if v := r.val(); v != "" && 700+g.textW(v, 1.3) > 814 {
				t.Errorf("[%s] value %q of %q is closer than 6 px to the end of the row highlight (x=820)", lang, v, r.label)
			}
		}
		uiLang = old
	}
}

func TestDebugPoliceGroupsAndDaily(t *testing.T) {
	snapshotGlobals(t)
	progressFrozen, debugSnap, debugPrevFrozen, dbgPoliceGroups = false, nil, false, 0
	g := layerGame(t, false)
	g.progress.ChallengeLock = today()
	g.debugRows()
	step := debugSteps["Grupos de policía desbloqueados"]
	if step == nil {
		t.Fatal("no police groups row")
	}
	for i := 0; i < 9; i++ {
		step(1)
	}
	if dbgPoliceGroups != len(groupStat) || len(policeGroups()) != 5 {
		t.Errorf("groups %d / %d", dbgPoliceGroups, len(policeGroups()))
	}
	step(-1)
	if len(policeGroups()) != 4 {
		t.Errorf("4 expected, got %d", len(policeGroups()))
	}
	if g.dailyState() != dailyLost {
		t.Fatal("the challenge should be locked")
	}
	g.dbg.dailyFree = true
	if g.dailyState() != dailyOpen {
		t.Error("dailyFree must open the challenge")
	}
	g.debugEnd()
	if dbgPoliceGroups != 0 || g.dailyState() != dailyLost {
		t.Errorf("debugEnd left groups=%d daily=%d", dbgPoliceGroups, g.dailyState())
	}
}

// "Toda la tienda" + "TODOS los trofeos" cover the new things: 5 police groups, every killer, every shop item.
func TestDebugAllCoversNew(t *testing.T) {
	oldG, oldS, oldP, oldPG := progressFrozen, debugSnap, debugPrevFrozen, dbgPoliceGroups
	t.Cleanup(func() { progressFrozen, debugSnap, debugPrevFrozen, dbgPoliceGroups = oldG, oldS, oldP, oldPG })
	progressFrozen, debugSnap, debugPrevFrozen, dbgPoliceGroups = false, nil, false, 0
	g := layerGame(t, false)
	g.dbg.shop, g.dbg.levels, g.dbg.trophyMode = true, true, 1
	g.debugApply()
	t.Cleanup(g.debugEnd)
	for _, list := range shopCatalog {
		for _, it := range list {
			if !ownedID(it.id) {
				t.Errorf("shop item %s not owned", it.id)
			}
		}
	}
	if len(policeGroups()) != len(groupStat) {
		t.Errorf("police groups %d of %d (t_cop1..4)", len(policeGroups()), len(groupStat))
	}
	for _, f := range copForces { // the 4 groups come with the trophies / the shop, the local police always
		if !g.copGot(f.unit) && f.unit != unitPolice {
			t.Errorf("police force %s not unlocked", f.es)
		}
	}
	g.dbg.album = 3 // killers and characters come from the album row
	g.debugApply()
	for _, c := range albumCards() {
		if !g.albumUnlocked(c) {
			t.Errorf("card %s (kind %d) not won with album=all won", c.key, c.kind())
		}
	}
	if g.progress.Unlocked != campaignN {
		t.Errorf("unlocked %d of %d", g.progress.Unlocked, campaignN)
	}
}

// the row "Álbum": all won gives every card (304 characters, 50 VIPs, 27 killers, 5 cops), nothing empties them, and
// the real progress comes back at the end.
func TestDebugAlbumStates(t *testing.T) {
	oldG, oldS, oldP, oldPG := progressFrozen, debugSnap, debugPrevFrozen, dbgPoliceGroups
	t.Cleanup(func() { progressFrozen, debugSnap, debugPrevFrozen, dbgPoliceGroups = oldG, oldS, oldP, oldPG })
	progressFrozen, debugSnap, debugPrevFrozen, dbgPoliceGroups = false, nil, false, 0
	g := layerGame(t, false)
	real := g.progress.Album["x_real_marker"]
	g.progress.Album = map[string]int{"x_real_marker": 7}
	_ = real
	for _, k := range []int{cardChar, cardVIP, cardKiller, cardCop} {
		g.dbg.album = 3
		g.debugApply()
		if got, all := g.careerGot(k); got != all || all == 0 {
			t.Errorf("all won, kind %d: %d / %d", k, got, all)
		}
		g.dbg.album = 4
		g.debugApply()
		if got, _ := g.careerGot(k); got > 0 && k != cardKiller && k != cardCop && k != cardChar {
			t.Errorf("none, kind %d: %d won", k, got)
		}
	}
	g.dbg.album = 2
	g.debugApply()
	n := 0
	for _, c := range albumCards() {
		if g.albumSeenOnly(c) {
			n++
		}
	}
	if n == 0 {
		t.Error("all eliminated: no character shows ELIMINADO")
	}
	g.debugEnd()
	if g.progress.Album["x_real_marker"] != 7 || g.progress.Album["vip:"] != 0 || len(g.progress.Album) != 1 {
		t.Errorf("the real album did not come back: %v", g.progress.Album)
	}
}
