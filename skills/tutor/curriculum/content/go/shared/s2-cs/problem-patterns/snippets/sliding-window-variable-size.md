> **In Go:** substring problems mean runes, not bytes — the strings-runes
> lesson applies in full. Convert once with `[]rune(s)` and index that, and
> keep the sightings in a `map[rune]int`.
