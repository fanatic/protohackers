package pestcontrol

import (
	"bytes"
	"fmt"
	"io"
)

func ReadMessage(r io.Reader) (byte, []byte, error) {
	// Read message type
	msgTypeRaw := make([]byte, 1)
	_, err := io.ReadFull(r, msgTypeRaw)
	if err != nil {
		return 0x00, nil, fmt.Errorf("reading message type: %w", err)
	}
	msgType := msgTypeRaw[0]

	// Read message length (u32)
	msgLengthRaw := make([]byte, 4)
	_, err = io.ReadFull(r, msgLengthRaw)
	if err != nil {
		return 0x00, nil, fmt.Errorf("reading message length: %w", err)
	}
	msgLength := bytesToUint32(msgLengthRaw)

	if msgLength > 1_000_000 {
		// Discard the rest of the message (takes too long)
		//io.CopyN(io.Discard, r, int64(msgLength-1-4))
		return 0x00, nil, fmt.Errorf("message too long: %d", msgLength)
	}

	// Read message content
	msgContents := make([]byte, msgLength-1-4-1)
	_, err = io.ReadFull(r, msgContents)
	if err != nil {
		return 0x00, nil, fmt.Errorf("reading message contents: %w", err)
	}

	// Read message checksum
	msgChecksum := make([]byte, 1)
	_, err = io.ReadFull(r, msgChecksum)
	if err != nil {
		return 0x00, nil, fmt.Errorf("reading message checksum: %w", err)
	}

	// Verify checksum
	ck := checksum(msgType, msgLengthRaw, msgContents)
	if ck != msgChecksum[0] {
		return 0x00, nil, fmt.Errorf("invalid checksum: %d != %d", ck, msgChecksum[0])
	}

	// fmt.Printf("!!! %x %d %d %d\n", msgType, msgLength, len(msgContents), msgChecksum[0])
	// fmt.Printf("### %x %x %x %x\n", msgTypeRaw, msgLengthRaw, msgContents, msgChecksum)

	return msgType, msgContents, nil
}

func readString(r *bytes.Buffer) (string, error) {
	length, err := readU32(r)
	if err != nil {
		return "", err
	}
	if r.Len() < int(length) {
		return "", fmt.Errorf("string length %d exceeds remaining buffer length %d", length, r.Len())
	}
	b := make([]byte, length)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func readU32(r *bytes.Buffer) (uint32, error) {
	if r.Len() < 4 {
		return 0, fmt.Errorf("buffer length %d less than 4", r.Len())
	}
	b := make([]byte, 4)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return 0, err
	}
	return bytesToUint32(b), nil
}

func bytesToUint32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func checksum(msgType byte, msgLengthRaw, msgContents []byte) byte {
	sum := int64(msgType) + int64(msgLengthRaw[0]) + int64(msgLengthRaw[1]) + int64(msgLengthRaw[2]) + int64(msgLengthRaw[3])
	for i := 0; i < len(msgContents); i++ {
		sum += int64(msgContents[i])
	}
	return byte(256 - sum%256)
}

type Observation struct {
	Species string
	Count   uint32
}

func readObservationArray(r *bytes.Buffer) ([]Observation, error) {
	length, err := readU32(r)
	if err != nil {
		return nil, err
	}
	arr := make([]Observation, length)
	for i := uint32(0); i < length; i++ {
		species, err := readString(r)
		if err != nil {
			return nil, err
		}
		count, err := readU32(r)
		if err != nil {
			return nil, err
		}
		arr[i] = Observation{Species: species, Count: count}
	}

	return arr, nil
}

type Target struct {
	Species string
	Min     uint32
	Max     uint32
}

func readTargetArray(r *bytes.Buffer) ([]Target, error) {
	length, err := readU32(r)
	if err != nil {
		return nil, err
	}
	arr := make([]Target, length)
	for i := uint32(0); i < length; i++ {
		species, err := readString(r)
		if err != nil {
			return nil, err
		}
		min, err := readU32(r)
		if err != nil {
			return nil, err
		}
		max, err := readU32(r)
		if err != nil {
			return nil, err
		}
		arr[i] = Target{Species: species, Min: min, Max: max}
	}

	return arr, nil
}
