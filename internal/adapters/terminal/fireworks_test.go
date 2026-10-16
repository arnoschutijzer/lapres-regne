package terminal

import (
	"testing"
	"time"
)

// fireworkPixels finds the canvas pixels the fireworks lit.
func fireworkPixels(c *forestCanvas) (xs, ys []int) {
	for i, p := range c.pixels {
		for _, ink := range fireworkInks {
			if p.ink == ink {
				xs, ys = append(xs, i%c.width), append(ys, i/c.width)
			}
		}
	}
	return xs, ys
}

func TestNoFireworksGoOffWithoutALevelUp(t *testing.T) {
	c := newRenderer(besideTree(), "L’Après-Règne").render(100, 27, 0)
	if xs, _ := fireworkPixels(c); len(xs) != 0 {
		t.Fatalf("%d firework pixels without a level-up, want none", len(xs))
	}
}

func TestALevelUpSetsOffFireworksAroundTheHero(t *testing.T) {
	w := levelledUp(t, 75, 400*time.Millisecond)
	c := newRenderer(w, "L’Après-Règne").render(100, 27, 0)
	xs, ys := fireworkPixels(c)
	if len(xs) == 0 {
		t.Fatal("a level-up must set off fireworks")
	}
	heroX, heroZ := w.Hero().Position()
	headX, headY, _ := c.project(heroX, 1.6, heroZ)
	_, feetY, _ := c.project(heroX, 0, heroZ)
	for i := range xs {
		if dx := float64(xs[i]) - headX; dx < -40 || dx > 40 || float64(ys[i]) > feetY || float64(ys[i]) < headY-50 {
			t.Fatalf("a spark at (%d, %d) is far from the hero's head at (%.0f, %.0f)", xs[i], ys[i], headX, headY)
		}
	}
}

func TestTheFireworksBurstOneColourAfterAnother(t *testing.T) {
	colours := map[int]bool{}
	for after := 100 * time.Millisecond; after < time.Duration(fireworksDuration*float64(time.Second)); after += 150 * time.Millisecond {
		c := newRenderer(levelledUp(t, 75, after), "L’Après-Règne").render(100, 27, 0)
		for _, p := range c.pixels {
			for _, ink := range fireworkInks {
				if p.ink == ink {
					colours[ink] = true
				}
			}
		}
	}
	if len(colours) != len(fireworkInks) {
		t.Fatalf("the bursts showed %d colours, want all %d", len(colours), len(fireworkInks))
	}
}

func TestTheFireworksDieDownWhileTheBannerStillShows(t *testing.T) {
	after := time.Duration((fireworksDuration + 0.1) * float64(time.Second))
	c := newRenderer(levelledUp(t, 75, after), "L’Après-Règne").render(100, 27, 0)
	if xs, _ := fireworkPixels(c); len(xs) != 0 {
		t.Fatalf("%d firework pixels after the display, want none", len(xs))
	}
	if fireworksDuration >= bannerFadeIn+bannerRest {
		t.Fatalf("the fireworks last %.1f s, want them over before the banner fades", fireworksDuration)
	}
}
