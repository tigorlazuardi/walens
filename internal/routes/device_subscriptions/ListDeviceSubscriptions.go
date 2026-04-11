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
		OperationID: "ListDeviceSubscriptions",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/device_subscriptions/ListDeviceSubscriptions"),
		Summary:     "List device subscriptions",
		Description: "Returns device source subscription rows, optionally filtered by device IDs, source IDs, and search term (matches device name or source name). Ordered by creation time.",
		Tags:        []string{"Device Subscriptions"},
	}
}

// ListDeviceSubscriptionsInput describes the request body for ListDeviceSubscriptions.
type ListDeviceSubscriptionsInput struct {
	Body subsvc.ListSubscriptionsRequest
}

// ListDeviceSubscriptionsOutput describes the response body for ListDeviceSubscriptions.
type ListDeviceSubscriptionsOutput struct {
	Body subsvc.ListSubscriptionsResponse
}

// ListDeviceSubscriptions handles POST /api/v1/device_subscriptions/ListDeviceSubscriptions.
// Returns device source subscription rows, optionally filtered by device IDs, source IDs, and search term.
func ListDeviceSubscriptions(ctx context.Context, input *ListDeviceSubscriptionsInput, svc *subsvc.Service) (*ListDeviceSubscriptionsOutput, error) {
	resp, err := svc.ListSubscriptions(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &ListDeviceSubscriptionsOutput{
		Body: resp,
	}, nil
}
