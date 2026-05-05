//go:build !windows

package installer

import "errors"

func readMSIMetadata(path string) (msiMetadata, error) {
	return msiMetadata{}, errors.New("MSI metadata validation is only available on Windows")
}

func checkMSISignature(path string) (SignatureResult, error) {
	return SignatureResult{Checked: true, Status: "unavailable", Error: "signature validation is only available on Windows"}, errors.New("signature validation is only available on Windows")
}
