package terminal

import (
	"math"
	"time"
)

// forestScale is the canvas pixels per world unit of the normal view.
const forestScale = 3.6

// zoomSteps is how many wheel steps, each zoomStep times closer, lead to the close-up.
const zoomSteps, zoomStep = 12, 1.25

var closeUpScale = forestScale * math.Pow(zoomStep, zoomSteps)

// zoom eases the view between the forest and a close-up of the hero, one wheel step at a time.
type zoom struct {
	target  int
	current float64
}

func (z *zoom) scroll(steps int) {
	z.target = min(zoomSteps, max(0, z.target+steps))
}

// step glides toward the scrolled level, slowly enough to feel calm.
func (z *zoom) step(dt time.Duration) {
	z.current += (float64(z.target) - z.current) * (1 - math.Exp(-6*dt.Seconds()))
	if math.Abs(float64(z.target)-z.current) < 0.002 {
		z.current = float64(z.target)
	}
}

func (z *zoom) scale() float64 {
	return forestScale * math.Pow(zoomStep, z.current)
}
