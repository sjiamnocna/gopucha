//go:build !nogui
// +build !nogui

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/sjiamnocna/gopucha/internal/gameplay"
	"time"
)

type GameState int

const (
	StateSettings GameState = iota
	StateLevelStart
	StatePlaying
	StateLevelComplete
	StateGameOver
	StateWon
)

type GUIGame struct {
	app                   fyne.App
	window                fyne.Window
	game                  *gameplay.Game
	blockSize             float32
	offsetX               float32 // X offset for centering the game
	offsetY               float32 // Y offset for centering the game
	canvas                *fyne.Container
	statusBarHeight       float32
	keyCatcher            *keyCatcher
	ticker                *time.Ticker
	tickInterval          time.Duration
	mapFile               string
	infoLabel             *widget.Label
	controlsLabel         *widget.Label
	livesDisplay          *fyne.Container // Hearts display for lives
	lastLives             int
	state                 GameState
	countdownStart        time.Time
	pauseTicks            int
	tickerDone            chan bool
	mouthOpen             bool
	mouthOpenRatio        float64
	mouthAnimDir          int
	mouthTicker           *time.Ticker
	disableMonsters       bool
	monsterTeethBlink     bool
	monsterTeethBlinkLast time.Time
	cachedMapRender       []fyne.CanvasObject // Cached static map layer
}

type renderPos struct {
	x float32
	y float32
}

type keyCatcher struct {
	widget.BaseWidget
	onKey func(*fyne.KeyEvent)
}
