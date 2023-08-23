package tests

import (
	"context"
	"io"
	"net"
	"testing"

	insecuresocketslayer "github.com/fanatic/protohackers/8_insecuresocketslayer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevel8InsecureSocketsLayer(t *testing.T) {
	ctx := context.Background()
	s, err := insecuresocketslayer.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("reversebits", func(t *testing.T) {
		client, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client.Close()

		// <-- reversebits
		sendBytes(t, client, []byte{0x01, 0x00})

		// <-- 4x dog,5x car\n
		sendBytes(t, client, []byte{0x2c, 0x1e, 0x04, 0x26, 0xf6, 0xe6, 0x34, 0xac, 0x1e, 0x04, 0xc6, 0x86, 0x4e, 0x50})

		// --> 5x car\n
		expectBytes(t, client, []byte{0xac, 0x1e, 0x04, 0xc6, 0x86, 0x4e, 0x50})
	})

	t.Run("xorpos", func(t *testing.T) {
		client, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client.Close()

		// <-- xorpos
		sendBytes(t, client, []byte{0x03, 0x00})

		// <-- 4x dog,5x car\n
		sendBytes(t, client, []byte{0x34, 0x79, 0x22, 0x67, 0x6b, 0x62, 0x2a, 0x32, 0x70, 0x29, 0x69, 0x6a, 0x7e, 0x07})

		// --> 5x car\n
		expectBytes(t, client, []byte{0x35, 0x79, 0x22, 0x60, 0x65, 0x77, 0xc})
	})

	t.Run("addpos", func(t *testing.T) {
		client, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client.Close()

		// <-- addpos
		sendBytes(t, client, []byte{0x05, 0x00})

		// <-- 4x dog,5x car\n
		sendBytes(t, client, []byte{0x34, 0x79, 0x22, 0x67, 0x73, 0x6c, 0x32, 0x3c, 0x80, 0x29, 0x6d, 0x6c, 0x7e, 0x17})

		// --> 5x car\n
		expectBytes(t, client, []byte{0x35, 0x79, 0x22, 0x66, 0x65, 0x77, 0x10})
	})

	t.Run("reversebits x2", func(t *testing.T) {
		client, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client.Close()

		// <-- reversebits
		sendBytes(t, client, []byte{0x01, 0x00})

		// <-- 4x dog,5x car\n
		oo := sendEncryptedBytes(t, client, "4x dog,5x car\n", []byte{0x01}, 0)

		// --> 5x car\n
		io := expectEncryptedBytes(t, client, "5x car\n", []byte{0x01}, 0)

		// <-- 3x rat,2x cat\n
		sendEncryptedBytes(t, client, "3x rat,2x cat\n", []byte{0x01}, oo)

		// --> 3x rat\n
		expectEncryptedBytes(t, client, "3x rat\n", []byte{0x01}, io)
	})

	t.Run("happy-path", func(t *testing.T) {
		client, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client.Close()

		// <-- xor(123),addpos,reversebits
		sendBytes(t, client, []byte{0x02, 0x7b, 0x05, 0x01, 0x00})

		// <-- 4x dog,5x car\n
		sendBytes(t, client, []byte{0xf2, 0x20, 0xba, 0x44, 0x18, 0x84, 0xba, 0xaa, 0xd0, 0x26, 0x44, 0xa4, 0xa8, 0x7e})

		// --> 5x car\n
		expectBytes(t, client, []byte{0x72, 0x20, 0xba, 0xd8, 0x78, 0x70, 0xee})

		// <-- 3x rat,2x cat\n
		sendBytes(t, client, []byte{0x6a, 0x48, 0xd6, 0x58, 0x34, 0x44, 0xd6, 0x7a, 0x98, 0x4e, 0x0c, 0xcc, 0x94, 0x31})

		// --> 3x rat\n
		expectBytes(t, client, []byte{0xf2, 0xd0, 0x26, 0xc8, 0xa4, 0xd8, 0x7e})
	})

	t.Run("invalid cipher spec", func(t *testing.T) {
		client, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client.Close()

		// <-- invalid
		sendBytes(t, client, []byte{0x00})

		_, err = client.Read(make([]byte, 1))
		require.Error(t, err)
	})
}

func sendBytes(t *testing.T, conn net.Conn, msg []byte) {
	_, err := conn.Write(msg)
	require.NoError(t, err)
}

func sendEncryptedBytes(t *testing.T, conn net.Conn, msg string, cipherSpec []byte, offset int) int {
	encrypted := []byte(msg)
	insecuresocketslayer.Encode(cipherSpec, encrypted, offset, false)
	_, err := conn.Write(encrypted)
	require.NoError(t, err)
	return len(msg)
}

func expectBytes(t *testing.T, conn net.Conn, msg []byte) {
	received := make([]byte, len(msg))
	n, err := io.ReadFull(conn, received)
	require.NoError(t, err)
	assert.Equal(t, msg, received[:n]) // will be truncated to expected message length
}

func expectEncryptedBytes(t *testing.T, conn net.Conn, msg string, cipherSpec []byte, offset int) int {
	received := make([]byte, len(msg))
	n, err := io.ReadFull(conn, received)
	require.NoError(t, err)
	encoded := received[:n] // will be truncated to expected message length
	insecuresocketslayer.Encode(cipherSpec, encoded, offset, true)
	assert.Equal(t, msg, string(encoded))
	return len(msg)
}
