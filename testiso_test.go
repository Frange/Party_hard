package main

import (
	"os"
	"sync"
	"testing"
	"time"
	_ "unsafe" // go:linkname
)

// Under go test the graveyard acts and the graffiti event stay off unless a test asks for them (gyInTests,
// gfInTests): the other tests' parties stay as they were. Production code never asks whether it runs in a test.
func init() { gyTestsOff, gfTestsOff = true, true }

// ---- ebiten's deferred draw queue (memory of the test binary) ----
// The tests never start ebiten's game loop (no window): every DrawImage, WritePixels or Deallocate made outside a
// frame is queued by ebiten (internal/atlas, appendDeferred) to run at the next BeginFrame, which never comes in a
// test. Nothing reads those images back (ReadPixels panics before the game starts), but the queue keeps a copy of
// every vertex and pixel buffer and every image alive: about 10 GB after a few hundred tests that load levels
// (renderBG, groundDetail), and macOS killed the full pass ("signal: killed"). The queue is dropped every few
// seconds: it can never run in a test, so nothing observable changes, and the memory goes back to the GC.

//go:linkname ebitenDeferred github.com/hajimehoshi/ebiten/v2/internal/atlas.deferred
var ebitenDeferred []func()

//go:linkname ebitenDeferredM github.com/hajimehoshi/ebiten/v2/internal/atlas.deferredM
var ebitenDeferredM sync.Mutex

// dropEbitenDeferred discards the draw commands queued for a frame that will never come; it reports how many.
func dropEbitenDeferred() int {
	ebitenDeferredM.Lock()
	n := len(ebitenDeferred)
	ebitenDeferred = nil
	ebitenDeferredM.Unlock()
	return n
}

func TestMain(m *testing.M) {
	stop := make(chan struct{})
	go func() {
		tk := time.NewTicker(2 * time.Second)
		defer tk.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tk.C:
				dropEbitenDeferred()
			}
		}
	}()
	code := m.Run()
	close(stop)
	os.Exit(code)
}

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

// snapshotLang puts the three language globals back when the test ends. Any test that sets uiLang,
// gameLang or editorLang, or draws with g.Draw (pickLanguage copies gameLang into uiLang), must call it:
// a test once left uiLang = "en" and the next ones ran in English. TestTestsRestoreLanguage checks it.
func snapshotLang(t *testing.T) {
	t.Helper()
	ui, gl, el := uiLang, gameLang, editorLang
	t.Cleanup(func() { uiLang, gameLang, editorLang = ui, gl, el })
}
