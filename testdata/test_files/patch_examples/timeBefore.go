package patch_examples

import (
	"fmt"
	"time"
)

func main() {
	startOfYear := time.Date(2021, 01, 01, 0, 0, 0, 0, time.UTC)
	result := time.Now().Sub(startOfYear)
	fmt.Println(result)
}
