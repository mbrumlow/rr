package tui

import (
	"fmt"
	"os"
	"rr/gen/go/proto/rr"
)

type Simple struct {
}

func (s *Simple) Render(event *rr.Event) error {
	switch e := event.Event.(type) {
	case *rr.Event_RunEvent:
		fmt.Fprintf(os.Stdout, "> %v\n", e.RunEvent.Command)
	case *rr.Event_StdoutEvent:
		fmt.Fprintln(os.Stdout, string(e.StdoutEvent.Data))
	case *rr.Event_StderrEvent:
		fmt.Fprintln(os.Stderr, string(e.StderrEvent.Data))
	}

	return nil
}
