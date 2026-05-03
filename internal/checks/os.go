package checks

import "runtime"

func OSName() string {
	return runtime.GOOS
}
