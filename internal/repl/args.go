package repl

import (
	"flag"
	"io"
)

// ParseArgs parses argv, excluding the program name, into Options.
func ParseArgs(name string, args []string, out io.Writer) (Options, error) {
	var opts Options
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(out)
	flags.Var((*configFlags)(&opts.Config), "c", "config key=value")
	flags.BoolVar(&opts.Raw, "raw", false, "emit raw JSONL")

	if err := flags.Parse(args); err != nil {
		return Options{}, err
	}
	return opts, nil
}

type configFlags []string

func (f *configFlags) String() string {
	return ""
}

func (f *configFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}
