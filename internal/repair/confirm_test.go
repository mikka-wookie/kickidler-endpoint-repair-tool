package repair

import (
	"bytes"
	"errors"
	"testing"

	"kigrepair/internal/app"
)

func TestConfirmRepair(t *testing.T) {
	tests := []struct {
		name string
		ctx  app.AppContext
		yes  bool
		in   string
		err  error
	}{
		{name: "yes proceeds", yes: true},
		{name: "non interactive without yes", ctx: app.AppContext{NonInteractive: true}, err: ErrRepairConfirmationRequired},
		{name: "quiet without yes", ctx: app.AppContext{Quiet: true}, err: ErrRepairConfirmationRequired},
		{name: "interactive accepted", in: "YES\n"},
		{name: "interactive declined", in: "yes\n", err: ErrRepairConfirmationDeclined},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ConfirmRepair(&tt.ctx, tt.yes, bytes.NewBufferString(tt.in), &bytes.Buffer{})
			if !errors.Is(err, tt.err) {
				t.Fatalf("ConfirmRepair() error = %v, want %v", err, tt.err)
			}
		})
	}
}
