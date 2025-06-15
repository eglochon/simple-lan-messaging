package comms

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// SendMessage sends a raw encrypted message to a peer, connecting if needed.
func (pm *PeerManager) SendMessage(peerID string, message []byte) error {
	pm.mu.RLock()
	peer, exists := pm.peers[peerID]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("peer not found: %s", peerID)
	}

	if peer.Conn == nil {
		_, err := pm.Connect(peerID)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if peer.Conn == nil {
		return errors.New("no active connection after connect")
	}

	return peer.Conn.WriteEncrypted(message)
}

// SendProto marshals and sends a protobuf message securely.
func (pm *PeerManager) SendProto(peerID string, msg any) error {
	data, err := marshalProto(msg)
	if err != nil {
		return err
	}
	return pm.SendMessage(peerID, data)
}

func marshalProto(msg any) ([]byte, error) {
	m, ok := msg.(proto.Message)
	if !ok {
		return nil, errors.New("invalid protobuf message")
	}
	return proto.Marshal(m)
}
