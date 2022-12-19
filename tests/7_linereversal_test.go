package tests

import (
	"context"
	"testing"

	linereversal "github.com/fanatic/protohackers/7_linereversal"
	"github.com/stretchr/testify/require"
)

func TestLevel7LineReversal(t *testing.T) {
	ctx := context.Background()
	s, err := linereversal.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy-path", func(t *testing.T) {

	})
}
