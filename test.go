package main

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func sendUDP(host string, port int, data []byte) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(data)
	if err != nil {
		return err
	}
	return nil
}

// Note: Run the main UDP server before running tests
// Terminal 1: go run main.go
// Terminal 2: go test -v

func TestSendSimpleText(t *testing.T) {
	data := []byte("Hello World")
	if err := sendUDP("localhost", 8080, data); err != nil {
		t.Fatalf("Failed to send simple text: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
}

func TestSendBinaryData(t *testing.T) {
	binaryData := make([]byte, 256)
	for i := 0; i < 256; i++ {
		binaryData[i] = byte(i)
	}

	if err := sendUDP("localhost", 8080, binaryData); err != nil {
		t.Fatalf("Failed to send binary data: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
}

func TestSendLargeData(t *testing.T) {
	largeData := make([]byte, 2048)
	for i := 0; i < 1024; i++ {
		largeData[i] = 'A'
	}
	for i := 1024; i < 2048; i++ {
		largeData[i] = 'B'
	}

	if err := sendUDP("localhost", 8080, largeData); err != nil {
		t.Fatalf("Failed to send large data: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
}

func TestSendMixedContent(t *testing.T) {
	mixedData := []byte("START\x00\x01\x02\x03\xff\xfe\xfdEND")

	if err := sendUDP("localhost", 8080, mixedData); err != nil {
		t.Fatalf("Failed to send mixed content: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
}

func TestSendJSONData(t *testing.T) {
	jsonData := []byte(`{"name":"test","value":12345,"data":[1,2,3,4,5]}`)

	if err := sendUDP("localhost", 8080, jsonData); err != nil {
		t.Fatalf("Failed to send JSON data: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
}

func TestSendVeryLargeData(t *testing.T) {
	veryLargeData := make([]byte, 5000)
	for i := 0; i < len(veryLargeData); i++ {
		veryLargeData[i] = byte(i % 256)
	}

	if err := sendUDP("localhost", 8080, veryLargeData); err != nil {
		t.Fatalf("Failed to send very large data: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
}

func TestSendMultiplePackets(t *testing.T) {
	for i := 1; i <= 10; i++ {
		data := []byte(fmt.Sprintf("Packet #%d", i))
		if err := sendUDP("localhost", 8080, data); err != nil {
			t.Fatalf("Failed to send packet #%d: %v", i, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
