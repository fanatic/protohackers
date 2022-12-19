package speeddaemon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
)

type Error struct { // Server -> Client
	Msg string
}

func (e *Error) Write(w io.Writer) error {
	log.Printf("6_speeddaemon --> Error{msg: %q}\n", e.Msg)
	if _, err := w.Write([]byte{0x10}); err != nil {
		return err
	}
	return writeString(w, e.Msg)
}

type Plate struct { // Client -> Server
	Plate     string
	Timestamp uint32
}

type Ticket struct { // Server -> Client
	Plate      string
	Road       uint16
	Mile1      uint16
	Timestamp1 uint32
	Mile2      uint16
	Timestamp2 uint32
	Speed      uint16 // (100x miles per hour)
}

func (t *Ticket) Write(w io.Writer) error {
	log.Printf("6_speeddaemon --> Ticket{plate: %q, road: %d, mile1: %d, timestamp1: %d, mile2: %d, timestamp2: %d, speed: %d}\n", t.Plate, t.Road, t.Mile1, t.Timestamp1, t.Mile2, t.Timestamp2, t.Speed)

	if _, err := w.Write([]byte{0x21}); err != nil {
		return err
	}

	if err := writeString(w, t.Plate); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, t.Road); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, t.Mile1); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, t.Timestamp1); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, t.Mile2); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, t.Timestamp2); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, t.Speed); err != nil {
		return err
	}

	return nil
}

type WantHeartbeat struct { // Client -> Server
	Interval uint32 // deciseconds
}

type Heartbeat struct{} // Server -> Client

func (h *Heartbeat) Write(w io.Writer) error {
	//log.Printf("6_speeddaemon --> Heartbeat{}\n")

	_, err := w.Write([]byte{0x41})
	return err
}

type IAmCamera struct { // Client -> Server
	Road  uint16
	Mile  uint16
	Limit uint16 // (miles per hour)
}

type IAmDispatcher struct { // Client -> Server
	Roads []uint16
}

// A string of characters in a length-prefixed format. A str is
// transmitted as a single u8 containing the string's length (0
// to 255), followed by that many bytes of u8, in order, containing
// ASCII character codes.
func readString(in io.Reader) (string, error) {
	b := make([]byte, 1)
	n, err := in.Read(b)
	if err != nil {
		return "", err
	} else if n != 1 {
		return "", fmt.Errorf("readString: l read %d characters, expected 1", n)
	}

	var length uint8 = b[0]

	b = make([]byte, length)
	n, err = in.Read(b)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	} else if n != int(length) {
		return "", fmt.Errorf("readString: read %d characters, expected %d", n, length)
	}

	// end of file or no error
	return string(b), err
}

type Flusher interface {
	// Flush sends any buffered data to the client.
	Flush() error
}

func writeString(w io.Writer, s string) error {
	var out []byte
	out = append(out, uint8(len(s)))
	out = append(out, []byte(s)...)
	_, err := w.Write(out)
	return err
}

func readUint8(in io.Reader) (uint8, error) {
	b := make([]byte, 1)
	n, err := in.Read(b)
	if err != nil {
		return 0, err
	} else if n != 1 {
		return 0, fmt.Errorf("readString: l read %d characters, expected 1", n)
	}

	return b[0], nil
}

func readUint16(in io.Reader) (uint16, error) {
	var b uint16
	err := binary.Read(in, binary.BigEndian, &b)
	if err != nil {
		return 0, err
	}

	return b, nil
}

func readUint32(in io.Reader) (uint32, error) {
	var b uint32
	err := binary.Read(in, binary.BigEndian, &b)
	if err != nil {
		return 0, err
	}

	return b, nil
}
