package gopucha

import (
	"fmt"
	"time"
)

type Game struct {
	CurrentMap    *Map
	Maps          []Map
	CurrentLevel  int
	Player        *Player
	Monsters      []Monster
	GameOver      bool
	Won           bool
	Score         int
	Lives         int
	LifeLost      bool
	DisableMonsters bool
	DotEaten      bool
	CurrentSpeedModifier float64
	LevelCompleted bool
}

func NewGame(maps []Map, disableMonsters bool) *Game {
	if len(maps) == 0 {
		return nil
	}
	
	g := &Game{
		Maps:         maps,
		CurrentLevel: 0,
		Score:        0,
		Lives:        3,
		DisableMonsters: disableMonsters,
	}
	
	g.loadLevel(0)
	return g
}

func (g *Game) loadLevel(level int) {
	if level >= len(g.Maps) {
		g.Won = true
		return
	}
	
	g.CurrentLevel = level
	g.CurrentMap = &g.Maps[level]
	g.CurrentSpeedModifier = g.CurrentMap.SpeedModifier
	
	// Place player at first non-wall position
	g.Player = NewPlayer(1, 1)
	for y := 0; y < g.CurrentMap.Height; y++ {
		for x := 0; x < g.CurrentMap.Width; x++ {
			if !g.CurrentMap.IsWall(x, y) {
				g.Player = NewPlayer(x, y)
				goto PlayerPlaced
			}
		}
	}
PlayerPlaced:
	// Remove dot at player's starting position
	g.CurrentMap.EatDot(g.Player.X, g.Player.Y)
	if g.DisableMonsters {
		g.Monsters = nil
		return
	}

	// Use map-defined monster count
	numMonsters := g.CurrentMap.MonsterCount
	if numMonsters < 0 {
		numMonsters = 0
	}
	if numMonsters == 0 {
		g.Monsters = nil
		return
	}
	
	// Place monsters
	g.Monsters = []Monster{}
	// Corner positions: top-left, top-right, bottom-left, bottom-right
	cornerPositions := [][2]int{
		{1, 1},                                    // Top-left
		{g.CurrentMap.Width - 2, 1},              // Top-right
		{1, g.CurrentMap.Height - 2},             // Bottom-left
		{g.CurrentMap.Width - 2, g.CurrentMap.Height - 2}, // Bottom-right
	}
	
	usedCorners := make(map[int]bool)
	
	for i := 0; i < numMonsters; i++ {
		var x, y int
		found := false
		
		// Try to find an unblocked corner
		for j := 0; j < len(cornerPositions); j++ {
			cornerIdx := (i + j) % len(cornerPositions)
			if usedCorners[cornerIdx] {
				continue
			}
			
			pos := cornerPositions[cornerIdx]
			cx, cy := pos[0], pos[1]
			
			// Adjust if out of bounds
			if cx >= g.CurrentMap.Width {
				cx = g.CurrentMap.Width - 1
			}
			if cy >= g.CurrentMap.Height {
				cy = g.CurrentMap.Height - 1
			}
			
			// Check if corner is valid
			if !g.CurrentMap.IsWall(cx, cy) && (cx != g.Player.X || cy != g.Player.Y) {
				x, y = cx, cy
				usedCorners[cornerIdx] = true
				found = true
				break
			}
		}
		
		// If no corner available, search for any valid position
		if !found {
			for dy := 0; dy < g.CurrentMap.Height && !found; dy++ {
				for dx := 0; dx < g.CurrentMap.Width && !found; dx++ {
					if !g.CurrentMap.IsWall(dx, dy) && (dx != g.Player.X || dy != g.Player.Y) {
						x, y = dx, dy
						found = true
					}
				}
			}
		}
		
		if !found {
			continue // Skip this monster if no valid position found
		}
		
		dir := Direction(i % 4)
		g.Monsters = append(g.Monsters, *NewMonster(x, y, dir))
	}
}

func (g *Game) Update() {
	if g.GameOver || g.Won {
		return
	}
	
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
	
	// Check collision with monsters
	for _, monster := range g.Monsters {
		if g.Player.X == monster.X && g.Player.Y == monster.Y {
			g.Lives--
			g.LifeLost = true
			if g.Lives <= 0 {
				g.GameOver = true
			} else {
				// Reset player position on this level
				g.Player = NewPlayer(1, 1)
				for y := 0; y < g.CurrentMap.Height; y++ {
					for x := 0; x < g.CurrentMap.Width; x++ {
						if !g.CurrentMap.IsWall(x, y) {
							g.Player = NewPlayer(x, y)
							goto PlayerReset
						}
					}
				}
			PlayerReset:
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
		g.Player.SetDirection(Up)
	case 's', 'S':
		g.Player.SetDirection(Down)
	case 'a', 'A':
		g.Player.SetDirection(Left)
	case 'd', 'D':
		g.Player.SetDirection(Right)
	case 'q', 'Q':
		g.GameOver = true
	}
}

func RunGame(mapFile string) error {
	maps, err := LoadMapsFromFile(mapFile)
	if err != nil {
		return fmt.Errorf("failed to load maps: %v", err)
	}
	
	if len(maps) == 0 {
		return fmt.Errorf("no maps found in file")
	}
	
	game := NewGame(maps, false)
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
