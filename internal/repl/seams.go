package repl

import (
	"context"
	"io"
	"time"
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

// LoginFlow is one in-flight subscription OAuth login.
type LoginFlow interface {
	AuthorizeURL() string
	Complete(ctx context.Context, path, pastedRedirectURL string) error
}

// Options is the parsed launch surface.
type Options struct {
	Config []string
	Raw    bool
}

// Deps are the composition-root dependencies Run needs.
type Deps struct {
	IO         IO
	Getenv     func(string) string
	Now        func() time.Time
	Waiter     Waiter
	LogDir     string
	AuthFile   string
	BeginLogin func() (LoginFlow, error)
}
