package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type logHandler []LogEvent

var logHandlers logHandler

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
	execute(internalId int, loginName string, logLine string) error
}

type ReportBotStatus struct {
	online       bool
	proxyBlocked bool
}

func (r ReportBotStatus) execute(internalId int, loginName string, logLine string) error {
	if Conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}

	if r.proxyBlocked {
		fmt.Println(loginName + " has been detected as using a blocked proxy.")
		sendEncryptedPacket(Conn, "updateBot", fmt.Sprintf(`{"Id":%d,"status":"ProxyBlocked"}`, internalId))
		return nil
	}

	if r.online {
		sendEncryptedPacket(Conn, "updateBot", fmt.Sprintf(`{"Id":%d,"status":"Running"}`, internalId))
	} else {
		sendEncryptedPacket(Conn, "updateBot", fmt.Sprintf(`{"Id":%d,"status":"Stopped"}`, internalId))
	}

	return nil
}

type ReportBan struct{}

func (r ReportBan) execute(internalId int, loginName string, logLine string) error {
	if Conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}
	fmt.Println(loginName + " has been been detected as banned")
	sendEncryptedPacket(Conn, "updateBot", fmt.Sprintf(`{"Id":%d,"status":"Banned"}`, internalId))
	return nil
}

type ReportLock struct{}

func (r ReportLock) execute(internalId int, loginName string, logLine string) error {
	if Conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}
	fmt.Println(loginName + " has been detected as locked")
	sendEncryptedPacket(Conn, "updateBot", fmt.Sprintf(`{"Id":%d,"status":"Locked"}`, internalId))
	return nil
}

type ReportCompleted struct{}

func (r ReportCompleted) execute(internalId int, loginName string, logLine string) error {
	if Conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}
	fmt.Println(loginName + " has been detected as completed")
	sendEncryptedPacket(Conn, "updateBot", fmt.Sprintf(`{"Id":%d,"status":"Completed"}`, internalId))
	return nil
}

type ReportNoScript struct{}

func (r ReportNoScript) execute(internalId int, loginName string, logLine string) error {
	if Conn == nil {
		return errors.New("was not connected to BotBuddy network")
	}
	fmt.Println(loginName + " has been detected as running no script")
	sendEncryptedPacket(Conn, "updateBot", fmt.Sprintf(`{"Id":%d,"status":"NoScript"}`, internalId))
	return nil
}

type ReportWrapperData struct{}

func (r ReportWrapperData) execute(internalId int, loginName string, logLine string) error {
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

	fmt.Println(send)

	return nil
}
