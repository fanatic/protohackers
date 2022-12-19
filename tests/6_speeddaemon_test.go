package tests

import (
	"context"
	"testing"

	speeddaemon "github.com/fanatic/protohackers/6_speeddaemon"
	"github.com/stretchr/testify/require"
)

func TestLevel6SpeedDaemon(t *testing.T) {
	ctx := context.Background()
	s, err := speeddaemon.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy-path", func(t *testing.T) {

	})
}
