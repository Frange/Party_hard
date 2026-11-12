package main

import "testing"

// With the objects of the assets folder loaded (what the real game and the editor do), every party of the campaign
// still builds: the portable toilet of the first export had no use points, ensureToilets then added booths with no
// toilet spot and the ESTACIÓN ESPACIAL failed its validation (campaignN 25 of 26). TestPaperContinueGoesStraightOn
// panicked after TestPaperFactsTwoPlayers (testLevels applies the assets and left campaignN at 25).
func TestAssetObjectsKeepEveryPartyBuilding(t *testing.T) {
	snapshotPropKinds(t)
	prevN, prevOv := campaignN, levelOverrides
	t.Cleanup(func() { campaignN, levelOverrides = prevN, prevOv })
	st := loadAssetStore()
	if st == nil {
		t.Skip("no assets folder")
	}
	applyObjects(st.Objs)
	if k := propKinds["portapotty"]; k == nil || len(k.Use) == 0 || k.UseKind != "toilet" {
		t.Fatalf("the exported portable toilet lost its use point: %+v", k)
	}
	if genNamed("station", 239, "") == nil {
		t.Fatal("the space station does not build with the assets loaded")
	}
	builtinLevels()
	if campaignN != len(campaignLevels) {
		t.Fatalf("only %d of %d parties build with the assets loaded", campaignN, len(campaignLevels))
	}
}
