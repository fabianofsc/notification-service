package clock_test

import (
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/clock"
)

func TestUUIDv7Generator_ProducesValidUUIDv7(t *testing.T) {
	gen := clock.UUIDv7Generator{}
	id := gen.NewID()
	require.Equal(t, uuid.Version(7), id.Version())
}

func TestUUIDv7Generator_AThousandIDsSortInGenerationOrder(t *testing.T) {
	gen := clock.UUIDv7Generator{}

	const n = 1000
	generated := make([]string, n)
	for i := range generated {
		generated[i] = gen.NewID().String()
	}

	sorted := make([]string, n)
	copy(sorted, generated)
	sort.Strings(sorted)

	require.Equal(t, generated, sorted)
}

func TestUUIDv7Generator_NeverProducesTheNilUUID(t *testing.T) {
	gen := clock.UUIDv7Generator{}
	require.NotEqual(t, uuid.Nil, gen.NewID())
}
