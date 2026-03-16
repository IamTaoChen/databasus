package backups_controllers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	backups_dto "databasus-backend/internal/features/backups/backups/dto"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	"databasus-backend/internal/features/databases"
)

// PostgreWalBackupController handles WAL backup endpoints used by the databasus-cli agent.
// Authentication is via a plain agent token in the Authorization header (no Bearer prefix).
type PostgreWalBackupController struct {
	databaseService *databases.DatabaseService
	walService      *backups_services.PostgreWalBackupService
}

func (c *PostgreWalBackupController) RegisterRoutes(router *gin.RouterGroup) {
	walRoutes := router.Group("/backups/postgres/wal")

	walRoutes.GET("/next-full-backup-time", c.GetNextFullBackupTime)
	walRoutes.GET("/is-wal-chain-valid-since-last-full-backup", c.IsWalChainValidSinceLastBackup)
	walRoutes.POST("/error", c.ReportError)
	walRoutes.POST("/upload/wal", c.UploadWalSegment)
	walRoutes.POST("/upload/full-start", c.StartFullBackupUpload)
	walRoutes.POST("/upload/full-complete", c.CompleteFullBackupUpload)
	walRoutes.GET("/restore/plan", c.GetRestorePlan)
	walRoutes.GET("/restore/download", c.DownloadBackupFile)
}

// GetNextFullBackupTime
// @Summary Get next full backup time
// @Description Returns the next scheduled full basebackup time for the authenticated database
// @Tags backups-wal
// @Produce json
// @Security AgentToken
// @Success 200 {object} backups_dto.GetNextFullBackupTimeResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /backups/postgres/wal/next-full-backup-time [get]
func (c *PostgreWalBackupController) GetNextFullBackupTime(ctx *gin.Context) {
	database, err := c.getDatabase(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	response, err := c.walService.GetNextFullBackupTime(database)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// ReportError
// @Summary Report agent error
// @Description Records a fatal error from the agent against the database record and marks it as errored
// @Tags backups-wal
// @Accept json
// @Security AgentToken
// @Param request body backups_dto.ReportErrorRequest true "Error details"
// @Success 200
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /backups/postgres/wal/error [post]
func (c *PostgreWalBackupController) ReportError(ctx *gin.Context) {
	database, err := c.getDatabase(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	var request backups_dto.ReportErrorRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.walService.ReportError(database, request.Error); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

// IsWalChainValidSinceLastBackup
// @Summary Check WAL chain validity since last full backup
// @Description Checks whether the WAL chain is continuous since the last completed full backup.
// Returns isValid=true if the chain is intact, or isValid=false with error details if not.
// @Tags backups-wal
// @Produce json
// @Security AgentToken
// @Success 200 {object} backups_dto.IsWalChainValidResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /backups/postgres/wal/is-wal-chain-valid-since-last-full-backup [get]
func (c *PostgreWalBackupController) IsWalChainValidSinceLastBackup(ctx *gin.Context) {
	database, err := c.getDatabase(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	response, err := c.walService.IsWalChainValid(database)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// UploadWalSegment
// @Summary Stream upload a WAL segment
// @Description Accepts a zstd-compressed WAL segment binary stream and stores it in the database's configured storage.
// WAL segments are accepted unconditionally.
// @Tags backups-wal
// @Accept application/octet-stream
// @Security AgentToken
// @Param X-Wal-Segment-Name header string true "24-hex WAL segment identifier (e.g. 0000000100000001000000AB)"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /backups/postgres/wal/upload/wal [post]
func (c *PostgreWalBackupController) UploadWalSegment(ctx *gin.Context) {
	database, err := c.getDatabase(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	walSegmentName := ctx.GetHeader("X-Wal-Segment-Name")
	if walSegmentName == "" {
		ctx.JSON(
			http.StatusBadRequest,
			gin.H{"error": "X-Wal-Segment-Name is required for wal uploads"},
		)
		return
	}

	uploadErr := c.walService.UploadWalSegment(
		ctx.Request.Context(),
		database,
		walSegmentName,
		ctx.Request.Body,
	)

	if uploadErr != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": uploadErr.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// StartFullBackupUpload
// @Summary Stream upload a full basebackup (Phase 1)
// @Description Accepts a zstd-compressed basebackup binary stream and stores it in the database's configured storage.
// Returns a backupId that must be completed via /upload/full-complete with WAL segment names.
// @Tags backups-wal
// @Accept application/octet-stream
// @Produce json
// @Security AgentToken
// @Success 200 {object} backups_dto.UploadBasebackupResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /backups/postgres/wal/upload/full-start [post]
func (c *PostgreWalBackupController) StartFullBackupUpload(ctx *gin.Context) {
	database, err := c.getDatabase(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	backupID, uploadErr := c.walService.UploadBasebackup(
		ctx.Request.Context(),
		database,
		ctx.Request.Body,
	)

	if uploadErr != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": uploadErr.Error()})
		return
	}

	ctx.JSON(http.StatusOK, backups_dto.UploadBasebackupResponse{
		BackupID: backupID,
	})
}

// CompleteFullBackupUpload
// @Summary Complete a previously uploaded basebackup (Phase 2)
// @Description Sets WAL segment names and marks the basebackup as completed, or marks it as failed if an error is provided.
// @Tags backups-wal
// @Accept json
// @Security AgentToken
// @Param request body backups_dto.FinalizeBasebackupRequest true "Completion details"
// @Success 200
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /backups/postgres/wal/upload/full-complete [post]
func (c *PostgreWalBackupController) CompleteFullBackupUpload(ctx *gin.Context) {
	database, err := c.getDatabase(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	var request backups_dto.FinalizeBasebackupRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.walService.FinalizeBasebackup(
		database,
		request.BackupID,
		request.StartSegment,
		request.StopSegment,
		request.Error,
	); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

// GetRestorePlan
// @Summary Get restore plan
// @Description Resolves the full backup and all required WAL segments needed for recovery. Validates the WAL chain is continuous.
// @Tags backups-wal
// @Produce json
// @Security AgentToken
// @Param backupId query string false "UUID of a specific full backup to restore from; defaults to the most recent"
// @Success 200 {object} backups_dto.GetRestorePlanResponse
// @Failure 400 {object} map[string]string "Broken WAL chain or no backups available"
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /backups/postgres/wal/restore/plan [get]
func (c *PostgreWalBackupController) GetRestorePlan(ctx *gin.Context) {
	database, err := c.getDatabase(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	var backupID *uuid.UUID
	if raw := ctx.Query("backupId"); raw != "" {
		parsed, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backupId format"})
			return
		}

		backupID = &parsed
	}

	response, planErr, err := c.walService.GetRestorePlan(database, backupID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if planErr != nil {
		ctx.JSON(http.StatusBadRequest, planErr)
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// DownloadBackupFile
// @Summary Download a backup or WAL segment file for restore
// @Description Retrieves the backup file by ID (validated against the authenticated database), decrypts it server-side if encrypted, and streams the zstd-compressed result to the agent
// @Tags backups-wal
// @Produce application/octet-stream
// @Security AgentToken
// @Param backupId query string true "Backup ID from the restore plan response"
// @Success 200 {file} file
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /backups/postgres/wal/restore/download [get]
func (c *PostgreWalBackupController) DownloadBackupFile(ctx *gin.Context) {
	database, err := c.getDatabase(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	backupIDRaw := ctx.Query("backupId")
	if backupIDRaw == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "backupId is required"})
		return
	}

	backupID, err := uuid.Parse(backupIDRaw)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backupId format"})
		return
	}

	reader, err := c.walService.DownloadBackupFile(database, backupID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = reader.Close() }()

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Status(http.StatusOK)

	_, _ = io.Copy(ctx.Writer, reader)
}

func (c *PostgreWalBackupController) getDatabase(
	ctx *gin.Context,
) (*databases.Database, error) {
	token := ctx.GetHeader("Authorization")
	return c.databaseService.GetDatabaseByAgentToken(token)
}
