package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

func main() {
	width := flag.Int("width", 24, "Width of the map (must be even, minimum 10)")
	height := flag.Int("height", 10, "Height of the map (minimum 5)")
	output := flag.String("output", "generated_map.txt", "Output file path")
	levels := flag.Int("levels", 1, "Number of levels to generate")
	template := flag.String("t", "", "Template map file to use (overrides width/height)")

	flag.Parse()

	file, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// If template is provided, use it
	if *template != "" {
		templateMap, err := readTemplateMap(*template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading template: %v\n", err)
			os.Exit(1)
		}

		// Write the template map multiple times
		for i := 0; i < *levels; i++ {
			if i > 0 {
				fmt.Fprintln(file, "---")
			}
			for _, row := range templateMap {
				fmt.Fprintln(file, row)
			}
		}

		fmt.Printf("Generated %d level(s) from template %s in %s\n", *levels, *template, *output)
		return
	}

	// Validate dimensions for random generation
	if *width < 10 || *width%2 != 0 {
		fmt.Fprintf(os.Stderr, "Width must be at least 10 and even\n")
		os.Exit(1)
	}
	if *height < 5 {
		fmt.Fprintf(os.Stderr, "Height must be at least 5\n")
		os.Exit(1)
	}

	// Note: rand.Seed deprecated in Go 1.20+, package-level functions auto-seeded

	for i := 0; i < *levels; i++ {
		if i > 0 {
			fmt.Fprintln(file, "---")
		}
		generateMap(file, *width, *height)
	}

	fmt.Printf("Generated %d level(s) with dimensions %dx%d in %s\n", *levels, *width, *height, *output)
}

func readTemplateMap(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var templateMap []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Skip empty lines and metadata lines
		if line == "" {
			continue
		}

		// Stop at level separator
		if line == "---" {
			break
		}

		// Skip metadata lines (they don't start with O or -)
		if !strings.HasPrefix(line, "O") && !strings.HasPrefix(line, "-") {
			// This is a metadata line like "name:" or "speedModifier:"
			continue
		}

		// This is a map grid line
		templateMap = append(templateMap, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(templateMap) == 0 {
		return nil, fmt.Errorf("no map grid found in template file")
	}

	return templateMap, nil
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

	// Create more varied maze patterns with multiple algorithms
	createVariedMazePattern(grid, width, height)

	// Write the map to file
	for _, row := range grid {
		fmt.Fprintln(file, string(row))
	}
}

func createVariedMazePattern(grid [][]rune, width, height int) {
	// Mix multiple maze patterns for variety
	
	// Pattern 1: Random wall segments with varying widths and directions
	for segment := 0; segment < 5; segment++ {
		startX := 2 + rand.Intn(width-8)
		startY := 2 + rand.Intn(height-6)
		
		// Horizontal or vertical segment
		isHorizontal := rand.Float32() < 0.5
		segmentLength := 4 + rand.Intn(6)
		wallWidth := 1 + rand.Intn(2)
		
		if isHorizontal {
			for x := 0; x < segmentLength && startX+x < width-2; x++ {
				for wy := 0; wy < wallWidth && startY+wy < height-1; wy++ {
					if startY+wy > 0 && startY+wy < height-1 {
						grid[startY+wy][startX+x] = 'O'
					}
				}
			}
		} else {
			for y := 0; y < segmentLength && startY+y < height-2; y++ {
				for wx := 0; wx < wallWidth && startX+wx < width-1; wx++ {
					if startX+wx > 0 && startX+wx < width-1 {
						grid[startY+y][startX+wx] = 'O'
					}
				}
			}
		}
	}

	// Pattern 2: Staggered wall blocks for organic look
	for y := 2; y < height-3; y += 3 + rand.Intn(3) {
		for x := 2; x < width-3; x += 4 + rand.Intn(3) {
			if rand.Float32() < 0.6 {
				// Place irregular block
				blockH := 2 + rand.Intn(3)
				blockW := 2 + rand.Intn(2)
				for by := 0; by < blockH && y+by < height-1; by++ {
					for bx := 0; bx < blockW && x+bx < width-1; bx++ {
						grid[y+by][x+bx] = 'O'
					}
				}
			}
		}
	}

	// Pattern 3: Winding corridors - create paths that snake through
	for corridor := 0; corridor < 3; corridor++ {
		x := 3 + rand.Intn(width-6)
		y := 3 + rand.Intn(height-6)
		
		// Create a winding corridor
		for steps := 0; steps < 10+rand.Intn(8); steps++ {
			if x > 1 && x < width-2 && y > 1 && y < height-2 {
				// Randomly choose direction with bias toward forward
				direction := rand.Intn(100)
				if direction < 60 {
					x += rand.Intn(3) - 1 // Move right, left, or stay
				} else if direction < 80 {
					y += rand.Intn(3) - 1 // Move down slightly
				} else {
					x += 2 - rand.Intn(5) // Random jump
				}
				
				// Carve out small area
				if grid[y][x] == 'O' && steps%2 == 0 {
					grid[y][x] = '-'
					if x+1 < width-1 {
						grid[y][x+1] = '-'
					}
				}
			}
		}
	}

	// Pattern 4: Ensure connectivity with strategic corridors
	// Horizontal corridors at varied heights
	for i := 0; i < 2; i++ {
		midY := 3 + rand.Intn(height-6)
		for x := 2; x < width-2; x += 2 + rand.Intn(2) {
			if grid[midY][x] == 'O' && rand.Float32() < 0.4 {
				grid[midY][x] = '-'
			}
		}
	}

	// Vertical corridors at varied positions
	for i := 0; i < 2; i++ {
		midX := 3 + rand.Intn(width-8)
		for y := 2; y < height-2; y += 2 + rand.Intn(2) {
			if grid[y][midX] == 'O' && rand.Float32() < 0.4 {
				grid[y][midX] = '-'
			}
		}
	}

	// Pattern 5: Diagonal-like walls using staggered patterns
	for y := 2; y < height-3; y += 4 {
		offset := (y / 4) % 4
		for x := 2 + offset; x < width-2; x += 4 {
			if rand.Float32() < 0.5 && grid[y][x] == '-' {
				grid[y][x] = 'O'
				if x+1 < width-1 && grid[y][x+1] == '-' {
					grid[y][x+1] = 'O'
				}
			}
		}
	}
}
