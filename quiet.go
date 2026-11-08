package main

import (
	"flag"

	"github.com/hajimehoshi/ebiten/v2"
)

// quietMode: debug/capture/benchmark runs (or PH_QUIET=1) must not disturb the user: no focus theft,
// parked far outside the screen (never minimized: macOS stops giving drawables to minimized windows), no decoration, mouse passthrough, low CPU priority. -visible forces a normal window.
func quietMode(fs *flag.FlagSet, env string) bool {
	visible := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "visible" {
			visible = true
		}
	})
	if visible {
		return false
	}
	return env == "1" || debugFlagsActive(fs)
}

// quietSetup is called before RunGame (quiet mode only).
func quietSetup() {
	lowerPriority()
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowMousePassthrough(true)
	ebiten.SetWindowFloating(false)
	ebiten.SetWindowPosition(quietOffscreen, quietOffscreen)
}

// quietOffscreen: window origin far outside any monitor (the window still draws and can be captured).
const quietOffscreen = -30000

// quietOptions are the RunGameOptions for quiet mode.
func quietOptions() *ebiten.RunGameOptions {
	return &ebiten.RunGameOptions{InitUnfocused: true}
}

// quietGame parks the window off screen on its first Update (position is re-applied once the loop runs).
type quietGame struct {
	ebiten.Game
	done bool
}

func (q *quietGame) Update() error {
	if !q.done {
		q.done = true
		ebiten.SetWindowPosition(quietOffscreen, quietOffscreen)
	}
	return q.Game.Update()
}
