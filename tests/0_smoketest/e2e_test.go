package smoketest_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"testing"

	smoketest "github.com/fanatic/protohackers/0_smoketest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevel0Smoketest(t *testing.T) {
	ctx := context.Background()
	s, err := smoketest.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	conn, err := net.Dial("tcp", s.Addr)
	require.NoError(t, err)
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "Hello, World!")
	require.NoError(t, err)

	// Close write-side of the connection
	if cw, ok := conn.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
	} else {
		t.Fatal("Can't half-close conneciton")
	}

	b, err := ioutil.ReadAll(conn)
	require.NoError(t, err)

	assert.Equal(t, "Hello, World!", string(b))
}
