package main

import (
	"math/rand"
	"time"
)

// How often a tank crosses the road: at most ONE per party and on average in 1 party of 5. Same pattern as the snakes
// (snakechance.go) and the VIP zone (vipchance.go): each party that has a street with traffic rolls tankChance with
// its OWN dice (g.rng is not moved), never two parties in a row (the roll is higher to make up for it: 0.25 with no
// two in a row gives 1 in 5 overall) and after tankPityAfter parties in a row without one the next one gets it for
// sure. The counters live in memory only (nothing is written to progress.json). -fastevents still forces it.
const (
	tankChance    = 0.25
	tankPityAfter = 9
	tankSpawnP    = 0.3 // once the party has a tank due: chance that each vehicle of the road is the tank
)

var (
	tankRng    *rand.Rand
	tankSeed   int64 // tests: non-zero = deterministic dice
	tankMisses int   // parties in a row without a tank
	tankLast   bool  // the previous party had one
)

func tankDice() *rand.Rand {
	if tankRng == nil {
		sd := tankSeed
		if sd == 0 {
			sd = time.Now().UnixNano()
		}
		tankRng = rand.New(rand.NewSource(sd))
	}
	return tankRng
}

// tankReseed: tests. Starts the dice again with that seed and clears the counters.
func tankReseed(seed int64) { tankSeed, tankRng, tankMisses, tankLast = seed, nil, 0, false }

// tankDecide: does this party get a tank? misses is the pity counter, last whether the previous party had one.
func tankDecide(r *rand.Rand, misses *int, last *bool) bool {
	yes := !*last && (r.Float64() < tankChance || *misses >= tankPityAfter)
	if yes {
		*misses = 0
	} else {
		*misses++
	}
	*last = yes
	return yes
}
