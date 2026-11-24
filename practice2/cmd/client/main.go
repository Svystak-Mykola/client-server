package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-udp" {
		c, e := net.Dial("udp", "127.0.0.1:9001")
		if e != nil {
			fmt.Println(e)
			return
		}
		defer c.Close()
		msg := strings.Join(os.Args[2:], " ")
		if msg == "" {
			msg = "temp=24.5;pressure=750"
		}
		_, e = c.Write([]byte(msg))
		if e != nil {
			fmt.Println(e)
			return
		}
		fmt.Println("UDP sent:", msg)
		return
	}
	c, e := net.Dial("tcp", "127.0.0.1:9000")
	if e != nil {
		fmt.Println(e)
		return
	}
	defer c.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		r := bufio.NewScanner(c)
		for r.Scan() {
			fmt.Println(r.Text())
		}
	}()
	input := make(chan string)
	go func() {
		defer close(input)
		r := bufio.NewScanner(os.Stdin)
		for r.Scan() {
			select {
			case input <- r.Text():
			case <-done:
				return
			}
		}
	}()
	for {
		select {
		case <-done:
			return
		case line, ok := <-input:
			if !ok {
				return
			}
			if _, e := fmt.Fprintln(c, line); e != nil {
				return
			}
			if strings.EqualFold(strings.TrimSpace(line), "QUIT") || strings.EqualFold(strings.TrimSpace(line), "CLOSE") {
				<-done
				return
			}
		}
	}
}
