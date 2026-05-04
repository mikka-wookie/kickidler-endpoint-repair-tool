package repair

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"kigrepair/internal/app"
)

var (
	ErrRepairConfirmationRequired = errors.New("Repair in non-interactive mode requires --yes")
	ErrRepairConfirmationDeclined = errors.New("repair confirmation declined")
)

func ConfirmRepair(ctx *app.AppContext, yes bool, in io.Reader, out io.Writer) error {
	if yes {
		return nil
	}
	if ctx.Quiet || ctx.NonInteractive {
		return ErrRepairConfirmationRequired
	}
	if out != nil {
		fmt.Fprint(out, "Proceed with repair? Type YES to continue: ")
	}
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return ErrRepairConfirmationDeclined
	}
	if scanner.Text() != "YES" {
		return ErrRepairConfirmationDeclined
	}
	return nil
}
