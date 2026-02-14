//go:build nogui
// +build nogui

package ui

import "fmt"

func RunGUIGame(mapFile string, disableMonsters bool) error {
	return fmt.Errorf("GUI mode not available in this build. Rebuild with: go build -tags gui")
}
