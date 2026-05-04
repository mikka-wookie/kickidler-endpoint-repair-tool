package cleaner

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"kigrepair/internal/app"
)

var (
	ErrConfirmationRequired = errors.New("real cleanup in non-interactive mode requires --yes")
	ErrConfirmationDeclined = errors.New("cleanup confirmation declined")
)

func ConfirmCleanup(ctx *app.AppContext, yes bool, in io.Reader, out io.Writer) error {
	if yes {
		return nil
	}
	if ctx.Quiet || ctx.NonInteractive {
		return ErrConfirmationRequired
	}
	if out != nil {
		fmt.Fprint(out, "Proceed with cleanup? Type YES to continue: ")
	}
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return ErrConfirmationDeclined
	}
	if strings.TrimRight(scanner.Text(), "\r\n") != "YES" {
		return ErrConfirmationDeclined
	}
	return nil
}
