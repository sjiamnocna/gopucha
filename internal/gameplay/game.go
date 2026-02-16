package gameplay

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/sjiamnocna/gopucha/internal/actors"
	"github.com/sjiamnocna/gopucha/internal/maps"
)

func NewGame(mapsList []maps.Map, disableMonsters bool) *Game {
	if len(mapsList) == 0 {
		return nil
	}

	g := &Game{
		Maps:            mapsList,
		CurrentLevel:    0,
		Score:           0,
		Lives:           4,
		DisableMonsters: disableMonsters,
	}

	g.LoadLevel(0)
	return g
}

func (g *Game) LoadLevel(level int) {
	if level >= len(g.Maps) {
		g.Won = true
		return
	}

	g.CurrentLevel = level
	g.CurrentMap = &g.Maps[level]
	g.CurrentSpeedModifier = g.CurrentMap.SpeedModifier

	g.placePlayer()
	// Remove dot at player's starting position
	g.CurrentMap.EatDot(g.Player.X, g.Player.Y)
	g.placeMonsters()
}

func (g *Game) placePlayer() {
	// Reset player to starting position with cleared input queue
	if g.CurrentMap.PlayerStart != nil {
		g.Player = actors.NewPlayer(g.CurrentMap.PlayerStart.X, g.CurrentMap.PlayerStart.Y)
		return
	}

	pos, ok := g.randomWalkable(nil)
	if ok {
		g.Player = actors.NewPlayer(pos.X, pos.Y)
		return
	}

	// Fallback
	g.Player = actors.NewPlayer(1, 1)
}

func (g *Game) placeMonsters() {
	if g.DisableMonsters {
		g.Monsters = nil
		return
	}

	numMonsters := g.CurrentMap.MonsterCount
	if numMonsters < 0 {
		numMonsters = 0
	}
	if numMonsters == 0 {
		g.Monsters = nil
		return
	}

	// Place monsters
	g.Monsters = []actors.Monster{}
	used := make(map[string]bool)
	used[fmt.Sprintf("%d,%d", g.Player.X, g.Player.Y)] = true
	distMap := distanceMapFrom(g.CurrentMap, g.Player.X, g.Player.Y)

	// Use explicit starts first
	startIdx := 0
	for i := 0; i < numMonsters; i++ {
		var x, y int
		found := false
		if startIdx < len(g.CurrentMap.MonsterStarts) {
			pos := g.CurrentMap.MonsterStarts[startIdx]
			startIdx++
			key := fmt.Sprintf("%d,%d", pos.X, pos.Y)
			if !g.CurrentMap.IsWall(pos.X, pos.Y) && !used[key] {
				x, y = pos.X, pos.Y
				used[key] = true
				found = true
			}
		}

		if !found {
			pos, ok := g.randomWalkableWithMinDistance(used, distMap, defaultMinMonsterDistance)
			if !ok {
				pos, ok = g.randomWalkable(used)
			}
			if ok {
				x, y = pos.X, pos.Y
				used[fmt.Sprintf("%d,%d", x, y)] = true
				found = true
			}
		}

		if !found {
			continue // Skip this monster if no valid position found
		}

		dir := actors.Direction(i % 4)
		g.Monsters = append(g.Monsters, *actors.NewMonster(x, y, dir))
	}
}

func (g *Game) randomWalkableWithMinDistance(exclude map[string]bool, distMap [][]int, minDist int) (maps.StartPos, bool) {
	positions := make([]maps.StartPos, 0)
	for y := 0; y < g.CurrentMap.Height; y++ {
		for x := 0; x < g.CurrentMap.Width; x++ {
			if g.CurrentMap.IsWall(x, y) {
				continue
			}
			if distMap[y][x] < minDist {
				continue
			}
			key := fmt.Sprintf("%d,%d", x, y)
			if exclude != nil && exclude[key] {
				continue
			}
			positions = append(positions, maps.StartPos{X: x, Y: y})
		}
	}

	if len(positions) == 0 {
		return maps.StartPos{}, false
	}

	idx := rand.Intn(len(positions))
	return positions[idx], true
}

func distanceMapFrom(m *maps.Map, startX, startY int) [][]int {
	dist := make([][]int, m.Height)
	for y := 0; y < m.Height; y++ {
		dist[y] = make([]int, m.Width)
		for x := 0; x < m.Width; x++ {
			dist[y][x] = -1
		}
	}

	if startX < 0 || startY < 0 || startX >= m.Width || startY >= m.Height {
		return dist
	}
	if m.IsWall(startX, startY) {
		return dist
	}

	queueX := []int{startX}
	queueY := []int{startY}
	dist[startY][startX] = 0

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
			if m.IsWall(nx, ny) || dist[ny][nx] != -1 {
				continue
			}
			dist[ny][nx] = dist[y][x] + 1
			queueX = append(queueX, nx)
			queueY = append(queueY, ny)
		}
	}

	return dist
}

func (g *Game) randomWalkable(exclude map[string]bool) (maps.StartPos, bool) {
	positions := make([]maps.StartPos, 0)
	for y := 0; y < g.CurrentMap.Height; y++ {
		for x := 0; x < g.CurrentMap.Width; x++ {
			if g.CurrentMap.IsWall(x, y) {
				continue
			}
			key := fmt.Sprintf("%d,%d", x, y)
			if exclude != nil && exclude[key] {
				continue
			}
			positions = append(positions, maps.StartPos{X: x, Y: y})
		}
	}

	if len(positions) == 0 {
		return maps.StartPos{}, false
	}

	idx := rand.Intn(len(positions))
	return positions[idx], true
}

func (g *Game) Update() {
	if g.GameOver || g.Won {
		return
	}

	// Clear last-tick flags so UI doesn't stay in death/pause state.
	g.LifeLost = false
	g.BustPaused = false

	// Pause briefly after a bust so the collision is visible.
	if g.pendingRespawn {
		if time.Now().Before(g.bustPauseUntil) {
			g.BustPaused = true
			return
		}
		g.pendingRespawn = false
		g.placePlayer()
		g.placeMonsters()
	}

	// Store monster positions before they move
	oldMonsterPos := make([][2]int, len(g.Monsters))
	for i := range g.Monsters {
		oldMonsterPos[i] = [2]int{g.Monsters[i].X, g.Monsters[i].Y}
	}

	// Store player position before moving
	oldPlayerX, oldPlayerY := g.Player.X, g.Player.Y

	// Move player
	g.Player.Move(g.CurrentMap)

	// Check if player ate a dot
	if g.CurrentMap.HasDot(g.Player.X, g.Player.Y) {
		g.CurrentMap.EatDot(g.Player.X, g.Player.Y)
		baseScore := 10
		adjustedScore := int(float64(baseScore) * g.CurrentSpeedModifier)
		g.Score += adjustedScore
		g.DotEaten = true
	} else {
		g.DotEaten = false
	}

	// Move monsters
	for i := range g.Monsters {
		g.Monsters[i].Move(g.CurrentMap, g.Player.X, g.Player.Y, g.Monsters)
	}

	// Check collision with monsters (including position swaps)
	for i, monster := range g.Monsters {
		// Same cell collision
		if g.Player.X == monster.X && g.Player.Y == monster.Y {
			g.Lives--
			if g.Lives <= 0 {
				g.GameOver = true
				g.LifeLost = false
			} else {
				g.LifeLost = true
				g.BustPaused = true
				g.pendingRespawn = true
				g.bustPauseUntil = time.Now().Add(1 * time.Second)
			}
			return
		}

		// Swap collision (player and monster passed through each other)
		if g.Player.X == oldMonsterPos[i][0] && g.Player.Y == oldMonsterPos[i][1] &&
			monster.X == oldPlayerX && monster.Y == oldPlayerY {
			// Snap the monster onto the player's cell so the bust is visible.
			g.Monsters[i].X = g.Player.X
			g.Monsters[i].Y = g.Player.Y
			g.Lives--
			if g.Lives <= 0 {
				g.GameOver = true
				g.LifeLost = false
			} else {
				g.LifeLost = true
				g.BustPaused = true
				g.pendingRespawn = true
				g.bustPauseUntil = time.Now().Add(1 * time.Second)
			}
			return
		}
	}

	// Check if all dots are eaten
	if g.CurrentMap.CountDots() == 0 {
		// Mark level as completed, GUI will handle pause and advance
		g.LevelCompleted = true
	}
}

func (g *Game) Render() {
	g.CurrentMap.Render(g.Player.X, g.Player.Y, g.Monsters)
	fmt.Printf("\nLevel: %d | Score: %d | Dots: %d\n", g.CurrentLevel+1, g.Score, g.CurrentMap.CountDots())
	fmt.Println("Controls: W=Up, S=Down, A=Left, D=Right, Q=Quit")

	if g.GameOver {
		fmt.Println("\n\033[31mGAME OVER!\033[0m")
	}
	if g.Won {
		fmt.Println("\n\033[32mYOU WON! All levels completed!\033[0m")
	}
}

func (g *Game) HandleInput(input string) {
	if len(input) == 0 {
		return
	}

	switch input[0] {
	case 'w', 'W':
		g.Player.SetDirection(actors.Up)
	case 's', 'S':
		g.Player.SetDirection(actors.Down)
	case 'a', 'A':
		g.Player.SetDirection(actors.Left)
	case 'd', 'D':
		g.Player.SetDirection(actors.Right)
	case 'q', 'Q':
		g.GameOver = true
	}
}

func RunGame(mapFile string) error {
	mapsList, err := maps.LoadMapsFromFile(mapFile)
	if err != nil {
		return fmt.Errorf("failed to load maps: %v", err)
	}

	if len(mapsList) == 0 {
		return fmt.Errorf("no maps found in file")
	}

	game := NewGame(mapsList, false)
	if game == nil {
		return fmt.Errorf("failed to create game")
	}

	// Game loop
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// Input channel
	inputChan := make(chan string, 10)
	go func() {
		for {
			var input string
			fmt.Scanln(&input)
			inputChan <- input
		}
	}()

	for !game.GameOver && !game.Won {
		select {
		case <-ticker.C:
			game.Update()
			game.Render()
		case input := <-inputChan:
			game.HandleInput(input)
		}
	}

	// Final render
	game.Render()

	// Wait a bit before exiting
	time.Sleep(2 * time.Second)

	return nil
}
