package lint_example

import (
	"errors"
	"fmt"
)

func foo() error {
	err := errors.New("test")
	return fmt.Errorf("error: %v", err)
}

func main() {
	fmt.Println(foo())
}
