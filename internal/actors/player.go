package actors

import (
	"github.com/sjiamnocna/gopucha/internal/maps"
)

func NewPlayer(x, y int) *Player {
	return &Player{
		X:         x,
		Y:         y,
		Direction: Right,
		Desired:   Right,
		Queue:     nil,
	}
}

func (p *Player) Move(m *maps.Map) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If desired direction is available, turn immediately.
	if p.Desired != p.Direction {
		dx, dy := directionDelta(p.Desired)
		if !m.IsWall(p.X+dx, p.Y+dy) {
			p.Direction = p.Desired
			p.dropQueued(p.Direction)
		} else {
			p.applyQueuedTurn(m)
		}
	} else {
		p.applyQueuedTurn(m)
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
	p.mu.Lock()
	defer p.mu.Unlock()

	if d == p.Desired {
		return
	}

	// Last pressed direction always wins, but keep a short queue for fast turns.
	p.Desired = d
	if len(p.Queue) == 0 || p.Queue[len(p.Queue)-1] != d {
		p.Queue = append(p.Queue, d)
		if len(p.Queue) > 2 {
			p.Queue = p.Queue[len(p.Queue)-2:]
		}
	}
}

func (p *Player) applyQueuedTurn(m *maps.Map) {
	if len(p.Queue) == 0 {
		return
	}

	for i, d := range p.Queue {
		dx, dy := directionDelta(d)
		if !m.IsWall(p.X+dx, p.Y+dy) {
			p.Desired = d
			p.Direction = d
			p.Queue = p.Queue[i+1:]
			return
		}
	}
}

func (p *Player) dropQueued(d Direction) {
	if len(p.Queue) == 0 {
		return
	}

	filtered := p.Queue[:0]
	for _, q := range p.Queue {
		if q != d {
			filtered = append(filtered, q)
		}
	}
	p.Queue = filtered
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
