package repl

import (
	"context"
	"io"
	"time"

	"github.com/ikigenba/agentkit/openai/subscription"
)

type IO struct {
	In       io.Reader
	Out, Err io.Writer
	IsTTY    bool
}

type Getenv func(string) string

type Now func() time.Time

type Waiter interface {
	Start(model string)
	Stop()
}

type nopWaiter struct{}

func (nopWaiter) Start(string) {}

func (nopWaiter) Stop() {}

// Options is the parsed launch surface.
type Options struct {
	Config []string
	Raw    bool
}

// Deps are the composition-root dependencies Run needs.
type Deps struct {
	IO       IO
	Getenv   func(string) string
	Now      func() time.Time
	Waiter   Waiter
	LogDir   string
	AuthFile string
	Login    func(context.Context, string, subscription.LoginIO) error
}
