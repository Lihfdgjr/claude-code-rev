package core

import "context"

type Command interface {
	Name() string
	Synopsis() string
	Run(ctx context.Context, args string, sess Session) error
}

type CommandRegistry interface {
	Get(name string) (Command, bool)
	All() []Command
}
