package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: dither-cli\n\nAn interactive, dithered 3D tree. Requires a macOS or Linux terminal.\n\n  ← / J   rotate left\n  → / L   rotate right\n  Q / Ctrl-C   quit")
	}
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "dither-cli:", err)
		os.Exit(1)
	}
}

// stty keeps terminal handling dependency-free on both macOS and Linux.
func stty(args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func terminalSize() (int, int, error) {
	value, err := stty("size")
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(value)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("cannot read terminal dimensions")
	}
	rows, rowErr := strconv.Atoi(fields[0])
	cols, colErr := strconv.Atoi(fields[1])
	if rowErr != nil || colErr != nil || rows < 1 || cols < 1 {
		return 0, 0, fmt.Errorf("invalid terminal dimensions")
	}
	return cols, rows, nil
}

func run() error {
	state, err := stty("-g")
	if err != nil {
		return fmt.Errorf("open an interactive terminal to run the tree (%w)", err)
	}
	width, height, err := terminalSize()
	if err != nil {
		return err
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
	if _, err := stty("raw", "-echo"); err != nil {
		return err
	}
	defer stty(state)
	defer fmt.Fprint(os.Stdout, "\x1b[0m\x1b[?25h\x1b[?1049l")
	if _, err := fmt.Fprint(os.Stdout, "\x1b[?1049h\x1b[?25l\x1b[2J"); err != nil {
		return err
	}

	keys := make(chan byte, 64)
	go func() {
		defer close(keys)
		buffer := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(buffer)
			for _, key := range buffer[:n] {
				keys <- key
			}
			if err != nil {
				return
			}
		}
	}()

	frames := time.NewTicker(time.Second / 30)
	defer frames.Stop()
	resize := time.NewTicker(500 * time.Millisecond)
	defer resize.Stop()
	position, angle := 0.5, 0.0
	tree := newTree()
	dirty := true
	var decoder keyDecoder
	for {
		if dirty {
			if _, err := io.WriteString(os.Stdout, tree.frame(width, height, position, angle)); err != nil {
				return err
			}
			dirty = false
		}
		select {
		case <-signals:
			return nil
		case key, ok := <-keys:
			if !ok {
				return nil
			}
			action := decoder.feed(key)
			if action == quit {
				return nil
			}
			if action != 0 {
				// Snap to a terminal column before moving to avoid half-cell rounding drift.
				span := max(1, width-1)
				column := int(math.Round(position * float64(span)))
				next := float64(max(0, min(span, column+action))) / float64(span)
				if next != position {
					position, dirty = next, true
				}
			}
		case <-resize.C:
			if w, h, err := terminalSize(); err == nil && (w != width || h != height) {
				width, height, dirty = w, h, true
			}
		case <-frames.C:
			target := (position - 0.5) * 2 * math.Pi
			if math.Abs(target-angle) > 0.0005 {
				angle += (target - angle) * 0.24
				dirty = true
			}
		}
	}
}

const quit = 2

type keyDecoder struct{ escape byte }

func (d *keyDecoder) feed(key byte) int {
	if key == 3 || key == 4 || key == 'q' || key == 'Q' {
		return quit
	}
	if key == 27 {
		d.escape = 1
		return 0
	}
	if d.escape == 1 {
		d.escape = 0
		if key == '[' || key == 'O' {
			d.escape = 2
			return 0
		}
	}
	if d.escape == 2 {
		// Consume parameters too, so modified arrows work across read boundaries.
		if key >= 0x40 && key <= 0x7e {
			d.escape = 0
			if key == 'D' {
				return -1
			}
			if key == 'C' {
				return 1
			}
		}
		return 0
	}
	switch key {
	case 'j', 'J':
		return -1
	case 'l', 'L':
		return 1
	}
	return 0
}
