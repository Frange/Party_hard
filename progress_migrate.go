package main

import "fmt"

// ---------------------------------------------------------------------------
// Saves from older campaign layouts. Progress.CampaignRev tells which layout a save was written with (it is stamped
// on save); every step below moves a save one layout forward, in order, and each step runs only once.
//
// Rev 0 -> 1: the 3 new parties (birthday, ice hockey, 70s disco).
//   Old campaign: 21 parties, the space station was the 21st (index 20: stats "won_20", editor key c21_station).
//   New campaign: 24 parties, the station is the 24th (index 23) and the three new ones sit at 20, 21, 22.
//
// Rev 1 -> 2: ITALIA inserted right after the AUTOPISTA (index italyIdx = 19).
//   Every party from index 19 on (train station, birthday, hockey, 70s disco, station) moves one place forward:
//   won_19..won_23 become won_20..won_24 and the unlock frontier moves with them when it was past the highway.
//
// Rev 2 -> 3: the BARCO PIRATA inserted right before the space station (index pirateIdx = 24).
//   Only the station moves (24 -> 25): won_24 becomes won_25, and whoever had the station open (frontier 25, the
//   70s disco beaten) keeps it open and gets the pirate ship open too (frontier 26). Nothing else is keyed by the
//   index of a party: the album, the trophies, the stats and the medals go by name; "jp_unlocked" (the island),
//   the scenes ("seen_*") and the counters stay as they are.
//
// Rev 3 -> 4 (v1.4.0): the DISCOTECA (index 9) and the CINE (index 13) are RETIRED (campaign_retired.go). Nothing is
//   renumbered: they keep their index, their won_N, their editor key and their album/medal data. Only the frontier
//   moves: a save whose newest open party (Unlocked-1) was one of them gets the next party open (Unlocked+1), so the
//   party it was about to play is the next one of the list. Everything else stays exactly as it was. (This step reads
//   the retired parties at their rev-3 places, 9 and 13: the rev-5 insertion below moved them to 10 and 14.)
//
// Rev 4 -> 5 (v1.4.0): ALCALÁ DE HENARES inserted at index alcalaIdx = 6 (right after the TORRE EIFFEL, the 7th row
//   of the list). Every party from index 6 on moves one place forward: won_6..won_25 become won_7..won_26, and
//   whoever had beaten the TORRE EIFFEL (frontier past it) keeps everything open and gets ALCALÁ open too (not won).
//   Medals, album, trophies and the island go by name/icon: untouched.
//
// Rev 5 -> 6 (v1.4.0): GUADALAJARA inserted at index guadalajaraIdx = 15 (right after the retired CINE, the 14th row
//   of the list, before the CIRCUITO). Every party from index 15 on moves one place forward: won_15..won_26 become
//   won_16..won_27, and whoever had the CIRCUITO open (frontier past the CEMENTERIO, the retired CINE skipped) keeps
//   everything open and gets GUADALAJARA open too (not won). The retired parties (10 and 14) do not move.
//   Medals, album, trophies and the island go by name/icon: untouched.
//
// Rules (nobody loses anything, nobody gets a win they did not earn):
//   - won_N below the insertion point and the frontier below it stay exactly as they were.
//   - Whoever had already passed the party before the insertion keeps everything open that was open, and the new
//     parties open too (nothing that was playable gets locked).
//   - The new parties start as NOT won. "distinct" (the count of different parties won) is untouched.
//   - Editor corrections keep working: their keys are looked up with the old number too (campaign.go).
// ---------------------------------------------------------------------------

const (
	campaignRev      = 6  // the layout with 28 parties (ALCALÁ DE HENARES the 7th, GUADALAJARA the 16th), two of them retired
	oldCampaignN     = 21 // rev 0: parties before the three new ones
	oldStationIdx    = 20 // rev 0: index of the space station
	firstNewPartyIdx = 20 // rev 1: the birthday party: where the three new parties start
	rev1StationIdx   = 23 // rev 1: index of the space station (24 parties)
	rev2ItalyIdx     = 19 // rev 2: ITALIA, right after the AUTOPISTA (index 18 then)
	rev1CampaignN    = 24
	rev3PirateIdx    = 24 // rev 3: the BARCO PIRATA, right before the space station (then)
	rev2CampaignN    = 25 // rev 2: parties before the pirate ship
	rev3CampaignN    = 26 // rev 3 and 4: parties before ALCALÁ DE HENARES
	alcalaIdx        = 6  // rev 5: ALCALÁ DE HENARES, right after the TORRE EIFFEL
	rev5CampaignN    = 27 // rev 5: parties before GUADALAJARA
	guadalajaraIdx   = 15 // rev 6: GUADALAJARA, right after the retired CINE (the 14th row of the list)
	// where those parties are NOW (the current layout)
	italyIdx  = 21
	pirateIdx = 26
)

// rev3Retired: the retired parties at their places in the rev-3 layout (the DISCOTECA and the CINE).
var rev3Retired = map[int]bool{9: true, 13: true}

// migrateProgress brings a save written with an old layout to the current one. Idempotent.
func migrateProgress(p *Progress) {
	if p.CampaignRev < 1 {
		migrateRev1(p)
		p.CampaignRev = 1
	}
	if p.CampaignRev < 2 {
		migrateRev2(p)
		p.CampaignRev = 2
	}
	if p.CampaignRev < 3 {
		migrateRev3(p)
		p.CampaignRev = 3
	}
	if p.CampaignRev < 4 {
		migrateRev4(p)
		p.CampaignRev = 4
	}
	if p.CampaignRev < 5 {
		migrateRev5(p)
		p.CampaignRev = 5
	}
	if p.CampaignRev < 6 {
		migrateRev6(p)
		p.CampaignRev = 6
	}
	// Not a campaign layout: the POLICÍAS cards of the police types the career kills have not brought yet are cleared
	// (goalcops.go reconcileCopCards; only "cop:N" album keys, nothing else). Its own counter, CampaignRev stays.
	if p.CopRev < copRev {
		reconcileCopCards(p)
		p.CopRev = copRev
	}
}

// copRev: the revision of the POLICÍAS cards reconciliation (Progress.CopRev).
const copRev = 1

// migrateRev1: 21 parties -> 24 (the three new ones before the station).
func migrateRev1(p *Progress) {
	if p.Unlocked >= oldCampaignN { // the station was open: it stays open, and so do the new three
		p.Unlocked = rev1StationIdx + 1
	}
	if p.Stats != nil {
		moveStat(p.Stats, fmt.Sprintf("won_%d", oldStationIdx), fmt.Sprintf("won_%d", rev1StationIdx))
	}
}

// migrateRev2: 24 parties -> 25 (ITALIA inserted at italyIdx).
func migrateRev2(p *Progress) {
	if p.Unlocked > rev2ItalyIdx { // the highway had been beaten: the frontier moves with its party and ITALIA is open
		p.Unlocked = minI(p.Unlocked+1, rev1CampaignN+1)
	}
	if p.Stats != nil {
		for i := rev1CampaignN - 1; i >= rev2ItalyIdx; i-- { // from the end, so nothing is overwritten
			moveStat(p.Stats, fmt.Sprintf("won_%d", i), fmt.Sprintf("won_%d", i+1))
		}
	}
}

// migrateRev3: 25 parties -> 26 (the BARCO PIRATA inserted at pirateIdx, before the station).
func migrateRev3(p *Progress) {
	if p.Unlocked > rev3PirateIdx { // the station was open: it stays open, and the pirate ship opens too (not won)
		p.Unlocked = minI(p.Unlocked+1, rev2CampaignN+1)
	}
	if p.Stats != nil {
		for i := rev2CampaignN - 1; i >= rev3PirateIdx; i-- { // from the end, so nothing is overwritten
			moveStat(p.Stats, fmt.Sprintf("won_%d", i), fmt.Sprintf("won_%d", i+1))
		}
	}
}

// migrateRev4: the DISCOTECA and the CINE retired (v1.4.0): a frontier sitting on one of them moves to the next party.
// It reads the rev-3 layout (rev3Retired), not campaignLevels: rev 5 moved the retired parties one place.
func migrateRev4(p *Progress) {
	for !campaignCustom && p.Unlocked >= 1 && p.Unlocked < rev3CampaignN && rev3Retired[p.Unlocked-1] {
		p.Unlocked++
	}
}

// migrateRev5: 26 parties -> 27 (ALCALÁ DE HENARES inserted at alcalaIdx, right after the TORRE EIFFEL).
func migrateRev5(p *Progress) {
	if p.Unlocked > alcalaIdx { // the TORRE EIFFEL had been beaten: the frontier moves with its party, ALCALÁ opens
		p.Unlocked = minI(p.Unlocked+1, rev3CampaignN+1)
	}
	if p.Stats != nil {
		for i := rev3CampaignN - 1; i >= alcalaIdx; i-- { // from the end, so nothing is overwritten
			moveStat(p.Stats, fmt.Sprintf("won_%d", i), fmt.Sprintf("won_%d", i+1))
		}
	}
}

// rev5Retired: the retired parties at their places in the rev-5 layout (the DISCOTECA and the CINE; GUADALAJARA is
// inserted behind both, so they keep these places).
var rev5Retired = map[int]bool{10: true, 14: true}

// migrateRev6: 27 parties -> 28 (GUADALAJARA inserted at guadalajaraIdx, right after the retired CINE).
func migrateRev6(p *Progress) {
	// the CIRCUITO (rev-5 index 15) was open: the frontier moves with its party and GUADALAJARA opens. A frontier that
	// sat on the retired CINE (Unlocked 15) meant the CIRCUITO was open too (campaignFrontier skips the retired one).
	if p.Unlocked > guadalajaraIdx || !campaignCustom && p.Unlocked == guadalajaraIdx && rev5Retired[guadalajaraIdx-1] {
		p.Unlocked = minI(maxI(p.Unlocked, guadalajaraIdx+1)+1, rev5CampaignN+1)
	}
	if p.Stats != nil {
		for i := rev5CampaignN - 1; i >= guadalajaraIdx; i-- { // from the end, so nothing is overwritten
			moveStat(p.Stats, fmt.Sprintf("won_%d", i), fmt.Sprintf("won_%d", i+1))
		}
	}
}

// moveStat renames a stat (a zero or a missing value just disappears from the old name).
func moveStat(st map[string]int, from, to string) {
	v, ok := st[from]
	if !ok {
		return
	}
	delete(st, from)
	if v > 0 {
		st[to] = v
	}
}
