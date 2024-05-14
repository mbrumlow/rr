package run

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"rr/gen/go/proto/rr"
	"rr/pkg/eventdb"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/shlex"
	"github.com/oklog/ulid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

type Run struct {
	Name     string   `yaml:"name"`
	Commands Commands `yaml:"commands"`
}

type Execution struct {
	id string

	spec []byte

	run *Run
	log *eventdb.EventDB

	count int64

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
	Render(*rr.Event) error
}

func File(p string, follower Follower) error {
	spec, err := os.ReadFile(p)
	if err != nil {
		return err
	}

	var run Run
	if err := yaml.Unmarshal(spec, &run); err != nil {
		return err
	}

	return Exec(&run, spec, true, follower)
}

func Exec(run *Run, spec []byte, local bool, follower Follower) error {
	t := time.Now().UnixNano()
	randSource := rand.New(rand.NewSource(t))

	ulid := ulid.MustNew(ulid.Timestamp(time.Now()), randSource)

	dbPath := filepath.Join("test", ulid.String()+".db")
	// TODO: create uniqe id for the run.

	// Create database for the run.
	log, err := eventdb.NewEventDB(dbPath, eventMap)
	if err != nil {
		return err
	}

	var mu sync.Mutex
	exe := Execution{
		id:   ulid.String(),
		spec: spec,
		run:  run,
		log:  log,
		cond: sync.NewCond(&mu),
	}

	go exe.Run(local)
	return exe.Attach(follower)
}

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
				last = event.Id
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

func (e *Execution) Events(n, max int64) ([]*rr.Event, error) {
	return e.log.Events(n, max)
}

func (e *Execution) Commands(local bool) []Command {
	if local {
		return e.run.Commands.Local
	}
	return e.run.Commands.Remote
}

func (e *Execution) Run(local bool) error {

	fmt.Printf("RUN CALLED: %v\n", local)

	for cmdID, c := range e.Commands(local) {
		fmt.Printf("DEBUG: RUNNING COMMAND: %v -> %v: %v\n",
			local, cmdID, c.Run)
		if err := e.runCommand(cmdID, c); err != nil {
			log.Printf("COMMAND FAILED: %v\n", err)
			return err
		}
	}

	if local {
		if err := e.remote(); err != nil {
			return err
		}
	}

	e.cond.L.Lock()
	e.done = true
	e.cond.Broadcast()
	e.cond.L.Unlock()

	return nil
}

func (e *Execution) remote() error {

	conn, err := net.Dial("tcp", "127.0.0.1:3923")
	if err != nil {
		return err
	}

	client := &Client{}

	if err := client.WriteMessage(conn,
		&rr.Message{
			Msg: &rr.Message_Run{
				Run: &rr.Run{
					Id:         e.id,
					Spec:       e.spec,
					FirstEvent: atomic.AddInt64(&e.events, 1),
				},
			},
		},
	); err != nil {
		return err
	}

	for {
		rr, err := client.ReadMessage(conn)
		if err != nil {
			return err
		}
		fmt.Printf("NEW RR: %v\n", rr)
	}

	return nil

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

	if err := e.Event([]*rr.Event{&rr.Event{
		Id:        atomic.AddInt64(&e.count, 1),
		Timestamp: timestamppb.Now(),
		Group:     int32(cmdID),
		Event: &rr.Event_RunEvent{
			RunEvent: &rr.RunEvent{
				Command: c.Run,
			},
		},
	}}); err != nil {
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

func (e *Execution) Event(events []*rr.Event) error {
	if len(events) == 0 {
		return nil
	}
	err := e.log.Event(events)
	e.cond.L.Lock()
	if err == nil {
		e.events = events[len(events)-1].Id
	} else {
		e.done = true
	}
	e.cond.Broadcast()
	e.cond.L.Unlock()
	return err
}

func (e *Execution) logLines(reader io.Reader, grp, typ int, wg *sync.WaitGroup) {
	defer wg.Done()

	ch := make(chan *rr.Event, 100)
	defer close(ch)

	wg.Add(1)
	go e.chanToDB(ch, wg)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// TODO: process command filters and transformations

		event := &rr.Event{
			Id:        atomic.AddInt64(&e.count, 1),
			Group:     int32(grp),
			Timestamp: timestamppb.Now(),
		}

		switch typ {
		case STDOUT:
			event.Event = &rr.Event_StdoutEvent{
				StdoutEvent: &rr.StdOutEvent{
					Data: []byte(line),
				},
			}
		case STDERR:
			event.Event = &rr.Event_StderrEvent{
				StderrEvent: &rr.StdErrEvent{
					Data: []byte(line),
				},
			}
		}

		ch <- event
	}
}

func (e *Execution) chanToDB(ch <-chan *rr.Event, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		events := make([]*rr.Event, 0)
		event, ok := <-ch
		if !ok {
			return
		}
		events = append(events, event)
		for {
			done := false
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}
				events = append(events, event)
			default:
				done = true
			}
			if done || len(events) >= 2 {
				break
			}
		}
		if err := e.Event(events); err != nil {
			// Lets not mess with the execution and just log that we
			// could not to the event db.
			fmt.Printf("failed to log event: %v", err)
		}
	}
}
