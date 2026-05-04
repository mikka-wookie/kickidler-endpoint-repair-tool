package detector

import "testing"

func TestIsPathCoveredByExclusion(t *testing.T) {
	tests := []struct {
		name      string
		required  string
		exclusion string
		want      bool
	}{
		{
			name:      "exact match",
			required:  `C:\Program Files\TeleLinkSoft\bin`,
			exclusion: `C:\Program Files\TeleLinkSoft\bin`,
			want:      true,
		},
		{
			name:      "trailing slash",
			required:  `C:\Program Files\TeleLinkSoft\bin`,
			exclusion: `C:\Program Files\TeleLinkSoft\bin\`,
			want:      true,
		},
		{
			name:      "case-insensitive",
			required:  `C:\Program Files\TeleLinkSoft\bin`,
			exclusion: `c:\program files\telelinksoft\BIN`,
			want:      true,
		},
		{
			name:      "parent covers child",
			required:  `C:\Program Files\TeleLinkSoft\bin`,
			exclusion: `C:\Program Files\TeleLinkSoft`,
			want:      true,
		},
		{
			name:      "child does not cover parent",
			required:  `C:\Program Files\TeleLinkSoft`,
			exclusion: `C:\Program Files\TeleLinkSoft\bin`,
			want:      false,
		},
		{
			name:      "boundary protection",
			required:  `C:\Program Files\TeleLinkSoft2`,
			exclusion: `C:\Program Files\TeleLinkSoft`,
			want:      false,
		},
		{
			name:      "wmi parent covers bin",
			required:  `C:\Windows\System32\wmi\bin`,
			exclusion: `C:\Windows\System32\wmi`,
			want:      true,
		},
		{
			name:      "whitespace and quotes",
			required:  `C:\Program Files\TeleLinkSoft\bin`,
			exclusion: `" C:\Program Files\TeleLinkSoft\bin "`,
			want:      true,
		},
		{
			name:      "forward slash normalization",
			required:  `C:\Program Files\TeleLinkSoft\bin`,
			exclusion: `C:/Program Files/TeleLinkSoft/bin`,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPathCoveredByExclusion(tt.required, tt.exclusion)
			if got != tt.want {
				t.Fatalf("IsPathCoveredByExclusion(%q, %q) = %v, want %v", tt.required, tt.exclusion, got, tt.want)
			}
		})
	}
}

func TestNormalizeWindowsPath(t *testing.T) {
	t.Setenv("ProgramFiles", `C:\Program Files`)

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "trims whitespace quotes and trailing slash",
			path: `" C:\Program Files\TeleLinkSoft\bin\ "`,
			want: `C:\Program Files\TeleLinkSoft\bin`,
		},
		{
			name: "expands environment variables and forward slashes",
			path: `%ProgramFiles%/TeleLinkSoft/bin`,
			want: `C:\Program Files\TeleLinkSoft\bin`,
		},
		{
			name: "normalizes long path prefix",
			path: `\\?\C:\Program Files\TeleLinkSoft\bin`,
			want: `C:\Program Files\TeleLinkSoft\bin`,
		},
		{
			name: "keeps drive root slash",
			path: `C:\`,
			want: `C:\`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeWindowsPath(tt.path)
			if got != tt.want {
				t.Fatalf("NormalizeWindowsPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestParseDefenderExclusionJSONTreatsAdminPlaceholderAsUnavailable(t *testing.T) {
	_, err := parseDefenderExclusionJSON(`["N/A: Must be an administrator to view exclusions"]`)
	if err == nil {
		t.Fatal("expected unavailable error")
	}
}

func TestStandardBinInstallDefenderCoverageAcceptsBinOrRootExclusion(t *testing.T) {
	tests := []struct {
		name      string
		exclusion string
	}{
		{
			name:      "bin exclusion",
			exclusion: `C:\Program Files\TeleLinkSoft\bin`,
		},
		{
			name:      "root exclusion",
			exclusion: `C:\Program Files\TeleLinkSoft`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := standardBinReport(true, []string{tt.exclusion})
			enriched := EnrichDetectionReport(report)

			if len(enriched.Defender.CoveredPaths) != 1 || enriched.Defender.CoveredPaths[0] != `C:\Program Files\TeleLinkSoft\bin` {
				t.Fatalf("covered paths = %#v", enriched.Defender.CoveredPaths)
			}
			if len(enriched.Defender.MissingPaths) != 0 {
				t.Fatalf("missing paths = %#v", enriched.Defender.MissingPaths)
			}
			if len(enriched.MissingDefenderPaths) != 0 {
				t.Fatalf("top-level missing paths = %#v", enriched.MissingDefenderPaths)
			}
		})
	}
}

func TestStandardRootRequirementDoesNotAcceptBinExclusion(t *testing.T) {
	report := standardTLSReport(true, []string{`C:\Program Files\TeleLinkSoft\bin`})
	enriched := EnrichDetectionReport(report)

	if len(enriched.Defender.CoveredPaths) != 0 {
		t.Fatalf("covered paths = %#v", enriched.Defender.CoveredPaths)
	}
	if len(enriched.Defender.MissingPaths) != 1 || enriched.Defender.MissingPaths[0] != `C:\Program Files\TeleLinkSoft` {
		t.Fatalf("missing paths = %#v", enriched.Defender.MissingPaths)
	}
}

func standardBinReport(serviceExecutableExists bool, exclusions []string) DetectionReport {
	report := baseReport()
	report.Services = []ServiceState{
		{Name: "ngs", Exists: true, Status: "running", ImagePath: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`},
		{Name: "tls", Exists: false},
		{Name: wmiProviderService, Exists: false},
	}
	report.Files = []FileState{
		{Path: `C:\Program Files\TeleLinkSoft\bin`, Exists: true, Type: "folder"},
		{Path: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`, Exists: serviceExecutableExists, Type: "file"},
	}
	report.Defender = DefenderState{Available: true, ExclusionPaths: exclusions}
	return report
}
