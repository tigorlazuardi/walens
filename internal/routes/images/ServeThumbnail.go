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

// ServeThumbnailInput describes the request body for ServeThumbnail.
type ServeThumbnailInput struct {
	Body struct {
		ImageID dbtypes.UUID `json:"image_id" doc:"Image ID"`
	}
}

// ServeThumbnailOutput describes the response body for ServeThumbnail.
type ServeThumbnailOutput struct {
	Body struct {
		Path string `json:"path" doc:"Filesystem path to the thumbnail"`
		URL  string `json:"url" doc:"URL to serve the thumbnail through the app"`
	}
}

// ServeThumbnailOperation returns the Huma operation metadata for ServeThumbnail.
func ServeThumbnailOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "ServeThumbnail",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/images/ServeThumbnail"),
		Summary:     "Get thumbnail serving URL",
		Description: "Returns a URL path that can be used to retrieve the thumbnail image.",
		Tags:        []string{"Images"},
	}
}

// ServeThumbnail handles POST /api/v1/images/ServeThumbnail.
func ServeThumbnail(ctx context.Context, input *ServeThumbnailInput, svc *imagesvc.Service, basePath string) (*ServeThumbnailOutput, error) {
	thumb, err := svc.GetImageThumbnail(ctx, input.Body.ImageID)
	if err != nil {
		if err == imagesvc.ErrThumbnailNotFound {
			return nil, huma.Error404NotFound("thumbnail not found")
		}
		return nil, err
	}

	output := &ServeThumbnailOutput{}
	output.Body.Path = thumb.Path
	output.Body.URL = fmt.Sprintf("%s/api/v1/images/thumbnail/%s", basePath, thumb.ImageID.UUID.String())
	return output, nil
}

// GetServeThumbnailHandler returns an HTTP handler for serving thumbnails directly.
type ServeThumbnailHandler struct {
	BasePath string
	ImageSvc *imagesvc.Service
}

func (h *ServeThumbnailHandler) Pattern() string {
	return path.Join(h.BasePath, "/api/v1/images/thumbnail/{imageID}")
}

func (h *ServeThumbnailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	imageIDStr := path.Base(r.URL.Path)
	imageID, err := dbtypes.NewUUIDFromString(imageIDStr)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	thumb, err := h.ImageSvc.GetImageThumbnail(r.Context(), imageID)
	if err != nil {
		if err == imagesvc.ErrThumbnailNotFound {
			http.Error(w, "thumbnail not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get thumbnail", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, thumb.Path)
}
