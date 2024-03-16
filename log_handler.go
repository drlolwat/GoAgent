package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type logHandler []LogEvent

var logHandlers logHandler

var completedLast = make(map[int]int64)
var mutex = &sync.Mutex{}

func init() {
	logHandlers = append(logHandlers,
		LogEvent{"has started successfully", ReportBotStatus{online: true, proxyBlocked: false}},
		LogEvent{"being set to banned status", ReportBan{}},
		LogEvent{"response: locked", ReportLock{}},
		LogEvent{"reached target ttl and qp", ReportCompleted{}},
		LogEvent{"reached non-99 target levels and qp", ReportCompleted{}},
		LogEvent{"SCRIPT HAS COMPLETED. THANKS FOR RUNNING!", ReportCompleted{}},
		LogEvent{"tutorial island complete! stopping script", ReportCompleted{}},
		LogEvent{"trade unrestricted, stopping", ReportCompleted{}},
		LogEvent{"running: none", ReportNoScript{}},
		LogEvent{"there was a problem authorizing your account", ReportNoScript{}},
		LogEvent{"BB_OUTPUT", ReportWrapperData{}},
		LogEvent{"blocked from the game", ReportBotStatus{online: false, proxyBlocked: true}},
		LogEvent{"initialize on thread", HandleBrowser{}},
		LogEvent{"successfully authorized your account", DeleteTemps{}},
	)
}

type LogEvent struct {
	waitingFor string
	action     Action
}

type Action interface {
	execute(conn net.Conn, internalId int, loginName string, logLine string, script string) error
}

type DeleteTemps struct{}

func (d DeleteTemps) execute(_ net.Conn, _ int, _ string, _ string, _ string) error {
	tempDir := os.TempDir()
	files, err := filepath.Glob(filepath.Join(tempDir, "wct*.tmp"))
	if err != nil {
		return nil
	}

	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			_ = err
		}
	}

	return nil
}

type HandleBrowser struct{}

func (h HandleBrowser) execute(_ net.Conn, internalId int, _ string, _ string, _ string) error {
	safeClients.mux.RLock()
	client, exists := safeClients.clients[internalId]
	safeClients.mux.RUnlock()

	if !exists {
		fmt.Println("Client with internalId", internalId, "does not exist")
		return errors.New("client does not exist")
	}

	if client.HandledLogin {
		return nil
	}

	client.HandledLogin = true
	safeClients.mux.Lock()
	safeClients.clients[internalId] = client
	safeClients.mux.Unlock()

	port := client.Port
	email := client.LoginName
	password := client.LoginPass
	totpSecret := client.LoginTotp

	cmdInstallDrissionpage := exec.Command("pip", "install", "drissionpage")
	err := cmdInstallDrissionpage.Run()
	if err != nil {
		return err
	}

	cmdInstallPyotp := exec.Command("pip", "install", "pyotp")
	err = cmdInstallPyotp.Run()
	if err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	browsePy, err := FS.ReadFile("Q9e8utMX9z/O1D4c0zKvV")
	if err != nil {
		return err
	}

	tmpfileScript, err := os.CreateTemp("", "wct*.tmp")
	if err != nil {
		return err
	}

	_, err = tmpfileScript.Write(browsePy)
	if err != nil {
		return err
	}

	cmd := exec.Command("python", tmpfileScript.Name(), "--port", strconv.Itoa(port), "--email", email, "--password", password, "--totp_secret", totpSecret)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	go func() {
		var err error
		maxRetries := 1
		for i := 0; i < maxRetries; i++ {
			_, err = cmd.Output()
			if err == nil {
				err := DeleteTemps{}.execute(nil, internalId, "", "", "")
				if err != nil {
					break
				}
				break
			}
			time.Sleep(3 * time.Second)
		}

		defer func(name string) {
			err := os.Remove(name)
			if err != nil {

			}
		}(tmpfileScript.Name())
	}()

	return nil
}

type ReportBotStatus struct {
	online       bool
	proxyBlocked bool
}

func (r ReportBotStatus) execute(conn net.Conn, internalId int, loginName string, logLine string, script string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

	if r.proxyBlocked {
		log.Println(loginName + " has been detected as having a " + Red + "blocked proxy" + Reset + ".")
		ChangeClientStatus(internalId, "ProxyBlocked")
		err := sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"ProxyBlocked","Script":"%s"}`, internalId, script))
		if err != nil {
			return err
		}
		return nil
	}

	if r.online {
		log.Println(loginName + " has been detected as " + Green + "running" + Reset + ".")
		ChangeClientStatus(internalId, "Running")
		err := sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Running","Script":"%s"}`, internalId, script))
		if err != nil {
			return err
		}
	} else {
		log.Println(loginName + " has been detected as " + Red + "stopped" + Reset + ".")
		ChangeClientStatus(internalId, "Stopped")
		err := sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Stopped","Script":"%s"}`, internalId, script))
		if err != nil {
			return err
		}
	}

	return nil
}

type ReportBan struct{}

func (r ReportBan) execute(conn net.Conn, internalId int, loginName string, logLine string, script string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

	log.Println(loginName + " has been been detected as " + Red + "banned" + Reset + ".")
	ChangeClientStatus(internalId, "Banned")

	err := sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Banned"}`, internalId))
	if err != nil {
		return err
	}
	return nil
}

type ReportLock struct{}

func (r ReportLock) execute(conn net.Conn, internalId int, loginName string, logLine string, script string) error {
	log.Println(loginName + " has been detected as " + Red + "locked" + Reset + ".")
	ChangeClientStatus(internalId, "Locked")
	err := sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Locked","Script":"%s"}`, internalId, script))
	if err != nil {
		return err
	}
	return nil
}

type ReportCompleted struct{}

func (r ReportCompleted) execute(conn net.Conn, internalId int, loginName string, logLine string, script string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

	mutex.Lock()
	lastCompleted, exists := completedLast[internalId]
	mutex.Unlock()

	if !exists || time.Now().Unix()-lastCompleted >= 10 {
		log.Println(loginName + " has been detected as " + Green + "completed" + Reset + ".")
		ChangeClientStatus(internalId, "Completed")
		err := sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Completed","Script":"%s"}`, internalId, script))
		if err != nil {
			return err
		}

		mutex.Lock()
		completedLast[internalId] = time.Now().Unix()
		mutex.Unlock()
	}

	return nil
}

type ReportNoScript struct{}

func (r ReportNoScript) execute(conn net.Conn, internalId int, loginName string, logLine string, script string) error {
	if GetClientUptime(internalId) >= 30 {
		log.Println(loginName + " has been detected as " + Red + "scriptless" + Reset + ".")
		ChangeClientStatus(internalId, "NoScript")
		err := sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"NoScript","Script":"%s"}`, internalId, script))
		if err != nil {
			return err
		}
	}

	return nil
}

type ReportWrapperData struct{}

func (r ReportWrapperData) execute(conn net.Conn, internalId int, loginName string, logLine string, script string) error {
	data := make(map[int]map[string]interface{})
	innerMap := make(map[string]interface{})

	parts := strings.Split(logLine, "BB_OUTPUT:")
	if len(parts) > 1 {
		content := strings.TrimSpace(parts[1])
		if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
			var nestedMap map[string]interface{}
			err := json.Unmarshal([]byte(content), &nestedMap)
			if err != nil {
				return err
			}
			innerMap["BB_OUTPUT"] = nestedMap
		} else {
			innerMap["BB_OUTPUT"] = content
		}
	}

	data[internalId] = innerMap

	send, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = sendEncryptedPacket(conn, "wrapperData", string(send))
	if err != nil {
		return err
	}

	return nil
}
