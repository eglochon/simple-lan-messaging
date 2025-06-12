package discovery

import (
	"fmt"
	"net"
	"time"
)

type DiscoveryService struct {
	Message   []byte
	Interval  time.Duration
	running   bool
	onMessage func(data []byte, addr *net.UDPAddr)
}

// NewDiscoveryService creates a new instance
func NewDiscoveryService(message []byte, interval time.Duration, onMessage func(data []byte, addr *net.UDPAddr)) *DiscoveryService {
	return &DiscoveryService{
		Message:   message,
		Interval:  interval,
		onMessage: onMessage,
	}
}

const (
	multicastAddr = "224.0.0.250:9999"
)

func (d *DiscoveryService) Start() {
	d.running = true

	go d.listen()
	go d.broadcast()
}

func (d *DiscoveryService) Stop() {
	d.running = false
}

func (d *DiscoveryService) listen() {
	addr, err := net.ResolveUDPAddr("udp", multicastAddr)
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}

	// Listen on all interfaces
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for d.running {
		n, src, err := conn.ReadFromUDP(buf)
		if err == nil && d.onMessage != nil {
			d.onMessage(buf[:n], src)
		}
	}
}

func (d *DiscoveryService) broadcast() {
	groupAddr, err := net.ResolveUDPAddr("udp", multicastAddr)
	if err != nil {
		fmt.Println("Broadcast resolve error:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, groupAddr)
	if err != nil {
		fmt.Println("Broadcast dial error:", err)
		return
	}
	defer conn.Close()

	for d.running {
		_, err := conn.Write(d.Message)
		if err != nil {
			fmt.Println("Broadcast write error:", err)
		}
		time.Sleep(d.Interval)
	}
}
