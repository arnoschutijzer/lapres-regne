package world

// Item is a kind of thing the hero can carry in the inventory or leave on the ground.
type Item byte

const (
	Logs Item = iota
	OakLogs
	WillowLogs
	YewLogs
)

func (i Item) String() string {
	return [...]string{"logs", "oak logs", "willow logs", "yew logs"}[i]
}
