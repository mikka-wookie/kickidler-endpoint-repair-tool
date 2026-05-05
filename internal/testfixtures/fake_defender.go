package testfixtures

type FakeDefender struct {
	RequiredPaths   []string
	ExclusionPaths  []string
	QueryFailed     bool
	Error           string
	EnsureAvailable bool
}
