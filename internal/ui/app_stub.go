//go:build !windows

package ui

import "errors"

func runGUI() error {
	return errors.New("kigrepair GUI is only supported on Windows")
}
