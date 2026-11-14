package main

// ---------------------------------------------------------------------------
// RETIRED PARTIES (v1.4.0, user: "quitar las pantallas CINE y DISCOTECA"). The DISCOTECA (kind "club", icon "disco",
// index 9) and the CINE (kind "cinema", icon "cinema", index 13) are no longer played, but they are NOT removed from
// campaignLevels: every index, every "won_N" stat, every editor key (cNN_icon) and every album/medal entry of a save
// stays exactly where it was, so no save of the testers breaks and nothing has to be renumbered. They are only:
//   - hidden from the party list (the list shows "slots": the parties that are played, numbered 1..campaignShown());
//   - skipped by the unlock chain (unlockNext, migrateRev4: a frontier that sat on one moves to the next party);
//   - skipped when the next party is started (campaignNext) and by the interludes (they go by slot: after the 5th,
//     10th and 15th party of the list);
//   - left out of the counts (careerLevelsDone, campaignShown) and of the random mode and the RETO DE HOY.
// The 70s disco (DISCO SETENTERA, icon "disco70") is NOT retired. A custom background set (assets_game.go) replaces the
// campaign wholesale and retires nothing (campaignCustom).
// ---------------------------------------------------------------------------

// campaignCustom: the levels on disk replaced the built-in campaign (rebuildLevels).
var campaignCustom bool

// campaignLen: the campaign parties (campaignN once the levels are built; the list itself before that).
func campaignLen() int {
	if campaignN > 0 {
		return campaignN
	}
	return len(campaignLevels)
}

// campaignRetired: campaign party i is retired (kept for the saves, never played).
func campaignRetired(i int) bool {
	return !campaignCustom && i >= 0 && i < len(campaignLevels) && campaignLevels[i].retired
}

// campaignSlots: the campaign indices the party list shows, in order.
func campaignSlots() []int {
	out := make([]int, 0, campaignLen())
	for i := 0; i < campaignLen(); i++ {
		if !campaignRetired(i) {
			out = append(out, i)
		}
	}
	return out
}

// campaignShown: how many campaign parties are played (the list, the counts "x/N").
func campaignShown() int {
	n := 0
	for i := 0; i < campaignLen(); i++ {
		if !campaignRetired(i) {
			n++
		}
	}
	return n
}

// campaignSlotOf: the position (0 based) of campaign party idx in the list, -1 if it is retired or not a campaign party.
func campaignSlotOf(idx int) int {
	if idx < 0 || idx >= campaignLen() || campaignRetired(idx) {
		return -1
	}
	s := 0
	for i := 0; i < idx; i++ {
		if !campaignRetired(i) {
			s++
		}
	}
	return s
}

// campaignAtSlot: the campaign index shown at list position s, -1 if there is none.
func campaignAtSlot(s int) int {
	for i := 0; i < campaignLen(); i++ {
		if campaignRetired(i) {
			continue
		}
		if s == 0 {
			return i
		}
		s--
	}
	return -1
}

// campaignNext: the party that follows idx (the retired ones are skipped). Past the campaign (random mode) it is idx+1.
func campaignNext(idx int) int {
	n := idx + 1
	for n < campaignLen() && campaignRetired(n) {
		n++
	}
	return n
}

// campaignFrontier: an Unlocked value whose newest open party (Unlocked-1) is retired moves on to the next party
// that is played (the one after it opens). Unlocked never goes past campaignLevels.
func campaignFrontier(u int) int {
	limit := campaignLen()
	for u >= 1 && u < limit && campaignLevels[u-1].retired && !campaignCustom {
		u++
	}
	return u
}

// campaignNumber: the number a campaign party shows ("N ITALIA", the newspaper): its position in the list, 1 based.
func campaignNumber(idx int) int {
	if s := campaignSlotOf(idx); s >= 0 {
		return s + 1
	}
	return idx + 1
}

// campaignMenuSel: the list row of the newest open party (where CONTINUAR puts the cursor).
func (g *Game) campaignMenuSel() int {
	s := campaignSlotOf(campaignFrontier(g.progress.Unlocked) - 1)
	return clampI(s, 0, maxI(campaignShown()-1, 0))
}

// retiredParties: the indices of the retired campaign parties.
func retiredParties() []int {
	var out []int
	for i, e := range campaignLevels {
		if e.retired {
			out = append(out, i)
		}
	}
	return out
}
