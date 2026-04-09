package device_subscriptions

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	subsvc "github.com/walens/walens/internal/services/device_subscriptions"
)

// ListDeviceSubscriptionsOperation returns the Huma operation metadata for ListDeviceSubscriptions.
func ListDeviceSubscriptionsOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-device_subscriptions-list-device_subscriptions",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/device_subscriptions/ListDeviceSubscriptions"),
		Summary:     "List all device subscriptions",
		Description: "Returns all device source subscription rows, ordered by creation time.",
		Tags:        []string{"device_subscriptions"},
	}
}

// ListDeviceSubscriptionsInput describes the request body for ListDeviceSubscriptions.
type ListDeviceSubscriptionsInput struct {
	Body subsvc.ListSubscriptionsRequest
}

// ListDeviceSubscriptionsOutput describes the response body for ListDeviceSubscriptions.
type ListDeviceSubscriptionsOutput struct {
	Body struct {
		Items []subsvc.SubscriptionRow `json:"items" doc:"List of device source subscriptions."`
	}
}

// ListDeviceSubscriptions handles POST /api/v1/device_subscriptions/ListDeviceSubscriptions.
// Returns all device source subscription rows.
func ListDeviceSubscriptions(ctx context.Context, input *ListDeviceSubscriptionsInput, svc *subsvc.Service) (*ListDeviceSubscriptionsOutput, error) {
	resp, err := svc.ListSubscriptions(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &ListDeviceSubscriptionsOutput{
		Body: struct {
			Items []subsvc.SubscriptionRow `json:"items" doc:"List of device source subscriptions."`
		}{
			Items: resp.Items,
		},
	}, nil
}
