package device_subscriptions

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	subsvc "github.com/walens/walens/internal/services/device_subscriptions"
)

// CreateDeviceSubscriptionOperation returns the Huma operation metadata for CreateDeviceSubscription.
func CreateDeviceSubscriptionOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-device_subscriptions-create-device_subscription",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/device_subscriptions/CreateDeviceSubscription"),
		Summary:     "Create a new device subscription",
		Description: "Creates a new device source subscription, linking an enabled device to an enabled source row.",
		Tags:        []string{"device_subscriptions"},
	}
}

// CreateDeviceSubscriptionInput describes the request body for CreateDeviceSubscription.
type CreateDeviceSubscriptionInput struct {
	Body subsvc.CreateSubscriptionInput
}

// CreateDeviceSubscriptionOutput describes the response body for CreateDeviceSubscription.
type CreateDeviceSubscriptionOutput struct {
	Body subsvc.SubscriptionRow
}

// CreateDeviceSubscription handles POST /api/v1/device_subscriptions/CreateDeviceSubscription.
// Creates a new device source subscription.
func CreateDeviceSubscription(ctx context.Context, input *CreateDeviceSubscriptionInput, svc *subsvc.Service) (*CreateDeviceSubscriptionOutput, error) {
	sub, err := svc.CreateSubscription(ctx, &input.Body)
	if err != nil {
		if errors.Is(err, subsvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
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
		return nil, huma.Error500InternalServerError("failed to create device subscription", err)
	}

	return &CreateDeviceSubscriptionOutput{
		Body: *sub,
	}, nil
}
