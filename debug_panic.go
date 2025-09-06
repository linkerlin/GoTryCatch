package main

import (
	"fmt"
)

func testBasicPanic() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered: %v (type: %T)\n", r, r)
		}
	}()
	
	panic("test panic")
}

func main() {
	fmt.Println("Testing basic panic mechanism:")
	testBasicPanic()
	fmt.Println("Done")
}
