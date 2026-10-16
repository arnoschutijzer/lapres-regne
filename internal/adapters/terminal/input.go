package terminal

import (
	"strconv"
	"strings"
)

type action byte

const (
	none action = iota
	left
	right
	up
	down
	quit
	nextTrack
	previousTrack
	interact
	firstChoice
	toggleInventory
	dropLog
	previousSlot
	nextSlot
)

type keyEvent struct {
	key      int
	action   action
	release  bool
	enhanced bool
	focusOut bool
	shift    bool
	repeat   bool
	mouse    bool
	click    bool
	// zoom is 1 for the wheel scrolling up and -1 for scrolling down.
	zoom int
	// pan is -1 for the wheel scrolling left and 1 for scrolling right.
	pan    int
	column int
	row    int
}

const (
	keyLeft = 0x110000 + iota
	keyRight
	keyUp
	keyDown
	keyShift
)

// Kitty's keyboard protocol reports independent key press/repeat/release events.
// Flags 1 + 2 + 8 request disambiguation, event types, and encoded letter keys.
const keyboardStart = "\x1b[>11u\x1b[?u\x1b[?1004h"
const keyboardStop = "\x1b[?1004l\x1b[<u"

// Any-motion tracking with SGR encoding reports the pointer cell even with no button held.
const mouseStart = "\x1b[?1003h\x1b[?1006h"
const mouseStop = "\x1b[?1006l\x1b[?1003l"

type keyDecoder struct {
	escape   byte
	params   []byte
	enhanced bool
}

func keyAction(key int) action {
	switch key {
	case keyLeft, 'j', 'J', 'a', 'A':
		return left
	case keyRight, 'l', 'L', 'd', 'D':
		return right
	case keyUp, 'i', 'I', 'w', 'W':
		return up
	case keyDown, 'k', 'K', 's', 'S':
		return down
	case 3, 4, 'q', 'Q':
		return quit
	case 'n', 'N':
		return nextTrack
	case 'p', 'P':
		return previousTrack
	case 'e', 'E':
		return interact
	case '1':
		return firstChoice
	case '\t':
		return toggleInventory
	case 'x', 'X':
		return dropLog
	case '[':
		return previousSlot
	case ']':
		return nextSlot
	}
	return none
}

func (d *keyDecoder) feed(key byte) keyEvent {
	if key == 3 || key == 4 {
		d.escape = 0
		return keyEvent{key: int(key), action: quit}
	}
	if key == 27 {
		d.escape = 1
		d.params = d.params[:0]
		return keyEvent{}
	}
	if d.escape == 1 {
		d.escape = 0
		if key == '[' || key == 'O' {
			d.escape = 2
			return keyEvent{}
		}
	}
	if d.escape == 2 {
		if key >= 0x40 && key <= 0x7e {
			d.escape = 0
			return d.sequence(key)
		}
		if len(d.params) >= 96 {
			d.escape = 0
			return keyEvent{}
		}
		d.params = append(d.params, key)
		return keyEvent{}
	}
	return keyEvent{key: int(key), action: keyAction(int(key)), shift: key >= 'A' && key <= 'Z'}
}

func (d *keyDecoder) sequence(final byte) keyEvent {
	params := string(d.params)
	if final == 'u' && strings.HasPrefix(params, "?") {
		flags, err := strconv.Atoi(params[1:])
		d.enhanced = err == nil && flags&11 == 11
		return keyEvent{}
	}
	if strings.HasPrefix(params, "<") && (final == 'M' || final == 'm') {
		return mouseEvent(params[1:], final)
	}
	if final == 'O' && params == "" {
		return keyEvent{focusOut: true}
	}
	fields := strings.Split(params, ";")
	event := keyEvent{enhanced: d.enhanced}
	modifiers := 1
	if len(fields) > 1 {
		parts := strings.Split(fields[1], ":")
		if parts[0] != "" {
			value, err := strconv.Atoi(parts[0])
			if err != nil || value < 1 {
				return keyEvent{}
			}
			modifiers = value
		}
		if len(parts) > 1 {
			typeID, err := strconv.Atoi(parts[1])
			if err != nil || typeID < 1 || typeID > 3 {
				return keyEvent{}
			}
			event.enhanced, event.release = true, typeID == 3
			event.repeat = typeID == 2
		}
	}
	switch final {
	case 'D':
		event.key = keyLeft
	case 'C':
		event.key = keyRight
	case 'A':
		event.key = keyUp
	case 'B':
		event.key = keyDown
	case 'u':
		code, err := strconv.Atoi(strings.Split(fields[0], ":")[0])
		if err != nil || code < 1 || code > 0x10ffff {
			return keyEvent{}
		}
		event.key = code
	default:
		return keyEvent{}
	}
	event.action = keyAction(event.key)
	event.shift = (modifiers-1)&1 != 0
	if (event.key == 'c' || event.key == 'd') && (modifiers-1)&4 != 0 {
		event.action = quit
	}
	return event
}

// mouseEvent reports pointer movement, left clicks and the wheel in any direction, and ignores releases and other buttons.
func mouseEvent(params string, final byte) keyEvent {
	fields := strings.Split(params, ";")
	if len(fields) != 3 || final != 'M' {
		return keyEvent{}
	}
	button, buttonErr := strconv.Atoi(fields[0])
	column, columnErr := strconv.Atoi(fields[1])
	row, rowErr := strconv.Atoi(fields[2])
	if buttonErr != nil || columnErr != nil || rowErr != nil {
		return keyEvent{}
	}
	if button&64 != 0 {
		switch button & 3 {
		case 0:
			return keyEvent{zoom: 1}
		case 1:
			return keyEvent{zoom: -1}
		case 2:
			return keyEvent{pan: -1}
		case 3:
			return keyEvent{pan: 1}
		}
		return keyEvent{}
	}
	if button&32 != 0 {
		return keyEvent{mouse: true, column: column, row: row}
	}
	if button&3 == 0 {
		return keyEvent{click: true, column: column, row: row}
	}
	return keyEvent{}
}

type heldKeys map[int]action

func (keys heldKeys) update(event keyEvent) {
	if event.focusOut {
		clear(keys)
		return
	}
	// Every enhanced event carries the current modifier state, including Shift
	// pressed before a direction and either Shift released while the other is held.
	if event.enhanced && event.key != 0 {
		if event.shift {
			keys[keyShift] = none
		} else {
			delete(keys, keyShift)
		}
	}
	if event.release {
		delete(keys, event.key)
	} else if event.enhanced && event.action >= left && event.action <= down {
		keys[event.key] = event.action
	}
}

func (keys heldKeys) sprinting() bool {
	_, held := keys[keyShift]
	return held
}

func (keys heldKeys) direction() (dx, dz float64) {
	var active [5]bool
	for _, direction := range keys {
		active[direction] = true
	}
	if active[left] {
		dx--
	}
	if active[right] {
		dx++
	}
	if active[up] {
		dz--
	}
	if active[down] {
		dz++
	}
	return dx, dz
}

// vector turns a single movement action into a walking direction.
func (a action) vector() (dx, dz float64) {
	switch a {
	case left:
		return -1, 0
	case right:
		return 1, 0
	case up:
		return 0, -1
	case down:
		return 0, 1
	}
	return 0, 0
}
