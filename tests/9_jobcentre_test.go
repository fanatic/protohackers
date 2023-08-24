package tests

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	jobcentre "github.com/fanatic/protohackers/9_jobcentre"
	"github.com/stretchr/testify/require"
)

func TestLevel9JobCentre(t *testing.T) {
	ctx := context.Background()
	s, err := jobcentre.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy path", func(t *testing.T) {
		client, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client.Close()

		assertRequest(t, client,
			`{"request":"put","queue":"queue1","job":{"title":"example-job"},"pri":123}`,
			`{"status":"ok","id":1}`,
		)

		assertRequest(t, client,
			`{"request":"get","queues":["queue1"]}`,
			`{"status":"ok","id":1,"job":{"title":"example-job"},"pri":123,"queue":"queue1"}`,
		)

		assertRequest(t, client,
			`{"request":"abort","id":1}`,
			`{"status":"ok"}`,
		)

		assertRequest(t, client,
			`{"request":"get","queues":["queue1"]}`,
			`{"status":"ok","id":1,"job":{"title":"example-job"},"pri":123,"queue":"queue1"}`,
		)

		assertRequest(t, client,
			`{"request":"get","queues":["queue1"]}`,
			`{"status":"no-job"}`,
		)

		assertRequest(t, client,
			`{"request":"delete","id":1}`,
			`{"status":"ok"}`,
		)

		assertRequest(t, client,
			`{"request":"get","queues":["queue1"]}`,
			`{"status":"no-job"}`,
		)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			assertRequest(t, client,
				`{"request":"get","queues":["queue1", "queue2"],"wait":true}`,
				`{"status":"ok","id":2,"job":{"title":"example-job2"},"pri":1,"queue":"queue2"}`,
			)
			wg.Done()
		}()

		// wait for request to be fired off
		time.Sleep(100 * time.Millisecond)

		client2, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		assertRequest(t, client2,
			`{"request":"put","queue":"queue2","job":{"title":"example-job2"},"pri":1}`,
			`{"status":"ok","id":2}`,
		)
		client2.Close()

		wg.Wait()

	})

	t.Run("bad request", func(t *testing.T) {
		client, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer client.Close()

		assertRequest(t, client,
			`{}`,
			`{"status":"error","error":"unsupported method"}`,
		)

		assertRequest(t, client,
			`{`,
			`{"status":"error","error":"unexpected end of JSON input"}`,
		)

	})
}

func assertRequest(t *testing.T, conn net.Conn, request, expectedJson string) {
	_, err := conn.Write([]byte(request + "\n"))
	require.NoError(t, err)

	var actual jobcentre.Response
	err = json.NewDecoder(conn).Decode(&actual)
	require.NoError(t, err)

	var expected jobcentre.Response
	err = json.Unmarshal([]byte(expectedJson), &expected)
	require.NoError(t, err)

	require.Equal(t, expected, actual)
}
