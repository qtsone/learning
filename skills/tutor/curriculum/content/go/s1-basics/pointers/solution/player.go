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
	*a, *b = *b, *a
}

// ValueOr returns the value p points to, or fallback if p is nil.
func ValueOr(p *int, fallback int) int {
	if p == nil {
		return fallback
	}
	return *p
}

// Heal raises p's HP by amount, clamped to MaxHP.
// A nil p is ignored: healing nobody does nothing.
func Heal(p *Player, amount int) {
	if p == nil {
		return
	}
	p.HP += amount
	if p.HP > p.MaxHP {
		p.HP = p.MaxHP
	}
}

// NewPlayer creates a Player at full health and returns a pointer to it.
func NewPlayer(name string, maxHP int) *Player {
	return &Player{Name: name, HP: maxHP, MaxHP: maxHP}
}
