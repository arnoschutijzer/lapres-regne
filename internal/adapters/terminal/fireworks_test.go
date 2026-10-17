package terminal

import (
	"math"
	"slices"
	"testing"
	"time"

	"lapresregne/internal/world"
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
	c := newRenderer(besideTree(), NewCamera(0, 0), "L’Après-Règne").render(100, 27, 0)
	if xs, _ := fireworkPixels(c); len(xs) != 0 {
		t.Fatalf("%d firework pixels without a level-up, want none", len(xs))
	}
}

func TestALevelUpSetsOffFireworksAroundTheHero(t *testing.T) {
	w := levelledUp(t, 75, 400*time.Millisecond)
	c := newRenderer(w, NewCamera(0, 0), "L’Après-Règne").render(100, 27, 0)
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
	for after := 100 * time.Millisecond; after < time.Duration(displayFor(world.NoMilestone).duration()*float64(time.Second)); after += 150 * time.Millisecond {
		c := newRenderer(levelledUp(t, 75, after), NewCamera(0, 0), "L’Après-Règne").render(100, 27, 0)
		for _, p := range c.pixels {
			for _, ink := range fireworkInks {
				if p.ink == ink {
					colours[ink] = true
				}
			}
		}
	}
	if want := len(displayFor(world.NoMilestone).inks); len(colours) != want {
		t.Fatalf("the bursts showed %d colours, want all %d", len(colours), want)
	}
}

func TestTheFireworksDieDownWhileTheBannerStillShows(t *testing.T) {
	ordinary := displayFor(world.NoMilestone)
	after := time.Duration((ordinary.duration() + 0.1) * float64(time.Second))
	c := newRenderer(levelledUp(t, 75, after), NewCamera(0, 0), "L’Après-Règne").render(100, 27, 0)
	if xs, _ := fireworkPixels(c); len(xs) != 0 {
		t.Fatalf("%d firework pixels after the display, want none", len(xs))
	}
	if ordinary.duration() >= bannerFadeIn+bannerRest {
		t.Fatalf("the fireworks last %.1f s, want them over before the banner fades", ordinary.duration())
	}
}

// displayColours and displayWidth watch a display over its whole run on a bare canvas around a
// hero at the origin: every colour it shows, and the widest it spreads.
func displayColours(display fireworksDisplay) (colours map[int]bool, widest int) {
	colours = map[int]bool{}
	for since := 0.05; since < display.duration(); since += 0.1 {
		c := &forestCanvas{width: 400, height: 300, scale: forestScale}
		c.turnTo(0)
		c.pixels = make([]forestPixel, c.width*c.height)
		for i := range c.pixels {
			c.pixels[i].depth = math.Inf(1)
		}
		drawDisplay(c, display, 7, since, 0, 0, func(level, burst int, salt uint64) float64 {
			return world.New(7).Random(level, burst, salt)
		})
		xs, _ := fireworkPixels(c)
		left, right := c.width, 0
		for _, x := range xs {
			left, right = min(left, x), max(right, x)
		}
		widest = max(widest, right-left)
		for _, p := range c.pixels {
			if p.ink != 0 {
				colours[p.ink] = true
			}
		}
	}
	return colours, widest
}

func TestA99SetsOffBiggerMoreColourfulFireworks(t *testing.T) {
	plainColours, plainWidth := displayColours(displayFor(world.NoMilestone))
	masteryColours, masteryWidth := displayColours(displayFor(world.SkillMastered))
	if len(masteryColours) <= len(plainColours) {
		t.Fatalf("a 99 shows %d colours, an ordinary level %d; want more", len(masteryColours), len(plainColours))
	}
	if masteryWidth <= plainWidth*3/2 {
		t.Fatalf("a 99 spreads %d pixels wide, an ordinary level %d; want it much bigger", masteryWidth, plainWidth)
	}
	mastery, plain := displayFor(world.SkillMastered), displayFor(world.NoMilestone)
	if mastery.activeBursts(2) <= plain.activeBursts(2) {
		t.Fatalf("%d bursts in the air at a 99, %d at an ordinary level; want more at once", mastery.activeBursts(2), plain.activeBursts(2))
	}
}

func TestTheMaxTotalLevelSetsOffRedAndGoldFireworksForLonger(t *testing.T) {
	maxed := displayFor(world.TotalMaxed)
	if maxed.duration() <= displayFor(world.SkillMastered).duration() || maxed.duration() <= bannerFadeIn+bannerRest+bannerFadeOut {
		t.Fatalf("the max total display lasts %.1f s, want it to outlast a 99's and the banner", maxed.duration())
	}
	// Level 99 in the only skill is the max total level.
	for _, after := range []time.Duration{500 * time.Millisecond, 3 * time.Second, 6500 * time.Millisecond} {
		c := newRenderer(levelledUp(t, 13034420, after), NewCamera(0, 0), "L’Après-Règne").render(100, 27, 0)
		xs, _ := fireworkPixels(c)
		if len(xs) == 0 {
			t.Fatalf("no fireworks %v after reaching the max total level", after)
		}
		for _, p := range c.pixels {
			if slices.Contains(fireworkInks[:], p.ink) && p.ink != redFireworks && p.ink != goldFireworks {
				t.Fatalf("a spark of ink %d %v after the max total level, want only red and gold", p.ink, after)
			}
		}
	}
}

func TestTheBurstsFitAboveTheHerosHead(t *testing.T) {
	w := levelledUp(t, 13034420, 0)
	r := newRenderer(w, NewCamera(7, 0), "L’Après-Règne")
	sparks := 0
	for range 28 {
		w.Advance(0, 0, 250*time.Millisecond, false)
		c := r.render(100, 27, 0)
		_, ys := fireworkPixels(c)
		sparks += len(ys)
		for _, y := range ys {
			if y < 2 {
				t.Fatalf("a spark at canvas row %d; want the bursts to fit below the top edge", y)
			}
		}
	}
	if sparks == 0 {
		t.Fatal("want fireworks after reaching the max total level")
	}
}
