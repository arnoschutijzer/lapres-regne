# Dither tree

A full-screen Go CLI with a procedurally generated 3D tree, shaded branches,
ordered dithering, and a movable underscore that controls the camera angle.

```sh
go run .
```

Move the underscore with **← / →** or **J / L**. Hold a key to keep rotating.
Press **Q** or **Ctrl-C** to quit. The view adapts when the terminal resizes.

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
