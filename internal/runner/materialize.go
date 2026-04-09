// Package runner provides job processing and wallpaper materialization.
package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/services/images"
	"github.com/walens/walens/internal/services/jobs"
	"github.com/walens/walens/internal/sources"
	"github.com/walens/walens/internal/storage"
)

// Storage kind constants.
const (
	StorageKindCanonical = "canonical"
	StorageKindHardlink  = "hardlink"
	StorageKindCopy      = "copy"
)

// MaterializeRequest contains the parameters for materializing images.
type MaterializeRequest struct {
	JobID        dbtypes.UUID
	SourceID     dbtypes.UUID
	SourceType   string
	SourceParams []byte
	LookupCount  int
	Devices      []model.Devices
}

// MaterializeResult contains the outcome of a materialization run.
type MaterializeResult struct {
	DownloadedCount int64
	ReusedCount     int64
	HardlinkedCount int64
	CopiedCount     int64
	SkippedCount    int64
	StoredCount     int64
}

// Materializer handles image materialization from sources to devices.
type Materializer struct {
	logger     *slog.Logger
	storageSvc *storage.Service
	imageSvc   *images.Service
	jobsSvc    *jobs.Service
}

// NewMaterializer creates a new Materializer.
func NewMaterializer(logger *slog.Logger) *Materializer {
	return &Materializer{
		logger: logger,
	}
}

// SetStorageService sets the storage service.
func (m *Materializer) SetStorageService(svc *storage.Service) {
	m.storageSvc = svc
}

// SetImageService sets the image service.
func (m *Materializer) SetImageService(svc *images.Service) {
	m.imageSvc = svc
}

// SetJobsService sets the jobs service.
func (m *Materializer) SetJobsService(svc *jobs.Service) {
	m.jobsSvc = svc
}

// MaterializeImage processes source image metadata and materializes to eligible devices.
func (m *Materializer) MaterializeImage(ctx context.Context, req MaterializeRequest, src sources.Source) (MaterializeResult, error) {
	result := MaterializeResult{}

	// Fetch images from source
	fetchReq := sources.FetchRequest{
		Params:      req.SourceParams,
		LookupCount: req.LookupCount,
	}

	processedCount := 0
	for item, err := range src.Fetch(ctx, fetchReq) {
		if err != nil {
			m.logger.Warn("error fetching image", "error", err)
			continue
		}

		processedCount++

		// Build unique ID for this image
		uniqueID, err := src.BuildUniqueID(item)
		if err != nil {
			m.logger.Warn("cannot build unique ID", "error", err)
			continue
		}

		// Check blacklist - skip if blacklisted
		blacklisted, err := m.isBlacklisted(ctx, req.SourceID, uniqueID)
		if err != nil {
			m.logger.Warn("blacklist check failed", "error", err)
			continue
		}
		if blacklisted {
			m.logger.Debug("image is blacklisted, skipping", "unique_id", uniqueID)
			result.SkippedCount++
			continue
		}

		// Get or create image record
		img, _, err := m.imageSvc.GetOrCreateImage(ctx, images.CreateImageRequest{
			SourceID:             req.SourceID,
			UniqueIdentifier:     uniqueID,
			SourceType:           req.SourceType,
			OriginalFilename:     nil,
			PreviewURL:           ptrString(item.PreviewURL),
			OriginURL:            ptrString(item.OriginURL),
			SourceItemIdentifier: ptrString(item.SourceItemID),
			OriginalIdentifier:   ptrString(item.OriginalID),
			Uploader:             ptrString(item.Uploader),
			Artist:               ptrString(item.Artist),
			MimeType:             ptrString(item.MimeType),
			FileSizeBytes:        ptrInt64(item.FileSizeBytes),
			Width:                ptrInt64(int64(item.Width)),
			Height:               ptrInt64(int64(item.Height)),
			AspectRatio:          ptrFloat64(item.AspectRatio),
			IsAdult:              item.IsAdult,
			IsFavorite:           false,
			JSONMeta:             dbtypes.RawJSON("{}"),
		})
		if err != nil {
			m.logger.Warn("failed to get or create image", "error", err, "unique_id", uniqueID)
			continue
		}

		// Get existing locations for this image
		existingLocations, err := m.imageSvc.GetImageLocations(ctx, *img.ID)
		if err != nil && !errors.Is(err, images.ErrLocationNotFound) {
			m.logger.Warn("failed to get image locations", "error", err)
			continue
		}

		// Build map of device_id -> location for existing locations
		locationByDevice := make(map[string]model.ImageLocations)
		canonicalLocation := ""
		for _, loc := range existingLocations {
			locationByDevice[loc.DeviceID.UUID.String()] = loc
			if loc.StorageKind == StorageKindCanonical && bool(loc.IsActive) {
				canonicalLocation = loc.Path
			}
		}

		// For each subscribed device, apply materialization rules
		for _, device := range req.Devices {
			deviceIDStr := device.ID.UUID.String()

			// Check if already assigned to this device
			_, err := m.imageSvc.GetImageAssignment(ctx, *img.ID, *device.ID)
			hasAssignment := err == nil
			if err != nil && !errors.Is(err, images.ErrAssignmentNotFound) {
				m.logger.Warn("failed to check assignment", "error", err)
				continue
			}

			// Get device-specific location if exists
			deviceLocation, hasLocation := locationByDevice[deviceIDStr]
			hasFile := false
			if hasLocation {
				hasFile = m.storageSvc.FileExists(deviceLocation.Path)
			}

			if hasAssignment && hasFile {
				// Rule 1: If image is assigned to device AND file exists → skip
				result.SkippedCount++
				m.logger.Debug("rule 1: assigned + file exists, skipping",
					"device", device.Slug, "unique_id", uniqueID)
				continue
			}

			if hasAssignment && !hasFile {
				// Rule 2: If image is assigned to device BUT file missing → re-download
				m.logger.Debug("rule 2: assigned but file missing, re-downloading",
					"device", device.Slug, "unique_id", uniqueID)
				path, err := m.downloadToDevice(ctx, item, device, uniqueID)
				if err != nil {
					m.logger.Warn("failed to re-download", "error", err, "device", device.Slug)
					continue
				}
				result.DownloadedCount++

				// Create location record
				_, err = m.imageSvc.CreateImageLocation(ctx, images.CreateImageLocationRequest{
					ImageID:     *img.ID,
					DeviceID:    *device.ID,
					Path:        path,
					StorageKind: StorageKindCanonical,
					IsPrimary:   true,
					IsActive:    true,
				})
				if err != nil {
					m.logger.Warn("failed to create location record", "error", err)
				}
				continue
			}

			// Not assigned to this device - check if exists elsewhere
			if len(existingLocations) > 0 && canonicalLocation != "" && m.storageSvc.FileExists(canonicalLocation) {
				// Rule 3: If not assigned but exists elsewhere → hard link, fallback to copy
				targetPath := m.deviceImagePath(device, uniqueID, item.MimeType)
				err = m.storageSvc.CreateHardLink(canonicalLocation, targetPath)
				if err != nil {
					m.logger.Debug("hard link failed, falling back to copy",
						"error", err, "device", device.Slug)
					// Fallback to copy
					err = m.storageSvc.CopyFile(canonicalLocation, targetPath)
					if err != nil {
						m.logger.Warn("copy failed", "error", err, "device", device.Slug)
						continue
					}
					result.CopiedCount++
					m.logger.Debug("rule 3: copied to device",
						"device", device.Slug, "unique_id", uniqueID)
				} else {
					result.HardlinkedCount++
					m.logger.Debug("rule 3: hard linked to device",
						"device", device.Slug, "unique_id", uniqueID)
				}

				// Create assignment
				_, err = m.imageSvc.CreateImageAssignment(ctx, *img.ID, *device.ID)
				if err != nil && !errors.Is(err, images.ErrAssignmentNotFound) {
					m.logger.Warn("failed to create assignment", "error", err)
				}

				// Create location
				_, err = m.imageSvc.CreateImageLocation(ctx, images.CreateImageLocationRequest{
					ImageID:     *img.ID,
					DeviceID:    *device.ID,
					Path:        targetPath,
					StorageKind: StorageKindHardlink,
					IsPrimary:   true,
					IsActive:    true,
				})
				if err != nil {
					m.logger.Warn("failed to create location record", "error", err)
				}
				continue
			}

			// Rule 4: Not assigned and nowhere else → download
			m.logger.Debug("rule 4: not assigned and no source, downloading",
				"device", device.Slug, "unique_id", uniqueID)
			path, err := m.downloadToDevice(ctx, item, device, uniqueID)
			if err != nil {
				m.logger.Warn("failed to download", "error", err, "device", device.Slug)
				continue
			}
			result.DownloadedCount++

			// Create assignment
			_, err = m.imageSvc.CreateImageAssignment(ctx, *img.ID, *device.ID)
			if err != nil && !errors.Is(err, images.ErrAssignmentNotFound) {
				m.logger.Warn("failed to create assignment", "error", err)
			}

			// Create location
			_, err = m.imageSvc.CreateImageLocation(ctx, images.CreateImageLocationRequest{
				ImageID:     *img.ID,
				DeviceID:    *device.ID,
				Path:        path,
				StorageKind: StorageKindCanonical,
				IsPrimary:   true,
				IsActive:    true,
			})
			if err != nil {
				m.logger.Warn("failed to create location record", "error", err)
			}
			result.StoredCount++
		}

		// Periodically update job counters (every 10 images)
		if processedCount%10 == 0 && m.jobsSvc != nil {
			m.updateJobCounters(ctx, req.JobID, result)
		}
	}

	// Final counter update
	if m.jobsSvc != nil {
		m.updateJobCounters(ctx, req.JobID, result)
	}

	m.logger.Info("materialization complete",
		"processed", processedCount,
		"downloaded", result.DownloadedCount,
		"reused", result.ReusedCount,
		"hardlinked", result.HardlinkedCount,
		"copied", result.CopiedCount,
		"skipped", result.SkippedCount,
		"stored", result.StoredCount)

	return result, nil
}

// isBlacklisted checks if an image is blacklisted for a source.
func (m *Materializer) isBlacklisted(ctx context.Context, sourceID dbtypes.UUID, uniqueID string) (bool, error) {
	if m.imageSvc == nil {
		return false, nil
	}

	var count struct {
		Count int64 `alias:"count"`
	}
	stmt := SELECT(
		COUNT(ImageBlacklists.ID).AS("count"),
	).FROM(
		ImageBlacklists,
	).WHERE(
		ImageBlacklists.SourceID.EQ(String(sourceID.UUID.String())).
			AND(ImageBlacklists.UniqueIdentifier.EQ(String(uniqueID))),
	)

	if err := stmt.QueryContext(ctx, m.imageSvc.DB(), &count); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check blacklist: %w", err)
	}

	return count.Count > 0, nil
}

// downloadToDevice downloads an image to a device's storage.
func (m *Materializer) downloadToDevice(ctx context.Context, item sources.ImageMetadata, device model.Devices, uniqueID string) (string, error) {
	// Use origin URL if available, otherwise preview URL
	downloadURL := item.OriginURL
	if downloadURL == "" {
		downloadURL = item.PreviewURL
	}
	if downloadURL == "" {
		return "", fmt.Errorf("no download URL available")
	}

	// Download to temp
	tempPath, cleanup, err := m.storageSvc.DownloadToTemp(ctx, downloadURL)
	if err != nil {
		return "", fmt.Errorf("download to temp: %w", err)
	}
	defer cleanup()

	// Get file extension from mime type or URL
	ext := m.extFromMimeType(item.MimeType)
	if ext == "" {
		ext = m.extFromURL(downloadURL)
	}
	if ext == "" {
		ext = "jpg"
	}

	// Move to canonical location
	canonicalPath, err := m.storageSvc.MoveToCanonical(tempPath, device.Slug, uniqueID, ext)
	if err != nil {
		return "", fmt.Errorf("move to canonical: %w", err)
	}

	return canonicalPath, nil
}

// deviceImagePath returns the target path for a device's image.
func (m *Materializer) deviceImagePath(device model.Devices, uniqueID, mimeType string) string {
	ext := m.extFromMimeType(mimeType)
	if ext == "" {
		ext = "jpg"
	}
	baseDir := m.storageSvc.BaseDir()
	return filepath.Join(baseDir, "images", device.Slug, fmt.Sprintf("%s.%s", uniqueID, ext))
}

// extFromMimeType extracts file extension from MIME type.
func (m *Materializer) extFromMimeType(mimeType string) string {
	switch strings.ToLower(mimeType) {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/bmp":
		return "bmp"
	case "image/tiff":
		return "tiff"
	default:
		return ""
	}
}

// extFromURL extracts file extension from URL path.
func (m *Materializer) extFromURL(urlStr string) string {
	ext := filepath.Ext(urlStr)
	if ext != "" {
		ext = ext[1:] // Remove leading dot
	}
	return strings.ToLower(ext)
}

// updateJobCounters updates the job counters in the database.
func (m *Materializer) updateJobCounters(ctx context.Context, jobID dbtypes.UUID, result MaterializeResult) {
	delta := jobs.IncrementJobCountersRequest{
		ID: jobID,
		Deltas: jobs.UpdateJobCountersRequest{
			DownloadedImageCount: ptrInt64(result.DownloadedCount),
			ReusedImageCount:     ptrInt64(result.ReusedCount),
			HardlinkedImageCount: ptrInt64(result.HardlinkedCount),
			CopiedImageCount:     ptrInt64(result.CopiedCount),
			StoredImageCount:     ptrInt64(result.StoredCount),
			SkippedImageCount:    ptrInt64(result.SkippedCount),
		},
	}
	_, err := m.jobsSvc.IncrementJobCounters(ctx, delta)
	if err != nil {
		m.logger.Warn("failed to update job counters", "error", err)
	}
}

// Helper functions.
func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrInt64(v int64) *int64 {
	return &v
}

func ptrFloat64(v float64) *float64 {
	return &v
}
