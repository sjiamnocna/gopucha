package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gopucha <map-file>")
		fmt.Println("Example: gopucha maps.txt")
		os.Exit(1)
	}
	
	mapFile := os.Args[1]
	
	if err := RunGame(mapFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
