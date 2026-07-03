## Why

On macOS, `ktctl connect` (tun2socks mode) creates a `utun` device whose lifetime is bound to the tun2socks engine's file descriptor. Today the engine is stopped **only** by a signal-handler goroutine registered inside `tun.ToSocks`. The central teardown path `general.CleanupWorkspace()` never stops the engine, so any exit that races with (or bypasses) that goroutine — a fast `os.Exit` after cleanup, `SIGHUP` from closing the terminal (not in the notify set), or `ktctl clean` — leaves the `utun` device alive. Because `GetName()` deliberately picks `utunN+1`, each leaked device pushes the next connect to a higher number, and devices accumulate (utun0–utun7). Leaked devices keep their routes (macOS `RestoreRoute` is a no-op that assumes the device is destroyed), which can disrupt normal networking even when ktctl is not running.

## What Changes

- Add an explicit engine-shutdown operation to the tun `Tunnel` interface (e.g. `Shutdown()`), backed by `engine.Stop()`, that is idempotent and safe to call more than once.
- Call the new shutdown from `general.CleanupWorkspace()` (the single teardown path used by both normal exit and `ktctl clean`) so the `utun` device is destroyed deterministically, independent of the signal-handler goroutine.
- Make engine stop idempotent so the existing `ToSocks` signal goroutine and `CleanupWorkspace` cannot double-stop (tun2socks `engine.Stop()` calls `log.Fatalf` on error).
- Reset the cached `tunName` on shutdown so a subsequent connect in the same process (or the name-selection logic) does not skip numbers based on stale state.
- Only invoke shutdown when a tun device was actually created (skip `--disableTunDevice` sessions).

## Capabilities

### New Capabilities
- `tun-device-lifecycle`: Governs deterministic creation and teardown of the local tun (`utun`) device across connect setup, normal exit, and `ktctl clean`, ensuring no device or route leaks after a session ends.

### Modified Capabilities
<!-- none: no existing spec governs tun device teardown -->

## Impact

- `pkg/kt/service/tun/types.go` — new `Shutdown()` method on `Tunnel`.
- `pkg/kt/service/tun/tun.go` — implement idempotent `Shutdown()` calling `engine.Stop()`; guard the signal goroutine's stop through the same idempotent path.
- `pkg/kt/service/tun/tun_darwin.go` / `tun_linux.go` / `tun_windows.go` — reset cached `tunName`; per-platform no-op where the engine handles teardown.
- `pkg/kt/command/general/teardown.go` — `CleanupWorkspace()` calls `tun.Ins().Shutdown()` for connect sessions that created a tun device.
- No new dependencies. Behavior change is teardown-only; connect setup is unaffected.
