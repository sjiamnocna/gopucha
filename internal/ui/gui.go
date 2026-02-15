//go:build !nogui
// +build !nogui

package ui

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/sjiamnocna/gopucha/internal/actors"
	"github.com/sjiamnocna/gopucha/internal/gameplay"
	"github.com/sjiamnocna/gopucha/internal/maps"
)

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
		app:             app.New(),
		blockSize:       defaultBlockSize,
		tickInterval:    defaultTickInterval,
		mapFile:         mapFile,
		state:           StateSettings,
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

	closed := false
	handleClose := func(apply bool) {
		if closed {
			return
		}
		closed = true
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
			// Ensure focus after dialog closes
			if g.keyCatcher != nil {
				g.window.Canvas().Focus(g.keyCatcher)
			}
			return
		}

		// If cancelled and game is over or not started, restart fresh
		if g.state == StateGameOver || g.state == StateSettings {
			g.startGame()
			g.initControls()
			// Ensure focus after dialog closes
			if g.keyCatcher != nil {
				g.window.Canvas().Focus(g.keyCatcher)
			}
			return
		}

		// If cancelled and game was running, resume it
		if (g.state == StatePlaying || g.state == StateLevelStart || g.state == StateLevelComplete) && g.game != nil {
			g.startGameLoop()
			g.initControls()
			// Ensure focus after dialog closes
			if g.keyCatcher != nil {
				g.window.Canvas().Focus(g.keyCatcher)
			}
		}
	}

	var d *dialog.ConfirmDialog
	key := newKeyCatcher(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			handleClose(false)
			if d != nil {
				d.Hide()
			}
		} else if ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter {
			handleClose(true)
			if d != nil {
				d.Hide()
			}
		}
	})
	stack := container.NewStack(content, key)

	d = dialog.NewCustomConfirm("Settings", "Apply", "Cancel", stack, func(apply bool) {
		handleClose(apply)
	}, g.window)

	// Set up canvas-level key handler for the dialog
	originalHandler := g.window.Canvas().OnTypedKey()
	g.window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			handleClose(false)
			d.Hide()
			if originalHandler != nil {
				g.window.Canvas().SetOnTypedKey(originalHandler)
			}
		} else if ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter {
			handleClose(true)
			d.Hide()
			if originalHandler != nil {
				g.window.Canvas().SetOnTypedKey(originalHandler)
			}
		}
	})

	d.SetOnClosed(func() {
		if originalHandler != nil {
			g.window.Canvas().SetOnTypedKey(originalHandler)
		}
	})

	d.Show()
	g.window.Canvas().Focus(key)
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
	mapsList, err := maps.LoadMapsFromFile(g.mapFile)
	if err != nil {
		g.showMapErrorAndClose(err)
		return
	}

	if len(mapsList) == 0 {
		g.showMapErrorAndClose(fmt.Errorf("no maps found in file"))
		return
	}

	g.game = gameplay.NewGame(mapsList, g.disableMonsters)
	if g.game == nil {
		g.showMapErrorAndClose(fmt.Errorf("failed to create game"))
		return
	}

	g.state = StateLevelStart
	g.countdownStart = time.Now()
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

	content := container.NewVBox(msg)

	var d dialog.Dialog
	key := newKeyCatcher(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape || ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter {
			if d != nil {
				d.Hide()
			}
			g.window.Close()
		}
	})
	stack := container.NewStack(content, key)

	d = dialog.NewCustom("Map Error", "Exit", stack, g.window)

	originalHandler := g.window.Canvas().OnTypedKey()
	g.window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape || ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter {
			d.Hide()
			g.window.Close()
			if originalHandler != nil {
				g.window.Canvas().SetOnTypedKey(originalHandler)
			}
		}
	})

	d.SetOnClosed(func() {
		g.window.Close()
		if originalHandler != nil {
			g.window.Canvas().SetOnTypedKey(originalHandler)
		}
	})
	if d != nil {
		d.Show()
	}
	g.window.Canvas().Focus(key)
}

func (g *GUIGame) setupGameUI() {
	// Create the game canvas
	g.canvas = container.NewWithoutLayout()
	g.cachedMapRender = nil

	// Info panel - left side stats
	g.infoLabel = widget.NewLabel(fmt.Sprintf("%s | Score: %d | Dots: %d",
		g.levelDisplayName(), g.game.Score, g.game.CurrentMap.CountDots()))
	g.infoLabel.TextStyle = fyne.TextStyle{Bold: true}

	g.controlsLabel = widget.NewLabel("Controls: Arrow Keys to move | F2 restart | +/- zoom | ESC settings")
	g.controlsLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Create styled status bar background
	statusBarBg := canvas.NewRectangle(color.RGBA{40, 40, 50, 255})
	g.infoLabel.Importance = widget.HighImportance

	// Create status bar - only show level, score, dots and hearts (no controls line)
	g.livesDisplay = g.createLivesDisplay(g.game.Lives)
	g.lastLives = g.game.Lives
	topBarContent := container.NewHBox(
		g.infoLabel,
		layout.NewSpacer(), // Flexible spacer that grows to push hearts to the far right
		g.livesDisplay,     // Hearts display on the far right
	)
	// Create status bar with minimal padding for fixed height
	statusBar := container.NewStack(statusBarBg, container.NewPadded(topBarContent))
	g.statusBarHeight = statusBar.MinSize().Height

	// Main container with scroll
	scroll := container.NewScroll(g.canvas)

	// Key capture overlay to ensure arrow keys are received reliably
	g.initControls()
	gameArea := container.NewStack(scroll, g.keyCatcher)

	// Combine status bar and game area vertically, filling entire space
	content := container.NewBorder(statusBar, nil, nil, nil, gameArea)

	// Set window background to match game background (black)
	windowBg := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	contentWithBg := container.NewStack(windowBg, content)

	g.window.SetContent(contentWithBg)
	g.window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			g.showSettings()
			return
		}
		g.handleKeyPress(ev, g.infoLabel)
	})
	// Size window to map based on initial block size, then recalc for exact fit
	g.calculateBlockSize()
	g.resizeWindowForMap()
	g.calculateBlockSize()

	// Focus key catcher so arrow keys (including Up) are not swallowed by scroll
	g.window.Canvas().Focus(g.keyCatcher)

	// Ensure keyCatcher stays focused
	g.keyCatcher.Refresh()

	// Set up mouse scroll for zoom (CTRL+Scroll)
	// Note: Fyne doesn't directly support scroll events, so we'll rely on keyboard shortcuts

	// Render initial state
	g.renderGame(g.infoLabel)

	// Add window size tracking for dynamic resizing
	go func() {
		lastWidth := float32(0)
		lastHeight := float32(0)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			if g.window == nil || g.window.Canvas() == nil {
				continue
			}
			size := g.window.Canvas().Size()
			if size.Width != lastWidth || size.Height != lastHeight {
				lastWidth = size.Width
				lastHeight = size.Height
				// Window size changed, recalculate block size
				g.calculateBlockSize()
				fyne.Do(func() {
					g.renderGame(g.infoLabel)
				})
			}
		}
	}()
}

func (g *GUIGame) resizeWindowForMap() {
	if g.game == nil || g.game.CurrentMap == nil || g.window == nil {
		return
	}

	mapWidth := float32(g.game.CurrentMap.Width+borderBlocks*2) * g.blockSize
	mapHeight := float32(g.game.CurrentMap.Height+borderBlocks*2)*g.blockSize + g.currentStatusBarHeight()

	winWidth := mapWidth
	winHeight := mapHeight
	if winWidth < minWindowSize {
		winWidth = minWindowSize
	}
	if winHeight < minWindowSize {
		winHeight = minWindowSize
	}

	g.window.Resize(fyne.NewSize(winWidth, winHeight))
}

func (g *GUIGame) createLivesDisplay(lives int) *fyne.Container {
	heartsContainer := container.NewHBox()
	for i := 0; i < lives; i++ {
		heart := widget.NewLabel("❤️")
		heartsContainer.Add(heart)
	}
	return heartsContainer
}

func (g *GUIGame) initControls() {
	if g.keyCatcher == nil {
		g.keyCatcher = newKeyCatcher(func(ev *fyne.KeyEvent) {
			g.handleKeyPress(ev, g.infoLabel)
		})
	}
}

func (g *GUIGame) renderGame(infoLabel *widget.Label) {
	playerPos, monsterPos := g.capturePositions()
	g.renderGameAt(infoLabel, playerPos, monsterPos)
}

func (g *GUIGame) renderGameAt(infoLabel *widget.Label, playerPos renderPos, monsterPos []renderPos) {
	if g.game == nil || g.game.CurrentMap == nil {
		return
	}

	m := g.game.CurrentMap

	// Calculate canvas dimensions
	canvasWidth := float32(m.Width+borderBlocks*2) * g.blockSize
	canvasHeight := float32(m.Height+borderBlocks*2) * g.blockSize
	g.canvas.Resize(fyne.NewSize(canvasWidth, canvasHeight))

	mapOriginX, mapOriginY := g.mapOrigin()

	// Rebuild cache if needed (map changed or block size changed)
	if len(g.cachedMapRender) == 0 {
		objects := make([]fyne.CanvasObject, 0, (m.Width+borderBlocks*2)*(m.Height+borderBlocks*2)*3)

		// Pre-render backgrounds and walls (static parts)
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				rect := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
				rect.Resize(fyne.NewSize(g.blockSize, g.blockSize))
				rect.Move(fyne.NewPos(mapOriginX+float32(x)*g.blockSize, mapOriginY+float32(y)*g.blockSize))
				objects = append(objects, rect)

				if m.Cells[y][x] == maps.Wall {
					// Inline wall drawing to avoid extra function calls
					g.drawWallCellIntoAt(mapOriginX+float32(x)*g.blockSize, mapOriginY+float32(y)*g.blockSize, x, y, m, &objects)
				}
			}
		}
		if borderBlocks > 0 {
			borderMap := g.buildBorderMap(m)
			for y := 0; y < borderMap.Height; y++ {
				for x := 0; x < borderMap.Width; x++ {
					if x != 0 && y != 0 && x != borderMap.Width-1 && y != borderMap.Height-1 {
						continue
					}
					originX := g.offsetX + float32(x)*g.blockSize
					originY := g.offsetY + float32(y)*g.blockSize
					g.drawWallCellIntoAt(originX, originY, x, y, borderMap, &objects)
				}
			}
		}
		g.cachedMapRender = objects
	}

	// Start with cached static layer
	g.canvas.Objects = append([]fyne.CanvasObject{}, g.cachedMapRender...)

	// Add dots (dynamic, eaten dots disappear)
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if m.Cells[y][x] == maps.Dot {
				dotSize := g.blockSize * 0.35
				dot := canvas.NewCircle(color.RGBA{255, 230, 0, 255})
				dot.StrokeColor = color.RGBA{180, 90, 0, 255}
				dot.StrokeWidth = dotSize * 0.2
				dot.Resize(fyne.NewSize(dotSize, dotSize))
				dot.Move(fyne.NewPos(mapOriginX+float32(x)*g.blockSize+(g.blockSize-dotSize)/2, mapOriginY+float32(y)*g.blockSize+(g.blockSize-dotSize)/2))
				g.canvas.Add(dot)
			}
		}
	}

	// Render monsters
	for i, monster := range g.game.Monsters {
		pos := renderPos{x: float32(monster.X), y: float32(monster.Y)}
		if i < len(monsterPos) {
			pos = monsterPos[i]
		}
		moving := math.Abs(float64(pos.x-float32(monster.X))) > 0.001 || math.Abs(float64(pos.y-float32(monster.Y))) > 0.001
		blinkSwap := g.monsterTeethBlinkSwap(moving)
		g.drawMonster(mapOriginX+pos.x*g.blockSize+g.blockSize*0.1, mapOriginY+pos.y*g.blockSize+g.blockSize*0.1, g.blockSize*0.8, blinkSwap)
	}

	// Render player (yellow semi-circle with mouth)
	g.drawPacman(mapOriginX+playerPos.x*g.blockSize+g.blockSize*0.05, mapOriginY+playerPos.y*g.blockSize+g.blockSize*0.05, g.blockSize*0.9, g.game.Player.Direction)

	// Update info
	infoLabel.SetText(fmt.Sprintf("%s | Score: %d | Dots: %d",
		g.levelDisplayName(), g.game.Score, g.game.CurrentMap.CountDots()))

	// Update lives display only when needed
	if g.lastLives != g.game.Lives {
		g.livesDisplay.Objects = nil
		for i := 0; i < g.game.Lives; i++ {
			heart := widget.NewLabel("❤️")
			g.livesDisplay.Add(heart)
		}
		g.livesDisplay.Refresh()
		g.lastLives = g.game.Lives
	}

	// Show/hide controls based on state
	g.updateControlsVisibility()

	g.canvas.Refresh()
}

func (g *GUIGame) updateControlsVisibility() {
	if g.controlsLabel == nil {
		return
	}

	// Only show controls during countdown/level start
	if g.state == StateLevelStart && time.Since(g.countdownStart) < 3*time.Second {
		g.controlsLabel.SetText("Controls: Arrow Keys to move | F2 restart | +/- zoom | ESC settings")
	} else {
		g.controlsLabel.SetText("")
	}
}

func (g *GUIGame) mapOrigin() (float32, float32) {
	borderOffset := float32(borderBlocks) * g.blockSize
	return g.offsetX + borderOffset, g.offsetY + borderOffset
}

func (g *GUIGame) levelDisplayName() string {
	if g.game == nil || g.game.CurrentMap == nil {
		return ""
	}
	name := strings.TrimSpace(g.game.CurrentMap.Name)
	if name == "" {
		return fmt.Sprintf("Level %d", g.game.CurrentLevel+1)
	}
	return name
}

func (g *GUIGame) currentStatusBarHeight() float32 {
	if g.statusBarHeight > 0 {
		return g.statusBarHeight
	}
	return statusBarHeight
}

func (g *GUIGame) buildBorderMap(m *maps.Map) *maps.Map {
	width := m.Width + borderBlocks*2
	height := m.Height + borderBlocks*2
	cells := make([][]maps.Cell, height)
	for y := 0; y < height; y++ {
		cells[y] = make([]maps.Cell, width)
		for x := 0; x < width; x++ {
			if x == 0 || y == 0 || x == width-1 || y == height-1 {
				cells[y][x] = maps.Wall
				continue
			}
			cells[y][x] = m.Cells[y-1][x-1]
		}
	}

	return &maps.Map{
		Width:    width,
		Height:   height,
		Cells:    cells,
		Material: m.Material,
	}
}

func (g *GUIGame) drawWallCell(x, y int, m *maps.Map) {
	objs := make([]fyne.CanvasObject, 0)
	mapOriginX, mapOriginY := g.mapOrigin()
	g.drawWallCellIntoAt(mapOriginX+float32(x)*g.blockSize, mapOriginY+float32(y)*g.blockSize, x, y, m, &objs)
	for _, obj := range objs {
		g.canvas.Add(obj)
	}
}

func (g *GUIGame) drawWallCellInto(x, y int, m *maps.Map, objs *[]fyne.CanvasObject) {
	mapOriginX, mapOriginY := g.mapOrigin()
	originX := mapOriginX + float32(x)*g.blockSize
	originY := mapOriginY + float32(y)*g.blockSize
	g.drawWallCellIntoAt(originX, originY, x, y, m, objs)
}

func (g *GUIGame) drawWallCellIntoAt(originX, originY float32, x, y int, m *maps.Map, objs *[]fyne.CanvasObject) {
	line := g.blockSize * 0.08
	if line < 1 {
		line = 1
	}

	mat := strings.ToLower(strings.TrimSpace(m.Material))
	hasTop := y-1 >= 0 && m.Cells[y-1][x] == maps.Wall
	hasBottom := y+1 < m.Height && m.Cells[y+1][x] == maps.Wall
	hasLeft := x-1 >= 0 && m.Cells[y][x-1] == maps.Wall
	hasRight := x+1 < m.Width && m.Cells[y][x+1] == maps.Wall
	if mat == "brick" || mat == "bricks" {
		base := canvas.NewRectangle(color.RGBA{160, 75, 25, 255})
		base.Resize(fyne.NewSize(g.blockSize, g.blockSize))
		base.Move(fyne.NewPos(originX, originY))
		*objs = append(*objs, base)

		lineColor := color.RGBA{30, 20, 10, 255}
		if !hasTop {
			top := canvas.NewRectangle(lineColor)
			top.Resize(fyne.NewSize(g.blockSize, line))
			top.Move(fyne.NewPos(originX, originY))
			*objs = append(*objs, top)
		}

		if !hasBottom {
			bottom := canvas.NewRectangle(lineColor)
			bottom.Resize(fyne.NewSize(g.blockSize, line))
			bottom.Move(fyne.NewPos(originX, originY+g.blockSize-line))
			*objs = append(*objs, bottom)
		}

		mid := canvas.NewRectangle(lineColor)
		mid.Resize(fyne.NewSize(g.blockSize, line))
		mid.Move(fyne.NewPos(originX, originY+g.blockSize/2-line/2))
		*objs = append(*objs, mid)

		vLeft := canvas.NewRectangle(lineColor)
		vLeft.Resize(fyne.NewSize(line, g.blockSize/2))
		vLeft.Move(fyne.NewPos(originX+g.blockSize*0.33, originY))
		*objs = append(*objs, vLeft)

		vRight := canvas.NewRectangle(lineColor)
		vRight.Resize(fyne.NewSize(line, g.blockSize/2))
		vRight.Move(fyne.NewPos(originX+g.blockSize*0.66, originY+g.blockSize/2))
		*objs = append(*objs, vRight)
		return
	}

	if mat == "purpledots" || mat == "purple-dots" || mat == "purple dots" {
		base := canvas.NewRectangle(color.RGBA{55, 15, 70, 255})
		base.Resize(fyne.NewSize(g.blockSize, g.blockSize))
		base.Move(fyne.NewPos(originX, originY))
		*objs = append(*objs, base)

		seed := int64(x+1)*10007 + int64(y+1)*1009 + int64(m.Width)*37 + int64(m.Height)*97
		rnd := rand.New(rand.NewSource(seed))
		dotCount := 6 + rnd.Intn(7)

		centerX := originX + g.blockSize*(0.4+0.2*float32(rnd.Float64()))
		centerY := originY + g.blockSize*(0.4+0.2*float32(rnd.Float64()))
		clusterRadius := g.blockSize * (0.28 + 0.08*float32(rnd.Float64()))
		padding := float32(1)

		for i := 0; i < dotCount; i++ {
			angle := rnd.Float64() * 2 * math.Pi
			radius := rnd.Float64() * float64(clusterRadius)
			dx := float32(math.Cos(angle)) * float32(radius)
			dy := float32(math.Sin(angle)) * float32(radius)
			dotSize := g.blockSize * (0.06 + 0.06*float32(rnd.Float64()))
			xPos := centerX + dx - dotSize/2
			yPos := centerY + dy - dotSize/2

			minX := originX + padding
			minY := originY + padding
			maxX := originX + g.blockSize - dotSize - padding
			maxY := originY + g.blockSize - dotSize - padding
			if xPos < minX {
				xPos = minX
			} else if xPos > maxX {
				xPos = maxX
			}
			if yPos < minY {
				yPos = minY
			} else if yPos > maxY {
				yPos = maxY
			}

			shadeR := uint8(120 + rnd.Intn(71))
			shadeG := uint8(40 + rnd.Intn(51))
			shadeB := uint8(160 + rnd.Intn(71))
			dot := canvas.NewCircle(color.RGBA{shadeR, shadeG, shadeB, 255})
			dot.Resize(fyne.NewSize(dotSize, dotSize))
			dot.Move(fyne.NewPos(xPos, yPos))
			*objs = append(*objs, dot)
		}
		return
	}

	if mat == "" || mat == "classic" || mat == "steel" || mat == "metal" {
		base := canvas.NewRectangle(color.RGBA{180, 185, 195, 255})
		base.Resize(fyne.NewSize(g.blockSize, g.blockSize))
		base.Move(fyne.NewPos(originX, originY))
		*objs = append(*objs, base)

		border := color.RGBA{120, 125, 135, 255}
		if !hasTop {
			top := canvas.NewRectangle(border)
			top.Resize(fyne.NewSize(g.blockSize, line))
			top.Move(fyne.NewPos(originX, originY))
			*objs = append(*objs, top)
		}

		if !hasBottom {
			bottom := canvas.NewRectangle(border)
			bottom.Resize(fyne.NewSize(g.blockSize, line))
			bottom.Move(fyne.NewPos(originX, originY+g.blockSize-line))
			*objs = append(*objs, bottom)
		}

		if !hasLeft {
			left := canvas.NewRectangle(border)
			left.Resize(fyne.NewSize(line, g.blockSize))
			left.Move(fyne.NewPos(originX, originY))
			*objs = append(*objs, left)
		}

		if !hasRight {
			right := canvas.NewRectangle(border)
			right.Resize(fyne.NewSize(line, g.blockSize))
			right.Move(fyne.NewPos(originX+g.blockSize-line, originY))
			*objs = append(*objs, right)
		}
		return
	}

	if mat == "graybricks" || mat == "gray-bricks" || mat == "gray bricks" {
		base := canvas.NewRectangle(color.RGBA{150, 155, 160, 255})
		base.Resize(fyne.NewSize(g.blockSize, g.blockSize))
		base.Move(fyne.NewPos(originX, originY))
		*objs = append(*objs, base)

		lineColor := color.RGBA{60, 65, 70, 255}
		if !hasTop {
			top := canvas.NewRectangle(lineColor)
			top.Resize(fyne.NewSize(g.blockSize, line))
			top.Move(fyne.NewPos(originX, originY))
			*objs = append(*objs, top)
		}

		if !hasBottom {
			bottom := canvas.NewRectangle(lineColor)
			bottom.Resize(fyne.NewSize(g.blockSize, line))
			bottom.Move(fyne.NewPos(originX, originY+g.blockSize-line))
			*objs = append(*objs, bottom)
		}

		mid := canvas.NewRectangle(lineColor)
		mid.Resize(fyne.NewSize(g.blockSize, line))
		mid.Move(fyne.NewPos(originX, originY+g.blockSize/2-line/2))
		*objs = append(*objs, mid)

		vLeft := canvas.NewRectangle(lineColor)
		vLeft.Resize(fyne.NewSize(line, g.blockSize/2))
		vLeft.Move(fyne.NewPos(originX+g.blockSize*0.33, originY))
		*objs = append(*objs, vLeft)

		vRight := canvas.NewRectangle(lineColor)
		vRight.Resize(fyne.NewSize(line, g.blockSize/2))
		vRight.Move(fyne.NewPos(originX+g.blockSize*0.66, originY+g.blockSize/2))
		*objs = append(*objs, vRight)
		return
	}

	defaultWall := canvas.NewRectangle(color.RGBA{0, 0, 255, 255})
	defaultWall.Resize(fyne.NewSize(g.blockSize, g.blockSize))
	defaultWall.Move(fyne.NewPos(originX, originY))
	*objs = append(*objs, defaultWall)
}

func (g *GUIGame) monsterTeethBlinkSwap(moving bool) bool {
	if !moving {
		return false
	}

	now := time.Now()
	if g.monsterTeethBlinkLast.IsZero() {
		g.monsterTeethBlinkLast = now
		return g.monsterTeethBlink
	}
	if now.Sub(g.monsterTeethBlinkLast) >= monsterTeethBlinkInterval {
		g.monsterTeethBlink = !g.monsterTeethBlink
		g.monsterTeethBlinkLast = now
	}
	return g.monsterTeethBlink
}

func (g *GUIGame) drawMonster(x, y, size float32, blinkSwap bool) {
	bodyColor := color.RGBA{255, 0, 0, 255}
	radius := size * 0.2

	// Body with rounded top corners
	body := canvas.NewRectangle(bodyColor)
	body.Resize(fyne.NewSize(size, size-radius))
	body.Move(fyne.NewPos(x, y+radius))
	g.canvas.Add(body)

	leftTop := canvas.NewCircle(bodyColor)
	leftTop.Resize(fyne.NewSize(radius*2, radius*2))
	leftTop.Move(fyne.NewPos(x, y))
	g.canvas.Add(leftTop)

	rightTop := canvas.NewCircle(bodyColor)
	rightTop.Resize(fyne.NewSize(radius*2, radius*2))
	rightTop.Move(fyne.NewPos(x+size-radius*2, y))
	g.canvas.Add(rightTop)

	// Teeth row at about 3/4 height
	teethY := y + size*0.75
	toothHeight := size * 0.12
	toothWidth := size * 0.12
	startX := x + size*0.18
	gap := size * 0.05
	for i := 0; i < 4; i++ {
		toothColor := color.RGBA{255, 255, 255, 255}
		if i%2 == 1 {
			toothColor = color.RGBA{0, 0, 0, 255}
		}
		if blinkSwap {
			toothColor = color.RGBA{255 - toothColor.R, 255 - toothColor.G, 255 - toothColor.B, 255}
		}
		tooth := canvas.NewRectangle(toothColor)
		tooth.Resize(fyne.NewSize(toothWidth, toothHeight))
		tooth.Move(fyne.NewPos(startX+float32(i)*(toothWidth+gap), teethY))
		g.canvas.Add(tooth)
	}
}

func (g *GUIGame) capturePositions() (renderPos, []renderPos) {
	playerPos := renderPos{x: float32(g.game.Player.X), y: float32(g.game.Player.Y)}
	monsterPos := make([]renderPos, len(g.game.Monsters))
	for i, monster := range g.game.Monsters {
		monsterPos[i] = renderPos{x: float32(monster.X), y: float32(monster.Y)}
	}
	return playerPos, monsterPos
}

func (g *GUIGame) animateMovement(infoLabel *widget.Label, startPlayer, endPlayer renderPos, startMonsters, endMonsters []renderPos) {
	steps := 2 // Reduced from 4 to cut rendering work in half
	stepDuration := g.tickInterval / time.Duration(steps)
	if stepDuration < 10*time.Millisecond {
		stepDuration = 10 * time.Millisecond
	}

	// Do animation synchronously but quickly
	for i := 1; i <= steps; i++ {
		progress := float32(i) / float32(steps)
		playerPos := renderPos{
			x: startPlayer.x + (endPlayer.x-startPlayer.x)*progress,
			y: startPlayer.y + (endPlayer.y-startPlayer.y)*progress,
		}

		monsterPos := make([]renderPos, len(endMonsters))
		for idx := range endMonsters {
			start := renderPos{}
			if idx < len(startMonsters) {
				start = startMonsters[idx]
			}
			monsterPos[idx] = renderPos{
				x: start.x + (endMonsters[idx].x-start.x)*progress,
				y: start.y + (endMonsters[idx].y-start.y)*progress,
			}
		}

		fyne.DoAndWait(func() {
			g.renderGameAt(infoLabel, playerPos, monsterPos)
		})

		// Small sleep between frames
		if i < steps {
			time.Sleep(stepDuration)
		}
	}
}

func (g *GUIGame) drawPacman(x, y, size float32, dir actors.Direction) {
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
	case actors.Right:
		directionAngle = 0
	case actors.Down:
		directionAngle = math.Pi / 2
	case actors.Left:
		directionAngle = math.Pi
	case actors.Up:
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

	// Draw eye (two-thirds from left, one-third from top)
	eyeSize := size * 0.12
	eyeX := x + size*(2.0/3.0) - eyeSize/2
	eyeY := y + size*(1.0/3.0) - eyeSize/2
	eye := canvas.NewCircle(color.RGBA{180, 180, 180, 255})
	eye.Resize(fyne.NewSize(eyeSize, eyeSize))
	eye.Move(fyne.NewPos(eyeX, eyeY))
	g.canvas.Add(eye)
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
		}
	}()
}

func (g *GUIGame) renderGameWithCountdown(infoLabel *widget.Label) {
	// Render the same as normal game
	g.renderGame(infoLabel)

	// Add countdown display in center
	elapsed := time.Since(g.countdownStart).Seconds()
	if elapsed < 3 {
		remaining := int(3 - elapsed)
		countdownText := canvas.NewText(fmt.Sprintf("%d", remaining), color.RGBA{255, 255, 255, 255})
		countdownText.TextSize = 48
		countdownText.Alignment = fyne.TextAlignCenter

		m := g.game.CurrentMap
		mapOriginX, mapOriginY := g.mapOrigin()
		centerX := mapOriginX + float32(m.Width)*g.blockSize/2
		centerY := mapOriginY + float32(m.Height)*g.blockSize/2

		countdownText.Move(fyne.NewPos(centerX-40, centerY-40))
		g.canvas.Add(countdownText)
		g.canvas.Refresh()
	}
}

func (g *GUIGame) calculateBlockSize() {
	if g.game == nil || g.game.CurrentMap == nil || g.window == nil {
		return
	}

	m := g.game.CurrentMap
	canvasSize := g.window.Canvas().Size()

	// Account for status bar (~80 pixels)
	availHeight := canvasSize.Height - g.currentStatusBarHeight()
	availWidth := canvasSize.Width

	// Calculate block size based on map dimensions to fill available space
	blockSizeByHeight := availHeight / float32(m.Height+borderBlocks*2)
	blockSizeByWidth := availWidth / float32(m.Width+borderBlocks*2)

	// Use the smaller to fit the entire map in window
	g.blockSize = blockSizeByHeight
	if blockSizeByWidth < blockSizeByHeight {
		g.blockSize = blockSizeByWidth
	}

	// Cap to maximum block size
	if g.blockSize > maxBlockSize {
		g.blockSize = maxBlockSize
	}

	// Clamp to reasonable minimum
	if g.blockSize < minBlockSize {
		g.blockSize = minBlockSize
	}

	// Invalidate cache if block size changed
	g.cachedMapRender = nil

	// Calculate actual map dimensions in pixels
	mapPixelWidth := float32(m.Width+borderBlocks*2) * g.blockSize
	mapPixelHeight := float32(m.Height+borderBlocks*2) * g.blockSize

	// Calculate offsets to center the game if there's extra space
	g.offsetX = 0
	g.offsetY = 0

	if mapPixelWidth < availWidth {
		// Center horizontally
		g.offsetX = (availWidth - mapPixelWidth) / 2
	}

	if mapPixelHeight < availHeight {
		// Center vertically (within available space after status bar)
		g.offsetY = (availHeight - mapPixelHeight) / 2
	}
}

func (g *GUIGame) handleKeyPress(ev *fyne.KeyEvent, infoLabel *widget.Label) {
	if ev.Name == fyne.KeyEscape {
		g.showSettings()
		return
	}
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
			g.game.Player.SetDirection(actors.Up)
		}
	case fyne.KeyDown:
		if g.state == StatePlaying || g.state == StateLevelStart || g.state == StateLevelComplete {
			g.game.Player.SetDirection(actors.Down)
		}
	case fyne.KeyLeft:
		if g.state == StatePlaying || g.state == StateLevelStart || g.state == StateLevelComplete {
			g.game.Player.SetDirection(actors.Left)
		}
	case fyne.KeyRight:
		if g.state == StatePlaying || g.state == StateLevelStart || g.state == StateLevelComplete {
			g.game.Player.SetDirection(actors.Right)
		}
	case fyne.KeyF2:
		g.handleF2NewGame()
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

func (g *GUIGame) handleF2NewGame() {
	// If game is in progress, show confirmation dialog
	if g.state == StatePlaying || g.state == StateLevelStart || g.state == StateLevelComplete {
		msg := widget.NewLabel("Start a new game? Current progress will be lost.")
		msg.Wrapping = fyne.TextWrapWord
		content := container.NewVBox(msg)

		var d *dialog.ConfirmDialog
		handleChoice := func(ok bool) {
			if ok {
				g.restartGame()
			}
			// Ensure focus returns to keyCatcher after dialog closes
			if g.keyCatcher != nil {
				g.window.Canvas().Focus(g.keyCatcher)
			}
		}

		key := newKeyCatcher(func(ev *fyne.KeyEvent) {
			if ev.Name == fyne.KeyEscape {
				handleChoice(false)
				if d != nil {
					d.Hide()
				}
			} else if ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter {
				handleChoice(true)
				if d != nil {
					d.Hide()
				}
			}
		})
		stack := container.NewStack(content, key)

		d = dialog.NewCustomConfirm("New Game", "Yes", "No", stack, handleChoice, g.window)

		originalHandler := g.window.Canvas().OnTypedKey()
		g.window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
			if ev.Name == fyne.KeyEscape {
				handleChoice(false)
				d.Hide()
				if originalHandler != nil {
					g.window.Canvas().SetOnTypedKey(originalHandler)
				}
			} else if ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter {
				handleChoice(true)
				d.Hide()
				if originalHandler != nil {
					g.window.Canvas().SetOnTypedKey(originalHandler)
				}
			}
		})

		d.SetOnClosed(func() {
			if originalHandler != nil {
				g.window.Canvas().SetOnTypedKey(originalHandler)
			}
		})

		d.Show()
		g.window.Canvas().Focus(key)
	} else {
		// Game is over or won, just restart
		g.restartGame()
	}
}

func (g *GUIGame) restartGame() {
	if g.ticker != nil {
		g.ticker.Stop()
	}
	g.state = StatePlaying
	g.startGame()
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

		tickCount := 0

		for range g.ticker.C {
			if g.game.GameOver {
				g.ticker.Stop()
				g.state = StateGameOver
				return
			}

			if g.game.Won {
				g.ticker.Stop()
				g.state = StateWon
				fyne.Do(func() {
					msg := widget.NewLabel(fmt.Sprintf("Final Score: %d", g.game.Score))
					msg.Alignment = fyne.TextAlignCenter
					content := container.NewVBox(msg)

					var d dialog.Dialog
					key := newKeyCatcher(func(ev *fyne.KeyEvent) {
						if ev.Name == fyne.KeyEscape || ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter {
							if d != nil {
								d.Hide()
							}
						}
					})
					stack := container.NewStack(content, key)

					d = dialog.NewCustom("You Won!", "OK", stack, g.window)

					originalHandler := g.window.Canvas().OnTypedKey()
					g.window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
						if ev.Name == fyne.KeyEscape || ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter {
							d.Hide()
							if originalHandler != nil {
								g.window.Canvas().SetOnTypedKey(originalHandler)
							}
						}
					})

					d.SetOnClosed(func() {
						if originalHandler != nil {
							g.window.Canvas().SetOnTypedKey(originalHandler)
						}
					})

					d.Show()
					g.window.Canvas().Focus(key)
				})
				return
			}

			// Periodically ensure keyCatcher has focus
			tickCount++
			if tickCount%20 == 0 && g.keyCatcher != nil {
				fyne.Do(func() {
					if g.window != nil && g.window.Canvas() != nil {
						g.window.Canvas().Focus(g.keyCatcher)
					}
				})
			}

			// Handle level start countdown phase
			if g.state == StateLevelStart {
				if time.Since(g.countdownStart) < 3*time.Second {
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
				g.game.LoadLevel(g.game.CurrentLevel + 1)
				g.cachedMapRender = nil // Invalidate cache for new level
				g.state = StateLevelStart
				g.countdownStart = time.Now()
				g.pauseTicks = 0
				continue
			}

			// Only update game during StatePlaying
			if g.state != StatePlaying {
				fyne.DoAndWait(func() {
					g.renderGame(g.infoLabel)
				})
				continue
			}

			startPlayer, startMonsters := g.capturePositions()
			g.game.Update()
			endPlayer, endMonsters := g.capturePositions()

			pendingDot := false
			dotX, dotY := 0, 0
			if g.game.DotEaten {
				dotX, dotY = g.game.Player.X, g.game.Player.Y
				pendingDot = true
			}

			// Mouth animation: smooth open/close on dot eat
			if g.game.DotEaten {
				g.startMouthAnimation()
			}

			lifeLost := g.game.LifeLost
			levelCompleted := g.game.LevelCompleted

			// Restart countdown if a life was lost
			if lifeLost {
				g.countdownStart = time.Now()
			}

			// Check if level is completed
			if levelCompleted {
				g.game.LevelCompleted = false

				// Check if all levels are done
				if g.game.CurrentLevel+1 >= len(g.game.Maps) {
					g.game.Won = true
				} else {
					g.state = StateLevelComplete
					g.pauseTicks = 2 // 2 tick pause
				}
			}

			if lifeLost || levelCompleted || g.game.GameOver || g.game.Won {
				fyne.DoAndWait(func() {
					g.renderGame(g.infoLabel)
				})
				continue
			}

			// Keep the dot visible during movement animation, then remove it at the end
			if pendingDot {
				if dotY >= 0 && dotY < g.game.CurrentMap.Height && dotX >= 0 && dotX < g.game.CurrentMap.Width {
					if g.game.CurrentMap.Cells[dotY][dotX] == maps.Empty {
						g.game.CurrentMap.Cells[dotY][dotX] = maps.Dot
					}
				}
			}

			g.animateMovement(g.infoLabel, startPlayer, endPlayer, startMonsters, endMonsters)

			if pendingDot {
				g.game.CurrentMap.EatDot(dotX, dotY)
			}
		}
	}()
}
