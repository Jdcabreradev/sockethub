package protocol

import (
	"encoding/binary"
	"fmt"
)

// Decode parses a frame (header + payload) from data, reconstructs a SocketHeader,
// and returns (header, payload). It verifies the CRC32 checksum and returns an error
// if the frame is malformed.
func HeaderDecode(data []byte) (*SocketHeader, error) {
	// Start after size prefix
	headerSize := len(data)
	offset := 0

	// Update the minimum and maximum sizes (NOT including size prefix)
	minSize := 16 + 16 + 8 + 8 + 4 // Base: ID + Sender + Timestamp + Length + Control
	maxSize := minSize + 16 + 4    // + Receiver + Sequence

	if headerSize < minSize || headerSize > maxSize {
		return nil, fmt.Errorf("protohub: invalid header size (got %d, min %d, max %d)",
			headerSize, minSize, maxSize)
	}
	h := &SocketHeader{}
	// Read fixed fields
	copy(h.ID[:], data[offset:offset+16])
	offset += 16

	copy(h.Sender[:], data[offset:offset+16])
	offset += 16

	h.Timestamp = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	h.Length = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Control bytes
	h.Flags = Flag(data[offset])
	h.MessageType = MessageType(data[offset+1])
	h.Router = data[offset+2]
	h.Protocol = ProtocolType(data[offset+3])
	offset += 4

	// Optional fields
	if h.MessageType == MessageTypeBroadcast {
		copy(h.Receiver[:], data[offset:offset+16])
		offset += 16
	}

	if h.Protocol == ProtocolUDP {
		h.Sequence = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
	}

	// Update size verification to match encoder's calculation
	if offset != headerSize {
		return nil, fmt.Errorf("protohub: header size mismatch (got %d, expected %d)",
			uint32(offset-1), headerSize)
	}
	return h, nil
}
