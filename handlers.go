package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

type handlerMap map[string]func(net.Conn, string) error

var handlers handlerMap

func init() {
	handlers = handlerMap{
		"initHandshake":   initHandshake,
		"handshakeOk":     handshakeOk,
		"ping":            ping,
		"listRunningBots": listRunningBots,
		"startBot":        startBot,
		"stopBot":         stopBot,
	}
}

func initHandshake(conn net.Conn, data string) error {
	sendPacket(conn, "initHandshake", `{"machineId":"`+CLIENT_UUID+`"}`)
	return nil
}

func handshakeOk(conn net.Conn, data string) error {
	customerId, err := strconv.Atoi(data)
	if err != nil {
		return err
	}
	CUSTOMER_ID = &customerId
	return nil
}

func ping(conn net.Conn, data string) error {
	log.Println("Heartbeat received")
	return nil
}

type RunningBots map[string][]string

func listRunningBots(conn net.Conn, data string) error {
	// todo: track all running accounts and list them here
	runningBots := RunningBots{
		"agent_uuid_here": []string{"account_id_here", "another_account_id_here"},
	}
	runningBotsJson, _ := json.Marshal(runningBots)

	log.Println("Sending list of running bots")

	sendEncryptedPacket(conn, "agentData", string(runningBotsJson))
	return nil
}

type startBotData struct {
	ServerId        string `json:"serverId"`
	InternalId      int    `json:"internalId"`
	JarLocation     string `json:"jarLocation"`
	ScriptsLocation string `json:"scriptsLocation"`
	ScriptName      string `json:"scriptName"`
	ScriptParams    string `json:"scriptParams"`
	ClientName      string `json:"clientName"`
	ClientPassword  string `json:"clientPassword"`
	AccountUsername string `json:"accountUsername"`
	AccountPassword string `json:"accountPassword"`
	ProxyHost       string `json:"proxyHost"`
	ProxyPort       int    `json:"proxyPort"`
	ProxyUsername   string `json:"proxyUsername"`
	ProxyPassword   string `json:"proxyPassword"`
	AccountTotp     string `json:"accountTotp"`
	Fps             int    `json:"fps"`
	World           string `json:"world"`
}

func startBot(conn net.Conn, data string) error {
	var args startBotData
	err := json.Unmarshal([]byte(data), &args)
	if err != nil {
		return err
	}
	fmt.Println(args)

	log.Println("Starting account ID", args.InternalId)

	clients[args.InternalId] = &struct{}{}

	go func() {
		cmdArgs := []string{
			"-jar",
			args.JarLocation,
			"-script",
			"BotBuddyWrapper",
			//args.ScriptParams,
			"-username",
			args.ClientName,
			"-password",
			args.ClientPassword,
			"-accountUsername",
			args.AccountUsername,
			"-accountPassword",
			args.AccountPassword,
			"-userhome",
			"BotBuddy/" + strconv.Itoa(args.InternalId),
			"-destroy",
		}

		if args.World != "" {
			cmdArgs = append(cmdArgs, "-world", args.World)
		}

		if args.Fps > 0 {
			cmdArgs = append(cmdArgs, "-fps", strconv.Itoa(args.Fps))
		}

		if args.ProxyHost != "" {
			cmdArgs = append(cmdArgs, "-proxyHost", args.ProxyHost)
			cmdArgs = append(cmdArgs, "-proxyPort", strconv.Itoa(args.ProxyPort))

			if args.ProxyUsername != "" {
				cmdArgs = append(cmdArgs, "-proxyUser", args.ProxyUsername)
			}

			if args.ProxyPassword != "" {
				cmdArgs = append(cmdArgs, "-proxyPass", args.ProxyPassword)
			}
		}

		cmdArgs = append(cmdArgs, "-params", args.ScriptName)

		if args.ScriptParams != "" {
			cmdArgs = append(cmdArgs, args.ScriptParams)
		}

		cmd := exec.Command("java", cmdArgs...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Println("Error creating StdoutPipe for Cmd", err)
			return
		}

		err = cmd.Start()
		if err != nil {
			fmt.Println("Error starting Cmd", err)
			return
		}

		pid := cmd.Process.Pid
		fmt.Println(pid)
		//pids[pid] = internalId

		reader := bufio.NewReader(stdout)

		for {
			line, err := reader.ReadString('\n')
			if err != nil || io.EOF == err {
				break
			}

			line = strings.Trim(line, "\n")
			fmt.Println(line)

			if len(logHandlers) != 0 {
				for _, l := range logHandlers {
					if strings.Contains(strings.ToLower(line), strings.ToLower(l.waitingFor)) {
						err := l.action.execute(args.InternalId, args.AccountUsername, line)
						if err != nil {
							continue
						}
					}
				}
			}

			/*if len(logEvents) != 0 {
				for _, l := range logEvents {
					if strings.Contains(strings.ToLower(line), strings.ToLower(l.waitingFor())) {
						//l.action().execute(internalId, login, line)
					}
				}
			}*/
		}

		err = cmd.Wait()
		if err != nil {
			fmt.Println("Error waiting for Cmd", err)
			return
		}

		//sendProcessExitNotification(internalId, login)
		//delete(pids, pid)
	}()

	return nil
}

type stopBotData struct {
	InternalId int `json:"internalId"`
}

func stopBot(conn net.Conn, data string) error {
	var args stopBotData
	err := json.Unmarshal([]byte(data), &args)
	if err != nil {
		return err
	}

	log.Println("Stopping account ID", args.InternalId)

	if _, ok := clients[args.InternalId]; ok {
		delete(clients, args.InternalId)
		// todo: stop the client jar
	}

	return nil
}
