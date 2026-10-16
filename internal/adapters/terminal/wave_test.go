package terminal

import (
	"math"
	"testing"
)

func TestAWaveFollowsTheSineClosely(t *testing.T) {
	for x := -1000.0; x <= 1000; x += 0.0137 {
		if difference := math.Abs(wave(x) - math.Sin(x)); difference > 0.002 {
			t.Fatalf("wave(%v) = %v, want within 0.002 of sin %v", x, wave(x), math.Sin(x))
		}
	}
}
