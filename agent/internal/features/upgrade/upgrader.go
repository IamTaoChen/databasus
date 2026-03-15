package upgrade

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"databasus-agent/internal/features/api"
)

// CheckAndUpdate ensures the agent binary matches the server's expected version.
// It fetches the server version, downloads the new binary if different, verifies it,
// replaces the current executable, and re-execs the process with the same arguments.
// Skipped in development mode. Runs once on startup before the main agent loop.
func CheckAndUpdate(apiClient *api.Client, currentVersion string, isDev bool, log *slog.Logger) error {
	if isDev {
		log.Info("Skipping update check (development mode)")
		return nil
	}

	serverVersion, err := apiClient.FetchServerVersion(context.Background())
	if err != nil {
		log.Warn("Could not reach server for update check, continuing", "error", err)

		return fmt.Errorf(
			"unable to check version, please verify Databasus server is available: %w",
			err,
		)
	}

	if serverVersion == currentVersion {
		log.Info("Agent version is up to date", "version", currentVersion)
		return nil
	}

	log.Info("Updating agent...", "current", currentVersion, "target", serverVersion)

	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}

	tempPath := selfPath + ".update"

	defer func() {
		_ = os.Remove(tempPath)
	}()

	if err := apiClient.DownloadAgentBinary(context.Background(), runtime.GOARCH, tempPath); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	if err := os.Chmod(tempPath, 0o755); err != nil {
		return fmt.Errorf("failed to set permissions on update: %w", err)
	}

	if err := verifyBinary(tempPath, serverVersion); err != nil {
		return fmt.Errorf("update verification failed: %w", err)
	}

	if err := os.Rename(tempPath, selfPath); err != nil {
		return fmt.Errorf("failed to replace binary (try --skip-update if this persists): %w", err)
	}

	log.Info("Update complete, re-executing...")

	return syscall.Exec(selfPath, os.Args, os.Environ())
}

func verifyBinary(binaryPath, expectedVersion string) error {
	cmd := exec.CommandContext(context.Background(), binaryPath, "version")

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("binary failed to execute: %w", err)
	}

	got := strings.TrimSpace(string(output))
	if got != expectedVersion {
		return fmt.Errorf("version mismatch: expected %q, got %q", expectedVersion, got)
	}

	return nil
}
