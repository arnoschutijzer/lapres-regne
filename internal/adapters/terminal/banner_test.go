package terminal

import (
	"strings"
	"testing"
	"time"

	"lapresregne/internal/world"
)

// savedProgress resumes the world from one saved snapshot.
type savedProgress struct{ snapshot world.Snapshot }

func (p savedProgress) Load() (world.Snapshot, bool, error) { return p.snapshot, true, nil }
func (p savedProgress) Save(world.Snapshot) error           { return nil }

// levelledUp fells the seed 7 tree at x 6.0 with a hero whose xp that log lifts to the next level,
// then lets some time pass since the level-up.
func levelledUp(t *testing.T, xp float64, after time.Duration) *world.World {
	t.Helper()
	w, err := world.Resume(7, savedProgress{world.Snapshot{Seed: 1, WoodcuttingXP: xp}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		w.Walk(0, 1, false)
	}
	for range 9 {
		w.Walk(1, 0, false)
	}
	w.Follow(time.Second)
	w.Chop()
	chopUntilFelled(t, w)
	w.Advance(0, 0, after, false)
	return w
}

func TestTheBannerEasesInRestsAndEasesOutWithTheJingle(t *testing.T) {
	if bannerAlpha(0) != 0 || bannerAlpha(bannerFadeIn/2) <= 0 || bannerAlpha(bannerFadeIn/2) >= 1 {
		t.Fatalf("alpha = %.2f at the start, %.2f halfway in; want it to ease in from nothing", bannerAlpha(0), bannerAlpha(bannerFadeIn/2))
	}
	if bannerAlpha(bannerFadeIn) != 1 || bannerAlpha(bannerFadeIn+bannerRest) != 1 {
		t.Fatal("the banner must rest fully visible between its fades")
	}
	end := bannerFadeIn + bannerRest + bannerFadeOut
	if fading := bannerAlpha(end - bannerFadeOut/2); fading <= 0 || fading >= 1 {
		t.Fatalf("alpha = %.2f halfway out, want it easing away", fading)
	}
	if bannerAlpha(end) != 0 || bannerAlpha(end+1) != 0 {
		t.Fatal("the banner must be gone once the jingle ends")
	}
	if end != 6 {
		t.Fatalf("the banner lasts %.1f s, want the jingle's 6 s", end)
	}
}

func TestALevelUpShowsABannerOverTheTopOfTheForest(t *testing.T) {
	r := newRenderer(besideTree(), "L’Après-Règne")
	if frame := r.frame(100, 30, 0); strings.Contains(frame, "Congratulations") {
		t.Fatal("no banner must show without a level-up")
	}
	w := levelledUp(t, 75, 2*time.Second)
	r = newRenderer(w, "L’Après-Règne")
	frame := r.frame(100, 30, 0)
	if !strings.Contains(frame, "Congratulations!") || !strings.Contains(frame, "Woodcutting level 2") {
		t.Fatal("a level-up must announce the skill and its new level")
	}
	if strings.Contains(frame, "You can now chop") {
		t.Fatal("level 2 unlocks no tree and must not claim one")
	}
	icon, veiled := 0, 0
	for row := range bannerHeight {
		for _, cell := range r.cells[row*100 : (row+1)*100] {
			if cell.glyph >= 0x2800 && cell.glyph <= 0x28ff && cell.ink == 4 {
				icon++
			}
			if cell.glyph == 0 && cell.veil > 0 {
				veiled++
			}
		}
	}
	if icon == 0 {
		t.Fatal("the banner must show the woodcutting icon's bark-colored trunk")
	}
	if veiled == 0 {
		t.Fatal("the forest under the banner must be dimmed")
	}
	if below := r.cells[bannerHeight*100 : (bannerHeight+1)*100]; below[50].veil != 0 {
		t.Fatal("the forest below the banner must stay undimmed")
	}
}

func TestTheBannerNamesTheTreesANewLevelUnlocks(t *testing.T) {
	r := newRenderer(levelledUp(t, 2400, 2*time.Second), "L’Après-Règne")
	frame := r.frame(100, 30, 0)
	if !strings.Contains(frame, "Woodcutting level 15") || !strings.Contains(frame, "You can now chop oaks.") {
		t.Fatal("level 15 must announce that oaks can now be chopped")
	}
}

func TestTheBannerFitsTheSmallestTerminal(t *testing.T) {
	r := newRenderer(levelledUp(t, 2400, 2*time.Second), "L’Après-Règne")
	if frame := r.frame(40, 14, 0); !strings.Contains(frame, "You can now chop oaks.") {
		t.Fatal("the whole banner must fit 40 columns")
	}
}

func TestTheForestReturnsOnceTheBannerFades(t *testing.T) {
	w := levelledUp(t, 75, 2*time.Second)
	r := newRenderer(w, "L’Après-Règne")
	r.frame(100, 30, 0)
	w.Advance(0, 0, 5*time.Second, false)
	if frame := r.frame(100, 30, 0); strings.Contains(frame, "Congratulations") {
		t.Fatal("the banner must not be drawn after it fades")
	}
	for i, cell := range r.cells {
		if cell.glyph != 0 || cell.veil != 0 {
			t.Fatalf("cell %d = %+v, want the forest fully redrawn", i, cell)
		}
	}
}
