package main

import (
	"log"
	"net"
	"strconv"
)

type handlerMap map[string]func(net.Conn, string) error

var handlers handlerMap

func init() {
	handlers = handlerMap{
		"initHandshake": initHandshake,
		"handshakeOk":   handshakeOk,
		"ping":          ping,
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
