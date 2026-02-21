# WebSocket & SSE

---

## WebSocket (`pkg/ws`)

Kashvi's WebSocket support uses the [gorilla/websocket](https://github.com/gorilla/websocket) library with a Hub/Client pattern for broadcasting to multiple connected clients.

### 1. Create and start a Hub

```go
// In your package (e.g. app/hubs/chat.go):
package hubs

import "github.com/shashiranjanraj/kashvi/pkg/ws"

var Chat = ws.NewHub()

func init() {
    go Chat.Run() // starts the event loop
}
```

### 2. Register the WebSocket route

```go
import (
    "github.com/shashiranjanraj/kashvi/app/hubs"
    appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"
    "github.com/shashiranjanraj/kashvi/pkg/ws"
)

r.Get("/ws/chat", "ws.chat", appctx.Wrap(func(c *appctx.Context) {
    ws.Upgrade(c.W, c.R, hubs.Chat)
}))
```

### 3. Handle inbound messages

```go
hubs.Chat.OnMessage = func(hub *ws.Hub, msg ws.Message) {
    // Echo back to all clients
    hub.Broadcast <- msg.Data

    // Or respond only to the sender
    msg.Client.Send([]byte(`{"type":"ack"}`))
}
```

### 4. Broadcast from anywhere

```go
// From a controller, job, or anywhere:
hubs.Chat.Broadcast <- []byte(`{"type":"update","data":"live score changed"}`)

// Check connected clients
count := hubs.Chat.ClientCount()
```

### Features

- **Ping/Pong keepalive** — automatically sends WebSocket `ping` frames every 54s
- **Client buffer** — each client has a 256-message send buffer; slow clients are disconnected
- **Origin check** — configurable (allow-all by default):
  ```go
  ws.SetCheckOrigin(func(r *http.Request) bool {
      return r.Header.Get("Origin") == "https://myapp.com"
  })
  ```

### WebSocket JavaScript client

```javascript
const socket = new WebSocket("ws://localhost:8080/ws/chat");

socket.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log("received:", data);
};

socket.send(JSON.stringify({ type: "message", text: "Hello!" }));
```

---

## Server-Sent Events (`pkg/sse`)

SSE is a one-way push from server to browser over a regular HTTP connection. Great for live feeds, notifications, dashboards.

### Route handler

```go
r.Get("/events/feed", "sse.feed", appctx.Wrap(func(c *appctx.Context) {
    stream := sse.New(c.W, c.R)
    if stream == nil {
        return // client doesn't support SSE
    }

    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case t := <-ticker.C:
            stream.Send("tick", map[string]any{
                "time":  t.Format(time.RFC3339),
                "count": hubs.Chat.ClientCount(),
            })
        }

        if stream.IsClosed() {
            break // client disconnected
        }
    }
}))
```

### SSE Methods

```go
stream := sse.New(w, r)

// Named event with JSON data
stream.Send("update", map[string]any{"id": 1, "status": "done"})

// Plain data line
stream.SendRaw("hello world")

// Keepalive heartbeat (comment line)
stream.Comment("heartbeat")

// Check if client disconnected
if stream.IsClosed() { return }
```

### JavaScript client

```javascript
const es = new EventSource("/events/feed");

es.addEventListener("tick", (event) => {
    const data = JSON.parse(event.data);
    console.log("tick:", data.time);
});

es.addEventListener("update", (event) => {
    const data = JSON.parse(event.data);
    console.log("update:", data);
});
```

---

## WebSocket vs SSE

| | WebSocket | SSE |
|---|---|---|
| Direction | Bidirectional | Server → Client only |
| Protocol | Custom (ws://) | HTTP |
| Reconnect | Manual | Automatic |
| Use case | Chat, games, live collab | Notifications, feeds, dashboards |
| Browser support | All | All (IE11+) |
