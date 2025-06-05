package sockethub_config

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/Jdcabreradev/sockethub/logger"
)

// ProtocolType represents the transport protocol
type ProtocolType int

const (
	ProtocolTCP ProtocolType = iota
	ProtocolUDP
)

// SocketConfig holds configuration for the server
type SocketConfig struct {
	IP                string            // IP to bind
	Port              uint16            // Port to listen on
	TLSConfig         *tls.Config       // TLS settings (nil for no TLS)
	LogMode           socketlog.LogMode // Logging verbosity
	Protocol          ProtocolType      // TCP or UDP
	MaxClients        uint32            // Maximum simultaneous clients (zero for no limit)
	BufferSize        int               // Buffer size for reads
	ReadTimeout       *time.Duration    // Read timeout per-client
	WriteTimeout      *time.Duration    // Write timeout per-client
	IdleTimeout       *time.Duration    // Idle timeout per-client
	SendChanSize      int               // Size of client send channels
	HeartbeatInterval *time.Duration    // Heartbeat interval for connection health (zero for disabled)
	EnableCompression bool              // Enable message compression
	MaxMessageSize    int               // Maximum message size in bytes (zero for no limits)
}

// DefaultConfig returns a reasonable default configuration
func DefaultConfig() *SocketConfig {
    defaultReadTimeout := 30 * time.Second
    defaultWriteTimeout := 10 * time.Second
    defaultIdleTimeout := 5 * time.Minute
    defaultHeartbeat := 30 * time.Second

    return &SocketConfig{
        IP:                "127.0.0.1",
        Port:              8080,
        LogMode:           socketlog.DEV,
        Protocol:          ProtocolTCP,
        MaxClients:        10000,
        BufferSize:        8192,
        ReadTimeout:       &defaultReadTimeout,
        WriteTimeout:      &defaultWriteTimeout,
        IdleTimeout:       &defaultIdleTimeout,
        SendChanSize:      256,
        HeartbeatInterval: &defaultHeartbeat,
        EnableCompression: true,
        MaxMessageSize:    4 * 1024 * 1024, // 4MB
    }
}

// Validate checks if the configuration is valid
func (c *SocketConfig) Validate() error {
	if c.Port == 0 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	if c.MaxClients <= 0 {
		return fmt.Errorf("maxClients must be greater than 0")
	}
	if c.BufferSize <= 0 {
		return fmt.Errorf("bufferSize must be greater than 0")
	}
	if c.SendChanSize <= 0 {
		return fmt.Errorf("sendChanSize must be greater than 0")
	}
	if c.MaxMessageSize <= 0 {
		return fmt.Errorf("maxMessageSize must be greater than 0")
	}
	return nil
}