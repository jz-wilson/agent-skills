package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

const (
	maxBody    = 32 << 20         // refuse to buffer more than 32 MiB
	maxBackoff = 30 * time.Second // also caps a server's Retry-After
)

// Client does idempotent GETs with a timeout, a User-Agent and a few bounded retries.
type Client struct {
	http      *http.Client
	userAgent string
	log       *slog.Logger
	tries     int           // total attempts, including the first
	backoff   time.Duration // first wait; doubles each attempt
}

func newClient(userAgent string, timeout time.Duration, tries int, log *slog.Logger) *Client {
	return &Client{
		http:      &http.Client{Timeout: timeout}, // never http.DefaultClient: it has no timeout
		userAgent: userAgent,
		log:       log,
		tries:     max(tries, 1),
		backoff:   500 * time.Millisecond,
	}
}

// Get returns the response body of a 2xx reply. Only GET is retried, because
// only GET is safe to repeat.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		body, retryAfter, err := c.get(ctx, url)
		if err == nil {
			return body, nil
		}
		if retryAfter < 0 || attempt+1 >= c.tries {
			return nil, err
		}
		wait := retryAfter
		if wait == 0 {
			wait = c.backoff << attempt
		}
		wait = min(wait, maxBackoff)
		c.log.Warn("retrying", "wait", wait, "attempt", attempt+1, "of", c.tries, "err", err)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// get makes one attempt. retryAfter is -1 when retrying cannot help, 0 for
// "retry with the default backoff", or the server's requested wait.
func (c *Client) get(ctx context.Context, url string) (body []byte, retryAfter time.Duration, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, -1, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, -1, err // we were cancelled: do not retry
		}
		return nil, 0, err // transport error or timeout: worth another try
	}
	defer resp.Body.Close() //nolint:errcheck // read-only: a close error loses no data

	switch {
	case resp.StatusCode/100 == 2:
		// Read one byte past the cap so an oversized body is an error, not silently cut.
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
		if err == nil && len(body) > maxBody {
			err = fmt.Errorf("response over %d MiB (GET %s)", maxBody>>20, url)
		}
		return body, -1, err
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return nil, parseRetryAfter(resp.Header.Get("Retry-After")), fmt.Errorf("%s (GET %s)", resp.Status, url)
	default:
		// Cause first: a capped table cell keeps "404 Not Found", not the URL.
		return nil, -1, fmt.Errorf("%s (GET %s)", resp.Status, url) // 4xx: our request is wrong
	}
}

// parseRetryAfter reads both forms: delay-seconds and an HTTP date. 0 means absent.
func parseRetryAfter(v string) time.Duration {
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(time.Until(t), 0)
	}
	return 0
}
