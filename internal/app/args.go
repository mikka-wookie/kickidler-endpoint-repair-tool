package app

import "strings"

const ElevatedChildFlag = "--elevated-child"

func AppendElevatedChildArg(args []string) []string {
	if HasFlag(args, ElevatedChildFlag) {
		return append([]string(nil), args...)
	}
	childArgs := append([]string(nil), args...)
	childArgs = append(childArgs, ElevatedChildFlag)
	return childArgs
}

func HasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
	}
	return false
}

func MaskSensitiveArgs(args []string) []string {
	masked := append([]string(nil), args...)
	for i := 0; i < len(masked); i++ {
		arg := masked[i]
		if arg == "--invite" {
			if i+1 < len(masked) {
				masked[i+1] = "*****"
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "--invite=") {
			masked[i] = "--invite=*****"
		}
	}
	return masked
}

func ArgsContainInvite(args []string) bool {
	for _, arg := range args {
		if arg == "--invite" || strings.HasPrefix(arg, "--invite=") {
			return true
		}
	}
	return false
}
