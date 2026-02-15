//go:build !nogui
// +build !nogui

package ui

import "time"

const (
	defaultBlockSize          = 20
	minBlockSize              = 10
	maxBlockSize              = 50
	minWindowSize             = 640
	statusBarHeight           = 80
	defaultTickInterval       = 220 * time.Millisecond
	monsterTeethBlinkInterval = 150 * time.Millisecond
	borderBlocks              = 1
)
