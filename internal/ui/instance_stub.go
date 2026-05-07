//go:build !windows

package ui

type singleInstanceGuard struct{}

func acquireSingleInstance(allowMultiple bool, elevatedChild bool) (*singleInstanceGuard, error) {
	return &singleInstanceGuard{}, nil
}

func (g *singleInstanceGuard) Release() {}
