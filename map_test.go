package gopucha

import (
	"os"
	"testing"
)

func TestLoadMapsFromFile(t *testing.T) {
	// Create a temporary test map file
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
	
	// Check second map dimensions
	if maps[1].Height != 3 {
		t.Errorf("Expected height 3 for second map, got %d", maps[1].Height)
	}
	if maps[1].Width != 4 {
		t.Errorf("Expected width 4 for second map, got %d", maps[1].Width)
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
