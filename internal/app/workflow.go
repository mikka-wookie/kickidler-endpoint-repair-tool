package app

type Workflow interface {
	Name() string
	Run(ctx *AppContext) error
}
