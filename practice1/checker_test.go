package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestModes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	c := HTTPMonitor{Client: &http.Client{Timeout: time.Second}}
	urls := []string{server.URL, server.URL + "/missing", "invalid-url"}
	want := Stats{Total: 3, OK: 1, Other: 1, Errors: 1}
	_, a := sequential(c, urls)
	_, b := concurrent(c, urls)
	var d Stats
	for r := range stream(c, urls) {
		d.Add(r)
	}
	if a != want || b != want || d != want {
		t.Fatalf("unexpected stats: %v %v %v", a, b, d)
	}
}
