package sash

import "github.com/zackarysantana/sash/src/runner"

type Options = runner.Options

type Runner = runner.Runner

type StubRunner = runner.StubRunner

func Run(opts Options) error {
	return runner.Run(opts)
}
