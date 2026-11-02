package main

import "github.com/hajimehoshi/ebiten/v2"

// Characters edited in the editor replace the built-in drawing of the character
// with the same name wherever the game builds it (guests, famous groups, the kids...).

var charOverrides = map[string]*CharAsset{}
var overrideBase = map[string]*CharSet{}

func (g *Game) loadOverrides() {
	charOverrides = map[string]*CharAsset{}
	overrideBase = map[string]*CharSet{}
	if g.store == nil {
		return
	}
	for _, cat := range charCategories {
		for _, ca := range g.store.Chars[cat] {
			if ca.Meta.Edited && ca.Meta.Name != "" {
				charOverrides[ca.Meta.Name] = ca
			}
		}
	}
}

// overrideCharSet returns the edited version of the character s describes, or nil.
func overrideCharSet(s *CharSpec) *CharSet {
	ca := charOverrides[s.Name]
	if ca == nil {
		return nil
	}
	base := overrideBase[s.Name]
	if base == nil {
		base = newRigCharSet(ca)
		overrideBase[s.Name] = base
	}
	spec := *base.Spec
	spec.Ride, spec.RideCol, spec.Face = s.Ride, s.RideCol, s.Face
	if s.DisplayName != "" {
		spec.DisplayName = s.DisplayName
	}
	return &CharSet{Spec: &spec, cache: map[int]*ebiten.Image{}, rig: base.rig, over: base.over}
}
