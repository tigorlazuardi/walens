package device_subscriptions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

var ErrSubscriptionNotFound = errors.New("device subscription not found")
var ErrDeviceNotFound = errors.New("device not found")
var ErrSourceNotFound = errors.New("source not found")
var ErrDuplicateSubscription = errors.New("device is already subscribed to this source")

type SubscriptionRow = model.DeviceSourceSubscriptions

type CreateSubscriptionRequest struct {
	DeviceID  string `json:"device_id" required:"true" doc:"Reference to the device to subscribe."`
	SourceID  string `json:"source_id" required:"true" doc:"Reference to the source to subscribe to."`
	IsEnabled bool   `json:"is_enabled" required:"true" doc:"Whether this subscription is active."`
}

type CreateSubscriptionResponse = model.DeviceSourceSubscriptions

type UpdateSubscriptionRequest struct {
	ID        string `json:"id" required:"true" doc:"Unique subscription identifier."`
	DeviceID  string `json:"device_id" required:"true" doc:"Reference to the device."`
	SourceID  string `json:"source_id" required:"true" doc:"Reference to the source."`
	IsEnabled bool   `json:"is_enabled" required:"true" doc:"Whether this subscription is active."`
}

type UpdateSubscriptionResponse = model.DeviceSourceSubscriptions

type ListSubscriptionsRequest struct {
	DeviceIDs  []dbtypes.UUID                   `json:"device_ids" doc:"Filter by device IDs (subsets of all matching subscriptions)"`
	SourceIDs  []dbtypes.UUID                   `json:"source_ids" doc:"Filter by source IDs (subsets of all matching subscriptions)"`
	Search     *string                          `json:"search" doc:"Search by device name or source name"`
	Pagination *dbtypes.CursorPaginationRequest `json:"pagination,omitempty"`
}

type ListSubscriptionsResponse struct {
	Items      []model.DeviceSourceSubscriptions `json:"items" doc:"List of device source subscriptions."`
	Pagination *dbtypes.CursorPaginationResponse `json:"pagination"`
	Total      int64                             `json:"total" doc:"Total count of subscriptions matching filters, independent of pagination"`
}

type GetSubscriptionRequest struct {
	ID dbtypes.UUID `json:"id" doc:"Unique subscription identifier."`
}

type GetSubscriptionResponse = model.DeviceSourceSubscriptions

type DeleteSubscriptionRequest struct {
	ID dbtypes.UUID `json:"id" required:"true" doc:"Unique subscription identifier."`
}

type DeleteSubscriptionResponse struct{}

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) countDevices(ctx context.Context, id dbtypes.UUID) (int64, error) {
	var count struct {
		Count int64 `alias:"count"`
	}
	stmt := SELECT(COUNT(Devices.ID).AS("count")).FROM(Devices).WHERE(Devices.ID.EQ(String(id.UUID.String())))
	if err := stmt.QueryContext(ctx, s.db, &count); err != nil {
		return 0, fmt.Errorf("check device exists: %w", err)
	}
	return count.Count, nil
}

func (s *Service) countSources(ctx context.Context, id dbtypes.UUID) (int64, error) {
	var count struct {
		Count int64 `alias:"count"`
	}
	stmt := SELECT(COUNT(Sources.ID).AS("count")).FROM(Sources).WHERE(Sources.ID.EQ(String(id.UUID.String())))
	if err := stmt.QueryContext(ctx, s.db, &count); err != nil {
		return 0, fmt.Errorf("check source exists: %w", err)
	}
	return count.Count, nil
}

func (s *Service) countSubscriptions(ctx context.Context, condition BoolExpression) (int64, error) {
	var count struct {
		Count int64 `alias:"count"`
	}
	stmt := SELECT(COUNT(DeviceSourceSubscriptions.ID).AS("count")).FROM(DeviceSourceSubscriptions).WHERE(condition)
	if err := stmt.QueryContext(ctx, s.db, &count); err != nil {
		return 0, fmt.Errorf("count subscriptions: %w", err)
	}
	return count.Count, nil
}

func (s *Service) ListSubscriptions(ctx context.Context, req ListSubscriptionsRequest) (ListSubscriptionsResponse, error) {
	var items []model.DeviceSourceSubscriptions
	baseCond := Bool(true)

	// Filter by device IDs using IN expression
	if len(req.DeviceIDs) > 0 {
		deviceIDExprs := make([]Expression, len(req.DeviceIDs))
		for i, d := range req.DeviceIDs {
			deviceIDExprs[i] = String(d.UUID.String())
		}
		baseCond = baseCond.AND(DeviceSourceSubscriptions.DeviceID.IN(deviceIDExprs...))
	}

	// Filter by source IDs using IN expression
	if len(req.SourceIDs) > 0 {
		sourceIDExprs := make([]Expression, len(req.SourceIDs))
		for i, src := range req.SourceIDs {
			sourceIDExprs[i] = String(src.UUID.String())
		}
		baseCond = baseCond.AND(DeviceSourceSubscriptions.SourceID.IN(sourceIDExprs...))
	}

	// Filter by search term matching device name or source name
	if req.Search != nil && *req.Search != "" {
		pattern := String("%" + *req.Search + "%")
		searchSubquery := SELECT(DeviceSourceSubscriptions.ID).
			FROM(DeviceSourceSubscriptions.
				INNER_JOIN(Devices, DeviceSourceSubscriptions.DeviceID.EQ(Devices.ID)).
				INNER_JOIN(Sources, DeviceSourceSubscriptions.SourceID.EQ(Sources.ID))).
			WHERE(Devices.Name.LIKE(pattern).OR(Sources.Name.LIKE(pattern)))
		baseCond = baseCond.AND(DeviceSourceSubscriptions.ID.IN(searchSubquery))
	}

	// Get total count before pagination filters
	total, err := s.countSubscriptions(ctx, baseCond)
	if err != nil {
		return ListSubscriptionsResponse{}, err
	}

	// Pagination - use zero-value if omitted so methods don't panic
	pagination := dbtypes.CursorPaginationRequest{}
	if req.Pagination != nil {
		pagination = *req.Pagination
	}

	// Build condition with cursor filters
	cond := baseCond
	next := pagination.NextToken()
	prev := pagination.PrevToken()
	isPrev := next == "" && prev != ""
	if next != "" {
		cond = cond.AND(DeviceSourceSubscriptions.ID.GT(String(next)))
	}
	if isPrev {
		cond = cond.AND(DeviceSourceSubscriptions.ID.LT(String(prev)))
	}

	orderBy, err := pagination.BuildOrderByClause(DeviceSourceSubscriptions.AllColumns)
	if err != nil {
		return ListSubscriptionsResponse{}, err
	}
	if len(orderBy) == 0 {
		orderBy = append(orderBy, DeviceSourceSubscriptions.CreatedAt.ASC())
	}
	if isPrev {
		orderBy = append(orderBy, DeviceSourceSubscriptions.ID.DESC())
	} else {
		orderBy = append(orderBy, DeviceSourceSubscriptions.ID.ASC())
	}

	limit := pagination.GetLimitOrDefault(20, 100)
	stmt := SELECT(DeviceSourceSubscriptions.AllColumns).
		FROM(DeviceSourceSubscriptions).
		WHERE(cond).
		ORDER_BY(orderBy...).
		LIMIT(limit + 1).
		OFFSET(pagination.GetOffset())
	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		return ListSubscriptionsResponse{}, huma.Error500InternalServerError("failed to list device subscriptions", err)
	}
	if len(items) == 0 {
		return ListSubscriptionsResponse{Items: []model.DeviceSourceSubscriptions{}, Total: total}, nil
	}

	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	cursor := &dbtypes.CursorPaginationResponse{}
	if isPrev {
		slices.Reverse(items)
	}
	if hasMore {
		nextID := items[len(items)-1].ID
		cursor.Next = &nextID
	}
	if next != "" {
		prevID := items[0].ID
		cursor.Prev = &prevID
	}
	return ListSubscriptionsResponse{Items: items, Pagination: cursor, Total: total}, nil
}

func (s *Service) GetSubscription(ctx context.Context, req GetSubscriptionRequest) (GetSubscriptionResponse, error) {
	var sub model.DeviceSourceSubscriptions
	stmt := SELECT(DeviceSourceSubscriptions.AllColumns).
		FROM(DeviceSourceSubscriptions).
		WHERE(DeviceSourceSubscriptions.ID.EQ(String(req.ID.UUID.String()))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &sub); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return GetSubscriptionResponse{}, huma.Error404NotFound("device subscription not found", ErrSubscriptionNotFound)
		}
		return GetSubscriptionResponse{}, huma.Error500InternalServerError("failed to get device subscription", err)
	}
	return sub, nil
}

func (s *Service) CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (CreateSubscriptionResponse, error) {
	deviceID, err := dbtypes.NewUUIDFromString(req.DeviceID)
	if err != nil {
		return CreateSubscriptionResponse{}, huma.Error400BadRequest("invalid device ID format", err)
	}
	sourceID, err := dbtypes.NewUUIDFromString(req.SourceID)
	if err != nil {
		return CreateSubscriptionResponse{}, huma.Error400BadRequest("invalid source ID format", err)
	}
	deviceCount, err := s.countDevices(ctx, deviceID)
	if err != nil {
		return CreateSubscriptionResponse{}, huma.Error500InternalServerError("failed to validate device", err)
	}
	if deviceCount == 0 {
		return CreateSubscriptionResponse{}, huma.Error400BadRequest("device not found", ErrDeviceNotFound)
	}
	sourceCount, err := s.countSources(ctx, sourceID)
	if err != nil {
		return CreateSubscriptionResponse{}, huma.Error500InternalServerError("failed to validate source", err)
	}
	if sourceCount == 0 {
		return CreateSubscriptionResponse{}, huma.Error400BadRequest("source not found", ErrSourceNotFound)
	}
	duplicateCount, err := s.countSubscriptions(ctx,
		DeviceSourceSubscriptions.DeviceID.EQ(String(deviceID.UUID.String())).
			AND(DeviceSourceSubscriptions.SourceID.EQ(String(sourceID.UUID.String()))),
	)
	if err != nil {
		return CreateSubscriptionResponse{}, huma.Error500InternalServerError("failed to check duplicate device subscription", err)
	}
	if duplicateCount > 0 {
		return CreateSubscriptionResponse{}, huma.Error409Conflict("device is already subscribed to this source", ErrDuplicateSubscription)
	}
	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return CreateSubscriptionResponse{}, huma.Error500InternalServerError("failed to generate device subscription id", err)
	}
	row := model.DeviceSourceSubscriptions{
		ID:        id,
		DeviceID:  deviceID,
		SourceID:  sourceID,
		IsEnabled: dbtypes.BoolInt(req.IsEnabled),
		CreatedAt: now,
		UpdatedAt: now,
	}
	stmt := DeviceSourceSubscriptions.INSERT(
		DeviceSourceSubscriptions.ID,
		DeviceSourceSubscriptions.DeviceID,
		DeviceSourceSubscriptions.SourceID,
		DeviceSourceSubscriptions.IsEnabled,
		DeviceSourceSubscriptions.CreatedAt,
		DeviceSourceSubscriptions.UpdatedAt,
	).MODEL(row)
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return CreateSubscriptionResponse{}, huma.Error500InternalServerError("failed to create device subscription", err)
	}
	return s.GetSubscription(ctx, GetSubscriptionRequest{ID: id})
}

func (s *Service) UpdateSubscription(ctx context.Context, req UpdateSubscriptionRequest) (UpdateSubscriptionResponse, error) {
	id, err := dbtypes.NewUUIDFromString(req.ID)
	if err != nil {
		return UpdateSubscriptionResponse{}, huma.Error400BadRequest("invalid subscription ID format", err)
	}
	deviceID, err := dbtypes.NewUUIDFromString(req.DeviceID)
	if err != nil {
		return UpdateSubscriptionResponse{}, huma.Error400BadRequest("invalid device ID format", err)
	}
	sourceID, err := dbtypes.NewUUIDFromString(req.SourceID)
	if err != nil {
		return UpdateSubscriptionResponse{}, huma.Error400BadRequest("invalid source ID format", err)
	}
	existing, err := s.GetSubscription(ctx, GetSubscriptionRequest{ID: id})
	if err != nil {
		return UpdateSubscriptionResponse{}, err
	}
	deviceCount, err := s.countDevices(ctx, deviceID)
	if err != nil {
		return UpdateSubscriptionResponse{}, huma.Error500InternalServerError("failed to validate device", err)
	}
	if deviceCount == 0 {
		return UpdateSubscriptionResponse{}, huma.Error400BadRequest("device not found", ErrDeviceNotFound)
	}
	sourceCount, err := s.countSources(ctx, sourceID)
	if err != nil {
		return UpdateSubscriptionResponse{}, huma.Error500InternalServerError("failed to validate source", err)
	}
	if sourceCount == 0 {
		return UpdateSubscriptionResponse{}, huma.Error400BadRequest("source not found", ErrSourceNotFound)
	}
	duplicateCount, err := s.countSubscriptions(ctx,
		DeviceSourceSubscriptions.DeviceID.EQ(String(deviceID.UUID.String())).
			AND(DeviceSourceSubscriptions.SourceID.EQ(String(sourceID.UUID.String()))).
			AND(DeviceSourceSubscriptions.ID.NOT_EQ(String(id.UUID.String()))),
	)
	if err != nil {
		return UpdateSubscriptionResponse{}, huma.Error500InternalServerError("failed to check duplicate device subscription", err)
	}
	if duplicateCount > 0 {
		return UpdateSubscriptionResponse{}, huma.Error409Conflict("device is already subscribed to this source", ErrDuplicateSubscription)
	}
	updated := existing
	updated.DeviceID = deviceID
	updated.SourceID = sourceID
	updated.IsEnabled = dbtypes.BoolInt(req.IsEnabled)
	updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()
	stmt := DeviceSourceSubscriptions.UPDATE(
		DeviceSourceSubscriptions.DeviceID,
		DeviceSourceSubscriptions.SourceID,
		DeviceSourceSubscriptions.IsEnabled,
		DeviceSourceSubscriptions.UpdatedAt,
	).MODEL(updated).WHERE(DeviceSourceSubscriptions.ID.EQ(String(id.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return UpdateSubscriptionResponse{}, huma.Error500InternalServerError("failed to update device subscription", err)
	}
	return s.GetSubscription(ctx, GetSubscriptionRequest{ID: id})
}

func (s *Service) DeleteSubscription(ctx context.Context, req DeleteSubscriptionRequest) (DeleteSubscriptionResponse, error) {
	if _, err := s.GetSubscription(ctx, GetSubscriptionRequest{ID: req.ID}); err != nil {
		return DeleteSubscriptionResponse{}, err
	}
	stmt := DeviceSourceSubscriptions.DELETE().WHERE(DeviceSourceSubscriptions.ID.EQ(String(req.ID.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return DeleteSubscriptionResponse{}, huma.Error500InternalServerError("failed to delete device subscription", err)
	}
	return DeleteSubscriptionResponse{}, nil
}
