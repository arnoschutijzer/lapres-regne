package terminal

import "testing"

func feed(d *keyDecoder, input string) []keyEvent {
	var events []keyEvent
	for i := range len(input) {
		events = append(events, d.feed(input[i]))
	}
	return events
}

func TestMouseMotionReportsTheCell(t *testing.T) {
	var d keyDecoder
	events := feed(&d, "\x1b[<35;12;7M")
	last := events[len(events)-1]
	if !last.mouse || last.column != 12 || last.row != 7 {
		t.Fatalf("event = %+v, want mouse motion at column 12, row 7", last)
	}
}

func TestLeftClicksReportTheCell(t *testing.T) {
	var d keyDecoder
	events := feed(&d, "\x1b[<0;12;7M")
	last := events[len(events)-1]
	if !last.click || last.mouse || last.column != 12 || last.row != 7 {
		t.Fatalf("event = %+v, want a left click at column 12, row 7", last)
	}
}

func TestMouseReleasesAndOtherButtonsAreIgnored(t *testing.T) {
	var d keyDecoder
	for _, event := range feed(&d, "\x1b[<0;12;7m\x1b[<2;12;7M") {
		if event.mouse || event.click || event.zoom != 0 || event.panRight != 0 || event.panDown != 0 || event.action != none {
			t.Fatalf("event = %+v, want it ignored", event)
		}
	}
}

func TestTheWheelZoomsInAndOut(t *testing.T) {
	var d keyDecoder
	events := feed(&d, "\x1b[<64;12;7M")
	if in := events[len(events)-1]; in.zoom != 1 || in.click || in.mouse {
		t.Fatalf("event = %+v, want scrolling up to zoom in", in)
	}
	events = feed(&d, "\x1b[<65;12;7M")
	if out := events[len(events)-1]; out.zoom != -1 {
		t.Fatalf("event = %+v, want scrolling down to zoom out", out)
	}
	// Some terminals mark wheel events that arrive while the pointer moves.
	events = feed(&d, "\x1b[<96;12;7M")
	if moving := events[len(events)-1]; moving.zoom != 1 {
		t.Fatalf("event = %+v, want scrolling up while moving to zoom in", moving)
	}
}

func TestKeysStillDecodeAfterMouseInput(t *testing.T) {
	var d keyDecoder
	events := feed(&d, "\x1b[<35;1;1Md")
	if last := events[len(events)-1]; last.action != right {
		t.Fatalf("event = %+v, want D to walk right", last)
	}
}

func TestEOpensTheActionMenuAndDigitsChoose(t *testing.T) {
	var d keyDecoder
	if event := d.feed('e'); event.action != interact {
		t.Fatalf("event = %+v, want E to interact", event)
	}
	if events := feed(&d, "\x1b[101;1u"); events[len(events)-1].action != interact {
		t.Fatalf("event = %+v, want an encoded E to interact", events[len(events)-1])
	}
	if event := d.feed('1'); event.action != firstChoice {
		t.Fatalf("event = %+v, want 1 to pick the first menu entry", event)
	}
}

func TestTabTogglesTheInventory(t *testing.T) {
	var d keyDecoder
	if event := d.feed('\t'); event.action != toggleInventory {
		t.Fatalf("event = %+v, want Tab to toggle the inventory", event)
	}
	if events := feed(&d, "\x1b[9u"); events[len(events)-1].action != toggleInventory {
		t.Fatalf("event = %+v, want an encoded Tab to toggle the inventory", events[len(events)-1])
	}
}

func TestXDropsALog(t *testing.T) {
	var d keyDecoder
	if event := d.feed('x'); event.action != dropLog {
		t.Fatalf("event = %+v, want X to drop a log", event)
	}
}

func TestBracketsMoveBetweenInventorySlots(t *testing.T) {
	var d keyDecoder
	if event := d.feed('['); event.action != previousSlot {
		t.Fatalf("event = %+v, want [ to select the previous slot", event)
	}
	if event := d.feed(']'); event.action != nextSlot {
		t.Fatalf("event = %+v, want ] to select the next slot", event)
	}
}

func TestTheWheelWithOptionPans(t *testing.T) {
	var d keyDecoder
	// Option sets the wheel button's Meta bit, 8.
	for _, wheel := range []struct {
		button      string
		right, down int
	}{{"72", 0, -1}, {"73", 0, 1}, {"74", -1, 0}, {"75", 1, 0}} {
		events := feed(&d, "\x1b[<"+wheel.button+";12;7M")
		if event := events[len(events)-1]; event.panRight != wheel.right || event.panDown != wheel.down || event.zoom != 0 {
			t.Fatalf("event = %+v for button %s, want a pan of (%d, %d)", event, wheel.button, wheel.right, wheel.down)
		}
	}
}
