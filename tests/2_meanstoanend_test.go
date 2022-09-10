package tests

import (
	"context"
	"net"
	"testing"

	meanstoanend "github.com/fanatic/protohackers/2_meanstoanend"
	"github.com/stretchr/testify/require"
)

func TestLevel2Meanstoanend(t *testing.T) {
	ctx := context.Background()
	s, err := meanstoanend.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy-path", func(t *testing.T) {
		conn, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		// <-- I 12345 101
		_, err = conn.Write([]byte{0x49, 0x00, 0x00, 0x30, 0x39, 0x00, 0x00, 0x00, 0x65})
		require.NoError(t, err)

		// <-- I 12346 102
		_, err = conn.Write([]byte{0x49, 0x00, 0x00, 0x30, 0x3a, 0x00, 0x00, 0x00, 0x66})
		require.NoError(t, err)

		// <-- I 12347 100
		_, err = conn.Write([]byte{0x49, 0x00, 0x00, 0x30, 0x3b, 0x00, 0x00, 0x00, 0x64})
		require.NoError(t, err)

		// <-- I 40960 5
		_, err = conn.Write([]byte{0x49, 0x00, 0x00, 0xa0, 0x3b, 0x00, 0x00, 0x00, 0x05})
		require.NoError(t, err)

		// <-- Q 12288 16384
		_, err = conn.Write([]byte{0x51, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x40, 0x00})
		require.NoError(t, err)

		b := make([]byte, 4)
		_, err = conn.Read(b)
		require.NoError(t, err)

		// --> 101
		require.Equal(t, []byte{0x00, 0x00, 0x00, 0x65}, b)
	})
}
