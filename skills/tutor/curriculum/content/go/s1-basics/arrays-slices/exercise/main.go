package main

import "fmt"

func main() {
	scores := []int{72, 90, 45, 88}
	fmt.Println("scores:  ", scores)
	fmt.Println("clone:   ", Clone(scores))
	fmt.Println("insert:  ", Insert(scores, 2, 100))
	fmt.Println("remove:  ", Remove(scores, 0))
	fmt.Println("above 80:", KeepAbove(scores, 80))
	fmt.Println("original:", scores) // must still be 72 90 45 88
}
