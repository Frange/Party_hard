package main

import "testing"

// snapshotPropKinds takes a deep copy of the prop registry and puts it back when the test ends.
// Every test that mutates propKinds (applyObjects, exportObjects, withAssetProps...) must call it first:
// props rebuilt from asset PNGs have no Draw func and would break any test that runs after.
func snapshotPropKinds(t *testing.T) {
	t.Helper()
	saved := make(map[string]*PropKind, len(propKinds))
	for n, k := range propKinds {
		if k == nil {
			saved[n] = nil
			continue
		}
		c := *k
		saved[n] = &c
	}
	t.Cleanup(func() {
		for n := range propKinds {
			delete(propKinds, n)
		}
		for n, k := range saved {
			propKinds[n] = k
		}
	})
}

// snapshotGlobals restores the debug/progress globals a test may touch.
func snapshotGlobals(t *testing.T) {
	t.Helper()
	pf, ds, dp, pg := progressFrozen, debugSnap, debugPrevFrozen, dbgPoliceGroups
	t.Cleanup(func() { progressFrozen, debugSnap, debugPrevFrozen, dbgPoliceGroups = pf, ds, dp, pg })
}
