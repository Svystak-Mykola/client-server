package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

func printResult(r Result) {
	if r.Err != nil {
		fmt.Printf("%s | error: %v | %v\n", r.URL, r.Err, r.Duration.Round(time.Millisecond))
		return
	}
	fmt.Printf("%s | %d %s | %v\n", r.URL, r.Status, http.StatusText(r.Status), r.Duration.Round(time.Millisecond))
}

func summary(s Stats, elapsed time.Duration) {
	fmt.Printf("Total: %d | OK: %d | Other HTTP: %d | Errors: %d\nTime: %v\n", s.Total, s.OK, s.Other, s.Errors, elapsed.Round(time.Millisecond))
}

func main() {
	demo := flag.Bool("demo", false, "use local test server")
	flag.Parse()
	urls := []string{"https://go.dev", "https://github.com", "https://example.com", "https://httpbin.org/status/404", "https://url-monitor.invalid"}
	if *demo {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/home":
				time.Sleep(120 * time.Millisecond)
				w.WriteHeader(200)
			case "/api":
				time.Sleep(180 * time.Millisecond)
				w.WriteHeader(200)
			case "/missing":
				time.Sleep(90 * time.Millisecond)
				w.WriteHeader(404)
			default:
				time.Sleep(150 * time.Millisecond)
				w.WriteHeader(503)
			}
		}))
		defer server.Close()
		urls = []string{server.URL + "/home", server.URL + "/api", server.URL + "/missing", server.URL + "/offline", "invalid-url"}
	}
	if flag.NArg() > 0 {
		urls = flag.Args()
	}
	var c Checker = HTTPMonitor{Client: &http.Client{Timeout: 5 * time.Second}}
	fmt.Println("=== URL Monitor ===\n\nLevel 1: Sequential")
	start := time.Now()
	results, stats := sequential(c, urls)
	first := time.Since(start)
	for _, r := range results {
		printResult(r)
	}
	summary(stats, first)
	fmt.Println("\nLevel 2: Goroutines + Mutex")
	start = time.Now()
	results, stats = concurrent(c, urls)
	second := time.Since(start)
	for _, r := range results {
		printResult(r)
	}
	summary(stats, second)
	fmt.Println("\nLevel 3: Channels")
	start = time.Now()
	stats = Stats{}
	for r := range stream(c, urls) {
		printResult(r)
		stats.Add(r)
	}
	third := time.Since(start)
	summary(stats, third)
	if third > 0 {
		fmt.Printf("\nSpeedup: %.2fx\n", float64(first)/float64(third))
	}
}
