package full_backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-agent/internal/config"
	"databasus-agent/internal/features/api"
	"databasus-agent/internal/logger"
)

const (
	testChainValidPath     = "/api/v1/backups/postgres/wal/is-wal-chain-valid-since-last-full-backup"
	testNextBackupTimePath = "/api/v1/backups/postgres/wal/next-full-backup-time"
	testUploadPath         = "/api/v1/backups/postgres/wal/upload"
	testReportErrorPath    = "/api/v1/backups/postgres/wal/error"
)

func Test_RunFullBackup_WhenChainBroken_BasebackupTriggered(t *testing.T) {
	var mu sync.Mutex
	var uploadReceived bool
	var uploadHeaders http.Header
	var uploadQuery string

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChainValidPath:
			writeJSON(w, api.WalChainValidityResponse{
				IsValid:               false,
				Error:                 "wal_chain_broken",
				LastContiguousSegment: "000000010000000100000011",
			})
		case testUploadPath:
			mu.Lock()
			uploadReceived = true
			uploadHeaders = r.Header.Clone()
			uploadQuery = r.URL.RawQuery
			mu.Unlock()

			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	fb := newTestFullBackuper(server.URL)
	fb.cmdBuilder = mockCmdBuilder(t, "test-backup-data", validStderr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go fb.Run(ctx)
	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return uploadReceived
	}, 5*time.Second)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	assert.True(t, uploadReceived)
	assert.Equal(t, "basebackup", uploadHeaders.Get("X-Upload-Type"))
	assert.Equal(t, "application/octet-stream", uploadHeaders.Get("Content-Type"))
	assert.Equal(t, "test-token", uploadHeaders.Get("Authorization"))
	assert.Contains(t, uploadQuery, "fullBackupWalStartSegment=000000010000000000000002")
	assert.Contains(t, uploadQuery, "fullBackupWalStopSegment=000000010000000000000002")
}

func Test_RunFullBackup_WhenScheduledBackupDue_BasebackupTriggered(t *testing.T) {
	var mu sync.Mutex
	var uploadReceived bool

	pastTime := time.Now().UTC().Add(-1 * time.Hour)

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChainValidPath:
			writeJSON(w, api.WalChainValidityResponse{IsValid: true})
		case testNextBackupTimePath:
			writeJSON(w, api.NextFullBackupTimeResponse{NextFullBackupTime: &pastTime})
		case testUploadPath:
			mu.Lock()
			uploadReceived = true
			mu.Unlock()

			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	fb := newTestFullBackuper(server.URL)
	fb.cmdBuilder = mockCmdBuilder(t, "scheduled-backup-data", validStderr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go fb.Run(ctx)
	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return uploadReceived
	}, 5*time.Second)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	assert.True(t, uploadReceived)
}

func Test_RunFullBackup_WhenNoFullBackupExists_ImmediateBasebackupTriggered(t *testing.T) {
	var mu sync.Mutex
	var uploadReceived bool

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChainValidPath:
			writeJSON(w, api.WalChainValidityResponse{
				IsValid: false,
				Error:   "no_full_backup",
			})
		case testUploadPath:
			mu.Lock()
			uploadReceived = true
			mu.Unlock()

			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	fb := newTestFullBackuper(server.URL)
	fb.cmdBuilder = mockCmdBuilder(t, "first-backup-data", validStderr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go fb.Run(ctx)
	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return uploadReceived
	}, 5*time.Second)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	assert.True(t, uploadReceived)
}

func Test_RunFullBackup_WhenUploadFails_RetriesAfterDelay(t *testing.T) {
	var mu sync.Mutex
	var uploadAttempts int
	var errorReported bool

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChainValidPath:
			writeJSON(w, api.WalChainValidityResponse{
				IsValid: false,
				Error:   "no_full_backup",
			})
		case testUploadPath:
			_, _ = io.ReadAll(r.Body)

			mu.Lock()
			uploadAttempts++
			attempt := uploadAttempts
			mu.Unlock()

			if attempt == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"storage unavailable"}`))
				return
			}

			w.WriteHeader(http.StatusNoContent)
		case testReportErrorPath:
			mu.Lock()
			errorReported = true
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	fb := newTestFullBackuper(server.URL)
	fb.cmdBuilder = mockCmdBuilder(t, "retry-backup-data", validStderr())

	origRetryDelay := retryDelay
	setRetryDelay(100 * time.Millisecond)
	defer setRetryDelay(origRetryDelay)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go fb.Run(ctx)
	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return uploadAttempts >= 2
	}, 10*time.Second)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	assert.GreaterOrEqual(t, uploadAttempts, 2)
	assert.True(t, errorReported)
}

func Test_RunFullBackup_WhenAlreadyRunning_SkipsExecution(t *testing.T) {
	var mu sync.Mutex
	var uploadCount int

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChainValidPath:
			writeJSON(w, api.WalChainValidityResponse{
				IsValid: false,
				Error:   "no_full_backup",
			})
		case testUploadPath:
			_, _ = io.ReadAll(r.Body)

			mu.Lock()
			uploadCount++
			mu.Unlock()

			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	fb := newTestFullBackuper(server.URL)
	fb.cmdBuilder = mockCmdBuilder(t, "data", validStderr())

	fb.isRunning.Store(true)

	fb.checkAndRunIfNeeded(context.Background())

	mu.Lock()
	count := uploadCount
	mu.Unlock()

	assert.Equal(t, 0, count, "should not trigger backup when already running")
}

func Test_RunFullBackup_WhenContextCancelled_StopsCleanly(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChainValidPath:
			writeJSON(w, api.WalChainValidityResponse{
				IsValid: false,
				Error:   "no_full_backup",
			})
		case testUploadPath:
			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusInternalServerError)
		case testReportErrorPath:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	fb := newTestFullBackuper(server.URL)
	fb.cmdBuilder = mockCmdBuilder(t, "data", validStderr())

	origRetryDelay := retryDelay
	setRetryDelay(5 * time.Second)
	defer setRetryDelay(origRetryDelay)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		fb.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run should have stopped after context cancellation")
	}
}

func Test_RunFullBackup_WhenChainValidAndNotScheduled_NoBasebackupTriggered(t *testing.T) {
	var uploadReceived atomic.Bool

	futureTime := time.Now().UTC().Add(24 * time.Hour)

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChainValidPath:
			writeJSON(w, api.WalChainValidityResponse{IsValid: true})
		case testNextBackupTimePath:
			writeJSON(w, api.NextFullBackupTimeResponse{NextFullBackupTime: &futureTime})
		case testUploadPath:
			uploadReceived.Store(true)

			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	fb := newTestFullBackuper(server.URL)
	fb.cmdBuilder = mockCmdBuilder(t, "data", validStderr())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go fb.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	cancel()

	assert.False(t, uploadReceived.Load(), "should not trigger backup when chain valid and not scheduled")
}

func Test_RunFullBackup_WhenStderrParsingFails_ReportsErrorAndRetries(t *testing.T) {
	var mu sync.Mutex
	var errorReported bool
	var uploadAttempts int

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChainValidPath:
			writeJSON(w, api.WalChainValidityResponse{
				IsValid: false,
				Error:   "no_full_backup",
			})
		case testUploadPath:
			mu.Lock()
			uploadAttempts++
			mu.Unlock()

			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		case testReportErrorPath:
			mu.Lock()
			errorReported = true
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	fb := newTestFullBackuper(server.URL)
	fb.cmdBuilder = mockCmdBuilder(t, "data", "pg_basebackup: unexpected output with no LSN info")

	origRetryDelay := retryDelay
	setRetryDelay(100 * time.Millisecond)
	defer setRetryDelay(origRetryDelay)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go fb.Run(ctx)
	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return errorReported
	}, 2*time.Second)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	assert.True(t, errorReported)
	assert.Equal(t, 0, uploadAttempts, "should not upload when stderr parsing fails")
}

func Test_RunFullBackup_WhenNextBackupTimeNull_BasebackupTriggered(t *testing.T) {
	var mu sync.Mutex
	var uploadReceived bool

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChainValidPath:
			writeJSON(w, api.WalChainValidityResponse{IsValid: true})
		case testNextBackupTimePath:
			writeJSON(w, api.NextFullBackupTimeResponse{NextFullBackupTime: nil})
		case testUploadPath:
			mu.Lock()
			uploadReceived = true
			mu.Unlock()

			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	fb := newTestFullBackuper(server.URL)
	fb.cmdBuilder = mockCmdBuilder(t, "first-run-data", validStderr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go fb.Run(ctx)
	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return uploadReceived
	}, 5*time.Second)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	assert.True(t, uploadReceived)
}

func Test_RunFullBackup_WhenUploadSucceeds_BodyIsZstdCompressed(t *testing.T) {
	var mu sync.Mutex
	var receivedBody []byte

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testChainValidPath:
			writeJSON(w, api.WalChainValidityResponse{
				IsValid: false,
				Error:   "no_full_backup",
			})
		case testUploadPath:
			body, _ := io.ReadAll(r.Body)

			mu.Lock()
			receivedBody = body
			mu.Unlock()

			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	originalContent := "test-backup-content-for-compression-check"
	fb := newTestFullBackuper(server.URL)
	fb.cmdBuilder = mockCmdBuilder(t, originalContent, validStderr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go fb.Run(ctx)
	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(receivedBody) > 0
	}, 5*time.Second)
	cancel()

	mu.Lock()
	body := receivedBody
	mu.Unlock()

	decoder, err := zstd.NewReader(nil)
	require.NoError(t, err)
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(body, nil)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(decompressed))
}

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return server
}

func newTestFullBackuper(serverURL string) *FullBackuper {
	cfg := &config.Config{
		DatabasusHost: serverURL,
		DbID:          "test-db-id",
		Token:         "test-token",
		PgHost:        "localhost",
		PgPort:        5432,
		PgUser:        "postgres",
		PgPassword:    "password",
		PgType:        "host",
	}

	apiClient := api.NewClient(serverURL, cfg.Token, logger.GetLogger())

	return NewFullBackuper(cfg, apiClient, logger.GetLogger())
}

func mockCmdBuilder(t *testing.T, stdoutContent, stderrContent string) CmdBuilder {
	t.Helper()

	return func(ctx context.Context) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0],
			"-test.run=TestHelperProcess",
			"--",
			stdoutContent,
			stderrContent,
		)

		cmd.Env = append(os.Environ(), "GO_TEST_HELPER_PROCESS=1")

		return cmd
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}

	if len(args) >= 1 {
		_, _ = fmt.Fprint(os.Stdout, args[0])
	}

	if len(args) >= 2 {
		_, _ = fmt.Fprint(os.Stderr, args[1])
	}

	os.Exit(0)
}

func validStderr() string {
	return `pg_basebackup: initiating base backup, waiting for checkpoint to complete
pg_basebackup: checkpoint completed
pg_basebackup: write-ahead log start point: 0/2000028, on timeline 1
pg_basebackup: checkpoint redo point at 0/2000028
pg_basebackup: write-ahead log end point: 0/2000100
pg_basebackup: base backup completed`
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(v); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("condition not met within %v", timeout)
}

func setRetryDelay(d time.Duration) {
	retryDelayOverride = &d
}

func init() {
	retryDelayOverride = nil
}
