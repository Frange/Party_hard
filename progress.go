package main

import (
	"fmt"
	"path/filepath"
	"time"
)

// Progress is how many of the fixed campaign levels are unlocked (1 = only
// the first). It persists next to the assets folder, same place/pattern as
// every other piece of on-disk state (loadAssetStore, VERSION file).
type Progress struct {
	Unlocked int               `json:"unlocked"`
	Stats    map[string]int    `json:"stats"`
	Trophies map[string]bool   `json:"trophies"`
	Wallet   int               `json:"wallet"`
	GuestMin int               `json:"guestMin"` // fewest guests at the start of a party (0 = 45)
	GuestMax int               `json:"guestMax"` // most guests at the start of a party (0 = 60)
	Owned    []string          `json:"owned"`
	Equip    map[string]string `json:"equip"`             // what player 1 wears (and the party: uniforms, weapons)
	EquipP2  map[string]string `json:"equipP2,omitempty"` // what player 2 wears (old saves have none)
	Album    map[string]int    `json:"album,omitempty"`   // collector cards: key -> albumSeen|albumKilled|albumHelped bits (album.go; old saves have none)
	// RETO DE HOY (daily.go): the date YYYY-MM-DD of the day it was lost / completed; locked while it equals today's date.
	ChallengeLock string `json:"challengeLock,omitempty"`
	ChallengeDone string `json:"challengeDone,omitempty"`
	// KillerWait (goalkillers.go): parties left until the next killer is guaranteed after one was killed (0 = none; old saves have none).
	KillerWait int `json:"killerWait,omitempty"`
	// CampaignRev (progress_migrate.go): which campaign layout the save was written with (0 = the old 21 parties).
	CampaignRev int `json:"campaignRev,omitempty"`
	// Escapes (copsets.go): times the player was wanted (cuffs mark) and got away without being caught; every few of
	// them unlock a new police uniform (old saves have none: 0).
	Escapes int `json:"escapes,omitempty"`
	// CharKills (goalchars.go): times the player killed each character (key = slug). charKillsNeed of them win the
	// character. Old saves have none: a character they had already killed (albumKilled) stays won, nothing is lost.
	CharKills map[string]int `json:"charKills,omitempty"`
	// Started: when this game was begun (unix seconds). New games stamp it; old saves get the age of their file
	// (progress_store.go). Shown at the bottom of CARRERA.
	Started int64 `json:"started,omitempty"`
	// Medals (medals.go): the best medal of each campaign party, by its ICON (campaign.go; stable when a party is
	// inserted), 1 bronze 2 silver 3 gold. Old saves have none: no medals, nothing else changes.
	Medals map[string]int `json:"medals,omitempty"`
	// MenuTier (menutier.go): the best menu tier ever reached; the menu never goes back below it. MedalRev = 1 once the
	// save has been through the medals (a v1.0.0 save gets its tier computed once without the new trophy).
	MenuTier int `json:"menuTier,omitempty"`
	MedalRev int `json:"medalRev,omitempty"`
	// CopRev (progress_migrate.go): 1 once the POLICÍAS cards were reconciled with the career kills (a card marked for
	// a police type the kills had not brought yet is cleared, goalcops.go reconcileCopCards). Old saves have 0.
	CopRev int `json:"copRev,omitempty"`
}

// progressNow: the stamp of a game begun right now.
func progressNow() int64 { return time.Now().Unix() }

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
	if fp, found := readProgressFile(path); found { // signed save, old plain JSON, or the .bak (progress_store.go)
		p = fp
	} else {
		p.Started = progressNow() // no save: a new game begins
	}
	migrateProgress(&p)
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
	p.CampaignRev = campaignRev
	if p.Started == 0 {
		p.Started = progressNow()
	}
	writeProgressFile(path, p) // obfuscated + signed, atomic, keeps progress.json.bak (progress_store.go)
}

// unlockNext advances the campaign by one level, but only when the level
// just finished was the current frontier (replaying an earlier campaign
// level, or finishing a random-mode level, never skips anyone ahead).
func (g *Game) unlockNext() {
	if g.levelIdx >= campaignN || g.levelIdx != g.progress.Unlocked-1 {
		return
	}
	if g.progress.Unlocked < campaignN {
		g.progress.Unlocked = campaignFrontier(g.progress.Unlocked + 1) // a retired party is skipped (campaign_retired.go)
		saveProgress(g.progress)
	}
}

// maxStart / guestMax: the fewest and the most guests a party starts with (45 and 60 by
// default, changeable in the settings; the number of each party is drawn between them).
// guestMax is also the strict cap of guests present at once (see cap.go).
func (g *Game) maxStart() int {
	if g.progress.GuestMin <= 0 {
		return 45
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
	c.Album = nil
	if p.Album != nil {
		c.Album = map[string]int{}
		for k, v := range p.Album {
			c.Album[k] = v
		}
	}
	c.CharKills = nil
	if p.CharKills != nil {
		c.CharKills = map[string]int{}
		for k, v := range p.CharKills {
			c.CharKills[k] = v
		}
	}
	c.Medals = nil
	if p.Medals != nil {
		c.Medals = map[string]int{}
		for k, v := range p.Medals {
			c.Medals[k] = v
		}
	}
	c.EquipP2 = nil
	if p.EquipP2 != nil {
		c.EquipP2 = map[string]string{}
		for k, v := range p.EquipP2 {
			c.EquipP2[k] = v
		}
	}
	return c
}

// careerSinceLine: the bottom line of CARRERA, with the day and hour the game was begun (day/month/year, local).
func careerSinceLine(started int64) string {
	if started <= 0 {
		return "Toda tu carrera: se cuenta desde la primera fiesta."
	}
	t := time.Unix(started, 0).Local()
	return fmt.Sprintf("Toda tu carrera: desde el %d/%d/%d, %02d:%02d.", t.Day(), int(t.Month()), t.Year(), t.Hour(), t.Minute())
}
