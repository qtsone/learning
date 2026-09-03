package main

import "fmt"

// Days of the week, numbered 1-7 like on a calendar.
const (
	Monday = iota + 1
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
	Sunday
)

// ZeroReport declares one variable of each basic type without assigning a
// value and reports what Go initialized them to, in the form:
//
//	count=0 price=0 name="" active=false
func ZeroReport() string {
	var count int
	var price float64
	var name string
	var active bool
	return fmt.Sprintf("count=%d price=%v name=%q active=%t", count, price, name, active)
}

// Average returns sum divided by count, keeping the fraction.
func Average(sum int, count int) float64 {
	return float64(sum) / float64(count)
}

// PriceTag formats a price given in cents as a label like "coffee: $3.50".
func PriceTag(item string, cents int) string {
	dollars := float64(cents) / 100
	return fmt.Sprintf("%s: $%.2f", item, dollars)
}
