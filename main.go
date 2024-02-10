package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
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
	// Combine the header and data with a '\r' separator
	packet := header + "\r" + data

	// Prepare the length field
	length := uint32(len(packet))
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)

	// Send the length field and the packet
	_, err := conn.Write(append(lengthBytes, []byte(packet)...))
	if err != nil {
		log.Fatal(err)
	}
}

func handleData(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		// Read the length field (4 bytes)
		lengthBytes := make([]byte, 4)
		_, err := reader.Read(lengthBytes)
		if err != nil {
			log.Println(err)
			return
		}

		// Convert the length field to int
		length := binary.BigEndian.Uint32(lengthBytes)

		// Read the header until the '\r' separator
		header, err := reader.ReadString('\r')
		if err != nil {
			log.Println(err)
			return
		}

		// Calculate the length of the data
		dataLength := length - uint32(len(header))

		// Read the data
		dataBytes := make([]byte, dataLength)
		_, err = reader.Read(dataBytes)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println("Received header: " + header)
		fmt.Println("Received data: " + string(dataBytes))
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

	go handleData(conn)

	sendPacket(conn, "initHandshake", "123")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	conn.Close()
}
