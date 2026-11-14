package main

// ---------------------------------------------------------------------------
// THE VERSION OF THE GAME: the only place where it is written.
//
// Semantic versioning (CHANGELOG.md explains it): MAJOR.MINOR.PATCH
//   - small change (fixes, polish)            -> x.y.Z   (1.0.0 -> 1.0.1)
//   - medium change (new features, new party) -> x.Y.0   (1.0.1 -> 1.1.0)
//   - big change (a new mode, a new campaign)  -> X.0.0   (1.4.2 -> 2.0.0)
//
// Every batch of work with something new raises it and adds its entry at the top of CHANGELOG.md BEFORE the
// build (version_test.go checks that both agree; build.sh reads it from here for VERSION.txt and Info.plist).
// It is shown in the bottom right corner of the main menu, in Settings > Credits and in the window title.
// ---------------------------------------------------------------------------

const gameVersion = "1.4.0"

// versionLabel: how the version is written on screen ("v.1.0.0", the owner's style).
func versionLabel() string { return "v." + gameVersion }

// gameWindowTitle / editorWindowTitle: the window titles of the game and of the editor.
func gameWindowTitle() string { return "Party Hardy - Fiesta Nocturna " + versionLabel() }

func editorWindowTitle() string {
	return "Party Hardy - Editor de sprites y escenarios " + versionLabel()
}
