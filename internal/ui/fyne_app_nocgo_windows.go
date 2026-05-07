//go:build windows && !cgo && !oldgui

package ui

import (
	"errors"
	"fmt"
)

type windowSize struct {
	Width  float32
	Height float32
}

func runGUI() error {
	return errors.New("kigrepair GUI requires CGO and a C compiler because the default frontend uses Fyne")
}

func MinWindowSize() windowSize {
	return windowSize{Width: guiMinWidth, Height: guiMinHeight}
}

func capTimeline(items []TimelineItem, limit int) []TimelineItem {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	out := append([]TimelineItem(nil), items[:limit-1]...)
	out = append(out, TimelineItem{Step: "timeline-summary", Status: "info", Message: fmt.Sprintf("%d additional events saved in report files.", len(items)-len(out))})
	return out
}
