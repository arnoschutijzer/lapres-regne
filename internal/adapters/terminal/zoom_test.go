package terminal

import (
	"math"
	"testing"
	"time"

	"lapresregne/internal/world"
)

func TestTheViewStartsAtTheForestScale(t *testing.T) {
	var z zoom
	if got := z.scale(); got != forestScale {
		t.Fatalf("scale = %v, want the forest scale %v", got, forestScale)
	}
}

func TestTheViewEasesTowardTheCameraZoom(t *testing.T) {
	var z zoom
	z.step(3, 50*time.Millisecond)
	partway := z.scale()
	if partway <= forestScale {
		t.Fatalf("scale = %v after a moment, want it moving closer than %v", partway, forestScale)
	}
	z.step(3, 2*time.Second)
	if settled, want := z.scale(), forestScale*math.Pow(zoomStep, 3); settled != want {
		t.Fatalf("scale = %v once settled, want %v three steps in", settled, want)
	}
}

func TestAResumedViewOpensAtTheKeptZoom(t *testing.T) {
	w := world.New(7)
	w.ZoomBy(world.CloseUp)
	if c := newRenderer(w, "L’Après-Règne").render(100, 27, 0); c.scale != closeUpScale {
		t.Fatalf("scale = %v on the first frame, want the close-up %v without easing in", c.scale, closeUpScale)
	}
}
