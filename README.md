# Party Hard (homage clone) with a built-in sprite and level editor

A fan-made homage to *Party Hard* (Pinokl Games / tinyBuild), written in Go with [Ebiten](https://ebitengine.org/).
All art is procedural pixel art; everything else (characters, objects, animals, levels) can be replaced from the `assets/` folder.
Made by **Frange**, with Claude. Credits and the full homage notice are in the in-game *Settings > Credits* screen.

- Windows: `PartyHard.exe` (game) and `PartyHardEditor.exe` (editor).
- macOS: `PartyHard.app` and `PartyHardEditor.app` (first time: right-click > Open).
- The `assets` folder must sit next to them (next to the apps on macOS). If you delete it, it is recreated with the defaults.
- Inside the game, **F2** opens the editor; "Back to game (F2)" in the editor returns.

## The game

You are a killer at a party. Kill everybody (guests, guards, DJ and band) without getting caught.

- **Move** WASD / arrows, **run** Shift, **stab** Space, **dance** F, **use** E (carry bodies, hide, traps, drinks, phones, items). All keys and pad buttons can be rebound in Settings; the mouse works in every menu.
- Everybody only sees in front of them: sneak up from behind. A stab in the back of the special killer cannot be answered at once.
- Witnesses phone the police from the **red booths** (at least two per party). Cutting a booth's cable (once each) stops that phone; a guest who sees it cut goes to the **yellow technician phone** and a yellow van brings technicians to repair the line.
- Police escalate with your criminality stars: police > FBI > military > SWAT > Power Rangers. They bag bodies, an ambulance comes for many bodies, and if you are hunted you will see handcuffs over your head. Places without a road (cruise ship, space station) get boats or ships in each vehicle's colour.
- **Items**: poison bottles (to poison drink bowls, punch and buffets), disguise bags (an automatic outfit change that makes the police lose you), chairs, and more traps (explosive, electric, gas leak, fireworks, falling piano). Anything you can use is marked with a bobbing diamond and its name when you get close.
- **Points** for kills, animals, explosions and traps, hidden bodies, cut phone lines, killing cops with traps, and killing the special killer.
- **Trophies**: 50 trophies, each giving something (clothes, characters, events, powers). The last one, *Leyenda de la fiesta*, needs all the others and gives powers (immunity with golden halo and cop killing, summon famous guests, send a killer, rare events), turns the main menu golden, and unlocks golden items.
- **Campaign**: 21 parties, from the house party to the space station (always last), including a six-lane motorway, a railway station, a casino, a ski resort and a graveyard. A free random mode and generated parties are also available.
- **Rare events**: UFO abductions, Superman, Satan and God, zombies, T-Rex, ghosts, F1 drivers tearing through the party, escaped herds, skydivers, ball games, Halloween / Christmas / summer groups, taxis, monster trucks, Rocket League cars, trains, cheerleaders, snakes... Some trophies let you trigger rare events on demand.
- Famous guests have catchphrases, chat with each other in short scenes and do their own special moves.
- Hidden **DEBUG mode** (Konami code in the main menu): toggles for immunity, cop killing, powers, unlocking everything and more; nothing is saved while it is on.

## Editor

- **Characters**: paint the figure (front, side, back) and mark head, torso, arms and legs. Animations are generated automatically; any frame can be repainted.
- **Objects / animals**: animated frames, anchor, collision footprint and properties (solid, hide, phone, poisonable, explosive or electric trap, light, use points, crosses the street...).
- **Levels**: paint the background, draw the logic (walls, fences, water, walking and dance zones), place objects, player start, guard routes, DJ and musicians, lights and police arrival. "Test" button.
- Shortcuts: P pencil, E eraser, L line, R rectangle, O ellipse, F fill, I picker, M symmetry, Ctrl+Z/Ctrl+Y, Ctrl+S, wheel = zoom.
- Drag PNG files onto the editor window to import your own art.
- The content format is documented in [`ASSETS.md`](ASSETS.md).

## Building

Requires Go. `./build.sh` builds the game and the editor for Windows and macOS into `dist/` and zips them (the editor is the same code built with `-tags editor`).

Debug flags: `-level N`, `-event N`, `-mute`, `-check`, `-selftest N`, `-shot f.png`, `-noassets`, `-editor`, `-immune`, `-fastevents`, `-with slug,slug`.

Tests: `go test .`
