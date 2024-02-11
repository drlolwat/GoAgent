package main

import (
	"encoding/json"
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
