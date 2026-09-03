package main

import "fmt"

func main() {
	fmt.Println(ZeroReport())
	fmt.Println(PriceTag("coffee", 350))
	fmt.Println("average:", Average(7, 2))
	fmt.Println("Monday is day", Monday)

	language := "Go"
	fmt.Println("learning", language)
}
