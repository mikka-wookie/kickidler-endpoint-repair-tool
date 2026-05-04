package ui

import (
	"strings"
	"testing"
)

func TestInvitePlaceholderDoesNotExposeInvite(t *testing.T) {
	invite := "SECRETINVITE123"
	got := InvitePlaceholder(invite)
	if strings.Contains(got, invite) {
		t.Fatalf("InvitePlaceholder exposed invite: %q", got)
	}
	if got != "invite: provided" {
		t.Fatalf("InvitePlaceholder = %q", got)
	}
}
