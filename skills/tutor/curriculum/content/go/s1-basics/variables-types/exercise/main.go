package main

import "fmt"

func main() {
	fmt.Println(ZeroReport())
	fmt.Println(PriceTag("coffee", 350))
	fmt.Println("average:", Average(7, 2))
	fmt.Println("Monday is day", Monday)

	// TODO: declare a variable holding "Go" with := and print
	// "learning Go" using it. Check the output with go run .
}
