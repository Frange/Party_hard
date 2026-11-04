package main

import (
	"os"
	"path/filepath"
	"testing"
)

func setCell(rows []string, cx, cy int, ch byte) []string {
	out := append([]string(nil), rows...)
	b := []byte(out[cy])
	b[cx] = ch
	out[cy] = string(b)
	return out
}

func coveredBy(rects []Rect, x, y float64) bool {
	for _, r := range rects {
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return true
		}
	}
	return false
}

// an override changes only what the user changed: props, walls, zones; the rest stays generated
func TestLevelOverrideApplies(t *testing.T) {
	idx := 8 // stadium
	def := resolveCampaign(campaignLevels[idx])
	if def == nil {
		t.Skip("scenario did not build")
	}
	la := levelAssetFromDef(def, campaignKey(idx))
	la.Meta.Override = campaignKey(idx)
	nProps := len(la.Meta.Props)
	if nProps < 3 {
		t.Fatal("stadium has too few props")
	}
	// drop the first prop, add a barrel
	la.Meta.Props = append(la.Meta.Props[1:], LevelProp{Kind: "barrel", X: 480, Y: 440})
	// a new wall block in a free spot (cells 100..103 x 20..23 => design 400..416 x 80..92) and a hole in an existing wall
	free := func() (int, int) {
		for cy := 4; cy < len(la.Meta.Solid)-4; cy++ {
			for cx := 4; cx < len(la.Meta.Solid[cy])-4; cx++ {
				if la.Meta.Solid[cy][cx] == '.' {
					return cx, cy
				}
			}
		}
		return -1, -1
	}
	fx, fy := free()
	la.Meta.Solid = setCell(la.Meta.Solid, fx, fy, '#')
	wx, wy := -1, -1
	for cy := range la.Meta.Solid {
		for cx := range la.Meta.Solid[cy] {
			if la.Meta.Solid[cy][cx] == '#' && wx < 0 && !(cx == fx && cy == fy) {
				wx, wy = cx, cy
			}
		}
	}
	if wx < 0 {
		t.Skip("no wall to cut in this scenario")
	}
	la.Meta.Solid = setCell(la.Meta.Solid, wx, wy, '.')
	// a new walkable cell in the zone grid
	la.Meta.Zone = setCell(la.Meta.Zone, fx, fy, 'r')

	d2 := resolveCampaign(campaignLevels[idx])
	wallsBefore := len(d2.Walls)
	title, hasHull := d2.Title, d2.Hull
	applyLevelOverride(d2, la)
	if len(d2.Props) != nProps { // one removed, one added
		t.Fatalf("props: %d, want %d", len(d2.Props), nProps)
	}
	k := float64(logicCell) / wk
	var wallRects []Rect
	for _, w := range d2.Walls {
		wallRects = append(wallRects, w.R)
	}
	if !coveredBy(wallRects, float64(fx)*k+k/2, float64(fy)*k+k/2) {
		t.Fatal("new wall cell is not solid")
	}
	if coveredBy(wallRects, float64(wx)*k+k/2, float64(wy)*k+k/2) {
		t.Fatal("removed wall cell is still solid")
	}
	if !coveredBy(d2.Roam, float64(fx)*k+k/2, float64(fy)*k+k/2) {
		t.Fatal("new walkable cell is not in the roam zone")
	}
	if len(d2.Walls) == 0 || wallsBefore == 0 || d2.Title != title || d2.Hull != hasHull {
		t.Fatal("untouched parts of the scenario changed")
	}
	// an untouched override changes nothing at all
	same := levelAssetFromDef(resolveCampaign(campaignLevels[idx]), campaignKey(idx))
	d3 := resolveCampaign(campaignLevels[idx])
	n, p, r := len(d3.Walls), len(d3.Props), len(d3.Roam)
	applyLevelOverride(d3, same)
	if len(d3.Walls) != n || len(d3.Props) != p || len(d3.Roam) != r {
		t.Fatalf("untouched override changed the scenario: walls %d->%d props %d->%d roam %d->%d", n, len(d3.Walls), p, len(d3.Props), r, len(d3.Roam))
	}
}

// overrides live beside the user's own scenarios without making them replace the campaign
func TestOverridesDoNotReplaceCampaign(t *testing.T) {
	ov := &LevelAsset{Meta: LevelMeta{Name: campaignKey(2), Override: campaignKey(2)}}
	mine := &LevelAsset{Meta: LevelMeta{Name: "mio"}}
	custom, over := splitLevels([]*LevelAsset{ov, mine})
	if len(custom) != 1 || custom[0] != mine || over[campaignKey(2)] != ov {
		t.Fatal("split wrong")
	}
	if custom, _ = splitLevels([]*LevelAsset{ov}); len(custom) != 0 {
		t.Fatal("an override alone must not count as a custom scenario")
	}
	if !isCampaignKey(campaignKey(0)) || isCampaignKey("mio") {
		t.Fatal("isCampaignKey wrong")
	}
}

// saved override is read back from disk with its key, and the editor's list hides it
func TestOverrideRoundTripOnDisk(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assets")
	la := levelAssetFromDef(&LevelDef{Title: "X", W: 960, H: 540}, campaignKey(0))
	la.Meta.Override = campaignKey(0)
	if err := saveLevelAsset(root, la); err != nil {
		t.Fatal(err)
	}
	back, err := loadLevelAsset(filepath.Join(root, "backgrounds", campaignKey(0)))
	if err != nil || back.Meta.Override != campaignKey(0) {
		t.Fatalf("override key lost: %v %v", err, back)
	}
	if _, err := os.Stat(filepath.Join(root, "backgrounds", campaignKey(0), "level.json")); err != nil {
		t.Fatal(err)
	}
}
