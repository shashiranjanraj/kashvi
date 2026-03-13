package main

import (
	"fmt"
	"errors"
)

type MyErr struct {}

func main() {
	var err error = &MyErr{}
	fmt.Printf("%v\n", err)
	fmt.Printf("%s\n", err)
}
