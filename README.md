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
Lines starting with `//` are comments.

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

Walls, fences, gates and graves block the way, a bridge's rails keep the hero
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
