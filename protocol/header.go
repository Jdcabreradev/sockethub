// Package protocol provides socket header structures for the SocketHub protocol.
// Defines the core SocketHeader struct (80 bytes) for network message routing and integrity.
package protocol

import (
	"time"

	"github.com/google/uuid"
)

type SocketHeader struct {
	ID          uuid.UUID    // Unique identifier for the message/packet
	Sender      uuid.UUID    // ID of the sender
	Receiver    uuid.UUID    // ID of the receiver (for direct or broadcast messages)
	Timestamp   uint64       // Unix timestamp (e.g., milliseconds) when the message was sent
	Length      uint64       // Length of the payload in bytes
	Sequence    uint32       // Monotonically increasing sequence number for ordering and reliability (UDP Only)
	Protocol    ProtocolType // Protocol type (e.g., TCP, UDP)
	Flags       Flag         // Flags for the message (e.g., ACK, Compressed, Encrypted, IsError)
	MessageType MessageType  // Type of message (e.g., Data, Control, Heartbeat, LoginRequest, LoginResponse)
	Router      uint8        // Router ID for routing messages to specific handlers
}

// IsBroadcast reports true if the MessageType is a broadcast message.
func (h *SocketHeader) IsBroadcast() bool {
	return h.MessageType == MessageTypeBroadcast && h.Receiver != uuid.Nil
}

// SetTimestampIfZero sets the Timestamp to “now” (in ms) if it is still zero.
func (h *SocketHeader) SetTimestampIfZero() {
	if h.Timestamp == 0 {
		h.Timestamp = uint64(time.Now().UnixMilli())
	}
}

// headerSize returns the serialized length of the header (excluding payload).
// It includes Receiver when IsBroadcast() and Sequence when Protocol == ProtocolUDP.
func (h *SocketHeader) HeaderSize() int {
	// Base size always emitted, in struct order:
	//   ID(16) + Sender(16) + Receiver(16, if broadcast) + Sequence(4, if UDP) +
	//   Timestamp(8) + Length(8) +
	//   Flags(1) + MessageType(1) + Router(1) + Protocol(1)
	size := 16 + 16 + 8 + 8 + 1 + 1 + 1 + 1

	if h.IsBroadcast() {
		size += 16 // Receiver
	}
	if h.Protocol == ProtocolUDP {
		size += 4 // Sequence
	}
	return size
}
