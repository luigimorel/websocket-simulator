package utils

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/gorilla/websocket"
)

type Packet struct {
	Seq  int    `json:"seq"`
	Data string `json:"data"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func RunReceiver() {
	port := ReadInput("Enter port to listen on (default 8080): ")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/ws", handleWebSocket)
	addr := ":" + port
	fmt.Printf("\nReceiver listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func RunSender() {
	ip := ReadInput("Enter receiver's IP address: ")
	port := ReadInput("Enter receiver's port (default 8080): ")
	if port == "" {
		port = "8080"
	}

	filepath := ReadInput("Enter the path to the file containing the message: ")
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatal("Error reading file:", err)
	}

	wsURL := fmt.Sprintf("ws://%s:%s/ws", ip, port)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatal("Connection failed:", err)
	}
	defer conn.Close()

	packets := createPackets(string(content))
	fmt.Printf("\nSending %d packets...\n\n", len(packets))

	for i, packet := range packets {
		if err := conn.WriteJSON(packet); err != nil {
			log.Printf("Failed to send packet %d: %v\n", i, err)
			continue
		}
		// Display sent packet
		fmt.Printf("Sent     [%4d]: %q\n", packet.Seq, packet.Data)
	}

	conn.WriteJSON(Packet{Seq: -1, Data: "EOT"})
	fmt.Println("\nTransmission complete")
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	var packets []Packet
	fmt.Println("\nConnected to sender. Receiving data...")

	packetsReceived := 0

	for {
		var packet Packet
		if err := conn.ReadJSON(&packet); err != nil {
			log.Println("Read error:", err)
			break
		}

		if packet.Seq == -1 && packet.Data == "EOT" {
			fmt.Printf("\nReceived end of transmission\n")
			fmt.Printf("\nTotal packets received: %d\n", packetsReceived)
			fmt.Println("\nReconstructed message:")
			displayMessage(packets)
			break
		}

		// Display received packet
		fmt.Printf("Received [%4d]: %q\n", packet.Seq, packet.Data)
		packets = append(packets, packet)
		packetsReceived++
	}
}

func createPackets(content string) []Packet {
	var packets []Packet
	chunkSize := 5

	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		packets = append(packets, Packet{
			Seq:  i / chunkSize,
			Data: content[i:end],
		})
	}
	return packets
}

func displayMessage(packets []Packet) {
	sort.Slice(packets, func(i, j int) bool {
		return packets[i].Seq < packets[j].Seq
	})

	var message strings.Builder
	for _, p := range packets {
		message.WriteString(p.Data)
	}

	fmt.Printf("%s\n", message.String())
}

func ReadInput(prompt string) string {
	if prompt != "" {
		fmt.Print(prompt)
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}
