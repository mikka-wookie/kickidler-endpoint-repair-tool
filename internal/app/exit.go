package app

type ExitError struct {
	Code int
}

func (e ExitError) Error() string {
	return ""
}
