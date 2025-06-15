package comms

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/eglochon/simple-lan-messaging/models"
	"github.com/eglochon/simple-lan-messaging/pkg/identity"
)

type PeerManager struct {
	self *identity.Identity

	mu      sync.RWMutex
	running bool
	peers   map[string]*Peer // peerID → *Peer

	onMessage          func(peerID string, payload []byte)
	onPeerDisconnected func(peerID string)
}

// [NewPeerManager] creates a new peer manager for "self"
func NewPeerManager(self *identity.Identity) *PeerManager {
	return &PeerManager{
		self:    self,
		running: true,
		peers:   make(map[string]*Peer),
	}
}

// [Stop] stops all running read-loops of connections
func (pm *PeerManager) Stop() {
	pm.running = false
}

// [RegisterDiscovery] handles incoming discovery messages and registers or updates peers.
func (pm *PeerManager) RegisterDiscovery(msg *models.DiscoveryMessage, addr net.Addr) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var encPubKey [32]byte
	var peerIP string
	peerPort := uint16(0)

	peerID := msg.GetId()

	// Decode base64-encoded EncPubKey string
	encPubKeyStr := msg.GetEnc()
	encPubKeyBytes, err := base64.RawURLEncoding.DecodeString(encPubKeyStr)
	if err == nil && len(encPubKeyBytes) == 32 {
		copy(encPubKey[:], encPubKeyBytes)
	} else {
		log.Printf("[WARN] Invalid EncPubKey from peer %s at %s: %v (IGNORED)", peerID, addr.String(), err)
		return err
	}

	// Update IP/port & last seen
	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		peerIP = udpAddr.IP.String()
		peerPort = uint16(msg.GetPort())
	} else {
		return errors.New("could not parse IP addr")
	}

	peer, exists := pm.peers[peerID]
	if !exists {
		peer = &Peer{
			ID:        peerID,
			EncPubKey: encPubKey,
			IP:        peerIP,
			Port:      peerPort,
			LastSeen:  time.Now(),
		}
		pm.peers[peerID] = peer
	} else {
		peer.EncPubKey = encPubKey
		peer.LastSeen = time.Now()

		if peerID != peer.IP || peerPort != peer.Port {
			peer.IP = peerID
			peer.Port = peerPort

			if peer.Conn != nil {
				peer.Conn.Close()
				peer.Conn = nil
			}
		}
	}
	return nil
}

// [Connect] initiates a secure connection to a peer, performing handshake if needed.
func (pm *PeerManager) Connect(peerID string) (*SecureConn, error) {
	pm.mu.Lock()
	peer, exists := pm.peers[peerID]
	pm.mu.Unlock()

	if !exists {
		return nil, errors.New("peer not found")
	}
	if peer.Conn != nil {
		// Already connected
		return peer.Conn, nil
	}

	conn, _, err := DialSecurePeer(peer.Addr(), pm.self)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	pm.mu.Lock()
	peer.Conn = conn
	pm.mu.Unlock()

	// Start background read loop (optional)
	go pm.readLoop(peer)

	return conn, nil
}

// [Accept] accepts an inbound connection, performs handshake, and registers the peer.
func (pm *PeerManager) Accept(conn net.Conn) error {
	sc, peerID, err := AcceptSecureConn(conn, pm.self)
	if err != nil {
		return err
	}

	pm.mu.Lock()
	peer, exists := pm.peers[peerID]
	if !exists {
		peer = &Peer{ID: peerID}
		pm.peers[peerID] = peer
	}
	peer.Conn = sc
	pm.mu.Unlock()

	go pm.readLoop(peer)

	return nil
}

// [OnMessage] registers a callback that will be called on message receipt.
func (pm *PeerManager) OnMessage(fn func(peerID string, payload []byte)) {
	pm.onMessage = fn
}

// [readLoop] continuously reads decrypted messages from a peer and calls the message handler.
func (pm *PeerManager) readLoop(peer *Peer) {
	for pm.running {
		msg, err := peer.Conn.ReadEncrypted()
		if err != nil {
			log.Printf("[INFO] Disconnected from peer %s: %v", peer.ID, err)

			pm.mu.Lock()
			peer.Conn.Close()
			peer.Conn = nil
			pm.mu.Unlock()

			// Optional: notify upper layer peer went offline
			if pm.onPeerDisconnected != nil {
				pm.onPeerDisconnected(peer.ID)
			}
			return
		}
		if pm.onMessage != nil {
			pm.onMessage(peer.ID, msg)
		}
	}
}
