package terminal

import (
	"math"
	"time"

	"lapresregne/internal/world"
)

// forestScale is the canvas pixels per world unit of the normal view.
const forestScale = 3.6

// zoomStep is how much closer each of the camera's zoom steps brings the view.
const zoomStep = 1.25

var closeUpScale = forestScale * math.Pow(zoomStep, world.CloseUp)

// zoom eases the view toward the camera's zoom, one step at a time.
type zoom struct {
	current float64
}

// step glides toward the camera's zoom step, slowly enough to feel calm.
func (z *zoom) step(target int, dt time.Duration) {
	z.current += (float64(target) - z.current) * (1 - math.Exp(-6*dt.Seconds()))
	if math.Abs(float64(target)-z.current) < 0.002 {
		z.current = float64(target)
	}
}

func (z *zoom) scale() float64 {
	return forestScale * math.Pow(zoomStep, z.current)
}
