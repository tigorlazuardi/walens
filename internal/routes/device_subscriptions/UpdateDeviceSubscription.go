package device_subscriptions

import (
	"context"
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
	Body subsvc.UpdateSubscriptionRequest
}

// UpdateDeviceSubscriptionOutput describes the response body for UpdateDeviceSubscription.
type UpdateDeviceSubscriptionOutput struct {
	Body subsvc.SubscriptionRow
}

// UpdateDeviceSubscription handles POST /api/v1/device_subscriptions/UpdateDeviceSubscription.
// Updates an existing device source subscription.
func UpdateDeviceSubscription(ctx context.Context, input *UpdateDeviceSubscriptionInput, svc *subsvc.Service) (*UpdateDeviceSubscriptionOutput, error) {
	sub, err := svc.UpdateSubscription(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &UpdateDeviceSubscriptionOutput{
		Body: sub,
	}, nil
}
