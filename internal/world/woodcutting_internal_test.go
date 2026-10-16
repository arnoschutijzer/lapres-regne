package world

import "testing"

func TestTheChanceToCutALogRisesWithLevel(t *testing.T) {
	if chance := logChance(1); chance != 65.0/256 {
		t.Fatalf("chance at level 1 = %v, want 65/256", chance)
	}
	if chance := logChance(99); chance != 201.0/256 {
		t.Fatalf("chance at level 99 = %v, want 201/256", chance)
	}
	for level := 2; level <= 99; level++ {
		if logChance(level) < logChance(level-1) {
			t.Fatalf("chance at level %d drops below level %d", level, level-1)
		}
	}
}
