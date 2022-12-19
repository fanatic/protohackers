package tests

import (
	"context"
	"testing"

	mobinthemiddle "github.com/fanatic/protohackers/5_mobinthemiddle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevel5MobInTheMiddle(t *testing.T) {
	ctx := context.Background()
	s, err := mobinthemiddle.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	tests := []struct {
		input string
		want  string
	}{
		{"7F1u3wSD5RbOHQmupo9nx4TnhQ asdf", "MATCH asdf"},
		{"asdf 7iKDZEwPZSqIvDnHvVN2r0hUWXD5rHX", "asdf MATCH"},
		{"7LOrwbDlS8NujgjddyogWgIM93MV5N2VR", "MATCH"},
		{"asdf 7adNeSwJkMakpEcln9HEtthSRtxdmEHOT8T asdf", "asdf MATCH asdf"},
	}

	for _, tc := range tests {
		t.Run("rewrite", func(t *testing.T) {
			got, _ := s.Boguscoin.Replace(tc.input, "MATCH", -1, -1)

			assert.Equal(t, tc.want, got)
		})
	}
}
