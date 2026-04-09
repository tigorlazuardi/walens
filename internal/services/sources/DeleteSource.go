package sources

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type DeleteSourceRequest struct {
	ID dbtypes.UUID `json:"id" doc:"Unique source identifier."`
}

type DeleteSourceResponse struct{}

// DeleteSource deletes a source by ID.
func (s *Service) DeleteSource(ctx context.Context, req DeleteSourceRequest) (DeleteSourceResponse, error) {
	if _, err := s.GetSource(ctx, GetSourceRequest{ID: req.ID}); err != nil {
		return DeleteSourceResponse{}, err
	}

	stmt := Sources.DELETE().WHERE(Sources.ID.EQ(String(req.ID.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return DeleteSourceResponse{}, huma.Error500InternalServerError("failed to delete source", err)
	}

	return DeleteSourceResponse{}, nil
}
