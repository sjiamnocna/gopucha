package main

import (
	"bufio"
	"fmt"
	"os"
)

type Cell int

const (
	Empty Cell = iota
	Wall
	Dot
)

type Map struct {
	Width  int
	Height int
	Cells  [][]Cell
}

func LoadMapsFromFile(filename string) ([]Map, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var maps []Map
	var currentLines []string
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Check for level separator
		if line == "---" {
			if len(currentLines) > 0 {
				m := parseMap(currentLines)
				maps = append(maps, m)
				currentLines = nil
			}
			continue
		}
		
		if line != "" {
			currentLines = append(currentLines, line)
		}
	}
	
	// Don't forget the last map
	if len(currentLines) > 0 {
		m := parseMap(currentLines)
		maps = append(maps, m)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	return maps, nil
}

func parseMap(lines []string) Map {
	if len(lines) == 0 {
		return Map{}
	}
	
	height := len(lines)
	width := 0
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}
	
	cells := make([][]Cell, height)
	for i := range cells {
		cells[i] = make([]Cell, width)
	}
	
	for y, line := range lines {
		for x, ch := range line {
			switch ch {
			case 'O', 'o', '0':
				cells[y][x] = Wall
			case '-':
				cells[y][x] = Dot
			default:
				cells[y][x] = Empty
			}
		}
	}
	
	return Map{
		Width:  width,
		Height: height,
		Cells:  cells,
	}
}

func (m *Map) IsWall(x, y int) bool {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return true
	}
	return m.Cells[y][x] == Wall
}

func (m *Map) HasDot(x, y int) bool {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return false
	}
	return m.Cells[y][x] == Dot
}

func (m *Map) EatDot(x, y int) {
	if x >= 0 && y >= 0 && x < m.Width && y < m.Height {
		m.Cells[y][x] = Empty
	}
}

func (m *Map) CountDots() int {
	count := 0
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if m.Cells[y][x] == Dot {
				count++
			}
		}
	}
	return count
}

func (m *Map) Render(playerX, playerY int, monsters []Monster) {
	fmt.Print("\033[H\033[2J") // Clear screen
	
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			// Check if player is at this position
			if x == playerX && y == playerY {
				fmt.Print("\033[33mC\033[0m") // Yellow C (Pac-Man)
				continue
			}
			
			// Check if any monster is at this position
			isMonster := false
			for _, monster := range monsters {
				if monster.X == x && monster.Y == y {
					fmt.Print("\033[31mM\033[0m") // Red M (Monster)
					isMonster = true
					break
				}
			}
			if isMonster {
				continue
			}
			
			// Render the cell
			switch m.Cells[y][x] {
			case Wall:
				fmt.Print("\033[34mO\033[0m") // Blue O (Wall)
			case Dot:
				fmt.Print("\033[37m·\033[0m") // White dot
			case Empty:
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}
}
