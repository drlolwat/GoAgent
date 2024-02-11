package main

import (
	"bufio"
	"crypto/aes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
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

func parsePacket(reader *bufio.Reader) (string, string, error) {
	lengthBytes := make([]byte, 4)
	_, err := reader.Read(lengthBytes)
	if err != nil {
		if err == io.EOF {
			return "", "", errors.New("connection closed by peer")
		}
		return "", "", err
	}

	length := binary.BigEndian.Uint32(lengthBytes)

	header, err := reader.ReadString('\r')
	if err != nil {
		return "", "", err
	}
	header = strings.Trim(header, "\r")

	dataLength := length - uint32(len(header))
	dataBytes := make([]byte, dataLength)
	_, err = reader.Read(dataBytes)
	if err != nil {
		return "", "", err
	}

	data := strings.Trim(string(dataBytes), "\x00")

	return header, data, nil
}

func parseEncryptedPacket(reader *bufio.Reader) (string, string, error) {
	lengthBytes := make([]byte, 4)
	_, err := reader.Read(lengthBytes)
	if err != nil {
		if err == io.EOF {
			return "", "", errors.New("connection closed by peer")
		}
		return "", "", err
	}

	encoded, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	encoded = strings.Trim(encoded, "\n")

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", err
	}

	key, err := generateKey(CLIENT_KEY)
	if err != nil {
		return "", "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}

	decrypter := NewECBDecrypter(block)
	decryptedText := make([]byte, len(decoded))
	decrypter.CryptBlocks(decryptedText, decoded)

	finalText := pKCS5Unpadding(decryptedText)

	header, data := splitPacketData(string(finalText))

	data = strings.Trim(data, "\x00")

	return header, data, nil
}

func splitPacketData(data string) (string, string) {
	parts := strings.SplitN(data, "\r", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func handleData(conn net.Conn) {
	reader := bufio.NewReader(conn)

	for {
		var header, data string
		var err error
		if CUSTOMER_ID != nil {
			header, data, err = parseEncryptedPacket(reader)
		} else {
			header, data, err = parsePacket(reader)
		}
		if err != nil {
			log.Println(err)
			return
		}

		if _, ok := handlers[header]; !ok {
			log.Printf("Unknown packet: Header: \"%s\", Data: \"%s\"\n", header, data)
			continue
		}

		log.Printf("Header: \"%s\", Data: \"%s\"\n", header, data)

		err = handlers[header](conn, data)
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
