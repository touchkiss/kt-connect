## 1. Add idempotent Shutdown to the tun service

- [x] 1.1 Add `Shutdown() error` to the `Tunnel` interface in `pkg/kt/service/tun/types.go`
- [x] 1.2 Implement `Shutdown()` in `pkg/kt/service/tun/tun.go` using a package-level `sync.Once` that calls `engine.Stop()` and a `resetTunName()` hook
- [x] 1.3 Change the `ToSocks` signal-handler goroutine to call `s.Shutdown()` instead of `engine.Stop()` directly, so both paths share the `sync.Once`
- [x] 1.4 Add `resetTunName()` to each platform file: clear the cached `tunName` in `tun_darwin.go`; no-op in `tun_linux.go` and `tun_windows.go`

## 2. Invoke Shutdown from central teardown

- [x] 2.1 In `pkg/kt/command/general/teardown.go:CleanupWorkspace()`, call `tun.Ins().Shutdown()` when `opt.Store.Component == util.ComponentConnect` and `!opt.Get().Connect.DisableTunDevice`
- [x] 2.2 Confirm `ktctl clean` reaches the same teardown path (verify no separate connect-clean bypass skips the shutdown) — connect process exit routes through `main.go` → `CleanupWorkspace`; `clean`'s `TidyLocalResources` is a separate process and cannot stop another process's engine (documented Non-Goal)

## 3. Tests

- [x] 3.1 Add a test asserting `Shutdown()` is safe to call twice without panicking / fatal (idempotency; implemented via mutex+bool guard with an injectable `stopEngine` seam — the design's stated `sync.Once` alternative, made testable)
- [x] 3.2 Add a teardown test (extend `pkg/kt/command/general/teardown_test.go`) verifying `Shutdown` is invoked for a connect+tun session and skipped for `--disableTunDevice`
- [x] 3.3 Add a `tun_darwin` test verifying `GetName()` re-scans after `resetTunName()` and does not climb across a clean shutdown

## 4. Manual verification (macOS)

- [ ] 4.1 Run `ifconfig | grep utun` before; run `ktctl connect` then disconnect; confirm the created `utun` is gone afterward — **requires a live cluster + sudo; must be run by the user on macOS**
- [ ] 4.2 Repeat connect/disconnect 3+ times and confirm the `utun` number does not monotonically climb and no devices accumulate — **requires a live cluster + sudo; must be run by the user on macOS**
- [x] 4.3 Run `go test ./pkg/kt/service/tun/... ./pkg/kt/command/general/... -count=1` (7 passed) and `go vet` on affected packages (no new findings; pre-existing `setup.go:59` warning only)
