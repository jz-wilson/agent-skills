package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// flaky fails the first n requests with status, then serves "ok".
func flaky(t *testing.T, n int32, status int, retryAfter string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "test/1" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		if hits.Add(1) <= n {
			if retryAfter != "" {
				w.Header().Set("Retry-After", retryAfter)
			}
			http.Error(w, "nope", status)
			return
		}
		fmt.Fprint(w, "ok")
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func testClient() *Client {
	c := newClient("test/1", time.Second, 3, slog.New(slog.DiscardHandler))
	c.backoff = time.Millisecond
	return c
}

func TestRetriesServerErrors(t *testing.T) {
	srv, hits := flaky(t, 2, http.StatusServiceUnavailable, "")
	body, err := testClient().Get(t.Context(), srv.URL)
	if err != nil || string(body) != "ok" || hits.Load() != 3 {
		t.Fatalf("body=%q err=%v hits=%d", body, err, hits.Load())
	}
}

func TestGivesUpAfterTries(t *testing.T) {
	srv, hits := flaky(t, 99, http.StatusBadGateway, "")
	if _, err := testClient().Get(t.Context(), srv.URL); err == nil || hits.Load() != 3 {
		t.Fatalf("err=%v hits=%d", err, hits.Load())
	}
}

func TestClientErrorsAreNotRetried(t *testing.T) {
	srv, hits := flaky(t, 99, http.StatusNotFound, "")
	if _, err := testClient().Get(t.Context(), srv.URL); err == nil || hits.Load() != 1 {
		t.Fatalf("err=%v hits=%d", err, hits.Load())
	}
}

func TestRefusesOversizedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, maxBody+1))
	}))
	t.Cleanup(srv.Close)
	if _, err := testClient().Get(t.Context(), srv.URL); err == nil {
		t.Fatal("want an error for a body over maxBody")
	}
}

func TestHonorsRetryAfter(t *testing.T) {
	srv, _ := flaky(t, 1, http.StatusTooManyRequests, "1")
	start := time.Now()
	if _, err := testClient().Get(t.Context(), srv.URL); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < time.Second {
		t.Fatalf("retried after %v, want >= 1s", d)
	}
}
