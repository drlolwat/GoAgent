package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
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

	// todo: start the client jar with the given args

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
