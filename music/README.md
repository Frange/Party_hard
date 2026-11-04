# music/ — main song of each party

Drop **MP3** files here (only `.mp3` is read; other formats are ignored).

The file name must **start with the party number**, followed by `_`, `-` or a space and anything else:

```
1_house_party.mp3
2 - rooftop.mp3
7_cruise.mp3
```

- `1` is the first party in the campaign list, `2` the second, and so on (the number shown in the level menu).
- If two files have the same number, the first one in alphabetical order is used.
- A party without a file uses the built-in synthesised music.

The songs are not included in the repository (they may be copyrighted); add your own.

## Which style does each party have?

Every party has its own genre. When there is no mp3 for a party the game plays its own synthesised music in that style
(a short title at the start of the party tells you which). Use this table to choose which song goes where:

| # | Party | Kind of party | Genre |
|---|-------|---------------|-------|
| 1 | House party | Living-room kegger | House |
| 2 | Miami Beach | Lawless beach shack | Reggaeton |
| 3 | Glow Neon | Neon rave | Synthwave |
| 4 | Diana (ranch) | Bonfire in boots | Country |
| 5 | Bullring | Cape and capers bash | Pasodoble / flamenco |
| 6 | Eiffel Tower | Black-tie waltz dinner | Classical (waltz) |
| 7 | Night cruise | The captain's bash | Disco |
| 8 | Funfair | Fairground, no seatbelt | Chiptune |
| 9 | Stadium | Cup final blowout | Pop |
| 10 | Disco club | Sunrise session | Techno |
| 11 | Festival | Mud festival | Electronic |
| 12 | Skate park | Bowl jam | Punk |
| 13 | Cemetery | Wake with riffs | Metal |
| 14 | Cinema | Late-night screening | Orchestral |
| 15 | Circuit | Brakeless race | Electronic |
| 16 | Ski resort | Après-ski with cocoa | Pop |
| 17 | Rooftop | Rooftop pool bash | Hip-hop (lo-fi) |
| 18 | Casino | Black-tie night | Jazz (swing) |
| 19 | Motorway | Traffic jam with speakers | Rock |
| 20 | Train station | Ticketless party | Rap (boom bap) |
| 21 | Space station | Zero-gravity rave | Electronic |
| – | Isla Nublar (secret) | Last Jurassic party | Epic orchestral (wonder and danger) |

The level select shows "kind of party · genre" under the name of every party, and the title card of the party shows the
kind and the music (`Música: ...`) for a few seconds.

## Name your files after the style (so they cannot clash)

If the **last word** of a file name is a genre (`13_cementerio_metal.mp3`, `7_cruise_disco.mp3`, `extra_03_house_2.mp3`;
a trailing version number is ignored) the game checks it against the genre of the party:

- the same genre: the file plays and the title card says so;
- another genre (old batches such as `13_cine_electro.mp3` playing in the Cemetery): the file is **ignored** with a
  line in the console and the music of the game plays instead;
- no genre in the name (`5_fiesta.mp3`): it always plays, and the card says nothing about the style.

Genre words: house, electro/electronica, techno, disco, reggaeton, punk, pop, metal, rap, hiphop/hip-hop/lofi,
clasica/classical, jazz, country, flamenco/pasodoble, rock, synthwave, orquestal/orchestral, epico/epic, chiptune.

`more/*.mp3` (the songs that follow the main one) are filtered the same way: only the files of the style of the party
play; if none fits, the synthesised music of the party's genre does. The files in `DJ/` and `bands/` are not filtered.
`./PartyHard -exportmusic music -musicbatch 3` writes files already named with their style.

The random-mode venues use Rock (pub), Pasodoble (European square) and Pop (mall). When the DJ dies the punk takes over;
a band of its own style (Metallica = metal, The Beatles = pop, Elvis = country, Lost Prophets and Despistaos = rock) plays
its genre, and the punk bands play the style of the party.
