package tests

import (
	"context"
	"testing"

	pestcontrol "github.com/fanatic/protohackers/11_pestcontrol"
	"github.com/stretchr/testify/require"
)

func TestLevel11PestControl(t *testing.T) {
	ctx := context.Background()
	s, err := pestcontrol.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()
}
