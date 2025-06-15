package discovery

import (
	"fmt"
	"net"
	"time"

	"github.com/eglochon/simple-lan-messaging/config"
	"golang.org/x/net/ipv4"
)

type DiscoveryService struct {
	Message   []byte
	Interval  time.Duration
	running   bool
	onMessage func(data []byte, addr *net.UDPAddr)
	selfAddr  *SelfAddress
}

// NewDiscoveryService creates a new instance
func NewDiscoveryService(message []byte, interval time.Duration, onMessage func(data []byte, addr *net.UDPAddr)) (*DiscoveryService, error) {
	selfAddr, err := NewSelfAddress()
	if err != nil {
		return nil, err
	}
	return &DiscoveryService{
		Message:   message,
		Interval:  interval,
		onMessage: onMessage,
		selfAddr:  selfAddr,
	}, nil
}

func (d *DiscoveryService) SetInterval(interval time.Duration) {
	d.Interval = interval
}

func (d *DiscoveryService) Start() {
	d.running = true

	go d.listen()
	go d.broadcast()
}

func (d *DiscoveryService) Stop() {
	d.running = false
}

func (d *DiscoveryService) listen() {
	addr, err := net.ResolveUDPAddr("udp", config.MULTICAST_ADDR)
	if err != nil {
		fmt.Println("Resolve error:", err)
		return
	}

	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Listen error:", err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)

	for d.running {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		if src.IP.String() != d.selfAddr.IP {
			d.onMessage(buf[:n], src)
		}
	}
}

func (d *DiscoveryService) broadcast() {
	groupAddr, err := net.ResolveUDPAddr("udp4", config.MULTICAST_ADDR)
	if err != nil {
		fmt.Println("Broadcast resolve error:", err)
		return
	}

	conn, err := net.DialUDP("udp4", nil, groupAddr)
	if err != nil {
		fmt.Println("Broadcast dial error:", err)
		return
	}
	defer conn.Close()

	p := ipv4.NewPacketConn(conn)
	err = p.SetMulticastLoopback(false)
	if err != nil {
		fmt.Println("Failed to disable loopback:", err)
	}

	for d.running {
		_, err := conn.Write(d.Message)
		if err != nil {
			fmt.Println("Broadcast write error:", err)
		}
		time.Sleep(d.Interval)
	}
}
