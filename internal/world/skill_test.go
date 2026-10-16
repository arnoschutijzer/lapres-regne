package world_test

import (
	"testing"

	"dither-cli/internal/world"
)

func TestLevelsFollowTheOldSchoolExperienceCurve(t *testing.T) {
	cases := []struct {
		xp    float64
		level int
	}{
		{0, 1}, {82.9, 1}, {83, 2}, {1153, 9}, {1154, 10}, {13034430, 98}, {13034431, 99}, {200_000_000, 99},
	}
	for _, c := range cases {
		if level := world.LevelAt(c.xp); level != c.level {
			t.Errorf("LevelAt(%v) = %d, want %d", c.xp, level, c.level)
		}
	}
}
