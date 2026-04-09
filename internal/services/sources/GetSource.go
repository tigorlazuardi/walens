package sources

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type GetSourceRequest struct {
	ID dbtypes.UUID `json:"id" doc:"Unique source identifier."`
}

type GetSourceResponse = model.Sources

// GetSource returns a single source by ID.
func (s *Service) GetSource(ctx context.Context, req GetSourceRequest) (GetSourceResponse, error) {
	var src model.Sources
	stmt := SELECT(Sources.AllColumns).
		FROM(Sources).
		WHERE(Sources.ID.EQ(String(req.ID.UUID.String()))).
		LIMIT(1)

	if err := stmt.QueryContext(ctx, s.db, &src); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return GetSourceResponse{}, huma.Error404NotFound("source not found", ErrSourceNotFound)
		}
		return GetSourceResponse{}, huma.Error500InternalServerError("failed to get source", err)
	}

	return src, nil
}
