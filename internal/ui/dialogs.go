package ui

import "context"

func ConfirmExactYES(ctx context.Context, prompt ConfirmationRequest, entered string) (bool, error) {
	if prompt.RequiredText == "" {
		return false, nil
	}
	return AcceptExactYES(entered), nil
}
