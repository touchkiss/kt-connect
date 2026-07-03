package tun

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/rs/zerolog/log"
	"github.com/xjasonlyu/tun2socks/v2/engine"
)

// stopMu / stopped guarantee the tun2socks engine is stopped exactly once,
// even when both the ToSocks signal goroutine and CleanupWorkspace request
// shutdown. engine.Stop() calls log.Fatalf on a second stop, so this guard is
// required. stopEngine is a seam so tests can observe shutdown without a real
// engine (a live engine needs a tun device / root).
var (
	stopMu     sync.Mutex
	stopped    bool
	stopEngine = engine.Stop
)

// ToSocks create a tun and connect to socks endpoint
func (s *Cli) ToSocks(sockAddr string) error {
	logLevel := "warn"
	if opt.Get().Global.Debug {
		logLevel = "debug"
	}
	var key = new(engine.Key)
	key.Proxy = sockAddr
	key.Device = fmt.Sprintf("tun://%s", s.GetName())
	key.LogLevel = logLevel
	engine.Insert(key)
	engine.Start()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		_ = s.Shutdown()
		log.Info().Msgf("Tun device %s stopped", key.Device)
	}()
	return nil
}

// Shutdown stops the tun2socks engine, which closes the tun device fd and lets
// the OS destroy the tun (utun) device and its routes. Safe to call multiple
// times; only the first call stops the engine.
func (s *Cli) Shutdown() error {
	stopMu.Lock()
	defer stopMu.Unlock()
	if stopped {
		return nil
	}
	stopEngine()
	resetTunName()
	stopped = true
	return nil
}
