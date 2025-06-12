package discovery

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/ipv4"
)

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

type DiscoveryService struct {
	Message   []byte
	Interval  time.Duration
	running   bool
	onMessage func(data []byte, addr *net.UDPAddr)
	localIP   *string
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
	multicastAddr = "224.0.0.250:40400"
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
		fmt.Println("Resolve error:", err)
		return
	}

	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Listen error:", err)
		return
	}
	defer conn.Close()

	// if d.localIP == nil {
	// 	localIP := getLocalIP()
	// 	d.localIP = &localIP
	// }
	buf := make([]byte, 1024)

	for d.running {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		//if src.IP.String() != *d.localIP {
		d.onMessage(buf[:n], src)
		//}
	}
}

func (d *DiscoveryService) broadcast() {
	groupAddr, err := net.ResolveUDPAddr("udp4", multicastAddr)
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
