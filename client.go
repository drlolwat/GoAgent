package main

import (
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type Client struct {
	Pid          int
	InternalId   int
	Script       string
	Status       string
	StartedAt    int64
	Port         int
	LoginName    string
	LoginPass    string
	LoginTotp    string
	HandledLogin bool
}

type SafeClients struct {
	clients map[int]*Client
	mux     sync.RWMutex
}

var safeClients = SafeClients{clients: make(map[int]*Client)}

func NewClient(pid int, internalId int, status string, script string, port int, loginName string, loginPass string, loginTotp string) *Client {
	client := &Client{
		Pid:          pid,
		InternalId:   internalId,
		Status:       status,
		Script:       script,
		StartedAt:    time.Now().Unix(),
		Port:         port,
		LoginName:    loginName,
		LoginPass:    loginPass,
		LoginTotp:    loginTotp,
		HandledLogin: false,
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
		killProcess(internalId, client.LoginName)
		delete(safeClients.clients, internalId)
	}

	safeClients.mux.Unlock()
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

func killProcess(internalId int, email string) {
	userhome := "BotBuddy/" + strconv.Itoa(internalId)

	var pids []int32
	procs, err := process.Processes()
	if err == nil {
		for _, p := range procs {
			name, _ := p.Name()
			if name != "" && !strings.Contains(strings.ToLower(name), "java") {
				continue
			}

			cmdline, err := p.Cmdline()
			if err != nil || cmdline == "" {
				continue
			}

			if strings.Contains(strings.ToLower(cmdline), strings.ToLower(userhome)) {
				pids = append(pids, p.Pid)
			}
		}
	}

	if len(pids) == 0 {
		return
	}

	for _, pid := range pids {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(int(pid)))
		} else {
			cmd = exec.Command("kill", "-9", strconv.Itoa(int(pid)))
		}
		_ = cmd.Run()
	}

	time.Sleep(3 * time.Second)

	stillRunning := false
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			continue
		}
		r, err := p.IsRunning()
		if err == nil && r {
			stillRunning = true
			break
		}
	}

	if !stillRunning {
		err := ReportBotStatus{online: false, proxyBlocked: false}.execute(Master, internalId, email, "Client killed successfully", "", "")
		if err != nil {
			log.Println("1:", err)
			return
		}
	}
}
