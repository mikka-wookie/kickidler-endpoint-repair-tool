package installer

import (
	"strings"
	"testing"
)

func TestInviteMasking(t *testing.T) {
	invite := "SECRETINVITE123"
	if got := strings.Join(MaskedMSIInstallArgs(`C:\grabber.msi`, `C:\msi.log`), " "); strings.Contains(got, invite) || !strings.Contains(got, "invite=***") {
		t.Fatalf("masked args = %q", got)
	}
	if got := MaskInviteInText("failed command invite=SECRETINVITE123", invite); strings.Contains(got, invite) || !strings.Contains(got, "invite=***") {
		t.Fatalf("masked text = %q", got)
	}
}
