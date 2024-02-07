package diff_example

import (
	"errors"
	"fmt"
)

func foo() error {
	err := errors.New("test")
	return errors.New(fmt.Sprintf("error: %v", err))
}

func main() {
	fmt.Println(foo())
}
