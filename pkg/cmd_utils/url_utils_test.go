package cmd_utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDownloadFileThroughUrl_NonSuccessStatus(t *testing.T) {
	transport := &queueTransport{
		responses: []transportResponse{
			{
				response: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("boom")),
					Header:     make(http.Header),
				},
			},
		},
	}

	replaceDefaultClient(t, transport)

	file := filepath.Join(t.TempDir(), "test.bin")

	err := DownloadFileThroughUrl(file, "https://example.com/fail", 0)
	if err == nil {
		t.Fatalf("expected error for non-success status")
	}
	if !strings.Contains(err.Error(), "status code 500") {
		t.Fatalf("unexpected error: %v", err)
	}

	info, statErr := os.Stat(file)
	if statErr != nil {
		t.Fatalf("expected file to exist: %v", statErr)
	}
	if info.Size() != 0 {
		t.Fatalf("expected empty file after failure, got %d bytes", info.Size())
	}
}

func TestDownloadFileThroughUrl_RetryTruncatesFile(t *testing.T) {
	expected := []byte("abcdefghijklmnopqrstuvwxyz")

	var attempts int32
	transport := &queueTransport{
		responses: []transportResponse{
			{
				before: func() { atomic.AddInt32(&attempts, 1) },
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body: &partialReadCloser{
						data: expected[:len(expected)/2],
					},
					ContentLength: int64(len(expected)),
					Header:        make(http.Header),
				},
			},
			{
				before: func() { atomic.AddInt32(&attempts, 1) },
				response: &http.Response{
					StatusCode:    http.StatusOK,
					Body:          io.NopCloser(bytes.NewReader(expected)),
					ContentLength: int64(len(expected)),
					Header:        make(http.Header),
				},
			},
		},
	}

	replaceDefaultClient(t, transport)

	prevMin, prevMax := retryWaitMin, retryWaitMax
	retryWaitMin, retryWaitMax = 10*time.Millisecond, 20*time.Millisecond
	t.Cleanup(func() {
		retryWaitMin, retryWaitMax = prevMin, prevMax
	})

	file := filepath.Join(t.TempDir(), "retry.bin")

	if err := DownloadFileThroughUrl(file, "https://example.com/retry", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !bytes.Equal(data, expected) {
		t.Fatalf("expected %q, got %q", string(expected), string(data))
	}

	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func replaceDefaultClient(t *testing.T, transport http.RoundTripper) {
	t.Helper()

	prevClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: transport}
	t.Cleanup(func() {
		http.DefaultClient = prevClient
	})
}

type queueTransport struct {
	mu        sync.Mutex
	responses []transportResponse
}

type transportResponse struct {
	response *http.Response
	err      error
	before   func()
}

func (qt *queueTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	qt.mu.Lock()
	defer qt.mu.Unlock()

	if len(qt.responses) == 0 {
		return nil, fmt.Errorf("unexpected request to %s", req.URL)
	}

	next := qt.responses[0]
	qt.responses = qt.responses[1:]

	if next.before != nil {
		next.before()
	}

	if next.response != nil {
		next.response.Request = req
	}

	return next.response, next.err
}

type partialReadCloser struct {
	served bool
	data   []byte
}

func (p *partialReadCloser) Read(b []byte) (int, error) {
	if !p.served {
		p.served = true
		n := copy(b, p.data)
		return n, nil
	}
	return 0, io.ErrUnexpectedEOF
}

func (p *partialReadCloser) Close() error { return nil }
