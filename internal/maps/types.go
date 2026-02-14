package maps

type Map struct {
	Width         int
	Height        int
	Cells         [][]Cell
	Name          string
	Material      string
	MonsterCount  int
	SpeedModifier float64
	PlayerStart   *StartPos
	MonsterStarts []StartPos
}

type StartPos struct {
	X int
	Y int
}

// Creature represents anything with X, Y coordinates (used for rendering)
type Creature interface {
	GetX() int
	GetY() int
}
