package terminal

import (
	"math"
	"time"
)

// turning eases the view toward the camera's yaw, as calmly as the zoom.
type turning struct {
	current float64
}

func (t *turning) step(target float64, dt time.Duration) {
	t.current += (target - t.current) * (1 - math.Exp(-6*dt.Seconds()))
	if math.Abs(target-t.current) < 0.0005 {
		t.current = target
	}
}
