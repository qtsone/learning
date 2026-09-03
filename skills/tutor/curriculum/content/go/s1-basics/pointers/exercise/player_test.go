package player

import "testing"

func TestSwap(t *testing.T) {
	a, b := 1, 2
	Swap(&a, &b)
	if a != 2 || b != 1 {
		t.Errorf("after Swap(&a, &b) with a=1, b=2: got a=%d, b=%d, want a=2, b=1", a, b)
	}
}

func TestValueOr(t *testing.T) {
	seven := 7
	cases := []struct {
		name     string
		p        *int
		fallback int
		want     int
	}{
		{"non-nil pointer wins", &seven, 42, 7},
		{"nil pointer falls back", nil, 42, 42},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ValueOr(c.p, c.fallback); got != c.want {
				t.Errorf("ValueOr(p, %d) in case %q = %d, want %d", c.fallback, c.name, got, c.want)
			}
		})
	}
}

func TestHeal(t *testing.T) {
	cases := []struct {
		name   string
		hp     int
		amount int
		want   int
	}{
		{"heals by amount", 50, 30, 80},
		{"clamps at MaxHP", 90, 30, 100},
		{"zero amount changes nothing", 40, 0, 40},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Player{Name: "Gopher", HP: c.hp, MaxHP: 100}
			Heal(&p, c.amount)
			if p.HP != c.want {
				t.Errorf("Heal(&p, %d) with HP=%d, MaxHP=100: p.HP = %d, want %d", c.amount, c.hp, p.HP, c.want)
			}
		})
	}
	t.Run("nil player does not panic", func(t *testing.T) {
		Heal(nil, 10) // must be a no-op, not a crash
	})
}

func TestNewPlayer(t *testing.T) {
	p := NewPlayer("Gopher", 100)
	if p == nil {
		t.Fatal("NewPlayer returned nil — return a pointer to a Player you created inside the function")
	}
	if p.Name != "Gopher" || p.HP != 100 || p.MaxHP != 100 {
		t.Errorf("NewPlayer(%q, 100) = %+v, want Name=Gopher HP=100 MaxHP=100", "Gopher", *p)
	}
	t.Run("each call returns an independent Player", func(t *testing.T) {
		a := NewPlayer("A", 100)
		b := NewPlayer("B", 100)
		if a == b {
			t.Fatal("NewPlayer returned the same pointer twice — each call must create a fresh Player")
		}
		a.HP = 1
		if b.HP != 100 {
			t.Errorf("mutating one player changed another: b.HP = %d, want 100", b.HP)
		}
	})
}
