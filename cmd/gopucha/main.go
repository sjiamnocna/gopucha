package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sjiamnocna/gopucha/internal/gopucha"
)

func main() {
	// Default to maps directory if no argument provided
	mapFile := "maps/maps.txt"

	if len(os.Args) >= 2 {
		mapFile = os.Args[1]
	}

	// If the path doesn't exist and doesn't contain a directory separator,
	// try looking in the maps directory
	if _, err := os.Stat(mapFile); os.IsNotExist(err) && !filepath.IsAbs(mapFile) && filepath.Dir(mapFile) == "." {
		mapsPath := filepath.Join("maps", mapFile)
		if _, err := os.Stat(mapsPath); err == nil {
			mapFile = mapsPath
		}
	}

	// Run GUI game only
	err := gopucha.RunGUIGame(mapFile)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
