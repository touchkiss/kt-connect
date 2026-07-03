package tun

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestShutdownStopsEngineOnlyOnce verifies Shutdown is idempotent: the real
// engine.Stop() calls log.Fatalf on a second stop, so the guard must ensure the
// stop func runs at most once no matter how many times Shutdown is called.
func TestShutdownStopsEngineOnlyOnce(t *testing.T) {
	origStop := stopEngine
	origStopped := stopped
	t.Cleanup(func() {
		stopEngine = origStop
		stopped = origStopped
	})

	calls := 0
	stopEngine = func() { calls++ }
	stopped = false

	c := &Cli{}
	require.NoError(t, c.Shutdown())
	require.NoError(t, c.Shutdown())
	require.NoError(t, c.Shutdown())

	require.Equal(t, 1, calls, "engine stop must run exactly once across repeated Shutdown calls")
}
