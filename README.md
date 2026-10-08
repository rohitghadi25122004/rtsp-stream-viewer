# RTSP Stream Viewer

Add RTSP camera URLs and watch them live in the browser.

**Live demo:** `<frontend URL>`

The backend runs on Render's free tier, so the first load can take about a minute.

## Stack

- Frontend: React, TypeScript, Vite
- Backend: Go, FFmpeg
- Video is sent over WebSockets and played with Media Source Extensions

## How it works

```
camera --RTSP--> FFmpeg --> Go server --WebSocket--> browser
```

- The backend starts one FFmpeg process per camera when someone opens it, and all viewers share it.
- H.264 cameras are passed through without re-encoding. Other codecs are converted to H.264.
- If nobody watches a stream for 30 seconds, its FFmpeg process is stopped.
- If a camera drops, the backend retries and the tile shows a countdown. A wrong password shows an error with a Retry button.

## Run locally

You need Go 1.25+, Node 20.19+ and FFmpeg.

1. Start the test cameras. Download [MediaMTX](https://github.com/bluenviron/mediamtx/releases), put it in `rtsp-demo/`, then:
   ```sh
   cd rtsp-demo
   ./mediamtx mediamtx.yml
   ```
2. Start the backend:
   ```sh
   cd backend
   go run ./cmd/server
   ```
3. Start the frontend:
   ```sh
   cd frontend
   npm install
   npm run dev
   ```
4. Open http://localhost:5173 and add a stream.

### Test streams

| URL | Notes |
|---|---|
| `rtsp://localhost:8554/pattern` | 720p, H.264 |
| `rtsp://localhost:8554/bars` | 360p, H.264 |
| `rtsp://localhost:8554/legacy` | MPEG-4, gets converted |
| `rtsp://viewer:secret@localhost:8554/secure` | Needs a password |

### Settings

Backend:

| Variable | Default |
|---|---|
| `PORT` | `8080` |
| `ALLOWED_ORIGINS` | `http://localhost:5173` |
| `SEED_STREAMS` | none. Format: `Name=rtsp://...,Name2=rtsp://...` |
| `MAX_STREAMS` | `16` |

Frontend: `VITE_API_URL`, default `http://localhost:8080`.

## Deploy

- **Backend (Render):** create a Web Service with Docker runtime and root directory `backend`. Set `ALLOWED_ORIGINS` to the frontend URL.
- **Frontend (Vercel):** use root directory `frontend`. Set `VITE_API_URL` to the backend URL.
- **Test cameras (Railway):** deploy `rtsp-demo/` with a TCP proxy on port 8554. Add those URLs to the backend's `SEED_STREAMS`.

## Project structure

```
backend/
  cmd/server/        entry point
  internal/stream/   stream list and URL checks
  internal/ffmpeg/   runs FFmpeg
  internal/fmp4/     splits FFmpeg output into chunks
  internal/hub/      shares one FFmpeg process between viewers
  internal/httpapi/  REST and WebSocket handlers
frontend/src/
  api/               backend calls
  player/            video player
  features/streams/  page, grid, tiles, form
rtsp-demo/           test camera config
```

## Limitations

- No audio.
- Streams are kept in memory, so they are lost when the server restarts.
- No login.
