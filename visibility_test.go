package main

import (
	"path/filepath"
	"testing"
)

func visOpaque(b *pixBuf) int {
	n := 0
	for i := 3; i < len(b.p); i += 4 {
		if b.p[i] > 0 {
			n++
		}
	}
	return n
}

// The guests of the real game are drawn from the asset folders (rigs). A guest that picks up a
// weapon used to be rebuilt from their nearly empty spec: an invisible body with just a knife.
func TestArmedGuestStaysVisible(t *testing.T) {
	for _, cat := range []string{"special_people", "people"} {
		names := spriteNames("assets", cat)
		for i, n := range names {
			if i%6 != 0 {
				continue
			}
			ca, err := loadCharAsset(spriteDirOf("assets", cat, n))
			if err != nil {
				t.Fatal(err)
			}
			base := newRigCharSet(ca)
			for _, hold := range []int{HoldKnife, HoldGun} {
				armed := armedSet(base, hold)
				for _, a := range []int{AnimIdle, AnimWalk, AnimStab} {
					for _, vw := range []int{vFront, vSide, vBack} {
						b0 := visOpaque(base.rigFrame(a, vw, 1))
						b1 := visOpaque(armed.rigFrame(a, vw, 1))
						if b1 < b0*7/10 || b1 < 60 {
							t.Errorf("%s/%s armed with %d: %d opaque pixels, unarmed %d", cat, n, hold, b1, b0)
						}
					}
				}
			}
		}
	}
}

// a spec that only carries a name (what an asset-drawn character has) must never paint nothing
func TestEmptySpecIsNotInvisible(t *testing.T) {
	for _, name := range []string{"hawaiian", "homer", "no_such_character", ""} {
		cs := newCharSet(&CharSpec{Name: name, Carry: HoldKnife})
		if cs.Spec.Skin.A == 0 {
			t.Fatalf("%q: still skinless", name)
		}
		for _, vw := range []int{vFront, vSide, vBack} {
			if n := visOpaque(genChar(cs.Spec, vw, poseFor(AnimStab, 1))); n < 150 {
				t.Errorf("%q view %d: only %d opaque pixels", name, vw, n)
			}
		}
	}
	_ = filepath.Join
}
