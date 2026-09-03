package main

// Sign reports whether n is "negative", "zero", or "positive".
func Sign(n int) string {
	if n < 0 {
		return "negative"
	} else if n == 0 {
		return "zero"
	}
	return "positive"
}

// Award returns the medal for a podium place:
// 1 → "gold", 2 → "silver", 3 → "bronze", anything else → "none".
func Award(place int) string {
	switch place {
	case 1:
		return "gold"
	case 2:
		return "silver"
	case 3:
		return "bronze"
	default:
		return "none"
	}
}

// SumEvens returns the sum of the even numbers from 1 through limit.
func SumEvens(limit int) int {
	sum := 0
	for i := 1; i <= limit; i++ {
		if r := i % 2; r != 0 {
			continue
		}
		sum += i
	}
	return sum
}

// Repeat returns word concatenated times times; "" when times <= 0.
func Repeat(word string, times int) string {
	out := ""
	for range times {
		out += word
	}
	return out
}

// CollatzSteps returns how many steps n (>= 1) takes to reach 1, where a
// step halves an even number and turns an odd n into 3n+1.
func CollatzSteps(n int) int {
	steps := 0
	for n != 1 {
		if n%2 == 0 {
			n /= 2
		} else {
			n = 3*n + 1
		}
		steps++
	}
	return steps
}

// FirstPowerAbove returns the smallest power of two strictly greater
// than limit.
func FirstPowerAbove(limit int) int {
	power := 1
	for {
		if power > limit {
			break
		}
		power *= 2
	}
	return power
}

// CountPrimes returns how many primes exist from 2 through limit.
// A prime is divisible only by 1 and itself.
func CountPrimes(limit int) int {
	count := 0
candidates:
	for n := 2; n <= limit; n++ {
		// Any composite n has a divisor no larger than its square root,
		// so d*d <= n checks far fewer candidates than d < n.
		for d := 2; d*d <= n; d++ {
			if n%d == 0 {
				continue candidates
			}
		}
		count++
	}
	return count
}
