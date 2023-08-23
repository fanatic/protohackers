package insecuresocketslayer

import (
	"errors"
	"io"
)

type CipherReaderWriter struct {
	rw         io.ReadWriter
	inOffset   int
	outOffset  int
	cipherSpec []byte
}

func NewCipherReaderWriter(rw io.ReadWriter, cipherSpec []byte) *CipherReaderWriter {
	return &CipherReaderWriter{rw: rw, cipherSpec: cipherSpec}
}

func (crw *CipherReaderWriter) Validate() error {
	testString := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	testString2 := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	Encode(crw.cipherSpec, testString, 0, false)
	for i := range testString {
		if testString[i] != testString2[i] {
			return nil
		}
	}

	return errors.New("cipher spec is invalid")
}

func (crw *CipherReaderWriter) Read(p []byte) (n int, err error) {
	n, err = crw.rw.Read(p)
	if err != nil {
		return
	}
	Decode(crw.cipherSpec, p[:n], crw.inOffset, false)
	crw.inOffset += n
	return
}

func (crw *CipherReaderWriter) Write(p []byte) (n int, err error) {
	Encode(crw.cipherSpec, p, crw.outOffset, true)
	crw.outOffset += len(p)
	return crw.rw.Write(p)
}

func Decode(cipherSpec []byte, msg []byte, pos int, reverse bool) {
	//fmt.Printf("Decode(%x (%q), %d, %t)\n", msg, msg, pos, reverse)

	reversedCipherSpecs := make([]byte, len(cipherSpec))
	for i := 0; i < len(cipherSpec); i++ {
		if cipherSpec[i] == 0x02 || cipherSpec[i] == 0x04 {
			reversedCipherSpecs[len(cipherSpec)-i-2] = cipherSpec[i]
			reversedCipherSpecs[len(cipherSpec)-i-1] = cipherSpec[i+1]
			i++ // skip operand
		} else {
			reversedCipherSpecs[len(cipherSpec)-i-1] = cipherSpec[i]
		}
	}

	Encode(reversedCipherSpecs, msg, pos, reverse)
}

func Encode(cipherSpec []byte, msg []byte, pos int, reverse bool) {
	//fmt.Printf("Encode(%x (%q), %d, %t)\n", msg, msg, pos, reverse)
	// Apply each operation in the cipher spec to the message
	for i := 0; i < len(cipherSpec); i++ {
		applyCipher(cipherSpec, msg, pos, reverse, i)
		if cipherSpec[i] == 0x02 || cipherSpec[i] == 0x04 {
			i++ // skip operand
		}
	}
	//fmt.Printf("-> %x (%q)\n", msg, msg)
}

func applyCipher(cipherSpec []byte, msg []byte, pos int, reverse bool, i int) {
	switch cipherSpec[i] {
	case 0x01:
		// Reverse the order of bits in the byte, so the least-significant bit becomes the most-significant bit, the 2nd-least-significant becomes the 2nd-most-significant, and so on.
		for j := range msg {
			msg[j] = reverseBits(msg[j])
		}
	case 0x02:
		// XOR the byte by the value N. N is the next byte in the cipher spec.
		for j := range msg {
			msg[j] ^= cipherSpec[i+1]
		}
	case 0x03:
		// XOR the byte by its position in the stream, starting from 0.
		for j := range msg {
			msg[j] ^= byte(pos + j)
		}
	case 0x04:
		// Add N to the byte, modulo 256. Note that 0 is a valid value for N, and addition wraps, so that 255+1=0, 255+2=1, and so on.
		for j := range msg {
			msg[j] = addN(msg[j], int(cipherSpec[i+1]), reverse)
		}
	case 0x05:
		// Add the position in the stream to the byte, modulo 256, starting from 0. Addition wraps, so that 255+1=0, 255+2=1, and so on.
		for j := range msg {
			msg[j] = addN(msg[j], pos+j, reverse)
		}
	}
}

// Add N to the byte, modulo 256. Note that 0 is a valid value for N, and addition wraps, so that 255+1=0, 255+2=1, and so on.
func addN(b byte, n int, reverse bool) byte {
	reverseHelper := -1
	if reverse {
		reverseHelper = 1
	}
	return byte((int(b) + (n * reverseHelper)) % 256)
}

// Reverse the order of bits in the byte, so the least-significant bit becomes the most-significant bit, the 2nd-least-significant becomes the 2nd-most-significant, and so on.
func reverseBits(b byte) byte {
	var out byte
	for i := 0; i < 8; i++ {
		out <<= 1
		out |= b & 1
		b >>= 1
	}
	return out
}
