package main

import (
	"assignment/utils"
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ipAddress := os.Getenv("IP_ADDRESS")
	port := os.Getenv("PORT")
	if ipAddress == "" || port == "" {
		log.Fatal("IP_ADDRESS or PORT environment variables not set")
	}

	http.HandleFunc("/ws", utils.HandleConnections)

	go func() {
		log.Println("Starting server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	fmt.Println("Enter the path to the file containing the message:")
	reader := bufio.NewReader(os.Stdin)
	filepath, _ := reader.ReadString('\n')
	filepath = strings.TrimSpace(filepath)

	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatal("Error reading file:", err)
	}

	packets := utils.CreatePackets(string(content))

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(packets), func(i, j int) {
		packets[i], packets[j] = packets[j], packets[i]
	})

	url := fmt.Sprintf("ws://%s:%s/ws", ipAddress, port)

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("Dial:", err)
	}
	defer c.Close()

	go utils.ReceivePackets()

	for _, packet := range packets {
		err := c.WriteJSON(packet)
		if err != nil {
			log.Println("Error sending packet:", err)
			continue
		}
		fmt.Printf("Sent packet %d: '%s'\n", packet.SequenceNumber, packet.Data)
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)

	utils.DisplayReconstructedMessage()
}
