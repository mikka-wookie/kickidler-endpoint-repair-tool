package checks

import "kigrepair/internal/winapi"

func IsAdmin() bool {
	return winapi.IsAdmin()
}
