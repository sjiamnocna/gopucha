//go:build !nogui
// +build !nogui

package gopucha

import (
	"fmt"
	"image/color"
	"math"
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
	app          fyne.App
	window       fyne.Window
	game         *Game
	blockSize    float32
	canvas       *fyne.Container
	keyCatcher   *keyCatcher
	ticker       *time.Ticker
	tickInterval time.Duration
	mapFile      string
	infoLabel    *widget.Label
	state        GameState
	countdownTicks int
	pauseTicks     int
	tickerDone    chan bool
	mouthOpen     bool
	mouthOpenRatio float64
	mouthAnimDir   int
	mouthTicker    *time.Ticker
	disableMonsters bool
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

func RunGUIGame(mapFile string, disableMonsters bool) error {
	guiGame := &GUIGame{
		app:          app.New(),
		blockSize:    defaultBlockSize,
		tickInterval: 340 * time.Millisecond,
		mapFile:      mapFile,
		state:        StateSettings,
		disableMonsters: disableMonsters,
	}

	guiGame.window = guiGame.app.NewWindow("Gopucha - Pac-Man Game")
	guiGame.window.Resize(fyne.NewSize(800, 600))
	guiGame.window.SetMaster()

	// Start game immediately (settings available via ESC)
	guiGame.startGame()

	guiGame.window.ShowAndRun()
	return nil
}

func (g *GUIGame) showSettings() {
	// Stop game loop while settings are open
	if g.ticker != nil {
		g.ticker.Stop()
	}

	// Speed slider
	speedLabel := widget.NewLabel("Speed:")
	speedValue := binding.NewFloat()
	// Invert: slider value 550 - tickInterval in ms
	invertedSpeed := 550 - g.tickInterval.Milliseconds()
	speedValue.Set(float64(invertedSpeed))

	speedSlider := widget.NewSliderWithData(50, 500, speedValue)
	speedSlider.Step = 50

	// Display shows actual milliseconds, inverted from slider
	speedDisplay := widget.NewLabel("")
	speedValue.AddListener(binding.NewDataListener(func() {
		val, _ := speedValue.Get()
		actualMs := 550 - int64(val)
		speedDisplay.SetText(fmt.Sprintf("%d ms (slower ← faster)", actualMs))
	}))

	// Map file selection
	mapFiles := g.findMapFiles()
	mapLabel := widget.NewLabel("Select Map:")
	mapSelect := widget.NewSelect(mapFiles, func(selected string) {})
	if g.mapFile != "" {
		mapSelect.SetSelected(g.mapFile)
	} else if len(mapFiles) > 0 {
		mapSelect.SetSelected(mapFiles[0])
	}

	content := container.NewVBox(
		widget.NewLabel("Settings"),
		widget.NewSeparator(),
		speedLabel,
		speedSlider,
		speedDisplay,
		widget.NewSeparator(),
		mapLabel,
		mapSelect,
	)

	dialog.ShowCustomConfirm("Settings", "Apply", "Cancel", content, func(apply bool) {
		if apply {
			speed, _ := speedValue.Get()
			// Invert the slider value: 550 - sliderValue = actual milliseconds
			actualMs := 550 - int64(speed)
			g.tickInterval = time.Duration(actualMs) * time.Millisecond
			if selected := mapSelect.Selected; selected != "" {
				g.mapFile = selected
			}
			g.startGame()
			g.initControls()
			return
		}

		// If cancelled and game is over or not started, restart fresh
		if g.state == StateGameOver || g.state == StateSettings {
			g.startGame()
			g.initControls()
			return
		}

		// If cancelled and game was running, resume it
		if g.state == StatePlaying && g.game != nil {
			g.startGameLoop()
			g.initControls()
		}
	}, g.window)
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
	// Stop and wait for previous ticker to finish
	if g.ticker != nil {
		g.ticker.Stop()
		// Wait for ticker goroutine to finish
		if g.tickerDone != nil {
			select {
			case <-g.tickerDone:
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	if g.mouthTicker != nil {
		g.mouthTicker.Stop()
		g.mouthTicker = nil
	}

	if g.mapFile == "" {
		mapFiles := g.findMapFiles()
		if len(mapFiles) > 0 {
			g.mapFile = mapFiles[0]
		}
	}

	// Load maps
	maps, err := LoadMapsFromFile(g.mapFile)
	if err != nil {
		g.showMapErrorAndClose(err)
		return
	}

	if len(maps) == 0 {
		g.showMapErrorAndClose(fmt.Errorf("no maps found in file"))
		return
	}

	g.game = NewGame(maps, g.disableMonsters)
	if g.game == nil {
		g.showMapErrorAndClose(fmt.Errorf("failed to create game"))
		return
	}

	g.state = StateLevelStart
	g.countdownTicks = 5
	g.pauseTicks = 0
	g.mouthOpen = false
	g.mouthOpenRatio = 0
	g.mouthAnimDir = 0
	g.setupGameUI()
	g.startGameLoop()
	g.initControls()
}

func (g *GUIGame) showMapErrorAndClose(err error) {
	msg := widget.NewLabel(err.Error())
	msg.Wrapping = fyne.TextWrapWord

	mapFiles := g.findMapFiles()
	selectLabel := widget.NewLabel("Select another map:")
	mapSelect := widget.NewSelect(mapFiles, func(selected string) {})
	if len(mapFiles) > 0 {
		mapSelect.SetSelected(mapFiles[0])
	}

	content := container.NewVBox(msg)
	if len(mapFiles) > 0 {
		content.Add(selectLabel)
		content.Add(mapSelect)
	}
	key := newKeyCatcher(func(ev *fyne.KeyEvent) {
		g.window.Close()
	})
	stack := container.NewStack(content, key)

	d := dialog.NewCustomConfirm("Map Error", "Load selected", "Exit", stack, func(apply bool) {
		if apply {
			if mapSelect.Selected != "" {
				g.mapFile = mapSelect.Selected
				g.startGame()
				return
			}
		}
		g.window.Close()
	}, g.window)
	d.Show()
	g.window.Canvas().Focus(key)
}

func (g *GUIGame) setupGameUI() {
	// Create the game canvas
	g.canvas = container.NewWithoutLayout()

	// Info panel
	g.infoLabel = widget.NewLabel(fmt.Sprintf("Level: %d | Score: %d | Lives: %d | Dots: %d",
		g.game.CurrentLevel+1, g.game.Score, g.game.Lives, g.game.CurrentMap.CountDots()))

	controls := widget.NewLabel("Controls: Arrow Keys to move | +/- to zoom | ESC for settings")

	topBar := container.NewVBox(g.infoLabel, controls)

	// Main container with scroll
	scroll := container.NewScroll(g.canvas)

	// Key capture overlay to ensure arrow keys are received reliably
	g.initControls()
	gameArea := container.NewStack(scroll, g.keyCatcher)

	content := container.NewBorder(topBar, nil, nil, nil, gameArea)

	g.window.SetContent(content)

	// Focus key catcher so arrow keys (including Up) are not swallowed by scroll
	g.window.Canvas().Focus(g.keyCatcher)

	// Set up mouse scroll for zoom (CTRL+Scroll)
	// Note: Fyne doesn't directly support scroll events, so we'll rely on keyboard shortcuts

	// Render initial state
	g.renderGame(g.infoLabel)
	g.calculateBlockSize()
}

func (g *GUIGame) initControls() {
	if g.keyCatcher == nil {
		g.keyCatcher = newKeyCatcher(func(ev *fyne.KeyEvent) {
			g.handleKeyPress(ev, g.infoLabel)
		})
	}
	if g.window != nil && g.window.Canvas() != nil {
		g.window.Canvas().Focus(g.keyCatcher)
	}
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

	// Render player (yellow semi-circle with mouth)
	g.drawPacman(float32(g.game.Player.X)*g.blockSize+g.blockSize*0.05, float32(g.game.Player.Y)*g.blockSize+g.blockSize*0.05, g.blockSize*0.9, g.game.Player.Direction)

	// Update info
	infoLabel.SetText(fmt.Sprintf("Level: %d | Score: %d | Lives: %d | Dots: %d | MapSize: %dx%d",
		g.game.CurrentLevel+1, g.game.Score, g.game.Lives, g.game.CurrentMap.CountDots(), m.Width, m.Height))

	g.canvas.Refresh()
}

func (g *GUIGame) drawPacman(x, y, size float32, dir Direction) {
	// Draw the Pac-Man circle with mouth cutout
	// We'll draw filled arcs by creating many small filled rectangles
	
	centerX := float64(x + size/2)
	centerY := float64(y + size/2)
	radius := float64(size / 2)
	
	// Mouth opening angle: 45 degrees on each side = 90 degrees total opening
	mouthHalfAngle := math.Pi / 4 // 45 degrees
	
	// Default mouth position (no rotation offset)
	defaultMouthAngle := 0.0
	
	// Screen coordinates: 0° = right, 90° = down, 180° = left, 270° = up
	var directionAngle float64
	switch dir {
	case Right:
		directionAngle = 0
	case Down:
		directionAngle = math.Pi / 2
	case Left:
		directionAngle = math.Pi
	case Up:
		directionAngle = 3 * math.Pi / 2
	}
	
	// Mouth center = default position + direction angle
	mouthCenterAngle := defaultMouthAngle + directionAngle
	
	effectiveHalfAngle := mouthHalfAngle * g.mouthOpenRatio
	mouthStartAngle := mouthCenterAngle - effectiveHalfAngle
	mouthEndAngle := mouthCenterAngle + effectiveHalfAngle
	
	// Draw Pac-Man as filled circle with mouth opening
	// We'll draw filled wedges/sectors except for the mouth area
	step := 1.0 // degrees per iteration
	for i := -180; i < 180; i += int(step) {
		angle := float64(i) * math.Pi / 180.0
		
		// Check if this angle is within the mouth opening (with wrap-around)
		if g.mouthOpenRatio > 0 && g.angleInRange(angle, mouthStartAngle, mouthEndAngle) {
			continue // Skip mouth area
		}
		
		// Draw small filled rectangle to form the circle
		thickness := 1.5
		for dist := 0.0; dist < radius; dist += thickness {
			// Point along the radial line
			px := centerX + (dist/radius)*radius*math.Cos(angle)
			py := centerY + (dist/radius)*radius*math.Sin(angle)
			
			pixel := canvas.NewRectangle(color.RGBA{255, 255, 0, 255}) // Yellow
			pixel.Resize(fyne.NewSize(float32(thickness), float32(thickness)))
			pixel.Move(fyne.NewPos(float32(px-thickness/2), float32(py-thickness/2)))
			g.canvas.Add(pixel)
		}
	}
}

func (g *GUIGame) angleInRange(angle, start, end float64) bool {
	// Normalize angles to (-pi, pi]
	norm := func(a float64) float64 {
		for a <= -math.Pi {
			a += 2 * math.Pi
		}
		for a > math.Pi {
			a -= 2 * math.Pi
		}
		return a
	}

	a := norm(angle)
	s := norm(start)
	e := norm(end)

	if s <= e {
		return a >= s && a <= e
	}
	// Wrap-around case
	return a >= s || a <= e
}

func (g *GUIGame) startMouthAnimation() {
	// Reset animation state
	g.mouthOpenRatio = 0
	g.mouthAnimDir = 1
	if g.mouthTicker != nil {
		return
	}

	// Run smooth open/close within a single game tick
	interval := g.tickInterval / 8
	if interval < 25*time.Millisecond {
		interval = 25 * time.Millisecond
	}

	g.mouthTicker = time.NewTicker(interval)
	go func() {
		for range g.mouthTicker.C {
			if g.mouthAnimDir == 0 {
				g.mouthTicker.Stop()
				g.mouthTicker = nil
				return
			}

			// Step size for smoother animation
			step := 0.25
			g.mouthOpenRatio += float64(g.mouthAnimDir) * step
			if g.mouthOpenRatio >= 1 {
				g.mouthOpenRatio = 1
				g.mouthAnimDir = -1
			} else if g.mouthOpenRatio <= 0 {
				g.mouthOpenRatio = 0
				g.mouthAnimDir = 0
			}

			fyne.DoAndWait(func() {
				g.renderGame(g.infoLabel)
			})
		}
	}()
}

func (g *GUIGame) minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func (g *GUIGame) maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func (g *GUIGame) renderGameWithCountdown(infoLabel *widget.Label) {
	// Render the same as normal game
	g.renderGame(infoLabel)

	// Add countdown display in center
	if g.countdownTicks >= 0 {
		countdownText := canvas.NewText(fmt.Sprintf("%d", g.countdownTicks), color.RGBA{255, 255, 255, 255})
		countdownText.TextSize = 48
		countdownText.Alignment = fyne.TextAlignCenter

		m := g.game.CurrentMap
		centerX := float32(m.Width) * g.blockSize / 2
		centerY := float32(m.Height) * g.blockSize / 2

		countdownText.Move(fyne.NewPos(centerX-40, centerY-40))
		g.canvas.Add(countdownText)
		g.canvas.Refresh()
	}
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
	// Handle arrow keys and space after game over to restart
	if g.state == StateGameOver || g.state == StateWon {
		switch ev.Name {
		case fyne.KeyUp, fyne.KeyDown, fyne.KeyLeft, fyne.KeyRight, fyne.KeySpace:
			if g.ticker != nil {
				g.ticker.Stop()
			}
			g.state = StatePlaying
			g.startGame()
			return
		}
	}

	if g.game == nil {
		return
	}

	// Allow direction input during countdown/pause to queue movement
	switch ev.Name {
	case fyne.KeyUp:
		if g.state == StatePlaying || g.state == StateLevelStart || g.state == StateLevelComplete {
			g.game.Player.SetDirection(Up)
		}
	case fyne.KeyDown:
		if g.state == StatePlaying || g.state == StateLevelStart || g.state == StateLevelComplete {
			g.game.Player.SetDirection(Down)
		}
	case fyne.KeyLeft:
		if g.state == StatePlaying || g.state == StateLevelStart || g.state == StateLevelComplete {
			g.game.Player.SetDirection(Left)
		}
	case fyne.KeyRight:
		if g.state == StatePlaying || g.state == StateLevelStart || g.state == StateLevelComplete {
			g.game.Player.SetDirection(Right)
		}
	case fyne.KeyEscape:
		if g.state == StatePlaying {
			g.showSettings()
		}
	case fyne.KeyEqual, fyne.KeyPlus:
		// + to zoom in (only during playing, not during countdown/pause)
		if g.state == StatePlaying {
			g.zoomIn(infoLabel)
		}
	case fyne.KeyMinus:
		// - to zoom out (only during playing, not during countdown/pause)
		if g.state == StatePlaying {
			g.zoomOut(infoLabel)
		}
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
	g.tickerDone = make(chan bool)
	g.ticker = time.NewTicker(g.tickInterval)

	go func() {
		defer func() {
			g.tickerDone <- true
		}()

		for range g.ticker.C {
			// Handle level start countdown phase
			if g.state == StateLevelStart {
				if g.countdownTicks > 0 {
					g.countdownTicks--
					fyne.DoAndWait(func() {
						g.renderGameWithCountdown(g.infoLabel)
					})
					continue
				}
				// Countdown finished, start playing
				g.state = StatePlaying
				continue
			}

			// Handle level completion pause
			if g.state == StateLevelComplete {
				if g.pauseTicks > 0 {
					g.pauseTicks--
					fyne.DoAndWait(func() {
						g.renderGame(g.infoLabel)
					})
					continue
				}
				// Pause finished, move to next level
				g.game.loadLevel(g.game.CurrentLevel + 1)
				g.state = StateLevelStart
				g.countdownTicks = 5
				g.pauseTicks = 0
				continue
			}

			if g.game.GameOver {
				g.ticker.Stop()
				g.state = StateGameOver
				return
			}

			if g.game.Won {
				g.ticker.Stop()
				g.state = StateWon
				fyne.Do(func() {
					dialog.ShowInformation("You Won!", fmt.Sprintf("Final Score: %d", g.game.Score), g.window)
				})
				return
			}

			// Only update game during StatePlaying
			if g.state != StatePlaying {
				fyne.DoAndWait(func() {
					g.renderGame(g.infoLabel)
				})
				continue
			}

			g.game.Update()

			// Mouth animation: smooth open/close on dot eat
			if g.game.DotEaten {
				g.startMouthAnimation()
			}

			// Restart countdown if a life was lost
			if g.game.LifeLost {
				g.countdownTicks = 5
				g.game.LifeLost = false
				g.state = StateLevelStart
			}

			// Check if level is completed
			if g.game.LevelCompleted {
				g.game.LevelCompleted = false
				
				// Check if all levels are done
				if g.game.CurrentLevel+1 >= len(g.game.Maps) {
					g.game.Won = true
				} else {
					g.state = StateLevelComplete
					g.pauseTicks = 2 // 2 tick pause
				}
			}

			fyne.DoAndWait(func() {
				g.renderGame(g.infoLabel)
			})
		}
	}()
}
