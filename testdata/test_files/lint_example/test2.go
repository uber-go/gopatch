package lint_example

import (
	"errors"
	"fmt"
)

func boo() error {
	err := errors.New("test")
	return errors.New(fmt.Sprintf("error: %v", err))
}

func main() {
	fmt.Println(boo())
}
