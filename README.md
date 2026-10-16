# Dither wilds

A full-screen, dithered forest you can walk through from an elevated top-down
view. A tiny traveler explores an endless, procedurally generated landscape:
winding rivers, tall-grass clearings, deep woods, and loose canopies that let
patches of sunlight reach the ground.

```sh
go run .
```

Walk with the **arrow keys**, **WASD**, or **IJKL** (I up, J left, K down,
L right). Hold a key to keep walking. Press **Q** or **Ctrl-C** to quit.
The camera follows the traveler and the view adapts when the terminal resizes.

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

Requires Go 1.22+, macOS or Linux with `stty`, and a UTF-8 terminal with
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

Benchmark animated rendering at three terminal sizes:

```sh
go test -run '^$' -bench BenchmarkForestFrame -benchmem
```
