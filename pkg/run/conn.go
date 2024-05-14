package run

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"rr/gen/go/proto/rr"
	"rr/pkg/eventdb"
	"rr/pkg/tui"
	"sync"

	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

type Client struct {
	conn net.Conn
}

func Serve() error {

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

		client := &Client{}
		go client.Serve(conn)
	}

}

func (c *Client) Serve(conn net.Conn) {
	for {
		fmt.Print("reading message\n")
		message, err := c.ReadMessage(conn)
		if err != nil {
			log.Printf("failed to read message: %v", err)
			return
		}

		switch msg := message.Msg.(type) {
		case *rr.Message_Run:
			if err := c.Run(conn, msg); err != nil {
				log.Printf("failed to run: %v", err)
				return
			}
		}
	}
}

func (c *Client) ReadMessage(conn net.Conn) (*rr.Message, error) {

	fmt.Printf("about to read message len\n")

	var length uint32
	if err := binary.Read(
		conn,
		binary.LittleEndian,
		&length,
	); err != nil {
		return nil, err
	}

	fmt.Printf("GOT LEN: %v", length)

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

	buf, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	length := uint32(len(buf))

	fmt.Printf("sending message length: %v\n", length)
	if err := binary.Write(conn, binary.LittleEndian, &length); err != nil {
		return err
	}

	fmt.Printf("sending message of length: %v", length)
	if _, err := conn.Write(buf); err != nil {
		return err
	}

	fmt.Printf("done sending message\n")

	return nil
}

func (c *Client) Run(conn net.Conn, msg *rr.Message_Run) error {

	fmt.Printf("spec: %v\n", string(msg.Run.Spec))

	var run Run
	if err := yaml.Unmarshal(msg.Run.Spec, &run); err != nil {
		return err
	}

	dbPath := filepath.Join("test", msg.Run.Id+".db")

	// TODO: if file exist, then we should log to db here, but only signal
	// remote(local) listener when the log is updated so it can process it.

	eventdb, err := eventdb.NewEventDB(dbPath, eventMap)
	if err != nil {
		log.Printf("failed to open database: %v", err)
		return err
	}

	fmt.Printf("LOG: %v\n", eventdb)

	var mu sync.Mutex
	exe := Execution{
		id:     msg.Run.Id,
		spec:   msg.Run.Spec,
		run:    &run,
		log:    eventdb,
		cond:   sync.NewCond(&mu),
		count:  msg.Run.FirstEvent,
		events: msg.Run.FirstEvent,
	}

	go exe.Run(false)

	f := &tui.Simple{}
	return exe.Attach(f)

	// we have raw run yaml in this.
	// copy id
	// construct run object
	// run run object
	// stream renders back to the network over the wire.
}
