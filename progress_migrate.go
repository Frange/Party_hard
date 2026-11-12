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
// Rules (nobody loses anything, nobody gets a win they did not earn):
//   - won_N below the insertion point and the frontier below it stay exactly as they were.
//   - Whoever had already passed the party before the insertion keeps everything open that was open, and the new
//     parties open too (nothing that was playable gets locked).
//   - The new parties start as NOT won. "distinct" (the count of different parties won) is untouched.
//   - Editor corrections keep working: their keys are looked up with the old number too (campaign.go).
// ---------------------------------------------------------------------------

const (
	campaignRev      = 3  // the layout with 26 parties (the BARCO PIRATA before the ESTACIÓN ESPACIAL)
	oldCampaignN     = 21 // rev 0: parties before the three new ones
	oldStationIdx    = 20 // rev 0: index of the space station
	firstNewPartyIdx = 20 // rev 1: the birthday party: where the three new parties start
	rev1StationIdx   = 23 // rev 1: index of the space station (24 parties)
	italyIdx         = 19 // rev 2: ITALIA, right after the AUTOPISTA (index 18)
	rev1CampaignN    = 24
	pirateIdx        = 24 // rev 3: the BARCO PIRATA, right before the space station
	rev2CampaignN    = 25 // rev 2: parties before the pirate ship
)

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
	if p.Unlocked > italyIdx { // the highway had been beaten: the frontier moves with its party and ITALIA is open
		p.Unlocked = minI(p.Unlocked+1, rev1CampaignN+1)
	}
	if p.Stats != nil {
		for i := rev1CampaignN - 1; i >= italyIdx; i-- { // from the end, so nothing is overwritten
			moveStat(p.Stats, fmt.Sprintf("won_%d", i), fmt.Sprintf("won_%d", i+1))
		}
	}
}

// migrateRev3: 25 parties -> 26 (the BARCO PIRATA inserted at pirateIdx, before the station).
func migrateRev3(p *Progress) {
	if p.Unlocked > pirateIdx { // the station was open: it stays open, and the pirate ship opens too (not won)
		p.Unlocked = minI(p.Unlocked+1, rev2CampaignN+1)
	}
	if p.Stats != nil {
		for i := rev2CampaignN - 1; i >= pirateIdx; i-- { // from the end, so nothing is overwritten
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
