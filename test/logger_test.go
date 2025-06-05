package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/Jdcabreradev/sockethub/logger"
)

// TestNewLogger tests logger creation with different modes
func TestNewLogger(t *testing.T) {
	tempDir := t.TempDir() // Creates temporary directory that's automatically cleaned up

	tests := []struct {
		name    string
		mode    socketlog.LogMode
		wantErr bool
	}{
		{"DEV mode", socketlog.DEV, false},
		{"RELEASE mode", socketlog.RELEASE, false},
		{"VERBOSE mode", socketlog.VERBOSE, false},
		{"HIDDEN mode", socketlog.HIDDEN, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := socketlog.NewLogger(tempDir, tt.mode)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if logger == nil && !tt.wantErr {
				t.Error("NewLogger() returned nil logger without error")
				return
			}

			if logger != nil {
				defer logger.Close()
			}
		})
	}
}

// TestLoggerLog tests the main logging functionality
func TestLoggerLog(t *testing.T) {
	tempDir := t.TempDir()

	// Test different modes
	modes := []socketlog.LogMode{socketlog.DEV, socketlog.RELEASE, socketlog.VERBOSE, socketlog.HIDDEN}
	
	for _, mode := range modes {
		t.Run("Mode_"+modeToString(mode), func(t *testing.T) {
			logger, err := socketlog.NewLogger(tempDir, mode)
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}
			defer logger.Close()

			// Test all log types
			logger.Log("AuthService", socketlog.INFO, "User authenticated successfully")
			logger.Log("CacheService", socketlog.WARNING, "High memory usage detected")
			logger.Log("PaymentService", socketlog.ERROR, "Transaction failed")
			logger.Log("DebugService", socketlog.DEBUG, "Cache invalidated")

			// Flush to ensure data is written
			if err := logger.Flush(); err != nil {
				t.Errorf("Failed to flush logger: %v", err)
			}
		})
	}
}

// TestLoggerSetColor tests color changing functionality
func TestLoggerSetColor(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := socketlog.NewLogger(tempDir, socketlog.DEV)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Test setting valid colors
	validColors := []string{socketlog.Red, socketlog.Green, socketlog.Yellow, socketlog.Blue, socketlog.Magenta, socketlog.Cyan}
	
	for _, color := range validColors {
		logger.SetColor(socketlog.INFO, color)
		// Color change doesn't return error, so we just ensure it doesn't panic
		logger.Log("TestService", socketlog.INFO, "Testing color: "+color)
	}
}

// TestLoggerFlush tests the flush functionality
func TestLoggerFlush(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := socketlog.NewLogger(tempDir, socketlog.VERBOSE)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log some messages
	logger.Log("TestService", socketlog.INFO, "Message before flush")
	
	// Test flush
	if err := logger.Flush(); err != nil {
		t.Errorf("Flush() returned error: %v", err)
	}

	logger.Log("TestService", socketlog.INFO, "Message after flush")
}

// TestLoggerClose tests proper resource cleanup
func TestLoggerClose(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := socketlog.NewLogger(tempDir, socketlog.RELEASE)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log something to ensure file is created
	logger.Log("TestService", socketlog.INFO, "Test message")

	// Test close
	if err := logger.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Test that multiple closes don't cause errors
	if err := logger.Close(); err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}

	// Test that logging after close doesn't panic
	logger.Log("TestService", socketlog.INFO, "This should be ignored")
}

// TestLogFileCreation tests that log files are created correctly
func TestLogFileCreation(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test modes that should create files
	fileModes := []socketlog.LogMode{socketlog.RELEASE, socketlog.VERBOSE, socketlog.HIDDEN}
	
	for _, mode := range fileModes {
		t.Run("FileCreation_"+modeToString(mode), func(t *testing.T) {
			logger, err := socketlog.NewLogger(tempDir, mode)
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}
			
			// Log something
			logger.Log("TestService", socketlog.INFO, "Test message")
			logger.Flush()
			logger.Close()

			// Check if log file was created
			files, err := filepath.Glob(filepath.Join(tempDir, "*.log"))
			if err != nil {
				t.Fatalf("Failed to glob log files: %v", err)
			}
			
			if len(files) == 0 {
				t.Error("No log file was created")
			}
		})
	}
}

// TestLogContent tests that log content is written correctly
func TestLogContent(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := socketlog.NewLogger(tempDir, socketlog.VERBOSE)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	testMessage := "This is a test message"
	testConsumer := "TestService"
	
	logger.Log(testConsumer, socketlog.INFO, testMessage)
	logger.Flush()
	logger.Close()

	// Read the log file
	files, err := filepath.Glob(filepath.Join(tempDir, "*.log"))
	if err != nil || len(files) == 0 {
		t.Fatalf("No log file found")
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	
	// Check if our message and consumer are in the log
	if !strings.Contains(logContent, testMessage) {
		t.Errorf("Log content doesn't contain test message: %s", logContent)
	}
	
	if !strings.Contains(logContent, testConsumer) {
		t.Errorf("Log content doesn't contain test consumer: %s", logContent)
	}
	
	if !strings.Contains(logContent, "INFO") {
		t.Errorf("Log content doesn't contain log type: %s", logContent)
	}
}

// BenchmarkLogger benchmarks logging performance
func BenchmarkLogger(b *testing.B) {
	tempDir := b.TempDir()
	logger, err := socketlog.NewLogger(tempDir, socketlog.RELEASE)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		logger.Log("BenchService", socketlog.INFO, "Benchmark message")
	}
	
	logger.Flush()
}

// Helper function to convert LogMode to string for test names
func modeToString(mode socketlog.LogMode) string {
	switch mode {
	case socketlog.DEV:
		return "DEV"
	case socketlog.RELEASE:
		return "RELEASE"
	case socketlog.VERBOSE:
		return "VERBOSE"
	case socketlog.HIDDEN:
		return "HIDDEN"
	default:
		return "UNKNOWN"
	}
}