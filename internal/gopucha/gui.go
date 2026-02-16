//go:build !nogui
// +build !nogui

package gopucha

import (
	"fmt"
	"image/color"
	"math"
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
	"fyne.io/fyne/v2/widget"
)

const (
	defaultBlockSize = 20
	minBlockSize     = 10
	maxBlockSize     = 50
	minWindowSize    = 640
	statusBarHeight  = 80
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
	app             fyne.App
	window          fyne.Window
	game            *Game
	blockSize       float32
	canvas          *fyne.Container
	keyCatcher      *keyCatcher
	ticker          *time.Ticker
	tickInterval    time.Duration
	mapFile         string
	infoLabel       *widget.Label
	controlsLabel   *widget.Label
	state           GameState
	countdownTicks  int
	pauseTicks      int
	tickerDone      chan bool
	mouthOpen       bool
	mouthOpenRatio  float64
	mouthAnimDir    int
	mouthTicker     *time.Ticker
	disableMonsters bool
}

type renderPos struct {
	x float32
	y float32
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
		app:             app.New(),
		blockSize:       defaultBlockSize,
		tickInterval:    260 * time.Millisecond,
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
		if g.state == StatePlaying && g.game != nil {
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
		}
	})
	stack := container.NewStack(content, key)

	d = dialog.NewCustomConfirm("Settings", "Apply", "Cancel", stack, func(apply bool) {
		handleClose(apply)
	}, g.window)
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

	if len(mapFiles) <= 1 {
		d := dialog.NewCustom("Map Error", "Exit", stack, g.window)
		d.SetOnClosed(func() {
			g.window.Close()
		})
		d.Show()
		g.window.Canvas().Focus(key)
		return
	}

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

	// Info panel with styled background
	g.infoLabel = widget.NewLabel(fmt.Sprintf("Level: %d | Score: %d | Lives: %d | Dots: %d",
		g.game.CurrentLevel+1, g.game.Score, g.game.Lives, g.game.CurrentMap.CountDots()))
	g.infoLabel.TextStyle = fyne.TextStyle{Bold: true}

	g.controlsLabel = widget.NewLabel("Controls: Arrow Keys to move | F2 restart | +/- zoom | ESC settings")
	g.controlsLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Create styled status bar background
	statusBarBg := canvas.NewRectangle(color.RGBA{40, 40, 50, 255})
	g.infoLabel.Importance = widget.HighImportance

	// Create status bar with padding
	infoBox := container.NewVBox(g.infoLabel, g.controlsLabel)
	topBar := container.NewPadded(infoBox)
	statusBar := container.NewStack(statusBarBg, topBar)

	// Main container with scroll
	scroll := container.NewScroll(g.canvas)

	// Key capture overlay to ensure arrow keys are received reliably
	g.initControls()
	gameArea := container.NewStack(scroll, g.keyCatcher)

	content := container.NewBorder(statusBar, nil, nil, nil, gameArea)
	minRect := canvas.NewRectangle(color.Transparent)
	minRect.Resize(fyne.NewSize(minWindowSize, minWindowSize))
	contentWithMin := container.NewMax(minRect, content)

	g.window.SetContent(contentWithMin)
	g.resizeWindowForMap()

	// Focus key catcher so arrow keys (including Up) are not swallowed by scroll
	g.window.Canvas().Focus(g.keyCatcher)

	// Ensure keyCatcher stays focused
	g.keyCatcher.Refresh()

	// Set up mouse scroll for zoom (CTRL+Scroll)
	// Note: Fyne doesn't directly support scroll events, so we'll rely on keyboard shortcuts

	// Render initial state
	g.renderGame(g.infoLabel)
	g.calculateBlockSize()
}

func (g *GUIGame) resizeWindowForMap() {
	if g.game == nil || g.game.CurrentMap == nil || g.window == nil {
		return
	}

	mapWidth := float32(g.game.CurrentMap.Width) * g.blockSize
	mapHeight := float32(g.game.CurrentMap.Height)*g.blockSize + statusBarHeight

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
			g.canvas.Add(rect)

			switch m.Cells[y][x] {
			case Wall:
				g.drawWallCell(x, y, m)
			case Dot:
				dotSize := g.blockSize * 0.35
				dot := canvas.NewCircle(color.RGBA{255, 230, 0, 255}) // Yellow fill
				dot.StrokeColor = color.RGBA{180, 90, 0, 255}         // Dark orange border
				dot.StrokeWidth = dotSize * 0.2
				dot.Resize(fyne.NewSize(dotSize, dotSize))
				dot.Move(fyne.NewPos(float32(x)*g.blockSize+(g.blockSize-dotSize)/2, float32(y)*g.blockSize+(g.blockSize-dotSize)/2))
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
		g.drawMonster(pos.x*g.blockSize+g.blockSize*0.1, pos.y*g.blockSize+g.blockSize*0.1, g.blockSize*0.8)
	}

	// Render player (yellow semi-circle with mouth)
	g.drawPacman(playerPos.x*g.blockSize+g.blockSize*0.05, playerPos.y*g.blockSize+g.blockSize*0.05, g.blockSize*0.9, g.game.Player.Direction)

	// Update info
	infoLabel.SetText(fmt.Sprintf("Level: %d | Score: %d | Lives: %d | Dots: %d | MapSize: %dx%d",
		g.game.CurrentLevel+1, g.game.Score, g.game.Lives, g.game.CurrentMap.CountDots(), m.Width, m.Height))

	// Show/hide controls based on state
	g.updateControlsVisibility()

	// Render warning overlay on top of the game.
	if g.state == StateGameOver {
		box := g.newWarningBox("Game over\nPress arrow to start again", false, canvasWidth*0.7)
		box.Move(fyne.NewPos((canvasWidth-box.Size().Width)/2, (canvasHeight-box.Size().Height)/2))
		g.canvas.Add(box)
	} else if g.state == StateWon {
		message := fmt.Sprintf("You won!\nFinal score: %d\nPress arrow to start again", g.game.Score)
		box := g.newWarningBox(message, false, canvasWidth*0.7)
		box.Move(fyne.NewPos((canvasWidth-box.Size().Width)/2, (canvasHeight-box.Size().Height)/2))
		g.canvas.Add(box)
	}

	g.canvas.Refresh()
}

func (g *GUIGame) newWarningBox(text string, showControls bool, width float32) *fyne.Container {
	padding := float32(16)
	if g.blockSize > 0 {
		padding = g.blockSize * 0.6
		if padding < 12 {
			padding = 12
		}
	}

	lines := strings.Split(text, "\n")
	labels := make([]fyne.CanvasObject, 0, len(lines)+1)
	for i, line := range lines {
		label := widget.NewLabel(line)
		label.Alignment = fyne.TextAlignCenter
		if i == 0 {
			label.TextStyle = fyne.TextStyle{Bold: true}
		}
		labels = append(labels, label)
	}
	if showControls {
		controls := widget.NewLabel("OK / Cancel")
		controls.Alignment = fyne.TextAlignCenter
		controls.TextStyle = fyne.TextStyle{Italic: true}
		labels = append(labels, controls)
	}

	content := container.NewVBox(labels...)
	centered := container.NewCenter(content)
	contentSize := centered.MinSize()

	if width < contentSize.Width+padding*2 {
		width = contentSize.Width + padding*2
	}
	boxHeight := contentSize.Height + padding*2

	box := container.NewWithoutLayout()
	box.Resize(fyne.NewSize(width, boxHeight))

	bg := canvas.NewRectangle(color.RGBA{20, 20, 20, 235})
	bg.Resize(fyne.NewSize(width, boxHeight))
	box.Add(bg)

	stripeColor := color.RGBA{245, 245, 245, 255}
	stripeThickness := float32(2)
	stripeLen := float32(8)
	if g.blockSize > 0 {
		stripeLen = g.blockSize * 0.5
		if stripeLen < 6 {
			stripeLen = 6
		}
	}

	for x := float32(0); x < width; x += stripeLen * 2 {
		seg := stripeLen
		if x+seg > width {
			seg = width - x
		}
		top := canvas.NewRectangle(stripeColor)
		top.Resize(fyne.NewSize(seg, stripeThickness))
		top.Move(fyne.NewPos(x, 0))
		box.Add(top)

		bottom := canvas.NewRectangle(stripeColor)
		bottom.Resize(fyne.NewSize(seg, stripeThickness))
		bottom.Move(fyne.NewPos(x, boxHeight-stripeThickness))
		box.Add(bottom)
	}

	for y := float32(0); y < boxHeight; y += stripeLen * 2 {
		seg := stripeLen
		if y+seg > boxHeight {
			seg = boxHeight - y
		}
		left := canvas.NewRectangle(stripeColor)
		left.Resize(fyne.NewSize(stripeThickness, seg))
		left.Move(fyne.NewPos(0, y))
		box.Add(left)

		right := canvas.NewRectangle(stripeColor)
		right.Resize(fyne.NewSize(stripeThickness, seg))
		right.Move(fyne.NewPos(width-stripeThickness, y))
		box.Add(right)
	}

	centered.Resize(fyne.NewSize(width, boxHeight))
	box.Add(centered)

	return box
}

func (g *GUIGame) updateControlsVisibility() {
	if g.controlsLabel == nil {
		return
	}

	// Only show controls during countdown/level start
	if g.state == StateLevelStart && g.countdownTicks > 0 {
		g.controlsLabel.SetText("Controls: Arrow Keys to move | F2 restart | +/- zoom | ESC settings")
	} else {
		g.controlsLabel.SetText("")
	}
}

func (g *GUIGame) drawWallCell(x, y int, m *Map) {
	originX := float32(x) * g.blockSize
	originY := float32(y) * g.blockSize
	line := g.blockSize * 0.08
	if line < 1 {
		line = 1
	}

	mat := strings.ToLower(strings.TrimSpace(m.Material))
	hasTop := y-1 >= 0 && m.Cells[y-1][x] == Wall
	hasBottom := y+1 < m.Height && m.Cells[y+1][x] == Wall
	hasLeft := x-1 >= 0 && m.Cells[y][x-1] == Wall
	hasRight := x+1 < m.Width && m.Cells[y][x+1] == Wall
	if mat == "brick" || mat == "bricks" {
		base := canvas.NewRectangle(color.RGBA{160, 75, 25, 255})
		base.Resize(fyne.NewSize(g.blockSize, g.blockSize))
		base.Move(fyne.NewPos(originX, originY))
		g.canvas.Add(base)

		lineColor := color.RGBA{30, 20, 10, 255}
		if !hasTop {
			top := canvas.NewRectangle(lineColor)
			top.Resize(fyne.NewSize(g.blockSize, line))
			top.Move(fyne.NewPos(originX, originY))
			g.canvas.Add(top)
		}

		if !hasBottom {
			bottom := canvas.NewRectangle(lineColor)
			bottom.Resize(fyne.NewSize(g.blockSize, line))
			bottom.Move(fyne.NewPos(originX, originY+g.blockSize-line))
			g.canvas.Add(bottom)
		}

		mid := canvas.NewRectangle(lineColor)
		mid.Resize(fyne.NewSize(g.blockSize, line))
		mid.Move(fyne.NewPos(originX, originY+g.blockSize/2-line/2))
		g.canvas.Add(mid)

		vLeft := canvas.NewRectangle(lineColor)
		vLeft.Resize(fyne.NewSize(line, g.blockSize/2))
		vLeft.Move(fyne.NewPos(originX+g.blockSize*0.33, originY))
		g.canvas.Add(vLeft)

		vRight := canvas.NewRectangle(lineColor)
		vRight.Resize(fyne.NewSize(line, g.blockSize/2))
		vRight.Move(fyne.NewPos(originX+g.blockSize*0.66, originY+g.blockSize/2))
		g.canvas.Add(vRight)
		return
	}

	if mat == "" || mat == "classic" || mat == "steel" || mat == "metal" {
		base := canvas.NewRectangle(color.RGBA{180, 185, 195, 255})
		base.Resize(fyne.NewSize(g.blockSize, g.blockSize))
		base.Move(fyne.NewPos(originX, originY))
		g.canvas.Add(base)

		border := color.RGBA{120, 125, 135, 255}
		if !hasTop {
			top := canvas.NewRectangle(border)
			top.Resize(fyne.NewSize(g.blockSize, line))
			top.Move(fyne.NewPos(originX, originY))
			g.canvas.Add(top)
		}

		if !hasBottom {
			bottom := canvas.NewRectangle(border)
			bottom.Resize(fyne.NewSize(g.blockSize, line))
			bottom.Move(fyne.NewPos(originX, originY+g.blockSize-line))
			g.canvas.Add(bottom)
		}

		if !hasLeft {
			left := canvas.NewRectangle(border)
			left.Resize(fyne.NewSize(line, g.blockSize))
			left.Move(fyne.NewPos(originX, originY))
			g.canvas.Add(left)
		}

		if !hasRight {
			right := canvas.NewRectangle(border)
			right.Resize(fyne.NewSize(line, g.blockSize))
			right.Move(fyne.NewPos(originX+g.blockSize-line, originY))
			g.canvas.Add(right)
		}
		return
	}

	defaultWall := canvas.NewRectangle(color.RGBA{0, 0, 255, 255})
	defaultWall.Resize(fyne.NewSize(g.blockSize, g.blockSize))
	defaultWall.Move(fyne.NewPos(originX, originY))
	g.canvas.Add(defaultWall)
}

func (g *GUIGame) drawMonster(x, y, size float32) {
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
	steps := 4
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
		content := g.newWarningBox("Start a new game?\nCurrent progress will be lost.", true, 360)

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

		d = dialog.NewCustomConfirm("New Game", "OK", "Cancel", stack, handleChoice, g.window)

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
				fyne.DoAndWait(func() {
					g.renderGame(g.infoLabel)
				})
				return
			}

			if g.game.Won {
				g.ticker.Stop()
				g.state = StateWon
				fyne.DoAndWait(func() {
					g.renderGame(g.infoLabel)
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
				g.countdownTicks = 5
				g.game.LifeLost = false
				g.state = StateLevelStart
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

			if lifeLost {
				// Keep the dot visible during the bust animation, then remove it.
				if pendingDot {
					if dotY >= 0 && dotY < g.game.CurrentMap.Height && dotX >= 0 && dotX < g.game.CurrentMap.Width {
						if g.game.CurrentMap.Cells[dotY][dotX] == Empty {
							g.game.CurrentMap.Cells[dotY][dotX] = Dot
						}
					}
				}

				g.animateMovement(g.infoLabel, startPlayer, endPlayer, startMonsters, endMonsters)

				if pendingDot {
					g.game.CurrentMap.EatDot(dotX, dotY)
				}

				fyne.DoAndWait(func() {
					g.renderGame(g.infoLabel)
				})
				continue
			}

			if levelCompleted || g.game.GameOver || g.game.Won {
				fyne.DoAndWait(func() {
					g.renderGame(g.infoLabel)
				})
				continue
			}

			// Keep the dot visible during movement animation, then remove it at the end
			if pendingDot {
				if dotY >= 0 && dotY < g.game.CurrentMap.Height && dotX >= 0 && dotX < g.game.CurrentMap.Width {
					if g.game.CurrentMap.Cells[dotY][dotX] == Empty {
						g.game.CurrentMap.Cells[dotY][dotX] = Dot
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
