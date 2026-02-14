package actors

import "sync"

type Player struct {
	X         int
	Y         int
	Direction Direction
	Desired   Direction
	Queue     []Direction
	mu        sync.Mutex // Protects Direction, Desired, and Queue
}

type Monster struct {
	X         int
	Y         int
	Direction Direction
}
