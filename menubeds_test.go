package main

import (
	"math"
	"testing"
	"unicode/utf8"
)

// the menu shows one bed with one player and two (blue and red, separate) with two, and the first never
// touches the list of options, the second never leaves the screen and the table fits in between.
func TestMenuBedsOneOrTwo(t *testing.T) {
	if n := len(menuBedsX(false)); n != 1 {
		t.Fatalf("one player: %d beds", n)
	}
	b := menuBedsX(true)
	if len(b) != 2 {
		t.Fatalf("two players: %d beds", len(b))
	}
	if b[0][1] >= b[1][0] {
		t.Errorf("the two beds touch: %v", b)
	}
	if !twoBedsApart() {
		t.Error("the table does not fit between the beds")
	}
	if hb := b[0][0] - 14; hb < menuListRight {
		t.Errorf("the first headboard (x=%.0f) is over the list (ends at %.0f)", hb, menuListRight)
	}
	if end := b[1][1] + 14; end > screenW-8 {
		t.Errorf("the second bed ends at x=%.0f, outside the screen", end)
	}
}

// every option of the list that is level with the beds (the headboards start at y=250, plus the slide down
// that drawMenu gives them when the JUGADORES row is level with them), selected (with its arrow) or not,
// ends before the first headboard of the two beds; and the JUGADORES row (with its colour squares) too, or
// else it ends above them. All the combinations: DEBUG row, CONTINUAR row, Spanish and English.
func TestMenuListClearOfBeds(t *testing.T) {
	g := textGame()
	defer func(l string) { uiLang = l }(uiLang)
	hb := menuBedsX(true)[0][0] - 14
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, dbg := range []bool{false, true} {
			for _, save := range []bool{false, true} {
				list := []int{menuContinue, menuNewGame, menuPlayers, menuShop, menuTrophies, menuScenes, menuSettings, menuQuit}
				if !save {
					list = list[1:]
				}
				if dbg {
					list = append([]int{menuDebug}, list...)
				}
				dy := 0.0
				for i, id := range list {
					if id == menuPlayers {
						dy = menuBedShiftYFor(menuRowY(i, len(list)))
					}
				}
				top := menuBedTop + dy
				if top+190 > screenH-60 { // the beds (and their legs) must not reach the footer texts
					t.Errorf("%s dbg=%v save=%v: the beds slide down to y=%.0f", lang, dbg, save, top+190)
				}
				for i, id := range list {
					y := menuRowY(i, len(list))
					if y+24 <= top {
						continue // the row ends above the beds
					}
					end := 110 + g.textW(menuLabel(id)+" ◄", 2.2)
					if id == menuPlayers { // label, arrow, the two squares, the 2 and the other arrow
						end = 110 + g.textW(menuLabel(id), 2.2) + 14 + g.textW("◄", 2.2) + 8 + 24 + g.textW("2", 2.2) + 8 + g.textW("►", 2.2)
					}
					if end > hb {
						t.Errorf("%s dbg=%v save=%v: %q (row y=%.0f, to x=%.0f) is over the headboard (x=%.0f, top y=%.0f)", lang, dbg, save, menuLabel(id), y, end, hb, top)
					}
				}
			}
		}
	}
}

// the funny line of the JUGADORES row stays at least 4 px (real distance between ink rectangles, shadow
// included) from every other row of the list, from the logo, from the centred sentence and from the beds,
// in both languages, with and without DEBUG / a save, with the real positions drawPlayersRow uses.
func TestPlayersNoteClear(t *testing.T) {
	g := textGame()
	defer func(l string) { uiLang = l }(uiLang)
	const logoBottom = 30 + 34 + 70 + 4 + 3 // the ring of the logo, with its bob
	type rc struct{ x0, y0, x1, y1 float64 }
	gap := func(a, b rc) float64 { // the distance between two rectangles (0 if they touch or overlap)
		dx := math.Max(0, math.Max(a.x0-b.x1, b.x0-a.x1))
		dy := math.Max(0, math.Max(a.y0-b.y1, b.y0-a.y1))
		return math.Hypot(dx, dy)
	}
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, dbg := range []bool{false, true} {
			for _, save := range []bool{false, true} {
				for _, two := range []bool{false, true} {
					g.debugMenu = dbg
					list := []int{menuContinue, menuNewGame, menuPlayers, menuShop, menuTrophies, menuScenes, menuSettings, menuQuit}
					if !save {
						list = list[1:]
					}
					if dbg {
						list = append([]int{menuDebug}, list...)
					}
					for i, id := range list {
						if id != menuPlayers {
							continue
						}
						y := menuRowY(i, len(list))
						x := 110 + g.textW(menuLabel(id), 2.2) + 14 + g.textW("◄", 2.2) + 8
						note := "tú solito"
						if two {
							x += 24
							note = "¡el doble de caos!"
						}
						val := "1"
						if two {
							val = "2"
						}
						x += g.textW(val, 2.2) + 8 + g.textW("►", 2.2)
						w := g.textW(note, 1)
						bedLeft := menuBedsX(two)[0][0] - 14
						if two {
							bedLeft = 1e9
						}
						nx, ny := playersNotePos(x, y, w, bedLeft)
						n := rc{nx, ny + 3, nx + w + 1, ny + 12 + 1} // glyphs of the line + shadow
						var others []rc
						for j, id2 := range list {
							if j == i {
								continue
							}
							y2 := menuRowY(j, len(list))
							others = append(others, rc{110, y2 + 5, 110 + g.textW(menuLabel(id2)+" ◄", 2.2) + 1, y2 + 24})
						}
						sw := g.textW("Nadie te dejó dormir. Ahora nadie saldrá de la fiesta.", 1)
						others = append(others, rc{640 - sw/2, 232 + 3, 640 + sw/2, 232 + 13})
						others = append(others, rc{0, 0, screenW, logoBottom})
						mw := g.textW("Termina la fiesta actual antes de empezar otra", 1.6)
						others = append(others, rc{660 - mw/2, 262 + 4, 660 + mw/2, 262 + 20}) // the message of the menu
						bx := menuBedsX(two)[0][0] - 14
						top := 250.0
						if two {
							top += menuBedShiftYFor(y)
						}
						others = append(others, rc{bx, top, bx + 60, 480})
						for k, o := range others {
							if d := gap(n, o); d < 4 {
								t.Errorf("%s two=%v dbg=%v save=%v: %q %v is %.1f px from %v (item %d), needs >= 4", lang, two, dbg, save, note, n, d, o, k)
							}
						}
						if n.x1 > 1100 {
							t.Errorf("%s: note too far right", lang)
						}
					}
				}
			}
		}
	}
}

// the two sleepers of the menu have different characters: different poses, never a mirror copy, and
// different sentences (short, translated, and none shared).
func TestMenuTwoCharacters(t *testing.T) {
	sat, tossed, plugged := false, false, false
	same := 0
	n := 0
	for tt := 0.0; tt < 90; tt += 0.1 {
		p := menuTwoPoses(tt)
		n++
		if p[0] == p[1] {
			same++
		}
		if p[0].sit > 0.01 || p[0].fist || p[0].puff {
			t.Fatalf("t=%.1f: the patient one must not sit, bang or fume: %+v", tt, p[0])
		}
		sat = sat || p[1].sit > 0.4
		tossed = tossed || p[1].turn > 0
		plugged = plugged || p[0].earplug
	}
	if !sat || !tossed || !plugged {
		t.Errorf("gags missing: sits=%v tosses=%v earplugs=%v", sat, tossed, plugged)
	}
	if same > n/10 {
		t.Errorf("the two sleepers do the same %d/%d of the time", same, n)
	}
	seen := map[string]bool{}
	for _, l := range menuLines1 {
		seen[l] = true
	}
	defer func(l string) { uiLang = l }(uiLang)
	for i, list := range [][]string{menuLines1, menuLines2} {
		for _, l := range list {
			if i == 1 && seen[l] {
				t.Errorf("%q is said by both", l)
			}
			if utf8.RuneCountInString(l) > 48 {
				t.Errorf("%q is longer than 48 characters", l)
			}
			uiLang = "en"
			if tr(l) == l {
				t.Errorf("%q has no English translation", l)
			}
		}
	}
	// at any moment they say different things
	for tt := 0.0; tt < 200; tt += 0.5 {
		if menuLine(0, tt) == menuLine(1, tt) {
			t.Fatalf("t=%.1f: both say %q", tt, menuLine(0, tt))
		}
	}
}
