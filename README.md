# L’Après-Règne

A full-screen, dithered forest you can walk through from an elevated top-down
view. A tiny traveler explores an endless, procedurally generated landscape:
winding rivers, tall-grass clearings, deep woods, and loose canopies that let
patches of sunlight reach the ground.

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

The original 3D tree and its underscore camera slider are available with
`go run . --tree`. Use **← / →** or **J / L** to orbit it.

Requires Go 1.25+, macOS or Linux with `stty`, and a UTF-8 terminal with
true color and a font that supports Unicode braille. Use at least 40 columns
and 14 rows; a larger window gives the dithering more detail.

To build a standalone executable:

```sh
go build -o dither-cli .
./dither-cli
```

Run the application-level terminal checks with Python 3:

```sh
python3 tests/terminal.py
```

Run the soundtrack and shuffle checks with `go test ./...`.

Benchmark animated rendering at three terminal sizes:

```sh
go test -run '^$' -bench BenchmarkForestFrame -benchmem
```
