package gopucha

type Direction int

const (
	Up Direction = iota
	Down
	Left
	Right
)

type Player struct {
	X         int
	Y         int
	Direction Direction
	Desired   Direction
}

func NewPlayer(x, y int) *Player {
	return &Player{
		X:         x,
		Y:         y,
		Direction: Right,
		Desired:   Right,
	}
}

func (p *Player) Move(m *Map) {
	// If desired direction is available, turn immediately
	if p.Desired != p.Direction {
		dx, dy := directionDelta(p.Desired)
		if !m.IsWall(p.X+dx, p.Y+dy) {
			p.Direction = p.Desired
		}
	}

	newX, newY := p.X, p.Y
	dx, dy := directionDelta(p.Direction)
	newX += dx
	newY += dy

	if !m.IsWall(newX, newY) {
		p.X = newX
		p.Y = newY
	}
}

func (p *Player) SetDirection(d Direction) {
	p.Desired = d
}

func directionDelta(d Direction) (int, int) {
	switch d {
	case Up:
		return 0, -1
	case Down:
		return 0, 1
	case Left:
		return -1, 0
	case Right:
		return 1, 0
	default:
		return 0, 0
	}
}
