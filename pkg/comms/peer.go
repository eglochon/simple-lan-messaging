package comms

import (
	"fmt"
	"time"
)

// Peer represents a known or connected remote client
type Peer struct {
	ID        string      // base64 Ed25519 public key
	Name      string      // optional display name
	IP        string      // last known IP address
	Port      uint16      // service port for TCP connection
	EncPubKey [32]byte    // X25519 public key
	LastSeen  time.Time   // last discovery or message
	Conn      *SecureConn // nil if not connected
}

// Addr returns the peer's TCP address as "IP:Port"
func (p *Peer) Addr() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

// IsConnected returns true if the peer currently has a live connection
func (p *Peer) IsConnected() bool {
	return p.Conn != nil
}
