package terminal

import (
	"strings"
	"testing"
)

func TestTheWorldUsesSolidColorInsteadOfDitheredDots(t *testing.T) {
	r := newRenderer(meadow(), "L’Après-Règne")
	frame := r.frame(100, 30, 0)
	for _, glyph := range frame {
		if glyph >= 0x2800 && glyph <= 0x28ff {
			t.Fatal("the world must use solid surfaces instead of braille dots")
		}
	}
	if !strings.Contains(frame, "\x1b[48;2;") {
		t.Fatal("the world must shade surfaces with background colors")
	}
}
