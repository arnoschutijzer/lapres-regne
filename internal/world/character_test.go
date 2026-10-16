package world_test

import (
	"testing"

	"lapresregne/internal/world"
)

func TestWalkingFacesTheWalkingDirectionUntilAimed(t *testing.T) {
	w := world.New(7)
	w.Walk(-1, 0, false)
	if facing := w.Hero().Facing(); facing != world.FacingLeft {
		t.Fatalf("facing = %v, want left after walking left", facing)
	}
	w.Walk(0, -1, false)
	if facing := w.Hero().Facing(); facing != world.FacingUp {
		t.Fatalf("facing = %v, want up after walking up", facing)
	}
}

func TestAimingFacesTheTargetInFourDirections(t *testing.T) {
	cases := []struct {
		name string
		x, z float64
		want world.Facing
	}{
		{"left", -3, 1, world.FacingLeft},
		{"right", 3, -1, world.FacingRight},
		{"up", 1, -3, world.FacingUp},
		{"down", -1, 3, world.FacingDown},
		{"diagonal tie", 2, -2, world.FacingUp},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := world.New(7)
			w.Hero().AimAt(c.x, c.z)
			if facing := w.Hero().Facing(); facing != c.want {
				t.Fatalf("facing = %v, want %v", facing, c.want)
			}
		})
	}
}

func TestAimingWinsOverWalking(t *testing.T) {
	w := world.New(7)
	w.Hero().AimAt(-5, 0)
	w.Walk(1, 0, false)
	w.Advance(1, 0, 0, false)
	if facing := w.Hero().Facing(); facing != world.FacingLeft {
		t.Fatalf("facing = %v, want left while walking right after aiming left", facing)
	}
	if x, _ := w.Hero().Position(); x <= 0 {
		t.Fatalf("x = %v, aiming must not stop the hero from walking", x)
	}
}

func TestAimingAtTheHeroKeepsTheLastFacing(t *testing.T) {
	w := world.New(7)
	w.Walk(1, 0, false)
	x, z := w.Hero().Position()
	w.Hero().AimAt(x, z)
	if facing := w.Hero().Facing(); facing != world.FacingRight {
		t.Fatalf("facing = %v, want right to be kept", facing)
	}
}
