package comms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net"

	"github.com/eglochon/simple-lan-messaging/models"
	"github.com/eglochon/simple-lan-messaging/pkg/identity"
	"golang.org/x/crypto/curve25519"
	"google.golang.org/protobuf/proto"
)

type SecureConn struct {
	conn   net.Conn
	stream cipher.AEAD
}

func (sc *SecureConn) WriteEncrypted(plaintext []byte) error {
	nonce := make([]byte, sc.stream.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	ciphertext := sc.stream.Seal(nonce, nonce, plaintext, nil)

	lenBuf := make([]byte, 2)
	lenBuf[0], lenBuf[1] = byte(len(ciphertext)>>8), byte(len(ciphertext))

	if _, err := sc.conn.Write(lenBuf); err != nil {
		return err
	}
	_, err := sc.conn.Write(ciphertext)
	return err
}

func (sc *SecureConn) ReadEncrypted() ([]byte, error) {
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(sc.conn, lenBuf); err != nil {
		return nil, err
	}
	length := int(lenBuf[0])<<8 | int(lenBuf[1])
	if length > 64*1024 {
		return nil, errors.New("message too large")
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(sc.conn, buf); err != nil {
		return nil, err
	}

	nonceSize := sc.stream.NonceSize()
	if len(buf) < nonceSize {
		return nil, errors.New("invalid ciphertext")
	}
	nonce := buf[:nonceSize]
	ciphertext := buf[nonceSize:]

	return sc.stream.Open(nil, nonce, ciphertext, nil)
}

func (sc *SecureConn) Close() error {
	return sc.conn.Close()
}

func DialSecurePeer(addr string, self *identity.Identity) (*SecureConn, string, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, "", err
	}
	aead, peerID, err := performHandshake(conn, self, true)
	if err != nil {
		conn.Close()
		return nil, "", err
	}
	return &SecureConn{conn: conn, stream: aead}, peerID, nil
}

func AcceptSecureConn(conn net.Conn, self *identity.Identity) (*SecureConn, string, error) {
	aead, peerID, err := performHandshake(conn, self, false)
	if err != nil {
		conn.Close()
		return nil, "", err
	}
	return &SecureConn{conn: conn, stream: aead}, peerID, nil
}

func performHandshake(conn net.Conn, self *identity.Identity, isInitiator bool) (cipher.AEAD, string, error) {
	var ephPriv [32]byte
	if _, err := rand.Read(ephPriv[:]); err != nil {
		return nil, "", err
	}
	var ephPub [32]byte
	curve25519.ScalarBaseMult(&ephPub, &ephPriv)

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", err
	}

	msgToSign := append(self.SigningPublicKey, ephPub[:]...)
	msgToSign = append(msgToSign, nonce...)
	signature := ed25519.Sign(self.SigningPrivateKey, msgToSign)

	hs := &models.Handshake{
		Id:    base64.RawURLEncoding.EncodeToString(self.SigningPublicKey),
		Enc:   base64.RawURLEncoding.EncodeToString(ephPub[:]),
		Sig:   base64.RawURLEncoding.EncodeToString(signature),
		Nonce: base64.RawURLEncoding.EncodeToString(nonce),
	}

	if isInitiator {
		if err := sendProto(conn, hs); err != nil {
			return nil, "", err
		}
	}

	peerHS := &models.Handshake{}
	if err := recvProto(conn, peerHS); err != nil {
		return nil, "", err
	}

	if !isInitiator {
		if err := sendProto(conn, hs); err != nil {
			return nil, "", err
		}
	}

	peerPub, err := base64.RawURLEncoding.DecodeString(peerHS.Id)
	if err != nil || len(peerPub) != ed25519.PublicKeySize {
		return nil, "", errors.New("invalid peer public key")
	}

	peerEncPub, err := base64.RawURLEncoding.DecodeString(peerHS.Enc)
	if err != nil || len(peerEncPub) != 32 {
		return nil, "", errors.New("invalid peer encryption key")
	}
	peerNonce, err := base64.RawURLEncoding.DecodeString(peerHS.Nonce)
	if err != nil {
		return nil, "", err
	}
	peerSig, err := base64.RawURLEncoding.DecodeString(peerHS.Sig)
	if err != nil {
		return nil, "", err
	}

	msgToVerify := append(peerPub, peerEncPub...)
	msgToVerify = append(msgToVerify, peerNonce...)
	if !ed25519.Verify(ed25519.PublicKey(peerPub), msgToVerify, peerSig) {
		return nil, "", errors.New("invalid handshake signature")
	}

	var peerEnc [32]byte
	copy(peerEnc[:], peerEncPub)
	sharedSecret, err := curve25519.X25519(ephPriv[:], peerEnc[:])
	if err != nil {
		return nil, "", err
	}
	key := sha256.Sum256(sharedSecret)

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, "", err
	}

	peerID := base64.RawURLEncoding.EncodeToString(peerPub)
	return aead, peerID, nil
}

func sendProto(w io.Writer, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	lenBuf := make([]byte, 2)
	lenBuf[0], lenBuf[1] = byte(len(data)>>8), byte(len(data))
	if _, err := w.Write(lenBuf); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func recvProto(r io.Reader, msg proto.Message) error {
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return err
	}
	size := int(lenBuf[0])<<8 | int(lenBuf[1])
	buf := make([]byte, size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	return proto.Unmarshal(buf, msg)
}
