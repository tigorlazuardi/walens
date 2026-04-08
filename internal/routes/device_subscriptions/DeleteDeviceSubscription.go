package device_subscriptions

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/dbtypes"
	subsvc "github.com/walens/walens/internal/services/device_subscriptions"
)

// DeleteDeviceSubscriptionOperation returns the Huma operation metadata for DeleteDeviceSubscription.
func DeleteDeviceSubscriptionOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID:   "post-device_subscriptions-delete-device_subscription",
		Method:        "POST",
		Path:          path.Join(basePath, "/api/v1/device_subscriptions/DeleteDeviceSubscription"),
		DefaultStatus: 204,
		Summary:       "Delete a device subscription by ID",
		Description:   "Deletes a device source subscription by its ID.",
		Tags:          []string{"device_subscriptions"},
	}
}

// DeleteDeviceSubscriptionInput describes the request body for DeleteDeviceSubscription.
type DeleteDeviceSubscriptionInput struct {
	Body struct {
		ID string `json:"id" doc:"Unique device subscription identifier (UUIDv7)."`
	}
}

// DeleteDeviceSubscription handles POST /api/v1/device_subscriptions/DeleteDeviceSubscription.
// Deletes a device source subscription by ID.
func DeleteDeviceSubscription(ctx context.Context, input *DeleteDeviceSubscriptionInput, svc *subsvc.Service) (*struct{}, error) {
	id, err := dbtypes.NewUUIDFromString(input.Body.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid device subscription ID format", err)
	}

	err = svc.DeleteSubscription(ctx, id)
	if err != nil {
		if errors.Is(err, subsvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, subsvc.ErrSubscriptionNotFound) {
			return nil, huma.Error404NotFound("device subscription not found")
		}
		return nil, huma.Error500InternalServerError("failed to delete device subscription", err)
	}

	return nil, nil
}
