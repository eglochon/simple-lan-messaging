package discovery

import (
	"errors"
	"fmt"
	"net"
	"os"
)

type SelfAddress struct {
	Hostname string
	IP       string
}

func NewSelfAddress() (*SelfAddress, error) {
	conn, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	strIP := localAddr.IP.String()

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("Error retrieving hostname: %v\n", err)
		return nil, errors.Join(errors.New("error retrieving hostname"), err)
	}

	return &SelfAddress{
		Hostname: hostname,
		IP:       strIP,
	}, nil
}

// Addr returns the peer's TCP address as "IP:Port"
func (sa *SelfAddress) Addr(port uint16) string {
	return fmt.Sprintf("%s:%d", sa.IP, port)
}
