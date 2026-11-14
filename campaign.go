package main

import "fmt"

// ---------------------------------------------------------------------------
// The 28 fixed campaign scenarios (two retired, campaign_retired.go), always in this order, unlocked one at a
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
	// retired: the party stays in the list (its index, its won_N, its editor key and its album/medal data are kept, so no
	// save breaks) but it is not played any more: not shown in the list, skipped by the unlock chain, the counts and
	// the random mode (campaign_retired.go). v1.4.0: DISCOTECA and CINE.
	retired bool
}

var campaignLevels = []campaignEntry{
	{level: level1, icon: "house"},
	{level: level4, icon: "beach"},
	{level: level3, icon: "neon"},
	// BARCELONA (v1.4.0, barcelona.go) took the place of the old ranch: same index, same icon "ranch", same won_N,
	// medal and editor key, so the saves keep everything (nothing to migrate)
	{kind: "barcelona", seed: 733, title: barcelonaTitle, icon: "ranch"},
	{kind: "bullring", seed: 11, title: "PLAZA DE TOROS", icon: "bull"},
	// PARÍS (v1.4.0: it was called TORRE EIFFEL; the tower stays, the city came around it: paris.go). Same icon "eiffel".
	{kind: "paris", seed: 83, title: parisTitle, icon: "eiffel"},
	// ALCALÁ DE HENARES (v1.4.0, alcala.go): the 7th party of the list; saves of the 26-party layout are moved by
	// progress_migrate.go (CampaignRev 5)
	{kind: "alcala", seed: 577, title: alcalaTitle, icon: "alcala"},
	{kind: "yacht", seed: 149, title: "CRUCERO NOCTURNO", icon: "boat"},
	{kind: "funfair", seed: 107, title: "PARQUE DE ATRACCIONES", icon: "funfair"},
	{kind: "stadium", seed: 59, title: "ESTADIO", icon: "stadium"},
	{kind: "club", seed: 23, title: "DISCOTECA", icon: "disco", retired: true}, // retired in v1.4.0 (campaign_retired.go)
	{kind: "festival", seed: 95, title: "FESTIVAL", icon: "festival"},
	{kind: "skatepark", seed: 211, title: "SKATE PARK", icon: "skate"},
	{kind: "graveyard", seed: 311, title: "CEMENTERIO", icon: "grave"},
	{kind: "cinema", seed: 47, title: "CINE", icon: "cinema", retired: true}, // retired in v1.4.0 (campaign_retired.go)
	// GUADALAJARA (v1.4.0, guadalajara.go): the 14th row of the list (index 15, behind the retired CINE); saves of the
	// 27-party layout are moved by progress_migrate.go (CampaignRev 6)
	{kind: "guadalajara", seed: 691, title: guadalajaraTitle, icon: "guadalajara"},
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
	// the BARCO PIRATA goes right before the space station (pirate.go); saves of the 25-party layout are moved by
	// progress_migrate.go (CampaignRev 3)
	{kind: "pirate", seed: 613, title: "BARCO PIRATA", icon: "pirate"},
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

// rev5Index: where party i (current layout) was in the 27-party layout (rev 5, before GUADALAJARA was inserted at
// guadalajaraIdx); -1 for GUADALAJARA itself.
func rev5Index(i int) int {
	switch {
	case i == guadalajaraIdx:
		return -1
	case i > guadalajaraIdx:
		return i - 1
	}
	return i
}

// rev3Index: where party i (current layout) was in the 26-party layout (rev 3/4, before ALCALÁ DE HENARES was
// inserted at alcalaIdx and GUADALAJARA at guadalajaraIdx); -1 for those two.
func rev3Index(i int) int {
	j := rev5Index(i)
	switch {
	case j < 0 || j == alcalaIdx:
		return -1
	case j > alcalaIdx:
		return j - 1
	}
	return j
}

// campaignKeyRev1: the editor key that party i had in the 24-party layout (before ITALIA was inserted at
// rev2ItalyIdx and the BARCO PIRATA at rev3PirateIdx), "" for those two, for ALCALÁ and for the parties that did not
// move.
func campaignKeyRev1(i int) string {
	j := rev3Index(i)
	if j < 0 || j <= rev2ItalyIdx || j == rev3PirateIdx || i >= len(campaignLevels) {
		return ""
	}
	old := j - 1 // the parties behind ITALIA moved one place
	if j > rev3PirateIdx {
		old-- // ...and the ones behind the pirate ship one more
	}
	return fmt.Sprintf("c%02d_%s", old+1, campaignLevels[i].icon)
}

// campaignKeyRev2: the editor key that party i had in the 25-party layout (before the BARCO PIRATA was inserted at
// rev3PirateIdx), "" for the pirate ship, ALCALÁ and the parties that did not move.
func campaignKeyRev2(i int) string {
	j := rev3Index(i)
	if j < 0 || j <= rev3PirateIdx || i >= len(campaignLevels) {
		return ""
	}
	return fmt.Sprintf("c%02d_%s", j, campaignLevels[i].icon)
}

// campaignKeyRev3: the editor key that party i had in the 26-party layout (before ALCALÁ DE HENARES was inserted at
// alcalaIdx), "" for ALCALÁ, GUADALAJARA and the parties before ALCALÁ (their key did not change).
func campaignKeyRev3(i int) string {
	j := rev5Index(i)
	if j <= alcalaIdx || i >= len(campaignLevels) {
		return ""
	}
	return fmt.Sprintf("c%02d_%s", j, campaignLevels[i].icon)
}

// campaignKeyRev5: the editor key that party i had in the 27-party layout (before GUADALAJARA was inserted at
// guadalajaraIdx), "" for GUADALAJARA and the parties before it (their key did not change).
func campaignKeyRev5(i int) string {
	if i <= guadalajaraIdx || i >= len(campaignLevels) {
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
			for _, k := range []string{campaignKeyRev5(i), campaignKeyRev3(i), campaignKeyRev2(i), campaignKeyRev1(i)} { // an editor
				if la == nil && k != "" { // correction saved before GUADALAJARA, ALCALÁ, the BARCO PIRATA or ITALIA keeps the old number
					la = levelOverrides[k]
				}
			}
			if la == nil && e.icon == "station" { // before the 3 new parties the station was the 21st (c21_station)
				la = levelOverrides[legacyStationKey]
			}
			if la != nil && staleOverride(la, e.icon) {
				la = nil // an editor correction of the old ranch: its layout is not the one of BARCELONA (barcelona.go)
			}
			if la != nil {
				applyLevelOverride(d, la)
				addMurray(d) // an edited level may have moved things next to him: re-check his spot
			}
			addMirrorBall(d, e.icon) // the dance parties under a roof hang a mirror ball (discoball.go)
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
		if e.kind == "" || e.retired { // a retired party is not played in the random mode either
			continue
		}
		if d := genNamed(e.kind, e.seed+1013, e.title); d != nil {
			addMirrorBall(d, e.icon)
			out = append(out, d)
		}
	}
	return out
}
