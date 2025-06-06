// Package protocol provides CRC32 checksum utilities for SocketHub protocol integrity verification.
// Uses IEEE polynomial for standard CRC32 calculation over header and payload data.
package protocol

import "hash/crc32"

// CRC32Table is the pre-computed IEEE polynomial table for efficient CRC32 calculations.
var CRC32Table = crc32.MakeTable(crc32.IEEE)

// Checksum calculates CRC32 over headerBytes (everything before Checksum field) + payload.
func Checksum(payload []byte) uint32 {
	crc := crc32.New(CRC32Table)
	crc.Write(payload)
	return crc.Sum32()
}
