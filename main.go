package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Packet struct {
	SequenceNumber int    `json:"seq"`
	Data           string `json:"data"`
}

type Mode int

const (
	Sender Mode = iota
	Receiver
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow any origin for mobile devices
		},
		EnableCompression: true, // Enable compression for better mobile performance
	}
	receivedPackets      = make(chan Packet, 100)
	receivedPacketsMutex sync.Mutex
	allPackets           []Packet
	mode                 Mode
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	modeFlag := flag.String("mode", "", "Operation mode: 'sender' or 'receiver'")
	flag.Parse()

	switch strings.ToLower(*modeFlag) {
	case "sender":
		mode = Sender
		runSender()
	case "receiver":
		mode = Receiver
		runReceiver()
	default:
		log.Fatal("Invalid mode. Use 'sender' or 'receiver'")
	}
}

func runReceiver() {
	reader := bufio.NewReader(os.Stdin)

	// Get available network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nAvailable network interfaces:")
	for i, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					fmt.Printf("[%d] %s (%s)\n", i, iface.Name, ipnet.IP.String())
				}
			}
		}
	}

	fmt.Print("\nEnter port to listen on (default 8080): ")
	port, _ := reader.ReadString('\n')
	port = strings.TrimSpace(port)
	if port == "" {
		port = "8080"
	}

	// Create a custom server with timeout configurations
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "WebSocket server running. Connect to ws://[IP]:%s/ws", port)
	})

	fmt.Printf("\nReceiver starting on port %s\n", port)
	fmt.Println("\nAvailable IP addresses for sender:")
	printIPAddresses()
	fmt.Println("\nWaiting for connections...")

	log.Fatal(server.ListenAndServe())
}

func printIPAddresses() {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Println("Error getting interfaces:", err)
		return
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					fmt.Printf("Interface: %-10s IP: %s\n", iface.Name, ipnet.IP.String())
				}
			}
		}
	}
}

func runSender() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter receiver's IP address: ")
	ipAddress, _ := reader.ReadString('\n')
	ipAddress = strings.TrimSpace(ipAddress)

	fmt.Print("Enter receiver's port (default 8080): ")
	port, _ := reader.ReadString('\n')
	port = strings.TrimSpace(port)
	if port == "" {
		port = "8080"
	}

	// Test connection first
	testURL := fmt.Sprintf("http://%s:%s", ipAddress, port)
	fmt.Printf("Testing connection to %s...\n", testURL)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(testURL)
	if err != nil {
		log.Printf("Warning: Could not connect to receiver: %v\n", err)
		fmt.Print("Continue anyway? (y/n): ")
		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			return
		}
	} else {
		resp.Body.Close()
	}

	fmt.Print("Enter the path to the file containing the message: ")
	filepath, _ := reader.ReadString('\n')
	filepath = strings.TrimSpace(filepath)

	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatal("Error reading file:", err)
	}

	packets := createPackets(string(content))
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(packets), func(i, j int) {
		packets[i], packets[j] = packets[j], packets[i]
	})

	wsURL := fmt.Sprintf("ws://%s:%s/ws", ipAddress, port)
	fmt.Printf("Connecting to %s...\n", wsURL)

	dialer := websocket.Dialer{
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: true,
	}

	c, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			log.Printf("HTTP Response Status: %s\n", resp.Status)
		}
		log.Fatal("Connection failed:", err)
	}
	defer c.Close()

	fmt.Println("Connected to receiver. Starting transmission...")

	for i, packet := range packets {
		err := c.WriteJSON(packet)
		if err != nil {
			log.Printf("Failed to send packet %d: %v\n", packet.SequenceNumber, err)
			continue
		}
		fmt.Printf("\rProgress: %d/%d packets sent", i+1, len(packets))
	}

	c.WriteJSON(Packet{SequenceNumber: -1, Data: "EOT"})
	fmt.Println("\nTransmission complete.")
}

func createPackets(content string) []Packet {
	var packets []Packet
	sequenceNumber := 0

	for i := 0; i < len(content); i += 5 {
		end := i + 5
		if end > len(content) {
			end = len(content)
		}
		chunk := content[i:end]
		packets = append(packets, Packet{
			SequenceNumber: sequenceNumber,
			Data:           strings.TrimSpace(chunk),
		})
		sequenceNumber++
	}
	return packets
}

func displayReconstructedMessage() {
	receivedPacketsMutex.Lock()
	defer receivedPacketsMutex.Unlock()

	if len(allPackets) == 0 {
		fmt.Println("\nNo packets received!")
		return
	}

	// Sort packets by sequence number
	sort.Slice(allPackets, func(i, j int) bool {
		return allPackets[i].SequenceNumber < allPackets[j].SequenceNumber
	})

	// Reconstruct message
	var reconstructedMessage strings.Builder
	for _, packet := range allPackets {
		reconstructedMessage.WriteString(packet.Data)
	}

	fmt.Println("\nReconstructed message:")
	fmt.Println(reconstructedMessage.String())
	fmt.Printf("Total packets received: %d\n", len(allPackets))
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade failed:", err)
		return
	}
	defer ws.Close()

	fmt.Printf("New connection from %s\n", ws.RemoteAddr().String())

	// Start a goroutine to handle displaying the reconstruction progress
	doneChan := make(chan bool)
	go func() {
		for {
			select {
			case <-doneChan:
				return
			default:
				time.Sleep(500 * time.Millisecond)
				receivedPacketsMutex.Lock()
				if len(allPackets) > 0 {
					fmt.Printf("\rReceived %d packets", len(allPackets))
				}
				receivedPacketsMutex.Unlock()
			}
		}
	}()

	for {
		var packet Packet
		err := ws.ReadJSON(&packet)
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		if packet.SequenceNumber == -1 && packet.Data == "EOT" {
			fmt.Println("\nReceived end of transmission")
			doneChan <- true
			displayReconstructedMessage()
			break
		}

		receivedPacketsMutex.Lock()
		allPackets = append(allPackets, packet)
		receivedPacketsMutex.Unlock()
	}
}
