package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	if *CUSTOMER_ID > 0 {
		log.Println(Green + "Connected to BotBuddy network." + Reset)
	}

	return nil
}

func ping(conn net.Conn, data string) error {
	return nil
}

func listRunningBots(conn net.Conn, data string) error {
	runningBots := make(map[string][]map[string]string)

	if len(clients) == 0 {
		runningBots[CLIENT_UUID] = []map[string]string{}
	} else {
		for _, client := range clients {
			runningBots[CLIENT_UUID] = append(runningBots[CLIENT_UUID], map[string]string{"internalId": strconv.Itoa(client.InternalId), "status": client.Status})
		}
	}

	runningBotsJson, _ := json.Marshal(runningBots)
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

func wrapperExists(scriptsFolder string) bool {
	filePath := filepath.Join(scriptsFolder, WRAPPER_JAR)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func downloadWrapper(scriptsFolder string) error {
	log.Println("Downloading BotBuddy wrapper...")
	filePath := filepath.Join(scriptsFolder, WRAPPER_JAR)

	// If the file exists, delete it
	if _, err := os.Stat(filePath); err == nil {
		err = os.Remove(filePath)
		if err != nil {
			return err
		}
	}

	// Download the file
	url := DIST_URL + "/" + WRAPPER_JAR
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer func(out *os.File) {
		err := out.Close()
		if err != nil {

		}
	}(out)

	_, err = io.Copy(out, resp.Body)
	return err
}

func startBot(conn net.Conn, data string) error {
	var args startBotData
	err := json.Unmarshal([]byte(data), &args)
	if err != nil {
		return err
	}

	if !wrapperExists(args.ScriptsLocation) {
		err = downloadWrapper(args.ScriptsLocation)
		if err != nil {
			return err
		}
	}

	if IsClientRunning(args.InternalId) {
		return errors.New("Client is already running for " + args.AccountUsername)
	}

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
		_ = NewClient(pid, args.InternalId, "Running")
		log.Println("Starting client for", args.AccountUsername)

		reader := bufio.NewReader(stdout)

		for {
			line, err := reader.ReadString('\n')
			if err != nil || io.EOF == err {
				break
			}

			line = strings.Trim(line, "\n")

			if len(logHandlers) > 0 {
				for _, l := range logHandlers {
					if strings.Contains(strings.ToLower(line), strings.ToLower(l.waitingFor)) {
						err := l.action.execute(conn, args.InternalId, args.AccountUsername, line)
						if err != nil {
							log.Println(err)
							continue
						}
					}
				}
			}
		}

		_ = cmd.Wait()
		RemoveClientByInternalId(args.InternalId)
		sendProcessExitNotification(conn, args.InternalId, args.AccountUsername)
	}()

	return nil
}

func sendProcessExitNotification(conn net.Conn, internalId int, loginName string) {
	if conn != nil {
		err := ReportBotStatus{
			online:       false,
			proxyBlocked: false,
		}.execute(conn, internalId, loginName, "")

		if err != nil {
			log.Println(err)
		}
	}
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

	if _, ok := clients[args.InternalId]; ok {
		StopBotByInternalId(args.InternalId)
	}

	return nil
}
