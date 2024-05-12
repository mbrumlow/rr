package run

const (
	_   = iota
	RUN = 100 + iota
	STDOUT
	STDERR
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
