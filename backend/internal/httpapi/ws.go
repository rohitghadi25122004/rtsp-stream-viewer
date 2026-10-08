package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/rohit/rtsp-stream-viewer/backend/internal/hub"
	"github.com/rohit/rtsp-stream-viewer/backend/internal/stream"
)

// Application close codes. Browsers cannot read the HTTP status of a failed
// upgrade, so these failures are reported after accepting the connection.
const (
	closeStreamRemoved  websocket.StatusCode = 4000
	closeStreamNotFound websocket.StatusCode = 4004
	closeTooSlow        websocket.StatusCode = 4008
)

const (
	writeTimeout = 10 * time.Second
	// pingInterval keeps proxies such as Render's from dropping a socket
	// that carries no video while the camera is reconnecting.
	pingInterval = 25 * time.Second
)

type statusFrame struct {
	Type string `json:"type"`
	hub.Status
}

type initFrame struct {
	Type  string `json:"type"`
	Codec string `json:"codec"`
}

func (a *api) streamSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns:  a.originHosts,
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	st, err := a.registry.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, stream.ErrNotFound) {
			conn.Close(closeStreamNotFound, "stream not found")
		} else {
			a.log.Error("load stream", "err", err)
			conn.Close(websocket.StatusInternalError, "internal error")
		}
		return
	}

	sub, err := a.hub.Subscribe(st.ID, st.URL)
	if err != nil {
		code, reason := closeCode(err)
		conn.Close(code, reason)
		return
	}
	defer sub.Close()

	ctx, cancel := context.WithCancel(conn.CloseRead(r.Context()))
	defer cancel()
	go keepAlive(ctx, conn)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sub.Done():
			code, reason := closeCode(sub.Err())
			conn.Close(code, reason)
			return
		case msg := <-sub.Messages():
			if err := writeMessage(ctx, conn, msg); err != nil {
				return
			}
		}
	}
}

func writeMessage(ctx context.Context, conn *websocket.Conn, msg hub.Message) error {
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	switch msg.Kind {
	case hub.MsgStatus:
		return writeText(ctx, conn, statusFrame{Type: "status", Status: msg.Status})
	case hub.MsgInit:
		if err := writeText(ctx, conn, initFrame{Type: "init", Codec: msg.Codec}); err != nil {
			return err
		}
		return conn.Write(ctx, websocket.MessageBinary, msg.Data)
	default:
		return conn.Write(ctx, websocket.MessageBinary, msg.Data)
	}
}

func writeText(ctx context.Context, conn *websocket.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, b)
}

func keepAlive(ctx context.Context, conn *websocket.Conn) {
	t := time.NewTicker(pingInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				conn.CloseNow()
				return
			}
		}
	}
}

func closeCode(err error) (websocket.StatusCode, string) {
	switch {
	case errors.Is(err, hub.ErrStreamRemoved):
		return closeStreamRemoved, "stream removed"
	case errors.Is(err, hub.ErrSlowConsumer):
		return closeTooSlow, "too slow"
	case errors.Is(err, hub.ErrShuttingDown):
		return websocket.StatusGoingAway, "server shutting down"
	default:
		return websocket.StatusNormalClosure, ""
	}
}
