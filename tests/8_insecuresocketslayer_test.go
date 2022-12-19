package tests

import (
	"context"
	"testing"

	insecuresocketslayer "github.com/fanatic/protohackers/8_insecuresocketslayer"
	"github.com/stretchr/testify/require"
)

func TestLevel8InsecureSocketsLayer(t *testing.T) {
	ctx := context.Background()
	s, err := insecuresocketslayer.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy-path", func(t *testing.T) {

	})
}
