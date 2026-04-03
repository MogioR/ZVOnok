# ZVOnok

[Русский README](./README.md)

An LLM slop Cursor-made video conferencing project. It is written in Go, uses little RAM, and is intended as a simple self-hosted base that you can run for yourself or adapt for your own needs.

If it helps someone, feel free to use it as a starting point.

## What It Is

A small WebRTC service with a Vue frontend and a Go backend.  
Go handles signaling, static file serving, and the music API, while the browser runs the realtime communication layer.

## Features

- ✅ Chat
- ✅ Statuses
- ✅ Mobile version
- ✅ Streams with quality selection
- ✅ Ability to queue music from YouTube
- 🚧 Minigames
- 🚧 Webcam video
- 🚧 Rooms

## Stack

- Backend: Go
- Frontend: Vue 3 + Vite
- Realtime: WebSocket + WebRTC
- Music: `yt-dlp` + `ffmpeg`
- Deployment: Docker Compose + Coturn

## Why It Might Be Useful

- Small and relatively simple codebase without heavy infrastructure
- Low memory usage thanks to the Go backend
- Easy to self-host on your own server
- Can be used as a base for a pet project or an internal tool

## Deployment

The recommended way to run it is Docker Compose.

### 1. Prepare environment variables

```bash
cp env.example .env
```

Then open `.env` and set real values for:

- `TURN_PUBLIC_IP` — public server IP
- `TURN_PRIVATE_IP` — private server IP
- `TURN_PASS` — TURN password
- optionally `HOST_PORT`

Your local `.env` should never be committed.

### 2. Start the project

```bash
docker compose up --build -d
```

After startup, the application will be available at:

```text
http://<SERVER_IP>:<HOST_PORT>
```

With default local settings, that is usually:

```text
http://localhost:8080
```

### 3. What gets started

- `zvonok` — the main application
- `coturn` — TURN server for WebRTC

### 4. Server requirements

- Docker and Docker Compose must be installed
- Required ports must be open in the firewall
- For stable WebRTC with external clients, TURN must be configured correctly

## Local Development

### Backend

```bash
cd backend
go mod tidy
go run .
```

By default the server listens on `http://localhost:8080`.

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The Vite dev server runs on `http://localhost:5173` and proxies `/ws` to the Go backend.

## Production Build Without Docker

Build the frontend:

```bash
cd frontend
npm install
npm run build
```

The built static files will be placed into `backend/static`.

Run the backend:

```bash
cd backend
go mod tidy
go run .
```

## Environment Variables

Main variables:

- `PORT` — application port inside the container
- `HOST_PORT` — exposed host port
- `STATIC_DIR` — static files directory
- `TZ` — timezone
- `TURN_PUBLIC_IP` — external IP for TURN
- `TURN_PRIVATE_IP` — internal IP for TURN
- `TURN_USER` — TURN username
- `TURN_PASS` — TURN password

The template is available in `env.example`.

## Notes

- This is not an enterprise-grade solution or a finished product
- Some parts of the UI, logic, and architecture are experimental
- The project is more about "quickly build a working thing" than creating a polished platform

## Usage

If this project is useful to you, use it however you want within your own fork, repo, or setup. Extend and adapt it freely.
