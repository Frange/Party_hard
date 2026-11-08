package main

import "testing"

// Every prelude and ending is built with no sound tables at all (a muted stand-in, nil maps) and with the real ones,
// and neither panics (the audit used to die with "assignment to entry in nil map" in preBuild).
func TestPreludesAndEndingsBuildWithAndWithoutAudio(t *testing.T) {
	starts := map[string]func(*Game){
		"bad":    func(g *Game) { g.startEndingPre(false) },
		"island": func(g *Game) { g.startEndingPre(true) },
		"letter": func(g *Game) { g.startInvitePre() },
	}
	snds := map[string]func() *Sounds{
		"nil-maps": func() *Sounds { return &Sounds{muted: true} },
		"audio":    func() *Sounds { return newSounds(true) },
	}
	for sn, mk := range snds {
		for name, start := range starts {
			g := preGame(t, false)
			g.snd = mk()
			start(g)
			runStages(g, 70, endInvite, endFinal)
			if g.end == nil {
				continue
			}
			g.preBuild(g.end)
			g.endBuild(g.end)
			g.partyBuild(g.end)
			if sn == "audio" && g.snd.sfx["rex_roar"] == nil {
				t.Errorf("%s/%s: rex_roar not registered", sn, name)
			}
		}
	}
}
