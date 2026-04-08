package main

import (
	"testing"
	"time"
)

func TestShouldStartDnsAndProxyWhenLaneProvided(t *testing.T) {
	dnsCalled := false
	proxyCalled := make(chan struct{}, 1)
	startDnsServer = func(int, string, string) {
		dnsCalled = true
	}
	startProxyServer = func(string) {
		proxyCalled <- struct{}{}
	}
	defer resetShadowStarters()

	runShadow("udp", "", "test-lane")

	if !dnsCalled {
		t.Fatal("expected dns server to start")
	}
	select {
	case <-proxyCalled:
	case <-time.After(time.Second):
		t.Fatal("expected proxy server to start when lane is provided")
	}
}
