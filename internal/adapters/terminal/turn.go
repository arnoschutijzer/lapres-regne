package terminal

import (
	"math"
	"time"
)

// turning eases the view toward the camera's yaw, as calmly as the zoom.
type turning struct {
	// current is the eased angle, and target the angle it eases toward.
	current, target float64
}

func (t *turning) step(target float64, dt time.Duration) {
	t.target = target
	t.current += (target - t.current) * (1 - math.Exp(-6*dt.Seconds()))
	if math.Abs(target-t.current) < 0.0005 {
		t.current = target
	}
}

// turnShown is the finest turn the view shows: a 16th of the camera's turning step, which moves the
// edge of the widest view about a pixel, so the ease's last fractions of a pixel leave the view
// still.
const turnShown = math.Pi / 2 / quarterTurn / 16

// shown is the angle the view is drawn at: the eased angle, in whole turnShown from its target.
func (t *turning) shown() float64 {
	return t.target + math.Round((t.current-t.target)/turnShown)*turnShown
}
