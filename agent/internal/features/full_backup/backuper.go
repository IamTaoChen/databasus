package full_backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"

	"databasus-agent/internal/config"
	"databasus-agent/internal/features/api"
)

const (
	checkInterval = 30 * time.Second
	retryDelay    = 1 * time.Minute
	uploadTimeout = 30 * time.Minute
)

var retryDelayOverride *time.Duration

type CmdBuilder func(ctx context.Context) *exec.Cmd

// FullBackuper runs pg_basebackup when the WAL chain is broken or a scheduled backup is due.
//
// Every 30 seconds it checks two conditions via the Databasus API:
//  1. WAL chain validity — if broken or no full backup exists, triggers an immediate basebackup.
//  2. Scheduled backup time — if the next full backup time has passed, triggers a basebackup.
//
// Only one basebackup runs at a time (guarded by atomic bool).
// On failure the error is reported to the server and the backup retries after 1 minute, indefinitely.
// WAL segment uploads (handled by wal.Streamer) continue independently and are not paused.
//
// pg_basebackup runs as "pg_basebackup -Ft -D - -X none --verbose". Stdout (tar) is zstd-compressed
// and uploaded to the server. Stderr is parsed for WAL start/stop segment names (LSN → segment arithmetic).
type FullBackuper struct {
	cfg        *config.Config
	apiClient  *api.Client
	log        *slog.Logger
	isRunning  atomic.Bool
	cmdBuilder CmdBuilder
}

func NewFullBackuper(cfg *config.Config, apiClient *api.Client, log *slog.Logger) *FullBackuper {
	backuper := &FullBackuper{
		cfg:       cfg,
		apiClient: apiClient,
		log:       log,
	}

	backuper.cmdBuilder = backuper.defaultCmdBuilder

	return backuper
}

func (backuper *FullBackuper) Run(ctx context.Context) {
	backuper.log.Info("Full backuper started")

	backuper.checkAndRunIfNeeded(ctx)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			backuper.log.Info("Full backuper stopping")
			return
		case <-ticker.C:
			backuper.checkAndRunIfNeeded(ctx)
		}
	}
}

func (backuper *FullBackuper) checkAndRunIfNeeded(ctx context.Context) {
	if backuper.isRunning.Load() {
		backuper.log.Debug("Skipping check: basebackup already in progress")
		return
	}

	chainResp, err := backuper.apiClient.CheckWalChainValidity(ctx)
	if err != nil {
		backuper.log.Error("Failed to check WAL chain validity", "error", err)
		return
	}

	if !chainResp.IsValid {
		backuper.log.Info("WAL chain is invalid, triggering basebackup",
			"error", chainResp.Error,
			"lastContiguousSegment", chainResp.LastContiguousSegment,
		)

		backuper.runBasebackupWithRetry(ctx)

		return
	}

	nextTimeResp, err := backuper.apiClient.GetNextFullBackupTime(ctx)
	if err != nil {
		backuper.log.Error("Failed to check next full backup time", "error", err)
		return
	}

	if nextTimeResp.NextFullBackupTime == nil || !nextTimeResp.NextFullBackupTime.After(time.Now().UTC()) {
		backuper.log.Info("Scheduled full backup is due, triggering basebackup")
		backuper.runBasebackupWithRetry(ctx)

		return
	}

	backuper.log.Debug("No basebackup needed",
		"nextFullBackupTime", nextTimeResp.NextFullBackupTime,
	)
}

func (backuper *FullBackuper) runBasebackupWithRetry(ctx context.Context) {
	if !backuper.isRunning.CompareAndSwap(false, true) {
		backuper.log.Debug("Skipping basebackup: already running")
		return
	}
	defer backuper.isRunning.Store(false)

	for {
		if ctx.Err() != nil {
			return
		}

		backuper.log.Info("Starting pg_basebackup")

		err := backuper.executeAndUploadBasebackup(ctx)
		if err == nil {
			backuper.log.Info("Basebackup completed successfully")
			return
		}

		backuper.log.Error("Basebackup failed", "error", err)
		backuper.reportError(ctx, err.Error())

		delay := retryDelay
		if retryDelayOverride != nil {
			delay = *retryDelayOverride
		}

		backuper.log.Info("Retrying basebackup after delay", "delay", delay)

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

func (backuper *FullBackuper) executeAndUploadBasebackup(ctx context.Context) error {
	cmd := backuper.cmdBuilder(ctx)

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pg_basebackup: %w", err)
	}

	var compressedBuf bytes.Buffer
	compressErr := backuper.compressToBuffer(&compressedBuf, stdoutPipe)

	cmdErr := cmd.Wait()
	if cmdErr != nil {
		return fmt.Errorf("pg_basebackup exited with error: %w (stderr: %s)", cmdErr, stderrBuf.String())
	}

	if compressErr != nil {
		return fmt.Errorf("compress basebackup: %w", compressErr)
	}

	stderrStr := stderrBuf.String()
	backuper.log.Debug("pg_basebackup stderr", "stderr", stderrStr)

	startSegment, stopSegment, err := ParseBasebackupStderr(stderrStr)
	if err != nil {
		return fmt.Errorf("parse pg_basebackup stderr: %w", err)
	}

	backuper.log.Info("Basebackup WAL segments parsed",
		"startSegment", startSegment,
		"stopSegment", stopSegment,
		"compressedSize", compressedBuf.Len(),
	)

	uploadCtx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	if err := backuper.apiClient.UploadBasebackup(uploadCtx, startSegment, stopSegment, &compressedBuf); err != nil {
		return fmt.Errorf("upload basebackup: %w", err)
	}

	return nil
}

func (backuper *FullBackuper) compressToBuffer(dst *bytes.Buffer, reader io.Reader) error {
	encoder, err := zstd.NewWriter(dst,
		zstd.WithEncoderLevel(zstd.EncoderLevel(5)),
		zstd.WithEncoderCRC(true),
	)
	if err != nil {
		return fmt.Errorf("create zstd encoder: %w", err)
	}

	if _, err := io.Copy(encoder, reader); err != nil {
		_ = encoder.Close()
		return fmt.Errorf("compress: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return fmt.Errorf("close encoder: %w", err)
	}

	return nil
}

func (backuper *FullBackuper) reportError(ctx context.Context, errMsg string) {
	if err := backuper.apiClient.ReportBackupError(ctx, errMsg); err != nil {
		backuper.log.Error("Failed to report error to server", "error", err)
	}
}

func (backuper *FullBackuper) defaultCmdBuilder(ctx context.Context) *exec.Cmd {
	switch backuper.cfg.PgType {
	case "docker":
		return backuper.buildDockerCmd(ctx)
	default:
		return backuper.buildHostCmd(ctx)
	}
}

func (backuper *FullBackuper) buildHostCmd(ctx context.Context) *exec.Cmd {
	binary := "pg_basebackup"
	if backuper.cfg.PgHostBinDir != "" {
		binary = filepath.Join(backuper.cfg.PgHostBinDir, "pg_basebackup")
	}

	cmd := exec.CommandContext(ctx, binary,
		"-Ft", "-D", "-", "-X", "none", "--verbose",
		"-h", backuper.cfg.PgHost,
		"-p", fmt.Sprintf("%d", backuper.cfg.PgPort),
		"-U", backuper.cfg.PgUser,
	)

	cmd.Env = append(os.Environ(), "PGPASSWORD="+backuper.cfg.PgPassword)

	return cmd
}

func (backuper *FullBackuper) buildDockerCmd(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "docker", "exec",
		"-e", "PGPASSWORD="+backuper.cfg.PgPassword,
		"-i", backuper.cfg.PgDockerContainerName,
		"pg_basebackup",
		"-Ft", "-D", "-", "-X", "none", "--verbose",
		"-h", backuper.cfg.PgHost,
		"-p", fmt.Sprintf("%d", backuper.cfg.PgPort),
		"-U", backuper.cfg.PgUser,
	)

	return cmd
}
