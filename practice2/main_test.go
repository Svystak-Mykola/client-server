package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestGateway(t *testing.T) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	p, e := net.ListenPacket("udp", "127.0.0.1:0")
	if e != nil {
		l.Close()
		t.Fatal(e)
	}
	s := &Server{clients: make(map[net.Conn]bool)}
	s.wg.Add(2)
	go s.accept(l)
	go s.telemetry(p)
	defer s.shutdown(l, p)
	connect := func() (net.Conn, *bufio.Reader) {
		c, e := net.Dial("tcp", l.Addr().String())
		if e != nil {
			t.Fatal(e)
		}
		c.SetDeadline(time.Now().Add(5 * time.Second))
		t.Cleanup(func() { c.Close() })
		return c, bufio.NewReader(c)
	}
	expect := func(r *bufio.Reader, want string) {
		t.Helper()
		for {
			line, e := r.ReadString('\n')
			if e != nil {
				t.Fatal(e)
			}
			if strings.Contains(line, want) {
				return
			}
		}
	}
	a, ra := connect()
	expect(ra, "Connected")
	b, rb := connect()
	expect(rb, "Connected")
	c, rc := connect()
	expect(rc, "Connected")
	fmt.Fprintln(a, "ECHO hello")
	expect(ra, "[ECHO]: hello")
	fmt.Fprintln(b, "TIME")
	expect(rb, "[TIME]:")
	fmt.Fprintln(a, "UNKNOWN")
	expect(ra, "[ERROR]")
	fmt.Fprintln(a, "BROADCAST test message")
	expect(ra, "Delivered to 2")
	expect(rb, "test message")
	expect(rc, "test message")
	u, e := net.Dial("udp", p.LocalAddr().String())
	if e != nil {
		t.Fatal(e)
	}
	defer u.Close()
	u.Write([]byte("temp=25"))
	expect(ra, "temp=25")
	expect(rb, "temp=25")
	expect(rc, "temp=25")
	fmt.Fprintln(c, "QUIT")
	expect(rc, "Goodbye")
	expect(ra, "Client left")
	fmt.Fprintln(b, "MSG remaining")
	expect(rb, "Delivered to 1")
	expect(ra, "remaining")
}
