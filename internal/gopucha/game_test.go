package gopucha

import (
	"testing"
)

func TestGameSpeedModifierScoring(t *testing.T) {
	tests := []struct {
		name             string
		speedModifier    float64
		baseDots         int
		expectedMinScore int
	}{
		{
			name:             "1.0x multiplier (normal speed)",
			speedModifier:    1.0,
			baseDots:         8,
			expectedMinScore: 80, // 8 dots * 10 * 1.0
		},
		{
			name:             "1.5x multiplier (faster)",
			speedModifier:    1.5,
			baseDots:         8,
			expectedMinScore: 120, // 8 dots * 10 * 1.5
		},
		{
			name:             "0.5x multiplier (slower)",
			speedModifier:    0.5,
			baseDots:         8,
			expectedMinScore: 40, // 8 dots * 10 * 0.5
		},
		{
			name:             "2.0x multiplier (double speed)",
			speedModifier:    2.0,
			baseDots:         8,
			expectedMinScore: 160, // 8 dots * 10 * 2.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test map with speedModifier
			mapLines := []string{
				"name: Speed Test",
				"speedModifier: " + formatFloat(tt.speedModifier),
				"OOOOO",
				"O---O",
				"O-O-O",
				"O---O",
				"OOOOO",
			}

			m, err := parseMap(mapLines)
			if err != nil {
				t.Fatalf("Failed to parse map: %v", err)
			}

			if m.SpeedModifier != tt.speedModifier {
				t.Errorf("Map SpeedModifier = %v, want %v", m.SpeedModifier, tt.speedModifier)
			}

			// Create game with this map
			game := NewGame([]Map{m}, false)
			if game == nil {
				t.Fatalf("Failed to create game")
			}

			if game.CurrentSpeedModifier != tt.speedModifier {
				t.Errorf("Game CurrentSpeedModifier = %v, want %v", 
					game.CurrentSpeedModifier, tt.speedModifier)
			}

			// Verify initial score is 0
			if game.Score != 0 {
				t.Errorf("Initial score = %d, want 0", game.Score)
			}
		})
	}
}

func TestGameLoadLevelSetsSpeedModifier(t *testing.T) {
	maps := []Map{
		{
			Width:         5,
			Height:        5,
			Name:          "Slow Map",
			SpeedModifier: 0.5,
			Cells: [][]Cell{
				{Wall, Wall, Wall, Wall, Wall},
				{Wall, Empty, Dot, Empty, Wall},
				{Wall, Dot, Empty, Dot, Wall},
				{Wall, Empty, Dot, Empty, Wall},
				{Wall, Wall, Wall, Wall, Wall},
			},
		},
		{
			Width:         5,
			Height:        5,
			Name:          "Fast Map",
			SpeedModifier: 2.0,
			Cells: [][]Cell{
				{Wall, Wall, Wall, Wall, Wall},
				{Wall, Empty, Dot, Empty, Wall},
				{Wall, Dot, Empty, Dot, Wall},
				{Wall, Empty, Dot, Empty, Wall},
				{Wall, Wall, Wall, Wall, Wall},
			},
		},
	}

	game := NewGame(maps, true)
	if game == nil {
		t.Fatalf("Failed to create game")
	}

	// Check first level
	if game.CurrentSpeedModifier != 0.5 {
		t.Errorf("Level 0: CurrentSpeedModifier = %v, want 0.5", game.CurrentSpeedModifier)
	}

	// Load second level
	game.loadLevel(1)
	if game.CurrentSpeedModifier != 2.0 {
		t.Errorf("Level 1: CurrentSpeedModifier = %v, want 2.0", game.CurrentSpeedModifier)
	}
}

func TestDotScoringWithSpeedModifier(t *testing.T) {
	// Create a simple test map with multiple dots
	mapLines := []string{
		"speedModifier: 1.5",
		"OOOOO",
		"O---O",
		"O-O-O",
		"OOOOO",
	}

	m, err := parseMap(mapLines)
	if err != nil {
		t.Fatalf("Failed to parse map: %v", err)
	}

	game := NewGame([]Map{m}, true) // disable monsters for simpler test
	if game == nil {
		t.Fatalf("Failed to create game")
	}

	initialScore := game.Score
	dotCountBefore := game.CurrentMap.CountDots()

	// Player starts at (1,1) and dot there is removed, so move to adjacent dot at (2,1)
	game.Player.X = 2
	game.Player.Y = 1
	game.Update()

	// Check score was increased by 10 * 1.5 = 15
	expectedScore := initialScore + int(float64(10)*1.5)
	if game.Score != expectedScore {
		t.Errorf("After eating dot: Score = %d, want %d", game.Score, expectedScore)
	}

	dotCountAfter := game.CurrentMap.CountDots()
	if dotCountBefore != dotCountAfter+1 {
		t.Errorf("Dot count should decrease by 1, was %d now %d", 
			dotCountBefore, dotCountAfter)
	}
}

// Helper function to format float for test strings
func formatFloat(f float64) string {
	switch f {
	case 0.5:
		return "0.5"
	case 1.0:
		return "1.0"
	case 1.5:
		return "1.5"
	case 2.0:
		return "2.0"
	default:
		return "1.0"
	}
}
