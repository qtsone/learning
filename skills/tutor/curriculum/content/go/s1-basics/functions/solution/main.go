package main

import "fmt"

func main() {
	prices := []int{1250, 899, 1451}

	share, remainder, err := SplitBill(3, prices...)
	if err != nil {
		fmt.Println("cannot split the bill:", err)
		return
	}
	fmt.Println("each pays:", FormatCents(share))
	if remainder > 0 {
		fmt.Println("left in the jar:", FormatCents(remainder))
	}
}
