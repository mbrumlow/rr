package log

import (
	"fmt"
	"log"
	"os"
)

type Logger interface {
	Log(string, ...any)
	Error(string, ...any)
	Debug(string, ...any)
}

type ServerLog struct {
	id    string
	debug bool
}

func NewServerLog(id string, debug bool) *ServerLog {
	return &ServerLog{
		id:    id,
		debug: debug,
	}
}

func Printf(format string, v ...any) {
	log.Printf(format, v...)
}

func (l *ServerLog) log(typ, format string, a ...any) {
	log.Printf(
		"%v - %v: %v",
		typ,
		l.id,
		fmt.Sprintf(format, a...),
	)
}

func (l *ServerLog) Log(format string, a ...any) {
	l.log("INFO ", format, a...)
}

func (l *ServerLog) Error(format string, a ...any) {
	l.log("ERROR", format, a...)
}

func (l *ServerLog) Debug(format string, a ...any) {
	if l.debug {
		l.log("DEBUG", format, a...)
	}
}

type SimpleLog struct {
	debug bool
}

func NewSimpleLog(debug bool) *SimpleLog {
	return &SimpleLog{
		debug: debug,
	}
}

func (l *SimpleLog) Log(format string, a ...any) {
	fmt.Printf("INFO: %v",
		fmt.Sprintf(format, a...))
}

func (l *SimpleLog) Error(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: %v",
		fmt.Sprintf(format, a...))
}

func (l *SimpleLog) Debug(format string, a ...any) {
	if l.debug {
		fmt.Printf("DEBUG: %v",
			fmt.Sprintf(format, a...))
	}
}
