package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	chainValidPath     = "/api/v1/backups/postgres/wal/is-wal-chain-valid-since-last-full-backup"
	nextBackupTimePath = "/api/v1/backups/postgres/wal/next-full-backup-time"
	walUploadPath      = "/api/v1/backups/postgres/wal/upload/wal"
	fullStartPath      = "/api/v1/backups/postgres/wal/upload/full-start"
	fullCompletePath   = "/api/v1/backups/postgres/wal/upload/full-complete"
	reportErrorPath    = "/api/v1/backups/postgres/wal/error"
	versionPath        = "/api/v1/system/version"
	agentBinaryPath    = "/api/v1/system/agent"

	apiCallTimeout   = 30 * time.Second
	maxRetryAttempts = 3
	retryBaseDelay   = 1 * time.Second
)

type Client struct {
	json   *resty.Client
	stream *resty.Client
	host   string
	log    *slog.Logger
}

func NewClient(host, token string, log *slog.Logger) *Client {
	setAuth := func(_ *resty.Client, req *resty.Request) error {
		if token != "" {
			req.SetHeader("Authorization", token)
		}

		return nil
	}

	jsonClient := resty.New().
		SetTimeout(apiCallTimeout).
		SetRetryCount(maxRetryAttempts - 1).
		SetRetryWaitTime(retryBaseDelay).
		SetRetryMaxWaitTime(4 * retryBaseDelay).
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			return err != nil || resp.StatusCode() >= 500
		}).
		OnBeforeRequest(setAuth)

	streamClient := resty.New().
		OnBeforeRequest(setAuth)

	return &Client{
		json:   jsonClient,
		stream: streamClient,
		host:   host,
		log:    log,
	}
}

func (c *Client) CheckWalChainValidity(ctx context.Context) (*WalChainValidityResponse, error) {
	var resp WalChainValidityResponse

	_, err := c.json.R().
		SetContext(ctx).
		SetResult(&resp).
		Get(c.buildURL(chainValidPath))
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) GetNextFullBackupTime(ctx context.Context) (*NextFullBackupTimeResponse, error) {
	var resp NextFullBackupTimeResponse

	_, err := c.json.R().
		SetContext(ctx).
		SetResult(&resp).
		Get(c.buildURL(nextBackupTimePath))
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) ReportBackupError(ctx context.Context, errMsg string) error {
	_, err := c.json.R().
		SetContext(ctx).
		SetBody(reportErrorRequest{Error: errMsg}).
		Post(c.buildURL(reportErrorPath))

	return err
}

func (c *Client) UploadBasebackup(
	ctx context.Context,
	body io.Reader,
) (*UploadBasebackupResponse, error) {
	resp, err := c.stream.R().
		SetContext(ctx).
		SetBody(body).
		SetHeader("Content-Type", "application/octet-stream").
		SetDoNotParseResponse(true).
		Post(c.buildURL(fullStartPath))
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer func() { _ = resp.RawBody().Close() }()

	if resp.StatusCode() != http.StatusOK {
		respBody, _ := io.ReadAll(resp.RawBody())

		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode(), string(respBody))
	}

	var result UploadBasebackupResponse
	if err := json.NewDecoder(resp.RawBody()).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode upload response: %w", err)
	}

	return &result, nil
}

func (c *Client) FinalizeBasebackup(
	ctx context.Context,
	backupID string,
	startSegment string,
	stopSegment string,
) error {
	resp, err := c.json.R().
		SetContext(ctx).
		SetBody(finalizeBasebackupRequest{
			BackupID:     backupID,
			StartSegment: startSegment,
			StopSegment:  stopSegment,
		}).
		Post(c.buildURL(fullCompletePath))
	if err != nil {
		return fmt.Errorf("finalize request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("finalize failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

func (c *Client) FinalizeBasebackupWithError(
	ctx context.Context,
	backupID string,
	errMsg string,
) error {
	resp, err := c.json.R().
		SetContext(ctx).
		SetBody(finalizeBasebackupRequest{
			BackupID: backupID,
			Error:    &errMsg,
		}).
		Post(c.buildURL(fullCompletePath))
	if err != nil {
		return fmt.Errorf("finalize-with-error request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("finalize-with-error failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

func (c *Client) UploadWalSegment(
	ctx context.Context,
	segmentName string,
	body io.Reader,
) (*UploadWalSegmentResult, error) {
	resp, err := c.stream.R().
		SetContext(ctx).
		SetBody(body).
		SetHeader("Content-Type", "application/octet-stream").
		SetHeader("X-Wal-Segment-Name", segmentName).
		SetDoNotParseResponse(true).
		Post(c.buildURL(walUploadPath))
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer func() { _ = resp.RawBody().Close() }()

	switch resp.StatusCode() {
	case http.StatusNoContent:
		return &UploadWalSegmentResult{IsGapDetected: false}, nil

	case http.StatusConflict:
		var errResp uploadErrorResponse

		if err := json.NewDecoder(resp.RawBody()).Decode(&errResp); err != nil {
			return &UploadWalSegmentResult{IsGapDetected: true}, nil
		}

		return &UploadWalSegmentResult{
			IsGapDetected:       true,
			ExpectedSegmentName: errResp.ExpectedSegmentName,
			ReceivedSegmentName: errResp.ReceivedSegmentName,
		}, nil

	default:
		respBody, _ := io.ReadAll(resp.RawBody())

		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode(), string(respBody))
	}
}

func (c *Client) FetchServerVersion(ctx context.Context) (string, error) {
	var ver versionResponse

	_, err := c.json.R().
		SetContext(ctx).
		SetResult(&ver).
		Get(c.buildURL(versionPath))
	if err != nil {
		return "", err
	}

	return ver.Version, nil
}

func (c *Client) DownloadAgentBinary(ctx context.Context, arch, destPath string) error {
	resp, err := c.stream.R().
		SetContext(ctx).
		SetQueryParam("arch", arch).
		SetDoNotParseResponse(true).
		Get(c.buildURL(agentBinaryPath))
	if err != nil {
		return err
	}
	defer func() { _ = resp.RawBody().Close() }()

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned %d for agent download", resp.StatusCode())
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = io.Copy(f, resp.RawBody())

	return err
}

func (c *Client) buildURL(path string) string {
	return c.host + path
}
