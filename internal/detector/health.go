package detector

type GrabberHealthStatus string

const (
	GrabberHealthHealthy          GrabberHealthStatus = "healthy"
	GrabberHealthBroken           GrabberHealthStatus = "broken"
	GrabberHealthPartiallyRemoved GrabberHealthStatus = "partially_removed"
	GrabberHealthNotInstalled     GrabberHealthStatus = "not_installed"
	GrabberHealthUnknown          GrabberHealthStatus = "unknown"
)
