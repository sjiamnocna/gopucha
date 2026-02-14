package gameplay

import (
	"github.com/sjiamnocna/gopucha/internal/actors"
	"github.com/sjiamnocna/gopucha/internal/maps"
)

type Game struct {
	CurrentMap           *maps.Map
	Maps                 []maps.Map
	CurrentLevel         int
	Player               *actors.Player
	Monsters             []actors.Monster
	GameOver             bool
	Won                  bool
	Score                int
	Lives                int
	LifeLost             bool
	DisableMonsters      bool
	DotEaten             bool
	CurrentSpeedModifier float64
	LevelCompleted       bool
}
