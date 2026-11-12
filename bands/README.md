# bands/ — songs played by the bands

Some parties have a band playing on a stage. If you drop **MP3** files here whose name **starts with the band name**,
that band plays your songs while it is on stage (only `.mp3` is read). Without a file the built-in music is used.

Letters are compared ignoring case, spaces and symbols, so all of these work for Sum 42:
`sum42_song.mp3`, `Sum 42 - Song.mp3`, `SUM42-Song.mp3`. A leading "The" may be omitted (`offsprung_x.mp3`).
Several files per band are fine: one is picked at random each time.

Names the game knows:

| Band | File name starts with |
|------|-----------------------|
| The Beetles | `thebeetles` or `beetles` |
| Sum 42 | `sum42` |
| Blink-183 | `blink183` |
| Simple Pan | `simplepan` |
| The Offsprung | `theoffsprung` or `offsprung` |
| Metalica | `metalica` |
| Punk Sailor | `punksailor` |
| Billy Tallent | `billytallent` |
| Lost Profets | `lostprofets` |
| Despistaus | `despistaus` |
| Elviz | `elviz` |

Examples: `beetles_song.mp3`, `Metalica - Song.mp3`, `blink-183 - Song.mp3`.

The songs are not included in the repository (they may be copyrighted); add your own.
