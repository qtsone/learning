// Package player is a tiny game-state package for practicing pointers.
package player

// Player is one participant in the game. HP must never exceed MaxHP.
type Player struct {
	Name  string
	HP    int
	MaxHP int
}

// Swap exchanges the values stored at a and b.
func Swap(a, b *int) {
	// TODO: swap the two values through the pointers.
}

// ValueOr returns the value p points to, or fallback if p is nil.
func ValueOr(p *int, fallback int) int {
	// TODO: guard against nil, then dereference.
	return 0
}

// Heal raises p's HP by amount, clamped to MaxHP.
// A nil p is ignored: healing nobody does nothing.
func Heal(p *Player, amount int) {
	// TODO: implement per the acceptance criteria in LESSON.md.
}

// NewPlayer creates a Player at full health and returns a pointer to it.
func NewPlayer(name string, maxHP int) *Player {
	// TODO: build the Player and return its address.
	return nil
}
