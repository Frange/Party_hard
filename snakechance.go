package main

import (
	"math/rand"
	"time"
)

// How often a snake shows up: not in every party. Same pattern as the VIP zone (vipchance.go): each party rolls
// snakeChance (1 in 4) with its OWN dice (g.rng is not moved) and after snakePityAfter parties in a row without
// one the next party gets it for sure. The counter lives in memory only (nothing is written to progress.json).
// -snakenow still forces one.
const (
	snakeChance    = 1.0 / 4
	snakePityAfter = 6
)

var (
	snakeRng    *rand.Rand
	snakeSeed   int64 // tests: non-zero = deterministic dice
	snakeMisses int   // parties in a row without a snake
)

func snakeDice() *rand.Rand {
	if snakeRng == nil {
		sd := snakeSeed
		if sd == 0 {
			sd = time.Now().UnixNano()
		}
		snakeRng = rand.New(rand.NewSource(sd))
	}
	return snakeRng
}

// snakeReseed: tests. Starts the dice again with that seed and clears the pity counter.
func snakeReseed(seed int64) { snakeSeed, snakeRng, snakeMisses = seed, nil, 0 }

// snakeDecide: does this party get a snake? misses is the pity counter.
func snakeDecide(r *rand.Rand, misses *int) bool {
	if r.Float64() < snakeChance || *misses >= snakePityAfter {
		*misses = 0
		return true
	}
	*misses++
	return false
}
