package safety

type ActionKind string

const (
	ActionReadOnly ActionKind = "read_only"
	ActionSkipped  ActionKind = "skipped"
)
