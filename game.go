package main

import (
	"fmt"
	"time"
)

type Game struct {
	CurrentMap   *Map
	Maps         []Map
	CurrentLevel int
	Player       *Player
	Monsters     []Monster
	GameOver     bool
	Won          bool
	Score        int
}

func NewGame(maps []Map) *Game {
	if len(maps) == 0 {
		return nil
	}
	
	g := &Game{
		Maps:         maps,
		CurrentLevel: 0,
		Score:        0,
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
	
	// Place 4 monsters
	g.Monsters = []Monster{}
	monsterPositions := [][2]int{
		{g.CurrentMap.Width - 2, 1},
		{1, g.CurrentMap.Height - 2},
		{g.CurrentMap.Width - 2, g.CurrentMap.Height - 2},
		{g.CurrentMap.Width / 2, g.CurrentMap.Height / 2},
	}
	
	for i, pos := range monsterPositions {
		x, y := pos[0], pos[1]
		if x >= g.CurrentMap.Width {
			x = g.CurrentMap.Width - 1
		}
		if y >= g.CurrentMap.Height {
			y = g.CurrentMap.Height - 1
		}
		
		// Check if this is a valid position (not a wall and not occupied by player)
		if g.CurrentMap.IsWall(x, y) || (x == g.Player.X && y == g.Player.Y) {
			// Search for a valid position
			found := false
			for dy := 0; dy < g.CurrentMap.Height && !found; dy++ {
				for dx := 0; dx < g.CurrentMap.Width && !found; dx++ {
					if !g.CurrentMap.IsWall(dx, dy) && (dx != g.Player.X || dy != g.Player.Y) {
						x, y = dx, dy
						found = true
					}
				}
			}
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
		g.Score += 10
	}
	
	// Move monsters
	for i := range g.Monsters {
		g.Monsters[i].Move(g.CurrentMap)
	}
	
	// Check collision with monsters
	for _, monster := range g.Monsters {
		if g.Player.X == monster.X && g.Player.Y == monster.Y {
			g.GameOver = true
			return
		}
	}
	
	// Check if all dots are eaten
	if g.CurrentMap.CountDots() == 0 {
		// Load next level
		g.loadLevel(g.CurrentLevel + 1)
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
	
	game := NewGame(maps)
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
