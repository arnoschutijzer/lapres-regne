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

// frameRate is how many frames a second the session aims to draw.
const frameRate = 120

// Run lets the traveler explore the world, drawn in one of the looks, until they quit. Music is
// nil when muted.
func (s *Session) Run(world *world.World, camera *Camera, looks *Looks, music soundtrack.Player, title string) error {
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

	// The world advances once a frame by the time since the last, as Ebitengine's SyncWithFPS mode
	// updates a game once per frame drawn: a frame drawn late drops the ticks it missed rather than
	// queueing them, and the world still moves at the same pace.
	frames := time.NewTicker(time.Second / frameRate)
	defer frames.Stop()
	started := time.Now()
	view := newRenderer(world, camera, title)
	view.style = looks.style()
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
			// The wheel scrolls the bank under the pointer, and zooms anywhere else.
			if !view.scrollBank(event.column, event.row, -event.zoom) {
				camera.zoomBy(event.zoom)
			}
			return false
		}
		if event.turn != 0 {
			camera.turnBy(event.turn)
			return false
		}
		if event.panRight != 0 || event.panDown != 0 {
			distance := view.zoom.panDistance()
			camera.panBy(float64(event.panRight)*distance, float64(event.panDown)*distance)
			return false
		}
		if event.rightClick {
			view.rightClick(event.column, event.row)
			return false
		}
		if event.click {
			view.menu = nil
			if slot, ok := view.slotAt(event.column, event.row); ok {
				view.clickSlot(slot)
				return false
			}
			if stack, ok := view.stackAt(event.column, event.row); ok {
				view.clickStack(stack)
				return false
			}
			// Clicking a tree walks there to chop it, clicking a door or gate walks there to open or
			// close it, clicking a staircase or ladder walks there to climb it, clicking the ground
			// walks there, and clicking anything else stops.
			if tree, ok := view.treeAt(event.column, event.row); ok {
				world.Approach(tree)
			} else if doorway, ok := view.doorwayAt(event.column, event.row); ok {
				camera.stopPanning()
				world.ApproachDoor(doorway)
			} else if flight, ok := view.flightAt(event.column, event.row); ok {
				camera.stopPanning()
				world.ApproachFlight(flight)
			} else if x, z, ok := view.groundAt(event.column, event.row); ok {
				camera.stopPanning()
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
			// E opens the action menu for what is in reach, or closes an open menu.
			if !event.repeat {
				view.toggleMenu()
			}
			return false
		}
		if event.action == dropLog {
			// Holding X keeps dropping, or depositing while the bank is open, one log per repeat.
			view.removeFromSlot()
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
		if event.action == previousStack || event.action == nextStack {
			if event.action == previousStack {
				view.moveBankCursor(-1)
			} else {
				view.moveBankCursor(1)
			}
			return false
		}
		if event.action == withdrawStack {
			// Holding Shift-X keeps withdrawing, one item per repeat.
			view.withdraw()
			return false
		}
		if event.action == toggleBank {
			if !event.repeat {
				view.toggleBank()
			}
			return false
		}
		if event.action == nextStyle {
			if !event.repeat {
				looks.next()
				view.style = looks.style()
			}
			return false
		}
		if event.action == toggleInventory {
			if !event.repeat {
				view.inventoryOpen = !view.inventoryOpen
			}
			return false
		}
		if event.action == firstChoice || event.action == secondChoice {
			if !event.repeat {
				view.choose(int(event.action - firstChoice))
			}
			return false
		}
		if event.action >= left && event.action <= down {
			view.menu = nil
		}
		if !event.enhanced {
			dx, dz := camera.walk(event.action.vector())
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
			dx, dz := camera.walk(held.direction())
			world.Advance(dx, dz, dt, held.sprinting())
			// Arriving at a staircase leading both up and down asks which way to climb.
			if world.OfferedClimb() && view.menu == nil {
				view.openMenu()
			}
			view.follow(dt)
			view.zoom.step(camera.Zoom(), dt)
			view.turn.step(camera.Yaw(), dt)
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
