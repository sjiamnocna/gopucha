package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sjiamnocna/gopucha/internal/gopucha"
)

func main() {
	noMonsters := flag.Bool("no-monsters", false, "disable monster spawning (debug)")
	mapFlag := flag.String("map", "maps/maps.txt", "path to map file")
	flag.Parse()

	// Default to maps directory if no argument provided
	mapFile := *mapFlag
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

	// Run GUI game only
	err := gopucha.RunGUIGame(mapFile, *noMonsters)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
