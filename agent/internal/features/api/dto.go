package api

import "time"

type WalChainValidityResponse struct {
	IsValid               bool   `json:"isValid"`
	Error                 string `json:"error,omitempty"`
	LastContiguousSegment string `json:"lastContiguousSegment,omitempty"`
}

type NextFullBackupTimeResponse struct {
	NextFullBackupTime *time.Time `json:"nextFullBackupTime"`
}

type UploadWalSegmentResult struct {
	IsGapDetected       bool
	ExpectedSegmentName string
	ReceivedSegmentName string
}

type reportErrorRequest struct {
	Error string `json:"error"`
}

type versionResponse struct {
	Version string `json:"version"`
}

type UploadBasebackupResponse struct {
	BackupID string `json:"backupId"`
}

type finalizeBasebackupRequest struct {
	BackupID     string  `json:"backupId"`
	StartSegment string  `json:"startSegment"`
	StopSegment  string  `json:"stopSegment"`
	Error        *string `json:"error,omitempty"`
}

type uploadErrorResponse struct {
	Error               string `json:"error"`
	ExpectedSegmentName string `json:"expectedSegmentName"`
	ReceivedSegmentName string `json:"receivedSegmentName"`
}
