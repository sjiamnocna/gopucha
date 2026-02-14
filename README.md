# gopucha

Golang Zoner Pampuch reimplemented to Golang with custom maps.

## Description

Gopucha is based on the Czech original Pampuch, where a semi-circle figure moves around eating dots while avoiding monsters (red squares). The game supports custom maps loaded from text files and runs in GUI mode.

## Features

- **GUI Mode**: Fyne-based interface
- **Custom Map Loading**: Load maps from TXT files
- **Map Generator**: Generate random coherent maps
- **Multiple Levels**: Separated by `---` in map files (all maps must have same dimensions)
- **GUI Features**:
  - Pre-game settings: adjust speed, select map
  - Arrow key controls
  - Zoom in/out with +/- keys or CTRL+Scroll
  - Visual graphics with colored blocks
- **4 Monsters**: Red squares that turn only when hitting walls and pick the shortest available path toward the Pampuch figure
- **Score Tracking**: Earn points for collecting dots
- **Progressive Difficulty**: Multiple levels with increasing challenge

## Map Format

Maps are defined in TXT files using the following characters:
- `O`, `o`, or `0`: Walls/masonry (blocks)
- `-`: Empty space with dots for the player to collect

**Important**: All levels in a single file must have the same dimensions (width x height). Different sizes will result in an error.

Multiple levels can be defined in a single file, separated by a line containing only `---`.

### Example Map

```
OOOOOOOOOOOOOOOOOOOOOOOO
O----------------------O
O--OOO-----OOO-----OOO-O
O----------------------O
O-----OOO--OOO--OOO----O
O----------------------O
OOOOOOOOOOOOOOOOOOOOOOOO
---
OOOOOOOOOOOOOOOOOOOOOOOO
O----------------------O
O--OOO--OOO--OOO--OOO--O
O----------------------O
O--OOO--OOO--OOO--OOO--O
OOOOOOOOOOOOOOOOOOOOOOOO
```

## Installation

### Dependencies
- **Go**: Install Go 1.26+ and ensure `go` is on your PATH. Builds are done directly with Go (no Docker).
- **System libs for GUI mode** (Linux):
  - Ubuntu/Debian: `sudo apt-get install libgl1-mesa-dev xorg-dev`
  - Fedora: `sudo dnf install mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel`
  - Arch: `sudo pacman -S mesa libxcursor libxrandr libxinerama libxi`

### Build (Makefile - preferred)
```bash
make build           # Standard build (GUI mode)
make build-optimized # Optimized build with size reduction
make test            # Run unit tests
```

### Run
```bash
make run             # Build and run with default map
./gopucha            # Run manually (default maps)
./gopucha maps/maps.txt
```

### Build (manual - fallback)
```bash
go build -o gopucha ./cmd/gopucha
```

### Build (manual)
```bash
go build -o gopucha ./cmd/gopucha
```

## Usage

### GUI
GUI is made with Fyne and is the default mode.

Run with default maps:
```bash
./gopucha
```

Run with a specific map file:
```bash
./gopucha maps/maps.txt
./gopucha simple.txt  # Automatically looks in maps/ directory
```

GUI mode features:
- Settings dialog before game starts
- Speed slider (50-500ms tick rate)
- Map file selector
- Visual graphics with color blocks
- Zoom controls

### Map Generator
Generate random maps:
```bash
go build -o bin/mapgen ./cmd/mapgen
./bin/mapgen -width 24 -height 10 -levels 3 -output custom_map.txt
```

Options:
- `-width`: Map width (must be even, minimum 10)
- `-height`: Map height (minimum 5)
- `-levels`: Number of levels to generate
- `-output`: Output file path

## Controls

### GUI Mode
- `Arrow Keys`: Move player
- `+/-`: Zoom in/out
- `ESC`: Quit game

## Gameplay

- Collect all dots to advance to the next level
- Avoid the monsters (red squares)
- Monsters only turn when they hit a wall, then choose the shortest available path toward the Pampuch figure
- Game ends when you collide with a monster
- Win by completing all levels

## Example Maps

Two example map files are included in the `maps/` directory:
- `maps/simple.txt`: A small simple map for testing
- `maps/maps.txt`: Three progressively challenging levels (24x10 each)

By default, the game loads `maps/maps.txt`. You can specify just the filename (e.g., `simple.txt`) and the game will automatically look in the `maps/` directory.

## Project Structure

```
gopucha/
├── cmd/
│   ├── gopucha/    # Main game executable
│   └── mapgen/     # Map generator tool
├── maps/           # Example map files
├── *.go            # Game logic (gopucha package)
└── Makefile        # Build automation
```

