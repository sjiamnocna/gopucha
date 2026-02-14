package gopucha

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Cell int

const (
	Empty Cell = iota
	Wall
	Dot
)

type Map struct {
	Width        int
	Height       int
	Cells        [][]Cell
	Name         string
	Material     string
	MonsterCount int
	SpeedModifier float64
	PlayerStart  *StartPos
	MonsterStarts []StartPos
}

type StartPos struct {
	X int
	Y int
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
				m, err := parseMap(currentLines)
				if err != nil {
					return nil, err
				}
				if err := validateMap(&m); err != nil {
					return nil, err
				}
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
		m, err := parseMap(currentLines)
		if err != nil {
			return nil, err
		}
		if err := validateMap(&m); err != nil {
			return nil, err
		}
		maps = append(maps, m)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	// Validate that all maps have the same dimensions
	if len(maps) > 1 {
		firstWidth := maps[0].Width
		firstHeight := maps[0].Height
		for i, m := range maps[1:] {
			if m.Width != firstWidth || m.Height != firstHeight {
				return nil, fmt.Errorf("map %d has different dimensions (%dx%d) than first map (%dx%d). All maps in a file must have the same dimensions", 
					i+2, m.Width, m.Height, firstWidth, firstHeight)
			}
		}
	}
	
	return maps, nil
}

func parseMapMetaLine(line string) (key, value string) {
	trimmed := strings.TrimSpace(line)
	// Try "key: value" format
	if idx := strings.Index(trimmed, ":"); idx != -1 {
		key = strings.TrimSpace(trimmed[:idx])
		value = strings.TrimSpace(trimmed[idx+1:])
		return
	}
	// Try "key=value" format
	if idx := strings.Index(trimmed, "="); idx != -1 {
		key = strings.TrimSpace(trimmed[:idx])
		value = strings.TrimSpace(trimmed[idx+1:])
		return
	}
	return "", ""
}

func parseMap(lines []string) (Map, error) {
	if len(lines) == 0 {
		return Map{}, fmt.Errorf("empty map")
	}

	name := ""
	material := ""
	monsterCount := 1
	monsterCountSet := false
	speedModifier := 1.0
	var playerStart *StartPos
	var monsterStarts []StartPos
	var gridLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		
		// Check if this looks like a metadata line
		if strings.Contains(trimmed, ":") || strings.Contains(trimmed, "=") {
			key, value := parseMapMetaLine(trimmed)
			key = strings.ToLower(key)
			
			switch key {
			case "name":
				name = value
				continue
			case "material":
				material = value
				continue
			case "playerstart":
				pos, err := parseStartPair(value)
				if err != nil {
					return Map{}, fmt.Errorf("invalid playerStart: %q", value)
				}
				playerStart = &pos
				continue
			case "monsterstart", "monsterstarts":
				list, err := parseStartList(value)
				if err != nil {
					return Map{}, fmt.Errorf("invalid monsterStarts: %q", value)
				}
				monsterStarts = append(monsterStarts, list...)
				continue
			case "monsters":
				count, err := strconv.Atoi(value)
				if err != nil || count < 0 {
					return Map{}, fmt.Errorf("invalid monsters count: %q", value)
				}
				monsterCount = count
				monsterCountSet = true
				continue
			case "speedmodifier":
				mod, err := strconv.ParseFloat(value, 64)
				if err != nil || mod < 0.5 || mod > 2.0 {
					return Map{}, fmt.Errorf("invalid speedModifier: %q (must be between 0.5 and 2.0)", value)
				}
				speedModifier = mod
				continue
			}
		}
		
		// If we get here, treat it as grid data
		gridLines = append(gridLines, trimmed)
	}

	if len(gridLines) == 0 {
		return Map{}, fmt.Errorf("map has no grid data")
	}
	if !monsterCountSet {
		monsterCount = 1
	}

	height := len(gridLines)
	width := 0
	for _, line := range gridLines {
		if len(line) > width {
			width = len(line)
		}
	}

	cells := make([][]Cell, height)
	for i := range cells {
		cells[i] = make([]Cell, width)
	}

	for y, line := range gridLines {
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
		Width:         width,
		Height:        height,
		Cells:         cells,
		Name:          name,
		Material:      material,
		MonsterCount:  monsterCount,
		SpeedModifier: speedModifier,
		PlayerStart:   playerStart,
		MonsterStarts: monsterStarts,
	}, nil
}

func parseStartPair(value string) (StartPos, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return StartPos{}, fmt.Errorf("expected x,y")
	}
	xStr := strings.TrimSpace(parts[0])
	yStr := strings.TrimSpace(parts[1])
	x, err := strconv.Atoi(xStr)
	if err != nil {
		return StartPos{}, err
	}
	y, err := strconv.Atoi(yStr)
	if err != nil {
		return StartPos{}, err
	}
	return StartPos{X: x, Y: y}, nil
}

func parseStartList(value string) ([]StartPos, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ";")
	positions := make([]StartPos, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pos, err := parseStartPair(part)
		if err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}
	return positions, nil
}

func validateMap(m *Map) error {
	startX, startY := -1, -1
	if m.PlayerStart != nil {
		if m.PlayerStart.X < 0 || m.PlayerStart.Y < 0 || m.PlayerStart.X >= m.Width || m.PlayerStart.Y >= m.Height {
			return fmt.Errorf("playerStart is out of bounds (%d,%d)", m.PlayerStart.X, m.PlayerStart.Y)
		}
		if m.Cells[m.PlayerStart.Y][m.PlayerStart.X] == Wall {
			return fmt.Errorf("playerStart is on a wall (%d,%d)", m.PlayerStart.X, m.PlayerStart.Y)
		}
		startX, startY = m.PlayerStart.X, m.PlayerStart.Y
	} else {
		for y := 0; y < m.Height && startX == -1; y++ {
			for x := 0; x < m.Width; x++ {
				if m.Cells[y][x] != Wall {
					startX, startY = x, y
					break
				}
			}
		}
		if startX == -1 {
			return fmt.Errorf("map has no walkable cells")
		}
	}

	used := make(map[string]bool)
	if m.PlayerStart != nil {
		key := fmt.Sprintf("%d,%d", m.PlayerStart.X, m.PlayerStart.Y)
		used[key] = true
	}
	for _, pos := range m.MonsterStarts {
		if pos.X < 0 || pos.Y < 0 || pos.X >= m.Width || pos.Y >= m.Height {
			return fmt.Errorf("monsterStart is out of bounds (%d,%d)", pos.X, pos.Y)
		}
		if m.Cells[pos.Y][pos.X] == Wall {
			return fmt.Errorf("monsterStart is on a wall (%d,%d)", pos.X, pos.Y)
		}
		key := fmt.Sprintf("%d,%d", pos.X, pos.Y)
		if used[key] {
			return fmt.Errorf("duplicate start position (%d,%d)", pos.X, pos.Y)
		}
		used[key] = true
	}


	dotCount := 0
	dotStartX, dotStartY := -1, -1
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if m.Cells[y][x] == Dot {
				dotCount++
				if dotStartX == -1 {
					dotStartX, dotStartY = x, y
				}
			}
		}
	}
	if dotCount == 0 {
		return nil
	}

	reachable := bfsReachable(m, startX, startY)
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if m.Cells[y][x] == Dot && !reachable[y][x] {
				return fmt.Errorf("map has unreachable dot at (%d,%d)", x, y)
			}
		}
	}

	dotReachable := bfsReachable(m, dotStartX, dotStartY)
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if m.Cells[y][x] == Dot && !dotReachable[y][x] {
				return fmt.Errorf("map has separated dots; dot at (%d,%d) is disconnected", x, y)
			}
		}
	}

	return nil
}

func bfsReachable(m *Map, startX, startY int) [][]bool {
	reachable := make([][]bool, m.Height)
	for i := range reachable {
		reachable[i] = make([]bool, m.Width)
	}

	queueX := []int{startX}
	queueY := []int{startY}
	reachable[startY][startX] = true

	for len(queueX) > 0 {
		x := queueX[0]
		y := queueY[0]
		queueX = queueX[1:]
		queueY = queueY[1:]

		neighbors := [][2]int{{x + 1, y}, {x - 1, y}, {x, y + 1}, {x, y - 1}}
		for _, n := range neighbors {
			nx, ny := n[0], n[1]
			if nx < 0 || ny < 0 || nx >= m.Width || ny >= m.Height {
				continue
			}
			if m.Cells[ny][nx] == Wall || reachable[ny][nx] {
				continue
			}
			reachable[ny][nx] = true
			queueX = append(queueX, nx)
			queueY = append(queueY, ny)
		}
	}

	return reachable
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
