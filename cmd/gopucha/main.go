package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sjiamnocna/gopucha"
)

func main() {
	// Add flag for GUI mode
	useGUI := flag.Bool("gui", false, "Use GUI mode (requires display)")
	flag.Parse()
	
	// Default to maps directory if no argument provided
	mapFile := "maps/maps.txt"
	
	if flag.NArg() >= 1 {
		mapFile = flag.Arg(0)
	}
	
	// If the path doesn't exist and doesn't contain a directory separator,
	// try looking in the maps directory
	if _, err := os.Stat(mapFile); os.IsNotExist(err) && !filepath.IsAbs(mapFile) && filepath.Dir(mapFile) == "." {
		mapsPath := filepath.Join("maps", mapFile)
		if _, err := os.Stat(mapsPath); err == nil {
			mapFile = mapsPath
		}
	}
	
	var err error
	if *useGUI {
		// Run GUI game
		err = gopucha.RunGUIGame(mapFile)
	} else {
		// Run terminal game
		fmt.Printf("Loading map: %s\n", mapFile)
		err = gopucha.RunGame(mapFile)
	}
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
