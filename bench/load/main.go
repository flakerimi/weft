// Command load measures one URL for the benchmark in bench/, from its own
// process, and prints the result as JSON.
//
// Load (the default): -c connections ask for the URL over and over, keeping
// their connections alive, for -warmup and then -d; only requests that start
// and finish inside -d count. It reports requests per second, p50 and p99.
//
// TTFB (-ttfb N): N requests one after another, each on a new connection,
// after a few untimed ones. Each is timed to the final response's headers,
// past any 103 Early Hints; the first byte of anything (the 103 included) is
// reported beside it. Every timed request is listed too, in order, so that
// several series can be pooled.
//
// Usage:
//
//	load -url http://127.0.0.1:4301/ [-c 64] [-d 10s] [-warmup 2s] [-encoding "br, gzip"]
//	load -url http://127.0.0.1:4301/ -ttfb 50
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptrace"
	"os"
	"slices"
	"sync"
	"time"
)

// ttfbWarmup is how many untimed requests come before a TTFB series.
const ttfbWarmup = 5

type LoadResult struct {
	URL         string  `json:"url"`
	Connections int     `json:"connections"`
	WarmupS     float64 `json:"warmup_s"`
	DurationS   float64 `json:"duration_s"`
	Requests    int     `json:"requests"`
	Errors      int     `json:"errors"`
	Bytes       int64   `json:"bytes"`
	RPS         float64 `json:"rps"`
	P50Ms       float64 `json:"p50_ms"`
	P99Ms       float64 `json:"p99_ms"`
}

type TTFBResult struct {
	URL               string    `json:"url"`
	N                 int       `json:"n"`
	MedianMs          float64   `json:"median_ms"`
	P25Ms             float64   `json:"p25_ms"`
	P75Ms             float64   `json:"p75_ms"`
	MinMs             float64   `json:"min_ms"`
	MaxMs             float64   `json:"max_ms"`
	FirstByteMedianMs float64   `json:"first_byte_median_ms"`
	SamplesMs         []float64 `json:"samples_ms"`
	FirstByteMs       []float64 `json:"first_byte_ms"`
}

// percentile is the nearest-rank percentile p (0 to 100) of sorted durations.
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	i := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	return sorted[max(0, min(i, len(sorted)-1))]
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func newRequest(url, encoding string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if encoding != "" {
		req.Header.Set("Accept-Encoding", encoding)
	}
	return req, nil
}

// fetch asks once and reads the whole body; any status but 200 is an error.
func fetch(rt http.RoundTripper, req *http.Request) (int64, error) {
	res, err := rt.RoundTrip(req)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if err != nil {
		return n, err
	}
	if res.StatusCode != http.StatusOK {
		return n, fmt.Errorf("%s: status %d", req.URL, res.StatusCode)
	}
	return n, nil
}

func runLoad(url string, conns int, warmup, duration time.Duration, encoding string) (LoadResult, error) {
	tr := &http.Transport{MaxIdleConnsPerHost: conns, MaxConnsPerHost: conns, DisableCompression: true}
	defer tr.CloseIdleConnections()
	if _, err := newRequest(url, encoding); err != nil {
		return LoadResult{}, err
	}

	type worker struct {
		latencies []time.Duration
		errors    int
		bytes     int64
	}
	workers := make([]worker, conns)
	from := time.Now().Add(warmup)
	until := from.Add(duration)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Go(func() {
			w := &workers[i]
			req, _ := newRequest(url, encoding)
			for {
				start := time.Now()
				if !start.Before(until) {
					return
				}
				n, err := fetch(tr, req)
				end := time.Now()
				if start.Before(from) || end.After(until) {
					continue
				}
				if err != nil {
					w.errors++
					continue
				}
				w.latencies = append(w.latencies, end.Sub(start))
				w.bytes += n
			}
		})
	}
	wg.Wait()

	res := LoadResult{URL: url, Connections: conns, WarmupS: warmup.Seconds(), DurationS: duration.Seconds()}
	var all []time.Duration
	for _, w := range workers {
		all = append(all, w.latencies...)
		res.Errors += w.errors
		res.Bytes += w.bytes
	}
	slices.Sort(all)
	res.Requests = len(all)
	res.RPS = float64(len(all)) / duration.Seconds()
	res.P50Ms = ms(percentile(all, 50))
	res.P99Ms = ms(percentile(all, 99))
	return res, nil
}

func runTTFB(url string, n int, encoding string) (TTFBResult, error) {
	if n < 1 {
		return TTFBResult{}, errors.New("load: -ttfb needs at least one request")
	}
	tr := &http.Transport{DisableKeepAlives: true, DisableCompression: true}
	defer tr.CloseIdleConnections()
	base, err := newRequest(url, encoding)
	if err != nil {
		return TTFBResult{}, err
	}
	var final, first []time.Duration
	var samples, firstBytes []float64
	for i := range ttfbWarmup + n {
		var firstByte time.Time
		req := base.WithContext(httptrace.WithClientTrace(base.Context(), &httptrace.ClientTrace{
			GotFirstResponseByte: func() { firstByte = time.Now() },
		}))
		start := time.Now()
		res, err := tr.RoundTrip(req)
		if err != nil {
			return TTFBResult{}, err
		}
		headers := time.Now()
		_, err = io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if err != nil {
			return TTFBResult{}, err
		}
		if res.StatusCode != http.StatusOK {
			return TTFBResult{}, fmt.Errorf("%s: status %d", url, res.StatusCode)
		}
		if i >= ttfbWarmup {
			final = append(final, headers.Sub(start))
			first = append(first, firstByte.Sub(start))
			samples = append(samples, ms(headers.Sub(start)))
			firstBytes = append(firstBytes, ms(firstByte.Sub(start)))
		}
	}
	slices.Sort(final)
	slices.Sort(first)
	return TTFBResult{
		URL:               url,
		N:                 n,
		MedianMs:          ms(percentile(final, 50)),
		P25Ms:             ms(percentile(final, 25)),
		P75Ms:             ms(percentile(final, 75)),
		MinMs:             ms(final[0]),
		MaxMs:             ms(final[len(final)-1]),
		FirstByteMedianMs: ms(percentile(first, 50)),
		SamplesMs:         samples,
		FirstByteMs:       firstBytes,
	}, nil
}

func main() {
	url := flag.String("url", "", "the URL to measure")
	conns := flag.Int("c", 64, "connections")
	duration := flag.Duration("d", 10*time.Second, "how long to measure")
	warmup := flag.Duration("warmup", 2*time.Second, "how long to load before measuring")
	encoding := flag.String("encoding", "br, gzip", "the Accept-Encoding to send")
	ttfb := flag.Int("ttfb", 0, "measure time to first byte over this many requests instead")
	flag.Parse()
	if *url == "" {
		fmt.Fprintln(os.Stderr, "load: -url is required")
		os.Exit(2)
	}
	var out any
	var err error
	if *ttfb > 0 {
		out, err = runTTFB(*url, *ttfb, *encoding)
	} else {
		out, err = runLoad(*url, *conns, *warmup, *duration, *encoding)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}
