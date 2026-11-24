package main

import "sync"

func sequential(c Checker, urls []string) ([]Result, Stats) {
	var results []Result
	var stats Stats
	for _, url := range urls {
		r := c.Check(url)
		results = append(results, r)
		stats.Add(r)
	}
	return results, stats
}

func concurrent(c Checker, urls []string) ([]Result, Stats) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []Result
	var stats Stats
	for _, url := range urls {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			r := c.Check(address)
			mu.Lock()
			results = append(results, r)
			stats.Add(r)
			mu.Unlock()
		}(url)
	}
	wg.Wait()
	return results, stats
}

func stream(c Checker, urls []string) <-chan Result {
	results := make(chan Result, len(urls))
	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			results <- c.Check(address)
		}(url)
	}
	go func() { wg.Wait(); close(results) }()
	return results
}
