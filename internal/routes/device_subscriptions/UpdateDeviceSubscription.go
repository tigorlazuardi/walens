package device_subscriptions

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	subsvc "github.com/walens/walens/internal/services/device_subscriptions"
)

// UpdateDeviceSubscriptionOperation returns the Huma operation metadata for UpdateDeviceSubscription.
func UpdateDeviceSubscriptionOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-device_subscriptions-update-device_subscription",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/device_subscriptions/UpdateDeviceSubscription"),
		Summary:     "Update an existing device subscription",
		Description: "Updates an existing device source subscription with full-object update semantics.",
		Tags:        []string{"device_subscriptions"},
	}
}

// UpdateDeviceSubscriptionInput describes the request body for UpdateDeviceSubscription.
type UpdateDeviceSubscriptionInput struct {
	Body subsvc.UpdateSubscriptionInput
}

// UpdateDeviceSubscriptionOutput describes the response body for UpdateDeviceSubscription.
type UpdateDeviceSubscriptionOutput struct {
	Body subsvc.SubscriptionRow
}

// UpdateDeviceSubscription handles POST /api/v1/device_subscriptions/UpdateDeviceSubscription.
// Updates an existing device source subscription.
func UpdateDeviceSubscription(ctx context.Context, input *UpdateDeviceSubscriptionInput, svc *subsvc.Service) (*UpdateDeviceSubscriptionOutput, error) {
	sub, err := svc.UpdateSubscription(ctx, &input.Body)
	if err != nil {
		if errors.Is(err, subsvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, subsvc.ErrSubscriptionNotFound) {
			return nil, huma.Error404NotFound("device subscription not found")
		}
		if errors.Is(err, subsvc.ErrDeviceNotFound) {
			return nil, huma.Error400BadRequest("device not found")
		}
		if errors.Is(err, subsvc.ErrSourceNotFound) {
			return nil, huma.Error400BadRequest("source not found")
		}
		if errors.Is(err, subsvc.ErrDuplicateSubscription) {
			return nil, huma.Error409Conflict("device is already subscribed to this source")
		}
		return nil, huma.Error500InternalServerError("failed to update device subscription", err)
	}

	return &UpdateDeviceSubscriptionOutput{
		Body: *sub,
	}, nil
}
