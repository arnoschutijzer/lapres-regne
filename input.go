package main

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
)

type keyEvent struct {
	key      int
	action   action
	release  bool
	enhanced bool
	focusOut bool
}

const (
	keyLeft = 0x110000 + iota
	keyRight
	keyUp
	keyDown
)

// Kitty's keyboard protocol reports independent key press/repeat/release events.
// Flags 1 + 2 + 8 request disambiguation, event types, and encoded letter keys.
const keyboardStart = "\x1b[>11u\x1b[?u\x1b[?1004h"
const keyboardStop = "\x1b[?1004l\x1b[<u"

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
	return keyEvent{key: int(key), action: keyAction(int(key))}
}

func (d *keyDecoder) sequence(final byte) keyEvent {
	params := string(d.params)
	if final == 'u' && strings.HasPrefix(params, "?") {
		flags, err := strconv.Atoi(params[1:])
		d.enhanced = err == nil && flags&11 == 11
		return keyEvent{}
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
	if (event.key == 'c' || event.key == 'd') && (modifiers-1)&4 != 0 {
		event.action = quit
	}
	return event
}

type heldKeys map[int]action

func (keys heldKeys) update(event keyEvent) {
	if event.focusOut {
		clear(keys)
	} else if event.release {
		delete(keys, event.key)
	} else if event.enhanced && event.action >= left && event.action <= down {
		keys[event.key] = event.action
	}
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
