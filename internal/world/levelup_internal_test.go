package world

import "testing"

func TestLevel99IsAMasteryUntilEverySkillIsMaxed(t *testing.T) {
	if got := milestoneOf(99, maxTotalLevel-1); got != SkillMastered {
		t.Fatalf("milestone = %v for a 99 short of the max total, want a mastered skill", got)
	}
	if got := milestoneOf(99, maxTotalLevel); got != TotalMaxed {
		t.Fatalf("milestone = %v for the 99 that maxes the total, want the max total level", got)
	}
	if got := milestoneOf(98, maxTotalLevel-1); got != NoMilestone {
		t.Fatalf("milestone = %v at level 98, want none", got)
	}
}
