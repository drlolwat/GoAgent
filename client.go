package main

import (
	"os/exec"
	"runtime"
	"strconv"
)

type Client struct {
	Pid        int
	InternalId int
	Status     string
}

var clients = make(map[int]*Client)

func NewClient(pid int, internalId int, status string) *Client {
	client := &Client{
		Pid:        pid,
		InternalId: internalId,
		Status:     status,
	}

	clients[internalId] = client
	return client
}

func RemoveClientByPid(pid int) {
	for id, client := range clients {
		if client.Pid == pid {
			delete(clients, id)
			break
		}
	}
}

func IsClientRunning(internalId int) bool {
	_, exists := clients[internalId]
	return exists
}

func RemoveClientByInternalId(internalId int) {
	delete(clients, internalId)
}

func StopBotByPid(pid int) {
	for id, client := range clients {
		if client.Pid == pid {
			killProcess(pid)
			delete(clients, id)
			break
		}
	}
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
	cmd.Run()
}
