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
