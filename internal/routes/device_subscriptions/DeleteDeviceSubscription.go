package device_subscriptions

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	subsvc "github.com/walens/walens/internal/services/device_subscriptions"
)

// DeleteDeviceSubscriptionOperation returns the Huma operation metadata for DeleteDeviceSubscription.
func DeleteDeviceSubscriptionOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID:   "DeleteDeviceSubscription",
		Method:        "POST",
		Path:          path.Join(basePath, "/api/v1/device_subscriptions/DeleteDeviceSubscription"),
		DefaultStatus: 204,
		Summary:       "Delete a device subscription by ID",
		Description:   "Deletes a device source subscription by its ID.",
		Tags:          []string{"Device Subscriptions"},
	}
}

// DeleteDeviceSubscriptionInput describes the request body for DeleteDeviceSubscription.
type DeleteDeviceSubscriptionInput struct {
	Body subsvc.DeleteSubscriptionRequest
}

// DeleteDeviceSubscription handles POST /api/v1/device_subscriptions/DeleteDeviceSubscription.
// Deletes a device source subscription by ID.
func DeleteDeviceSubscription(ctx context.Context, input *DeleteDeviceSubscriptionInput, svc *subsvc.Service) (*struct{}, error) {
	_, err := svc.DeleteSubscription(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
