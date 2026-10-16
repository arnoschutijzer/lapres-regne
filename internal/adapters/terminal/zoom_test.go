package terminal

import (
	"testing"
	"time"
)

func TestTheViewStartsAtTheForestScale(t *testing.T) {
	var z zoom
	if got := z.scale(); got != forestScale {
		t.Fatalf("scale = %v, want the forest scale %v", got, forestScale)
	}
}

func TestScrollingInEasesTheViewCloser(t *testing.T) {
	var z zoom
	z.scroll(1)
	z.step(50 * time.Millisecond)
	partway := z.scale()
	if partway <= forestScale {
		t.Fatalf("scale = %v after a moment, want it moving closer than %v", partway, forestScale)
	}
	z.step(2 * time.Second)
	if settled := z.scale(); settled <= partway {
		t.Fatalf("scale = %v once settled, want it closer than %v", settled, partway)
	}
}

func TestZoomStaysBetweenTheForestAndACloseUp(t *testing.T) {
	var z zoom
	for range 100 {
		z.scroll(1)
	}
	z.step(5 * time.Second)
	if got := z.scale(); got != closeUpScale {
		t.Fatalf("scale = %v after scrolling far in, want the close-up %v", got, closeUpScale)
	}
	for range 100 {
		z.scroll(-1)
	}
	z.step(5 * time.Second)
	if got := z.scale(); got != forestScale {
		t.Fatalf("scale = %v after scrolling far out, want the forest scale %v", got, forestScale)
	}
}
