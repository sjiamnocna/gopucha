package main

import (
"flag"
"fmt"
"math/rand"
"os"
)

func main() {
width := flag.Int("width", 24, "Width of the map (must be even, minimum 10)")
height := flag.Int("height", 10, "Height of the map (minimum 5)")
output := flag.String("output", "generated_map.txt", "Output file path")
levels := flag.Int("levels", 1, "Number of levels to generate")

flag.Parse()

// Validate dimensions
if *width < 10 || *width%2 != 0 {
fmt.Fprintf(os.Stderr, "Width must be at least 10 and even\n")
os.Exit(1)
}
if *height < 5 {
fmt.Fprintf(os.Stderr, "Height must be at least 5\n")
os.Exit(1)
}

// Note: rand.Seed deprecated in Go 1.20+, package-level functions auto-seeded

file, err := os.Create(*output)
if err != nil {
fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
os.Exit(1)
}
defer file.Close()

for i := 0; i < *levels; i++ {
if i > 0 {
fmt.Fprintln(file, "---")
}
generateMap(file, *width, *height)
}

fmt.Printf("Generated %d level(s) with dimensions %dx%d in %s\n", *levels, *width, *height, *output)
}

func generateMap(file *os.File, width, height int) {
// Create the map grid
grid := make([][]rune, height)
for i := range grid {
grid[i] = make([]rune, width)
for j := range grid[i] {
grid[i][j] = '-'
}
}

// Add border walls
for x := 0; x < width; x++ {
grid[0][x] = 'O'
grid[height-1][x] = 'O'
}
for y := 0; y < height; y++ {
grid[y][0] = 'O'
grid[y][width-1] = 'O'
}

// Add internal wall blocks with corridors
// Create a pattern of wall blocks with guaranteed paths
blockSize := 3
for y := 2; y < height-2; y += 4 {
for x := 2; x < width-2; x += 6 {
// Randomly decide whether to place a block here (70% chance)
if rand.Float32() < 0.7 && x+blockSize < width-1 {
// Place a small wall block
for by := 0; by < blockSize && y+by < height-1; by++ {
for bx := 0; bx < blockSize && x+bx < width-1; bx++ {
grid[y+by][x+bx] = 'O'
}
}
}
}
}

// Ensure there's always a path by clearing some strategic corridors
// Horizontal corridor in the middle
midY := height / 2
for x := 1; x < width-1; x++ {
if grid[midY][x] == 'O' && rand.Float32() < 0.3 {
grid[midY][x] = '-'
}
}

// Vertical corridors
for y := 1; y < height-1; y++ {
// Left corridor
if grid[y][2] == 'O' && rand.Float32() < 0.3 {
grid[y][2] = '-'
}
// Right corridor
if grid[y][width-3] == 'O' && rand.Float32() < 0.3 {
grid[y][width-3] = '-'
}
}

// Write the map to file
for _, row := range grid {
fmt.Fprintln(file, string(row))
}
}
