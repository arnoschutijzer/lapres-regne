# L’Après-Règne

```sh
go run .
```

Put your songs in `./music`, or choose a folder:

```sh
go run . --music-dir /path/to/songs
```

The game shuffles MP3, OGG/Vorbis, WAV, and FLAC files directly inside that
folder, adapting sample rates and mono/stereo audio automatically. Missing or
empty folders disable the soundtrack. Each shuffle plays all available songs
and avoids immediate repeats between cycles when there is more than one song.
Playback is built in; no `ffplay` or separate music player is needed. Use
`--mute` to disable the soundtrack. Music stops when you quit.

Press **N** for the next track or **P** for the previous track. Previous follows
playback history; Next retraces forward history before drawing another shuffled
song. Previous restarts the first track when there is no earlier history.
Every transition fades the current song out over three seconds, pauses in silence
for a quarter second, then fades the next song in over three seconds. Songs never
overlap. Short clips use shorter fades. N/P use the same transition; requests
during a fade-out select the song that starts after silence, keeping the latest
request.

Walk with the **arrow keys**, **WASD**, or **IJKL** (I up, J left, K down,
L right). Hold a key to keep walking. Press **Q** or **Ctrl-C** to quit.
Hold **Shift** to run at **2× speed**; release Shift to return to walking
while keeping your movement keys held.
The animation loop targets **60 FPS**. The camera follows the traveler and the
view adapts when the terminal resizes.

In terminals supporting the [Kitty keyboard protocol](https://sw.kovidgoyal.net/kitty/keyboard-protocol/),
including Ghostty and Kitty, movement uses independent held keys: hold LEFT + UP
to walk diagonally, then release LEFT to continue UP. Diagonals have the same
speed as straight movement. Losing window focus stops movement.
The app enables key release reporting automatically and restores it on exit.
Older terminals fall back to key-repeat movement because they cannot report
which keys were released.

Trees cast shadows and you can walk freely through their trunks. Tall grass
sways in front of the traveler, while a small canopy cutaway keeps the character
readable under the trees. Rivers ripple and can be crossed by wading at half speed.
The forest rolls in gentle hills that level out into the river valleys, where
the rivers run in gullies below steep banks; nearer hills hide what lies behind
them, and slopes catch or turn from the sun.

Terrain is generated from world coordinates with bounded memory, so revisiting
a place recreates the same forest. Choose another forest with a seed:

```sh
go run . --seed 42
```

Explore a hand-made map instead of the endless forest:

```sh
go run . --map maps/lumbridge.map
```

`maps/lumbridge.map` holds Old School RuneScape's Lumbridge, traced tile for
tile from the OSRS Wiki's world map (x 3150 to 3283, y 3184 to 3291): the forest
west of the castle, the castle and its courtyard, the village, Bob's axes, the
church and graveyard, the River Lum and its bridges, Groats' farm, the goblins'
yard round Lozar's house, the swamp's northern edge, and the Al Kharid toll
gate. You start where every adventurer does, in the castle courtyard.

A map file is plain text: optional settings, then a `terrain` grid with one
character per tile (two world units wide), rows running north to south.
Lines starting with `//` are comments. A `start <col> <row>` setting starts
the hero on a tile of any ground, in place of `@`.

```text
name The Pond

terrain
.T..~
.@O~~
WY...
```

| Tile | Meaning | Tile | Meaning |
| ---- | ------- | ---- | ------- |
| `.`  | grass   | `~`  | water (wade at half speed) |
| `T`  | tree    | `O`  | oak     |
| `W`  | willow  | `Y`  | yew     |
| `:`  | earth path | `_`  | stone floor |
| `#`  | wall    | `=`  | fence   |
| `G`  | gate    | `+`  | grave   |
| `D`  | open door | `b`  | wooden floor |
| `B`  | battlements | `H`  | bridge  |
| `@`  | where the hero starts | | |

Name parts of the map with `area <letter> <name>` settings and an `areas`
grid laid over the terrain, using `.` for none; the status line names the area
the hero stands in.

```text
area C Lumbridge Castle

areas
CCC..
```

Raise the ground with a `heights` grid laid over the terrain: each digit lifts
its tile to a level from 0 to 9, three quarters of a world unit apart. The
ground slopes smoothly from tile to tile, and tiles the grid leaves out lie at
level 0. Water and bridge decks lie flat on their own level: put a river below
its banks and they fall to meet it at its edge, while a bridge on the banks'
level spans the gully. Keep a building's tiles on one level so its walls and
roof sit square.

```text
heights
00122
```

Stand walls along the edges between tiles, as Old School RuneScape does, with a
`walls` grid at twice the terrain's resolution: even lines run along the north
edges of a row of tiles, odd lines along their west edges with each tile's
middle between. Along an edge, `#` is a wall, `B` battlements, `=` a fence and
`D` a doorway, or an open gate in a fence. A tile's middle may hold `/` or `\`
for a wall running diagonally across it, battlements or a fence when both
corners it joins are marked `B` or `=`; corners are otherwise free to mark.

```text
walls
+#D#+
#   #
+###+
```

A room closed in by walls and doors is roofed while the hero is outside it.
Lay each building's roof with a `roofs` grid over the terrain: `s` for slate,
`t` for thatch or `c` for red clay tiles; a building takes whichever most of its
rooms lie under, and tiles the grid leaves out, or marks `.`, lie under slate.

```text
roofs
.ccc.
```

Stand furniture and landmarks on the map with an `objects` grid laid over the
terrain, using `.` for none. A fixture covering several tiles repeats its
letter on each, and tiles of the same kind side by side join into one, such as
a long table.

```text
objects
..tt.
```

| Object | Meaning | Object | Meaning |
| ------ | ------- | ------ | ------- |
| `t`    | table   | `c`    | chair   |
| `T`    | throne  | `a`    | suit of armour |
| `r`    | cooking range | `o` | barrel |
| `x`    | crate   | `q`    | sacks   |
| `s`    | shelves | `S`    | staircase |
| `L`    | ladder  | `i`    | standing torch |
| `l`    | candles | `f`    | standard |
| `A`    | altar   | `p`    | pew     |
| `O`    | organ   | `k`    | coffin  |
| `F`    | fountain | `u`   | statue  |
| `w`    | well    | `h`    | haystack |
| `n`    | signpost | `C`   | cart    |
| `g`    | trough  | `y`    | scarecrow |
| `v`    | crops   | `N`    | anvil   |
| `K`    | counter | `b`    | bed     |

A fountain, well or haystack filling a square of four tiles stands as one
where they meet.

Chairs and thrones turn their backs on the table they stand at; other
fixtures back onto a wall beside them, and a row of pews faces along the
church.

Walls and fences stepping diagonally from tile to tile join across the corners
between them. Walls, fences, gates, graves and fixtures other than crops block the way, a bridge's rails keep the hero
out of the water beside it, and nothing beyond the map's edges can be walked on. Clicking the ground or a tree walks round whatever
blocks the way, or as close as the hero can get. Progress keeps the map's name,
so your position and stumps return only to the map they were saved on.

Requires Go 1.25+, macOS or Linux with `stty`, and a UTF-8 terminal with
true color and a font that supports Unicode braille and quadrant blocks. Use at
least 40 columns and 14 rows; a larger window gives the dithering more detail.
The scenery layers heavy dithering over subtle solid colors. The traveler and
axe use solid colors at their natural world scale to keep their details readable.

To build a standalone executable:

```sh
go build -o lapresregne .
./lapresregne
```

Run all tests, including the application-level terminal checks, with Go:

```sh
go test ./...
```

Benchmark animated rendering at three terminal sizes:

```sh
go test -run '^$' -bench BenchmarkForestFrame -benchmem ./internal/adapters/terminal
```
