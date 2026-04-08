package tun

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	opt "github.com/alibaba/kt-connect/pkg/kt/command/options"
	"github.com/rs/zerolog/log"
	"github.com/xjasonlyu/tun2socks/v2/engine"
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
		engine.Stop()
		log.Info().Msgf("Tun device %s stopped", key.Device)
	}()
	return nil
}
