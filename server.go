package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ServeWS handles incoming Websocket clients
func ServeWS(w http.ResponseWriter, r *http.Request, sh *StatsHandler) {
	// Allow any origin
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	// Set server as websocket server
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("upgrade:", err)
		return
	}
	// Register client to the StatsHandler object
	uid := sh.Register(ws)

	defer ws.Close()
	// Infinite loop reading received messages
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			sh.Unregister(uid)
			fmt.Println("read:", err)
			break
		}
		fmt.Printf("recv: %s", message)
	}
}
