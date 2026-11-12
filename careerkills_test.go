package main

import (
	"fmt"
	"testing"
)

// ---- careerKills: one count of the career kills for every screen (goalstats.go, 2026-11-11) ----

// careerRowValue: the value of one row of the CARRERA page.
func careerRowValue(g *Game, label string) string {
	l, r := g.careerRows()
	for _, row := range append(l, r...) {
		if row[0] == label {
			return row[1]
		}
	}
	return ""
}

// The synthetic save of the report (447 kills, 63 trap and 13 poison stats; old CARRERA said 523): CARRERA, the
// trophy counters, the shop and the police unlocks now read the same 447.
func TestCareerKillsOneCount447Not523(t *testing.T) {
	g := goalsGame(t)
	p := copReportSave()
	if old := p.Stats["kills"] + p.Stats["traps"] + p.Stats["poison"]; old != 523 {
		t.Fatalf("fixture: the old sum is %d, want 523", old)
	}
	g.progress = p
	g.syncShop()
	if got := careerKills(&g.progress); got != 447 {
		t.Fatalf("careerKills = %d, want 447", got)
	}
	if v := careerRowValue(g, "Bajas totales"); v != "447" {
		t.Errorf("CARRERA shows %q total kills, want 447", v)
	}
	for _, tr := range trophyList {
		if tr.stat == "kills" && progressStat(&g.progress, tr.stat) != 447 {
			t.Errorf("trophy %s counts %d kills", tr.id, progressStat(&g.progress, tr.stat))
		}
	}
	if progressStat(&g.progress, "traps") != 63 || progressStat(&g.progress, "best") != 0 {
		t.Errorf("other counters must stay raw: traps %d", progressStat(&g.progress, "traps"))
	}
	if g.copTypeOpen(unitFBI) {
		t.Error("FBI open with 447 kills (needs 500)")
	}
	if careerKills(nil) != 0 || careerKills(&Progress{}) != 0 {
		t.Error("an empty save must count 0")
	}
}

// A poisoned guest is ONE kill in the career (poison.go adds "kills" and "poison" for the same death).
func TestCareerKillsPoisonNotTwice(t *testing.T) {
	g := goalsGame(t)
	snapshotGlobals(t)
	progressFrozen = true // goalsGame -> killerGame keeps the real progress out; doubly sure here
	g.progress.Stats = map[string]int{}
	gu := liveGuest(g, 0)
	if gu == nil {
		t.Fatal("no guest")
	}
	g.startPoison(gu)
	before := careerKills(&g.progress)
	for i := 0; i < 4000 && !gu.dead; i++ {
		g.updatePoison(gu, 1.0/60)
	}
	if !gu.dead {
		t.Fatal("the poisoned guest never died")
	}
	if d := careerKills(&g.progress) - before; d != 1 {
		t.Fatalf("a poison death added %d career kills, want 1 (stats %v)", d, g.progress.Stats)
	}
	if v := careerRowValue(g, "Bajas totales"); v != fmt.Sprint(careerKills(&g.progress)) {
		t.Errorf("CARRERA %q vs careerKills %d", v, careerKills(&g.progress))
	}
}

// Milestones already reached stay: a save whose CARRERA showed more than its kills (traps/poison) and that already
// won the FBI trophy / owns its reward keeps the FBI open after loading + migration and after a trophy check; the
// lower number never takes a trophy, a reward or a police type away.
func TestCareerKillsKeepsMilestones(t *testing.T) {
	g := goalsGame(t)
	snapshotGlobals(t)
	progressFrozen = true
	p := copReportSave()
	p.Stats["kills"] = 480 // old CARRERA: 480+63+13 = 556 (> 500); the FBI was won some other way before
	p.Trophies["t_cop1"] = true
	p.Owned = append(p.Owned, "police:1")
	p.Album["cop:1"] = albumKilled
	p.CopRev = 0 // the reconciliation runs again: it must not close the FBI
	migrateProgress(&p)
	if _, ok := p.Album["cop:1"]; !ok {
		t.Fatal("the reconciliation deleted the FBI card of an open type")
	}
	g.progress = p
	g.syncShop()
	if !g.copTypeOpen(unitFBI) || !policeTierOpen(unitFBI) {
		t.Fatal("FBI closed again with 480 kills and its trophy won")
	}
	nT, nO := len(g.progress.Trophies), len(g.progress.Owned)
	g.trophyDirty = true
	g.checkTrophies()
	if len(g.progress.Trophies) < nT || len(g.progress.Owned) < nO || !g.progress.Trophies["t_cop1"] {
		t.Fatalf("trophies %d->%d owned %d->%d", nT, len(g.progress.Trophies), nO, len(g.progress.Owned))
	}
	if g.copTypeOpen(unitMilitary) {
		t.Error("the army opened with 480 kills")
	}
	// and the counter itself opens at exactly 500, as before
	q := copReportSave()
	q.Stats["kills"] = 500
	own := map[string]bool{}
	if !copTypeOpenIn(&q, unitFBI, func(id string) bool { return own[id] }) {
		t.Error("FBI not open at 500 career kills")
	}
	q.Stats["kills"] = 499
	if copTypeOpenIn(&q, unitFBI, func(id string) bool { return own[id] }) {
		t.Error("FBI open at 499 career kills")
	}
}
