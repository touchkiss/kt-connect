package tun

import (
	"strings"
	"testing"

	"github.com/alibaba/kt-connect/pkg/kt/util"
	"github.com/stretchr/testify/require"
)

// TestResetTunNameForcesReScan verifies that after a tun device is destroyed,
// resetTunName clears the cached name so GetName re-scans interfaces instead of
// returning a stale (climbing) device number.
func TestResetTunNameForcesReScan(t *testing.T) {
	orig := tunName
	t.Cleanup(func() { tunName = orig })

	tunName = "utun99"
	resetTunName()
	require.Equal(t, "", tunName, "cached tun name must be cleared on reset")

	name := (&Cli{}).GetName()
	require.NotEqual(t, "utun99", name, "GetName must re-scan after reset, not return the stale name")
	require.True(t, strings.HasPrefix(name, util.TunNameMac), "expected a %s* device name, got %q", util.TunNameMac, name)
}
