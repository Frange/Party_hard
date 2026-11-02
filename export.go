package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// Export of the built-in content as editable asset files
// ---------------------------------------------------------------------------

// charAssetFromSpec renders the standing pose of a procedural character in the three views,
// recording which body part painted each pixel, so it can be edited and re-animated.
func charAssetFromSpec(spec CharSpec, name, cat, role string) *CharAsset {
	return charAssetFromSpecOpt(spec, name, cat, role, false)
}

// charAssetFromSpecOpt: bake=true paints the held object into the pictures (the old way, kept to recognise old files).
func charAssetFromSpecOpt(spec CharSpec, name, cat, role string, bake bool) *CharAsset {
	ca := newCharAsset(name, cat, charW, charH)
	ca.Meta.Role = role
	if spec.Weapon != 0 {
		ca.Meta.Carry = holdName(spec.Weapon)
	} else if spec.Carry != 0 {
		ca.Meta.Carry = holdName(spec.Carry)
	}
	// The held object is NOT painted into the pictures: the animation adds it in the hand
	// (otherwise it would be there twice, one of them stuck to the body).
	if !bake {
		spec.Weapon, spec.Carry = 0, 0
	}
	trackParts = true
	defer func() { trackParts = false }()
	for i, vw := range []int{vFront, vSide, vBack} {
		b := renderChar(&spec, vw, poseFor(AnimIdle, 0))
		ca.Views[i] = b
		if b.pid == nil {
			b.pid = make([]uint8, b.w*b.h)
		}
	}
	return ca
}

func safeName(s string) string {
	s = strings.ToLower(s)
	var out []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	return string(out)
}

type builtinChar struct {
	spec      CharSpec
	name, cat string
	role      string
}

// builtinChars lists every built-in character in the order (and with the random
// choices) of the first export, so that files written back later match them.
func builtinChars() []builtinChar {
	rng := rand.New(rand.NewSource(20240615))
	var out []builtinChar
	add := func(spec CharSpec, name, cat, role string) { out = append(out, builtinChar{spec, name, cat, role}) }
	add(playerSpec(), "darius", "main_character", "")
	add(copSpec(), "cop", "police", "cop")
	add(medicSpec(), "medic", "police", "medic")
	for i := 1; i <= 4; i++ {
		add(guardSpec(rng), fmt.Sprintf("guard_%02d", i), "enemies", "guard")
	}
	for _, k := range killerPresets {
		add(k.F(rng), safeName(k.Name), "special_enemies", "killer")
	}
	for _, a := range archetypePresets {
		add(a.F(rng), safeName(a.Name), "people", "")
	}
	seen := map[string]int{}
	for _, f := range famousPresets {
		spec := f(rng)
		seen[spec.Name]++
		if seen[spec.Name] > 2 {
			continue
		}
		name := safeName(spec.Name)
		if seen[spec.Name] > 1 {
			name += "_2"
		}
		add(spec, name, "special_people", "")
	}
	for i := 1; i <= 40; i++ {
		add(genericSpec(rng), fmt.Sprintf("person_%02d", i), "people", "")
	}
	// responders and crews (they come last so the order above is never disturbed)
	u := unitSpecs()
	for _, c := range u.copAlt {
		add(c, c.Name, "police", "cop")
	}
	add(u.swat, "swat", "police", "swat")
	for _, c := range u.fbi {
		add(c, c.Name, "police", "fbi")
	}
	for _, c := range u.military {
		add(c, c.Name, "police", "military")
	}
	for _, c := range u.rangers {
		add(c, c.Name, "police", "ranger")
	}
	add(fireSpec(), "bombero", "police", "firefighter")
	ts := techSpec()
	ts.Name = "tecnico"
	add(ts, "tecnico", "police", "technician")
	return out
}

func exportCharacters(root string) int {
	n := 0
	for _, b := range builtinChars() {
		if err := saveCharAsset(root, charAssetFromSpec(b.spec, b.name, b.cat, b.role)); err != nil {
			fmt.Println("export:", err)
			continue
		}
		n++
	}
	return n
}

// refreshBakedAssets repairs the characters exported when the held object was still painted
// into the pictures (it showed twice in the animations, once stuck to the body). It redraws the
// built-in ones that were never edited, once.
func refreshBakedAssets(root string) int {
	marker := filepath.Join(root, ".held_objects_fixed")
	if root == "" {
		return 0
	}
	if _, err := os.Stat(marker); err == nil {
		return 0
	}
	n := 0
	for _, b := range builtinChars() {
		dir := filepath.Join(spritesDir(root, b.cat), b.name)
		old, err := loadCharAsset(dir)
		if err != nil || old.Meta.Edited {
			continue
		}
		plain := charAssetFromSpec(b.spec, b.name, b.cat, b.role)
		if plain.Meta.Carry == "" && old.Meta.Carry == "" {
			continue // nothing held: nothing was baked in
		}
		// The object is where the old export painted it: take out exactly those pixels (and only
		// those that still look like the object), leave everything else as it was.
		legacy := charAssetFromSpecOpt(b.spec, b.name, b.cat, b.role, true)
		changed := 0
		for i := range old.Views {
			o, l, p := old.Views[i], legacy.Views[i], plain.Views[i]
			if o.w != l.w || o.h != l.h {
				continue
			}
			for k := 0; k < len(o.p); k += 4 {
				if bytes.Equal(l.p[k:k+4], p.p[k:k+4]) {
					continue // not an object pixel
				}
				if !bytes.Equal(o.p[k:k+4], l.p[k:k+4]) {
					continue // somebody painted something else here: keep it
				}
				copy(o.p[k:k+4], p.p[k:k+4])
				if o.pid != nil && p.pid != nil {
					o.pid[k/4] = p.pid[k/4]
				}
				changed++
			}
		}
		if changed == 0 {
			continue
		}
		if old.Meta.Carry == "" {
			old.Meta.Carry = plain.Meta.Carry
		}
		if saveCharAsset(root, old) == nil {
			n++
		}
	}
	os.WriteFile(marker, []byte("ok\n"), 0o644)
	return n
}

// exportObjects writes every built-in prop (native scale, soft outline) with its metadata.
func exportObjects(root string) int {
	n := 0
	for name, k := range propKinds {
		cnt := k.Vars
		if cnt == 0 {
			cnt = 1
		}
		oa := &ObjAsset{}
		oa.Meta = ObjMeta{Name: name, Category: "furniture", Solid: !k.NoSolid, Hide: k.Hide, Phone: k.Phone, Poison: k.Poison,
			Trap: k.Trap, Light: k.Light, Variants: cnt > 1, SortOff: int(k.SortOff * wk), UseKind: k.UseKind}
		switch name {
		case "car", "motorbike":
			oa.Meta.Category = "vehicle"
		case "horse", "hen":
			oa.Meta.Category = "animal"
		case "tree", "palm", "bush", "plant":
			oa.Meta.Category = "nature"
		}
		if k.Trap != "" {
			oa.Meta.Category = "trap"
		}
		if name == "hen" {
			oa.Meta.Variants, oa.Meta.FPS = false, 2.5
		}
		for v := 0; v < cnt; v++ {
			oa.Frames = append(oa.Frames, renderPropSprite(name, k, v))
		}
		ax := int((float64(k.W)/2 + 1) * wk)
		ay := int((float64(k.H) + 1) * wk)
		oa.Meta.AnchorX, oa.Meta.AnchorY = ax, ay
		fw, fh := int(k.FW*wk+0.5), int(k.FH*wk+0.5)
		oa.Meta.FootW, oa.Meta.FootH = fw, fh
		oa.Meta.FootX, oa.Meta.FootY = ax-fw/2, ay-fh
		for _, u := range k.Use {
			oa.Meta.UseX = append(oa.Meta.UseX, int(u.X*wk))
			oa.Meta.UseY = append(oa.Meta.UseY, int(u.Y*wk))
		}
		if err := saveObjAsset(root, oa); err != nil {
			fmt.Println("export:", err)
			continue
		}
		n++
	}
	// vehicles and helpers used by the game
	save := func(name, cat string, b *pixBuf) {
		oa := &ObjAsset{Meta: ObjMeta{Name: name, Category: cat, AnchorX: b.w / 2, AnchorY: b.h - 1, FootX: 0, FootY: b.h - 4, FootW: b.w, FootH: 4}, Frames: []*pixBuf{b}}
		saveObjAsset(root, oa)
		n++
	}
	save("body_bag", "special", genBodyBag().scaled(wk).outlinedSel(0.45))
	save("heli", "vehicle", genHeli().scaled(wk).outlinedSel(0.45))
	save("tank", "vehicle", genTank().outlinedSel(0.45))
	save("ufo", "vehicle", genUFO().outlinedSel(0.45))
	return n
}

func exportDefaultsExist(root string) bool {
	st, err := os.Stat(filepath.Join(root, "sprites"))
	return err == nil && st.IsDir()
}

// exportMissingCharacters writes the built-in characters that were added after the assets
// folder was first created (the kids, Juancho, the newest famous faces...) so that the
// editor lists them. They are marked shop_only: the game does not use these files as random
// guests; they only matter once edited.
func exportMissingCharacters(root string) int {
	if root == "" {
		return 0
	}
	n := 0
	exists := func(name string) bool {
		for _, cat := range charCategories {
			if _, err := os.Stat(filepath.Join(spritesDir(root, cat), name, "meta.json")); err == nil {
				return true
			}
		}
		return false
	}
	for _, b := range builtinChars() {
		if b.name == "" || exists(b.name) {
			continue
		}
		spec := b.spec
		spec.Ride = 0 // the scooter or board is drawn by the game, not part of the picture
		ca := charAssetFromSpec(spec, b.name, b.cat, b.role)
		ca.Meta.ShopOnly = true
		ca.Meta.DisplayName = nameFromSpec(&spec)
		ca.Meta.Gender = "m"
		if spec.Female {
			ca.Meta.Gender = "f"
		}
		if err := saveCharAsset(root, ca); err != nil {
			fmt.Println("export:", err)
			continue
		}
		n++
	}
	return n
}
