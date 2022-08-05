package patch_examples

import (
	"errors"
	"fmt"
)

func boo() error {
	err := errors.New("test")
	return fmt.Errorf("error: %v", err)
}

func main() {
	fmt.Println(boo())
}
