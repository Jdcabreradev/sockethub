package protocol

import (
	"encoding/binary"
	"errors"
)

// HeaderEncode serializes the header and payload into a single byte slice.
func HeaderEncode(h *SocketHeader) ([]byte, error) {
	if h == nil {
		return nil, errors.New("protohub: header is nil")
	}

	// Set timestamp
	h.SetTimestampIfZero()

	// Calculate base size
	headerSize := h.HeaderSize()

	// Create buffer with size header
	buf := make([]byte, headerSize)
	offset := 0

	// Write fixed fields in same order as decoder
	copy(buf[offset:], h.ID[:])
	offset += 16

	copy(buf[offset:], h.Sender[:])
	offset += 16

	binary.BigEndian.PutUint64(buf[offset:], h.Timestamp)
	offset += 8

	binary.BigEndian.PutUint64(buf[offset:], h.Length)
	offset += 8

	// Control bytes
	buf[offset] = byte(h.Flags)
	buf[offset+1] = byte(h.MessageType)
	buf[offset+2] = h.Router
	buf[offset+3] = byte(h.Protocol)
	offset += 4

	// Optional fields
	if h.IsBroadcast() {
		copy(buf[offset:], h.Receiver[:])
		offset += 16
	}

	if h.Protocol == ProtocolUDP {
		binary.BigEndian.PutUint32(buf[offset:], h.Sequence)
		offset += 4
	}
	return buf, nil
}
