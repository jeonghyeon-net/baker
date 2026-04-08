package app

import (
	"context"
	"errors"
	osExec "os/exec"
)

type Lookup interface {
	Lookup(name string) (string, error)
}

type Preflight struct {
	Lookup Lookup
}

func (p Preflight) CheckCoreTools(ctx context.Context) error {
	_ = ctx
	if _, err := p.lookup("git"); err != nil {
		return errors.New("git is required")
	}
	return nil
}

func (p Preflight) CheckGitHubTools(ctx context.Context) error {
	_ = ctx
	if _, err := p.lookup("gh"); err != nil {
		return errors.New("gh is required for GitHub repository picker")
	}
	return nil
}

func (p Preflight) lookup(name string) (string, error) {
	if p.Lookup != nil {
		return p.Lookup.Lookup(name)
	}
	return osExec.LookPath(name)
}
