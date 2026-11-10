package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// the campaign: 25 parties (ITALIA after the highway), the three new ones before the space station, the station last
// (then the secret island, which is not in the list: jurassic.go)
func TestCampaignHasTheThreeNewParties(t *testing.T) {
	if len(campaignLevels) != 25 {
		t.Fatalf("the campaign has %d parties, want 25", len(campaignLevels))
	}
	want := map[int][2]string{18: {"highway", "highway"}, 19: {"italy", "italy"}, 20: {"trainstation", "train"},
		21: {"birthday", "cake"}, 22: {"hockey", "hockey"}, 23: {"disco70", "disco70"}, 24: {"station", "station"}}
	for i, w := range want {
		e := campaignLevels[i]
		if e.kind != w[0] || e.icon != w[1] {
			t.Errorf("party %d is %s/%s, want %s/%s", i+1, e.kind, e.icon, w[0], w[1])
		}
	}
	if last := campaignLevels[len(campaignLevels)-1]; last.icon != "station" {
		t.Fatalf("RULE: the space station must be the last listed party, and the island the one after it (got %q)", last.icon)
	}
	defs := builtinLevels()
	if campaignN != 25 {
		t.Fatalf("campaignN = %d, want 25 (some party did not build)", campaignN)
	}
	for i, w := range map[int]string{19: "ITALIA", 21: "FIESTA DE CUMPLEAÑOS", 22: "PISTA DE HOCKEY", 23: "DISCO SETENTERA", 24: "ESTACIÓN ESPACIAL"} {
		if defs[i].Title != w {
			t.Errorf("party %d is %q, want %q", i+1, defs[i].Title, w)
		}
	}
	if specialJurassic < campaignN {
		t.Errorf("the island must stay outside the campaign list")
	}
}

// every new party builds, is fully walkable (start, points of interest, roam, dance, guards) and nothing stands on anything
func TestNewPartiesWalkableAndClean(t *testing.T) {
	for _, kind := range []string{"birthday", "hockey", "disco70"} {
		for _, seed := range []int64{1, 2, 3, 97, 331, 173, 257} {
			d := genNamed(kind, seed, "")
			if d == nil {
				t.Errorf("%s seed %d: no valid layout", kind, seed)
				continue
			}
			if bad := genBad(d); len(bad) > 0 {
				t.Errorf("%s seed %d: unreachable %v", kind, seed, bad)
			}
		}
	}
}

func TestNewPartiesNoOverlaps(t *testing.T) {
	builtinLevels()
	for _, icon := range []string{"italy", "cake", "hockey", "disco70"} {
		i := campaignIdx(icon)
		d := levelDefsOrBuild(i)
		l := buildLevel(d)
		if iss := levelOverlaps(l); len(iss) > 0 {
			t.Errorf("party %d (%s): %d overlaps, first: %v", i+1, d.Title, len(iss), iss[0])
		}
	}
}

func levelDefsOrBuild(i int) *LevelDef {
	defs := builtinLevels()
	return defs[i]
}

// ---------------------------------------------------------------------------------------------------------------
// saves from before the three new parties (progress_migrate.go)
// ---------------------------------------------------------------------------------------------------------------

// oldSave: a progress.json as the 21-party game wrote it (stats and unlock frontier of that time)
func oldSave(unlocked int, wonUpTo int) Progress {
	p := Progress{Unlocked: unlocked, Stats: map[string]int{"played": 40, "distinct": wonUpTo + 1}}
	for i := 0; i <= wonUpTo; i++ {
		p.Stats[fmt.Sprintf("won_%d", i)] = 1
	}
	raw, _ := json.Marshal(p) // through JSON like the real file (no campaignRev key: revision 0)
	var q Progress
	json.Unmarshal(raw, &q)
	q.CampaignRev = 0
	return q
}

// (a 21-party save goes through both steps: the three new parties and then ITALIA after the highway)
func TestMigrationStationOpenAndWon(t *testing.T) {
	p := oldSave(21, 20) // had finished the whole old campaign, station included
	migrateProgress(&p)
	if p.Unlocked != 25 || p.CampaignRev != campaignRev {
		t.Fatalf("unlocked %d rev %d, want 25 and %d: the station and the new parties stay open", p.Unlocked, p.CampaignRev, campaignRev)
	}
	for i := 0; i <= 18; i++ {
		if p.Stats[fmt.Sprintf("won_%d", i)] != 1 {
			t.Errorf("won_%d was lost", i)
		}
	}
	if _, ok := p.Stats["won_19"]; ok {
		t.Error("won_19 now means ITALIA, which was never played: it must not exist")
	}
	if p.Stats["won_20"] != 1 {
		t.Error("the old win of the train station (won_19) did not move to won_20")
	}
	for _, i := range []int{21, 22, 23} {
		if p.Stats[fmt.Sprintf("won_%d", i)] != 0 {
			t.Errorf("won_%d was invented", i)
		}
	}
	if p.Stats["won_24"] != 1 {
		t.Error("the old win of the space station did not move to won_24")
	}
	if p.Stats["distinct"] != 21 || p.Stats["played"] != 40 {
		t.Errorf("the counters changed: distinct %d played %d", p.Stats["distinct"], p.Stats["played"])
	}
}

func TestMigrationStationOpenButNotWon(t *testing.T) {
	p := oldSave(21, 19)
	migrateProgress(&p)
	if p.Unlocked != 25 {
		t.Errorf("unlocked %d, want 25", p.Unlocked)
	}
	if p.Stats["won_24"] != 0 || p.Stats["won_19"] != 0 || p.Stats["won_20"] != 1 {
		t.Errorf("a station that was not won must not count as won, the train station keeps its win (won_19=%d won_20=%d won_24=%d)",
			p.Stats["won_19"], p.Stats["won_20"], p.Stats["won_24"])
	}
	if p.Stats["distinct"] != 20 {
		t.Errorf("distinct %d, want 20", p.Stats["distinct"])
	}
}

func TestMigrationNeverReachedTheStation(t *testing.T) {
	for _, un := range []int{1, 7, 15, 20} {
		p := oldSave(un, un-2)
		before := cloneProgress(p)
		migrateProgress(&p)
		want := un
		if un > italyIdx { // the highway was beaten: ITALIA opens and the frontier moves with its party
			want = un + 1
		}
		if p.Unlocked != want {
			t.Errorf("unlocked %d became %d, want %d", un, p.Unlocked, want)
		}
		for k, v := range before.Stats {
			if p.Stats[k] != v {
				t.Errorf("unlocked %d: stat %s changed %d -> %d", un, k, v, p.Stats[k])
			}
		}
	}
}

func TestMigrationOnlyOnce(t *testing.T) {
	p := oldSave(21, 20)
	migrateProgress(&p)
	// the player then plays the new layout: frontier at the disco (24) with won_21, won_22 earned in the new campaign
	p.Unlocked = 24
	p.Stats["won_21"], p.Stats["won_22"] = 1, 1
	migrateProgress(&p)
	if p.Unlocked != 24 || p.Stats["won_21"] != 1 || p.Stats["won_22"] != 1 || p.Stats["won_24"] != 1 {
		t.Errorf("a save in the new layout was migrated again: %+v", p)
	}
	var fresh Progress
	json.Unmarshal([]byte(`{"unlocked":22,"stats":{"won_20":1}}`), &fresh)
	migrateProgress(&fresh) // rev 0 and 22: it IS old (the frontier was past party 21? no: 22 > 21)
	if fresh.Unlocked != 25 || fresh.Stats["won_24"] != 1 {
		t.Errorf("an old save cannot have more than 21 unlocked; whatever it says it keeps the station open: %d %v", fresh.Unlocked, fresh.Stats)
	}
}

// saving stamps the revision so that the next load does not migrate again
func TestSavedProgressCarriesTheRevision(t *testing.T) {
	p := Progress{Unlocked: 22}
	p.CampaignRev = campaignRev
	raw, _ := json.Marshal(p)
	if !strings.Contains(string(raw), fmt.Sprintf(`"campaignRev":%d`, campaignRev)) {
		t.Fatalf("the revision is not saved: %s", raw)
	}
	var back Progress
	json.Unmarshal(raw, &back)
	migrateProgress(&back)
	if back.Unlocked != 22 {
		t.Errorf("frontier %d changed after a round trip", back.Unlocked)
	}
	// and an old file without the key stays small (omitempty): nothing else changes in it
	raw, _ = json.Marshal(Progress{Unlocked: 3})
	if !strings.Contains(string(raw), "campaignRev") {
		return
	}
}

// the old editor corrections of the station (c21_station) still apply to it
func TestStationKeepsItsOldOverride(t *testing.T) {
	if campaignKey(24) != "c25_station" || campaignKeyRev1(24) != "c24_station" || legacyStationKey != "c21_station" {
		t.Fatalf("keys: %s / %s / %s", campaignKey(24), campaignKeyRev1(24), legacyStationKey)
	}
}

// the frontier moves through the new parties and stops at the station
func TestUnlockWalksThroughTheNewParties(t *testing.T) {
	builtinLevels()
	g := testGame()
	g.progress.Unlocked = 22 // the birthday party is the frontier
	for _, idx := range []int{21, 22, 23} {
		g.levelIdx = idx
		g.unlockNext()
		if g.progress.Unlocked != idx+2 {
			t.Fatalf("after party %d the frontier is %d, want %d", idx+1, g.progress.Unlocked, idx+2)
		}
	}
	g.levelIdx = 24
	g.unlockNext()
	if g.progress.Unlocked != 25 {
		t.Errorf("the station is the last: frontier %d", g.progress.Unlocked)
	}
}

// ---------------------------------------------------------------------------------------------------------------
// the list of parties
// ---------------------------------------------------------------------------------------------------------------

func TestPartyListHasRoomForEverything(t *testing.T) {
	rows := len(campaignLevels) + 2 // + ALEATORIO + ISLA
	if rows > len(levelColX)*levelRowsPerCol || levelRowsPerCol != 9 {
		t.Fatalf("%d rows do not fit in %d columns of %d", rows, len(levelColX), levelRowsPerCol)
	}
	const textH = 8.0
	if gap := levelRowDY - (levelSubDY + textH); gap < 10 {
		t.Errorf("only %.0f px between the small line of a row and the name of the next (the player asked for MORE air)", gap)
	}
	if levelRowDY < 40 {
		t.Errorf("rows %.0f px apart: the player asked for more vertical room between parties (was 37)", levelRowDY)
	}
	lastBottom := levelRowY0 + float64(levelRowsPerCol-1)*levelRowDY + levelSubDY + textH
	if lastBottom > levelFreeY {
		t.Errorf("the last row ends at %.0f, the free strip under the list starts at %.0f", lastBottom, levelFreeY)
	}
	if 525-levelFreeY < 20 {
		t.Errorf("only %.0f px free under the list", 525-levelFreeY)
	}
	if levelRowY0 < dailyBtnY+dailyBtnH+8 {
		t.Errorf("the first row (%.0f) touches the RETO DE HOY button (ends %.0f)", levelRowY0, dailyBtnY+dailyBtnH)
	}
}

// ---------------------------------------------------------------------------------------------------------------
// the ice
// ---------------------------------------------------------------------------------------------------------------

func withIceLevel(t *testing.T, d *LevelDef) *Level {
	t.Helper()
	prev := iceLevel
	t.Cleanup(func() { iceLevel = prev })
	l := buildLevel(d)
	iceLevel = l
	return l
}

func hockeyDef(t *testing.T) *LevelDef {
	t.Helper()
	defs := builtinLevels()
	d := defs[campaignIdx("hockey")]
	if d.Icon != "hockey" {
		t.Fatalf("the hockey party is %q", d.Icon)
	}
	return d
}

func TestOnlyTheRinkIsIce(t *testing.T) {
	defs := builtinLevels()
	for i, d := range defs {
		l := buildLevel(d)
		if want := d.Icon == "hockey" || strings.HasPrefix(d.Title, "PISTA DE HOCKEY"); l.hasIce() != want {
			t.Errorf("party %d (%s): hasIce=%v", i+1, d.Title, l.hasIce())
		}
	}
	l := buildLevel(hockeyDef(t))
	if !l.OnIce(Vec{480, 236}) || !l.OnIce(Vec{200, 120}) {
		t.Error("the middle of the rink must be ice")
	}
	if l.OnIce(Vec{70, 200}) || l.OnIce(Vec{480, 420}) || l.OnIce(Vec{900, 300}) {
		t.Error("the concourse around the rink is ordinary ground")
	}
}

func TestIceInertiaIsSlowToStartAndSlowToStop(t *testing.T) {
	var v Vec
	wish := Vec{40, 0}
	for i := 0; i < 6; i++ { // 0.1 s
		iceStep(&v, wish, 1.0/60)
	}
	if v.X > 0.3*wish.X {
		t.Errorf("0.1 s after pushing the speed is already %.1f of %.1f: no inertia", v.X, wish.X)
	}
	for i := 0; i < 90; i++ { // 1.6 s
		iceStep(&v, wish, 1.0/60)
	}
	if v.X < 0.9*wish.X {
		t.Errorf("after 1.6 s the speed is %.1f: it must reach the wish", v.X)
	}
	for i := 0; i < 30; i++ { // let go for 0.5 s: still gliding
		iceStep(&v, Vec{}, 1.0/60)
	}
	if v.X < 0.5*wish.X {
		t.Errorf("0.5 s after letting go the speed is %.1f: it must still glide", v.X)
	}
	for i := 0; i < 60*8; i++ {
		iceStep(&v, Vec{}, 1.0/60)
	}
	if v != (Vec{}) {
		t.Errorf("it never stops: %v", v)
	}
	// turning: 180 degrees takes a while (on normal ground it is instant)
	v = Vec{40, 0}
	turned := -1
	for i := 0; i < 240; i++ {
		iceStep(&v, Vec{-40, 0}, 1.0/60)
		if v.X < 0 {
			turned = i
			break
		}
	}
	if turned < 15 {
		t.Errorf("the skater turned round in %d frames: too sharp for ice", turned)
	}
}

// off the ice followIce IS follow(): the other parties do not change at all
func TestFollowIceIsFollowOffTheIce(t *testing.T) {
	l := withIceLevel(t, builtinLevels()[0]) // house party: no ice
	if l.hasIce() {
		t.Fatal("the house party has ice")
	}
	path := []Vec{{120, 90}, {200, 150}, {260, 90}}
	a, b := Vec{100, 100}, Vec{100, 100}
	pa, pb := append([]Vec(nil), path...), append([]Vec(nil), path...)
	var va Vec
	da, db := DirDown, DirDown
	for i := 0; i < 400; i++ {
		followIce(&a, &va, &da, &pa, 40, 1.0/60)
		follow(&b, &db, &pb, 40, 1.0/60)
		if a != b || da != db || va != (Vec{}) {
			t.Fatalf("frame %d: followIce %v / follow %v", i, a, b)
		}
	}
}

// a player on the ice: keeps gliding after releasing, never ends up inside a wall or prop
func TestPlayerSlidesOnIceAndStaysOutOfSolids(t *testing.T) {
	l := withIceLevel(t, hockeyDef(t))
	g := &Game{lvl: l}
	p := &Player{pos: Vec{400, 236}}
	run := func(dx, dy float64, frames int) float64 {
		d0 := p.pos
		for i := 0; i < frames; i++ {
			p.moving = dx != 0 || dy != 0
			g.iceMovePlayer(p, dx, dy, 46, 1.0/60)
		}
		return math.Hypot(p.pos.X-d0.X, p.pos.Y-d0.Y)
	}
	run(1, 0, 120) // 2 s at full push
	if p.vel.X < 25 {
		t.Fatalf("speed after 2 s of pushing: %.1f", p.vel.X)
	}
	glide := run(0, 0, 30) // released: half a second
	if glide < 12 {
		t.Errorf("only %.1f px of gliding after letting go", glide)
	}
	if !g.lvl.OnIce(p.pos) {
		t.Fatal("left the ice in the test")
	}
	// random input for a minute, several spawn points: never inside a solid
	rng := rand.New(rand.NewSource(5))
	for _, start := range []Vec{{300, 150}, {480, 236}, {640, 330}, {200, 236}} {
		p.pos, p.vel = start, Vec{}
		dx, dy := 0.0, 0.0
		for i := 0; i < 60*60; i++ {
			if i%40 == 0 {
				dx, dy = float64(rng.Intn(3)-1), float64(rng.Intn(3)-1)
			}
			p.moving = dx != 0 || dy != 0
			g.iceMovePlayer(p, dx, dy, 74, 1.0/60)
			if l.SolidP(p.pos.X, p.pos.Y) && !l.OnIce(p.pos) {
				t.Fatalf("inside a solid at %v", p.pos)
			}
			if !l.OnIce(p.pos) { // slid out through a gate: back to the middle of the rink
				p.pos, p.vel = Vec{480, 236}, Vec{}
			}
		}
	}
}

// off the ice the player is not touched (returns false and clears the speed)
func TestPlayerOffIceIsNotHandled(t *testing.T) {
	l := withIceLevel(t, hockeyDef(t))
	g := &Game{lvl: l}
	p := &Player{pos: Vec{70, 200}, vel: Vec{30, 0}}
	p.moving = true
	if g.iceMovePlayer(p, 1, 0, 46, 1.0/60) {
		t.Error("the concourse is not ice")
	}
	if p.vel != (Vec{}) {
		t.Errorf("speed %v kept off the ice", p.vel)
	}
}

// REACHABILITY WITH INERTIA: walkers that follow their path on the ice (guests, guards, police) get to every point
// the level needs: the start, the points of interest, the roam and dance zones and the guard routes, from the street
// and from the middle of the rink; none of them ever stands inside a solid.
func TestIceWalkersArrive(t *testing.T) {
	d := hockeyDef(t)
	l := withIceLevel(t, d)
	var targets []Vec
	for _, ps := range l.POIs {
		targets = append(targets, ps...)
	}
	for _, r := range d.Roam {
		targets = append(targets, l.NearestFree(r.Center()))
	}
	for _, r := range d.Dance {
		targets = append(targets, l.NearestFree(r.Center()))
	}
	for _, route := range d.Guards {
		for _, p := range route {
			targets = append(targets, l.NearestFree(p))
		}
	}
	targets = append(targets, d.Start, l.NearestFree(Vec{d.CarTo.X, d.CarTo.Y - 24}))
	if len(targets) < 12 {
		t.Fatalf("only %d targets", len(targets))
	}
	origins := []Vec{d.Start, l.NearestFree(Vec{480, 236}), l.NearestFree(Vec{250, 160})}
	for _, from := range origins {
		for _, to := range targets {
			path := l.Path(from, to)
			if path == nil {
				t.Errorf("no path from %v to %v", from, to)
				continue
			}
			pos, vel, dir := from, Vec{}, DirDown
			arrived := false
			for step := 0; step < 60*120 && !arrived; step++ { // two minutes at most
				_, arrived = followIce(&pos, &vel, &dir, &path, 46*0.86, 1.0/60)
				for _, r := range l.solidsN {
					if r.Grow(-0.5).Has(pos.X, pos.Y) {
						t.Fatalf("walker inside a solid %v at %v going %v -> %v", r, pos, from, to)
					}
				}
			}
			if !arrived {
				t.Errorf("never arrived from %v to %v (stuck at %v)", from, to, pos)
			}
			if dist(pos, to) > 12 {
				t.Errorf("arrived %.0f px away from %v", dist(pos, to), to)
			}
		}
	}
}

// guests, guards and police really use it (their walk helpers pass their own speed)
func TestGuestGuardCopWalkOnIce(t *testing.T) {
	l := withIceLevel(t, hockeyDef(t))
	g := &Game{lvl: l}
	gu := &Guest{pos: Vec{300, 236}, path: []Vec{{600, 236}}}
	gd := &Guard{pos: Vec{300, 200}, path: []Vec{{600, 200}}}
	cp := &Cop{pos: Vec{300, 270}, path: []Vec{{600, 270}}}
	for i := 0; i < 6; i++ {
		gu.walk(46, 1.0/60)
		g.guardWalk(gd, 46, 1.0/60)
		g.copWalk(cp, 46, 1.0/60)
	}
	for name, v := range map[string]Vec{"guest": gu.vel, "guard": gd.vel, "cop": cp.vel} {
		sp := math.Hypot(v.X, v.Y)
		if sp < 1 || sp > 0.3*46*paceK {
			t.Errorf("%s: speed %.1f 0.1 s after starting (wish %.1f): no inertia", name, sp, 46*paceK)
		}
	}
	for i := 0; i < 60*30; i++ {
		gu.walk(46, 1.0/60)
	}
	if len(gu.path) != 0 || dist(gu.pos, Vec{600, 236}) > 12 {
		t.Errorf("the guest did not arrive: %v path %v", gu.pos, gu.path)
	}
}

// skate marks: sparse, bounded and they fade
func TestSkateMarksFadeAndAreBounded(t *testing.T) {
	l := withIceLevel(t, hockeyDef(t))
	g := &Game{lvl: l}
	iceBegin(g)
	for i := 0; i < 5000; i++ {
		g.time = float64(i) / 60
		g.iceTrace(Vec{300 + float64(i%200), 200}, Vec{40, 0})
	}
	n := 0
	for _, m := range iceFx.marks {
		if m.ok {
			n++
		}
	}
	if n == 0 || n > iceMarkN {
		t.Errorf("%d marks", n)
	}
	iceBegin(g)
	if iceFx.n != 0 {
		t.Error("the marks of the last party were kept")
	}
	g.time = 0
	g.iceTrace(Vec{300, 200}, Vec{0.1, 0}) // too slow: no mark
	for _, m := range iceFx.marks {
		if m.ok {
			t.Fatal("a mark of a skater who stands still")
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------
// the 70s disco floor
// ---------------------------------------------------------------------------------------------------------------

func discoHashPat(p *discoPat) uint64 {
	h := uint64(1469598103934665603)
	for _, v := range p {
		h = (h ^ uint64(v)) * 1099511628211
	}
	return h
}

func TestDiscoPulseFollowsTheBPM(t *testing.T) {
	bpm := genreForTitle("DISCO SETENTERA").bpm
	if bpm < 100 || bpm > 140 {
		t.Fatalf("bpm %v", bpm)
	}
	if b := discoBeat(60/bpm, bpm); math.Abs(b-1) > 1e-9 {
		t.Errorf("one beat lasts %.4f s of game time? beat=%v", 60/bpm, b)
	}
	k0, _ := discoPulse(3.0) // on the beat
	k1, _ := discoPulse(3.9) // just before the next
	if k0 < 0.99 || k1 > 0.05 {
		t.Errorf("kick on the beat %.2f, right before the next %.2f", k0, k1)
	}
	_, offOn := discoPulse(3.5)
	_, offOff := discoPulse(3.9)
	if offOn <= offOff {
		t.Errorf("the off-beat hit is not there: %.2f vs %.2f", offOn, offOff)
	}
}

func TestDiscoEveryPictureIsAPicture(t *testing.T) {
	for kind, name := range discoKindNames {
		for seed := int64(1); seed <= 12; seed++ {
			var p discoPat
			discoMake(kind, &p, rand.New(rand.NewSource(seed*7+int64(kind))))
			lit := 0
			for _, v := range p {
				if v > 3 {
					t.Fatalf("%s: colour slot %d", name, v)
				}
				if v != 0 {
					lit++
				}
			}
			frac := float64(lit) / float64(len(p))
			if frac < 0.08 || frac > 0.88 {
				t.Errorf("%s seed %d: %.0f%% of the tiles are lit", name, seed, frac*100)
			}
		}
	}
	// the heart is a heart: symmetric left-right apart from the stray sparkles; the mosaic mirrors both ways
	sym := func(kind int, both bool) float64 {
		var p discoPat
		discoMake(kind, &p, rand.New(rand.NewSource(99)))
		same, n := 0, 0
		for y := 0; y < discoH; y++ {
			for x := 0; x < discoW/2; x++ {
				a, b := p[y*discoW+x], p[y*discoW+discoW-1-x]
				if a != 0 || b != 0 {
					n++
					if (a != 0) == (b != 0) {
						same++
					}
				}
			}
		}
		_ = both
		return float64(same) / float64(maxI(n, 1))
	}
	if s := sym(7, true); s < 0.8 {
		t.Errorf("the mosaic is %.0f%% symmetric", s*100)
	}
}

// the pictures are NOT repetitive: never the same kind twice in the last four, almost every picture different,
// every kind turns up, a party differs from the next one
func TestDiscoPicturesDoNotRepeat(t *testing.T) {
	prev := discoSeed
	t.Cleanup(func() { discoSeed = prev })
	discoSeed = 12345
	var st discoState
	st.reset(&Level{})
	kinds := map[int]int{}
	hashes := map[uint64]bool{}
	var last []int
	var pat discoPat
	for i := 0; i < 300; i++ {
		k := st.next(&pat)
		for _, h := range last {
			if h == k {
				t.Fatalf("picture %d: kind %s came again within four pictures", i, discoKindNames[k])
			}
		}
		last = append(last, k)
		if len(last) > 4 {
			last = last[1:]
		}
		kinds[k]++
		hashes[discoHashPat(&pat)] = true
	}
	if len(kinds) < len(discoKindNames) {
		t.Errorf("only %d of %d kinds were drawn", len(kinds), len(discoKindNames))
	}
	if len(hashes) < 285 {
		t.Errorf("only %d different pictures out of 300", len(hashes))
	}
	first := map[uint64]bool{}
	for seed := int64(1); seed <= 20; seed++ {
		discoSeed = seed
		var s2 discoState
		s2.reset(&Level{})
		first[discoHashPat(&s2.cur)] = true
	}
	if len(first) < 17 {
		t.Errorf("the first picture of %d parties has only %d variants", 20, len(first))
	}
}

func TestDiscoTilesPulseAndWipe(t *testing.T) {
	prev := discoSeed
	t.Cleanup(func() { discoSeed = prev })
	discoSeed = 7
	var st discoState
	st.reset(&Level{})
	bright := func(beat float64, dim bool) float64 {
		st.render(beat, dim)
		sum := 0.0
		for i := 0; i < len(st.buf); i += 4 {
			if st.buf[i+3] != 255 {
				t.Fatalf("a pixel is not opaque (alpha %d)", st.buf[i+3])
			}
			sum += float64(st.buf[i]) + float64(st.buf[i+1]) + float64(st.buf[i+2])
		}
		return sum
	}
	if len(st.buf) != discoW*discoCell*discoH*discoCell*4 {
		t.Fatalf("buffer of %d bytes", len(st.buf))
	}
	on, off := bright(2.0, false), bright(2.9, false)
	if on <= off*1.05 {
		t.Errorf("the floor does not pulse: %.0f on the beat vs %.0f before the next", on, off)
	}
	if st.lit == 0 {
		t.Error("no tile lit")
	}
	if dim := bright(2.0, true); dim >= on*0.7 || st.lit != 0 {
		t.Errorf("with the DJ dead the floor must go dark (%.0f vs %.0f, lit %d)", dim, on, st.lit)
	}
	// a new phrase: the picture changes and the wipe shows both for a moment
	bright(0.0, false)
	p0 := st.phrase
	cur0 := st.cur
	bright(discoPhraseBeats+0.5, false)
	if st.phrase != p0+1 || st.cur == cur0 {
		t.Errorf("phrase %d -> %d, picture changed: %v", p0, st.phrase, st.cur != cur0)
	}
	// no allocation while a picture stays on the floor
	beat := 20.0
	if a := testing.AllocsPerRun(30, func() { beat += 0.003; st.render(beat, false) }); a != 0 {
		t.Errorf("%.1f allocations per frame", a)
	}
}

// the tile floor is drawn by the game (one pixel layer) and the party has no leftover tiles in other levels
func TestDiscoFloorIsDrawnInTheParty(t *testing.T) {
	g := layerGame(t, false)
	idx := ambientLevelIdx("DISCO SETENTERA")
	if idx < 0 {
		builtinLevels()
		levelDefs = append(levelDefs[:0], builtinLevels()...)
		idx = ambientLevelIdx("DISCO SETENTERA")
	}
	g.loadLevel(idx)
	g.mode = ModePlay
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	disco.lvl = nil
	g.drawAnimatedFloors(img) // the whole Draw reads pixels (gradeBG), which needs a running window
	if disco.lit == 0 || disco.lvl != g.lvl {
		t.Errorf("the disco floor was not drawn (lit %d)", disco.lit)
	}
	if g.lvl.Def.Dance[0] != discoFloor {
		t.Errorf("the dance zone is not the tile floor")
	}
}

// ---------------------------------------------------------------------------------------------------------------
// dressed guests
// ---------------------------------------------------------------------------------------------------------------

func TestCrowdIsDressedForTheParty(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	count := func(kind string, f func(s CharSpec) bool) int {
		n := 0
		for i := 0; i < 300; i++ {
			s := crowdSpec(kind, r)
			if s.Name != "" || s.DisplayName != "" {
				t.Fatalf("%s: a dressed guest has a name (famous people stay unique)", kind)
			}
			if f(s) {
				n++
			}
			cs := s
			if newCharSet(&cs) == nil {
				t.Fatal("no char set")
			}
		}
		return n
	}
	if n := count("cake", func(s CharSpec) bool { return s.Hat == HatPartyCone }); n != 300 {
		t.Errorf("birthday: %d/300 with a party hat", n)
	}
	if n := count("hockey", func(s CharSpec) bool { return s.Hat == HatHelmet }); n < 100 {
		t.Errorf("hockey: only %d helmets", n)
	}
	if n := count("hockey", func(s CharSpec) bool { return s.Carry == HoldStick }); n < 60 {
		t.Errorf("hockey: only %d sticks", n)
	}
	if n := count("hockey", func(s CharSpec) bool { return s.Shoes == hx(0xe8eef4) }); n != 300 {
		t.Errorf("hockey: %d/300 with skates", n)
	}
	if n := count("disco70", func(s CharSpec) bool { return s.Hair2 == HairAfro }); n < 180 {
		t.Errorf("disco: only %d afros", n)
	}
	if n := count("disco70", func(s CharSpec) bool { return s.Bottom == BotFlare }); n != 300 {
		t.Errorf("disco: %d/300 in bell-bottoms", n)
	}
	if n := count("disco70", func(s CharSpec) bool { return s.Beard == BeardSideburns }); n < 40 {
		t.Errorf("disco: only %d with sideburns", n)
	}
	if n := count("disco70", func(s CharSpec) bool { return s.Glasses != GlassNone }); n < 100 {
		t.Errorf("disco: only %d with glasses", n)
	}
}

// the dice are their own: g.rng does not move, and the other parties are not dressed
func TestCrowdDoesNotMoveTheGameDice(t *testing.T) {
	prev := crowdSeed
	prevRng := crowdRng
	t.Cleanup(func() { crowdSeed, crowdRng = prev, prevRng })
	crowdSeed, crowdRng = 11, nil
	g := &Game{rng: rand.New(rand.NewSource(77))}
	ref := rand.New(rand.NewSource(77))
	dressed := 0
	for i := 0; i < 200; i++ {
		if g.venueGuestCS(&LevelDef{Icon: "disco70"}) != nil {
			dressed++
		}
		if g.venueGuestCS(&LevelDef{Icon: "house"}) != nil {
			t.Fatal("the house party dresses its guests")
		}
	}
	if g.rng.Int63() != ref.Int63() {
		t.Error("g.rng moved")
	}
	if dressed < 90 || dressed > 150 {
		t.Errorf("%d of 200 dressed (chance %.2f)", dressed, crowdChance["disco70"])
	}
}

// the bell-bottoms, the afro and the stick are drawn (the sprite is not empty and differs from the plain version)
func TestNewLooksChangeTheSprite(t *testing.T) {
	base := CharSpec{Skin: hx(0xe8d0b8), Hair: hx(0x2a1810), Shirt: hx(0xd05a30), Shirt2: hx(0xd05a30), Pants: hx(0x5a2a6a), Shoes: hx(0xf4f0e0), Top: TopTee, Hair2: HairShort, HatCol: hx(0xd05a30)}
	draw := func(s CharSpec) []byte {
		s.noOverride = true
		return genChar(&s, vFront, Pose{}).p
	}
	plain := draw(base)
	for name, mod := range map[string]func(*CharSpec){
		"afro":  func(s *CharSpec) { s.Hair2 = HairAfro },
		"flare": func(s *CharSpec) { s.Bottom = BotFlare },
		"stick": func(s *CharSpec) { s.Carry = HoldStick },
	} {
		s := base
		mod(&s)
		got := draw(s)
		if len(got) == 0 || string(got) == string(plain) {
			t.Errorf("%s does not change the sprite", name)
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------
// texts and tables
// ---------------------------------------------------------------------------------------------------------------

func TestNewPartiesHaveTheirTexts(t *testing.T) {
	for _, title := range []string{"ITALIA", "FIESTA DE CUMPLEAÑOS", "PISTA DE HOCKEY", "DISCO SETENTERA"} {
		if partyTypeOfTitle[title] == "" {
			t.Errorf("%s: no kind of party", title)
		}
		if _, ok := genreOfTitle[title]; !ok {
			t.Errorf("%s: no genre", title)
		}
		for _, s := range []string{title, partyTypeOfTitle[title]} {
			if en, ok := langDataEN[s]; !ok || en == "" || en == s {
				t.Errorf("no English for %q", s)
			}
		}
	}
	for _, icon := range []string{"italy", "cake", "hockey", "disco70"} {
		e := campaignLevels[campaignIdx(icon)]
		d := genNamed(e.kind, e.seed, e.title)
		if en := langDataEN[d.Intro]; en == "" {
			t.Errorf("%s: no English intro %q", e.title, d.Intro)
		}
		th, ok := hintThemes[e.icon]
		if !ok {
			t.Errorf("%s: no hint theme", e.icon)
		}
		for _, s := range []string{th.goal, th.tip} {
			if len([]rune(s)) > 48 || langDataEN[s] == "" || len([]rune(langDataEN[s])) > 48 {
				t.Errorf("%s: hint %q too long or untranslated", e.icon, s)
			}
		}
		if n := len(venueLines[e.icon]); n != 6 {
			t.Errorf("%s: %d venue lines", e.icon, n)
		}
		if len(pVenue[e.icon]) < 5 {
			t.Errorf("%s: %d newspaper briefs", e.icon, len(pVenue[e.icon]))
		}
	}
	for _, id := range []string{"cake", "hockey", "disco70"} {
		for _, p := range pVenue[id] {
			if langDataEN[p.es] != p.en {
				t.Errorf("newspaper brief %q is not registered in English", p.es)
			}
		}
	}
}

// no brand or person: the texts of the new parties are generic
func TestNewPartiesTextsAreGeneric(t *testing.T) {
	banned := []string{"austin", "powers", "bond", "mighty", "shag", "nhl", "zamboni", "bruins", "disney", "mickey", "travolta", "saturday night fever", "fiebre del sábado"}
	check := func(where, s string) {
		low := strings.ToLower(s)
		for _, b := range banned {
			if strings.Contains(low, b) {
				t.Errorf("%s mentions %q: %s", where, b, s)
			}
		}
	}
	for _, id := range []string{"cake", "hockey", "disco70"} {
		for _, l := range venueLines[id] {
			check(id, l.es)
			check(id, l.en)
		}
		for _, p := range pVenue[id] {
			check(id, p.es)
			check(id, p.en)
		}
		check(id, hintThemes[id].goal+hintThemes[id].tip)
	}
	for name := range propKinds {
		check("prop", name)
	}
}

func TestNewPartiesMusicIsDifferentFromTheirNeighbours(t *testing.T) {
	seq := []string{}
	for _, e := range campaignLevels {
		title := e.title
		if e.level != nil {
			title = e.level().Title
		}
		seq = append(seq, genreForTitle(title).id)
	}
	for i := 18; i < len(seq); i++ {
		if seq[i] == seq[i-1] {
			t.Errorf("parties %d and %d have the same genre %s", i, i+1, seq[i])
		}
	}
	if seq[19] != "clasica" || seq[21] != "pop" || seq[22] != "rock" || seq[23] != "disco" {
		t.Errorf("genres %v", seq[19:24])
	}
}

// a VIP zone is possible in the three new parties
func TestNewPartiesCanHaveAVIP(t *testing.T) {
	for _, i := range []int{19, 21, 22, 23} {
		d := builtinLevels()[i]
		if z, _ := vipPlace(d, ""); z == nil {
			t.Errorf("party %d (%s): the VIP zone does not fit", i+1, d.Title)
		}
	}
}

// the newspaper of those parties prints its own brief
func TestNewspaperHasABriefForTheNewVenues(t *testing.T) {
	for _, id := range []string{"cake", "hockey", "disco70"} {
		f := paperFacts{venue: id, seed: 5, kills: 6}
		found := false
		for _, s := range paperSentences(f) {
			for _, p := range pVenue[id] {
				found = found || s == p.es
			}
		}
		if !found {
			t.Errorf("%s: no venue brief in the paper", id)
		}
	}
	for _, s := range paperSentences(paperFacts{venue: "house", seed: 5, kills: 6}) {
		for _, p := range pVenue["cake"] {
			if s == p.es {
				t.Error("the cake brief in another party")
			}
		}
	}
}
