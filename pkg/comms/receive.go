package comms

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/eglochon/simple-lan-messaging/pkg/identity"
)

// TCPReceiver listens for incoming peer connections and passes them to the PeerManager
type TCPReceiver struct {
	addr    string
	pm      *PeerManager
	self    *identity.Identity
	ln      net.Listener
	running bool
}

// NewTCPReceiver creates a new TCP server bound to the given address (e.g. ":9000")
func NewTCPReceiver(addr string, self *identity.Identity, pm *PeerManager) *TCPReceiver {
	return &TCPReceiver{
		addr: addr,
		self: self,
		pm:   pm,
	}
}

// Start begins accepting incoming TCP connections and registering them
func (r *TCPReceiver) Start() error {
	var err error
	r.ln, err = net.Listen("tcp", r.addr)
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}

	r.running = true
	log.Printf("[RECEIVER] Listening on %s", r.addr)

	go func() {
		for r.running {
			conn, err := r.ln.Accept()
			if err != nil {
				if r.running {
					log.Printf("[RECEIVER] Accept error: %v", err)
				}
				continue
			}

			go r.handleConnection(conn)
		}
	}()

	return nil
}

// Stop gracefully shuts down the listener
func (r *TCPReceiver) Stop() error {
	r.running = false
	if r.ln != nil {
		return r.ln.Close()
	}
	return nil
}

// handleConnection performs handshake and registers peer
func (r *TCPReceiver) handleConnection(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[RECEIVER] Recovered in handleConnection: %v", r)
			_ = conn.Close()
		}
	}()

	conn.SetDeadline(time.Now().Add(10 * time.Second)) // timeout for handshake
	sc, peerID, err := AcceptSecureConn(conn, r.self)
	if err != nil {
		log.Printf("[RECEIVER] Handshake failed: %v", err)
		_ = conn.Close()
		return
	}
	conn.SetDeadline(time.Time{}) // remove timeout

	r.pm.mu.Lock()
	peer, exists := r.pm.peers[peerID]
	if !exists {
		peer = &Peer{ID: peerID}
		r.pm.peers[peerID] = peer
	}
	peer.Conn = sc
	peer.LastSeen = time.Now()
	r.pm.mu.Unlock()

	log.Printf("[RECEIVER] Secure connection established with %s", peerID)

	go r.pm.readLoop(peer)
}
