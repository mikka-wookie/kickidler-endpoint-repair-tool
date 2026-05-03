package detector

import (
	"strings"
	"testing"
)

func TestKnownProcessNamesExcludeHiddenWMINames(t *testing.T) {
	for _, name := range knownProcessNames() {
		switch strings.ToLower(name) {
		case "svchost.exe", "wmiprvse.exe", "runtimebroker.exe":
			t.Fatalf("hidden WMI process name %q must only match by exact executable path", name)
		}
	}
}
