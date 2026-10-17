package world

import "math"

// maxLevel is the highest level a skill reaches.
const maxLevel = 99

// levelXP holds the experience each level starts at, following Old School RuneScape's curve.
var levelXP = func() [maxLevel + 1]float64 {
	var table [maxLevel + 1]float64
	points := 0.0
	for level := 2; level <= maxLevel; level++ {
		n := float64(level - 1)
		points += math.Floor(n + 300*math.Pow(2, n/7))
		table[level] = math.Floor(points / 4)
	}
	return table
}()

// LevelAt is the level reached with an amount of experience.
func LevelAt(xp float64) int {
	level := 1
	for level < maxLevel && xp >= levelXP[level+1] {
		level++
	}
	return level
}

// Woodcutting names the woodcutting skill.
const Woodcutting = "Woodcutting"

// skillNames lists every skill the hero trains.
var skillNames = [...]string{Woodcutting}

// Skill is the experience gained in one skill, kept in tenths so half points add up exactly.
type Skill struct {
	tenths int
}

func (s Skill) XP() float64 { return float64(s.tenths) / 10 }
func (s Skill) Level() int  { return LevelAt(s.XP()) }

func (s *Skill) gain(xp float64) {
	s.tenths += int(math.Round(xp * 10))
}
