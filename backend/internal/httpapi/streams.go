package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/rohit/rtsp-stream-viewer/backend/internal/hub"
	"github.com/rohit/rtsp-stream-viewer/backend/internal/stream"
)

type streamResponse struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	URL       string       `json:"url"`
	CreatedAt time.Time    `json:"createdAt"`
	Status    streamStatus `json:"status"`
}

type streamStatus struct {
	State   hub.State `json:"state"`
	Viewers int       `json:"viewers"`
}

func (a *api) toResponse(s stream.Stream) streamResponse {
	status, viewers := a.hub.Status(s.ID)
	return streamResponse{
		ID:        s.ID,
		Name:      s.Name,
		URL:       stream.RedactURL(s.URL),
		CreatedAt: s.CreatedAt,
		Status:    streamStatus{State: status.State, Viewers: viewers},
	}
}

func (a *api) listStreams(w http.ResponseWriter, r *http.Request) {
	streams, err := a.registry.List(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}

	out := make([]streamResponse, len(streams))
	for i, s := range streams {
		out[i] = a.toResponse(s)
	}
	writeJSON(w, http.StatusOK, map[string]any{"streams": out})
}

func (a *api) createStream(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	}
	if apiErr := decodeJSON(w, r, &req); apiErr != nil {
		writeError(w, *apiErr)
		return
	}

	s, err := stream.New(req.URL, req.Name)
	if err == nil {
		err = a.registry.Create(r.Context(), s)
	}

	switch {
	case err == nil:
		w.Header().Set("Location", "/api/streams/"+s.ID)
		writeJSON(w, http.StatusCreated, a.toResponse(s))
	case errors.Is(err, stream.ErrInvalidURL):
		writeError(w, apiError{http.StatusBadRequest, "invalid_url", userMessage(err, stream.ErrInvalidURL)})
	case errors.Is(err, stream.ErrInvalidName):
		writeError(w, apiError{http.StatusBadRequest, "invalid_name", userMessage(err, stream.ErrInvalidName)})
	case errors.Is(err, stream.ErrDuplicate):
		writeError(w, apiError{http.StatusConflict, "duplicate_stream", "This stream has already been added"})
	case errors.Is(err, stream.ErrLimitReached):
		writeError(w, apiError{http.StatusConflict, "stream_limit", "The stream limit has been reached"})
	default:
		a.internalError(w, r, err)
	}
}

func (a *api) getStream(w http.ResponseWriter, r *http.Request) {
	s, err := a.registry.Get(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, stream.ErrNotFound):
		writeError(w, errNotFound)
	case err != nil:
		a.internalError(w, r, err)
	default:
		writeJSON(w, http.StatusOK, a.toResponse(s))
	}
}

func (a *api) deleteStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.registry.Delete(r.Context(), id)
	switch {
	case errors.Is(err, stream.ErrNotFound):
		writeError(w, errNotFound)
	case err != nil:
		a.internalError(w, r, err)
	default:
		a.hub.Remove(id)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (a *api) internalError(w http.ResponseWriter, r *http.Request, err error) {
	a.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, apiError{http.StatusInternalServerError, "internal", "Internal server error"})
}

// userMessage turns "invalid stream url: host is missing" into "Host is
// missing" for display next to the form field.
func userMessage(err, sentinel error) string {
	msg := strings.TrimPrefix(err.Error(), sentinel.Error()+": ")
	if msg == "" {
		return sentinel.Error()
	}
	return strings.ToUpper(msg[:1]) + msg[1:]
}
