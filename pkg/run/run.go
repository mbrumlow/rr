package run

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"rr/pkg/logdb"
	"sync"
	"time"

	"github.com/google/shlex"
	"github.com/oklog/ulid"
	"gopkg.in/yaml.v3"
)

type Run struct {
	Name     string   `yaml:"name"`
	Commands Commands `yaml:"commands"`
}

type Execution struct {
	id string

	run *Run
	log *logdb.LogDB

	cond   *sync.Cond
	events int64
	done   bool
}

type Commands struct {
	Local  []Command `yaml:"local"`
	Remote []Command `yaml:"remote"`
}

type Command struct {
	Run string `yaml:"run"`
}

type Follower interface {
	Render(logdb.RawEvent) error
}

func File(p string, follower Follower) error {

	file, err := os.Open(p)
	if err != nil {
		return err
	}

	run := &Run{}
	if err := yaml.NewDecoder(file).Decode(run); err != nil {
		return err
	}

	return Exec(run, true, follower)
}

func Exec(run *Run, local bool, follower Follower) error {
	t := time.Now().UnixNano()
	randSource := rand.New(rand.NewSource(t))

	ulid := ulid.MustNew(ulid.Timestamp(time.Now()), randSource)

	dbPath := filepath.Join("test", ulid.String()+".db")
	// TODO: create uniqe id for the run.

	// Create database for the run.
	log, err := logdb.NewLogDB(dbPath)
	if err != nil {
		return err
	}

	var mu sync.Mutex
	exe := Execution{
		id:   ulid.String(),
		run:  run,
		log:  log,
		cond: sync.NewCond(&mu),
	}

	go exe.Run(local)
	return exe.Attach(follower)
}

// func Serve() error {

// 	// TODO: gob decode request?
// 	// TODO: new DB
// 	// TODO: Run remote
// 	// TODO: follower
// Follower copies events over the network with gob encode
// ID is for run is kept the same for the caller.

// We need to log
// ID, Hash of file, time we did it, name, total result.
// One file per run
// Show last 15 by default, but more can be shown.

// }

func (e *Execution) Attach(display Follower) error {
	var last int64
	for {
		max, more := e.WaitForEvent(last)
		for last < max {
			events, err := e.Events(last, 100)
			if err != nil {
				fmt.Printf("FIXME: failed to get lines\n")
				return err
			}
			for _, event := range events {
				last = event.ID
				if err := display.Render(event); err != nil {
					fmt.Printf("FIXME: failed to render event\n")
					return err
				}
			}
		}
		if !more {
			break
		}
	}
	return nil
}

func (e *Execution) WaitForEvent(n int64) (int64, bool) {
	e.cond.L.Lock()
	for n >= e.events && !e.done {
		e.cond.Wait()
	}
	lines, done := e.events, e.done
	e.cond.L.Unlock()
	return lines, !done
}

func (e *Execution) Events(n, max int64) ([]logdb.RawEvent, error) {
	return e.log.Events(n, max)
}

func (e *Execution) Commands(local bool) []Command {
	if local {
		return e.run.Commands.Local
	}
	return e.run.Commands.Remote
}

func (e *Execution) Run(local bool) error {

	for cmdID, c := range e.Commands(local) {
		if err := e.runCommand(cmdID, c); err != nil {
			return nil
		}
	}

	e.cond.L.Lock()
	e.done = true
	e.cond.Broadcast()
	e.cond.L.Unlock()

	return nil
}

func (e *Execution) remote() {

	// Gob Encode run obejct
	// Send run object.

	// stream results back.
	// linesb
	// command results.
	// done

	// lines and results are copied into the database.
	// local followers will receive updates from that end.
}

func (e *Execution) runCommand(cmdID int, c Command) error {

	if _, err := e.Event(Event{
		grp:  cmdID,
		typ:  RUN,
		data: []byte(c.Run),
	}); err != nil {
		return err
	}

	// TODO: interpolate command
	// TODO: interpolate env vars
	// TODO: execute command with env set

	args, err := shlex.Split(c.Run)
	if err != nil {
		return err
	}

	// TODO: maybe notify watchers the resolved command?

	cmd := exec.Command(args[0], args[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	cmd.Start()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go e.logLines(stdout, cmdID, STDOUT, &wg)
	wg.Add(1)
	go e.logLines(stderr, cmdID, STDERR, &wg)

	wg.Wait()
	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func (e *Execution) Event(event logdb.Event) (int64, error) {
	id, err := e.log.Event(event)
	e.cond.L.Lock()
	if err == nil {
		e.events = id
	} else {
		e.done = true
	}
	e.cond.Broadcast()
	e.cond.L.Unlock()
	return id, err
}

func (e *Execution) logLines(reader io.Reader, grp, typ int, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// TODO: process command filters and transformations

		event := Event{
			grp:  grp,
			typ:  typ,
			data: []byte(line),
		}

		if _, err := e.Event(event); err != nil {
			fmt.Printf("failed to log event: %v", err)
		}
	}
}
