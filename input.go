package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

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
// Mapping (standard layout, named as on a PlayStation pad; user, 2026-11-11):
//   move          left stick or D-pad
//   run (hold)    R1 (FrontTopRight)
//   attack/stab   X (RightBottom)
//   interact (E)  O, the circle (RightRight): pick a body up, set traps, hide, use things
//   dance (F)     Triángulo (RightTop)
//   pause / menu  START (CenterRight, fixed)
//   menus         X confirms, O goes back (actConfirm / actBack)
//   help          SELECT (CenterLeft, fixed)
//   minimap       Cuadrado (RightLeft, fixed, only while no action uses it)
// In a party O is "interact", so it does not open the pause there (only Esc / START do, pause.go).
// ---------------------------------------------------------------------------

const padDeadzone = 0.25

// Hardware access goes through these variables so that the tests can feed fake pads.
var (
	padIDsFn  = defaultPadIDs
	padHeldFn = ebiten.IsStandardGamepadButtonPressed
	padJustFn = inpututil.IsStandardGamepadButtonJustPressed
	padAxisFn = ebiten.StandardGamepadAxisValue
)

// defaultPadIDs lists the connected gamepads with a standard layout.
func defaultPadIDs() []ebiten.GamepadID {
	var out []ebiten.GamepadID
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			out = append(out, id)
		}
	}
	return out
}

func activeGamepad() (ebiten.GamepadID, bool) {
	if twoInput { // two players: each one has his own device (input2.go)
		return padOfPlayer(inputP)
	}
	if ids := padList(); len(ids) > 0 {
		return ids[0], true
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
	return padAxisOf(id)
}

// padAxisOf is the left stick / D-pad direction of one gamepad.
func padAxisOf(id ebiten.GamepadID) (dx, dy float64) {
	dx = padAxisFn(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
	dy = padAxisFn(id, ebiten.StandardGamepadAxisLeftStickVertical)
	if dx > -padDeadzone && dx < padDeadzone {
		dx = 0
	}
	if dy > -padDeadzone && dy < padDeadzone {
		dy = 0
	}
	if padHeldFn(id, ebiten.StandardGamepadButtonLeftLeft) {
		dx = -1
	}
	if padHeldFn(id, ebiten.StandardGamepadButtonLeftRight) {
		dx = 1
	}
	if padHeldFn(id, ebiten.StandardGamepadButtonLeftTop) {
		dy = -1
	}
	if padHeldFn(id, ebiten.StandardGamepadButtonLeftBottom) {
		dy = 1
	}
	return dx, dy
}

func padPressed(b ebiten.StandardGamepadButton) bool {
	id, ok := activeGamepad()
	return ok && padHeldFn(id, b)
}

func padJustPressed(b ebiten.StandardGamepadButton) bool {
	id, ok := activeGamepad()
	return ok && padJustFn(id, b)
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
	set(actRun, ebiten.KeyShiftLeft, noKey, ebiten.StandardGamepadButtonFrontTopRight) // R1; Shift der. is the run key of player 2
	set(actAttack, ebiten.KeySpace, ebiten.KeyJ, ebiten.StandardGamepadButtonRightBottom)
	set(actInteract, ebiten.KeyE, noKey, ebiten.StandardGamepadButtonRightRight)
	set(actDance, ebiten.KeyF, noKey, ebiten.StandardGamepadButtonRightTop)
	set(actBack, ebiten.KeyEscape, noKey, ebiten.StandardGamepadButtonRightRight)
	set(actConfirm, ebiten.KeyEnter, ebiten.KeySpace, ebiten.StandardGamepadButtonRightBottom)
	set(actEvent, ebiten.KeyV, noKey, ebiten.StandardGamepadButtonFrontTopLeft)
	set(actImmune, ebiten.KeyI, noKey, ebiten.StandardGamepadButtonFrontBottomRight) // R2 (R1 is run now)
	set(actFamous, ebiten.KeyO, noKey, ebiten.StandardGamepadButtonFrontBottomLeft)
	set(actKiller, ebiten.KeyK, noKey, ebiten.StandardGamepadButtonLeftStick)
	set(actPowerEvent, ebiten.KeyL, noKey, ebiten.StandardGamepadButtonRightStick)
	return b
}

var binds = defaultBindings()

func (a Action) held() bool {
	b := bindsFor(inputP)[a]
	for _, k := range b.Keys {
		if k != noKey && keyAllowed(inputP, k) && ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return b.Pad != noPad && padPressed(b.Pad)
}

// navFake: tests press actions without a keyboard (nil in the game).
var navFake map[Action]bool

func (a Action) just() bool {
	if navFake != nil && navFake[a] {
		return true
	}
	if a == actBack && backInject { // the on-screen back button (backbtn.go)
		return true
	}
	if a == actConfirm && backFocus { // Enter / A belongs to the focused back button
		return false
	}
	b := bindsFor(inputP)[a]
	for _, k := range b.Keys {
		if k != noKey && keyAllowed(inputP, k) && inpututil.IsKeyJustPressed(k) {
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
	case ebiten.KeyPeriod:
		return "."
	case ebiten.KeyComma:
		return ","
	case ebiten.KeySlash:
		return "/"
	}
	return k.String()
}

// padLabels: the buttons named as on a PlayStation-style pad, in plain letters and words (no logo; the font's
// ✕ ○ △ □ glyphs are too small to read at scale 1). The upper-case START / SELECT keep tr() off them ("Start" is
// a word of the dictionary); Triángulo / Cuadrado are translated (padLabel).
var padLabels = map[ebiten.StandardGamepadButton]string{
	ebiten.StandardGamepadButtonRightBottom: "X", ebiten.StandardGamepadButtonRightRight: "O",
	ebiten.StandardGamepadButtonRightLeft: "Cuadrado", ebiten.StandardGamepadButtonRightTop: "Triángulo",
	ebiten.StandardGamepadButtonFrontTopLeft: "L1", ebiten.StandardGamepadButtonFrontTopRight: "R1",
	ebiten.StandardGamepadButtonFrontBottomLeft: "L2", ebiten.StandardGamepadButtonFrontBottomRight: "R2",
	ebiten.StandardGamepadButtonCenterLeft: "SELECT", ebiten.StandardGamepadButtonCenterRight: "START",
	ebiten.StandardGamepadButtonLeftStick: "L3", ebiten.StandardGamepadButtonRightStick: "R3",
	ebiten.StandardGamepadButtonLeftTop: "D-pad ↑", ebiten.StandardGamepadButtonLeftBottom: "D-pad ↓",
	ebiten.StandardGamepadButtonLeftLeft: "D-pad ←", ebiten.StandardGamepadButtonLeftRight: "D-pad →",
}

func padLabel(b ebiten.StandardGamepadButton) string {
	if b == noPad {
		return "-"
	}
	if s, ok := padLabels[b]; ok {
		if b == ebiten.StandardGamepadButtonRightTop || b == ebiten.StandardGamepadButtonRightLeft {
			return tr(s) // a word: Triangle / Square in English
		}
		return s
	}
	return "Botón"
}

// padShort is padLabel for the written hints of the game: the D-pad as bare arrows.
func padShort(b ebiten.StandardGamepadButton) string {
	switch b {
	case ebiten.StandardGamepadButtonLeftTop:
		return "↑"
	case ebiten.StandardGamepadButtonLeftBottom:
		return "↓"
	case ebiten.StandardGamepadButtonLeftLeft:
		return "←"
	case ebiten.StandardGamepadButtonLeftRight:
		return "→"
	}
	return padLabel(b)
}

// lastInputPad: the last thing pressed was a gamepad button or stick (true) or a key (false). With one
// player the written hints show the pad buttons then (hintOnPad); trackInputDevice updates it each frame.
var lastInputPad bool

// dbgPadHints (-padhints, debug): the hints name the pad buttons with no pad plugged in (captures).
var dbgPadHints bool

// hardware access of trackInputDevice, for the tests
var (
	keysJustFn    = inpututil.AppendJustPressedKeys
	padButtonsFn  = inpututil.AppendJustPressedStandardGamepadButtons
	devKeyBuf     []ebiten.Key
	devPadBuf     []ebiten.StandardGamepadButton
	trackPadAxisT = 0.5 // a stick pushed this far counts as using the pad
)

func trackInputDevice() {
	devKeyBuf = keysJustFn(devKeyBuf[:0])
	if len(devKeyBuf) > 0 {
		lastInputPad = false
		return
	}
	for _, id := range padIDsFn() {
		devPadBuf = padButtonsFn(id, devPadBuf[:0])
		dx := padAxisFn(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		dy := padAxisFn(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if len(devPadBuf) > 0 || dx > trackPadAxisT || dx < -trackPadAxisT || dy > trackPadAxisT || dy < -trackPadAxisT {
			lastInputPad = true
			return
		}
	}
}

// hintOnPad: the hints of player p name his pad buttons. Two players: his device; one player: a pad is
// plugged in and it was the last thing used.
func hintOnPad(p int) bool {
	if dbgPadHints {
		return true
	}
	if twoInput {
		return playerDev[p&1] >= 0
	}
	return lastInputPad && padConnected()
}

// actionHint is the name of the first key bound to an action, for on-screen hints (the pad button when the
// player is playing with a pad).
func actionHint(a Action) string {
	if hintOnPad(0) && binds[a].Pad != noPad {
		return padShort(binds[a].Pad)
	}
	for _, k := range binds[a].Keys {
		if k != noKey {
			return keyLabel(k)
		}
	}
	return padShort(binds[a].Pad)
}

// padKeyText: a written hint that names ENTER / ESC (already translated) shows the pad buttons that do the same
// (✕ / ○ by default) while the player is on a pad; "R / ENTER" (retry) is just ✕ there.
func padKeyText(s string) string {
	if !hintOnPad(0) {
		return s
	}
	ok, back := padShort(binds[actConfirm].Pad), padShort(binds[actBack].Pad)
	return strings.NewReplacer("R / ENTER", ok, "ENTER", ok, "ESC", back).Replace(s)
}

// ---------------------------------------------------------------------------
// Fixed pad buttons and the clash between "interact" and "back" (both ○ by default).
// ---------------------------------------------------------------------------

// justKeys is just() without the pad button: the keys (and the on-screen BACK button) only. In a party ○ is
// interact, so the pause reads Esc this way and START apart.
func (a Action) justKeys() bool {
	if navFake != nil && navFake[a] {
		return true
	}
	if a == actBack && backInject {
		return true
	}
	b := bindsFor(inputP)[a]
	for _, k := range b.Keys {
		if k != noKey && keyAllowed(inputP, k) && inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

// justKeysEither: justKeys for any of the two players (one player: player 1).
func (a Action) justKeysEither() bool {
	if !twoInput {
		return a.justKeys()
	}
	old := inputP
	defer func() { inputP = old }()
	inputP = 0
	if a.justKeys() {
		return true
	}
	inputP = 1
	return a.justKeys()
}

// menuInteractJust: "interact" in a menu (switch J1/J2 in the wardrobe, open a card). Its pad button is ignored
// there when it is the button that confirms or goes back (○ by default would both swap and leave); □ does it
// on the pad then.
func menuInteractJust() bool {
	b := bindsFor(inputP)
	p := b[actInteract].Pad
	if p == noPad || p == b[actBack].Pad || p == b[actConfirm].Pad {
		return actInteract.justKeys() || (!padButtonUsed(ebiten.StandardGamepadButtonRightLeft) && padJustPressed(ebiten.StandardGamepadButtonRightLeft))
	}
	return actInteract.just()
}

// padButtonUsed: some action of either player is bound to the pad button b.
func padButtonUsed(b ebiten.StandardGamepadButton) bool {
	for a := Action(0); a < actCount; a++ {
		if binds[a].Pad == b || binds2[a].Pad == b {
			return true
		}
	}
	return false
}

type savedBinding struct {
	Keys []string `json:"keys"`
	Pad  int      `json:"pad"`
}

// settingsFileOverride lets the tests point the settings to a temporary file.
var settingsFileOverride string

func settingsPath() string {
	if settingsFileOverride != "" {
		return settingsFileOverride
	}
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

// readSettingsFile reads settings.json as a map of raw entries (old files only hold the actions).
func readSettingsFile() map[string]json.RawMessage {
	path := settingsPath()
	if (noAssets && settingsFileOverride == "") || path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	return raw
}

func bindingFrom(sb savedBinding, def Binding) Binding {
	var b Binding
	for s := 0; s < 2; s++ {
		b.Keys[s] = noKey
		if s < len(sb.Keys) {
			b.Keys[s] = keyByName(sb.Keys[s])
		}
	}
	b.Pad = ebiten.StandardGamepadButton(sb.Pad)
	return b
}

func bindingTo(b Binding) savedBinding {
	sb := savedBinding{Pad: int(b.Pad)}
	for _, k := range b.Keys {
		if k == noKey {
			sb.Keys = append(sb.Keys, "")
		} else {
			sb.Keys = append(sb.Keys, k.String())
		}
	}
	return sb
}

func loadBindings() Bindings {
	b := defaultBindings()
	raw := readSettingsFile()
	var saved [actCount]bool
	for i, id := range actionIDs {
		var sb savedBinding
		if m, ok := raw[id]; ok && json.Unmarshal(m, &sb) == nil {
			b[i], saved[i] = bindingFrom(sb, b[i]), true
		}
	}
	if settingsControlsRev(raw) < controlsRev {
		migratePads(&b, saved)
	}
	return b
}

// ---------------------------------------------------------------------------
// Revisions of the default pad buttons. settings.json keeps "controls_rev"; a file without it (or an older
// one) was written with older defaults. On load, every saved action whose pad button is still the default of
// its time takes the new default; a button the player changed is kept. Only memory changes: the file is
// written (with the new revision) the next time the player saves something. The keyboard did not change.
//
//	rev 0 -> 1 (2026-11-11, PlayStation layout): run R2 -> R1, interact □ -> ○, immunity R1 -> R2.
//
// ---------------------------------------------------------------------------

const controlsRev = 1

// oldPadDefaults: the pad buttons of rev 0 that changed (the same for both players).
var oldPadDefaults = map[Action]ebiten.StandardGamepadButton{
	actRun:      ebiten.StandardGamepadButtonFrontBottomRight,
	actInteract: ebiten.StandardGamepadButtonRightLeft,
	actImmune:   ebiten.StandardGamepadButtonFrontTopRight,
}

// settingsControlsRev: the revision a settings file was written with (0: none, a file of v1.0.0).
func settingsControlsRev(raw map[string]json.RawMessage) int {
	var r int
	if m, ok := raw["controls_rev"]; ok && json.Unmarshal(m, &r) == nil {
		return r
	}
	return 0
}

// menuAction: the actions that only drive the menus may share a button with a party action.
func menuAction(a Action) bool { return a == actBack || a == actConfirm }

// migratePads moves the saved actions still on an old default to the new one. A moved button that would land
// on a button the player chose for another party action stays where it was (his layout keeps working).
func migratePads(b *Bindings, saved [actCount]bool) {
	def := defaultBindings()
	var moved [actCount]bool
	for a, old := range oldPadDefaults {
		if saved[a] && b[a].Pad == old {
			b[a].Pad, moved[a] = def[a].Pad, true
		}
	}
	for changed := true; changed; { // a step back can clash with another moved one: repeat until stable
		changed = false
		for a := Action(0); a < actCount; a++ {
			if !moved[a] {
				continue
			}
			for o := Action(0); o < actCount; o++ {
				if o != a && !moved[o] && !menuAction(o) && b[o].Pad == b[a].Pad {
					b[a].Pad, moved[a], changed = oldPadDefaults[a], false, true
					break
				}
			}
		}
	}
}

// saveBindings writes every setting kept in settings.json: the actions of player 1 and 2 and the
// player mode and devices. Unknown entries of the file are kept.
func saveBindings() {
	path := settingsPath()
	if (noAssets && settingsFileOverride == "") || path == "" {
		return
	}
	out := readSettingsFile()
	if out == nil {
		out = map[string]json.RawMessage{}
	}
	put := func(key string, v any) {
		if data, err := json.Marshal(v); err == nil {
			out[key] = data
		}
	}
	for i, id := range actionIDs {
		put(id, bindingTo(binds[i]))
	}
	for i := 0; i < actPlayerCount; i++ {
		put("p2_"+actionIDs[i], bindingTo(binds2[i]))
	}
	put("players", savedPlayers{Two: twoPlayerMode, Dev1: devPref[0], Dev2: devPref[1]})
	put("controls_rev", controlsRev)
	if data, err := json.MarshalIndent(out, "", "  "); err == nil {
		os.WriteFile(path, data, 0o644)
	}
}
