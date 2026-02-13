# gopucha

Golang Zoner Pampuch reimplemented to Golang with custom maps.

## Description

Gopucha is a Pac-Man-like game where a semi-circle figure moves around eating dots while avoiding monsters (red squares). The game supports custom maps loaded from text files.

## Features

- Custom map loading from TXT files
- Multiple levels separated by `---` in map files
- Player character (displayed as `C`)
- 4 monsters (displayed as `M`) that move and turn at edges
- Dot collection system
- Score tracking
- Multiple levels

## Map Format

Maps are defined in TXT files using the following characters:
- `O`, `o`, or `0`: Walls/masonry (blocks)
- `-`: Empty space with dots for the player to collect

Multiple levels can be defined in a single file, separated by a line containing only `---`.

### Example Map

```
OOOOOOOOOO
O--------O
O--OO----O
O--------O
O----OO--O
O--------O
OOOOOOOOOO
---
OOOOOOOOOOOOOOOOOOOO
O------------------O
O--OOO-----OOO-----O
O------------------O
OOOOOOOOOOOOOOOOOOOO
```

## Installation

```bash
go build -o gopucha ./cmd/gopucha
```

## Usage

Run with default maps:
```bash
./gopucha
```

Run with a specific map file:
```bash
./gopucha maps/maps.txt
./gopucha simple.txt  # Automatically looks in maps/ directory
```

Or specify a full path:
```bash
./gopucha /path/to/custom/map.txt
```

## Controls

- `W`: Move up
- `S`: Move down
- `A`: Move left
- `D`: Move right
- `Q`: Quit game

Press Enter after each command.

## Gameplay

- Collect all dots to advance to the next level
- Avoid the monsters (red squares)
- Monsters can only turn when they reach edges or intersections
- Game ends when you collide with a monster
- Win by completing all levels

## Example Maps

Two example map files are included in the `maps/` directory:
- `maps/simple.txt`: A small simple map for testing
- `maps/maps.txt`: Three progressively challenging levels

By default, the game loads `maps/maps.txt`. You can specify just the filename (e.g., `simple.txt`) and the game will automatically look in the `maps/` directory.

