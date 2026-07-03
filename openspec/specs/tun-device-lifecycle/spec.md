# tun-device-lifecycle Specification

## Purpose

Guarantee that local tun devices created by `ktctl connect` (tun2socks mode) are reliably destroyed during workspace cleanup, that engine shutdown is idempotent, that routes bound to the device are cleared, and that device-name selection does not accumulate across clean disconnects.

## Requirements

### Requirement: Tun device is destroyed during workspace cleanup

When a `connect` session created a local tun device (tun2socks mode, i.e. `--disableTunDevice` was not set), `general.CleanupWorkspace()` SHALL stop the tun2socks engine so the tun device is destroyed before the process exits. This teardown SHALL NOT depend on the signal-handler goroutine registered inside `ToSocks`.

#### Scenario: Normal exit destroys the tun device
- **WHEN** a user runs `ktctl connect` in tun2socks mode and then terminates it (Ctrl+C or the process returns and `CleanupWorkspace()` runs)
- **THEN** the tun2socks engine is stopped and the `utun`/`kt0` device created for that session no longer appears in `ifconfig` / `ip link`

#### Scenario: ktctl clean destroys a stale tun device path
- **WHEN** `CleanupWorkspace()` runs for a connect component that created a tun device
- **THEN** the engine shutdown is invoked exactly once as part of cleanup

#### Scenario: Disable-tun sessions skip shutdown
- **WHEN** a connect session ran with `--disableTunDevice` (no tun device created)
- **THEN** cleanup SHALL NOT attempt to stop the tun engine

### Requirement: Engine shutdown is idempotent

The tun `Tunnel` interface SHALL expose a `Shutdown()` operation that stops the tun2socks engine at most once. Because both the `ToSocks` signal-handler goroutine and `CleanupWorkspace()` may request shutdown, calling shutdown more than once SHALL be safe and SHALL NOT abort the process (tun2socks `engine.Stop()` calls `log.Fatalf` on error when the engine is already stopped).

#### Scenario: Double shutdown does not crash
- **WHEN** the signal goroutine stops the engine and `CleanupWorkspace()` also calls `Shutdown()` (or vice versa)
- **THEN** the second call is a no-op and the process exits normally without a fatal error

### Requirement: No dangling routes remain after teardown

After the tun device is destroyed, the routes that pointed at it SHALL no longer redirect host traffic, so that normal networking works when ktctl is not running.

#### Scenario: Routes cleared with the device
- **WHEN** the tun device for a session is destroyed during cleanup
- **THEN** the cluster CIDR routes that were bound to that device are no longer present in the host route table

### Requirement: Device name selection does not accumulate across sessions

On platforms where the tun device name is derived from existing interfaces (macOS `utunN`), a completed and cleaned-up session SHALL NOT cause the next session to select a higher device number due to a leaked device or stale cached name.

#### Scenario: Sequential connects reuse a low device number
- **WHEN** a user runs `ktctl connect`, exits cleanly, and runs `ktctl connect` again
- **THEN** the second session does not observe a leaked device from the first session and does not monotonically climb (e.g. utun4 → utun5 → utun6) across clean disconnects
