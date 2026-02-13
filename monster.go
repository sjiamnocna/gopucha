package main

import (
	"math/rand"
)

type Monster struct {
	X         int
	Y         int
	Direction Direction
}

func NewMonster(x, y int, dir Direction) *Monster {
	return &Monster{
		X:         x,
		Y:         y,
		Direction: dir,
	}
}

func (mo *Monster) Move(m *Map) {
	// Try to move in current direction
	newX, newY := mo.X, mo.Y
	
	switch mo.Direction {
	case Up:
		newY--
	case Down:
		newY++
	case Left:
		newX--
	case Right:
		newX++
	}
	
	// If can't move forward or at an edge, try to turn
	if m.IsWall(newX, newY) || mo.shouldTurn(m) {
		mo.turn(m)
		// Try to move in new direction
		newX, newY = mo.X, mo.Y
		switch mo.Direction {
		case Up:
			newY--
		case Down:
			newY++
		case Left:
			newX--
		case Right:
			newX++
		}
	}
	
	if !m.IsWall(newX, newY) {
		mo.X = newX
		mo.Y = newY
	}
}

func (mo *Monster) shouldTurn(m *Map) bool {
	// Check if at an intersection (can turn)
	canTurnLeft := false
	canTurnRight := false
	
	switch mo.Direction {
	case Up, Down:
		canTurnLeft = !m.IsWall(mo.X-1, mo.Y)
		canTurnRight = !m.IsWall(mo.X+1, mo.Y)
	case Left, Right:
		canTurnLeft = !m.IsWall(mo.X, mo.Y-1)
		canTurnRight = !m.IsWall(mo.X, mo.Y+1)
	}
	
	// 30% chance to turn at an intersection
	if (canTurnLeft || canTurnRight) && rand.Float32() < 0.3 {
		return true
	}
	
	return false
}

func (mo *Monster) turn(m *Map) {
	// Get available directions
	directions := []Direction{}
	
	// Check all four directions
	if !m.IsWall(mo.X, mo.Y-1) {
		directions = append(directions, Up)
	}
	if !m.IsWall(mo.X, mo.Y+1) {
		directions = append(directions, Down)
	}
	if !m.IsWall(mo.X-1, mo.Y) {
		directions = append(directions, Left)
	}
	if !m.IsWall(mo.X+1, mo.Y) {
		directions = append(directions, Right)
	}
	
	// Don't go backwards if other options exist
	var backward Direction
	switch mo.Direction {
	case Up:
		backward = Down
	case Down:
		backward = Up
	case Left:
		backward = Right
	case Right:
		backward = Left
	}
	
	// Filter out backwards direction if there are other options
	filteredDirs := []Direction{}
	for _, d := range directions {
		if d != backward || len(directions) == 1 {
			filteredDirs = append(filteredDirs, d)
		}
	}
	
	if len(filteredDirs) > 0 {
		mo.Direction = filteredDirs[rand.Intn(len(filteredDirs))]
	}
}
