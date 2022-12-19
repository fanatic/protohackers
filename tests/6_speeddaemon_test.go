package tests

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	speeddaemon "github.com/fanatic/protohackers/6_speeddaemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevel6SpeedDaemon(t *testing.T) {
	ctx := context.Background()
	s, err := speeddaemon.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy-path", func(t *testing.T) {
		// Client 1: camera at mile 8
		client1, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client1.Close()

		_, err = client1.Write([]byte{0x80, 0x00, 0x7b, 0x00, 0x08, 0x00, 0x3c})
		require.NoError(t, err)

		_, err = client1.Write([]byte{0x20, 0x04, 0x55, 0x4e, 0x31, 0x58, 0x00, 0x00, 0x00, 0x00})
		require.NoError(t, err)

		// Client 2: camera at mile 9
		client2, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client2.Close()

		_, err = client2.Write([]byte{0x80, 0x00, 0x7b, 0x00, 0x09, 0x00, 0x3c})
		require.NoError(t, err)

		_, err = client2.Write([]byte{0x20, 0x04, 0x55, 0x4e, 0x31, 0x58, 0x00, 0x00, 0x00, 0x2d})
		require.NoError(t, err)

		// Client 3: ticket dispatcher
		client3, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client3.Close()

		time.Sleep(10 * time.Millisecond)

		_, err = client3.Write([]byte{0x81, 0x01, 0x00, 0x7b})
		require.NoError(t, err)

		b := make([]byte, 22)
		n, err := io.ReadFull(client3, b)
		require.NoError(t, err)
		assert.Equal(t, []byte{0x21, 0x04, 0x55, 0x4e, 0x31, 0x58, 0x00, 0x7b, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x2d, 0x1f, 0x40}, b[:n])
	})
}
