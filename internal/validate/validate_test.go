package validate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPasswordValidation(t *testing.T) {
	require.Error(t, Password("short"))
	require.NoError(t, Password("long-enough"))
}

func TestMarketTimingValidation(t *testing.T) {
	now := time.Now()
	require.Error(t, MarketTiming(now, now.Add(-time.Minute), now.Add(time.Hour)))
	require.Error(t, MarketTiming(now, now.Add(2*time.Hour), now.Add(time.Hour)))
	require.NoError(t, MarketTiming(now, now.Add(time.Hour), now.Add(2*time.Hour)))
}

func TestEvidenceURLValidation(t *testing.T) {
	require.Error(t, URL("not-a-url"))
	require.NoError(t, URL("https://example.com/evidence"))
}
