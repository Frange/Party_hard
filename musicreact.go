package main

// ---------------------------------------------------------------------------
// SOFT REACTIVE MUSIC. When the alert goes up (wanted, police called, a star, or the red phase of the STEALTH mode)
// the music of the party gets a little more present: its volume rises musicReactBoost (10 %) over musicReactFade
// seconds, and goes back to normal when things calm down. It does not touch the tempo: the mp3 player of the game
// cannot change speed without changing the pitch, so only the volume moves (a procedural rhythm layer was left out on
// purpose: over a song of unknown tempo it would clash). Only the music of a party is affected, never the interludes or
// the menus. musicReactEnabled switches the whole thing off.
// ---------------------------------------------------------------------------

const (
	musicReactBoost = 0.10 // extra volume at full alert (fraction of the normal volume)
	musicReactFade  = 2.0  // seconds to go from calm to alert and back
)

// musicReactEnabled: internal switch of the feature.
var musicReactEnabled = true

// musicReactTarget: 1 when the music should sound "alert", 0 otherwise (pure logic).
func (g *Game) musicReactTarget() float64 {
	if !musicReactEnabled || g.mode != ModePlay || g.lvl == nil {
		return 0
	}
	if g.wanted > 0 || g.policeCalled || g.stars > 0 || (g.stealth && g.sth.phase == sthRed) {
		return 1
	}
	return 0
}

// musicReactStep moves the current level k towards the target at 1/musicReactFade per second.
func musicReactStep(k, target, dt float64) float64 {
	d := dt / musicReactFade
	switch {
	case k < target:
		return minF(target, k+d)
	case k > target:
		return maxF(target, k-d)
	}
	return k
}

// musicReactVolume: the volume of the music given its normal volume and the level k (0..1).
func musicReactVolume(base, k float64) float64 { return base * (1 + musicReactBoost*k) }

func (g *Game) updateMusicReact(dt float64) {
	g.musicReact = musicReactStep(g.musicReact, g.musicReactTarget(), dt)
	if g.snd != nil {
		g.snd.SetReact(g.musicReact)
	}
}

// SetReact applies the reactive level to the song that is playing.
func (s *Sounds) SetReact(k float64) {
	if s.music == nil {
		return
	}
	base := 0.5
	if s.isFile {
		base = 0.6
	}
	s.music.SetVolume(musicReactVolume(base, k))
}
