package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/eglochon/simple-lan-messaging/pkg/discovery"
)

type Message struct {
	ID string `json:"id"`
}

func handleMessage(data []byte, addr *net.UDPAddr) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err == nil {
		fmt.Printf("[DISCOVERED] ID: %s from %s\n", msg.ID, addr.IP)
	} else {
		fmt.Printf("[BAD MESSAGE] from %s: %s\n", addr.IP, string(data))
	}
}

func main() {
	myID := fmt.Sprintf("client-%d", time.Now().Unix())
	msgStruct := Message{ID: myID}
	msgBytes, _ := json.Marshal(msgStruct)

	service := discovery.NewDiscoveryService(msgBytes, 3*time.Second, handleMessage)
	service.Start()

	fmt.Println("Discovery started. Press Ctrl+C to stop.")
	select {} // block forever
}
