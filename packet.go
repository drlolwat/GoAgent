package main

import (
	"bufio"
	"crypto/aes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"strings"
)

type Packet struct {
	Header string
	Data   string
}

func parsePacket(reader *bufio.Reader) (*Packet, error) {
	lengthBytes := make([]byte, 4)
	_, err := reader.Read(lengthBytes)
	if err != nil {
		if err == io.EOF {
			return nil, errors.New("connection closed by peer")
		}
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBytes)

	header, err := reader.ReadString('\r')
	if err != nil {
		return nil, err
	}
	header = strings.Trim(header, "\r")

	dataLength := length - uint32(len(header))
	dataBytes := make([]byte, dataLength)
	_, err = reader.Read(dataBytes)
	if err != nil {
		return nil, err
	}

	data := strings.Trim(string(dataBytes), "\x00")

	return &Packet{Header: header, Data: data}, nil
}

func parseEncryptedPacket(reader *bufio.Reader) (*Packet, error) {
	_, err := reader.Discard(4)

	encoded, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	// encrypted packets are delimited by a newline
	encoded = strings.Trim(encoded, "\n")

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	key, err := generateKey(CLIENT_KEY)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	decrypter := NewECBDecrypter(block)
	decryptedText := make([]byte, len(decoded))
	decrypter.CryptBlocks(decryptedText, decoded)

	finalText := pKCS5Unpadding(decryptedText)

	parts := strings.SplitN(string(finalText), "\r", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid packet")
	}
	header, data := parts[0], parts[1]
	data = strings.Trim(data, "\x00")

	return &Packet{Header: header, Data: data}, nil
}

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
