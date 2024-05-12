package eventdb

import (
	"database/sql"
	"rr/gen/go/proto/rr"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	_ "modernc.org/sqlite"
)

type EventDB struct {
	mu sync.Mutex
	db *sql.DB
	// stmt      *sql.Stmt
	stmtEvent *sql.Stmt
	eventMap  EventMap
}

type EventMap func(event *rr.Event) int

func NewEventDB(p string, eventMap EventMap) (*EventDB, error) {
	db, err := sql.Open("sqlite", p)
	if err != nil {
		return nil, err
	}

	if _, err = db.Exec(`
DROP TABLE IF EXISTS event;
CREATE TABLE event (
    id INTEGER PRIMARY KEY,
    timestamp DATETIME,
    grp INTEGER,
    typ INTEGER,
    data BLOB
);

`); err != nil {
		return nil, err
	}

	insertEventSQL := `INSERT INTO event (id, timestamp, grp, typ, data) VALUES (?, ?, ?, ?, ?)`
	stmtEvent, err := db.Prepare(insertEventSQL)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &EventDB{
		mu: sync.Mutex{},
		db: db,
		// stmt:      stmt, //
		stmtEvent: stmtEvent,
		eventMap:  eventMap,
	}, nil
}

func (l *EventDB) Close() {
	l.stmtEvent.Close()
	l.db.Close()
}

func (l *EventDB) Events(start, limit int64) ([]*rr.Event, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	query := `
        SELECT data
        FROM event
        WHERE id > ?
        ORDER BY id
        LIMIT ?`

	rows, err := l.db.Query(query, start, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*rr.Event, 0)
	for rows.Next() {
		var data []byte
		err = rows.Scan(&data)
		if err != nil {
			return nil, err
		}
		var event rr.Event
		if err := proto.Unmarshal(data, &event); err != nil {
			return nil, err
		}
		events = append(events, &event)
	}
	return events, nil
}

func (l *EventDB) Event(e *rr.Event) error {
	timestamp := e.Timestamp.AsTime().Format(time.RFC3339)
	data, err := proto.Marshal(e)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.stmtEvent.Exec(e.Id, timestamp, l.eventMap(e), e.Group, data); err != nil {
		return err
	}
	return nil
}
