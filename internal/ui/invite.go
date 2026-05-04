package ui

func InvitePlaceholder(invite string) string {
	if invite == "" {
		return "invite: missing"
	}
	return "invite: provided"
}
