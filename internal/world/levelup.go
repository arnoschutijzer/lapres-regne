package world

import "time"

// LevelUp is a skill level the hero reached while playing.
type LevelUp struct {
	Skill string
	Level int
	// Unlocked lists the trees the new level lets the hero chop for the first time.
	Unlocked []TreeKind
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
	levelUp := LevelUp{Skill: "Woodcutting", Level: to}
	for kind, tree := range species {
		if tree.level > from && tree.level <= to {
			levelUp.Unlocked = append(levelUp.Unlocked, TreeKind(kind))
		}
	}
	w.celebration = celebration{active: true, levelUp: levelUp}
	w.sounds.LevelUp(len(levelUp.Unlocked) > 0)
}

// ageCelebration lets time pass since the last level-up.
func (w *World) ageCelebration(dt time.Duration) {
	if w.celebration.active {
		w.celebration.since += dt
	}
}
