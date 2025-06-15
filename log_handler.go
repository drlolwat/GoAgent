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

var banQueue = make(chan banMessage, 200)
var proxyBlockedQueue = make(chan banMessage, 200)

type banMessage struct {
	conn       net.Conn
	internalId int
	loginName  string
	script     string
}

func init() {
	go processBanQueue()
	go processProxyBlockedQueue()
}

func processProxyBlockedQueue() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case msg := <-proxyBlockedQueue:
				processProxyBlockedMessage(msg)
			default:
				// No messages to process, continue
			}
		}
	}
}

func processBanQueue() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case msg := <-banQueue:
				processBanMessage(msg)
			default:
				// No messages to process, continue
			}
		}
	}
}

func processBanMessage(msg banMessage) {
	log.Println(msg.loginName + " has been detected as " + Red + "banned" + Reset + ".")
	ChangeClientStatus(msg.internalId, "Banned")

	err := sendEncryptedPacket(msg.conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Banned","Script":"%s"}`, msg.internalId, msg.script))
	if err != nil {
		log.Println("Error sending encrypted packet:", err)
	}
}

func processProxyBlockedMessage(msg banMessage) {
	log.Println(msg.loginName + " has been detected as having a " + Red + "blocked proxy" + Reset + ".")
	ChangeClientStatus(msg.internalId, "ProxyBlocked")

	err := sendEncryptedPacket(msg.conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"ProxyBlocked","Script":"%s"}`, msg.internalId, msg.script))
	if err != nil {
		log.Println("Error sending encrypted packet:", err)
	}
}

func ClearLogHandlers() {
	logHandlers = logHandler{}

	logHandlers = append(logHandlers,
		LogEvent{"botbuddy_system", "has started successfully", ReportBotStatus{online: true, proxyBlocked: false}},
		LogEvent{"botbuddy_system", "High severity server response, stopping script! Response: DISABLED", ReportBan{}},
		LogEvent{"botbuddy_system", "response: locked", ReportLock{}},
		LogEvent{"botbuddy_system", "there was a problem authorizing your account", ReportNoScript{}},
		LogEvent{"botbuddy_system", "BB_OUTPUT", ReportWrapperData{}},
		LogEvent{"botbuddy_system", "blocked from the game", ReportBotStatus{online: false, proxyBlocked: true}},
		LogEvent{"botbuddy_system", "Failed to connect to the game, retrying...", ReportBotStatus{online: false, proxyBlocked: true}},
		LogEvent{"botbuddy_system", "initialize on thread", HandleBrowser{}},
		LogEvent{"botbuddy_system", "Never successfully authed with the browser", ReportBotStatus{online: false, proxyBlocked: true}},

		// default handlers
		LogEvent{"botbuddy_system", ">>> Reached ", ReportCompleted{}},
		LogEvent{"botbuddy_system", "reached target ttl and qp", ReportCompleted{}},
		LogEvent{"botbuddy_system", "All goals completed, or you've run out of gold! Stopping GAIO.", ReportCompleted{}},
		LogEvent{"botbuddy_system", "reached non-99 target levels and qp", ReportCompleted{}},
		LogEvent{"botbuddy_system", "SCRIPT HAS COMPLETED. THANKS FOR RUNNING!", ReportCompleted{}},
		LogEvent{"botbuddy_system", "waio: job done", ReportCompleted{}},
		LogEvent{"botbuddy_system", "[ACTION] Stop Script", ReportCompleted{}},
		LogEvent{"botbuddy_system", "tutorial island complete! stopping script", ReportCompleted{}},
		LogEvent{"botbuddy_system", "Thank you for using braveTutorial", ReportCompleted{}},
		LogEvent{"botbuddy_system", "you have completed all your quest tasks", ReportCompleted{}},
		LogEvent{"botbuddy_system", "Finished dumping all items!", ReportCompleted{}},
		LogEvent{"botbuddy_system", "Build: Tutorial completed", ReportCompleted{}},
		LogEvent{"botbuddy_system", "Build: Account completed", ReportCompleted{}},
		LogEvent{"botbuddy_system", "CORE: Handling completion", ReportCompleted{}},
		LogEvent{"botbuddy_system", "trade unrestricted, stopping", ReportCompleted{}},
	)
}

func AddCompletionHandler(scriptName, line string) {
	for _, handler := range logHandlers {
		if handler.scriptName == scriptName && handler.waitingFor == line {
			log.Println("handler already exists: ", scriptName, line)
			return
		}
	}

	logHandlers = append(logHandlers, LogEvent{scriptName, line, ReportCompleted{}})
}

type LogEvent struct {
	scriptName string
	waitingFor string
	action     Action
}

type Action interface {
	execute(conn net.Conn, internalId int, loginName string, logLine string, script string, clientTotp string) error
}

type HandleBrowser struct{}

func (h HandleBrowser) execute(conn net.Conn, internalId int, _ string, _ string, _ string, clientTotp string) error {
	authType := "totp"
	if strings.HasPrefix(clientTotp, "mailtm:") {
		authType = "mail"
	}

	payload := fmt.Sprintf(`{"internalId":%d,"authType":"%s"}`, internalId, authType)
	err := sendEncryptedPacket(conn, "requestLink", payload)
	if err != nil {
		return err
	}
	return nil
}

type ReportBotStatus struct {
	online       bool
	proxyBlocked bool
}

func (r ReportBotStatus) execute(conn net.Conn, internalId int, loginName string, logLine string, script string, _ string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

	if r.proxyBlocked {
		msg := banMessage{
			conn:       conn,
			internalId: internalId,
			loginName:  loginName,
			script:     script,
		}
		proxyBlockedQueue <- msg
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

func (r ReportBan) execute(conn net.Conn, internalId int, loginName string, logLine string, script string, _ string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

	msg := banMessage{
		conn:       conn,
		internalId: internalId,
		loginName:  loginName,
		script:     script,
	}
	banQueue <- msg

	return nil
}

type ReportLock struct{}

func (r ReportLock) execute(conn net.Conn, internalId int, loginName string, _, script string, _ string) error {
	log.Println(loginName + " has been detected as " + Red + "locked" + Reset + ".")
	ChangeClientStatus(internalId, "Locked")
	err := sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Locked","Script":"%s"}`, internalId, script))
	if err != nil {
		return err
	}
	return nil
}

type ReportCompleted struct{}

func (r ReportCompleted) execute(conn net.Conn, internalId int, loginName string, logLine string, script string, _ string) error {
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

func (r ReportNoScript) execute(conn net.Conn, internalId int, loginName string, logLine string, script string, _ string) error {
	if GetClientUptime(internalId) >= 30 {
		StopBotByInternalId(internalId)
		log.Println(loginName + " has been detected as " + Red + "scriptless" + Reset + ", stopping client.")
		err := sendEncryptedPacket(conn, "updateBot", fmt.Sprintf(`{"Id":%d,"Status":"Stopped","Script":"%s"}`, internalId, script))
		if err != nil {
			return err
		}
	}

	return nil
}

type ReportWrapperData struct{}

func (r ReportWrapperData) execute(conn net.Conn, internalId int, loginName string, logLine string, _ string, _ string) error {
	if conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

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
