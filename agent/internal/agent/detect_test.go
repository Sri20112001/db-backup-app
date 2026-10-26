package agent

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

// serveFake accepts connections until the test finishes and runs fn on each.
func serveFake(t *testing.T, fn func(net.Conn)) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	t.Cleanup(func() {
		close(done)
		_ = ln.Close()
	})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				select {
				case <-done:
					return
				default:
				}
				fn(c)
			}(conn)
		}
	}()
	return ln
}

func portOf(t *testing.T, ln net.Listener) int {
	t.Helper()
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("addr: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("port %q: %v", portStr, err)
	}
	return port
}

func TestIdentifyPortPostgres(t *testing.T) {
	// A real backend answers the SSLRequest with 'S' or 'N'.
	ln := serveFake(t, func(c net.Conn) {
		buf := make([]byte, 8)
		if _, err := io.ReadFull(c, buf); err != nil {
			return
		}
		_, _ = c.Write([]byte{'N'})
	})
	inst, ok := identifyPort(context.Background(), portOf(t, ln))
	if !ok {
		t.Fatal("expected the fake server to be identified as PostgreSQL")
	}
	if inst.Type != "POSTGRES" || inst.Product != "PostgreSQL" || inst.Port != portOf(t, ln) {
		t.Fatalf("unexpected instance: %+v", inst)
	}
}

func TestIdentifyPortMSSQL(t *testing.T) {
	ln := serveFake(t, func(c net.Conn) {
		buf := make([]byte, 8)
		_, _ = io.ReadFull(c, buf) // prelogin header
		resp := make([]byte, 8)
		resp[0] = 0x04 // PRELOGINRESPONSE
		binary.BigEndian.PutUint16(resp[2:4], uint16(len(resp)))
		_, _ = c.Write(resp)
	})
	inst, ok := identifyPort(context.Background(), portOf(t, ln))
	if !ok {
		t.Fatal("expected the fake server to be identified as SQL Server")
	}
	if inst.Type != "MSSQL" {
		t.Fatalf("unexpected type %q", inst.Type)
	}
}

func TestIdentifyPortMongo(t *testing.T) {
	ln := serveFake(t, func(c net.Conn) {
		// Drain whatever probe arrives, then answer with an OP_MSG header.
		_ = c.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, _ = io.Copy(io.Discard, c)
		head := make([]byte, 16)
		binary.LittleEndian.PutUint32(head[0:4], 16)
		binary.LittleEndian.PutUint32(head[12:16], 2013)
		_, _ = c.Write(head)
	})
	inst, ok := identifyPort(context.Background(), portOf(t, ln))
	if !ok {
		t.Fatal("expected the fake server to be identified as MongoDB")
	}
	if inst.Type != "MONGODB" {
		t.Fatalf("unexpected type %q", inst.Type)
	}
}

func TestIdentifyPortUnrelatedService(t *testing.T) {
	// Anything that does not speak one of the three protocols must be left
	// alone rather than misreported as a database.
	ln := serveFake(t, func(c net.Conn) {
		_ = c.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, _ = io.Copy(io.Discard, c)
	})
	if _, ok := identifyPort(context.Background(), portOf(t, ln)); ok {
		t.Fatal("an unrelated service must not be reported as a database")
	}
}

func TestBsonDocFraming(t *testing.T) {
	doc := bsonDoc(bsonBool("isMaster", true), bsonString("$db", "admin"))
	if int(binary.LittleEndian.Uint32(doc[0:4])) != len(doc) {
		t.Fatalf("length prefix %d does not match document length %d",
			binary.LittleEndian.Uint32(doc[0:4]), len(doc))
	}
	if doc[len(doc)-1] != 0x00 {
		t.Fatal("document must end with a terminator byte")
	}
	if doc[4] != 0x08 {
		t.Fatalf("first element type = %#x, want 0x08 (bool)", doc[4])
	}
}
