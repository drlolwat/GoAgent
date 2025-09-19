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
	CLIENT_UUID  = "CLIENT_UUID_HERE"
	CLIENT_KEY   = "CLIENT_KEY_HERE"
	MASTER_HOST  = "MASTER_HOST_HERE"
	CUSTOMER_ID  *int
	WRAPPER_JAR  = "BotBuddyWrapper-2.0-dist.jar"
	DIST_URL     = "https://dist.botbuddy.net/"
	AGENT_VER    = "0.2"
	Master       net.Conn
	KeepRetrying = true
)

func handleData() {
	reader := bufio.NewReader(Master)

	defer func() {
		time.Sleep(time.Second)
		for {
			if !KeepRetrying {
				time.Sleep(120 * time.Second)
				return
			}

			if r := recover(); r != nil {
				if netErr, ok := r.(*net.OpError); ok {
					if netErr.Err == syscall.EPIPE {
						log.Println(Red + "Connection to BotBuddy has been lost due to a broken pipe. Reconnecting..." + Reset)
					} else {
						log.Println(Red + "Connection to BotBuddy has been lost. Reconnecting..." + Reset)
					}
				} else {
					log.Println(Red + "An error occurred. Reconnecting..." + Reset)
				}

				var err error
				Master, err = reconnect()
				if err != nil {
				} else {
					go handleData()
					return
				}
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
		}

		header, data := packet.Header, packet.Data

		if _, ok := handlers[header]; !ok {
			log.Printf("Unknown packet: Header: \"%s\", Data: \"%s\"\n", header, data)
			continue
		}

		go func() {
			err = handlers[header](Master, data)
			if err != nil {
				log.Println(err)
			}
		}()
	}
}

func reconnect() (net.Conn, error) {
	var err error

	CUSTOMER_ID = nil

	for {
		Master, err = net.Dial("tcp", MASTER_HOST)
		if err == nil {
			return Master, nil
		}
		time.Sleep(time.Second * 1)
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
	fmt.Println("Welcome to BotBuddy version 3." + AGENT_VER + ".")
	fmt.Println("Developed by the team at https://botbuddy.net")
	fmt.Println()

	_, err := reconnect()
	if err != nil {
		//log.Fatal(err)
	}

	log.Println("Initializing connection to BotBuddy...")
	go handleData()

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
