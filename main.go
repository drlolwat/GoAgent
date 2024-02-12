package main

import (
	"bufio"
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
	WRAPPER_JAR = "BotBuddyWrapper-1.0-SNAPSHOT-dep-included.jar"
	DIST_URL    = "https://botbuddy.net/dist"
)

func handleData(conn net.Conn) {
	reader := bufio.NewReader(conn)

	for {
		var packet *Packet
		var err error
		if CUSTOMER_ID != nil {
			packet, err = parseEncryptedPacket(reader)
		} else {
			packet, err = parsePacket(reader)
		}
		if err != nil {
			log.Println(err)
			return
		}

		header, data := packet.Header, packet.Data

		if _, ok := handlers[header]; !ok {
			log.Printf("Unknown packet: Header: \"%s\", Data: \"%s\"\n", header, data)
			continue
		}

		err = handlers[header](conn, data)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func main() {
	fmt.Println(Blue + "    ____        __  ____            __    __     ")
	fmt.Println("   / __ )____  / /_/ __ )__  ______/ /___/ /_  __")
	fmt.Println("  / __  / __ \\/ __/ __  / / / / __  / __  / / / /")
	fmt.Println(" / /_/ / /_/ / /_/ /_/ / /_/ / /_/ / /_/ / /_/ / ")
	fmt.Println("/_____/\\____/\\__/_____/\\__,_/\\__,_/\\__,_/\\__, /  ")
	fmt.Println("                                        /____/   " + Reset)
	fmt.Println()
	fmt.Println("Welcome to BotBuddy version 3.")
	fmt.Println("Developed by the team at https://botbuddy.net")
	fmt.Println()

	conn, err := net.Dial("tcp", "bbaas.botbuddy.net:7888")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Initializing connection to BotBuddy...")
	go handleData(conn)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"
)
