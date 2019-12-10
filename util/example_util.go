package util

import (
	"fmt"
)

func ExampleArangeByte_EndOnly() {
	fmt.Println(ArangeByte(10))
	// Output: [0 1 2 3 4 5 6 7 8 9]
}

func ExampleArangeByte_StartEnd() {
	fmt.Println(ArangeByte(5, 15))
	// Output: [5 6 7 8 9 10 11 12 13 14]
}

func ExampleArangeByte_StartEndStep() {
	fmt.Println(ArangeByte(10, 22, 2))
	// Output: [10 12 14 16 18 20]
}
