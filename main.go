package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
)

const (
	// Port used by the static file server and Websocket server
	PORT int = 11235
)

func main() {
	// Mutex to wait for clients to be connected
	var mux sync.Mutex
	// Lock it at first, waiting for clients
	mux.Lock()
	// Start fetching stats
	statsHandler := StatsHandler{fetchMutex: &mux}
	go statsHandler.FetchStats()

	addr_port := fmt.Sprintf(":%d", PORT)
	var addr = flag.String("addr", addr_port, "http service address")

	// Static file server
	http.Handle("/", http.FileServer(http.Dir("static")))
	// Websockets server
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(w, r, &statsHandler)
	})
	fmt.Printf("Listening on port %d\n", PORT)

	// Start both servers
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
