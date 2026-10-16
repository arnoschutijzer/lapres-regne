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

func TestMouseReleasesOtherButtonsAndWheelAreIgnored(t *testing.T) {
	var d keyDecoder
	for _, event := range feed(&d, "\x1b[<0;12;7m\x1b[<2;12;7M\x1b[<64;12;7M\x1b[<96;12;7M") {
		if event.mouse || event.click || event.action != none {
			t.Fatalf("event = %+v, want it ignored", event)
		}
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
