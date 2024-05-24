package run

import (
	"encoding/binary"
	"net"
	"os"
	"path/filepath"
	"rr/gen/go/proto/rr"
	"rr/pkg/eventdb"
	"rr/pkg/log"
	"sync"

	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

type Client struct {
	conn net.Conn
	log  log.Logger
}

func Serve(debug bool) error {

	// TODO: gob decode request?
	// TODO: new DB
	// TODO: Run remote
	// TODO: follower
	// Follower copies events over the network with gob encode
	// ID is for run is kept the same for the caller.

	// We need to log
	// ID, Hash of file, time we did it, name, total result.
	// One file per run
	// Show last 15 by default, but more can be shown.

	ln, err := net.Listen("tcp", ":3923")
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("connection failure: %v", err)
		}

		client := &Client{
			log: log.NewServerLog(
				conn.RemoteAddr().String(),
				debug,
			),
		}
		go client.Serve(conn)
	}

}

func (c *Client) Serve(conn net.Conn) {
	defer conn.Close()
	for {
		c.log.Debug("reading message")
		message, err := c.ReadMessage(conn)
		if err != nil {
			c.log.Error("failed to read message: %v", err)
			return
		}

		switch msg := message.Msg.(type) {
		case *rr.Message_Run:
			c.log.Debug("message Run received")
			c.log.Log("Run ID: %v", msg.Run.Id)
			if err := c.Run(conn, msg); err != nil {
				c.log.Error("failed to run: %v", err)
				return
			}
			return
		}
	}
}

func (c *Client) ReadMessage(conn net.Conn) (*rr.Message, error) {
	var length uint32
	if err := binary.Read(
		conn,
		binary.LittleEndian,
		&length,
	); err != nil {
		return nil, err
	}

	buf := make([]byte, length)
	if _, err := conn.Read(buf); err != nil {
		return nil, err
	}

	message := &rr.Message{}
	if err := proto.Unmarshal(buf, message); err != nil {
		return nil, err
	}
	return message, nil
}

func (c *Client) WriteMessage(conn net.Conn, msg *rr.Message) error {
	c.log.Debug("sending message")
	buf, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	length := uint32(len(buf))
	c.log.Debug("sending message length: %v", length)
	if err := binary.Write(conn, binary.LittleEndian, &length); err != nil {
		return err
	}
	c.log.Debug("sending message of length: %v", length)
	if _, err := conn.Write(buf); err != nil {
		return err
	}
	c.log.Debug("done sending message")
	return nil
}

func (c *Client) Run(conn net.Conn, msg *rr.Message_Run) error {
	var run Run
	if err := yaml.Unmarshal(msg.Run.Spec, &run); err != nil {
		return err
	}

	dbPath := filepath.Join("test", msg.Run.Id+".db")

	c.log.Debug("log path: %v", dbPath)

	eventNotify := false
	if _, err := os.Stat(dbPath); err == nil {
		eventNotify = true
	}

	eventdb, err := eventdb.NewEventDB(dbPath, eventMap)
	if err != nil {
		log.Printf("failed to open database: %v", err)
		return err
	}

	var mu sync.Mutex
	exe := &Execution{
		id:     msg.Run.Id,
		spec:   msg.Run.Spec,
		run:    &run,
		log:    eventdb,
		cond:   sync.NewCond(&mu),
		count:  msg.Run.FirstEvent,
		events: msg.Run.FirstEvent,
	}

	go exe.Run(false, c.log)
	return exe.Attach(c.relay(conn, eventNotify), msg.Run.FirstEvent)

}

func (c *Client) relay(conn net.Conn, eventNoitfy bool) Follower {
	if eventNoitfy {
		return &EventNotify{
			client: c,
			conn:   conn,
		}
	} else {
		return nil
	}
}

type EventNotify struct {
	client *Client
	conn   net.Conn
}

func (e *EventNotify) Render(event *rr.Event) error {
	return e.client.WriteMessage(e.conn, &rr.Message{
		Msg: &rr.Message_Event{
			Event: &rr.Event{
				Id: event.Id,
				Event: &rr.Event_NewEvent{
					NewEvent: &rr.NewEvent{
						Id: event.Id,
					},
				},
			},
		},
	})
}
