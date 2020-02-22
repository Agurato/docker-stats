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
	"github.com/elastic/go-sysinfo"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	BUFFER_SIZE int = 16
)

var (
	memLimit float64 = 0
)

type Stats struct {
	Id            string  `json:"id"`
	Name          string  `json:"name"`
	Memory        float64 `json:"memory"`
	MemoryLimit   float64 `json:"memoryLimit"`
	MemoryPercent float64 `json:"memoryPercent"`
	Cpu           float64 `json:"cpu"`
	NetIn         float64 `json:"netIn"`
	NetOut        float64 `json:"netOut"`
	BlockIn       uint64  `json:"blockIn"`
	BlockOut      uint64  `json:"blockOut"`
}

// StatsHandler is used to fetch docker stats and send it to clients
type StatsHandler struct {
	cli        *client.Client
	ctx        context.Context
	wsClients  map[uuid.UUID]*websocket.Conn
	fetchMutex *sync.Mutex

	containerNb  int
	currentStats []Stats
}

// FetchStats fetches stats from docker client SDK and API
func (sh *StatsHandler) FetchStats() {
	sh.ctx = context.Background()
	// Init docker client
	sh.cli, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	// Init clients list
	sh.wsClients = make(map[uuid.UUID]*websocket.Conn)

	host, _ := sysinfo.Host()
	hostMemory, _ := host.Memory()
	memLimit = float64(hostMemory.Total)

	// Infinite loop
	for {
		// Wait for mutex to be unlocked (= waiting for at least 1 client to be connected)
		sh.fetchMutex.Lock()
		sh.fetchMutex.Unlock()

		// Get all running containers
		containers, _ := sh.cli.ContainerList(sh.ctx, types.ContainerListOptions{})
		sh.containerNb = len(containers)

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
	var (
		fullData []byte

		blkRead, blkWrite uint64
		netRx, netTx      float64
		cpuPercent        float64
		mem, memPercent   float64
	)
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

			// Unmarshal JSON data
			var result types.StatsJSON
			err := json.Unmarshal(fullData, &result)
			if err != nil {
				fmt.Println(err)
			}

			// Set values
			if stats.OSType != "windows" {
				previousCPU := result.PreCPUStats.CPUUsage.TotalUsage
				previousSystem := result.PreCPUStats.SystemUsage
				cpuPercent = calculateCPUPercentUnix(previousCPU, previousSystem, &result)
				blkRead, blkWrite = calculateBlockIO(result.BlkioStats)
				mem = calculateMemUsageUnixNoCache(result.MemoryStats)
			} else {
				cpuPercent = calculateCPUPercentWindows(&result)
				blkRead = result.StorageStats.ReadSizeBytes
				blkWrite = result.StorageStats.WriteSizeBytes
				mem = float64(result.MemoryStats.PrivateWorkingSet)
			}
			memPercent = mem / memLimit * 100
			netRx, netTx = calculateNetwork(result.Networks)

			// Prepare message
			stats := Stats{
				Id:            result.ID,
				Name:          result.Name,
				Memory:        mem,
				MemoryLimit:   memLimit,
				MemoryPercent: memPercent,
				Cpu:           cpuPercent,
				NetIn:         netRx,
				NetOut:        netTx,
				BlockIn:       blkWrite,
				BlockOut:      blkRead,
			}
			sh.PrepareStats(stats)
			fullData = nil
		}
	}
}

// PrepareStats sends one message to all clients with stats for all of the containers at once
func (sh *StatsHandler) PrepareStats(stats Stats) {
	// Add container stats to the slice
	sh.currentStats = append(sh.currentStats, stats)
	// When slice is complete, send message to the clients and empty the slice
	if len(sh.currentStats) == sh.containerNb {
		message, _ := json.Marshal(sh.currentStats)
		sh.SendToClients(message)
		sh.currentStats = nil
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
func (sh *StatsHandler) SendToClients(message []byte) {
	for _, c := range sh.wsClients {
		c.WriteMessage(websocket.TextMessage, message)
	}
}
