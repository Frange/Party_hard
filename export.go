package main

import (
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
	ca := newCharAsset(name, cat, charW, charH)
	ca.Meta.Role = role
	if spec.Weapon != 0 {
		ca.Meta.Carry = holdName(spec.Weapon)
	} else if spec.Carry != 0 {
		ca.Meta.Carry = holdName(spec.Carry)
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

func exportCharacters(root string) int {
	rng := rand.New(rand.NewSource(20240615))
	n := 0
	save := func(ca *CharAsset) {
		if err := saveCharAsset(root, ca); err != nil {
			fmt.Println("export:", err)
			return
		}
		n++
	}
	save(charAssetFromSpec(playerSpec(), "darius", "main_character", ""))
	save(charAssetFromSpec(copSpec(), "cop", "police", "cop"))
	save(charAssetFromSpec(medicSpec(), "medic", "police", "medic"))
	for i := 1; i <= 4; i++ {
		save(charAssetFromSpec(guardSpec(rng), fmt.Sprintf("guard_%02d", i), "enemies", "guard"))
	}
	for _, k := range killerPresets {
		spec := k.F(rng)
		save(charAssetFromSpec(spec, safeName(k.Name), "special_enemies", "killer"))
	}
	for _, a := range archetypePresets {
		spec := a.F(rng)
		save(charAssetFromSpec(spec, safeName(a.Name), "people", ""))
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
		save(charAssetFromSpec(spec, name, "special_people", ""))
	}
	for i := 1; i <= 40; i++ {
		save(charAssetFromSpec(genericSpec(rng), fmt.Sprintf("person_%02d", i), "people", ""))
	}
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
