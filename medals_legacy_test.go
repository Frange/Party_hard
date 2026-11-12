package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// v1.0.0 saves (no MedalRev) keep EXACTLY the menu tier v1.0.0 gave them. v1.0.0 had 25 parties (every one but the
// BARCO PIRATA) and 70 trophies (every one but medalTrophy); the pirate ship and the new trophy only count from now
// on, as "new" under the 100 %. The expected tiers of the matrix were checked against the real v1.0.0 code (commit
// 07119f4): PH_V100_DUMP=<dir> writes every save of the matrix as JSON so the old code can read them.
// ---------------------------------------------------------------------------

// v100Trophies: the trophies of v1.0.0 (every one but the gold medals one); secret ones only when asked.
func v100Trophies(p *Progress, secret bool) {
	if p.Trophies == nil {
		p.Trophies = map[string]bool{}
	}
	for _, t := range trophyList {
		if t.id != medalTrophy && (secret || !secretTrophy[t.id]) {
			p.Trophies[t.id] = true
		}
	}
}

// v100Everything: every card of OBJETIVOS won and the whole shop bought (what COMPLETADO 100 % asks besides parties
// and trophies). The album and the shop are the same in v1.0.0 and now (only the trophy and the party are new).
func v100Everything(p *Progress) {
	if p.Album == nil {
		p.Album = map[string]int{}
	}
	for _, c := range albumCards() {
		switch c.kind() {
		case cardChar:
			p.Album[c.slug] |= albumKilled
		case cardKiller:
			p.Album["killer:"+c.slug] |= albumKilled
		case cardCop:
			p.Album[copKey(c.roleKind)] |= albumKilled
		default:
			p.Album[c.key] |= albumKilled
		}
	}
	for _, list := range shopCatalog {
		for _, it := range list {
			if shopBuyable(it) {
				p.Owned = append(p.Owned, it.id)
			}
		}
	}
}

type v100Case struct {
	name string
	save func() Progress // the file as v1.0.0 wrote it (CampaignRev 2, 25 parties)
	want int             // the menu tier v1.0.0 showed with it
}

func v100Matrix() []v100Case {
	all := func(trophies, secret, everything bool, extra map[string]int) func() Progress {
		return func() Progress {
			p := v100Save(25, upTo(25), extra)
			if trophies {
				v100Trophies(&p, secret)
			}
			if everything {
				v100Everything(&p)
			}
			return p
		}
	}
	island := map[string]int{"jp_unlocked": 1, "seen_interlude20": 1, "seen_end1": 1, "seen_end2": 1}
	return []v100Case{
		{"a_nothing", func() Progress { return v100Save(1, nil, nil) }, tierNormal},
		{"b_some", func() Progress { return v100Save(10, upTo(9), nil) }, tierNormal},
		{"b_24of25_alltrophies_everything", func() Progress {
			p := v100Save(25, upTo(24), nil)
			v100Trophies(&p, true)
			v100Everything(&p)
			return p
		}, tierNormal},
		{"c_all25_fewtrophies", all(false, false, false, nil), tierBronze},
		{"c_all25_everything_but_trophies", all(false, false, true, nil), tierBronze},
		{"d_all25_70trophies", all(true, true, false, nil), tierSilver},
		{"d_all25_trophies_nosecret_everything", all(true, false, true, nil), tierSilver},
		{"d_all25_70trophies_everything", all(true, true, true, nil), tierGold},
		{"e_island_bronze", all(false, false, false, island), tierBronze},
		{"e_island_gold", all(true, true, true, island), tierGold},
		{"f_station_only", func() Progress { return v100Save(25, []int{24}, nil) }, tierNormal},
		{"f_all25_station_no_trophies", func() Progress {
			p := v100Save(25, upTo(25), nil)
			p.Trophies = map[string]bool{}
			return p
		}, tierBronze},
	}
}

// v100Load: what newGame does with that file now: decode, migrate, sync the shop, the legacy medal step.
func v100Load(t *testing.T, g *Game, raw []byte) {
	t.Helper()
	p, _, err := decodeProgress(raw, true)
	if err != nil {
		t.Fatal(err)
	}
	migrateProgress(&p)
	g.progress = p
	g.syncShop()
	complFullCache.n, complCache.parts = 0, nil
	g.medalLegacyInit()
}

func medalLegacyGame(t *testing.T) *Game {
	g := medalGame(t)
	so := shopOwned
	t.Cleanup(func() { shopOwned = so; complFullCache.n, complCache.parts = 0, nil })
	g.dbg.tier = 0
	return g
}

func TestMedalV100SaveMatrix(t *testing.T) {
	g := medalLegacyGame(t)
	dump := os.Getenv("PH_V100_DUMP")
	for _, c := range v100Matrix() {
		pre := c.save()
		raw, _ := json.Marshal(pre)
		if dump != "" {
			if err := os.WriteFile(filepath.Join(dump, c.name+".json"), raw, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		v100Load(t, g, raw)
		if g.progress.MenuTier != c.want || g.menuTier() != c.want || g.progress.MedalRev != 1 {
			t.Errorf("%s: kept tier %d, menu %d (MedalRev %d), v1.0.0 showed %d", c.name, g.progress.MenuTier, g.menuTier(),
				g.progress.MedalRev, c.want)
			continue
		}
		if g.allTrophies() != (c.want >= tierSilver) {
			t.Errorf("%s: allTrophies %v with tier %d", c.name, g.allTrophies(), c.want)
		}
		parts := g.complParts()
		lines := g.complLines(parts)
		if (lines[0] == "100 %") != (c.want == tierGold) {
			t.Errorf("%s: COMPLETADO %q with tier %d", c.name, lines, c.want)
		}
		if c.want == tierGold { // the 100 % stays; what came after (the ship and the trophy) is "new"
			txt := strings.Join(lines[1:], " ")
			if !strings.HasPrefix(lines[1], tr("Nuevo:")) || !strings.Contains(txt, tr("1 trofeo")) ||
				!strings.Contains(txt, tr("1 fiesta")) {
				t.Errorf("%s: COMPLETADO with the kept gold: %q", c.name, lines)
			}
		}
		// idempotent: the step again, and a save + reload (the real signed file) + a new load
		g.medalLegacyInit()
		if g.progress.MenuTier != c.want || g.menuTier() != c.want {
			t.Errorf("%s: second run tier %d", c.name, g.menuTier())
		}
		path := filepath.Join(t.TempDir(), "progress.json")
		if err := writeProgressFile(path, g.progress); err != nil {
			t.Fatal(err)
		}
		q, found := readProgressFile(path)
		if !found || q.MedalRev != 1 || q.MenuTier != c.want || q.CampaignRev != campaignRev {
			t.Errorf("%s: saved MedalRev %d MenuTier %d rev %d", c.name, q.MedalRev, q.MenuTier, q.CampaignRev)
		}
		raw2, _ := json.Marshal(q)
		v100Load(t, g, raw2)
		if g.progress.MenuTier != c.want || g.menuTier() != c.want {
			t.Errorf("%s: after save and reload tier %d (kept %d)", c.name, g.menuTier(), g.progress.MenuTier)
		}
		if g.progress.Stats["won_25"] != pre.Stats["won_24"] || g.progress.Stats["won_24"] != 0 {
			t.Errorf("%s: station win moved wrong: %v", c.name, g.progress.Stats)
		}
	}
}

// The case of the report: the 25 parties and every trophy of v1.0.0 (the new one missing) is silver, and with
// everything else gold with COMPLETADO 100 %, exactly as in v1.0.0.
func TestMedalV100AllPartiesAllOldTrophies(t *testing.T) {
	g := medalLegacyGame(t)
	for _, everything := range []bool{false, true} {
		p := v100Save(25, upTo(25), nil)
		migrateProgress(&p)
		for _, tt := range trophyList {
			if tt.id != medalTrophy {
				p.Trophies[tt.id] = true
			}
		}
		want := tierSilver
		if everything {
			v100Everything(&p)
			want = tierGold
		}
		if p.Stats["won_24"] != 0 || p.Trophies[medalTrophy] {
			t.Fatal("the pirate ship or the new trophy came won from v1.0.0")
		}
		g.progress = p
		g.syncShop()
		complFullCache.n, complCache.parts = 0, nil
		g.medalLegacyInit()
		if g.progress.MenuTier != want || g.menuTier() != want {
			t.Fatalf("everything=%v: tier %d (kept %d), v1.0.0 gave %d", everything, g.menuTier(), g.progress.MenuTier, want)
		}
		if lines := g.complLines(g.complParts()); everything && lines[0] != "100 %" {
			t.Errorf("COMPLETADO lost: %q", lines)
		}
	}
}

// A save made by v1.1.0 (MedalRev 1) plays by the new rules: 26 parties and 71 trophies.
func TestMedalNewSaveNewRules(t *testing.T) {
	g := medalLegacyGame(t)
	mk := func(won int, medal, everything bool) Progress {
		p := Progress{Unlocked: campaignN, CampaignRev: campaignRev, MedalRev: 1, Stats: map[string]int{}, Trophies: map[string]bool{}}
		p.Stats = map[string]int{}
		for _, i := range upTo(won) {
			p.Stats[fmt.Sprintf("won_%d", i)] = 1
		}
		v100Trophies(&p, true)
		if medal {
			p.Trophies[medalTrophy] = true
		}
		if everything {
			v100Everything(&p)
		}
		return p
	}
	for _, c := range []struct {
		name              string
		won               int
		medal, everything bool
		want              int
	}{
		{"25 parties (no ship)", 25, true, true, tierNormal},
		{"26 parties, 70 trophies", 26, false, true, tierBronze},
		{"26 parties, 71 trophies", 26, true, false, tierSilver},
		{"26 parties, everything", 26, true, true, tierGold},
	} {
		g.progress = mk(c.won, c.medal, c.everything)
		g.syncShop()
		complFullCache.n, complCache.parts = 0, nil
		g.medalLegacyInit()
		if g.progress.MedalRev != 1 || g.menuTier() != c.want {
			t.Errorf("%s: tier %d, want %d", c.name, g.menuTier(), c.want)
		}
	}
	// a brand new game (no save at all) goes through the step once, gets nothing and is stamped
	g.progress = Progress{Unlocked: 1}
	g.syncShop()
	g.medalLegacyInit()
	if g.progress.MedalRev != 1 || g.progress.MenuTier != tierNormal || g.menuTier() != tierNormal {
		t.Errorf("new game: MedalRev %d tier %d", g.progress.MedalRev, g.menuTier())
	}
}
