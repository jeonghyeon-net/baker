package app

import (
	"context"
	"os"
)

type Mode string

const (
	ModeNeedsSource Mode = "needs-source"
	ModeInteractive Mode = "interactive"
)

type Result struct {
	Mode    Mode
	Message string
}

type Options struct {
	ResultFile string
}

type ShellInstaller interface {
	Ensure() (ready bool, message string, err error)
}

type Application struct {
	Shell ShellInstaller
}

func (a Application) Run(ctx context.Context, opts Options) (Result, error) {
	_ = ctx
	_ = opts
	if a.Shell == nil {
		return Result{Mode: ModeInteractive}, nil
	}

	ready, message, err := a.Shell.Ensure()
	if err != nil {
		return Result{}, err
	}
	if !ready {
		return Result{Mode: ModeNeedsSource, Message: message}, nil
	}
	return Result{Mode: ModeInteractive}, nil
}

func WriteShellResult(path string, target string) error {
	return os.WriteFile(path, []byte(target), 0o644)
}
