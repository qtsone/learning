package main

import (
	"fmt"

	"tutor.local/packages-modules/report"
)

func main() {
	fmt.Println(report.Line("Berlin", 25))
	fmt.Println(report.Line("Oslo", -8.5))
}
