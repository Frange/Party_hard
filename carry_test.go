package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

// The "Objeto" setting of the character editor must show up in the animation.
func TestCarryShowsInRig(t *testing.T) {
	root := assetsRoot(false)
	ca, err := loadCharAsset(filepath.Join(spritesDir(root, "main_character"), "darius"))
	if err != nil {
		t.Skip("no darius")
	}
	for _, kind := range []string{"beer", "cup", "pizza", "sign", "phone"} {
		for _, anim := range []int{AnimIdle, AnimWalk} {
			a := *ca
			a.Meta.Carry = ""
			plain := buildRig(&a)
			a.Meta.Carry = kind
			held := buildRig(&a)
			for _, vw := range []int{vFront, vSide, vBack} {
				p := plain.render(vw, poseFor(anim, 0))
				h := held.render(vw, poseFor(anim, 0))
				if bytes.Equal(p.p, h.p) {
					t.Errorf("carry %q does not change anim %d view %d", kind, anim, vw)
				}
			}
		}
	}
}
