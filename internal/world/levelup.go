package world

import "time"

// LevelUp is a skill level the hero reached while playing.
type LevelUp struct {
	Skill string
	Level int
	// Unlocked tells what the new level lets the hero do for the first time; empty for nothing.
	Unlocked string
	// Milestone marks the levels Old School RuneScape celebrates with fireworks of their own.
	Milestone Milestone
}

// Milestone is a level-up celebrated beyond the usual.
type Milestone int

const (
	NoMilestone Milestone = iota
	// SkillMastered is level 99 in a skill.
	SkillMastered
	// TotalMaxed is level 99 in every skill: the max total level, which outshines the mastery.
	TotalMaxed
)

// maxTotalLevel is the total level with every skill at the highest level.
const maxTotalLevel = len(skillNames) * maxLevel

// milestoneOf tells which milestone, if any, reaching a level in a skill is, given the total level
// across all skills it brings.
func milestoneOf(level, totalLevel int) Milestone {
	switch {
	case totalLevel == maxTotalLevel:
		return TotalMaxed
	case level == maxLevel:
		return SkillMastered
	}
	return NoMilestone
}

type celebration struct {
	active  bool
	levelUp LevelUp
	since   time.Duration
}

// LatestLevelUp is the last level the hero reached this session and how long ago they reached it.
func (w *World) LatestLevelUp() (LevelUp, time.Duration, bool) {
	return w.celebration.levelUp, w.celebration.since, w.celebration.active
}

// unlocks tells what going from one level of a skill to another lets the hero do for the first
// time, or empty for nothing.
type unlocks func(from, to int) string

// train gives the hero experience in a skill, celebrating any level it brings with what it unlocks.
func (w *World) train(skill string, xp float64, unlocked unlocks) {
	if from, to := w.hero.train(skill, xp); to > from {
		w.celebrate(LevelUp{Skill: skill, Level: to, Unlocked: unlocked(from, to), Milestone: milestoneOf(to, w.totalLevel())})
	}
}

// celebrate notes a level reached.
func (w *World) celebrate(levelUp LevelUp) {
	w.celebration = celebration{active: true, levelUp: levelUp}
	if levelUp.Unlocked != "" {
		w.play(UnlockingJingle)
	} else {
		w.play(LevelUpJingle)
	}
}

// totalLevel adds up the hero's levels across all skills.
func (w *World) totalLevel() int {
	total := 0
	for _, skill := range skillNames {
		total += w.hero.Skill(skill).Level()
	}
	return total
}

// ageCelebration lets time pass since the last level-up.
func (w *World) ageCelebration(dt time.Duration) {
	if w.celebration.active {
		w.celebration.since += dt
	}
}
