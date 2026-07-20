package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
	//"log"

	"excel-crud-app/internal/apperrors"
	"excel-crud-app/internal/middleware"
	"excel-crud-app/internal/models"
	"excel-crud-app/internal/response"
	"excel-crud-app/internal/services"

	"github.com/gin-gonic/gin"
)


// These are what a client polling /upload/status sees; the raw error is
// logged server-side via slog instead.
const (
	msgParseFailure  = "the uploaded file could not be parsed; please check that it is a valid, non-corrupted Excel file matching the expected column headers"
	msgInsertFailure = "the parsed records could not be saved to the database"
)

// UploadHandler serves the Excel-import endpoints.
type UploadHandler struct {
	employeeSvc *services.EmployeeService
	jobStore    *services.JobStore
	uploadDir   string
}

func NewUploadHandler(employeeSvc *services.EmployeeService, jobStore *services.JobStore, uploadDir string) *UploadHandler {
	return &UploadHandler{
		employeeSvc: employeeSvc,
		jobStore:    jobStore,
		uploadDir:   uploadDir,
	}
}

// UploadExcel handles POST /api/v1/upload: validates and stores the file,
// kicks off async parsing in a goroutine, and returns a job ID to poll.
func (h *UploadHandler) UploadExcel(c *gin.Context) {
	fileHeader, err := c.FormFile("file")

	// fmt.Println("fileHeader", fileHeader)
	if err != nil {
		response.Error(c, apperrors.NewBadRequest("no file provided; upload the excel file using multipart form field 'file'"))
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		response.Error(c, apperrors.NewInvalidFile("only .xlsx or .xls files are accepted"))
		return
	}

	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		slog.Error("upload: failed to prepare upload directory", "error", err)
		response.Error(c, apperrors.NewInternal())
		return
	}

	safeName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
	destPath := filepath.Join(h.uploadDir, safeName)

	if err := c.SaveUploadedFile(fileHeader, destPath); err != nil {
		slog.Error("upload: failed to save uploaded file", "error", err)
		response.Error(c, apperrors.NewInternal())
		return
	}

	job := h.jobStore.CreateJob(fileHeader.Filename)
	requestID := middleware.GetRequestID(c)

	// Own context/timeout: c.Request.Context() is cancelled once this
	// handler returns, but this goroutine must keep running after that.
	go h.processUploadAsync(job.ID, destPath, requestID)

	response.Accepted(c, "file accepted, processing started", gin.H{
		"job_id":     job.ID,
		"status_url": "/api/v1/upload/status/" + job.ID,
	})
}

// processUploadAsync does the actual parse-and-insert work in the
// background, always recording a terminal Completed/Failed status on jobStore.
func (h *UploadHandler) processUploadAsync(jobID, filePath, requestID string) {
	defer os.Remove(filePath) // clean up temp file regardless of outcome

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	h.jobStore.Update(jobID, func(j *models.ImportJob) {
		j.Status = models.JobStatusProcessing
	})

	employees, rowErrors, err := services.ParseExcel(filePath)
	if err != nil {
		// The real cause (bad header, corrupt file, etc.) is logged here;
		// the client only ever sees the fixed msgParseFailure text.
		slog.Error("import job failed during parse", "job_id", jobID, "request_id", requestID, "error", err)
		h.jobStore.Update(jobID, func(j *models.ImportJob) {
			j.Status = models.JobStatusFailed
			j.Errors = append(j.Errors, msgParseFailure)
			now := time.Now()
			j.CompletedAt = &now
		})
		return
	}

	inserted, err := h.employeeSvc.BulkInsert(ctx, employees)
	if err != nil {
		slog.Error("import job failed during bulk insert", "job_id", jobID, "request_id", requestID, "error", err)
		h.jobStore.Update(jobID, func(j *models.ImportJob) {
			j.Status = models.JobStatusFailed
			j.Errors = append(j.Errors, msgInsertFailure)
			now := time.Now()
			j.CompletedAt = &now
		})
		return
	}

	h.jobStore.Update(jobID, func(j *models.ImportJob) {
		j.Status = models.JobStatusCompleted
		j.TotalRows = len(employees) + len(rowErrors)
		j.InsertedRows = inserted
		j.FailedRows = len(rowErrors)
		for _, re := range rowErrors {
			j.Errors = append(j.Errors, re.String())
		}
		now := time.Now()
		j.CompletedAt = &now
	})

	slog.Info("import job completed", "job_id", jobID, "request_id", requestID, "inserted", inserted, "failed", len(rowErrors))
}

// GetUploadStatus handles GET /api/v1/upload/status/:job_id so clients can
// poll for the outcome of an asynchronous import.
func (h *UploadHandler) GetUploadStatus(c *gin.Context) {
	jobID := c.Param("job_id")

	job, ok := h.jobStore.Get(jobID)
	if !ok {
		response.Error(c, apperrors.NewNotFound("import job"))
		return
	}

	response.OK(c, "job status retrieved", job)
}
