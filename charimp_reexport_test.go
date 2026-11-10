package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// The game draws the famous guests of special_people from the exported pictures in
// assets/sprites/special_people, not from the code, so every redrawn guest needs its pictures
// written again (same method as TestCharimpAExport). This file covers the I-P and Q-Z passes,
// the guests renewed before them, and every other famous guest whose drawing does not depend on
// chance and whose pictures had fallen behind the code. Edited ones (meta "edited") are never touched.

// charimpRenewed: guests renewed before the A-H / I-P / Q-Z passes that live in special_people
// (guybrush, lechuck, the T-rex and the bull are not there: the game draws them from the code).
var charimpRenewed = []string{
	"vader", "sonic", "tails", "robocop", "asterix", "obelix", "wally", "bob_esponja", "fry",
	"cartman", "kenny", "kyle", "stan", "luffy", "zoro", "nami", "sanji", "chopper", "johnny5",
	"homer", "marge", "bart", "lisa", "maggie", // simpsons_art.go
	"crash", // crash_art.go
	"taz",   // taz_art.go
	// csm_art.go
	"chewbacca", "steve_cubos", "mario", "luigi", "wario", "waluigi", "gandalf", "frodo", "knuckles",
	"superman", "batman", "hulk", "capitan_america", "lobezno", "deadpool", // hero_art.go
	"capitan_planeta", "planetario_tierra", "planetario_fuego", "planetario_viento", "planetario_agua", "planetario_corazon", // planet_art.go
	"zipi", "zape", "pantera_rosa", "mr_bean", // zbp_art.go
	"doraemon", "nobita", "harry_potter", "hermione", "ron", // dnh_art.go
	"neo", "joker", "gb_venkman", "gb_stantz", "gb_spengler", "gb_winston", // njg_art.go
	"lara_croft", "marty_mcfly", "doc_brown", "yoda", // lmdy_art.go
	// pnk_art.go: Sum 41, Blink-182, The Offspring
	"deryck", "dave_baksh", "jason_mc", "frank_zummo", "mark_hoppus", "tom_delonge", "travis_barker",
	"off_cantante", "off_guitarra", "off_bajo", "off_bateria",
	// bandsb_art.go: Billy Talent, Lost Prophets (six), Despistaos, Punk Sailor
	"bt_cantante", "bt_guitarra", "bt_bajo", "bt_bateria",
	"lp_cantante", "lp_guitarra", "lp_bajo", "lp_bateria", "lp_ritmo", "lp_corista",
	"des_cantante", "des_guitarra", "des_bajo", "des_bateria",
	"ps_cantante", "ps_guitarra", "ps_bajo", "ps_bateria",
}

// charimpListed: the redrawn guests named one by one (I-P, Q-Z and the renewed ones).
func charimpListed() map[string]bool {
	out := map[string]bool{}
	for _, l := range [][]string{charimpIImproved, qzImproved, charimpRenewed} {
		for _, s := range l {
			out[s] = true
		}
	}
	return out
}

// charimpFixed: does this built-in always draw the same pictures, whatever the random source?
// Only those can be compared with the exported files (the others are one random variant).
func charimpFixed(b builtinChar) bool {
	count := map[string]int{}
	for _, f := range famousPresets {
		ref := f(newRand(1))
		count[ref.Name]++
		name := safeName(ref.Name)
		if count[ref.Name] == 2 {
			name += "_2"
		}
		if name != b.name {
			continue
		}
		a := charimpFresh(builtinChar{ref, b.name, b.cat, b.role})
		for _, seed := range []int64{7, 99, 12345, 20240615} {
			sp := f(newRand(seed))
			if !charimpSame(a, charimpFresh(builtinChar{sp, b.name, b.cat, b.role})) {
				return false
			}
		}
		return true
	}
	return false
}

// charimpFresh: the pictures the code draws today for a built-in special_people character.
func charimpFresh(b builtinChar) *CharAsset {
	spec := b.spec
	spec.Ride = 0
	return charAssetFromSpec(spec, b.name, b.cat, b.role)
}

func charimpSame(a, b *CharAsset) bool {
	for i := range a.Views {
		if a.Views[i].w != b.Views[i].w || a.Views[i].h != b.Views[i].h || string(a.Views[i].p) != string(b.Views[i].p) {
			return false
		}
	}
	return true
}

// charimpTargets: the special_people built-ins whose pictures must match the code: the listed
// ones, and every one that always draws the same.
func charimpTargets() []builtinChar {
	listed := charimpListed()
	var out []builtinChar
	for _, b := range builtinChars() {
		if b.cat != "special_people" || b.name == "_2" {
			continue
		}
		if listed[b.name] || charimpFixed(b) {
			out = append(out, b)
		}
	}
	return out
}

// CHARIMP_SCAN=assets-root go test -run TestCharimpStaleScan lists the special_people folders whose
// pictures differ from the code (edited ones apart).
func TestCharimpStaleScan(t *testing.T) {
	root := os.Getenv("CHARIMP_SCAN")
	if root == "" {
		t.Skip("set CHARIMP_SCAN to an assets folder")
	}
	var stale, random, edited []string
	for _, b := range builtinChars() {
		if b.cat != "special_people" {
			continue
		}
		old, err := loadCharAsset(filepath.Join(spritesDir(root, b.cat), b.name))
		if err != nil {
			continue
		}
		if old.Meta.Edited {
			edited = append(edited, b.name)
			continue
		}
		if !charimpSame(old, charimpFresh(b)) {
			if charimpListed()[b.name] || charimpFixed(b) {
				stale = append(stale, b.name)
			} else {
				random = append(random, b.name)
			}
		}
	}
	sort.Strings(stale)
	t.Logf("stale %d: %v", len(stale), stale)
	t.Logf("differ but drawn at random %d: %v", len(random), random)
	t.Logf("edited %d: %v", len(edited), edited)
}

// CHARIMP_EXPORT=assets-root go test -run TestCharimpReexport writes again, from the code, the
// pictures of the targets that differ; the whole meta.json is kept (only the held object follows
// the code). Edited ones are left alone.
func TestCharimpReexport(t *testing.T) {
	root := os.Getenv("CHARIMP_EXPORT")
	if root == "" {
		t.Skip("set CHARIMP_EXPORT to an assets folder")
	}
	n, same := 0, 0
	for _, b := range charimpTargets() {
		old, err := loadCharAsset(filepath.Join(spritesDir(root, b.cat), b.name))
		if err != nil || old.Meta.Edited {
			t.Logf("%s: skipped (%v)", b.name, err)
			continue
		}
		ca := charimpFresh(b)
		if charimpSame(old, ca) {
			same++
			continue
		}
		carry := ca.Meta.Carry
		ca.Meta = old.Meta
		ca.Meta.Carry = carry
		if err := saveCharAsset(root, ca); err != nil {
			t.Fatal(err)
		}
		n++
	}
	t.Logf("redrawn: %d, already right: %d", n, same)
}

// The pictures in assets/sprites/special_people of every target are the ones the code draws
// today, the rig built from them walks in the three views, and meta.json keeps its name, gender
// and shop flag.
func TestCharimpReexportedPictures(t *testing.T) {
	listed := charimpListed()
	seen := map[string]bool{}
	for _, b := range charimpTargets() {
		dir := filepath.Join(spritesDir("assets", b.cat), b.name)
		if _, err := os.Stat(filepath.Join(dir, "meta.json")); err != nil && !listed[b.name] {
			continue // not exported (guybrush, the skeletons...): the game draws it from the code
		}
		ca, err := loadCharAsset(dir)
		if err != nil {
			t.Fatalf("%s: %v", b.name, err)
		}
		if ca.Meta.Edited {
			continue
		}
		seen[b.name] = true
		if !charimpSame(ca, charimpFresh(b)) {
			t.Errorf("%s: the exported pictures are not the ones the code draws", b.name)
		}
		var raw map[string]any
		data, _ := os.ReadFile(filepath.Join(dir, "meta.json"))
		if json.Unmarshal(data, &raw) != nil || raw["name"] != b.name || raw["category"] != "special_people" {
			t.Errorf("%s: meta.json broken: %s", b.name, data)
		}
		rig := buildRig(ca)
		for vw := 0; vw < 3; vw++ {
			f0 := rig.render(vw, poseFor(AnimWalk, 0))
			if n, _ := charimpAPix(f0); n < 60 {
				t.Errorf("%s rig vw %d: only %d pixels", b.name, vw, n)
			}
			if string(rig.render(vw, poseFor(AnimWalk, 1)).p) == string(f0.p) {
				t.Errorf("%s rig vw %d: walking does not change the frames", b.name, vw)
			}
		}
	}
	for s := range listed {
		if !seen[s] {
			t.Errorf("%s: not checked", s)
		}
	}
	t.Logf("checked %d special_people folders", len(seen))
}
