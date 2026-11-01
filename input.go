package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// Gamepad support: a thin layer merged with the keyboard at each call site
// in game.go (there is no shared "virtual input" abstraction in this code
// base, so every site is touched individually). Only the first connected
// gamepad with a recognized standard layout is used — this is a single
// local player, so there is no need to merge several pads. The in-game
// editor stays mouse+keyboard only, it is not touched.
//
// Mapping (Xbox-style naming):
//   move          left stick or D-pad
//   run (hold)    right shoulder
//   attack/stab   A (RightBottom)
//   interact (E)  X (RightLeft)
//   dance (F)     Y (RightTop)
//   cancel/escape B (RightRight)
//   confirm       A (RightBottom)
//   menu up/down  D-pad up/down
// ---------------------------------------------------------------------------

const padDeadzone = 0.25

func activeGamepad() (ebiten.GamepadID, bool) {
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			return id, true
		}
	}
	return 0, false
}

func padConnected() bool {
	_, ok := activeGamepad()
	return ok
}

// padAxis returns the left stick / D-pad direction of the active gamepad,
// each roughly in [-1,1], or (0,0) if no gamepad is connected.
func padAxis() (dx, dy float64) {
	id, ok := activeGamepad()
	if !ok {
		return 0, 0
	}
	dx = ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
	dy = ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
	if dx > -padDeadzone && dx < padDeadzone {
		dx = 0
	}
	if dy > -padDeadzone && dy < padDeadzone {
		dy = 0
	}
	if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
		dx = -1
	}
	if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
		dx = 1
	}
	if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) {
		dy = -1
	}
	if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
		dy = 1
	}
	return dx, dy
}

func padPressed(b ebiten.StandardGamepadButton) bool {
	id, ok := activeGamepad()
	return ok && ebiten.IsStandardGamepadButtonPressed(id, b)
}

func padJustPressed(b ebiten.StandardGamepadButton) bool {
	id, ok := activeGamepad()
	return ok && inpututil.IsStandardGamepadButtonJustPressed(id, b)
}

// ---------------------------------------------------------------------------
// Rebindable actions. Every action has up to two keyboard keys and one gamepad
// button; defaults match the old fixed controls. Bindings persist in
// settings.json next to progress.json. Fixed shortcuts (1-9, R, G, M, F11, F2)
// are not rebindable.
// ---------------------------------------------------------------------------

type Action int

const (
	actUp Action = iota
	actDown
	actLeft
	actRight
	actRun
	actAttack
	actInteract
	actDance
	actBack
	actConfirm
	actEvent
	actImmune // legend powers (see powers.go)
	actFamous
	actKiller
	actPowerEvent
	actCount
)

var actionNames = [actCount]string{"Arriba", "Abajo", "Izquierda", "Derecha", "Correr", "Apuñalar", "Interactuar", "Bailar", "Atrás / cancelar", "Confirmar", "Provocar suceso", "Poder: inmunidad", "Poder: traer famosos", "Poder: enviar asesino", "Poder: sucesos"}
var actionIDs = [actCount]string{"up", "down", "left", "right", "run", "attack", "interact", "dance", "back", "confirm", "event", "immune", "famous", "killer", "powerevent"}

type Binding struct {
	Keys [2]ebiten.Key
	Pad  ebiten.StandardGamepadButton
	// unset slots: a key of -1 / pad of -1
}

type Bindings [actCount]Binding

const noKey = ebiten.Key(-1)
const noPad = ebiten.StandardGamepadButton(-1)

func defaultBindings() Bindings {
	var b Bindings
	set := func(a Action, k1, k2 ebiten.Key, pad ebiten.StandardGamepadButton) {
		b[a] = Binding{[2]ebiten.Key{k1, k2}, pad}
	}
	set(actUp, ebiten.KeyW, ebiten.KeyArrowUp, ebiten.StandardGamepadButtonLeftTop)
	set(actDown, ebiten.KeyS, ebiten.KeyArrowDown, ebiten.StandardGamepadButtonLeftBottom)
	set(actLeft, ebiten.KeyA, ebiten.KeyArrowLeft, ebiten.StandardGamepadButtonLeftLeft)
	set(actRight, ebiten.KeyD, ebiten.KeyArrowRight, ebiten.StandardGamepadButtonLeftRight)
	set(actRun, ebiten.KeyShiftLeft, ebiten.KeyShiftRight, ebiten.StandardGamepadButtonFrontBottomRight)
	set(actAttack, ebiten.KeySpace, ebiten.KeyJ, ebiten.StandardGamepadButtonRightBottom)
	set(actInteract, ebiten.KeyE, noKey, ebiten.StandardGamepadButtonRightLeft)
	set(actDance, ebiten.KeyF, noKey, ebiten.StandardGamepadButtonRightTop)
	set(actBack, ebiten.KeyEscape, noKey, ebiten.StandardGamepadButtonRightRight)
	set(actConfirm, ebiten.KeyEnter, ebiten.KeySpace, ebiten.StandardGamepadButtonRightBottom)
	set(actEvent, ebiten.KeyV, noKey, ebiten.StandardGamepadButtonFrontTopLeft)
	set(actImmune, ebiten.KeyI, noKey, ebiten.StandardGamepadButtonFrontTopRight)
	set(actFamous, ebiten.KeyO, noKey, ebiten.StandardGamepadButtonFrontBottomLeft)
	set(actKiller, ebiten.KeyK, noKey, ebiten.StandardGamepadButtonLeftStick)
	set(actPowerEvent, ebiten.KeyL, noKey, ebiten.StandardGamepadButtonRightStick)
	return b
}

var binds = defaultBindings()

func (a Action) held() bool {
	b := binds[a]
	for _, k := range b.Keys {
		if k != noKey && ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return b.Pad != noPad && padPressed(b.Pad)
}

func (a Action) just() bool {
	b := binds[a]
	for _, k := range b.Keys {
		if k != noKey && inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return b.Pad != noPad && padJustPressed(b.Pad)
}

func keyLabel(k ebiten.Key) string {
	if k == noKey {
		return "-"
	}
	switch k {
	case ebiten.KeyArrowUp:
		return "Flecha ↑"
	case ebiten.KeyArrowDown:
		return "Flecha ↓"
	case ebiten.KeyArrowLeft:
		return "Flecha ←"
	case ebiten.KeyArrowRight:
		return "Flecha →"
	case ebiten.KeySpace:
		return "Espacio"
	case ebiten.KeyShiftLeft:
		return "Shift izq."
	case ebiten.KeyShiftRight:
		return "Shift der."
	case ebiten.KeyEscape:
		return "Esc"
	case ebiten.KeyEnter:
		return "Enter"
	}
	return k.String()
}

var padLabels = map[ebiten.StandardGamepadButton]string{
	ebiten.StandardGamepadButtonRightBottom: "A", ebiten.StandardGamepadButtonRightRight: "B",
	ebiten.StandardGamepadButtonRightLeft: "X", ebiten.StandardGamepadButtonRightTop: "Y",
	ebiten.StandardGamepadButtonFrontTopLeft: "LB", ebiten.StandardGamepadButtonFrontTopRight: "RB",
	ebiten.StandardGamepadButtonFrontBottomLeft: "LT", ebiten.StandardGamepadButtonFrontBottomRight: "RT",
	ebiten.StandardGamepadButtonCenterLeft: "Select", ebiten.StandardGamepadButtonCenterRight: "Start",
	ebiten.StandardGamepadButtonLeftStick: "L3", ebiten.StandardGamepadButtonRightStick: "R3",
	ebiten.StandardGamepadButtonLeftTop: "D-pad ↑", ebiten.StandardGamepadButtonLeftBottom: "D-pad ↓",
	ebiten.StandardGamepadButtonLeftLeft: "D-pad ←", ebiten.StandardGamepadButtonLeftRight: "D-pad →",
}

func padLabel(b ebiten.StandardGamepadButton) string {
	if b == noPad {
		return "-"
	}
	if s, ok := padLabels[b]; ok {
		return s
	}
	return "Botón"
}

// actionHint is the name of the first key bound to an action, for on-screen hints.
func actionHint(a Action) string {
	for _, k := range binds[a].Keys {
		if k != noKey {
			return keyLabel(k)
		}
	}
	return padLabel(binds[a].Pad)
}

type savedBinding struct {
	Keys []string `json:"keys"`
	Pad  int      `json:"pad"`
}

func settingsPath() string {
	root := assetsRoot(true)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "settings.json")
}

func keyByName(name string) ebiten.Key {
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if k.String() == name {
			return k
		}
	}
	return noKey
}

func loadBindings() Bindings {
	b := defaultBindings()
	path := settingsPath()
	if noAssets || path == "" {
		return b
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return b
	}
	var saved map[string]savedBinding
	if json.Unmarshal(data, &saved) != nil {
		return b
	}
	for i, id := range actionIDs {
		sb, ok := saved[id]
		if !ok {
			continue
		}
		for s := 0; s < 2; s++ {
			b[i].Keys[s] = noKey
			if s < len(sb.Keys) {
				b[i].Keys[s] = keyByName(sb.Keys[s])
			}
		}
		b[i].Pad = ebiten.StandardGamepadButton(sb.Pad)
	}
	return b
}

func saveBindings() {
	path := settingsPath()
	if noAssets || path == "" {
		return
	}
	saved := map[string]savedBinding{}
	for i, id := range actionIDs {
		sb := savedBinding{Pad: int(binds[i].Pad)}
		for _, k := range binds[i].Keys {
			if k == noKey {
				sb.Keys = append(sb.Keys, "")
			} else {
				sb.Keys = append(sb.Keys, k.String())
			}
		}
		saved[id] = sb
	}
	if data, err := json.MarshalIndent(saved, "", "  "); err == nil {
		os.WriteFile(path, data, 0o644)
	}
}
