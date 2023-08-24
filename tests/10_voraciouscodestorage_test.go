package tests

import (
	"context"
	"testing"

	voraciouscodestorage "github.com/fanatic/protohackers/10_voraciouscodestorage"
	"github.com/stretchr/testify/require"
)

func TestLevel10VoraciousCodeStorage(t *testing.T) {
	ctx := context.Background()
	s, err := voraciouscodestorage.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()
}
