package main

import (
	"fmt"
	"github.com/linkerlin/gotrycatch/errors"
)

type DebugTryBlock struct {
	err     interface{}
	handled bool
}

func DebugTry(fn func()) *DebugTryBlock {
	tb := &DebugTryBlock{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic recovered in Try: %v (type: %T)\n", r, r)
				tb.err = r
			} else {
				fmt.Println("No panic occurred in Try")
			}
		}()

		fmt.Println("About to execute function")
		fn()
		fmt.Println("Function executed successfully")
	}()
	
	return tb
}

func DebugThrow(err interface{}) {
	fmt.Printf("About to panic with: %v (type: %T)\n", err, err)
	panic(err)
}

func main() {
	fmt.Println("Debug Try mechanism:")
	
	tb := DebugTry(func() {
		fmt.Println("Inside function, about to throw")
		DebugThrow(errors.NewValidationError("name", "姓名不能为空", 1001))
		fmt.Println("This should not print")
	})
	
	fmt.Printf("Final result - tb: %v\n", tb != nil)
	if tb != nil {
		fmt.Printf("Final result - err: %v, err type: %T\n", tb.err, tb.err)
	}
}
