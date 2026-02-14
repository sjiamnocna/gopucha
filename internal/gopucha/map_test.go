package gopucha

import (
	"os"
	"testing"
)

func TestLoadMapsFromFile(t *testing.T) {
	// Create a temporary test map file with same-sized maps
	content := `OOOOOO
O----O
O----O
OOOOOO
---
OOOOOO
O----O
O----O
OOOOOO
`
	tmpFile, err := os.CreateTemp("", "test_map_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Load maps
	maps, err := LoadMapsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load maps: %v", err)
	}

	// Check number of maps
	if len(maps) != 2 {
		t.Errorf("Expected 2 maps, got %d", len(maps))
	}

	// Check first map dimensions
	if maps[0].Height != 4 {
		t.Errorf("Expected height 4 for first map, got %d", maps[0].Height)
	}
	if maps[0].Width != 6 {
		t.Errorf("Expected width 6 for first map, got %d", maps[0].Width)
	}

	// Check second map dimensions (should be same as first)
	if maps[1].Height != 4 {
		t.Errorf("Expected height 4 for second map, got %d", maps[1].Height)
	}
	if maps[1].Width != 6 {
		t.Errorf("Expected width 6 for second map, got %d", maps[1].Width)
	}
}

func TestLoadMapsWithDifferentSizes(t *testing.T) {
	// Test that loading maps with different sizes returns an error
	content := `OOOOOO
O----O
O----O
OOOOOO
---
OOOO
O--O
OOOO
`
	tmpFile, err := os.CreateTemp("", "test_map_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Load maps - should fail
	_, err = LoadMapsFromFile(tmpFile.Name())
	if err == nil {
		t.Error("Expected error when loading maps with different sizes, got nil")
	}
}

func TestMapWallDetection(t *testing.T) {
	content := `OOO
O-O
OOO
`
	tmpFile, err := os.CreateTemp("", "test_map_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(content)
	tmpFile.Close()

	maps, _ := LoadMapsFromFile(tmpFile.Name())
	m := &maps[0]

	// Test wall detection
	if !m.IsWall(0, 0) {
		t.Error("Expected (0,0) to be a wall")
	}
	if m.IsWall(1, 1) {
		t.Error("Expected (1,1) to not be a wall")
	}
}

func TestMapDotCounting(t *testing.T) {
	content := `OOO
O-O
OOO
`
	tmpFile, err := os.CreateTemp("", "test_map_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(content)
	tmpFile.Close()

	maps, _ := LoadMapsFromFile(tmpFile.Name())
	m := &maps[0]

	// Count dots (should be 1, the '-' in the middle)
	count := m.CountDots()
	if count != 1 {
		t.Errorf("Expected 1 dot, got %d", count)
	}

	// Eat the dot
	m.EatDot(1, 1)

	// Count again
	count = m.CountDots()
	if count != 0 {
		t.Errorf("Expected 0 dots after eating, got %d", count)
	}
}

func TestParseMapGridStarts(t *testing.T) {
	mapLines := []string{
		"monsters: 5",
		"OOOOO",
		"OP-MO",
		"O---O",
		"OM--O",
		"OOOOO",
	}

	m, err := parseMap(mapLines)
	if err != nil {
		t.Fatalf("Failed to parse map: %v", err)
	}

	if m.PlayerStart == nil || m.PlayerStart.X != 1 || m.PlayerStart.Y != 1 {
		t.Errorf("PlayerStart = %+v, want (1,1)", m.PlayerStart)
	}

	if len(m.MonsterStarts) != 2 {
		t.Errorf("MonsterStarts len = %d, want 2", len(m.MonsterStarts))
	}

	if m.MonsterCount != 2 {
		t.Errorf("MonsterCount = %d, want 2", m.MonsterCount)
	}
}

func TestPlayerMovement(t *testing.T) {
	content := `OOO
O-O
OOO
`
	tmpFile, err := os.CreateTemp("", "test_map_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(content)
	tmpFile.Close()

	maps, _ := LoadMapsFromFile(tmpFile.Name())
	m := &maps[0]

	player := NewPlayer(1, 1)

	// Try to move right (should hit wall)
	player.SetDirection(Right)
	player.Move(m)
	if player.X != 1 || player.Y != 1 {
		t.Errorf("Player should not move through wall, got position (%d, %d)", player.X, player.Y)
	}

	// Try to move left (should hit wall)
	player.SetDirection(Left)
	player.Move(m)
	if player.X != 1 || player.Y != 1 {
		t.Errorf("Player should not move through wall, got position (%d, %d)", player.X, player.Y)
	}
}
func TestParseMapMetaLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
	}{
		{
			name:      "colon format",
			line:      "name: Test Map",
			wantKey:   "name",
			wantValue: "Test Map",
		},
		{
			name:      "equals format",
			line:      "speedmodifier=1.5",
			wantKey:   "speedmodifier",
			wantValue: "1.5",
		},
		{
			name:      "colon with spaces",
			line:      "  material  :  classic  ",
			wantKey:   "material",
			wantValue: "classic",
		},
		{
			name:      "equals with spaces",
			line:      "  monsters  =  2  ",
			wantKey:   "monsters",
			wantValue: "2",
		},
		{
			name:      "no separator",
			line:      "invalid line",
			wantKey:   "",
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value := parseMapMetaLine(tt.line)
			if key != tt.wantKey || value != tt.wantValue {
				t.Errorf("parseMapMetaLine(%q) = (%q, %q), want (%q, %q)",
					tt.line, key, value, tt.wantKey, tt.wantValue)
			}
		})
	}
}

func TestParseMapSpeedModifier(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		wantSpeed float64
		wantErr   bool
	}{
		{
			name: "valid speedModifier 1.0",
			lines: []string{
				"name: Test",
				"speedModifier: 1.0",
				"OOOOO",
				"O---O",
				"OOOOO",
			},
			wantSpeed: 1.0,
			wantErr:   false,
		},
		{
			name: "valid speedModifier 0.5",
			lines: []string{
				"speedModifier: 0.5",
				"OOOOO",
				"O---O",
				"OOOOO",
			},
			wantSpeed: 0.5,
			wantErr:   false,
		},
		{
			name: "valid speedModifier 2.0",
			lines: []string{
				"speedModifier: 2.0",
				"OOOOO",
				"O---O",
				"OOOOO",
			},
			wantSpeed: 2.0,
			wantErr:   false,
		},
		{
			name: "invalid speedModifier too low",
			lines: []string{
				"speedModifier: 0.4",
				"OOOOO",
				"O---O",
				"OOOOO",
			},
			wantErr: true,
		},
		{
			name: "invalid speedModifier too high",
			lines: []string{
				"speedModifier: 2.1",
				"OOOOO",
				"O---O",
				"OOOOO",
			},
			wantErr: true,
		},
		{
			name: "default speedModifier when not specified",
			lines: []string{
				"name: Test",
				"OOOOO",
				"O---O",
				"OOOOO",
			},
			wantSpeed: 1.0,
			wantErr:   false,
		},
		{
			name: "speedModifier with equals format",
			lines: []string{
				"speedModifier=1.5",
				"OOOOO",
				"O---O",
				"OOOOO",
			},
			wantSpeed: 1.5,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := parseMap(tt.lines)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && m.SpeedModifier != tt.wantSpeed {
				t.Errorf("parseMap() SpeedModifier = %v, want %v",
					m.SpeedModifier, tt.wantSpeed)
			}
		})
	}
}

func TestMapRequiresTwoEscapesWhenMonstersPresent(t *testing.T) {
	content := `monsters: 1
OOO
O-O
OOO
`
	tmpFile, err := os.CreateTemp("", "test_map_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	_, err = LoadMapsFromFile(tmpFile.Name())
	if err == nil {
		t.Error("Expected error when monsters are present and only one escape route exists")
	}
}

func TestMapAllowsSingleEscapeWithoutMonsters(t *testing.T) {
	content := `monsters: 0
OOO
O-O
OOO
`
	tmpFile, err := os.CreateTemp("", "test_map_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	_, err = LoadMapsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Expected map to load without monsters, got error: %v", err)
	}
}
