// Package httpapi exposes the stream catalog over REST and live video over
// WebSockets.
package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/rohit/rtsp-stream-viewer/backend/internal/hub"
	"github.com/rohit/rtsp-stream-viewer/backend/internal/stream"
)

type Options struct {
	Registry       stream.Registry
	Hub            *hub.Manager
	AllowedOrigins []string
	Log            *slog.Logger
}

type api struct {
	registry    stream.Registry
	hub         *hub.Manager
	log         *slog.Logger
	originHosts []string
}

func NewHandler(o Options) http.Handler {
	a := &api{
		registry:    o.Registry,
		hub:         o.Hub,
		log:         o.Log,
		originHosts: hostPatterns(o.AllowedOrigins),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/streams", a.listStreams)
	mux.HandleFunc("POST /api/streams", a.createStream)
	mux.HandleFunc("GET /api/streams/{id}", a.getStream)
	mux.HandleFunc("DELETE /api/streams/{id}", a.deleteStream)
	mux.HandleFunc("GET /ws/streams/{id}", a.streamSocket)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, errNotFound)
	})

	return recoverer(o.Log, requestLog(o.Log, cors(o.AllowedOrigins, mux)))
}

// hostPatterns converts origins to the host-only patterns the WebSocket
// library checks the Origin header against.
func hostPatterns(origins []string) []string {
	hosts := make([]string, 0, len(origins))
	for _, o := range origins {
		_, host, found := strings.Cut(o, "://")
		if !found {
			host = o
		}
		hosts = append(hosts, host)
	}
	return hosts
}
