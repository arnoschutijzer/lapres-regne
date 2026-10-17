package terminal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"lapresregne/internal/soundtrack"
	"lapresregne/internal/world"
)

// Session owns the interactive terminal: raw mode, signals, keys and frames.
type Session struct {
	state           string
	width, height   int
	signals, resize chan os.Signal
}

// Open checks for an interactive terminal and starts listening for signals.
func Open() (*Session, error) {
	state, err := stty("-g")
	if err != nil {
		return nil, fmt.Errorf("open an interactive terminal to explore the forest (%w)", err)
	}
	width, height, err := terminalSize()
	if err != nil {
		return nil, err
	}
	s := &Session{state: state, width: width, height: height, signals: make(chan os.Signal, 1), resize: make(chan os.Signal, 1)}
	signal.Notify(s.signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	signal.Notify(s.resize, syscall.SIGWINCH)
	return s, nil
}

func (s *Session) Close() {
	signal.Stop(s.resize)
	signal.Stop(s.signals)
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

// Run lets the traveler explore the world until they quit. Music is nil when muted.
func (s *Session) Run(world *world.World, music soundtrack.Player, title string) error {
	if _, err := stty("raw", "-echo"); err != nil {
		return err
	}
	defer stty(s.state)
	defer fmt.Fprint(os.Stdout, mouseStop+keyboardStop+"\x1b[0m\x1b[?25h\x1b[?1049l")
	if _, err := fmt.Fprint(os.Stdout, "\x1b[?1049h\x1b[?25l\x1b[2J"+keyboardStart+mouseStart); err != nil {
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
	started := time.Now()
	view := newRenderer(world, title)
	width, height := s.width, s.height
	dirty := true
	var decoder keyDecoder
	held := make(heldKeys)
	// The last pointer cell; the hero keeps aiming there while the camera follows.
	var cursorColumn, cursorRow int
	handleKey := func(key byte) bool {
		event := decoder.feed(key)
		held.update(event)
		if event.mouse {
			cursorColumn, cursorRow = event.column, event.row
			return false
		}
		if event.zoom != 0 {
			world.ZoomBy(event.zoom)
			return false
		}
		if event.turn != 0 {
			world.TurnBy(event.turn)
			return false
		}
		if event.panRight != 0 || event.panDown != 0 {
			distance := view.zoom.panDistance()
			world.PanBy(float64(event.panRight)*distance, float64(event.panDown)*distance)
			return false
		}
		if event.click {
			if slot, ok := view.slotAt(event.column, event.row); ok {
				view.cursor = slotCursor{active: true, slot: slot}
				return false
			}
			// Clicking a tree walks there to chop it, clicking the ground walks there, and clicking
			// anything else stops.
			view.menu = nil
			if tree, ok := view.treeAt(event.column, event.row); ok {
				world.Approach(tree)
			} else if x, z, ok := view.groundAt(event.column, event.row); ok {
				world.WalkTo(x, z)
			} else {
				world.Stop()
			}
			return false
		}
		if event.focusOut {
			cursorColumn, cursorRow = 0, 0
		}
		if event.release || event.focusOut {
			return false
		}
		if event.action == quit {
			return true
		}
		if event.action == nextTrack || event.action == previousTrack {
			if music != nil && !event.repeat {
				direction := 1
				if event.action == previousTrack {
					direction = -1
				}
				music.Skip(direction)
			}
			return false
		}
		if event.action == interact {
			// E opens the action menu for the tree or doorway in reach, or closes an open menu.
			if !event.repeat {
				view.toggleMenu()
			}
			return false
		}
		if event.action == dropLog {
			// Holding X keeps dropping, one log per repeat.
			view.cursor.drop(world)
			return false
		}
		if event.action == previousSlot || event.action == nextSlot {
			// Choosing a slot shows the inventory, so the highlight is never hidden.
			view.inventoryOpen = true
			if event.action == previousSlot {
				view.cursor.move(-1)
			} else {
				view.cursor.move(1)
			}
			return false
		}
		if event.action == toggleInventory {
			if !event.repeat {
				view.inventoryOpen = !view.inventoryOpen
			}
			return false
		}
		if event.action == firstChoice {
			if !event.repeat {
				view.chooseFirst()
			}
			return false
		}
		if event.action >= left && event.action <= down {
			view.menu = nil
		}
		if !event.enhanced {
			dx, dz := event.action.vector()
			world.Walk(dx, dz, event.shift)
		}
		return false
	}
	draw := func() error {
		output := view.frame(width, height, time.Since(started).Seconds())
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
		case <-s.signals:
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
		case <-s.resize:
			if w, h, err := terminalSize(); err == nil && (w != width || h != height) {
				width, height, dirty = w, h, true
			}
		case <-frames.C:
			if music != nil {
				if err := music.Err(); err != nil {
					return fmt.Errorf("soundtrack (use --mute for silent play): %w", err)
				}
			}
			now := time.Now()
			dt := min(now.Sub(lastFrame), 100*time.Millisecond)
			lastFrame = now
			dx, dz := held.direction()
			world.Advance(dx, dz, dt, held.sprinting())
			world.Follow(dt)
			view.zoom.step(world.Zoom(), dt)
			view.turn.step(world.Yaw(), dt)
			if x, z, ok := view.groundAt(cursorColumn, cursorRow); ok {
				world.AimAt(x, z)
			}
			dirty = true
			if dirty {
				if err := draw(); err != nil {
					return err
				}
			}
		}
	}
}
