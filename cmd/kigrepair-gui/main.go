//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"kigrepair/internal/app"
	"kigrepair/internal/safety"
	"kigrepair/internal/ui"
)

func main() {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintf(os.Stderr, "unexpected GUI error: %s\n", safety.RedactString(fmt.Sprint(recovered)))
			if os.Getenv("KIGREPAIR_DEBUG") == "1" {
				fmt.Fprintln(os.Stderr, safety.RedactString(string(debug.Stack())))
			}
			os.Exit(app.ExitUnexpectedError)
		}
	}()
	if err := ui.Run(); err != nil {
		fmt.Fprintln(os.Stderr, safety.RedactString(err.Error()))
		os.Exit(app.ExitUnexpectedError)
	}
}
