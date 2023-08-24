package tests

import (
	"bufio"
	"context"
	"net"
	"testing"

	voraciouscodestorage "github.com/fanatic/protohackers/10_voraciouscodestorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevel10VoraciousCodeStorage(t *testing.T) {
	ctx := context.Background()
	s, err := voraciouscodestorage.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("bad method", func(t *testing.T) {
		conn, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		scanner := bufio.NewScanner(conn)

		scanner.Scan()
		require.Equal(t, "READY", scanner.Text())

		_, err = conn.Write([]byte("\n"))
		require.NoError(t, err)

		scanner.Scan()
		assert.Equal(t, "ERR illegal method:", scanner.Text())
	})

	t.Run("help", func(t *testing.T) {
		conn, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		scanner := bufio.NewScanner(conn)

		scanner.Scan()
		require.Equal(t, "READY", scanner.Text())

		_, err = conn.Write([]byte("help\n"))
		require.NoError(t, err)

		scanner.Scan()
		assert.Equal(t, "OK usage: HELP|GET|PUT|LIST", scanner.Text())

		scanner.Scan()
		assert.Equal(t, "READY", scanner.Text())
	})

	t.Run("list", func(t *testing.T) {
		conn, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		scanner := bufio.NewScanner(conn)

		scanner.Scan()
		require.Equal(t, "READY", scanner.Text())

		t.Run("empty field", func(t *testing.T) {
			_, err = conn.Write([]byte("list\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "ERR usage: LIST dir", scanner.Text())
		})

		t.Run("bad dir name", func(t *testing.T) {
			_, err = conn.Write([]byte("list a\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "ERR illegal dir name", scanner.Text())
		})

		t.Run("empty dir", func(t *testing.T) {
			_, err = conn.Write([]byte("list /\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "OK 0", scanner.Text())
			scanner.Scan()
			assert.Equal(t, "READY", scanner.Text())
		})

		t.Run("non-empty dir", func(t *testing.T) {
			_, err = conn.Write([]byte("put /a 0\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "OK r1", scanner.Text())
			scanner.Scan()
			assert.Equal(t, "READY", scanner.Text())

			_, err = conn.Write([]byte("put /b/c 0\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "OK r1", scanner.Text())
			scanner.Scan()
			assert.Equal(t, "READY", scanner.Text())

			_, err = conn.Write([]byte("list /\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "OK 2", scanner.Text())
			scanner.Scan()
			assert.Equal(t, "a r1", scanner.Text())
			scanner.Scan()
			assert.Equal(t, "b/ DIR", scanner.Text())
			scanner.Scan()
			assert.Equal(t, "READY", scanner.Text())
		})

		t.Run("list subdirectory", func(t *testing.T) {
			_, err = conn.Write([]byte("list /b\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "OK 1", scanner.Text())
			scanner.Scan()
			assert.Equal(t, "c r1", scanner.Text())
			scanner.Scan()
			assert.Equal(t, "READY", scanner.Text())
		})
	})

	t.Run("put", func(t *testing.T) {
		conn, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		scanner := bufio.NewScanner(conn)

		scanner.Scan()
		require.Equal(t, "READY", scanner.Text())

		t.Run("put usage", func(t *testing.T) {
			_, err = conn.Write([]byte("put a\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "ERR usage: PUT file length newline data", scanner.Text())
		})

		t.Run("put empty file", func(t *testing.T) {
			_, err = conn.Write([]byte("put /c 0\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "OK r1", scanner.Text())
			scanner.Scan()
			require.Equal(t, "READY", scanner.Text())
		})

		t.Run("put file", func(t *testing.T) {
			_, err = conn.Write([]byte("put /c 5\n"))
			require.NoError(t, err)
			_, err = conn.Write([]byte("12345"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "OK r2", scanner.Text())
			scanner.Scan()
			require.Equal(t, "READY", scanner.Text())
		})

		t.Run("writing same data doesn't change revision", func(t *testing.T) {
			_, err = conn.Write([]byte("put /c 5\n"))
			require.NoError(t, err)
			_, err = conn.Write([]byte("12345"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "OK r2", scanner.Text())
			scanner.Scan()
			require.Equal(t, "READY", scanner.Text())
		})

		t.Run("put big file", func(t *testing.T) {
			_, err = conn.Write([]byte("put /c 50000\n"))
			require.NoError(t, err)
			for i := 0; i < 50000; i++ {
				_, err = conn.Write([]byte("a"))
				require.NoError(t, err)
			}

			scanner.Scan()
			assert.Equal(t, "OK r3", scanner.Text())
			scanner.Scan()
			require.Equal(t, "READY", scanner.Text())
		})
	})

	t.Run("get", func(t *testing.T) {
		conn, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		scanner := bufio.NewScanner(conn)

		scanner.Scan()
		require.Equal(t, "READY", scanner.Text())

		_, err = conn.Write([]byte("put /d 1\n1"))
		require.NoError(t, err)

		scanner.Scan()
		assert.Equal(t, "OK r1", scanner.Text())
		scanner.Scan()
		assert.Equal(t, "READY", scanner.Text())

		t.Run("get usage", func(t *testing.T) {

			_, err = conn.Write([]byte("get\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "ERR usage: GET file [revision]", scanner.Text())
		})

		t.Run("get file", func(t *testing.T) {
			_, err = conn.Write([]byte("get /d\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "OK 1", scanner.Text())

			scanner.Scan()
			assert.Equal(t, "1READY", scanner.Text())
		})

		t.Run("get file with revision", func(t *testing.T) {
			_, err = conn.Write([]byte("get /d r1\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "OK 1", scanner.Text())

			scanner.Scan()
			assert.Equal(t, "1READY", scanner.Text())
		})

		t.Run("get file with bad revision", func(t *testing.T) {
			_, err = conn.Write([]byte("get /d r0\n"))
			require.NoError(t, err)

			scanner.Scan()
			assert.Equal(t, "ERR illegal revision", scanner.Text())
		})
	})

	// t.Run("crashy cases", func(t *testing.T) {
	// 	conn, err := net.Dial("tcp", s.Addr)
	// 	require.NoError(t, err)
	// 	defer conn.Close()

	// 	scanner := bufio.NewScanner(conn)

	// 	scanner.Scan()
	// 	require.Equal(t, "READY", scanner.Text())

	// 	t.Run("", func(t *testing.T) {
	// 		_, err = conn.Write([]byte("GET /test.txt r0\n"))
	// 		require.NoError(t, err)

	// 		scanner.Scan()
	// 		assert.Equal(t, "OK 1", scanner.Text())

	// 		scanner.Scan()
	// 		assert.Equal(t, "1READY", scanner.Text())
	// 	})

	// })
}
