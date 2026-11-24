package main

import (
	"net/http"
	"time"
)

type Result struct {
	URL      string
	Status   int
	Duration time.Duration
	Err      error
}

type Checker interface{ Check(string) Result }

type HTTPMonitor struct{ Client *http.Client }

func (m HTTPMonitor) Check(url string) Result {
	start := time.Now()
	response, err := m.Client.Get(url)
	result := Result{URL: url, Duration: time.Since(start), Err: err}
	if err == nil {
		defer response.Body.Close()
		result.Status = response.StatusCode
	}
	return result
}

type Stats struct{ Total, OK, Other, Errors int }

func (s *Stats) Add(r Result) {
	s.Total++
	switch {
	case r.Err != nil:
		s.Errors++
	case r.Status == http.StatusOK:
		s.OK++
	default:
		s.Other++
	}
}
