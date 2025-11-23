package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgproto3/v2"
)

func setup(ctx context.Context, conn *pgconn.PgConn) {
	res := conn.Exec(ctx, "CREATE TABLE IF NOT EXISTS TEST (id INT UNIQUE PRIMARY KEY , name TEXT NOT NULL)")
	_, err := res.ReadAll()

	if err != nil {
		fmt.Fprintf(os.Stderr, "table creation failed: %s\n", err)
	}
}
func isBusy(ctx context.Context, conn *pgconn.PgConn) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		default:
			if conn.IsClosed() {
				return false
			}
			if conn.IsBusy() {
				return true
			} else {
				return false
			}
		}
	}
}
func insert(ctx context.Context, conn *pgconn.PgConn, id int, name string) {
	if isBusy(ctx, conn) {
		return
	}
	res := conn.Exec(ctx, fmt.Sprintf("insert into test (id, name) values (%d, '%s')", id, name))
	_, err := res.ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "insert failed: %v\n", err)
	}
}

func update(ctx context.Context, conn *pgconn.PgConn, id int, name string) {
	if isBusy(ctx, conn) {
		return
	}
	res := conn.Exec(ctx, fmt.Sprintf("update test set name = '%s' where id = %d", name, id))
	_, err := res.ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
	}
}

func tick1min(ctx context.Context, url string) {
	conn, _ := pgconn.Connect(ctx, url)
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	i := 1
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if conn.IsClosed() {
				conn, _ = pgconn.Connect(ctx, url)
			}
			fmt.Println("---------------------------------------------")
			update(ctx, conn, 1, "name"+fmt.Sprint(i))
			i++
		}
	}
}

func connect() {

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	defer cancel()
	url := "postgresql://root@localhost:26257/defaultdb?sslmode=disable"
	conn, err := pgconn.Connect(ctx, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect failed: %v\n", err)
		return
	}

	connCreate, _ := pgconn.Connect(ctx, url)
	connInsert, _ := pgconn.Connect(ctx, url)
	connSub, _ := pgconn.Connect(ctx, url)

	fmt.Println("Connected to CockroachDB")

	setup(ctx, connCreate)

	time.Sleep(2 * time.Second)
	insert(ctx, connInsert, 1, "anand")
	time.Sleep(2 * time.Second)
	insert(ctx, connInsert, 2, "joker")

	time.Sleep(2 * time.Second)
	insert(ctx, connInsert, 3, "jack")

	time.Sleep(2 * time.Second)
	// go tick1min(ctx, url)
	defer conn.Close(ctx)
	buf, _ := (&pgproto3.Query{String: "SUBSCRIBE TO SELECT * FROM test where id=1 "}).Encode(nil)
	connSub.SendBytes(ctx, buf)

	fmt.Println("Streaming started...")

	var columns []string

	for {
		msg, err := connSub.ReceiveMessage(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "receive error: %v\n", err)
			break
		}

		switch m := msg.(type) {
		case *pgproto3.RowDescription:
			// Get column names
			columns = make([]string, len(m.Fields))
			for i, field := range m.Fields {
				columns[i] = string(field.Name)
			}
			fmt.Printf("Columns: %s\n", strings.Join(columns, ", "))

		case *pgproto3.DataRow:
			// Print all column values
			values := make([]string, len(m.Values))
			for i, val := range m.Values {
				if val == nil {
					values[i] = "NULL"
				} else {
					values[i] = string(val)
				}
			}
			fmt.Printf("Row: %s\n", strings.Join(values, " | "))

		case *pgproto3.CommandComplete:
			fmt.Println("Command complete")
		case *pgproto3.ReadyForQuery:
			fmt.Println("Ready for query")
		case *pgproto3.ErrorResponse:
			fmt.Printf("Error: %s\n", m.Message)
		}
	}

}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down...")
			return
		default:
		}
		fmt.Println("Connecting to CockroachDB...")
		connect()
	}
}
