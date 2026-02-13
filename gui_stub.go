// +build !gui

package gopucha

import "fmt"

func RunGUIGame(mapFile string) error {
	return fmt.Errorf("GUI mode not available in this build. Rebuild with: go build -tags gui")
}
