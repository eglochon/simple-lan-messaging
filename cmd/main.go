package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eglochon/simple-lan-messaging/config"
	"github.com/eglochon/simple-lan-messaging/models"
	"github.com/eglochon/simple-lan-messaging/pkg/comms"
	"github.com/eglochon/simple-lan-messaging/pkg/discovery"
	"github.com/eglochon/simple-lan-messaging/pkg/identity"
	"google.golang.org/protobuf/proto"
)

func main() {
	config.Setup()

	// Self Adress
	selfAddr, err := discovery.NewSelfAddress()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get self address: %v\n", err)
		os.Exit(1)
	}

	// Load or generate identity
	idPath := identity.DefaultPath()
	id, err := identity.GetOrCreateIdentity(idPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load/generate identity: %v\n", err)
		os.Exit(1)
	}

	// Build discovery message
	myID := id.GetID()
	discMsg := &models.DiscoveryMessage{
		Id:   base64.RawURLEncoding.EncodeToString(id.SigningPublicKey),
		Enc:  base64.RawURLEncoding.EncodeToString(id.EncryptPublicKey[:]),
		Name: selfAddr.Hostname,
		Ip:   selfAddr.IP,
		Port: uint32(config.SERVICE_PORT),
	}

	msgBytes, err := proto.Marshal(discMsg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode discovery message: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Client ID: (%s) [%s] %s\n", selfAddr.Hostname, selfAddr.IP, myID)

	// Create a PeerManager
	peerManager := comms.NewPeerManager(id)
	peerManager.OnMessage(func(peerID string, payload []byte) {
	})

	serviceAddr := selfAddr.Addr(config.SERVICE_PORT)
	receiver := comms.NewTCPReceiver(serviceAddr, id, peerManager)
	receiver.Start()
	fmt.Println("Receiver started")

	// Start discovery service
	service, err := discovery.NewDiscoveryService(msgBytes, 60*time.Second, func(data []byte, addr *net.UDPAddr) {
		var msg models.DiscoveryMessage
		if err := proto.Unmarshal(data, &msg); err == nil {
			if err := peerManager.RegisterDiscovery(&msg, addr); err != nil {
				fmt.Printf("[REGISTER ERROR] from %s: %v\n", addr.IP, err)
			} else {
				fmt.Printf("[DISCOVERED] ID: %s, Name: %s, IP: %s, Port: %d\n", msg.Id, msg.Name, addr.IP, msg.Port)
			}
		} else {
			fmt.Printf("[INVALID MESSAGE] from %s: %v\n", addr.IP, err)
		}
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start discovery service: %v\n", err)
		os.Exit(1)
	}
	service.Start()
	fmt.Println("Discovery started. Press Ctrl+C to stop.")

	// Wait for interrupt to gracefully shut down
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down discovery service.")
	service.Stop()
	peerManager.Stop()
}
