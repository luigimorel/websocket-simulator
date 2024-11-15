package main

import (
	"assignment/utils"
	"fmt"
	"log"
	"strings"
)

func main() {
	fmt.Print("Enter mode (sender/receiver): ")
	mode := utils.ReadInput("")

	switch strings.ToLower(mode) {
	case "sender":
		utils.RunSender()
	case "receiver":
		utils.RunReceiver()
	default:
		log.Fatal("Invalid mode. Use 'sender' or 'receiver'")
	}
}
