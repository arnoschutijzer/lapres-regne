package terminal

import (
	"math"
	"time"
)

// forestScale is the canvas pixels per world unit of the normal view.
const forestScale = 3.6

// zoomStep is how much closer each of the camera's zoom steps brings the view.
const zoomStep = 1.25

var closeUpScale = forestScale * math.Pow(zoomStep, closeUpZoom)

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

// zoomShown is the finest change of zoom the view shows: a 48th of a step, which moves the edge of
// the widest view about a pixel, so the ease's last fractions of a pixel leave the view still.
const zoomShown = 1.0 / 48

func (z *zoom) scale() float64 {
	return forestScale * math.Pow(zoomStep, math.Round(z.current/zoomShown)*zoomShown)
}

// panColumns is how many terminal columns the view slides for one sideways scroll step.
const panColumns = 3

// panDistance is how far in world units one sideways scroll step pans, so the view slides the
// same columns at any zoom. Each column holds two canvas pixels.
func (z *zoom) panDistance() float64 {
	return panColumns * 2 / z.scale()
}
