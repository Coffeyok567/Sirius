package handlers

import (
    "log"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/gorilla/websocket"

    "messenger/internal/auth"
    wshub "messenger/internal/websocket"
)

var wsUpgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin:     func(r *http.Request) bool { return true },
}

type WebSocketHandler struct {
    hub  *wshub.Hub
    auth *auth.AuthService
}

func NewWebSocketHandler(hub *wshub.Hub, authSvc *auth.AuthService) *WebSocketHandler {
    return &WebSocketHandler{hub: hub, auth: authSvc}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
    token := c.Query("token")
    if token == "" {
        c.AbortWithStatus(http.StatusUnauthorized)
        return
    }

    claims, err := h.auth.ValidateToken(token)
    if err != nil {
        c.AbortWithStatus(http.StatusUnauthorized)
        return
    }

    conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Println("websocket upgrade:", err)
        return
    }

    client := &wshub.Client{
        ID:     uuid.New().String(),
        UserID: claims.UserID,
        Conn:   conn,
        Send:   make(chan []byte, 256),
        Hub:    h.hub,
    }

    h.hub.Join(client)
    go client.WritePump()
    client.ReadPump()
}
