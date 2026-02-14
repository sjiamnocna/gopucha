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

func (mo *Monster) chooseDirection(m *Map, playerX, playerY int, monsters []Monster) Direction {
	bestDir := mo.Direction
	bestDist := -1
	distMap := bfsDistanceMap(m, playerX, playerY, monsters, mo)

	for _, d := range []Direction{Up, Down, Left, Right} {
		dx, dy := directionDelta(d)
		nx, ny := mo.X+dx, mo.Y+dy
		if m.IsWall(nx, ny) || isOccupiedByMonster(nx, ny, monsters, mo) {
			continue
		}
		if ny >= 0 && ny < m.Height && nx >= 0 && nx < m.Width {
			dist := distMap[ny][nx]
			if dist >= 0 && (bestDist == -1 || dist < bestDist) {
				bestDist = dist
				bestDir = d
			}
		}
	}

	if bestDist == -1 {
		for _, d := range []Direction{Up, Down, Left, Right} {
			dx, dy := directionDelta(d)
			nx, ny := mo.X+dx, mo.Y+dy
			if !m.IsWall(nx, ny) && !isOccupiedByMonster(nx, ny, monsters, mo) {
				return d
			}
		}
	}

	return bestDir
}

func bfsDistanceMap(m *Map, targetX, targetY int, monsters []Monster, self *Monster) [][]int {
	dist := make([][]int, m.Height)
	for y := range dist {
		dist[y] = make([]int, m.Width)
		for x := range dist[y] {
			dist[y][x] = -1
		}
	}

	if targetX < 0 || targetY < 0 || targetX >= m.Width || targetY >= m.Height {
		return dist
	}
	if m.IsWall(targetX, targetY) {
		return dist
	}

	queueX := []int{targetX}
	queueY := []int{targetY}
	dist[targetY][targetX] = 0

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
			if dist[ny][nx] != -1 {
				continue
			}
			if m.IsWall(nx, ny) {
				continue
			}
			if isOccupiedByMonster(nx, ny, monsters, self) {
				continue
			}
			dist[ny][nx] = dist[y][x] + 1
			queueX = append(queueX, nx)
			queueY = append(queueY, ny)
		}
	}

	return dist
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
