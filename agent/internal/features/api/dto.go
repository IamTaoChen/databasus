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

type uploadErrorResponse struct {
	Error               string `json:"error"`
	ExpectedSegmentName string `json:"expectedSegmentName"`
	ReceivedSegmentName string `json:"receivedSegmentName"`
}
