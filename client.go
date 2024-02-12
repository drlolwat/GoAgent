package main

import (
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

type Client struct {
	Pid        int
	InternalId int
	Status     string
	StartedAt  int64
}

var clients = make(map[int]*Client)

func NewClient(pid int, internalId int, status string) *Client {
	client := &Client{
		Pid:        pid,
		InternalId: internalId,
		Status:     status,
		StartedAt:  time.Now().Unix(),
	}

	clients[internalId] = client
	return client
}

func IsClientRunning(internalId int) bool {
	_, exists := clients[internalId]
	return exists
}

func RemoveClientByInternalId(internalId int) {
	delete(clients, internalId)
}

func StopBotByInternalId(internalId int) {
	if client, exists := clients[internalId]; exists {
		killProcess(client.Pid)
		delete(clients, internalId)
	}
}

func killProcess(pid int) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	} else {
		cmd = exec.Command("kill", "-9", strconv.Itoa(pid))
	}

	err := cmd.Run()
	if err != nil {
		return
	}
}

func ChangeClientStatus(internalId int, newStatus string) {
	if client, exists := clients[internalId]; exists {
		client.Status = newStatus
	}
}

func GetClientUptime(internalId int) int64 {
	if client, exists := clients[internalId]; exists {
		return time.Now().Unix() - client.StartedAt
	}
	return -1
}
