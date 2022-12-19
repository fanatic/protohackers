package tests

import (
	"context"
	"testing"

	mobinthemiddle "github.com/fanatic/protohackers/5_mobinthemiddle"
	"github.com/stretchr/testify/require"
)

func TestLevel5MobInTheMiddle(t *testing.T) {
	ctx := context.Background()
	s, err := mobinthemiddle.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy-path", func(t *testing.T) {

	})
}
