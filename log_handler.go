package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
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
		LogEvent{"trade unrestricted, stopping", ReportCompleted{}},
		LogEvent{"running: none", ReportNoScript{}},
		LogEvent{"There was a problem authorizing your account", ReportNoScript{}},
		LogEvent{"BB_OUTPUT", ReportWrapperData{}},
		LogEvent{"blocked from the game", ReportBotStatus{online: false, proxyBlocked: true}},
	)
}

type LogEvent struct {
	waitingFor string
	action     Action
}

type Action interface {
	execute(conn net.Conn, internalId int, loginName string, logLine string) error
}

type ReportBotStatus struct {
	online       bool
	proxyBlocked bool
}

func (r ReportBotStatus) execute(conn net.Conn, internalId int, loginName string, logLine string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

	if r.proxyBlocked {
		log.Println(loginName + " has been detected as having a " + Red + "blocked proxy" + Reset + ".")
		ChangeClientStatus(internalId, "ProxyBlocked")
		sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"ProxyBlocked"}`, internalId))
		return nil
	}

	if r.online {
		log.Println(loginName + " has been detected as " + Green + "running" + Reset + ".")
		ChangeClientStatus(internalId, "Running")
		sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Running"}`, internalId))
	} else {
		log.Println(loginName + " has been detected as " + Red + "stopped" + Reset + ".")
		ChangeClientStatus(internalId, "Stopped")
		sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Stopped"}`, internalId))
	}

	return nil
}

type ReportBan struct{}

func (r ReportBan) execute(conn net.Conn, internalId int, loginName string, logLine string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

	log.Println(loginName + " has been been detected as " + Red + "banned" + Reset + ".")
	ChangeClientStatus(internalId, "Banned")

	sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Banned"}`, internalId))
	return nil
}

type ReportLock struct{}

func (r ReportLock) execute(conn net.Conn, internalId int, loginName string, logLine string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}
	log.Println(loginName + " has been detected as " + Red + "locked" + Reset + ".")
	ChangeClientStatus(internalId, "Locked")
	sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Locked"}`, internalId))
	return nil
}

type ReportCompleted struct{}

func (r ReportCompleted) execute(conn net.Conn, internalId int, loginName string, logLine string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

	mutex.Lock()
	lastCompleted, exists := completedLast[internalId]
	mutex.Unlock()

	if !exists || time.Now().Unix()-lastCompleted >= 10 {
		log.Println(loginName + " has been detected as " + Green + "completed" + Reset + ".")
		ChangeClientStatus(internalId, "Completed")
		sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Completed"}`, internalId))

		mutex.Lock()
		completedLast[internalId] = time.Now().Unix()
		mutex.Unlock()
	}

	return nil
}

type ReportNoScript struct{}

func (r ReportNoScript) execute(conn net.Conn, internalId int, loginName string, logLine string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

	if GetClientUptime(internalId) >= 30 {
		log.Println(loginName + " has been detected as " + Red + "scriptless" + Reset + ".")
		ChangeClientStatus(internalId, "NoScript")
		sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"NoScript"}`, internalId))
	}

	return nil
}

type ReportWrapperData struct{}

func (r ReportWrapperData) execute(conn net.Conn, internalId int, loginName string, logLine string) error {
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

	sendEncryptedPacket(conn, "wrapperData", string(send))

	return nil
}
