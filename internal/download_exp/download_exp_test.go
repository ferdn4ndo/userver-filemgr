package download_exp

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecondsFromSize(t *testing.T) {
	assert.Equal(t, int(math.Round(math.Pow(math.Log10(156144), 4.0))), SecondsFromSize(156144, 4.0))
	assert.Greater(t, SecondsFromSize(1_000_000_000, 4.25), 1000)
}

func TestSecondsFromEnv(t *testing.T) {
	t.Setenv("DOWNLOAD_EXP_BYTES_SECS_RATIO", "4")
	v := SecondsFromEnv(156144, 0)
	assert.Equal(t, int(math.Round(math.Pow(math.Log10(156144), 4.0))), v)
}
