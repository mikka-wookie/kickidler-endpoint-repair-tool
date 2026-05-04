package checks

import "os/exec"

func PowerShellAvailable() (bool, error) {
	_, err := exec.LookPath("powershell.exe")
	if err != nil {
		return false, err
	}
	return true, nil
}

func MSIExecAvailable() (bool, error) {
	_, err := exec.LookPath("msiexec.exe")
	if err != nil {
		return false, err
	}
	return true, nil
}
