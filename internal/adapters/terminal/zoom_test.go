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
	if c := newRenderer(world.New(7), NewCamera(closeUpZoom, 0), "L’Après-Règne").render(100, 27, 0); c.scale != closeUpScale {
		t.Fatalf("scale = %v on the first frame, want the close-up %v without easing in", c.scale, closeUpScale)
	}
}

func TestAPanStepCrossesTheSameColumnsAtAnyZoom(t *testing.T) {
	forest := zoom{current: 0}
	closeUp := zoom{current: closeUpZoom}
	forestColumns := forest.panDistance() * forest.scale() / 2
	closeUpColumns := closeUp.panDistance() * closeUp.scale() / 2
	if forestColumns != panColumns || math.Abs(closeUpColumns-panColumns) > 1e-9 {
		t.Fatalf("a pan step crosses %v columns in the forest and %v up close, want %v", forestColumns, closeUpColumns, panColumns)
	}
}
