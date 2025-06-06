package test

import (
	"testing"
	"time"

	"github.com/Jdcabreradev/sockethub/protocol"
	"github.com/google/uuid"
)

func TestHeaderCodec_MultipleCases(t *testing.T) {
	type testCase struct {
		name      string
		protocol  protocol.ProtocolType
		broadcast bool
	}

	cases := []testCase{
		{
			name:      "UDP without broadcast",
			protocol:  protocol.ProtocolUDP,
			broadcast: false,
		},
		{
			name:      "UDP with broadcast",
			protocol:  protocol.ProtocolUDP,
			broadcast: true,
		},
		{
			name:      "TCP without broadcast",
			protocol:  protocol.ProtocolTCP,
			broadcast: false,
		},
		{
			name:      "TCP with broadcast",
			protocol:  protocol.ProtocolTCP,
			broadcast: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test header
			header := &protocol.SocketHeader{
				ID:        uuid.New(),
				Sender:    uuid.New(),
				Protocol:  tc.protocol,
				Router:    1,
				Flags:     0,
				Length:    100,
				Timestamp: uint64(time.Now().UnixNano()),
			}

			if tc.broadcast {
				header.MessageType = protocol.MessageTypeBroadcast
				header.Receiver = uuid.New()
			} else {
				header.MessageType = protocol.MessageTypeData
			}

			if tc.protocol == protocol.ProtocolUDP {
				header.Sequence = 42
			}

			// Test encoding
			encoded, err := protocol.HeaderEncode(header)
			if err != nil {
				t.Fatalf("Failed to encode header: %v", err)
			}

			// Test decoding
			decoded, err := protocol.HeaderDecode(encoded)
			if err != nil {
				t.Fatalf("Failed to decode header: %v", err)
			}

			// Verify fields match
			if decoded.ID != header.ID {
				t.Errorf("ID mismatch: got %v, want %v", decoded.ID, header.ID)
			}
			if decoded.Sender != header.Sender {
				t.Errorf("Sender mismatch: got %v, want %v", decoded.Sender, header.Sender)
			}
			if decoded.Protocol != header.Protocol {
				t.Errorf("Protocol mismatch: got %v, want %v", decoded.Protocol, header.Protocol)
			}
			if decoded.MessageType != header.MessageType {
				t.Errorf("MessageType mismatch: got %v, want %v", decoded.MessageType, header.MessageType)
			}
			if tc.broadcast && decoded.Receiver != header.Receiver {
				t.Errorf("Receiver mismatch: got %v, want %v", decoded.Receiver, header.Receiver)
			}
			if tc.protocol == protocol.ProtocolUDP && decoded.Sequence != header.Sequence {
				t.Errorf("Sequence mismatch: got %v, want %v", decoded.Sequence, header.Sequence)
			}
		})
	}
}
