package main

import (
	"github.com/alibaba/kt-connect/pkg/common"
	"github.com/alibaba/kt-connect/pkg/shadow/dnsserver"
	shadowProxy "github.com/alibaba/kt-connect/pkg/shadow/proxy"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"strings"
)

const (
	// ArgLocalDomains application argument for local domain config
	ArgLocalDomains = "--local-domain"
	// ArgDnsProtocol application argument for shadow pod dns protocol
	ArgDnsProtocol = "--protocol"
	// ArgLogLevel application argument for shadow pod log level
	ArgLogLevel = "--log-level"
	// ArgLane application argument for shadow lane injection
	ArgLane = "--lane"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

var (
	startDnsServer = func(dnsPort int, dnsProtocol string, localDomain string) {
		dnsserver.Start(dnsPort, dnsProtocol, localDomain)
	}
	startProxyServer = func(lane string) {
		log.Info().Msgf("Shadow proxy enabled for lane %s", lane)
		if err := http.ListenAndServe(":80", shadowProxy.New(lane)); err != nil {
			log.Error().Err(err).Msg("Failed to start shadow proxy")
		}
	}
)

func main() {
	logLevel := getParameter(common.EnvVarLogLevel, ArgLogLevel, "info")
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse log level")
	}
	zerolog.SetGlobalLevel(level)
	dnsProtocol := getParameter(common.EnvVarDnsProtocol, ArgDnsProtocol, "udp")
	localDomain := getParameter(common.EnvVarLocalDomains, ArgLocalDomains, "")
	lane := getParameter(common.EnvVarLane, ArgLane, "")
	runShadow(dnsProtocol, localDomain, lane)
}

func runShadow(dnsProtocol, localDomain, lane string) {
	dnsPort := common.StandardDnsPort
	log.Info().Msgf("Shadow DNS on %s port %d", dnsProtocol, dnsPort)
	if localDomain != "" {
		log.Info().Msgf("Using local domain %s", localDomain)
	}
	if lane != "" {
		go startProxyServer(lane)
	}
	startDnsServer(dnsPort, dnsProtocol, localDomain)
}

func resetShadowStarters() {
	startDnsServer = func(dnsPort int, dnsProtocol string, localDomain string) {
		dnsserver.Start(dnsPort, dnsProtocol, localDomain)
	}
	startProxyServer = func(lane string) {
		log.Info().Msgf("Shadow proxy enabled for lane %s", lane)
		if err := http.ListenAndServe(":80", shadowProxy.New(lane)); err != nil {
			log.Error().Err(err).Msg("Failed to start shadow proxy")
		}
	}
}

func getParameter(envVar string, argVar string, defaultValue string) string {
	if os.Getenv(envVar) != "" {
		return os.Getenv(envVar)
	}
	for _, arg := range os.Args {
		kv := strings.SplitN(arg, "=", 2)
		if len(kv) > 1 && kv[0] == argVar && kv[1] != "" {
			return kv[1]
		}
	}
	return defaultValue
}
