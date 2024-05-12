package run

import "rr/gen/go/proto/rr"

const (
	_   = iota
	RUN = 100 + iota
	STDOUT
	STDERR
	RESULT
)

type Event struct {
	grp  int
	typ  int
	data []byte
}

func (l Event) Group() int {
	return l.grp
}
func (l Event) Type() int {
	return l.typ
}
func (l Event) Data() []byte {
	return l.data
}

func eventMap(event *rr.Event) int {
	switch event.Event.(type) {
	case *rr.Event_RunEvent:
		return RUN
	case *rr.Event_StdoutEvent:
		return STDOUT
	case *rr.Event_StderrEvent:
		return STDERR
	case *rr.Event_RestultEvent:
		return RESULT
	}
	// TODO: fix me, probably should error?
	return 0
}
