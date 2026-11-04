package main

import (
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
				list := []int{menuContinue, menuNewGame, menuPlayers, menuRandom, menuShop, menuTrophies, menuScenes, menuSettings, menuQuit}
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

// the funny line of the JUGADORES row sits in the gap between its row and the next, clear of the logo,
// of the window and of the bed, in both languages and with the DEBUG row.
func TestPlayersNoteClear(t *testing.T) {
	g := textGame()
	defer func(l string) { uiLang = l }(uiLang)
	const logoBottom = 30 + 34 + 70 + 4 + 3 // the ring of the logo, with its bob
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, dbg := range []bool{false, true} {
			for _, save := range []bool{false, true} {
				g.debugMenu = dbg
				list := []int{menuContinue, menuNewGame, menuPlayers, menuRandom, menuShop, menuTrophies, menuScenes, menuSettings, menuQuit}
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
					nextTop := menuRowY(i+1, len(list)) + 5 // the glyphs of the next row start 5 px below its y
					rowBot := y + 24
					for _, note := range []string{"tú solito", "¡el doble de caos!"} {
						x, ny := playersNotePos(y, menuRowY(1, len(list))-menuRowY(0, len(list)))
						w := g.textW(note, 1)
						top, bot := ny+3, ny+12 // the glyphs of the line
						if top < rowBot || bot > nextTop {
							t.Errorf("%s dbg=%v save=%v: %q (%.0f..%.0f) is not in the gap %.0f..%.0f", lang, dbg, save, note, top, bot, rowBot, nextTop)
						}
						if top < logoBottom {
							t.Errorf("%s: %q overlaps the logo", lang, note)
						}
						if x+w > menuBedsX(true)[0][0]-14 && bot > 250 && top < 440 {
							t.Errorf("%s: %q (to x=%.0f) reaches the first headboard", lang, note, x+w)
						}
						if x+w > menuListRight {
							t.Errorf("%s: %q is %.0f px wide, wider than the list", lang, note, w)
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
