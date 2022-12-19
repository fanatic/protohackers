package tests

import (
	"context"
	"net"
	"testing"
	"time"

	database "github.com/fanatic/protohackers/4_database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevel4UnusualDatabaseProgram(t *testing.T) {
	ctx := context.Background()
	s, err := database.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy-path", func(t *testing.T) {
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		// Read "version"
		_, err = conn.Write([]byte("version"))
		require.NoError(t, err)

		received := make([]byte, 1000)
		n, err := conn.Read(received)
		require.NoError(t, err)
		assert.Equal(t, "version=fanatic/protohackers", string(received[:n]))

		// Insert "color"
		_, err = conn.Write([]byte("color=blue"))
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		// Retrieve "color"
		_, err = conn.Write([]byte("color"))
		require.NoError(t, err)

		n, err = conn.Read(received)
		require.NoError(t, err)
		assert.Equal(t, "color=blue", string(received[:n]))
	})

	t.Run("odd-path", func(t *testing.T) {
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		// Read "doesnotexist"
		_, err = conn.Write([]byte("doesnotexist"))
		require.NoError(t, err)

		received := make([]byte, 1000)
		n, err := conn.Read(received)
		require.NoError(t, err)
		assert.Equal(t, "doesnotexist=", string(received[:n]))

		// Insert "color"
		_, err = conn.Write([]byte("color="))
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		// Retrieve "color"
		_, err = conn.Write([]byte("color"))
		require.NoError(t, err)

		n, err = conn.Read(received)
		require.NoError(t, err)
		assert.Equal(t, "color=", string(received[:n]))
	})
}
