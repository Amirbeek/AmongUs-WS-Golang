# Realtime Rooms Chat (Go + WebSocket + Redis)

A lightweight, horizontally scalable realtime **rooms** service built with **Go**, **Gorilla WebSocket**, and **Redis**.
It uses a **publish-only** pattern with **Redis Pub/Sub** (for cross-instance fan-out) and a short **Redis List** history (for quick backfill on join). Clean concurrency, minimal dependencies, production-friendly.

---

## Table of Contents

* [Description](#description)
* [Tech Stack](#tech-stack)
* [Key Features](#key-features)
* [Architecture](#architecture)
* [Installation](#installation)
* [Usage](#usage)
* [Contributing Guidelines](#contributing-guidelines)
* [License](#license)
* [Author Info](#author-info)

---

## Description

This service provides room-based realtime messaging over WebSockets. Messages from clients are published to Redis once, and **every** server instance subscribed to that room’s channel receives the message and fans it out to its local WebSocket clients. On join, clients get a short backfill (last N messages) from Redis so they don’t miss context.

Why it’s useful:

* Simple, robust pattern to scale horizontally (multiple app instances).
* Clear separation of **global distribution** (Redis) and **local fan-out** (server).
* Easy to reason about ordering and duplicates (single canonical code path).

---

## Tech Stack

* **Language:** Go 1.21+
* **WebSocket:** [gorilla/websocket](https://github.com/gorilla/websocket)
* **Data/Queue:** Redis 6+

  * **Pub/Sub** for realtime broadcast
  * **List** for last-N backfill
* **Redis Client:** [redis/go-redis](https://github.com/redis/go-redis)

---

## Key Features

* Room-scoped realtime messaging.
* **Publish-only** design: one canonical flow (prevents duplicates).
* Cross-instance fan-out via **Redis Pub/Sub**.
* Last-N history backfill via **Redis List** (`RPush`/`LTrim`).
* Presence/state broadcasting (`"state"` envelopes).
* Concurrency-safe (`sync.RWMutex`; guarded subscriber lifecycle).
* Origin check for WS security.
* Clean shutdown/restart behavior; rooms subscribe only when needed.

---

## Architecture

```
Client → WebSocket → readPump → Publish(msg) ────→ Redis Pub/Sub: room:{CODE}:pub
                                                ↘
                                     (every instance) subscribe() → r.broadcast → run()
                                                                ↘              ↘
                                                              clients' send → writePump → WS
```

* **Publish**: emits to Redis only (no direct local fan-out).
* **subscribe()**: receives from Redis and pushes into the room’s `broadcast` channel.
* **run()**: room event loop — fans out from `broadcast` to local clients; handles `register`/`unregister`; emits `state`.

---

## Installation

### Prerequisites

* Go **1.21+**
* Redis **6+** running on `localhost:6379` (or adjust the address in code)

### Clone & Build

```bash
git clone https://github.com/your-username/your-repo.git
cd your-repo

go mod tidy
go build ./...
```

### Run the Server

```bash
# Ensure Redis is running
redis-server &

# Start the Go server (listens on :3000)
go run ./...
```

### Configuration

* **Redis address/DB:** edit the `rdb := redis.NewClient(...)` options in `manager.go`.
* **Allowed origins (CORS-like):** update `checkOrigin` in `manager.go`:

```go
func checkOrigin(r *http.Request) bool {
    switch r.Header.Get("Origin") {
    case "http://localhost:3000":
        return true
    default:
        return false
    }
}
```

* **Backfill length:** change `LTrim(key, -50, -1)` in `client.go`.

---

## Usage

### Endpoints

* `GET /` — serves static files from `./public` (optional demo UI)
* `GET /rooms` — JSON list of rooms with client counts
* `GET /ws?room={CODE}&name={USERNAME}` — upgrades to WebSocket and joins a room

### Connect (example)

Open a WebSocket client to:

```
ws://localhost:3000/ws?room=ABCD&name=Alice
```

You can test with **wscat**:

```bash
npm i -g wscat
wscat -c "ws://localhost:3000/ws?room=ABCD&name=Alice"
```

Open a second client:

```bash
wscat -c "ws://localhost:3000/ws?room=ABCD&name=Bob"
```

### Send Messages (envelopes)

**Chat**

```json
{"type":"chat","data":{"text":"Hello","from":"Alice"}}
```

**Ready/Agree**

```json
{"type":"agree","data":{"username":"Alice"}}
```

**State** (server-generated example)

```json
{
  "type": "state",
  "data": {
    "room": "ABCD",
    "players": [
      {"id":"...","name":"Alice","alive":true,"ready":true},
      {"id":"...","name":"Bob","alive":true,"ready":false}
    ]
  }
}
```

> On join, the server sends the last N messages from Redis List as backfill.

### Troubleshooting

* **Duplicate sends for one recv:**

  * Ensure **only** `subscribe()` writes to `r.broadcast` (publish-only).
  * Ensure `run()` does **not** spawn `subscribe()` ad-hoc; use `r.ensureSubscriber()` only.
  * Check multiple browser tabs/sockets: each open socket gets its own copy.

* **Subscriber lifecycle:**

  * Rooms start subscribing on first client (`ensureSubscriber()`),
  * stop when empty (`stopSubscriber()`),
  * and restart on new joins.

---

## Contributing Guidelines

Contributions welcome!

1. Fork the repo and create a feature branch.
2. Keep changes small and focused.
3. Add tests where feasible; run `go vet` and `golangci-lint` if available.
4. Open a PR with a clear description and rationale.

*(Optional: add a `CONTRIBUTING.md` and link it here.)*

---

## License

**MIT** — feel free to use, modify, and distribute under the terms of the MIT license.

---

## Author Info

**Your Name**

* Email: [amirbek.shomurodov01@gmail.com](mailto:your.email@example.com)
* GitHub: [Amirbeek](https://github.com/Amirbeek)
