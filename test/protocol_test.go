package test

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Jdcabreradev/sockethub/protocol"
	"github.com/google/uuid"
)

func TestTCPConnection(t *testing.T) {
	// Start TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0") // Use :0 for random port
	if err != nil {
		t.Fatalf("TCP server listen error: %v", err)
	}
	defer listener.Close()

	serverAddr := listener.Addr().String()
	var wg sync.WaitGroup
	wg.Add(1)

	// Start server goroutine
	go func() {
		defer wg.Done()
		conn, err := listener.Accept()
		if err != nil {
			t.Errorf("TCP server accept error: %v", err)
			return
		}
		defer conn.Close()

		wrapped := protocol.NewTCPConnWrapper(conn)

		// Read incoming message
		header, payload, err := wrapped.ReadFrame()
		if err != nil {
			t.Errorf("TCP server ReadFrame error: %v", err)
			return
		}

		expectedPayload := "hello from tcp client"
		if string(payload) != expectedPayload {
			t.Errorf("TCP server received unexpected payload: got %s, want %s", string(payload), expectedPayload)
			return
		}

		// Verify header fields
		if header.MessageType != protocol.MessageTypeData {
			t.Errorf("TCP server unexpected message type: got %v, want %v", header.MessageType, protocol.MessageTypeData)
		}
		if header.Protocol != protocol.ProtocolTCP {
			t.Errorf("TCP server unexpected protocol: got %v, want %v", header.Protocol, protocol.ProtocolTCP)
		}
		if header.Router != 42 {
			t.Errorf("TCP server unexpected router: got %v, want %v", header.Router, 42)
		}

		t.Logf("[TCP SERVER] Received: %s", string(payload))

		// Send response
		header.MessageType = protocol.MessageTypeData
		responsePayload := []byte("hello back from tcp server")
		if err := wrapped.WriteFrame(header, responsePayload); err != nil {
			t.Errorf("TCP server WriteFrame error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	// Start client
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatalf("TCP client dial error: %v", err)
	}
	defer conn.Close()

	wrapped := protocol.NewTCPConnWrapper(conn)

	// Create and send message
	clientID := uuid.New()
	senderID := uuid.New()
	header := &protocol.SocketHeader{
		ID:          clientID,
		Sender:      senderID,
		MessageType: protocol.MessageTypeData,
		Protocol:    protocol.ProtocolTCP,
		Router:      42,
	}
	payload := []byte("hello from tcp client")

	if err := wrapped.WriteFrame(header, payload); err != nil {
		t.Fatalf("TCP client WriteFrame error: %v", err)
	}

	// Read response
	respHeader, respPayload, err := wrapped.ReadFrame()
	if err != nil {
		t.Fatalf("TCP client ReadFrame error: %v", err)
	}

	expectedResponse := "hello back from tcp server"
	if string(respPayload) != expectedResponse {
		t.Errorf("TCP client received unexpected response: got %s, want %s", string(respPayload), expectedResponse)
	}

	// Verify response header
	if respHeader.ID != clientID {
		t.Errorf("TCP client response ID mismatch: got %v, want %v", respHeader.ID, clientID)
	}

	t.Logf("[TCP CLIENT] Got response: %s", string(respPayload))

	// Wait for server to finish
	wg.Wait()
}

func TestUDPConnection(t *testing.T) {
	// Start UDP server
	pc, err := net.ListenPacket("udp", "127.0.0.1:0") // Use :0 for random port
	if err != nil {
		t.Fatalf("UDP server listen error: %v", err)
	}
	defer pc.Close()

	serverAddr := pc.LocalAddr()
	var wg sync.WaitGroup
	wg.Add(1)

	// Start server goroutine
	go func() {
		defer wg.Done()
		wrapped := protocol.NewUDPConnWrapper(pc, nil, 2048)

		// Read incoming message
		header, payload, err := wrapped.ReadFrame()
		if err != nil {
			t.Errorf("UDP server ReadFrame error: %v", err)
			return
		}

		expectedPayload := "hello from udp client"
		if string(payload) != expectedPayload {
			t.Errorf("UDP server received unexpected payload: got %s, want %s", string(payload), expectedPayload)
			return
		}

		// Verify header fields
		if header.MessageType != protocol.MessageTypeData {
			t.Errorf("UDP server unexpected message type: got %v, want %v", header.MessageType, protocol.MessageTypeData)
		}
		if header.Protocol != protocol.ProtocolUDP {
			t.Errorf("UDP server unexpected protocol: got %v, want %v", header.Protocol, protocol.ProtocolUDP)
		}
		if header.Router != 42 {
			t.Errorf("UDP server unexpected router: got %v, want %v", header.Router, 42)
		}

		t.Logf("[UDP SERVER] Received: %s", string(payload))

		// Send response
		header.MessageType = protocol.MessageTypeData
		responsePayload := []byte("hello back from udp server")
		if err := wrapped.WriteFrame(header, responsePayload); err != nil {
			t.Errorf("UDP server WriteFrame error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	// Create UDP client connection
	clientPC, err := net.ListenPacket("udp", ":0")
	if err != nil {
		t.Fatalf("UDP client listen error: %v", err)
	}
	defer clientPC.Close()

	wrapped := protocol.NewUDPConnWrapper(clientPC, serverAddr, 2048)

	// Create and send message
	clientID := uuid.New()
	senderID := uuid.New()
	header := &protocol.SocketHeader{
		ID:          clientID,
		Sender:      senderID,
		MessageType: protocol.MessageTypeData,
		Protocol:    protocol.ProtocolUDP,
		Router:      42,
	}
	payload := []byte("hello from udp client")

	if err := wrapped.WriteFrame(header, payload); err != nil {
		t.Fatalf("UDP client WriteFrame error: %v", err)
	}

	// Read response
	respHeader, respPayload, err := wrapped.ReadFrame()
	if err != nil {
		t.Fatalf("UDP client ReadFrame error: %v", err)
	}

	expectedResponse := "hello back from udp server"
	if string(respPayload) != expectedResponse {
		t.Errorf("UDP client received unexpected response: got %s, want %s", string(respPayload), expectedResponse)
	}

	// Verify response header
	if respHeader.ID != clientID {
		t.Errorf("UDP client response ID mismatch: got %v, want %v", respHeader.ID, clientID)
	}

	t.Logf("[UDP CLIENT] Got response: %s", string(respPayload))

	// Wait for server to finish
	wg.Wait()
}

func TestTCPConnectionWithMultipleMessages(t *testing.T) {
	// Start TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("TCP server listen error: %v", err)
	}
	defer listener.Close()

	serverAddr := listener.Addr().String()
	var wg sync.WaitGroup
	wg.Add(1)

	// Start server goroutine
	go func() {
		defer wg.Done()
		conn, err := listener.Accept()
		if err != nil {
			t.Errorf("TCP server accept error: %v", err)
			return
		}
		defer conn.Close()

		wrapped := protocol.NewTCPConnWrapper(conn)

		// Handle multiple messages
		for i := 0; i < 3; i++ {
			header, payload, err := wrapped.ReadFrame()
			if err != nil {
				t.Errorf("TCP server ReadFrame error on message %d: %v", i, err)
				return
			}

			expectedPayload := fmt.Sprintf("message %d", i)
			if string(payload) != expectedPayload {
				t.Errorf("TCP server message %d: got %s, want %s", i, string(payload), expectedPayload)
				return
			}

			// Echo back
			responsePayload := []byte(fmt.Sprintf("echo %d", i))
			if err := wrapped.WriteFrame(header, responsePayload); err != nil {
				t.Errorf("TCP server WriteFrame error on message %d: %v", i, err)
				return
			}
		}
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	// Start client
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatalf("TCP client dial error: %v", err)
	}
	defer conn.Close()

	wrapped := protocol.NewTCPConnWrapper(conn)

	// Send multiple messages
	for i := 0; i < 3; i++ {
		header := &protocol.SocketHeader{
			ID:          uuid.New(),
			Sender:      uuid.New(),
			MessageType: protocol.MessageTypeData,
			Protocol:    protocol.ProtocolTCP,
			Router:      uint8(i),
		}
		payload := []byte(fmt.Sprintf("message %d", i))

		if err := wrapped.WriteFrame(header, payload); err != nil {
			t.Fatalf("TCP client WriteFrame error on message %d: %v", i, err)
		}

		// Read response
		_, respPayload, err := wrapped.ReadFrame()
		if err != nil {
			t.Fatalf("TCP client ReadFrame error on message %d: %v", i, err)
		}

		expectedResponse := fmt.Sprintf("echo %d", i)
		if string(respPayload) != expectedResponse {
			t.Errorf("TCP client message %d response: got %s, want %s", i, string(respPayload), expectedResponse)
		}
	}

	// Wait for server to finish
	wg.Wait()
}

func TestConnectionInterfaces(t *testing.T) {
	// Test that both wrappers implement the Conn interface
	var tcpConn protocol.Conn
	var udpConn protocol.Conn

	// Create test connections
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create TCP listener: %v", err)
	}
	defer listener.Close()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create UDP listener: %v", err)
	}
	defer pc.Close()

	// Test TCP wrapper implements interface
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial TCP: %v", err)
	}
	defer conn.Close()

	tcpConn = protocol.NewTCPConnWrapper(conn)
	if tcpConn == nil {
		t.Error("TCP wrapper is nil")
	}

	// Test UDP wrapper implements interface
	udpConn = protocol.NewUDPConnWrapper(pc, pc.LocalAddr(), 2048)
	if udpConn == nil {
		t.Error("UDP wrapper is nil")
	}

	// Test interface methods are available
	testID := uuid.New()
	tcpConn.SetSender(testID)
	if tcpConn.GetSender() != testID {
		t.Error("TCP SetSender/GetSender failed")
	}

	udpConn.SetSender(testID)
	if udpConn.GetSender() != testID {
		t.Error("UDP SetSender/GetSender failed")
	}

	// Test address methods
	if tcpConn.LocalAddr() == nil {
		t.Error("TCP LocalAddr returned nil")
	}
	if tcpConn.RemoteAddr() == nil {
		t.Error("TCP RemoteAddr returned nil")
	}
	if udpConn.LocalAddr() == nil {
		t.Error("UDP LocalAddr returned nil")
	}
}