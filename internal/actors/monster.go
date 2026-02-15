package actors

import "github.com/sjiamnocna/gopucha/internal/maps"

func NewMonster(x, y int, dir Direction) *Monster {
	return &Monster{
		X:         x,
		Y:         y,
		Direction: dir,
	}
}

func (mo *Monster) Move(m *maps.Map, playerX, playerY int, monsters []Monster) {
	newX, newY := mo.X, mo.Y
	dx, dy := directionDelta(mo.Direction)
	newX += dx
	newY += dy

	// Check if next cell is blocked by wall or monster
	wallAhead := m.IsWall(newX, newY)
	monsterAhead := isOccupiedByMonster(newX, newY, monsters, mo)

	// If blocked, choose a new direction
	if wallAhead || monsterAhead {
		mo.Direction = mo.chooseDirection(m, playerX, playerY, monsters)
		newX, newY = mo.X, mo.Y
		dx, dy = directionDelta(mo.Direction)
		newX += dx
		newY += dy
	}

	// Move if the new cell is walkable
	if !m.IsWall(newX, newY) && !isOccupiedByMonster(newX, newY, monsters, mo) {
		mo.X = newX
		mo.Y = newY
	}
}

func (mo *Monster) chooseDirection(m *maps.Map, playerX, playerY int, monsters []Monster) Direction {
	dx := playerX - mo.X
	dy := playerY - mo.Y

	// Prioritize the axis with the larger distance
	// This gives simple chase behavior without pathfinding
	candidates := []Direction{}

	if dx > 0 {
		candidates = append(candidates, Right)
	} else if dx < 0 {
		candidates = append(candidates, Left)
	}

	if dy > 0 {
		candidates = append(candidates, Down)
	} else if dy < 0 {
		candidates = append(candidates, Up)
	}

	// Try each candidate direction, return the first valid one
	for _, d := range candidates {
		ndx, ndy := directionDelta(d)
		nx, ny := mo.X+ndx, mo.Y+ndy
		if !m.IsWall(nx, ny) && !isOccupiedByMonster(nx, ny, monsters, mo) {
			return d
		}
	}

	// If both preferred directions blocked, try the others
	allDirs := []Direction{Up, Down, Left, Right}
	for _, d := range allDirs {
		ndx, ndy := directionDelta(d)
		nx, ny := mo.X+ndx, mo.Y+ndy
		if !m.IsWall(nx, ny) && !isOccupiedByMonster(nx, ny, monsters, mo) {
			return d
		}
	}

	// Dead end, stay in place
	return mo.Direction
}

func isOccupiedByMonster(x, y int, monsters []Monster, self *Monster) bool {
	for i := range monsters {
		m := &monsters[i]
		if m == self {
			continue
		}
		if m.X == x && m.Y == y {
			return true
		}
	}
	return false
}

func manhattan(x1, y1, x2, y2 int) int {
	dx := x1 - x2
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y2
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}
