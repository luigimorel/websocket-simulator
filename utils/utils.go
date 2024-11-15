package utils

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type Packet struct {
	SequenceNumber int    `json:"seq"`
	Data           string `json:"data"`
}

// Global channel to store received packets
var receivedPackets = make(chan Packet, 100)
var receivedPacketsMutex sync.Mutex
var allPackets []Packet

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func DisplayReconstructedMessage() {
	receivedPacketsMutex.Lock()
	defer receivedPacketsMutex.Unlock()

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
}

func CreatePackets(content string) []Packet {
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
			Data:           chunk,
		})
		sequenceNumber++
	}
	return packets
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade:", err)
		return
	}
	defer ws.Close()

	for {
		var packet Packet
		err := ws.ReadJSON(&packet)
		if err != nil {
			log.Println("Read:", err)
			break
		}
		receivedPackets <- packet
	}
}

func ReceivePackets() {
	for packet := range receivedPackets {
		receivedPacketsMutex.Lock()
		allPackets = append(allPackets, packet)
		receivedPacketsMutex.Unlock()
		fmt.Printf("Received packet %d: '%s'\n", packet.SequenceNumber, packet.Data)
	}
}
