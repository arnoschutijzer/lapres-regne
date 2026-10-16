package world

import "time"

// LevelUp is a skill level the hero reached while playing.
type LevelUp struct {
	Skill string
	Level int
	// Unlocked lists the trees the new level lets the hero chop for the first time.
	Unlocked []TreeKind
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

// skillCount is how many skills the hero trains; woodcutting is the only one so far.
const skillCount = 1

// maxTotalLevel is the total level with every skill at the highest level.
const maxTotalLevel = skillCount * maxLevel

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

// celebrate notes a woodcutting level reached by going from one level to another.
func (w *World) celebrate(from, to int) {
	levelUp := LevelUp{Skill: "Woodcutting", Level: to, Milestone: milestoneOf(to, w.totalLevel())}
	for kind, tree := range species {
		if tree.level > from && tree.level <= to {
			levelUp.Unlocked = append(levelUp.Unlocked, TreeKind(kind))
		}
	}
	w.celebration = celebration{active: true, levelUp: levelUp}
	w.sounds.LevelUp(len(levelUp.Unlocked) > 0)
}

// totalLevel adds up the hero's levels across all skills.
func (w *World) totalLevel() int {
	return w.hero.woodcutting.Level()
}

// ageCelebration lets time pass since the last level-up.
func (w *World) ageCelebration(dt time.Duration) {
	if w.celebration.active {
		w.celebration.since += dt
	}
}
