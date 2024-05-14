package logdb

import (
	"database/sql"
	"rr/gen/go/proto/rr"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	_ "modernc.org/sqlite"
)

type LogDB struct {
	mu sync.Mutex
	db *sql.DB
	// stmt      *sql.Stmt
	stmtEvent *sql.Stmt
	eventMap  EventMap
}

type EventMap func(event *rr.Event) int

func NewLogDB(p string, eventMap EventMap) (*LogDB, error) {
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

	return &LogDB{
		mu: sync.Mutex{},
		db: db,
		// stmt:      stmt, //
		stmtEvent: stmtEvent,
		eventMap:  eventMap,
	}, nil
}

func (l *LogDB) Close() {
	l.stmtEvent.Close()
	l.db.Close()
}

// type LogLine struct {
// 	ID   int64
// 	CMD  int
// 	FD   int
// 	Data string
// }

// func (l *LogDB) GetLines(start, limit int64) ([]LogLine, error) {
// 	l.mu.Lock()
// 	defer l.mu.Unlock()

// 	query := `
//         SELECT id, cmd, fd, data
//         FROM log
//         WHERE id > ?
//         ORDER BY id
//         LIMIT ?`

// 	rows, err := l.db.Query(query, start, limit)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	logLines := make([]LogLine, 0)
// 	for rows.Next() {
// 		var id int64
// 		var cmd int
// 		var fd int
// 		var data string
// 		err = rows.Scan(&id, &cmd, &fd, &data)
// 		if err != nil {
// 			return nil, err
// 		}
// 		logLines = append(logLines,
// 			LogLine{
// 				ID:   id,
// 				CMD:  cmd,
// 				FD:   fd,
// 				Data: data,
// 			},
// 		)
// 	}
// 	return logLines, nil
// }

// func (l *LogDB) Log(cmd, fd int, data string) (int64, error) {
// 	l.mu.Lock()
// 	defer l.mu.Unlock()

// 	res, err := l.stmt.Exec(cmd, fd, data)
// 	if err != nil {
// 		return 0, err
// 	}
// 	last, err := res.LastInsertId()
// 	if err != nil {
// 		return 0, err
// 	}
// 	return last, nil
// }

type Event interface {
	Group() int
	Type() int
	Data() []byte
}

type RawEvent struct {
	ID        int64
	Timestamp string
	Group     int
	Type      int
	Data      []byte
}

// func (l *LogDB) Events(start, limit int64) ([]RawEvent, error) {
// 	l.mu.Lock()
// 	defer l.mu.Unlock()

// 	query := `
//         SELECT id, timestamp, grp, typ, data
//         FROM event
//         WHERE id > ?
//         ORDER BY id
//         LIMIT ?`

// 	rows, err := l.db.Query(query, start, limit)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	events := make([]RawEvent, 0)
// 	for rows.Next() {
// 		var id int64
// 		var timestamp string
// 		var grp int
// 		var typ int
// 		var data []byte
// 		err = rows.Scan(&id, &timestamp, &grp, &typ, &data)
// 		if err != nil {
// 			return nil, err
// 		}
// 		events = append(events,
// 			RawEvent{
// 				ID:    id,
// 				Group: grp,
// 				Type:  typ,
// 				Data:  data,
// 			},
// 		)
// 	}
// 	return events, nil
// }

func (l *LogDB) Events(start, limit int64) ([]*rr.Event, error) {
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

// func (l *LogDB) Event(e Event) (int64, error) {
// 	l.mu.Lock()
// 	defer l.mu.Unlock()
// 	currentTime := time.Now().Format(time.RFC3339)
// 	res, err := l.stmtEvent.Exec(currentTime, e.Group(), e.Type(), e.Data())
// 	if err != nil {
// 		return 0, err
// 	}
// 	last, err := res.LastInsertId()
// 	if err != nil {
// 		return 0, err
// 	}
// 	return last, nil
// }

func (l *LogDB) Event(e *rr.Event) error {
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
