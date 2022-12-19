package speeddaemon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpeed(t *testing.T) {
	tests := []struct {
		m1    uint16
		t1    uint32
		m2    uint16
		t2    uint32
		speed uint16
	}{
		{m1: 10, t1: 17007084, m2: 820, t2: 16964085, speed: 6782},
	}

	for _, tc := range tests {
		t.Run("speed", func(t *testing.T) {
			got := speed(tc.m2, tc.m1, tc.t2, tc.t1)

			assert.Equal(t, tc.speed, got)
		})
	}
}
