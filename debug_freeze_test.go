package main

import "testing"

func TestDebugFreezeRestored(t *testing.T) {
	oldF, oldS, oldP := progressFrozen, debugSnap, debugPrevFrozen
	t.Cleanup(func() { progressFrozen, debugSnap, debugPrevFrozen = oldF, oldS, oldP })
	for _, prev := range []bool{true, false} {
		progressFrozen, debugSnap, debugPrevFrozen = prev, nil, false
		g := layerGame(t, false)
		g.debugBegin()
		if !progressFrozen {
			t.Fatalf("prev=%v: DEBUG mode must freeze", prev)
		}
		g.debugEnd()
		if progressFrozen != prev {
			t.Errorf("prev=%v: after debugEnd progressFrozen=%v", prev, progressFrozen)
		}
	}
}
