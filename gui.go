//go:build !nogui
// +build !nogui

package gopucha

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	defaultBlockSize = 20
	minBlockSize     = 10
	maxBlockSize     = 50
)

type GUIGame struct {
	app          fyne.App
	window       fyne.Window
	game         *Game
	blockSize    float32
	canvas       *fyne.Container
	keyCatcher   *keyCatcher
	ticker       *time.Ticker
	tickInterval time.Duration
	gameStarted  bool
	mapFile      string
	infoLabel    *widget.Label
	gameOverFlag bool
}

type keyCatcher struct {
	widget.BaseWidget
	onKey func(*fyne.KeyEvent)
}

func newKeyCatcher(onKey func(*fyne.KeyEvent)) *keyCatcher {
	k := &keyCatcher{onKey: onKey}
	k.ExtendBaseWidget(k)
	return k
}

func (k *keyCatcher) FocusGained() {}

func (k *keyCatcher) FocusLost() {}

func (k *keyCatcher) TypedKey(ev *fyne.KeyEvent) {
	if k.onKey != nil {
		k.onKey(ev)
	}
}

func (k *keyCatcher) TypedRune(r rune) {}

func (k *keyCatcher) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	return widget.NewSimpleRenderer(bg)
}

func RunGUIGame(mapFile string) error {
	guiGame := &GUIGame{
		app:          app.New(),
		blockSize:    defaultBlockSize,
		tickInterval: 200 * time.Millisecond,
		mapFile:      mapFile,
	}

	guiGame.window = guiGame.app.NewWindow("Gopucha - Pac-Man Game")
	guiGame.window.Resize(fyne.NewSize(800, 600))
	guiGame.window.SetMaster()

	// Show settings dialog before starting
	guiGame.showSettings()

	guiGame.window.ShowAndRun()
	return nil
}

func (g *GUIGame) showSettings() {
	// Speed slider
	speedLabel := widget.NewLabel("Game Speed:")
	speedValue := binding.NewFloat()
	speedValue.Set(200)

	speedSlider := widget.NewSliderWithData(50, 500, speedValue)
	speedSlider.Step = 50

	speedDisplay := widget.NewLabelWithData(binding.FloatToStringWithFormat(speedValue, "%.0f ms"))

	// Map file selection
	mapFiles := g.findMapFiles()
	mapLabel := widget.NewLabel("Select Map:")
	mapSelect := widget.NewSelect(mapFiles, func(selected string) {
		if selected != "" {
			g.mapFile = selected
		}
	})
	if len(mapFiles) > 0 {
		mapSelect.SetSelected(mapFiles[0])
	}

	// Start button
	startButton := widget.NewButton("Start Game", func() {
		speed, _ := speedValue.Get()
		g.tickInterval = time.Duration(speed) * time.Millisecond
		g.startGame()
	})

	content := container.NewVBox(
		widget.NewLabel("Game Settings"),
		widget.NewSeparator(),
		speedLabel,
		speedSlider,
		speedDisplay,
		widget.NewSeparator(),
		mapLabel,
		mapSelect,
		widget.NewSeparator(),
		startButton,
	)

	g.window.SetContent(content)
}

func (g *GUIGame) findMapFiles() []string {
	var mapFiles []string

	// Check current directory and maps subdirectory
	dirs := []string{".", "maps"}

	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".txt" {
				path := filepath.Join(dir, file.Name())
				mapFiles = append(mapFiles, path)
			}
		}
	}

	// Add the initially specified map file if not in list
	if g.mapFile != "" {
		found := false
		for _, f := range mapFiles {
			if f == g.mapFile {
				found = true
				break
			}
		}
		if !found {
			mapFiles = append([]string{g.mapFile}, mapFiles...)
		}
	}

	return mapFiles
}

func (g *GUIGame) startGame() {
	// Load maps
	maps, err := LoadMapsFromFile(g.mapFile)
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	if len(maps) == 0 {
		dialog.ShowError(fmt.Errorf("no maps found in file"), g.window)
		return
	}

	g.game = NewGame(maps)
	if g.game == nil {
		dialog.ShowError(fmt.Errorf("failed to create game"), g.window)
		return
	}

	g.gameStarted = true
	g.setupGameUI()
	g.startGameLoop()
}

func (g *GUIGame) setupGameUI() {
	// Create the game canvas
	g.canvas = container.NewWithoutLayout()

	// Info panel
	g.infoLabel = widget.NewLabel(fmt.Sprintf("Level: %d | Score: %d | Dots: %d",
		g.game.CurrentLevel+1, g.game.Score, g.game.CurrentMap.CountDots()))

	controls := widget.NewLabel("Controls: Arrow Keys to move | +/- to zoom | N for new game | ESC to quit")

	topBar := container.NewVBox(g.infoLabel, controls)

	// Main container with scroll
	scroll := container.NewScroll(g.canvas)

	// Key capture overlay to ensure arrow keys are received reliably
	g.keyCatcher = newKeyCatcher(func(ev *fyne.KeyEvent) {
		g.handleKeyPress(ev, g.infoLabel)
	})
	gameArea := container.NewStack(scroll, g.keyCatcher)

	content := container.NewBorder(topBar, nil, nil, nil, gameArea)

	g.window.SetContent(content)

	// Focus key catcher so arrow keys (including Up) are not swallowed by scroll
	g.window.Canvas().Focus(g.keyCatcher)

	// Set up mouse scroll for zoom (CTRL+Scroll)
	// Note: Fyne doesn't directly support scroll events, so we'll rely on keyboard shortcuts

	// Render initial state
	g.gameOverFlag = false
	g.renderGame(g.infoLabel)
	g.calculateBlockSize()
}

func (g *GUIGame) renderGame(infoLabel *widget.Label) {
	if g.game == nil || g.game.CurrentMap == nil {
		return
	}

	g.canvas.Objects = nil

	m := g.game.CurrentMap
	
	// Calculate canvas dimensions
	canvasWidth := float32(m.Width) * g.blockSize
	canvasHeight := float32(m.Height) * g.blockSize
	g.canvas.Resize(fyne.NewSize(canvasWidth, canvasHeight))

	// Render cells
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			rect := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
			rect.Resize(fyne.NewSize(g.blockSize, g.blockSize))
			rect.Move(fyne.NewPos(float32(x)*g.blockSize, float32(y)*g.blockSize))

			switch m.Cells[y][x] {
			case Wall:
				rect.FillColor = color.RGBA{0, 0, 255, 255} // Blue walls
			case Dot:
				rect.FillColor = color.RGBA{255, 255, 255, 255} // White dots
				rect.Resize(fyne.NewSize(g.blockSize/3, g.blockSize/3))
				rect.Move(fyne.NewPos(float32(x)*g.blockSize+g.blockSize/3, float32(y)*g.blockSize+g.blockSize/3))
			case Empty:
				rect.FillColor = color.RGBA{0, 0, 0, 255} // Black empty space
			}

			g.canvas.Add(rect)
		}
	}

	// Render monsters
	for _, monster := range g.game.Monsters {
		rect := canvas.NewRectangle(color.RGBA{255, 0, 0, 255}) // Red monsters
		rect.Resize(fyne.NewSize(g.blockSize*0.8, g.blockSize*0.8))
		rect.Move(fyne.NewPos(float32(monster.X)*g.blockSize+g.blockSize*0.1, float32(monster.Y)*g.blockSize+g.blockSize*0.1))
		g.canvas.Add(rect)
	}

	// Render player (yellow circle)
	circle := canvas.NewCircle(color.RGBA{255, 255, 0, 255})
	circle.Resize(fyne.NewSize(g.blockSize*0.9, g.blockSize*0.9))
	circle.Move(fyne.NewPos(float32(g.game.Player.X)*g.blockSize+g.blockSize*0.05, float32(g.game.Player.Y)*g.blockSize+g.blockSize*0.05))
	g.canvas.Add(circle)

	// Update info
	infoLabel.SetText(fmt.Sprintf("Level: %d | Score: %d | Dots: %d | MapSize: %dx%d",
		g.game.CurrentLevel+1, g.game.Score, g.game.CurrentMap.CountDots(), m.Width, m.Height))

	g.canvas.Refresh()
}

func (g *GUIGame) calculateBlockSize() {
	if g.game == nil || g.game.CurrentMap == nil {
		return
	}
	
	m := g.game.CurrentMap
	canvasSize := g.window.Canvas().Size()
	
	// Account for UI elements (roughly 100 pixels for top bar)
	availHeight := canvasSize.Height - 100
	availWidth := canvasSize.Width
	
	blockSizeByHeight := availHeight / float32(m.Height)
	blockSizeByWidth := availWidth / float32(m.Width)
	
	// Use the smaller to fit in window
	g.blockSize = blockSizeByHeight
	if blockSizeByWidth < blockSizeByHeight {
		g.blockSize = blockSizeByWidth
	}
	
	// Clamp to reasonable range
	if g.blockSize < minBlockSize {
		g.blockSize = minBlockSize
	}
	if g.blockSize > maxBlockSize {
		g.blockSize = maxBlockSize
	}
}

func (g *GUIGame) handleKeyPress(ev *fyne.KeyEvent, infoLabel *widget.Label) {
	// Handle N for new game
	if ev.Name == fyne.KeyN {
		if g.ticker != nil {
			g.ticker.Stop()
		}
		g.gameStarted = false
		g.gameOverFlag = false
		g.showSettings()
		return
	}

	// Handle arrow keys after game over to restart
	if g.gameOverFlag {
		switch ev.Name {
		case fyne.KeyUp, fyne.KeyDown, fyne.KeyLeft, fyne.KeyRight:
			if g.ticker != nil {
				g.ticker.Stop()
			}
			g.gameStarted = false
			g.gameOverFlag = false
			g.showSettings()
			return
		}
	}

	if !g.gameStarted || g.game == nil {
		return
	}

	switch ev.Name {
	case fyne.KeyUp:
		g.game.Player.SetDirection(Up)
	case fyne.KeyDown:
		g.game.Player.SetDirection(Down)
	case fyne.KeyLeft:
		g.game.Player.SetDirection(Left)
	case fyne.KeyRight:
		g.game.Player.SetDirection(Right)
	case fyne.KeyEscape:
		g.app.Quit()
	case fyne.KeyEqual, fyne.KeyPlus:
		// + to zoom in
		g.zoomIn(infoLabel)
	case fyne.KeyMinus:
		// - to zoom out
		g.zoomOut(infoLabel)
	}
}

func (g *GUIGame) zoomIn(infoLabel *widget.Label) {
	if g.blockSize < maxBlockSize {
		g.blockSize += 2
		g.renderGame(infoLabel)
	}
}

func (g *GUIGame) zoomOut(infoLabel *widget.Label) {
	if g.blockSize > minBlockSize {
		g.blockSize -= 2
		g.renderGame(infoLabel)
	}
}

func (g *GUIGame) startGameLoop() {
	g.ticker = time.NewTicker(g.tickInterval)

	go func() {
		for range g.ticker.C {
			if g.game.GameOver || g.game.Won {
				g.ticker.Stop()
				g.gameOverFlag = true

				var msg string
				if g.game.GameOver {
					msg = "Game Over! Press any arrow key to play again."
				} else {
					msg = fmt.Sprintf("You Won! Final Score: %d. Press any arrow key to play again.", g.game.Score)
				}

				dialog.ShowInformation("Game Ended", msg, g.window)
				return
			}

			g.game.Update()
			g.renderGame(g.infoLabel)
		}
	}()
}
