package images

import (
	"context"
	"fmt"

	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// CreateImageRequest defines the input for creating a new image record.
type CreateImageRequest struct {
	SourceID             dbtypes.UUID    `json:"source_id" doc:"Reference to source this image came from"`
	UniqueIdentifier     string          `json:"unique_identifier" doc:"Source-provided unique identifier for dedupe"`
	SourceType           string          `json:"source_type" doc:"Source implementation type name"`
	OriginalFilename     *string         `json:"original_filename" doc:"Original filename from source"`
	PreviewURL           *string         `json:"preview_url" doc:"Low-resolution preview URL"`
	OriginURL            *string         `json:"origin_url" doc:"Original source URL for the image"`
	SourceItemIdentifier *string         `json:"source_item_identifier" doc:"External source item/post identifier"`
	OriginalIdentifier   *string         `json:"original_identifier" doc:"External original/source identifier"`
	Uploader             *string         `json:"uploader" doc:"Uploader/artist name from source"`
	Artist               *string         `json:"artist" doc:"Artist/creator name from source"`
	MimeType             *string         `json:"mime_type" doc:"Image MIME type"`
	FileSizeBytes        *int64          `json:"file_size_bytes" doc:"File size in bytes"`
	Width                *int64          `json:"width" doc:"Image width in pixels"`
	Height               *int64          `json:"height" doc:"Image height in pixels"`
	AspectRatio          *float64        `json:"aspect_ratio" doc:"Image aspect ratio (width/height)"`
	IsAdult              bool            `json:"is_adult" doc:"Whether the image contains adult content"`
	IsFavorite           bool            `json:"is_favorite" doc:"Whether the user marked this as favorite"`
	JSONMeta             dbtypes.RawJSON `json:"json_meta" doc:"Additional image metadata as JSON"`
}

// CreateImage inserts a new image record and returns the created image.
func (s *Service) CreateImage(ctx context.Context, req CreateImageRequest) (*model.Images, error) {
	now := dbtypes.NewUnixMilliTimeNow()
	id := dbtypes.MustNewUUIDV7()

	row := model.Images{
		ID:                   &id,
		SourceID:             &req.SourceID,
		UniqueIdentifier:     req.UniqueIdentifier,
		SourceType:           req.SourceType,
		OriginalFilename:     req.OriginalFilename,
		PreviewURL:           req.PreviewURL,
		OriginURL:            req.OriginURL,
		SourceItemIdentifier: req.SourceItemIdentifier,
		OriginalIdentifier:   req.OriginalIdentifier,
		Uploader:             req.Uploader,
		Artist:               req.Artist,
		MimeType:             req.MimeType,
		FileSizeBytes:        req.FileSizeBytes,
		Width:                req.Width,
		Height:               req.Height,
		AspectRatio:          req.AspectRatio,
		IsAdult:              dbtypes.BoolInt(req.IsAdult),
		IsFavorite:           dbtypes.BoolInt(req.IsFavorite),
		JSONMeta:             req.JSONMeta,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	stmt := Images.INSERT(
		Images.ID, Images.SourceID, Images.UniqueIdentifier, Images.SourceType,
		Images.OriginalFilename, Images.PreviewURL, Images.OriginURL,
		Images.SourceItemIdentifier, Images.OriginalIdentifier, Images.Uploader,
		Images.Artist, Images.MimeType, Images.FileSizeBytes, Images.Width,
		Images.Height, Images.AspectRatio, Images.IsAdult, Images.IsFavorite,
		Images.JSONMeta, Images.CreatedAt, Images.UpdatedAt,
	).MODEL(row)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("create image: %w", err)
	}

	return &row, nil
}
