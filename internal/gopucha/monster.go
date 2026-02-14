package gopucha



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

func (mo *Monster) Move(m *Map, playerX, playerY int, monsters []Monster) {
	newX, newY := mo.X, mo.Y
	dx, dy := directionDelta(mo.Direction)
	newX += dx
	newY += dy

	// Only turn when next cell is a wall or another monster
	if m.IsWall(newX, newY) || isOccupiedByMonster(newX, newY, monsters, mo) {
		mo.Direction = mo.chooseDirection(m, playerX, playerY, monsters)
		newX, newY = mo.X, mo.Y
		dx, dy = directionDelta(mo.Direction)
		newX += dx
		newY += dy
	}

	if !m.IsWall(newX, newY) && !isOccupiedByMonster(newX, newY, monsters, mo) {
		mo.X = newX
		mo.Y = newY
	}
}

func (mo *Monster) chooseDirection(m *Map, playerX, playerY int, monsters []Monster) Direction {
	bestDir := mo.Direction
	bestDist := -1

	for _, d := range []Direction{Up, Down, Left, Right} {
		dx, dy := directionDelta(d)
		nx, ny := mo.X+dx, mo.Y+dy
		if m.IsWall(nx, ny) || isOccupiedByMonster(nx, ny, monsters, mo) {
			continue
		}
		dist := manhattan(nx, ny, playerX, playerY)
		if bestDist == -1 || dist < bestDist {
			bestDist = dist
			bestDir = d
		}
	}

	return bestDir
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
