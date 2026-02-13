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
}

func NewPlayer(x, y int) *Player {
	return &Player{
		X:         x,
		Y:         y,
		Direction: Right,
	}
}

func (p *Player) Move(m *Map) {
	newX, newY := p.X, p.Y
	
	switch p.Direction {
	case Up:
		newY--
	case Down:
		newY++
	case Left:
		newX--
	case Right:
		newX++
	}
	
	if !m.IsWall(newX, newY) {
		p.X = newX
		p.Y = newY
	}
}

func (p *Player) SetDirection(d Direction) {
	p.Direction = d
}
