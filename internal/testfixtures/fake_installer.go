package testfixtures

type FakeInstaller struct {
	Path             string
	Exists           bool
	Valid            bool
	ValidationStatus string
	Error            string
}
