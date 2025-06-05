// Package socketlog provides a thread-safe, lightweight logger optimized for socket-based applications.
// It supports multiple log levels (INFO, WARNING, ERROR, DEBUG) and output modes (DEV, RELEASE, VERBOSE, HIDDEN).
//
// Key Features:
// - Built-in ANSI color output for console
// - File logging with automatic rotation (timestamped files)
// - Module-aware formatting (default: "[SocketHub]")
// - Thread-safe operations with sync.Mutex
//
// Example:
//
//	logger, _ := socketlog.NewLogger("./logs", socketlog.RELEASE)
//	logger.Log("AuthService", socketlog.INFO, "Client connected")
//	defer logger.Close()
//
// Designed for performance and clarity in networked systems.
package socketlog

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// LogType represents different severity levels for log messages
type LogType uint8

const (
	INFO    LogType = iota // Informational messages (normal operations)
	WARNING                // Warnings (potential issues, not errors)
	ERROR                  // Errors (something went wrong)
	DEBUG                  // Debugging messages (verbose output for development)
)

// LogMode controls how and where logs are output
type LogMode uint8

const (
	DEV     LogMode = iota // Console only, all logs
	RELEASE                // Console + file, no DEBUG
	VERBOSE                // Console + file, all logs
	HIDDEN                 // Console + file, INFO and ERROR only
)

// ANSI color codes for console output
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
)

// Application constants
const (
	module     = "[SocketHub]"
	timeFormat = "2006-01-02 15:04:05"
	bufferSize = 4096 // Buffer size for file writes
)

// Pre-computed string constants for log types
var logTypeStrings = [4]string{
	INFO:    "INFO",
	WARNING: "WARNING",
	ERROR:   "ERROR",
	DEBUG:   "DEBUG",
}

// Logger handles all logging operations, including thread safety, output mode, and file management.
type Logger struct {
	mu      sync.RWMutex    // RWMutex for thread-safe access (read/write)
	logFile *os.File        // File handle for log file (if enabled)
	writer  *bufio.Writer   // Buffered writer for file output
	colors  [4]string       // Array of colors indexed by LogType
	mode    LogMode         // Current logging mode (DEV, RELEASE, etc.)
	closed  bool            // Indicates if the logger has been closed
	sb      strings.Builder // Pre-allocated string builder for efficiency
}

// NewLogger creates a new logger instance with specified directory and mode
func NewLogger(logDir string, mode LogMode) (*Logger, error) {
	l := &Logger{
		mode: mode,
		colors: [4]string{
			INFO:    Green,
			WARNING: Yellow,
			ERROR:   Red,
			DEBUG:   Blue,
		},
	}

	// Pre-allocate string builder capacity
	l.sb.Grow(256)

	// Create log file for non-DEV modes
	if mode != DEV {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("[SocketLog] failed to create log directory: %w", err)
		}

		filename := fmt.Sprintf("%s/%s.log", logDir, time.Now().Format("20060102_150405"))
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("[SocketLog] failed to create log file: %w", err)
		}

		l.logFile = file
		l.writer = bufio.NewWriterSize(file, bufferSize)
	}

	return l, nil
}

// Log writes a log message based on the current mode and log type
func (l *Logger) Log(consumer string, logType LogType, message string) {
	// Fast path: check if we should log at all (no lock needed for read-only data)
	shouldPrint, shouldSave := l.shouldLogFast(l.mode, logType)
	if !shouldPrint && !shouldSave {
		return
	}

	// Pre-format timestamp before any locking
	timestamp := time.Now().Format(timeFormat)

	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}

	if shouldPrint {
		l.printToConsole(timestamp, consumer, logType, message)
	}
	if shouldSave {
		l.saveToFile(timestamp, consumer, logType, message)
	}
	l.mu.Unlock()
}

// Pre-computed complete lookup table [mode][logType] -> [shouldPrint, shouldSave]
var completeLogBehavior = [4][4][2]bool{
	// DEV mode
	{
		INFO:    {true, false},
		WARNING: {true, false},
		ERROR:   {true, false},
		DEBUG:   {true, false},
	},
	// RELEASE mode
	{
		INFO:    {true, true},
		WARNING: {true, true},
		ERROR:   {true, true},
		DEBUG:   {false, false}, // DEBUG disabled in RELEASE
	},
	// VERBOSE mode
	{
		INFO:    {true, true},
		WARNING: {true, true},
		ERROR:   {true, true},
		DEBUG:   {true, true},
	},
	// HIDDEN mode
	{
		INFO:    {true, true},
		WARNING: {false, false}, // WARNING disabled in HIDDEN
		ERROR:   {true, true},
		DEBUG:   {false, false}, // DEBUG disabled in HIDDEN
	},
}

// shouldLogFast uses complete lookup table for O(1) decision making
func (l *Logger) shouldLogFast(mode LogMode, logType LogType) (shouldPrint, shouldSave bool) {
	if mode >= 4 || logType >= 4 {
		return false, false
	}

	behavior := completeLogBehavior[mode][logType]
	return behavior[0], behavior[1]
}

// printToConsole outputs colored log message to console using pre-allocated builder
func (l *Logger) printToConsole(timestamp, consumer string, logType LogType, message string) {
	l.sb.Reset()
	// Pre-calculate capacity to avoid reallocations
	capacity := 1 + len(l.colors[logType]) + len(logTypeStrings[logType]) + len(Reset) +
		3 + len(timestamp) + 2 + len(module) + 2 + len(consumer) + 2 + len(message) + 1
	if l.sb.Cap() < capacity {
		l.sb.Grow(capacity - l.sb.Cap())
	}

	l.sb.WriteByte('[')
	l.sb.WriteString(l.colors[logType])
	l.sb.WriteString(logTypeStrings[logType])
	l.sb.WriteString(Reset)
	l.sb.WriteString("] [")
	l.sb.WriteString(timestamp)
	l.sb.WriteString("] ")
	l.sb.WriteString(module)
	l.sb.WriteString(" [")
	l.sb.WriteString(consumer)
	l.sb.WriteString("] ")
	l.sb.WriteString(message)
	l.sb.WriteByte('\n')

	fmt.Print(l.sb.String())
}

// saveToFile writes log message to file using pre-allocated builder
func (l *Logger) saveToFile(timestamp, consumer string, logType LogType, message string) {
	if l.writer == nil {
		return
	}

	l.sb.Reset()
	// Pre-calculate capacity to avoid reallocations
	capacity := 1 + len(logTypeStrings[logType]) + 3 + len(timestamp) + 2 +
		len(module) + 2 + len(consumer) + 2 + len(message) + 1
	if l.sb.Cap() < capacity {
		l.sb.Grow(capacity - l.sb.Cap())
	}

	l.sb.WriteByte('[')
	l.sb.WriteString(logTypeStrings[logType])
	l.sb.WriteString("] [")
	l.sb.WriteString(timestamp)
	l.sb.WriteString("] ")
	l.sb.WriteString(module)
	l.sb.WriteString(" [")
	l.sb.WriteString(consumer)
	l.sb.WriteString("] ")
	l.sb.WriteString(message)
	l.sb.WriteByte('\n')

	if _, err := l.writer.WriteString(l.sb.String()); err != nil {
		fmt.Printf("%s[ERROR]%s Failed to write log: %v\n", Red, Reset, err)
	}
}

// Flush forces any buffered log data to be written to disk
func (l *Logger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.writer != nil {
		return l.writer.Flush()
	}
	return nil
}

// Close safely closes the logger and releases resources
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true

	// Flush and close writer
	if l.writer != nil {
		if err := l.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush buffer: %w", err)
		}
	}

	// Close file
	if l.logFile != nil {
		if err := l.logFile.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
	}

	return nil
}

// SetColor allows changing the color for a specific log type
func (l *Logger) SetColor(logType LogType, color string) {
	if logType >= 4 {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Direct comparison instead of loop
	switch color {
	case Red, Green, Yellow, Blue, Magenta, Cyan:
		l.colors[logType] = color
	}
}
