package ui

import (
	"fyne.io/fyne/v2"
)

// CenterInBand centers a box within a horizontal band inside the canvas.
// bandTop is the Y offset of the band, and bandHeight is its height.
func CenterInBand(canvasSize fyne.Size, bandTop, bandHeight float32, boxSize fyne.Size) fyne.Position {
	sx := canvasSize.Width - boxSize.Width
	sy := bandHeight - boxSize.Height

	x := sx * (2.2 / 7.0) - 0.4 * boxSize.Width // Center at 1/7 of the canvas width
	y := sy * (2.5 / 7.0)  // Center at 1/3 of the band height
	return fyne.NewPos(x, y)
}

// PositionInBand lays out a wrapped object inside a container, centered in a vertical band.
// It resizes the container to the canvas and the wrapped object to its min size.
func PositionInBand(container *fyne.Container, wrapped fyne.CanvasObject, canvasSize fyne.Size, bandTop, bandHeight float32) {
	container.Resize(canvasSize)
	boxSize := wrapped.MinSize()
	wrapped.Resize(boxSize)
	pos := CenterInBand(canvasSize, bandTop, bandHeight, boxSize)
	wrapped.Move(pos)
	container.Refresh()
}
