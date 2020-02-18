package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	BUFFER_SIZE int = 16
)

// StatsHandler is used to fetch docker stats and send it to clients
type StatsHandler struct {
	cli        *client.Client
	ctx        context.Context
	wsClients  map[uuid.UUID]*websocket.Conn
	fetchMutex *sync.Mutex
}

// FetchStats fetches stats from docker client SDK and API
func (sh *StatsHandler) FetchStats() {
	sh.ctx = context.Background()
	// Init docker client
	sh.cli, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	// Init clients list
	sh.wsClients = make(map[uuid.UUID]*websocket.Conn)

	// Get all running containers
	containers, _ := sh.cli.ContainerList(sh.ctx, types.ContainerListOptions{})

	// Infinite loop
	for {
		// Wait for mutex to be unlocked (= waiting for at least 1 client to be connected)
		sh.fetchMutex.Lock()
		sh.fetchMutex.Unlock()
		// WaitGroup to wait for all goroutines to be finished
		var wg sync.WaitGroup
		for _, container := range containers {
			// Start reading stats for each container, 1 goroutine per container
			go sh.ReadStats(&wg, container)
			wg.Add(1)
		}
		wg.Wait()
	}
}

// ReadStats reads stats infinitely for one specific container
func (sh *StatsHandler) ReadStats(wg *sync.WaitGroup, container types.Container) {
	defer wg.Done()
	// Get stats stream from docker client SDK
	stats, _ := sh.cli.ContainerStats(sh.ctx, container.ID, true)

	buffer := make([]byte, BUFFER_SIZE)
	var fullData []byte
	// Infinite loop to read buffer
	for {
		// Read content into a buffer
		n, err := stats.Body.Read(buffer)
		if err != nil {
			if err == io.EOF {
				fmt.Print(string(buffer[:n]))
				break
			}
			fmt.Println(err)
			os.Exit(1)
		}
		buffer_s := string(buffer[:n])
		fullData = append(fullData, buffer[:n]...)

		// If the reading has reached the end of the current stats reading
		if n < BUFFER_SIZE || strings.Contains(buffer_s, "\n") {
			// If there are 0 clients, close the goroutine
			if sh.GetClientsNb() == 0 {
				stats.Body.Close()
				break
			}
			var result map[string]interface{}
			// Unmarshal JSON data
			json.Unmarshal(fullData, &result)
			// fmt.Println(result["name"])
			memory_stats := result["memory_stats"].(map[string]interface{})
			// fmt.Println(memory_stats["usage"].(float64))
			// Send to clients
			sh.SendToClients(fmt.Sprintf("%s: %f", result["name"], memory_stats["usage"].(float64)))
			fullData = nil
		}
	}
}

// Register adds a client to the list
func (sh *StatsHandler) Register(wsClient *websocket.Conn) (uid uuid.UUID) {
	// Generate uuid
	uid = uuid.New()
	// Add to list
	sh.wsClients[uid] = wsClient
	// If there is now 1 client (= there was 0 before this function was called),
	// unlock mutex to start fetching docker stats
	if len(sh.wsClients) == 1 {
		sh.fetchMutex.Unlock()
	}
	return
}

// Unregister removes a client from the list
func (sh *StatsHandler) Unregister(uid uuid.UUID) {
	// Remove client
	delete(sh.wsClients, uid)
	// If there are now 0 clients, lock mutex to stop fecthing docker stats
	if len(sh.wsClients) == 0 {
		sh.fetchMutex.Lock()
	}
}

// GetClientsNb returns the number of connected clients
func (sh *StatsHandler) GetClientsNb() int {
	return len(sh.wsClients)
}

// SendToClients sends a message to all clients in the list
func (sh *StatsHandler) SendToClients(message string) {
	for _, c := range sh.wsClients {
		c.WriteMessage(websocket.TextMessage, []byte(message))
	}
}
