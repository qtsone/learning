package sq

// Balanced reports whether every bracket in s — (), [], {} — is closed in
// the right order. Non-bracket characters are ignored.
func Balanced(s string) bool {
	pairs := map[rune]rune{')': '(', ']': '[', '}': '{'}
	var st Stack
	for _, r := range s {
		switch r {
		case '(', '[', '{':
			st.Push(r)
		case ')', ']', '}':
			open, ok := st.Pop()
			if !ok || open != pairs[r] {
				return false
			}
		}
	}
	return st.Len() == 0
}
