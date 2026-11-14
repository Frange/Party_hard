package main

import (
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// CIRCUITO (v1.4.0, user): the race track gets race traffic: the ordinary ambient cars, flat out and very often.
// Nothing new kills: it is the same run-over rule as every road (game.go runOver). What is new is the WARNING: when a
// race car comes in, every player standing on the track (the asphalt band between the fences) gets a "¡COCHE!"
// bubble and the alert sound, so nobody is run over without notice (at these speeds the car needs ~2 s to reach the
// gap in the fences, where people cross).
// ---------------------------------------------------------------------------

const (
	raceGapMin   = 1.1   // seconds between race cars: raceGapMin + rand*raceGapVar
	raceGapVar   = 1.4   //
	raceSpeedMin = 250.0 // px/s (ordinary cars do 100)
	raceSpeedVar = 60.0  //
	raceTrackH   = 41.0  // half height of the asphalt band around CarFrom.Y (circuit: 244..326, centre 285)
)

// raceTrack: the circuit (fast traffic but not the highway, which has its own rhythm).
func raceTrack(d *LevelDef) bool {
	return d != nil && d.FastTraffic && !d.Highway
}

// raceSpeed: the cruising speed of one race car.
func raceSpeed(r *rand.Rand) float64 {
	return raceSpeedMin + r.Float64()*raceSpeedVar
}

// onRaceTrack: p stands on the asphalt of the circuit.
func (g *Game) onRaceTrack(p Vec) bool {
	return math.Abs(p.Y-g.lvl.Def.CarFrom.Y) < raceTrackH
}

// raceWarn: a race car has just come in; warn the players on the track.
func (g *Game) raceWarn(a *Ambient) {
	warned := false
	g.eachPlayer(func(p *Player) {
		if p.dead || p.hidden || !g.onRaceTrack(p.pos) {
			return
		}
		g.bubbleAt(Vec{p.pos.X, p.pos.Y - 30}, raceWarnText, 1.2)
		warned = true
	})
	if warned {
		g.snd.Play("alert")
	}
}

const raceWarnText = "¡COCHE!"
