package main

import (
	"math/rand"
	"time"
)

// ---------------------------------------------------------------------------
// HOW OFTEN A VIP ZONE APPEARS AND WHICH VIP COMES OUT (vip.go has the zone itself).
//
//   - Each party (a level that is played; never STEALTH, never the Jurassic party; interludes and endings do not
//     load through loadLevel) rolls vipChance (1 in 5) with its OWN dice (vipRng): g.rng is not moved, like the
//     blackout and the killers. Tests seed it with vipReseed.
//   - Only levels where vipPlace finds room for the carpet can have it (the adaptive placement of vip.go; a level
//     without room counts as "no zone"). Eligible levels: see TestVIPEligibleLevels / docs/NOVEDADES.md.
//   - Pity: after vipPityAfter parties in a row without a zone, the next one gets it for sure (counter in memory, per
//     session; nothing is written to progress.json).
//   - Which VIP: the ones the player has NOT got yet (no ELIMINADO and no AMIGO bit in Progress.Album, nor killed in
//     this session). The one that suits the level's theme (vipPlan) is preferred half of the time when it is still
//     missing. When all 11 are got, any of them at random (repeats allowed). Old saves without Album = none got.
//   - -vipdemo forces a zone in every level (no dice).
// ---------------------------------------------------------------------------

const (
	vipChance    = 1.0 / 5 // chance per party that the level has a VIP zone
	vipPityAfter = 8       // parties in a row without a zone before the next one is certain
	vipAffinityP = 0.5     // chance to try the VIP that suits the theme first (when he is still missing)
)

var (
	vipRng    *rand.Rand // own dice (never g.rng)
	vipSeed   int64      // tests: non-zero = deterministic dice
	vipForce  bool       // tests: every party that has room gets a zone, like -vipdemo
	vipMisses int        // parties in a row without a zone (memory of the session)
)

func vipDice() *rand.Rand {
	if vipRng == nil {
		sd := vipSeed
		if sd == 0 {
			sd = time.Now().UnixNano()
		}
		vipRng = rand.New(rand.NewSource(sd))
	}
	return vipRng
}

// vipReseed: tests. Starts the dice again with that seed and clears the pity counter.
func vipReseed(seed int64) {
	vipSeed, vipRng, vipMisses = seed, nil, 0
}

// vipGot: the player has eliminated or helped this VIP (album bits), here or in an earlier session.
func vipGot(album map[string]int, id string) bool {
	return album["vip:"+id]&(albumKilled|albumHelped) != 0 || vipEliminated[id]
}

// vipPick: the VIP of a zone. Missing ones first (affinity preferred half of the time); all got = anyone.
func vipPick(r *rand.Rand, album map[string]int, title string) string {
	var left []string
	for _, d := range vipRegistry() {
		if !vipGot(album, d.id) {
			left = append(left, d.id)
		}
	}
	if len(left) == 0 {
		reg := vipRegistry()
		return reg[r.Intn(len(reg))].id
	}
	if aff, ok := vipPlan[title]; ok && r.Float64() < vipAffinityP {
		for _, id := range left {
			if id == aff {
				return id
			}
		}
	}
	return left[r.Intn(len(left))]
}

// vipDecide: the zone of one party ("" = none). misses is the pity counter. force skips the dice (-vipdemo).
func vipDecide(r *rand.Rand, def *LevelDef, album map[string]int, misses *int, stealth, force bool) string {
	if def == nil || def.Title == "" || stealth || vipOff {
		return ""
	}
	if !force && r.Float64() >= vipChance && *misses < vipPityAfter {
		*misses++
		return ""
	}
	id := vipPick(r, album, def.Title)
	if d, _ := vipPlace(def, id); d == nil {
		if !force {
			*misses++
		}
		return ""
	}
	*misses = 0
	return id
}

// vipRoll: loadLevel asks it before building the level.
func (g *Game) vipRoll(def *LevelDef) string {
	return vipDecide(vipDice(), def, g.progress.Album, &vipMisses, g.stealth, dbgVIPDemo != "" || vipForce)
}
