package main

import (
	"fmt"
	"os"

	"spysearch/cli" // Replace with your actual module path
)

func main() {
	if err := cli.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
