package devices

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/walens/walens/internal/db/generated/model"

	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
)

type GetDeviceBySlugRequest struct {
	Slug string `json:"slug" pattern:"^[a-z0-9]+(?:[_-][a-z0-9]+)*$" patternDescription:"url safe text value" minLength:"1" doc:"url safe uniquely identity of a device"`
}

type GetDeviceBySlugResponse = model.Devices

func (s *Service) GetDeviceBySlug(ctx context.Context, req GetDeviceBySlugRequest) (GetDeviceBySlugResponse, error) {
	var dev model.Devices
	stmt := SELECT(Devices.AllColumns).
		FROM(Devices).
		WHERE(Devices.Slug.EQ(String(req.Slug))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &dev); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return GetDeviceBySlugResponse{}, huma.Error404NotFound("device not found", err)
		}
		return GetDeviceBySlugResponse{}, huma.Error500InternalServerError("failed to get device by slug", err)
	}
	return dev, nil
}
