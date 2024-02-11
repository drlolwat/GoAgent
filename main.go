package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

var (
	CLIENT_UUID = "e8e32662-15f2-4070-9556-1e47fca35f37"
	CLIENT_KEY  = "0150024a2f03ff4e1d2b18b3867810b43e27f2098b688cd268807091bada987b"
	CUSTOMER_ID *int
	DEV_MODE    = true
	WRAPPER_JAR = "BotBuddyWrapper-1.0-SNAPSHOT-dep-included.jar"
	DIST_URL    = "https://botbuddy.net/dist"
)

func sendPacket(conn net.Conn, header string, data string) {
	packet := header + "\r" + data

	length := uint32(len(packet))
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)

	_, err := conn.Write(append(lengthBytes, []byte(packet)...))
	if err != nil {
		log.Fatal(err)
	}
}

func parseHeader(reader *bufio.Reader, conn net.Conn) error {
	lengthBytes := make([]byte, 4)
	_, err := reader.Read(lengthBytes)
	if err != nil {
		if err == io.EOF {
			return errors.New("connection closed by peer")
		}
		return err
	}

	length := binary.BigEndian.Uint32(lengthBytes)

	header, err := reader.ReadString('\r')
	if err != nil {
		return err
	}
	header = strings.Trim(header, "\r")

	dataLength := length - uint32(len(header))
	dataBytes := make([]byte, dataLength)
	_, err = reader.Read(dataBytes)
	if err != nil {
		return err
	}

	data := strings.Trim(string(dataBytes), "\x00")

	switch header {
	case "initHandshake":
		sendPacket(conn, "initHandshake", fmt.Sprintf("{\"machineId\":\"%s\"}", CLIENT_UUID))
		break
	case "handshakeOk":
		customerId, err := strconv.Atoi(data)
		if err != nil {
			return err
		}
		CUSTOMER_ID = &customerId
		break
	case "ping":
		log.Println("Heartbeat received")
		break
	default:
		log.Printf("Unknown packet: Header: \"%s\", Data: \"%s\"\n", header, data)
		break
	}

	return nil
}

func handleData(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		if CUSTOMER_ID != nil {
			// todo: create new reader with decrypted contents
		}
		err := parseHeader(reader, conn)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func main() {
	fmt.Println("    ____        __  ____            __    __     ")
	fmt.Println("   / __ )____  / /_/ __ )__  ______/ /___/ /_  __")
	fmt.Println("  / __  / __ \\/ __/ __  / / / / __  / __  / / / /")
	fmt.Println(" / /_/ / /_/ / /_/ /_/ / /_/ / /_/ / /_/ / /_/ / ")
	fmt.Println("/_____/\\____/\\__/_____/\\__,_/\\__,_/\\__,_/\\__, /  ")
	fmt.Println("                                        /____/   ")
	fmt.Println()
	fmt.Println("Welcome to BotBuddy version 3.")
	fmt.Println("Developed by the team at https://botbuddy.net")
	fmt.Println()

	conn, err := net.Dial("tcp", "127.0.0.1:7888")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	go handleData(conn)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
