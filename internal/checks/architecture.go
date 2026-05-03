package checks

import "runtime"

func Architecture() string {
	return runtime.GOARCH
}
