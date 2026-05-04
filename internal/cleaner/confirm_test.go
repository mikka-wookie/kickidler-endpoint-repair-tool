package cleaner

import (
	"errors"
	"strings"
	"testing"

	"kigrepair/internal/app"
)

func TestConfirmCleanupNonInteractiveWithoutYesRequiresYes(t *testing.T) {
	ctx := &app.AppContext{NonInteractive: true}

	err := ConfirmCleanup(ctx, false, strings.NewReader("YES\n"), nil)
	if !errors.Is(err, ErrConfirmationRequired) {
		t.Fatalf("ConfirmCleanup error = %v, want ErrConfirmationRequired", err)
	}
}

func TestConfirmCleanupNonInteractiveWithYesProceeds(t *testing.T) {
	ctx := &app.AppContext{NonInteractive: true}

	if err := ConfirmCleanup(ctx, true, strings.NewReader(""), nil); err != nil {
		t.Fatalf("ConfirmCleanup returned error: %v", err)
	}
}

func TestConfirmCleanupInteractiveDeclined(t *testing.T) {
	ctx := &app.AppContext{}

	err := ConfirmCleanup(ctx, false, strings.NewReader("yes\n"), nil)
	if !errors.Is(err, ErrConfirmationDeclined) {
		t.Fatalf("ConfirmCleanup error = %v, want ErrConfirmationDeclined", err)
	}
}
