package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Progress is how many of the fixed campaign levels are unlocked (1 = only
// the first). It persists next to the assets folder, same place/pattern as
// every other piece of on-disk state (loadAssetStore, VERSION file).
type Progress struct {
	Unlocked int               `json:"unlocked"`
	Stats    map[string]int    `json:"stats"`
	Trophies map[string]bool   `json:"trophies"`
	Wallet   int               `json:"wallet"`
	GuestMin int               `json:"guestMin"` // fewest guests at the start of a party (0 = 40)
	GuestMax int               `json:"guestMax"` // most guests at the start of a party (0 = 60)
	Owned    []string          `json:"owned"`
	Equip    map[string]string `json:"equip"`
}

func progressPath() string {
	root := assetsRoot(true)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "progress.json")
}

func loadProgress() Progress {
	p := Progress{Unlocked: 1}
	if noAssets {
		return p
	}
	path := progressPath()
	if path == "" {
		return p
	}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &p)
	}
	if p.Unlocked < 1 {
		p.Unlocked = 1
	}
	return p
}

func saveProgress(p Progress) {
	if noAssets || progressFrozen {
		return
	}
	path := progressPath()
	if path == "" {
		return
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0o644)
}

// unlockNext advances the campaign by one level, but only when the level
// just finished was the current frontier (replaying an earlier campaign
// level, or finishing a random-mode level, never skips anyone ahead).
func (g *Game) unlockNext() {
	if g.levelIdx >= campaignN || g.levelIdx != g.progress.Unlocked-1 {
		return
	}
	if g.progress.Unlocked < campaignN {
		g.progress.Unlocked++
		saveProgress(g.progress)
	}
}

// maxStart / guestMax: the fewest and the most guests a party starts with (40 and 60 by
// default, changeable in the settings; the number of each party is drawn between them).
// maxTotal adds some room on top for the people who arrive during the party.
func (g *Game) maxStart() int {
	if g.progress.GuestMin <= 0 {
		return 40
	}
	return clampI(g.progress.GuestMin, 10, 100)
}

func (g *Game) guestMax() int {
	hi := g.progress.GuestMax
	if hi <= 0 {
		hi = 60
	}
	return clampI(maxI(hi, g.maxStart()), 10, 100)
}

func (g *Game) maxTotal() int { return g.guestMax() + 25 }

// progressFrozen: while the DEBUG mode is in use nothing is written to disk, so the real game is untouched.
var progressFrozen bool

func cloneProgress(p Progress) Progress {
	c := p
	c.Stats = map[string]int{}
	for k, v := range p.Stats {
		c.Stats[k] = v
	}
	c.Trophies = map[string]bool{}
	for k, v := range p.Trophies {
		c.Trophies[k] = v
	}
	c.Owned = append([]string(nil), p.Owned...)
	c.Equip = map[string]string{}
	for k, v := range p.Equip {
		c.Equip[k] = v
	}
	return c
}
