package primetime_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"

	primetime "github.com/fanatic/protohackers/1_primetime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevel0Smoketest(t *testing.T) {
	ctx := context.Background()
	s, err := primetime.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy-path", func(t *testing.T) {
		conn, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		_, err = fmt.Fprintf(conn, `{"method":"isPrime","number":123}`+"\n")
		require.NoError(t, err)

		_, err = fmt.Fprintf(conn, `{"method":"isPrime","number":123.2}`+"\n")
		require.NoError(t, err)

		_, err = fmt.Fprintf(conn, `{"method":"isPrime","number":3}`+"\n")
		require.NoError(t, err)

		_, err = fmt.Fprintf(conn, `{`+"\n")
		require.NoError(t, err)

		_, err = fmt.Fprintf(conn, `{"method":"isPrime","number":"asdf"}`+"\n")
		require.NoError(t, err)

		_, err = fmt.Fprintf(conn, `{}`+"\n")
		require.NoError(t, err)

		// Close write-side of the connection
		if cw, ok := conn.(interface{ CloseWrite() error }); ok {
			cw.CloseWrite()
		} else {
			t.Fatal("Can't half-close conneciton")
		}

		b, err := ioutil.ReadAll(conn)
		require.NoError(t, err)

		assert.Equal(t,
			`{"method":"isPrime","prime":false}`+"\n"+
				`{"method":"isPrime","prime":false}`+"\n"+
				`{"method":"isPrime","prime":true}`+"\n"+
				`{"error":"unexpected end of JSON input"}`+"\n"+
				`{"error":"json: cannot unmarshal string into Go struct field Request.number of type float64"}`+"\n"+
				`{"error":"unsupported method"}`+"\n",
			string(b))
	})

	t.Run("lots-of-inputs", func(t *testing.T) {
		f, err := os.Open("inputs.txt")
		require.NoError(t, err)

		conn, err := net.Dial("tcp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			_, err = fmt.Fprintf(conn, `{"method":"isPrime","number":`+sc.Text()+`}`+"\n")
			require.NoError(t, err)

			var resp primetime.Response
			err = json.NewDecoder(conn).Decode(&resp)
			require.NoError(t, err)
			assert.Equal(t, "isPrime", resp.Method)
		}
	})
}
