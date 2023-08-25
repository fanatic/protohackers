package pestcontrol

import (
	"io"
)

func writeMessage(w io.Writer, typ byte, data []byte) error {
	if _, err := w.Write([]byte{typ}); err != nil {
		return err
	}

	msgLength := uint32(len(data) + 1 + 4 + 1)
	if _, err := w.Write(uint32ToBytes(msgLength)); err != nil {
		return err
	}

	_, err := w.Write(data)
	if err != nil {
		return err
	}

	cksum := checksum(typ, uint32ToBytes(msgLength), data)
	if _, err := w.Write([]byte{cksum}); err != nil {
		return err
	}

	return nil
}

func stringToBytes(s string) []byte {
	return append(uint32ToBytes(uint32(len(s))), []byte(s)...)
}

func uint32ToBytes(i uint32) []byte {
	return []byte{byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)}
}
