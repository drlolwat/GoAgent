package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	CLIENT_UUID = "CLIENT_UUID_HERE"
	CLIENT_KEY  = "CLIENT_KEY_HERE"
	CUSTOMER_ID *int
	WRAPPER_JAR = "BotBuddyWrapper-1.0-SNAPSHOT-dep-included.jar"
	DIST_URL    = "https://dist.botbuddy.net/"
)

func handleData(conn net.Conn) {
	reader := bufio.NewReader(conn)

	defer func() {
		for {
			if r := recover(); r != nil {
				CUSTOMER_ID = new(int)
				log.Println(Red + "Connection to BotBuddy has been lost. Reconnecting..." + Reset)
				var err error
				conn, err = reconnect()
				if err != nil {
					time.Sleep(5 * time.Second)
					continue
				} else {
					log.Println(Green + "Reconnected to BotBuddy network." + Reset)
				}

				handleData(conn)
			}
		}
	}()

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
			panic(err)
		}

		header, data := packet.Header, packet.Data

		if _, ok := handlers[header]; !ok {
			log.Printf("Unknown packet: Header: \"%s\", Data: \"%s\"\n", header, data)
			continue
		}

		err = handlers[header](conn, data)
		if err != nil {
			log.Println(err)
			panic(err)
		}
	}
}

func reconnect() (net.Conn, error) {
	var conn net.Conn
	var err error

	for {
		conn, err = net.Dial("tcp", "bbaas.botbuddy.net:7888")
		if err == nil {
			return conn, nil
		}

		time.Sleep(5 * time.Second)
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

	conn, err := reconnect()
	if err != nil {
		//log.Fatal(err)
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
