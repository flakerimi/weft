package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestPercentile(t *testing.T) {
	ms := func(n ...int) []time.Duration {
		d := make([]time.Duration, len(n))
		for i, v := range n {
			d[i] = time.Duration(v) * time.Millisecond
		}
		return d
	}
	sorted := ms(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	for _, c := range []struct {
		p    float64
		want time.Duration
	}{{0, time.Millisecond}, {50, 5 * time.Millisecond}, {99, 10 * time.Millisecond}, {100, 10 * time.Millisecond}} {
		if got := percentile(sorted, c.p); got != c.want {
			t.Errorf("percentile(1..10 ms, %v) = %v, want %v", c.p, got, c.want)
		}
	}
	if got := percentile(nil, 50); got != 0 {
		t.Errorf("percentile(none) = %v, want 0", got)
	}
}

// Requests made during the warm-up are not counted; the rest are, over the
// measured duration only.
func TestLoadCountsOnlyAfterWarmup(t *testing.T) {
	var served, encodings atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served.Add(1)
		if r.Header.Get("Accept-Encoding") == "br, gzip" {
			encodings.Add(1)
		}
		time.Sleep(time.Millisecond)
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	res, err := runLoad(srv.URL, 4, 200*time.Millisecond, 400*time.Millisecond, "br, gzip")
	if err != nil {
		t.Fatal(err)
	}
	if res.Requests == 0 || res.Errors != 0 {
		t.Fatalf("requests %d, errors %d; want some and none", res.Requests, res.Errors)
	}
	if int64(res.Requests) >= served.Load() {
		t.Errorf("counted %d of %d served requests; the warm-up's should not count", res.Requests, served.Load())
	}
	if encodings.Load() != served.Load() {
		t.Errorf("%d of %d requests sent Accept-Encoding: br, gzip", encodings.Load(), served.Load())
	}
	if want := float64(res.Requests) / 0.4; res.RPS < want*0.99 || res.RPS > want*1.01 {
		t.Errorf("rps %.0f, want requests / measured seconds = %.0f", res.RPS, want)
	}
	if res.P50Ms < 1 || res.P99Ms < res.P50Ms {
		t.Errorf("p50 %.2f ms, p99 %.2f ms; want p50 >= 1 ms (the handler sleeps) and p99 >= p50", res.P50Ms, res.P99Ms)
	}
	if res.Bytes != int64(res.Requests)*5 {
		t.Errorf("bytes %d, want 5 per counted request (%d)", res.Bytes, res.Requests*5)
	}
}

func TestLoadCountsErrorStatuses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusInternalServerError)
	}))
	defer srv.Close()
	res, err := runLoad(srv.URL, 2, 0, 100*time.Millisecond, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Errors == 0 || res.Requests != 0 {
		t.Errorf("errors %d, requests %d; a 500 is an error, not a request served", res.Errors, res.Requests)
	}
}

// TTFB is timed to the final response, past any 103 Early Hints, each request
// on a new connection; a few warm-up requests come first and are not timed.
func TestTTFB(t *testing.T) {
	var conns atomic.Int64
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", "</a.js>; rel=modulepreload")
		w.WriteHeader(http.StatusEarlyHints)
		time.Sleep(3 * time.Millisecond)
		w.Write([]byte("page"))
	}))
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			conns.Add(1)
		}
	}
	srv.Start()
	defer srv.Close()

	res, err := runTTFB(srv.URL, 9, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.N != 9 || res.MedianMs < 3 || res.FirstByteMedianMs >= res.MedianMs {
		t.Errorf("n %d, median %.2f ms, first byte %.2f ms; want 9, >= 3 ms, the 103 before the final response", res.N, res.MedianMs, res.FirstByteMedianMs)
	}
	if res.P25Ms > res.MedianMs || res.P75Ms < res.MedianMs {
		t.Errorf("p25 %.2f, median %.2f, p75 %.2f are out of order", res.P25Ms, res.MedianMs, res.P75Ms)
	}
	// Every timed request is in the result, so rounds can be pooled.
	if len(res.SamplesMs) != 9 || len(res.FirstByteMs) != 9 {
		t.Fatalf("%d samples and %d first bytes; want one of each per timed request", len(res.SamplesMs), len(res.FirstByteMs))
	}
	for i, s := range res.SamplesMs {
		if s < 3 || res.FirstByteMs[i] >= s || s < res.MinMs || s > res.MaxMs {
			t.Errorf("sample %d: %.2f ms, first byte %.2f ms; want >= 3 ms, after the 103, inside [%.2f, %.2f]", i, s, res.FirstByteMs[i], res.MinMs, res.MaxMs)
		}
	}
	if want := int64(9 + ttfbWarmup); conns.Load() != want {
		t.Errorf("%d connections for %d requests; want a new one each", conns.Load(), want)
	}
}
