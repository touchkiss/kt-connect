## Context

`ktctl connect` in tun2socks mode creates a local tun device via the `xjasonlyu/tun2socks/v2` engine:

- `pkg/kt/service/tun/tun.go:ToSocks` builds an `engine.Key`, calls `engine.Insert` + `engine.Start`, then spawns a goroutine that waits on `os.Interrupt`/`SIGTERM` and calls `engine.Stop`.
- On macOS the device is `utunN` where N is chosen by `tun_darwin.go:GetName` as `max(existing utun numbers) + 1`, cached in a package-level `tunName`. On Linux it is the fixed `kt0`; on Windows it is a wintun adapter.
- The device's lifetime is bound to the engine's tun file descriptor: when `engine.Stop` runs (or the process dies), the fd closes and the OS destroys the device and its routes. `RestoreRoute` on all platforms is a no-op that assumes "route auto-removed when tun device destroyed."

The central teardown path is `general.CleanupWorkspace()` (`pkg/kt/command/general/teardown.go`), called from `cmd/ktctl/main.go:66` after `rootCmd.Execute()` returns, and reused by `ktctl clean`. It cleans DNS, hosts, cluster resources — but **never stops the tun engine**. Engine stop today happens *only* in the `ToSocks` signal goroutine.

The result: whenever the engine is not cleanly stopped before the fd is abandoned — signal races, `SIGHUP` (terminal close, not in the notify set), or code paths that reach `CleanupWorkspace` without the goroutine firing — the `utun` device and its routes leak. `GetName`'s `max+1` strategy then climbs the device number every run, and leaked routes can hijack traffic even after ktctl exits.

## Goals / Non-Goals

**Goals:**
- Destroy the tun device deterministically on session end via the single, always-run `CleanupWorkspace()` path, not only the signal goroutine.
- Make engine stop idempotent so the goroutine and cleanup can both request it without a fatal double-stop.
- Avoid device-number accumulation caused by stale cached name / leaked devices across sequential connects.
- Keep the fix teardown-only; do not alter connect setup behavior.

**Non-Goals:**
- Reaping devices leaked by *previous* process instances (e.g. after `kill -9`). Out of scope — those are OS-reaped on fd close; a separate "clean stale utun" sweep could be a future change.
- Changing the tun2socks dependency or the transport/DNS design.
- Windows/wintun-specific adapter lifecycle beyond calling the shared engine stop.

## Decisions

### Decision 1: Add `Shutdown()` to the `Tunnel` interface, call it from `CleanupWorkspace()`

Add `Shutdown() error` to `pkg/kt/service/tun/types.go`. Implement it once in the shared `tun.go` (platform-independent) so all OSes share the engine-stop logic:

```go
var stopOnce sync.Once
func (s *Cli) Shutdown() error {
    stopOnce.Do(func() {
        engine.Stop()
        resetTunName() // platform hook; darwin clears cached tunName
    })
    return nil
}
```

`ToSocks`'s signal goroutine calls `s.Shutdown()` instead of `engine.Stop()` directly, so both entry points funnel through the same `sync.Once`.

`CleanupWorkspace()` invokes it for connect sessions that created a device:

```go
if opt.Store.Component == util.ComponentConnect && !opt.Get().Connect.DisableTunDevice {
    _ = tun.Ins().Shutdown()
}
```

**Why over alternatives:**
- *Rely on OS fd-close only* (do nothing): rejected — the observed leak proves the current implicit teardown is not reliable across all exit paths.
- *Stop the engine directly in teardown.go without an interface method*: rejected — `teardown.go` already depends on `tun.Ins()`; adding a method keeps the engine detail behind the existing seam and testable via the interface, consistent with `RestoreRoute`.

### Decision 2: `sync.Once` for idempotency

`engine.Stop()` calls `log.Fatalf` if the engine is already stopped (confirmed in the vendored `engine.go`). With two callers (signal goroutine + cleanup) a naive double call would abort the process during teardown. `sync.Once` guarantees exactly one real stop; later calls are no-ops.

**Alternative considered:** a boolean + mutex — equivalent but more code; `sync.Once` is the idiomatic one-liner.

### Decision 3: Reset cached device name on shutdown (macOS)

`tun_darwin.go` caches `tunName`. After a clean shutdown within the same process, reset it to empty so a later `GetName()` re-scans interfaces (the just-destroyed device is gone, so it won't climb). Linux/Windows `resetTunName` is a no-op (fixed names). This addresses the "does not accumulate across sessions" requirement for the in-process case; the cross-process case is handled simply because the device is actually destroyed now, so `max+1` no longer climbs.

## Risks / Trade-offs

- **[Race: signal goroutine and cleanup call Shutdown concurrently]** → `sync.Once` is safe under concurrency; the loser blocks until the winner's `engine.Stop` returns, then no-ops.
- **[`engine.Stop` blocks or hangs during cleanup]** → It is a bounded shutdown of the netstack; run it as the tun step of cleanup. If needed, it can be wrapped with a timeout, but current usage in the signal goroutine already calls it synchronously without issue.
- **[Package-level `sync.Once` prevents restart within one process]** → Acceptable: ktctl runs one connect session per process; the singleton `Cli` is per-process. If future multi-session-per-process support lands, move `stopOnce` into a per-session struct.
- **[Windows wintun teardown differs]** → `engine.Stop()` releases the wintun session the same way; no wintun-specific code added. If wintun needs an explicit adapter close, that is a follow-up scoped to Windows.

## Migration Plan

No data or API migration. Ship as a patch:
1. Add `Shutdown()` to the interface + shared impl; route the signal goroutine through it.
2. Call it from `CleanupWorkspace()`.
3. Verify on macOS: repeated `connect`/disconnect cycles leave no residual `utun` in `ifconfig` and the device number does not climb.

Rollback: revert the commit; behavior returns to signal-goroutine-only stop.

## Open Questions

- Should a separate `ktctl clean` sub-behavior actively sweep and delete `utun` devices leaked by *prior* crashed processes (SIGKILL case)? Deferred to a follow-up change per Non-Goals.
