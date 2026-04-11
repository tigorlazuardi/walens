package images

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/dbtypes"
	imagesvc "github.com/walens/walens/internal/services/images"
)

// ServeImageInput describes the request body for ServeImage.
type ServeImageInput struct {
	Body struct {
		ImageID  dbtypes.UUID  `json:"image_id" required:"true" doc:"Image ID"`
		DeviceID *dbtypes.UUID `json:"device_id,omitempty" doc:"Optional device ID to get device-specific location"`
	}
}

// ServeImageOutput describes the response body for ServeImage.
type ServeImageOutput struct {
	Body struct {
		Path string `json:"path" doc:"Filesystem path to the image"`
		URL  string `json:"url" doc:"URL to serve the image through the app"`
	}
}

// ServeImageOperation returns the Huma operation metadata for ServeImage.
func ServeImageOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "ServeImage",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/images/ServeImage"),
		Summary:     "Get image serving URL",
		Description: "Returns a URL path that can be used to retrieve the original image.",
		Tags:        []string{"Images"},
	}
}

// ServeImage handles POST /api/v1/images/ServeImage.
func ServeImage(ctx context.Context, input *ServeImageInput, svc *imagesvc.Service, basePath string) (*ServeImageOutput, error) {
	var loc *imagesvc.ImageLocationResult
	var err error

	if input.Body.DeviceID != nil {
		imgLoc, err := svc.GetDeviceImageLocation(ctx, input.Body.ImageID, *input.Body.DeviceID)
		if err != nil {
			if err == imagesvc.ErrLocationNotFound {
				return nil, huma.Error404NotFound("image location not found")
			}
			return nil, err
		}
		loc = &imagesvc.ImageLocationResult{
			ImageID: imgLoc.ImageID,
			Path:    imgLoc.Path,
		}
	} else {
		loc, err = svc.GetPrimaryImageLocation(ctx, input.Body.ImageID)
		if err != nil {
			if err == imagesvc.ErrLocationNotFound {
				return nil, huma.Error404NotFound("image location not found")
			}
			return nil, err
		}
	}

	output := &ServeImageOutput{}
	output.Body.Path = loc.Path
	output.Body.URL = fmt.Sprintf("%s/api/v1/images/image/%s", basePath, loc.ImageID.UUID.String())
	return output, nil
}

// GetServeImageHandler returns an HTTP handler for serving images directly.
type ServeImageHandler struct {
	BasePath string
	ImageSvc *imagesvc.Service
}

func (h *ServeImageHandler) Pattern() string {
	return path.Join(h.BasePath, "/api/v1/images/image/{imageID}")
}

func (h *ServeImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	imageIDStr := path.Base(r.URL.Path)
	imageID, err := dbtypes.NewUUIDFromString(imageIDStr)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	loc, err := h.ImageSvc.GetPrimaryImageLocation(r.Context(), imageID)
	if err != nil {
		if err == imagesvc.ErrLocationNotFound {
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get image", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, loc.Path)
}
