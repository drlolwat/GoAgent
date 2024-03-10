package main

import (
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type Client struct {
	Pid        int
	InternalId int
	Script     string
	Status     string
	StartedAt  int64
}

type SafeClients struct {
	clients map[int]*Client
	mux     sync.RWMutex
}

var safeClients = SafeClients{clients: make(map[int]*Client)}

func NewClient(pid int, internalId int, status string, script string) *Client {
	client := &Client{
		Pid:        pid,
		InternalId: internalId,
		Status:     status,
		Script:     script,
		StartedAt:  time.Now().Unix(),
	}

	safeClients.mux.Lock()
	safeClients.clients[internalId] = client
	safeClients.mux.Unlock()

	return client
}

func IsClientRunning(internalId int) bool {
	safeClients.mux.RLock()
	_, exists := safeClients.clients[internalId]
	safeClients.mux.RUnlock()

	return exists
}

func RemoveClientByInternalId(internalId int) {
	safeClients.mux.Lock()
	delete(safeClients.clients, internalId)
	safeClients.mux.Unlock()
}

func StopBotByInternalId(internalId int) {
	safeClients.mux.Lock()
	if client, exists := safeClients.clients[internalId]; exists {
		killProcess(client.Pid)
		delete(safeClients.clients, internalId)
	}
	safeClients.mux.Unlock()
}

func killProcess(pid int) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	} else {
		cmd = exec.Command("setsid", "kill", "-9", strconv.Itoa(pid))
	}

	err := cmd.Run()
	if err != nil {
		return
	}
}

func ChangeClientStatus(internalId int, newStatus string) {
	safeClients.mux.Lock()
	if client, exists := safeClients.clients[internalId]; exists {
		client.Status = newStatus
	}
	safeClients.mux.Unlock()
}

func GetClientUptime(internalId int) int64 {
	safeClients.mux.RLock()
	defer safeClients.mux.RUnlock()

	if client, exists := safeClients.clients[internalId]; exists {
		return time.Now().Unix() - client.StartedAt
	}
	return -1
}
