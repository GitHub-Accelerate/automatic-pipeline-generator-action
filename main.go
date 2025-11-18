package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Printf("Number of arguments: %d\n", len(os.Args))
	fmt.Println("Arguments:", os.Args)
}
