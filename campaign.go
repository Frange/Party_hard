package main

import "fmt"

// ---------------------------------------------------------------------------
// The 25 fixed campaign scenarios, always in this order, unlocked one at a
// time by finishing the previous one (see progress.go). Everything else
// (extraLevels below, plus a second roll of every campaign kind with a
// different seed) only feeds the free "random" mode and never gates
// progress.
// ---------------------------------------------------------------------------

type campaignEntry struct {
	level func() *LevelDef // set for the 4 hand-authored levels
	kind  string           // set for generated levels (goes through genNamed)
	seed  int64
	title string
	icon  string
}

var campaignLevels = []campaignEntry{
	{level: level1, icon: "house"},
	{level: level4, icon: "beach"},
	{level: level3, icon: "neon"},
	{level: level5, icon: "ranch"},
	{kind: "bullring", seed: 11, title: "PLAZA DE TOROS", icon: "bull"},
	{kind: "paris", seed: 83, title: "TORRE EIFFEL", icon: "eiffel"},
	{kind: "yacht", seed: 149, title: "CRUCERO NOCTURNO", icon: "boat"},
	{kind: "funfair", seed: 107, title: "PARQUE DE ATRACCIONES", icon: "funfair"},
	{kind: "stadium", seed: 59, title: "ESTADIO", icon: "stadium"},
	{kind: "club", seed: 23, title: "DISCOTECA", icon: "disco"},
	{kind: "festival", seed: 95, title: "FESTIVAL", icon: "festival"},
	{kind: "skatepark", seed: 211, title: "SKATE PARK", icon: "skate"},
	{kind: "graveyard", seed: 311, title: "CEMENTERIO", icon: "grave"},
	{kind: "cinema", seed: 47, title: "CINE", icon: "cinema"},
	{kind: "circuit", seed: 223, title: "CIRCUITO", icon: "circuit"},
	{kind: "ski", seed: 353, title: "ESTACIÓN DE ESQUÍ", icon: "snow"},
	{level: level2, icon: "roof"},
	{kind: "casino", seed: 397, title: "CASINO", icon: "casino"},
	{kind: "highway", seed: 131, title: "AUTOPISTA", icon: "highway"},
	// ITALIA goes right after the highway (italy.go); saves of the 24-party layout are moved by progress_migrate.go
	{kind: "italy", seed: 509, title: "ITALIA", icon: "italy"},
	{kind: "trainstation", seed: 71, title: "ESTACIÓN DE TREN", icon: "train"},
	{kind: "birthday", seed: 331, title: "FIESTA DE CUMPLEAÑOS", icon: "cake"},
	{kind: "hockey", seed: 173, title: "PISTA DE HOCKEY", icon: "hockey"},
	{kind: "disco70", seed: 257, title: "DISCO SETENTERA", icon: "disco70"},
	// RULE: the last two parties are ALWAYS the space station and then the secret island (jurassic.go, not listed here).
	{kind: "station", seed: 239, title: "ESTACIÓN ESPACIAL", icon: "station"},
}

// legacyStationKey: the editor override name of the space station in the 21-party campaign.
const legacyStationKey = "c21_station"

// campaignIdx: the position (0 based) of the campaign party with that icon, -1 if there is none.
func campaignIdx(icon string) int {
	for i, e := range campaignLevels {
		if e.icon == icon {
			return i
		}
	}
	return -1
}

// campaignKeyRev1: the editor key that party i had in the 24-party layout (before ITALIA was inserted at
// italyIdx), "" for ITALIA itself and for the parties that did not move.
func campaignKeyRev1(i int) string {
	if i <= italyIdx || i >= len(campaignLevels) {
		return ""
	}
	return fmt.Sprintf("c%02d_%s", i, campaignLevels[i].icon)
}

// campaignN: how many campaign levels built successfully. Filled in by
// builtinLevels(); the menu and the unlock logic use it to know where the
// fixed campaign block ends and the random pool begins.
var campaignN int

func resolveCampaign(e campaignEntry) *LevelDef {
	if e.level != nil {
		d := e.level()
		resolveOverlaps(d)
		addMurray(d)
		return d
	}
	d := genNamed(e.kind, e.seed, e.title)
	addMurray(d)
	return d
}

func builtinLevels() []*LevelDef {
	var out []*LevelDef
	for i, e := range campaignLevels {
		if d := resolveCampaign(e); d != nil {
			d.Icon = e.icon
			la := levelOverrides[campaignKey(i)]
			if la == nil { // an editor correction saved before ITALIA was inserted keeps the old number in its key
				if k := campaignKeyRev1(i); k != "" {
					la = levelOverrides[k]
				}
			}
			if la == nil && e.icon == "station" { // before the 3 new parties the station was the 21st (c21_station)
				la = levelOverrides[legacyStationKey]
			}
			if la != nil {
				applyLevelOverride(d, la)
				addMurray(d) // an edited level may have moved things next to him: re-check his spot
			}
			out = append(out, d)
		}
	}
	campaignN = len(out)
	// Random-mode pool: a few venues that are not part of the campaign, plus
	// a second, differently-seeded roll of every generated campaign kind so
	// "random" does not just repeat the same 20 layouts.
	for _, e := range extraLevels {
		if d := genNamed(e.kind, e.seed, e.title); d != nil {
			out = append(out, d)
		}
	}
	for _, e := range campaignLevels {
		if e.kind == "" {
			continue
		}
		if d := genNamed(e.kind, e.seed+1013, e.title); d != nil {
			out = append(out, d)
		}
	}
	return out
}
