package repl

import (
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

// Options is the parsed launch surface.
type Options struct {
	Config []string
	Raw    bool
}

// Deps are the composition-root dependencies Run needs.
type Deps struct {
	IO     IO
	Getenv func(string) string
	Now    func() time.Time
	LogDir string
}
