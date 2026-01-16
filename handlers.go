package main

import (
	"bufio"
	"context"
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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type handlerMap map[string]func(net.Conn, string) error

var handlers handlerMap

var startBotQueue = make(chan startBotData)

var basePort = 9222
var portMutex = &sync.Mutex{}
var downloadedWrapper = false

func init() {
	handlers = handlerMap{
		"initHandshake":   initHandshake,
		"handshakeOk":     handshakeOk,
		"ping":            ping,
		"listRunningBots": listRunningBots,
		"startBot":        startBot,
		"stopBot":         stopBot,
		"startLink":       linkJagex,
		"startLinkMailTm": linkJagexMailTm,
		"recvCompletions": recvCompletionMessage,
	}

	go func() {
		for botData := range startBotQueue {
			err := startBotImpl(botData)
			if err != nil {
				log.Println(Red+"Error starting bot:", err, Reset)
			}
			time.Sleep(1 * time.Second)
		}
	}()
}

func initHandshake(conn net.Conn, data string) error {
	if data != AGENT_VER {
		err := conn.Close()
		if err != nil {
			return err
		}
		KeepRetrying = false
		log.Println(Red + "Incompatible agent version. Please update your agent." + Reset)
		log.Println("Latest compatible version: " + Green + "3." + data + Reset)
		return nil
	}

	err := sendPacket(conn, "initHandshake", `{"machineId":"`+CLIENT_UUID+`"}`)
	if err != nil {
		return err
	}
	return nil
}

func handshakeOk(_ net.Conn, data string) error {
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

func ping(net.Conn, string) error {
	return nil
}

func listRunningBots(conn net.Conn, _ string) error {
	return nil
}

type CompletionMessage struct {
	ScriptName string `json:"scriptName"`
	Message    string `json:"message"`
}

type recvCompletionMessages struct {
	Data []CompletionMessage `json:"data"`
}

func recvCompletionMessage(_ net.Conn, data string) error {
	var completionMessages recvCompletionMessages
	err := json.Unmarshal([]byte(data), &completionMessages)
	if err != nil {
		return err
	}

	ClearLogHandlers()

	for _, item := range completionMessages.Data {
		scriptName := strings.TrimSpace(item.ScriptName)
		message := strings.TrimSpace(item.Message)
		if scriptName != "" && message != "" {
			AddCompletionHandler(scriptName, message)
		}
	}

	log.Println(Green + "Received completions from master" + Reset)

	return nil
}

type startBotData struct {
	ServerId            string   `json:"serverId"`
	InternalId          int      `json:"internalId"`
	JarLocation         string   `json:"jarLocation"`
	ScriptsLocation     string   `json:"scriptsLocation"`
	ScriptName          string   `json:"scriptName"`
	ScriptParams        string   `json:"scriptParams"`
	ClientName          string   `json:"clientName"`
	ClientPassword      string   `json:"clientPassword"`
	AccountUsername     string   `json:"accountUsername"`
	AccountPassword     string   `json:"accountPassword"`
	ProxyHost           string   `json:"proxyHost"`
	ProxyPort           int      `json:"proxyPort"`
	ProxyUsername       string   `json:"proxyUsername"`
	ProxyPassword       string   `json:"proxyPassword"`
	AccountTotp         string   `json:"accountTotp"`
	Fps                 int      `json:"fps"`
	World               string   `json:"world"`
	JavaXms             string   `json:"javaXms"`
	JavaXmx             string   `json:"javaXmx"`
	DisableBrowserProxy bool     `json:"disableBrowserProxy"`
	StartMinimized      bool     `json:"minimized"`
	RenderType          string   `json:"render"`
	DebugMode           bool     `json:"debug"`
	Destroy             bool     `json:"destroy"`
	DisableAnimations   bool     `json:"disableAnimations"`
	DisableModels       bool     `json:"disableModels"`
	DisableSounds       bool     `json:"disableSounds"`
	LowDetail           bool     `json:"lowDetail"`
	MenuManipulation    bool     `json:"menuManipulation"`
	NoClickWalk         bool     `json:"noClickWalk"`
	DismissRandomEvents bool     `json:"dismissRandomEvents"`
	Beta                bool     `json:"beta"`
	AccountPin          string   `json:"accountPin"`
	Conn                net.Conn `json:"-"`
}

func wrapperExists(scriptsFolder string) bool {
	filePath := filepath.Join(scriptsFolder, WRAPPER_JAR)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func downloadWrapper(scriptsFolder string) error {
	log.Println("Downloading BotBuddy wrapper...")
	pattern := filepath.Join(scriptsFolder, "BotBuddyWrapper*.jar")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, file := range matches {
		if err := os.Remove(file); err != nil {
			log.Printf("Failed to remove %s: %v", file, err)
		}
	}

	filePath := filepath.Join(scriptsFolder, WRAPPER_JAR)

	url := DIST_URL + "/" + WRAPPER_JAR
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	_, err = io.Copy(out, resp.Body)
	downloadedWrapper = true
	return err
}

func startBot(conn net.Conn, data string) error {
	var args startBotData
	err := json.Unmarshal([]byte(data), &args)
	if err != nil {
		return err
	}

	args.Conn = conn
	startBotQueue <- args

	return nil
}

func waitForNewestLogFile(ctx context.Context, dir string, timeout time.Duration) (string, time.Time, error) {
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return "", time.Time{}, ctx.Err()
		default:
		}

		path, mod, err := latestFileInDir(dir)
		if err == nil && path != "" {
			return path, mod, nil
		}

		if time.Now().After(deadline) {
			if err != nil {
				return "", time.Time{}, err
			}
			return "", time.Time{}, errors.New("no log file found")
		}

		time.Sleep(250 * time.Millisecond)
	}
}

func tailSpecificFileFromEnd(ctx context.Context, path string, out chan<- string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(f)
	buf := make([]byte, 0, 4096)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		b, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				time.Sleep(150 * time.Millisecond)
				continue
			}
			return err
		}

		if b == '\n' {
			if len(buf) > 0 && buf[len(buf)-1] == '\r' {
				buf = buf[:len(buf)-1]
			}
			out <- string(buf)
			buf = buf[:0]
			continue
		}

		buf = append(buf, b)
	}
}

func startBotImpl(args startBotData) error {
	//log.Println("STARTBOTIMPL MARKER 2026-01-15 A", args.InternalId, args.AccountUsername)

	if !wrapperExists(args.ScriptsLocation) || !downloadedWrapper {
		err := downloadWrapper(args.ScriptsLocation)
		if err != nil {
			return err
		}
	}

	if IsClientRunning(args.InternalId) {
		return errors.New("Client is already running for " + args.AccountUsername)
	}

	go func(args startBotData) {
		time.Sleep(1 * time.Second)

		userhome := "BotBuddy/" + strconv.Itoa(args.InternalId)

		cmdArgs := []string{
			"-Xms" + args.JavaXms,
			"-Xmx" + args.JavaXmx,
			"-jar",
			args.JarLocation,
			"-script",
			"BotBuddyWrapper",
			"-username",
			args.ClientName,
			"-password",
			args.ClientPassword,
			"-accountUsername",
			args.AccountUsername,
			"-accountPassword",
			args.AccountPassword,
			"-userhome",
			userhome,
		}

		if len(args.AccountPin) == 4 {
			cmdArgs = append(cmdArgs, "-accountPin", args.AccountPin)
		}
		if args.World != "" {
			cmdArgs = append(cmdArgs, "-world", args.World)
		}
		if args.Fps > 0 {
			cmdArgs = append(cmdArgs, "-fps", strconv.Itoa(args.Fps))
		}
		if args.StartMinimized {
			cmdArgs = append(cmdArgs, "-minimized")
		}
		if args.RenderType != "" {
			cmdArgs = append(cmdArgs, "-render", args.RenderType)
		}
		if args.Destroy {
			cmdArgs = append(cmdArgs, "-destroy")
		}
		if args.DisableAnimations {
			cmdArgs = append(cmdArgs, "-disableAnimations")
		}
		if args.DisableModels {
			cmdArgs = append(cmdArgs, "-disableModels")
		}
		if args.DisableSounds {
			cmdArgs = append(cmdArgs, "-disableSounds")
		}
		if args.LowDetail {
			cmdArgs = append(cmdArgs, "-lowDetail")
		}
		if args.MenuManipulation {
			cmdArgs = append(cmdArgs, "-menuManipulation")
		} else {
			cmdArgs = append(cmdArgs, "-disableMenuManipulation")
		}
		if args.NoClickWalk {
			cmdArgs = append(cmdArgs, "-noClickWalk")
		} else {
			cmdArgs = append(cmdArgs, "-disableNoClickWalk")
		}
		if args.DismissRandomEvents {
			cmdArgs = append(cmdArgs, "-dismiss-random-events")
		}

		cmdArgs = append(cmdArgs, "-debug")

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

		portMutex.Lock()
		clientPort := basePort
		basePort++
		portMutex.Unlock()

		if args.AccountTotp != "" {
			cmdArgs = append(cmdArgs, "-remote-debugging-port="+strconv.Itoa(clientPort))
			cmdArgs = append(cmdArgs, "-new-account-browser-login")
			cmdArgs = append(cmdArgs, "-accountTotp", args.AccountTotp)
		}

		if args.DisableBrowserProxy {
			cmdArgs = append(cmdArgs, "-disable-browser-proxy")
		}

		cmdArgs = append(cmdArgs, "-covert")
		cmdArgs = append(cmdArgs, "-params", args.ScriptName)

		if args.ScriptParams != "" {
			cmdArgs = append(cmdArgs, args.ScriptParams)
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("java", cmdArgs...)
		} else {
			cmdArgsWithSetsid := append([]string{"setsid", "java"}, cmdArgs...)
			cmd = exec.Command(cmdArgsWithSetsid[0], cmdArgsWithSetsid[1:]...)
		}

		err := cmd.Start()
		if err != nil {
			fmt.Println("Error starting Cmd", err)
			return
		}

		pid := cmd.Process.Pid

		totp := ""
		if args.AccountTotp != "" {
			totp = args.AccountTotp
		}

		_ = NewClient(pid, args.InternalId, "Starting", args.ScriptName, clientPort, args.AccountUsername, args.AccountPassword, totp)
		log.Println(args.AccountUsername, "has been detected as "+Yellow+"starting"+Reset+".")

		logDir := botbuddyLogDir(args.ScriptsLocation, args.InternalId)

		lines := make(chan string, 2000)
		logCtx, cancelLogs := context.WithCancel(context.Background())
		defer cancelLogs()

		waitCtx, waitCancel := context.WithCancel(context.Background())
		defer waitCancel()

		currentPath, currentMod, err := waitForNewestLogFile(waitCtx, logDir, 5*time.Minute)
		if err != nil {
			log.Println("Error waiting for new log files:", err)
			cancelLogs()
			RemoveClientByInternalId(args.InternalId)
			sendProcessExitNotification(Master, args.InternalId, args.AccountUsername, args.ScriptName)
			return
		}

		tailCtx, tailCancel := context.WithCancel(logCtx)
		go func(p string) {
			_ = tailSpecificFileFromEnd(tailCtx, p, lines)
		}(currentPath)

		lastActivity := time.Now()
		lastSeenMod := currentMod

		inactivityLimit := 60 * time.Second
		poll := time.NewTicker(500 * time.Millisecond)
		defer poll.Stop()

		for {
			select {
			case line := <-lines:
				lastActivity = time.Now()
				//log.Printf("[BOT %d] %s", args.InternalId, line)

				if len(logHandlers) > 0 {
					for _, l := range logHandlers {
						if strings.Contains(strings.ToLower(line), strings.ToLower(l.waitingFor)) && (args.ScriptName == l.scriptName || l.scriptName == "botbuddy_system") {
							err := l.action.execute(Master, args.InternalId, args.AccountUsername, line, args.ScriptName, totp)
							if err != nil {
								go func(line string, totp string) {
									for {
										time.Sleep(time.Second)
										err := l.action.execute(Master, args.InternalId, args.AccountUsername, line, args.ScriptName, totp)
										if err == nil {
											return
										}
									}
								}(line, totp)
								continue
							}
						}
					}
				}

			case <-poll.C:
				newestPath, newestMod, err := latestFileInDir(logDir)
				if err == nil && newestPath != "" {
					if newestPath != currentPath {
						tailCancel()
						currentPath = newestPath
						lastSeenMod = newestMod
						lastActivity = time.Now()

						tailCtx, tailCancel = context.WithCancel(logCtx)
						go func(p string) {
							_ = tailSpecificFileFromEnd(tailCtx, p, lines)
						}(currentPath)
					} else {
						if newestMod.After(lastSeenMod) {
							lastSeenMod = newestMod
							lastActivity = time.Now()
						}
					}
				}

				if time.Since(lastActivity) >= inactivityLimit {
					log.Printf("[BOT %d] stopping due to log inactivity >= %s", args.InternalId, inactivityLimit)
					tailCancel()
					cancelLogs()
					RemoveClientByInternalId(args.InternalId)
					sendProcessExitNotification(Master, args.InternalId, args.AccountUsername, args.ScriptName)
					return
				}
			}
		}
	}(args)

	return nil
}

func sendProcessExitNotification(conn net.Conn, internalId int, loginName string, script string) {
	if conn != nil {
		err := ReportBotStatus{
			online:       false,
			proxyBlocked: false,
		}.execute(conn, internalId, loginName, "", script, "")

		if err != nil {
			log.Println(err)
		}
	}
}

type stopBotData struct {
	InternalId int `json:"internalId"`
}

func stopBot(_ net.Conn, data string) error {
	var args stopBotData
	err := json.Unmarshal([]byte(data), &args)
	if err != nil {
		return err
	}

	safeClients.mux.RLock()
	_, exists := safeClients.clients[args.InternalId]
	safeClients.mux.RUnlock()

	if exists {
		StopBotByInternalId(args.InternalId)
	}

	return nil
}

type linkJagexData struct {
	InternalId int    `json:"internalId"`
	Payload    string `json:"payload"`
}

func linkJagex(_ net.Conn, data string) error {
	var args linkJagexData
	err := json.Unmarshal([]byte(data), &args)
	if err != nil {
		return err
	}

	safeClients.mux.RLock()
	client, exists := safeClients.clients[args.InternalId]
	safeClients.mux.RUnlock()

	if !exists {
		fmt.Println("Client with internalId", args.InternalId, "does not exist")
		return errors.New("client does not exist")
	}

	if client.HandledLogin {
		return nil
	}

	client.HandledLogin = true
	safeClients.mux.Lock()
	safeClients.clients[args.InternalId] = client
	safeClients.mux.Unlock()

	port := client.Port
	email := client.LoginName
	password := client.LoginPass
	totpSecret := client.LoginTotp

	cmdInstallDrissionpage := exec.Command("pip", "install", "DrissionPage==4.1.0.0b2")
	err = cmdInstallDrissionpage.Run()
	if err != nil {
		return err
	}

	cmdInstallPyotp := exec.Command("pip", "install", "pyotp")
	err = cmdInstallPyotp.Run()
	if err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	cmd := exec.Command("python", "-c", args.Payload, "--port", strconv.Itoa(port), "--email", email, "--password", password, "--totp_secret", totpSecret)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			if scanner.Text() == "Proxy blocked by Cloudflare" {
				err = ReportBotStatus{online: false, proxyBlocked: true}.execute(Master, args.InternalId, email, "Proxy blocked by Cloudflare", "", "")
				if err != nil {
					log.Println("1:", err)
					return
				}
			}
		}

		if err = scanner.Err(); err != nil {
			log.Println("2:", err)
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.Println("3:", err)
	}

	err = cmd.Wait()
	if err != nil {
		log.Println("4:", err)
	}

	return nil
}

func linkJagexMailTm(_ net.Conn, data string) error {
	var args linkJagexData
	err := json.Unmarshal([]byte(data), &args)
	if err != nil {
		return err
	}

	safeClients.mux.RLock()
	client, exists := safeClients.clients[args.InternalId]
	safeClients.mux.RUnlock()

	if !exists {
		fmt.Println("Client with internalId", args.InternalId, "does not exist")
		return errors.New("client does not exist")
	}

	if client.HandledLogin {
		return nil
	}

	client.HandledLogin = true
	safeClients.mux.Lock()
	safeClients.clients[args.InternalId] = client
	safeClients.mux.Unlock()

	port := client.Port
	email := client.LoginName
	password := client.LoginPass
	mailTmPass := strings.Split(client.LoginTotp, ":")[1]

	cmdInstallDrissionpage := exec.Command("pip", "install", "DrissionPage==4.1.0.0b2")
	err = cmdInstallDrissionpage.Run()
	if err != nil {
		return err
	}

	cmdInstallPyotp := exec.Command("pip", "install", "pyotp")
	err = cmdInstallPyotp.Run()
	if err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	cmd := exec.Command("python", "-c", args.Payload, "--port", strconv.Itoa(port), "--email", email, "--password", password, "--mail_password", mailTmPass)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			if scanner.Text() == "Proxy blocked by Cloudflare" {
				err = ReportBotStatus{online: false, proxyBlocked: true}.execute(Master, args.InternalId, email, "Proxy blocked by Cloudflare", "", "")
				if err != nil {
					log.Println("1:", err)
					return
				}
			}
		}

		if err = scanner.Err(); err != nil {
			log.Println("2:", err)
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.Println("3:", err)
	}

	err = cmd.Wait()
	if err != nil {
		log.Println("4:", err)
	}

	return nil
}

func dreambotRootFromScriptsLocation(scriptsLocation string) string {
	return filepath.Clean(filepath.Join(scriptsLocation, ".."))
}

func botbuddyLogDir(scriptsLocation string, internalID int) string {
	root := dreambotRootFromScriptsLocation(scriptsLocation)
	return filepath.Join(root, "Logs", "BotBuddy", strconv.Itoa(internalID))
}

func latestFileInDir(dir string) (string, time.Time, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", time.Time{}, err
	}

	var bestPath string
	var bestMod time.Time

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		mod := info.ModTime()
		if bestPath == "" || mod.After(bestMod) {
			bestMod = mod
			bestPath = filepath.Join(dir, e.Name())
		}
	}

	if bestPath == "" {
		return "", time.Time{}, errors.New("no files in log dir")
	}
	return bestPath, bestMod, nil
}

func tailFileFromEnd(ctx context.Context, path string, lines chan<- string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(f)
	buf := make([]byte, 0, 4096)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		b, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				time.Sleep(150 * time.Millisecond)
				continue
			}
			return err
		}

		if b == '\n' {
			if len(buf) > 0 && buf[len(buf)-1] == '\r' {
				buf = buf[:len(buf)-1]
			}
			lines <- string(buf)
			buf = buf[:0]
			continue
		}

		buf = append(buf, b)
	}
}
