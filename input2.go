package main

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// Input for two local players.
//
// Everything in the game asks "is this action held / just pressed?" through
// Action.held() / Action.just(). Those two look at inputP, the player whose
// turn it is (0 or 1): the game sets it while it runs each player's update
// (see Game.asPlayer). With one player inputP is always 0 and nothing changes.
//
// Two players:
//   keyboard 1 + keyboard 2  -> player 1 uses `binds` (W A S D, E, F, Space, Shift left),
//                               player 2 uses `binds2` (arrow keys, , . / and Shift right).
//                               A key bound by player 2 is ignored for player 1 (the arrow
//                               keys are the second key of the old bindings).
//   keyboard + gamepad       -> the player on the pad uses that pad, the other the keyboard.
//   two gamepads             -> player 1 the first pad, player 2 the second one.
// The device of each player lives in playerDev: -1 = keyboard, n >= 0 = the n-th pad.
// The 2-player menu (another part of the game) sets playerDev and rebinds binds2.
// ---------------------------------------------------------------------------

// inputP is the player whose input is being read right now (0 or 1).
var inputP int

// twoInput mirrors "two players are playing" for the input layer; the game sets it each frame
// while it is in play, so menus keep reading the first pad exactly as before.
var twoInput bool

// playerDev: -1 = keyboard, n >= 0 = n-th standard gamepad.
var playerDev = [2]int{-1, -1}

// binds2 are the keyboard (and pad) bindings of player 2.
var binds2 = defaultBindings2()

func defaultBindings2() Bindings {
	b := defaultBindings()
	set := func(a Action, k ebiten.Key, pad ebiten.StandardGamepadButton) {
		b[a] = Binding{[2]ebiten.Key{k, noKey}, pad}
	}
	set(actUp, ebiten.KeyArrowUp, ebiten.StandardGamepadButtonLeftTop)
	set(actDown, ebiten.KeyArrowDown, ebiten.StandardGamepadButtonLeftBottom)
	set(actLeft, ebiten.KeyArrowLeft, ebiten.StandardGamepadButtonLeftLeft)
	set(actRight, ebiten.KeyArrowRight, ebiten.StandardGamepadButtonLeftRight)
	set(actRun, ebiten.KeyShiftRight, ebiten.StandardGamepadButtonFrontBottomRight)
	set(actAttack, ebiten.KeyPeriod, ebiten.StandardGamepadButtonRightBottom)
	set(actInteract, ebiten.KeyComma, ebiten.StandardGamepadButtonRightLeft)
	set(actDance, ebiten.KeySlash, ebiten.StandardGamepadButtonRightTop)
	return b
}

// bindsFor are the bindings of a player.
func bindsFor(p int) *Bindings {
	if p == 1 && twoInput {
		return &binds2
	}
	return &binds
}

// keyOfPlayer2 says whether player 2 plays on the keyboard and uses key k for something.
func keyOfPlayer2(k ebiten.Key) bool {
	if !twoInput || playerDev[1] >= 0 {
		return false
	}
	for a := Action(0); a < actCount; a++ {
		if a > actDance {
			break
		}
		for _, bk := range binds2[a].Keys {
			if bk == k {
				return true
			}
		}
	}
	return false
}

// keyAllowed: player 1 does not read the keys player 2 plays with, and nobody reads the
// keyboard when the player is on a gamepad.
func keyAllowed(p int, k ebiten.Key) bool {
	if !twoInput {
		return true
	}
	if p == 1 {
		return playerDev[1] < 0
	}
	if keyOfPlayer2(k) {
		return false
	}
	return true
}

// padList: the standard gamepads that are connected, in a stable order.
func padList() []ebiten.GamepadID {
	out := append([]ebiten.GamepadID(nil), padIDsFn()...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// padOfPlayer is the gamepad a player plays with in the 2-player mode.
func padOfPlayer(p int) (ebiten.GamepadID, bool) {
	n := playerDev[p&1]
	if n < 0 {
		return 0, false
	}
	l := padList()
	if n >= len(l) {
		return 0, false
	}
	return l[n], true
}

// autoDevices picks the devices when the game starts a 2-player party: two pads if there are
// two, otherwise the second player takes the pad that exists, otherwise both share the keyboard.
func autoDevices() [2]int {
	return resolveDevices(devPref, len(padList())) // the player prefs (players.go) first, then the smart default
}

// smartDevices is the default when nothing is chosen for a player.
func smartDevices(n int) [2]int {
	switch {
	case n >= 2:
		return [2]int{0, 1}
	case n == 1:
		return [2]int{-1, 0}
	}
	return [2]int{-1, -1}
}

// justForPlayer is Action.just() for any player, regardless of inputP.
func (a Action) justFor(p int) bool {
	old := inputP
	inputP = p
	r := a.just()
	inputP = old
	return r
}

// heldFor is Action.held() for any player.
func (a Action) heldFor(p int) bool {
	old := inputP
	inputP = p
	r := a.held()
	inputP = old
	return r
}

var _ = inpututil.IsKeyJustPressed

// justEither is for the party-wide actions (the legend powers and the rare-event key): the
// action fires when either player presses it (with one player it is just()).
func (a Action) justEither() bool {
	if !twoInput {
		return a.just()
	}
	return a.justFor(0) || a.justFor(1)
}

// actionHintFor is actionHint for a player: the label of his own key (or button).
func actionHintFor(a Action, p int) string {
	if !twoInput {
		return actionHint(a)
	}
	b := binds
	if p == 1 {
		b = binds2
	}
	if playerDev[p&1] >= 0 { // on a gamepad: the button
		return padLabel(b[a].Pad)
	}
	for _, k := range b[a].Keys {
		if k != noKey {
			return keyLabel(k)
		}
	}
	return padLabel(b[a].Pad)
}
