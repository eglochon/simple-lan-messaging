package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eglochon/simple-lan-messaging/pkg/discovery"
	"github.com/eglochon/simple-lan-messaging/pkg/identity"
)

type Message struct {
	ID string `json:"id"`
}

func handleMessage(data []byte, addr *net.UDPAddr) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err == nil {
		fmt.Printf("[DISCOVERED] ID: %s from %s\n", msg.ID, addr.IP)
	} else {
		fmt.Printf("[INVALID MESSAGE] from %s: %s\n", addr.IP, string(data))
	}
}

func main() {
	path := identity.DefaultPath()
	id, err := identity.GetOrCreate(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load or generate identity: %v\n", err)
		os.Exit(1)
	}

	myID := id.GetID()
	msgStruct := Message{ID: myID}
	msgBytes, err := json.Marshal(msgStruct)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode message: %v\n", err)
		os.Exit(1)
	}

	service := discovery.NewDiscoveryService(msgBytes, 3*time.Second, handleMessage)
	service.Start()

	fmt.Printf("Client ID: %s\n", myID)
	fmt.Println("Discovery started. Press Ctrl+C to stop.")

	// Wait for interrupt signal to gracefully stop the service
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down discovery service.")
	service.Stop()
}
