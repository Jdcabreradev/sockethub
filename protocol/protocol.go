// File: connection/connection.go
package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/google/uuid"
)

// Conn abstracts a framed connection (TCP or UDP) with sender metadata.
// ReadFrame returns the full SocketHeader and payload; WriteFrame accepts a SocketHeader.
type Conn interface {
	ReadFrame() (*SocketHeader, []byte, error)
	WriteFrame(header *SocketHeader, payload []byte) error
	Close() error
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	SetSender(uuid.UUID)
	GetSender() uuid.UUID
}

// tcpConnWrapper wraps a net.Conn for framed I/O using protohub protocol.
type tcpConnWrapper struct {
	conn   net.Conn
	sender uuid.UUID
}

// NewTCPConnWrapper constructs a Conn from a net.Conn.
func NewTCPConnWrapper(c net.Conn) Conn {
	return &tcpConnWrapper{
		conn:   c,
		sender: uuid.New(), // default sender ID
	}
}

// ReadFrame reads a full frame (header + payload) from TCP.
func (t *tcpConnWrapper) ReadFrame() (*SocketHeader, []byte, error) {
	// Read the header size prefix to determine how much to read.
	HeaderSize := make([]byte, 1) // 1 byte for header size
	if _, err := io.ReadFull(t.conn, HeaderSize); err != nil {
		return nil, nil, fmt.Errorf("TCP: failed to read header size prefix: %w", err)
	}

	// Read the header
	headerBytes :=make([]byte, HeaderSize[0]) // Use the size from the prefix
	if _, err := io.ReadFull(t.conn, headerBytes); err != nil {
		return nil, nil, fmt.Errorf("TCP: failed to read header: %w", err)
	}
	// Decode the header
	h, err := HeaderDecode(headerBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("TCP: decode header error: %w", err)
	}

	// Now we allocate the buffer for the payload
	PayloadSize := make([]byte, h.Length+4) // +4 for checksum

	// Read the payload
	if _, err := io.ReadFull(t.conn, PayloadSize); err != nil {
		return nil, nil, fmt.Errorf("TCP: failed to read payload: %w", err)
	}

	// analyze the checksum
	checksumBytes := PayloadSize[len(PayloadSize)-4:]
	payloadData := PayloadSize[:len(PayloadSize)-4]
	checksum := binary.BigEndian.Uint32(checksumBytes)
	if checksum != Checksum(payloadData) {
		return nil, nil, fmt.Errorf("TCP: checksum mismatch")
	}

	// return the payload
	return h, payloadData, nil
}

// WriteFrame encodes the provided SocketHeader and payload, then writes to TCP.
func (t *tcpConnWrapper) WriteFrame(header *SocketHeader, payload []byte) error {
	if header == nil {
		return fmt.Errorf("TCP: header cannot be nil")
	}

	// Set payload length in header
	header.Length = uint64(len(payload))
	// (Other fields like Sender remain as is.)

	// Encode header and use its actual encoded length
	headerBytes, err := HeaderEncode(header)
	if err != nil {
		return fmt.Errorf("TCP: header encode error: %w", err)
	}
	encodedHeaderLen := len(headerBytes)

	// Calculate total message size using the actual header length
	messageSize := 1 + encodedHeaderLen + len(payload) + 4
	message := make([]byte, messageSize)

	// Write header size prefix using the encoded header length
	message[0] = uint8(encodedHeaderLen)
	// Copy header
	copy(message[1:], headerBytes)
	// Calculate payload start based on actual header length
	payloadStart := 1 + encodedHeaderLen
	copy(message[payloadStart:], payload)
	// Calculate checksum (over the payload)
	checksum := Checksum(payload)
	binary.BigEndian.PutUint32(message[len(message)-4:], checksum)

	_, err = t.conn.Write(message)
	return err
}

func (t *tcpConnWrapper) Close() error {
	return t.conn.Close()
}

func (t *tcpConnWrapper) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

func (t *tcpConnWrapper) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *tcpConnWrapper) SetReadDeadline(tm time.Time) error {
	return t.conn.SetReadDeadline(tm)
}

func (t *tcpConnWrapper) SetWriteDeadline(tm time.Time) error {
	return t.conn.SetWriteDeadline(tm)
}

func (t *tcpConnWrapper) SetSender(id uuid.UUID) {
	t.sender = id
}

func (t *tcpConnWrapper) GetSender() uuid.UUID {
	return t.sender
}

// udpConnWrapper wraps a net.PacketConn + remote address for framed I/O via protohub.
type udpConnWrapper struct {
	pc             net.PacketConn
	addr           net.Addr
	maxMessageSize int
	sender         uuid.UUID
	sequence       uint32
}

// NewUDPConnWrapper constructs a Conn from a PacketConn and remote Addr.
func NewUDPConnWrapper(pc net.PacketConn, addr net.Addr, maxSize int) Conn {
	return &udpConnWrapper{
		pc:             pc,
		addr:           addr,
		maxMessageSize: maxSize,
		sender:         uuid.New(),
		sequence:       0,
	}
}

// Corrected UDP ReadFrame and WriteFrame methods

// ReadFrame reads a full UDP datagram, decodes via protohub, and returns header + payload.
func (u *udpConnWrapper) ReadFrame() (*SocketHeader, []byte, error) {
	// Read complete UDP packet
	buf := make([]byte, u.maxMessageSize)
	n, addr, err := u.pc.ReadFrom(buf)
	if err != nil {
		return nil, nil, fmt.Errorf("UDP: read error: %w", err)
	}
	u.addr = addr

	// Read the header size prefix to determine how much to read.
	if n < 1 {
		return nil, nil, fmt.Errorf("UDP: packet too small for header size prefix")
	}
	headerSize := buf[0]

	// Verify we have enough data for header
	if n < 1+int(headerSize) {
		return nil, nil, fmt.Errorf("UDP: packet too small for header")
	}

	// Read the header
	headerBytes := buf[1:1+headerSize]
	
	// Decode the header
	h, err := HeaderDecode(headerBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("UDP: decode header error: %w", err)
	}

	// Calculate payload start position
	payloadStart := 1 + int(headerSize)
	
	// Verify we have enough data for payload + checksum
	if uint64(n) < uint64(payloadStart)+h.Length+4 { // +4 for checksum
		return nil, nil, fmt.Errorf("UDP: packet too small for payload and checksum")
	}

	// Now we allocate the buffer for the payload
	payloadSize := make([]byte, h.Length+4) // +4 for checksum
	copy(payloadSize, buf[payloadStart:payloadStart+int(h.Length+4)])

	// analyze the checksum
	checksumBytes := payloadSize[len(payloadSize)-4:]
	payloadData := payloadSize[:len(payloadSize)-4]
	checksum := binary.BigEndian.Uint32(checksumBytes)
	if checksum != Checksum(payloadData) {
		return nil, nil, fmt.Errorf("UDP: checksum mismatch")
	}

	// return the payload
	return h, payloadData, nil
}

// WriteFrame encodes the provided SocketHeader and payload, then sends as a UDP packet.
func (u *udpConnWrapper) WriteFrame(header *SocketHeader, payload []byte) error {
	if header == nil {
		return fmt.Errorf("UDP: header cannot be nil")
	}

	// Set UDP-specific fields (if needed)
	header.Sender = u.sender
	header.Protocol = ProtocolUDP
	header.Sequence = u.sequence
	u.sequence++

	// Set payload length in header
	header.Length = uint64(len(payload))
	// (Other fields like Sender remain as is.)

	// Encode header and use its actual encoded length
	headerBytes, err := HeaderEncode(header)
	if err != nil {
		return fmt.Errorf("UDP: header encode error: %w", err)
	}
	encodedHeaderLen := len(headerBytes)

	// Calculate total message size using the actual header length
	messageSize := 1 + encodedHeaderLen + len(payload) + 4
	if messageSize > u.maxMessageSize {
		return fmt.Errorf("UDP: message size %d exceeds maximum %d", messageSize, u.maxMessageSize)
	}
	
	message := make([]byte, messageSize)

	// Write header size prefix using the encoded header length
	message[0] = uint8(encodedHeaderLen)
	// Copy header
	copy(message[1:], headerBytes)
	// Calculate payload start based on actual header length
	payloadStart := 1 + encodedHeaderLen
	copy(message[payloadStart:], payload)
	// Calculate checksum (over the payload)
	checksum := Checksum(payload)
	binary.BigEndian.PutUint32(message[len(message)-4:], checksum)

	// Send UDP packet
	_, err = u.pc.WriteTo(message, u.addr)
	if err != nil {
		return fmt.Errorf("UDP: write error: %w", err)
	}

	return nil
}

func (u *udpConnWrapper) Close() error {
	return nil // server closes underlying PacketConn
}

func (u *udpConnWrapper) RemoteAddr() net.Addr {
	return u.addr
}

func (u *udpConnWrapper) LocalAddr() net.Addr {
	return u.pc.LocalAddr()
}

func (u *udpConnWrapper) SetReadDeadline(tm time.Time) error {
	return u.pc.SetDeadline(tm)
}

func (u *udpConnWrapper) SetWriteDeadline(tm time.Time) error {
	return u.pc.SetWriteDeadline(tm)
}

func (u *udpConnWrapper) SetSender(id uuid.UUID) {
	u.sender = id
}

func (u *udpConnWrapper) GetSender() uuid.UUID {
	return u.sender
}
