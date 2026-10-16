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
	treeMode := flag.Bool("tree", false, "show the original orbiting tree")
	seed := flag.Uint64("seed", 7, "procedural forest seed")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: dither-cli [--seed number] [--tree]\n\nWalk through an endless dithered forest.\n\n  Arrows / WASD / IJKL   walk (I up, J left, K down, L right)\n  Q / Ctrl-C            quit\n\nWith --tree, left/right or J/L orbit the original tree.")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*treeMode, *seed); err != nil {
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

func run(treeMode bool, seed uint64) error {
	state, err := stty("-g")
	if err != nil {
		return fmt.Errorf("open an interactive terminal to explore the forest (%w)", err)
	}
	width, height, err := terminalSize()
	if err != nil {
		return err
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
	resize := make(chan os.Signal, 1)
	signal.Notify(resize, syscall.SIGWINCH)
	defer signal.Stop(resize)
	if _, err := stty("raw", "-echo"); err != nil {
		return err
	}
	defer stty(state)
	defer fmt.Fprint(os.Stdout, keyboardStop+"\x1b[0m\x1b[?25h\x1b[?1049l")
	if _, err := fmt.Fprint(os.Stdout, "\x1b[?1049h\x1b[?25l\x1b[2J"+keyboardStart); err != nil {
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

	frames := time.NewTicker(time.Second / 60)
	defer frames.Stop()
	position, angle := 0.5, 0.0
	var original *tree
	if treeMode {
		original = newTree()
	}
	forest := newForest(seed)
	dirty := true
	var decoder keyDecoder
	held := make(heldKeys)
	handleKey := func(key byte) bool {
		event := decoder.feed(key)
		held.update(event)
		if event.release || event.focusOut {
			return false
		}
		if event.action == quit {
			return true
		}
		if !treeMode {
			if !event.enhanced {
				forest.walk(event.action)
			}
		} else if event.action == left || event.action == right {
			// Snap to a terminal column before moving to avoid half-cell rounding drift.
			span := max(1, width-1)
			column := int(position*float64(span) + 0.5)
			delta := 1
			if event.action == left {
				delta = -1
			}
			next := float64(max(0, min(span, column+delta))) / float64(span)
			if next != position {
				position, dirty = next, true
			}
		}
		return false
	}
	draw := func() error {
		var output string
		if treeMode {
			output = original.frame(width, height, position, angle)
		} else {
			output = forest.frame(width, height, time.Since(forest.started).Seconds())
		}
		_, err := io.WriteString(os.Stdout, output)
		dirty = false
		return err
	}
	if err := draw(); err != nil {
		return err
	}
	lastFrame := time.Now()
	for {
		select {
		case <-signals:
			return nil
		case key, ok := <-keys:
			if !ok {
				return nil
			}
			if handleKey(key) {
				return nil
			}
			// Consume pending repeats before rendering, so input cannot queue behind redraws.
			for len(keys) > 0 {
				if handleKey(<-keys) {
					return nil
				}
			}
		case <-resize:
			if w, h, err := terminalSize(); err == nil && (w != width || h != height) {
				width, height, dirty = w, h, true
			}
		case <-frames.C:
			now := time.Now()
			dt := min(now.Sub(lastFrame), 100*time.Millisecond)
			lastFrame = now
			if !treeMode {
				dx, dz := held.direction()
				forest.advance(dx, dz, dt)
				forest.follow(dt)
				dirty = true
			} else {
				target := (position - 0.5) * 2 * math.Pi
				if math.Abs(target-angle) > 0.0005 {
					angle += (target - angle) * (1 - math.Exp(-8.2*dt.Seconds()))
					dirty = true
				}
			}
			if dirty {
				if err := draw(); err != nil {
					return err
				}
			}
		}
	}
}
