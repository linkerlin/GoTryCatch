# Usage Examples

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/linkerlin/gotrycatch"
    "github.com/linkerlin/gotrycatch/errors"
)

func main() {
    tb := gotrycatch.Try(func() {
        // Code that might panic
        gotrycatch.Throw(errors.NewValidationError("email", "invalid format", 1001))
    })

    tb = gotrycatch.Catch[errors.ValidationError](tb, func(err errors.ValidationError) {
        fmt.Printf("Validation error: %s\n", err.Message)
    })

    tb.Finally(func() {
        fmt.Println("Cleanup completed")
    })
}
```

## Advanced Usage

See `examples/main.go` for comprehensive examples including:

- Multiple exception types
- Nested try-catch blocks
- Return values from catch handlers
- Real-world scenarios

## Running Examples

```bash
# Quick demo
go run ./cmd/demo

# Full examples
go run ./examples

# Run tests
go test -v
```
