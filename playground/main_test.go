package main

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlaygroundServes(t *testing.T) {
	srv := httptest.NewServer(handler())
	defer srv.Close()

	res, err := srv.Client().Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.Contains(string(b), `<div id="app">`) {
		t.Fatalf("GET /: status %d, body %.200s", res.StatusCode, b)
	}

	res, err = srv.Client().Get(srv.URL + "/api/info")
	if err != nil {
		t.Fatal(err)
	}
	var info struct{ Version string }
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil || info.Version == "" {
		t.Fatalf("GET /api/info: %v, %+v", err, info)
	}
}
