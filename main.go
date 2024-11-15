package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

// Packet represents a chunk of data with sequence number
type Packet struct {
	SequenceNumber int
	Data           string
}

// readFile reads the content of the specified file
func readFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content.WriteString(scanner.Text())
	}

	return content.String(), scanner.Err()
}

// createPackets splits content into packets of specified size
func createPackets(content string, packetSize int) []Packet {
	var packets []Packet
	sequenceNumber := 0

	for i := 0; i < len(content); i += packetSize {
		end := i + packetSize
		if end > len(content) {
			end = len(content)
		}

		packet := Packet{
			SequenceNumber: sequenceNumber,
			Data:           content[i:end],
		}
		packets = append(packets, packet)
		sequenceNumber++
	}

	return packets
}

// shufflePackets randomly reorders the packets to simulate network transmission
func shufflePackets(packets []Packet) []Packet {
	shuffled := make([]Packet, len(packets))
	copy(shuffled, packets)

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled
}

func simulateTransmission(packets []Packet) {
	fmt.Println("\nTransmitting packets in random order:")
	for _, packet := range packets {
		fmt.Printf("Sending packet %d: '%s'\n", packet.SequenceNumber, packet.Data)
		time.Sleep(100 * time.Millisecond)
	}
}

// receiveAndReconstructMessage simulates receiving and reconstructing the message
func receiveAndReconstructMessage(packets []Packet) string {
	sorted := make([]Packet, len(packets))
	copy(sorted, packets)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].SequenceNumber > sorted[j+1].SequenceNumber {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	// Reconstruct message
	var reconstructed strings.Builder
	for _, packet := range sorted {
		reconstructed.WriteString(packet.Data)
	}

	return reconstructed.String()
}

func main() {
	fmt.Print("Enter the filename containing the message to send: ")
	var filename string
	fmt.Scanln(&filename)

	content, err := readFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	packets := createPackets(content, 1)

	fmt.Println("\nOriginal message broken into packets:")
	for _, packet := range packets {
		fmt.Printf("Packet %d: '%s'\n", packet.SequenceNumber, packet.Data)
	}

	shuffledPackets := shufflePackets(packets)
	simulateTransmission(shuffledPackets)

	fmt.Println("\nReceiving and reconstructing message...")
	reconstructedMessage := receiveAndReconstructMessage(shuffledPackets)

	fmt.Printf("Reconstructed message: %s\n", reconstructedMessage)
}
