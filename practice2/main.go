package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	mu       sync.Mutex
	clients  map[net.Conn]bool
	stopping bool
	wg       sync.WaitGroup
}

func (s *Server) send(c net.Conn, msg string) bool {
	c.SetWriteDeadline(time.Now().Add(time.Second))
	_, err := fmt.Fprintln(c, msg)
	return err == nil
}
func (s *Server) broadcast(skip net.Conn, msg string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping {
		return 0
	}
	n := 0
	for c := range s.clients {
		if c != skip && s.send(c, msg) {
			n++
		}
	}
	return n
}
func (s *Server) reply(c net.Conn, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopping {
		s.send(c, msg)
	}
}
func (s *Server) handle(c net.Conn) {
	defer s.wg.Done()
	defer func() {
		s.mu.Lock()
		delete(s.clients, c)
		c.Close()
		s.mu.Unlock()
		s.broadcast(c, "[SYSTEM] Client left: "+c.RemoteAddr().String())
		fmt.Println("Disconnected:", c.RemoteAddr())
	}()
	r := bufio.NewReader(c)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		command, text, _ := strings.Cut(strings.TrimSpace(line), " ")
		text = strings.TrimSpace(text)
		switch strings.ToUpper(command) {
		case "ECHO":
			s.reply(c, "[ECHO]: "+text)
		case "TIME":
			s.reply(c, "[TIME]: "+time.Now().Format(time.RFC3339))
		case "QUIT", "CLOSE":
			s.reply(c, "[SERVER] Goodbye")
			return
		case "MSG", "BROADCAST":
			if text == "" {
				s.reply(c, "[ERROR] Enter a message")
				continue
			}
			n := s.broadcast(c, fmt.Sprintf("[MESSAGE from %s]: %s", c.RemoteAddr(), text))
			s.reply(c, fmt.Sprintf("[SERVER] Delivered to %d clients", n))
			fmt.Println("Broadcast:", text)
		default:
			s.reply(c, "[ERROR] Commands: ECHO, TIME, MSG, BROADCAST, CLOSE, QUIT")
		}
	}
}
func (s *Server) accept(l net.Listener) {
	defer s.wg.Done()
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		if s.stopping {
			c.Close()
			s.mu.Unlock()
			return
		}
		s.clients[c] = true
		s.send(c, "[SERVER] Connected to dispatcher")
		s.wg.Add(1)
		s.mu.Unlock()
		fmt.Println("Connected:", c.RemoteAddr())
		s.broadcast(c, "[SYSTEM] Client joined: "+c.RemoteAddr().String())
		go s.handle(c)
	}
}
func (s *Server) telemetry(p net.PacketConn) {
	defer s.wg.Done()
	b := make([]byte, 4096)
	for {
		n, a, e := p.ReadFrom(b)
		if e != nil {
			return
		}
		msg := fmt.Sprintf("[TELEMETRY from %s]: %s", a, strings.TrimSpace(string(b[:n])))
		fmt.Println(msg)
		s.broadcast(nil, msg)
	}
}
func (s *Server) shutdown(l net.Listener, p net.PacketConn) {
	s.mu.Lock()
	s.stopping = true
	l.Close()
	p.Close()
	for c := range s.clients {
		s.send(c, "[SYSTEM]: SERVER_SHUTTING_DOWN")
		c.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}
func main() {
	l, e := net.Listen("tcp", "127.0.0.1:9000")
	if e != nil {
		fmt.Println(e)
		return
	}
	p, e := net.ListenPacket("udp", "127.0.0.1:9001")
	if e != nil {
		l.Close()
		fmt.Println(e)
		return
	}
	s := &Server{clients: make(map[net.Conn]bool)}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	s.wg.Add(2)
	go s.accept(l)
	go s.telemetry(p)
	fmt.Println("Dispatcher ready: TCP 9000 | UDP 9001")
	<-stop
	s.shutdown(l, p)
	fmt.Println("Server stopped safely")
}
