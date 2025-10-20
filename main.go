package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 3000, "UDP port to listen on")
	logFile := flag.String("file", "", "Log file path (optional, if not set, only console output)")
	consoleOutput := flag.Bool("console", true, "Enable console output")
	flag.Parse()

	// Setup UDP address
	addr := fmt.Sprintf(":%d", *port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	// Create and bind UDP socket
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP port %d: %v", *port, err)
	}
	defer conn.Close()

	log.Printf("UDP server started on port %d", *port)

	// Setup file logging if specified
	var file *os.File
	if *logFile != "" {
		file, err = os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer file.Close()
		log.Printf("Logging to file: %s", *logFile)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start packet receiving goroutine
	packetChan := make(chan packetData, 100)
	go receivePackets(conn, packetChan)

	// Packet counter and sender tracking
	var packetCount int
	senderStats := make(map[string]*senderInfo)

	// Main loop
	for {
		select {
		case <-sigChan:
			log.Printf("\nReceived shutdown signal. Total packets received: %d", packetCount)
			printSenderSummary(senderStats)
			return

		case pkt := <-packetChan:
			packetCount++
			timestamp := time.Now().Format("2006-01-02 15:04:05.000")
			senderKey := pkt.addr.IP.String()

			// Update sender statistics
			if _, exists := senderStats[senderKey]; !exists {
				senderStats[senderKey] = &senderInfo{
					ip:          senderKey,
					firstSeen:   time.Now(),
					packetCount: 0,
					totalBytes:  0,
				}
			}
			senderStats[senderKey].packetCount++
			senderStats[senderKey].totalBytes += pkt.length
			senderStats[senderKey].lastSeen = time.Now()

			// Format message with clear sender identification
			separator := "=" + fmt.Sprintf("%s", repeatString("=", 78))
			message := fmt.Sprintf("\n%s\n", separator)
			message += fmt.Sprintf("📦 Packet #%d | Timestamp: %s\n", packetCount, timestamp)
			message += fmt.Sprintf("📍 Sender: %s (Port: %d)\n", senderKey, pkt.addr.Port)
			message += fmt.Sprintf("📊 Size: %d bytes | Total from this sender: %d packets, %d bytes\n",
				pkt.length, senderStats[senderKey].packetCount, senderStats[senderKey].totalBytes)
			message += fmt.Sprintf("%s\n", separator)
			message += formatHexDump(pkt.data)

			// Console output
			if *consoleOutput {
				fmt.Print(message)
			}

			// File logging
			if file != nil {
				if _, err := file.WriteString(message); err != nil {
					log.Printf("Failed to write to file: %v", err)
				}
			}
		}
	}
}

type packetData struct {
	data   []byte
	length int
	addr   *net.UDPAddr
}

type senderInfo struct {
	ip          string
	firstSeen   time.Time
	lastSeen    time.Time
	packetCount int
	totalBytes  int
}

func receivePackets(conn *net.UDPConn, packetChan chan<- packetData) {
	buffer := make([]byte, 65535) // Maximum UDP packet size

	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP packet: %v", err)
			continue
		}

		// Copy data to avoid buffer reuse issues
		data := make([]byte, n)
		copy(data, buffer[:n])

		packetChan <- packetData{
			data:   data,
			length: n,
			addr:   addr,
		}
	}
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func printSenderSummary(senderStats map[string]*senderInfo) {
	if len(senderStats) == 0 {
		return
	}

	fmt.Println("\n" + repeatString("=", 80))
	fmt.Println("📊 SENDER SUMMARY")
	fmt.Println(repeatString("=", 80))

	for ip, info := range senderStats {
		duration := info.lastSeen.Sub(info.firstSeen)
		fmt.Printf("\n📍 Sender: %s\n", ip)
		fmt.Printf("   First seen: %s\n", info.firstSeen.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Last seen:  %s\n", info.lastSeen.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Duration:   %s\n", duration.Round(time.Millisecond))
		fmt.Printf("   Packets:    %d\n", info.packetCount)
		fmt.Printf("   Total bytes: %d\n", info.totalBytes)
		if info.packetCount > 0 {
			fmt.Printf("   Avg packet size: %d bytes\n", info.totalBytes/info.packetCount)
		}
	}
	fmt.Println(repeatString("=", 80))
}

// formatHexDump formats bytes in hexdump style (16 bytes per line)
// Example output:
// 0000: 48 65 6c 6c 6f 20 57 6f 72 6c 64 0a 54 65 73 74  |Hello World.Test|
// 0010: 20 64 61 74 61                                   | data|
func formatHexDump(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var result string
	const bytesPerLine = 16

	for i := 0; i < len(data); i += bytesPerLine {
		// Offset
		result += fmt.Sprintf("%04x: ", i)

		// Hex bytes
		lineEnd := i + bytesPerLine
		if lineEnd > len(data) {
			lineEnd = len(data)
		}

		for j := i; j < i+bytesPerLine; j++ {
			if j < len(data) {
				result += fmt.Sprintf("%02x ", data[j])
			} else {
				result += "   " // padding for incomplete line
			}

			// Add extra space in the middle for readability
			if j == i+7 {
				result += " "
			}
		}

		// ASCII representation
		result += " |"
		for j := i; j < lineEnd; j++ {
			if data[j] >= 32 && data[j] <= 126 {
				result += string(data[j])
			} else {
				result += "."
			}
		}
		result += "|\n"
	}

	return result
}
