package gopucha

import "sync"

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
	Queue     []Direction
	mu        sync.Mutex // Protects Direction, Desired, and Queue
}

func NewPlayer(x, y int) *Player {
	return &Player{
		X:         x,
		Y:         y,
		Direction: Right,
		Desired:   Right,
		Queue:     nil,
	}
}

func (p *Player) Move(m *Map) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// If desired direction is available, turn immediately
	if p.Desired != p.Direction {
		dx, dy := directionDelta(p.Desired)
		if !m.IsWall(p.X+dx, p.Y+dy) {
			p.Direction = p.Desired
			if len(p.Queue) > 0 {
				p.Desired = p.Queue[0]
				p.Queue = p.Queue[1:]
			}
		}
	} else if len(p.Queue) > 0 {
		// Pop next direction from queue
		p.Desired = p.Queue[0]
		p.Queue = p.Queue[1:]
	}

	newX, newY := p.X, p.Y
	dx, dy := directionDelta(p.Direction)
	newX += dx
	newY += dy

	if !m.IsWall(newX, newY) {
		p.X = newX
		p.Y = newY
	}
	
	// Safety measure: if queue somehow grows beyond expected size, trim it
	// (should never happen with normal SetDirection usage)
	if len(p.Queue) > 5 {
		p.Queue = p.Queue[:3]
	}
}

func (p *Player) SetDirection(d Direction) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if d == p.Desired {
		return
	}

	if p.Desired == p.Direction {
		p.Desired = d
		return
	}

	if len(p.Queue) > 0 && p.Queue[len(p.Queue)-1] == d {
		return
	}

	if len(p.Queue) >= 3 {
		p.Queue[len(p.Queue)-1] = d
		return
	}
	p.Queue = append(p.Queue, d)
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
