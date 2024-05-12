package tui

import (
	"fmt"
	"os"
	"rr/pkg/logdb"
	"rr/pkg/run"
)

type Simple struct {
}

func (s *Simple) Render(event logdb.RawEvent) error {
	switch event.Type {
	case run.STDOUT:
		fmt.Fprintln(os.Stdout, string(event.Data))
	case run.STDERR:
		fmt.Fprintln(os.Stderr, string(event.Data))
	case run.RUN:
		fmt.Fprintf(os.Stdout, "> %v\n", string(event.Data))
	}
	return nil
}
