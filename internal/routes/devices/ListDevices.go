package devices

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	devicesvc "github.com/walens/walens/internal/services/devices"
)

// ListDevicesOperation returns the Huma operation metadata for ListDevices.
func ListDevicesOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "ListDevices",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/devices/ListDevices"),
		Summary:     "List all devices",
		Description: "Returns all device rows, ordered by name.",
		Tags:        []string{"Devices"},
	}
}

// ListDevicesInput describes the request body for ListDevices.
type ListDevicesInput struct {
	Body devicesvc.ListDevicesRequest
}

// ListDevicesOutput describes the response body for ListDevices.
type ListDevicesOutput struct {
	Body devicesvc.ListDevicesResponse
}

// ListDevices handles POST /api/v1/devices/ListDevices.
// Returns all device rows.
func ListDevices(ctx context.Context, input *ListDevicesInput, svc *devicesvc.Service) (*ListDevicesOutput, error) {
	resp, err := svc.ListDevices(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &ListDevicesOutput{
		Body: resp,
	}, nil
}
