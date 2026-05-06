//go:build !windows

package ui

import "fmt"

func openPath(path string) error {
	return fmt.Errorf("opening paths is only supported by the GUI on Windows: %s", path)
}
