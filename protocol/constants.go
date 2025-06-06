// Package protocol provides constants and types for message encoding/decoding in the SocketHub protocol.
// Defines message types, flags, protocol version, and header size for packet handling.
package protocol

// =============================================================================
// Protocol Constants
// =============================================================================

// CurrentVersion: SocketHub protocol version.
const CurrentVersion uint8 = 0x01

// HeaderSize: fixed size (bytes) of SocketHeader struct.
const HeaderSize = 76

// =============================================================================
// Message Types
// =============================================================================

// MessageType: semantic type of a message payload.
type MessageType uint8

const (
	MessageTypeUnknown   MessageType = iota // Uninitialized/default
	MessageTypeData                         // Application data
	MessageTypeBroadcast                    // Broadcast message
	MessageTypeHeartbeat                    // Keep-alive
	// Extend with more message types as needed.
)

// String returns the string representation of MessageType.
func (m MessageType) String() string {
	switch m {
	case MessageTypeUnknown:
		return "Unknown"
	case MessageTypeData:
		return "Data"
	case MessageTypeBroadcast:
		return "Broadcast"
	case MessageTypeHeartbeat:
		return "Heartbeat"
	default:
		return "InvalidMessageType"
	}
}

// IsValid returns true if the MessageType is within valid range.
func (m MessageType) IsValid() bool {
	return m <= MessageTypeHeartbeat
}

// =============================================================================
// Message Flags
// =============================================================================

// Flag: bitmask for message attributes.
type Flag uint8

const (
	FlagNone       Flag = iota // No flags set
	FlagACK                    // Acknowledgment
	FlagError                  // Indicates an error in the message
	FlagCompressed             // Indicates the payload is compressed
	FlagEncrypted              // Indicates the payload is encrypted
	// Add more flags as needed.
)

// String returns the string representation of Flag.
func (f Flag) String() string {
	switch f {
	case FlagNone:
		return "None"
	case FlagACK:
		return "ACK"
	case FlagError:
		return "Error"
	case FlagCompressed:
		return "Compressed"
	case FlagEncrypted:
		return "Encrypted"
	default:
		return "InvalidFlag"
	}
}

// IsValid returns true if the Flag is within valid range.
func (f Flag) IsValid() bool {
	return f <= FlagEncrypted
}

// HasFlag checks if a specific flag is set in a bitmask.
func HasFlag(flags, flag Flag) bool {
	return flags&flag != 0
}

// SetFlag sets a specific flag in a bitmask.
func SetFlag(flags, flag Flag) Flag {
	return flags | flag
}

// ClearFlag clears a specific flag from a bitmask.
func ClearFlag(flags, flag Flag) Flag {
	return flags &^ flag
}

// =============================================================================
// Protocol Types
// =============================================================================

// ProtocolType represents the transport protocol.
type ProtocolType uint8

const (
	ProtocolTCP ProtocolType = iota
	ProtocolUDP
)

// String returns the string representation of ProtocolType.
func (p ProtocolType) String() string {
	switch p {
	case ProtocolTCP:
		return "TCP"
	case ProtocolUDP:
		return "UDP"
	default:
		return "InvalidProtocol"
	}
}

// IsValid returns true if the ProtocolType is within valid range.
func (p ProtocolType) IsValid() bool {
	return p == ProtocolTCP || p == ProtocolUDP
}
