package test_files

import "fmt"

func f() string {
	fmt.Println("")
	return ""
}

func main() {
	str := f()
	fmt.Println(str)
}
