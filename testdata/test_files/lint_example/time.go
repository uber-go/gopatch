package lint_example

import (
	"fmt"
	"time"
)

func main() {
	startOfYear := time.Date(2021, 0o1, 0o1, 0, 0, 0, 0, time.UTC)
	result := time.Now().Sub(startOfYear)
	fmt.Println(result)
}
