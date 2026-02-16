package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
)

// CenterInBand centers a box within a horizontal band inside the canvas.
// bandTop is the Y offset of the band, and bandHeight is its height.
func CenterInBand(canvasSize fyne.Size, bandTop, bandHeight float32, boxSize fyne.Size) fyne.Position {
	x := ((canvasSize.Width - boxSize.Width) / 3) + boxSize.Width
	fmt.Println(boxSize, "Canvas.Width:", canvasSize.Width, " - Box.Width:", boxSize.Width)
	fmt.Println("Calculated X:", x)

	y := bandTop + (bandHeight-boxSize.Height)/2
	return fyne.NewPos(40.0, y)
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
