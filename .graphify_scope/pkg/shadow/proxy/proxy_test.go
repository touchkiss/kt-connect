package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestProxyInjectsLaneIntoBaggage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Header.Get("Baggage"))
	}))
	defer upstream.Close()

	proxyServer := httptest.NewServer(New("test-lane"))
	defer proxyServer.Close()

	proxyURL, err := url.Parse(proxyServer.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	request, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Baggage", "foo=bar")

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	baggage := string(body)
	if !strings.Contains(baggage, "foo=bar") {
		t.Fatalf("expected baggage to preserve existing members, got %q", baggage)
	}
	if !strings.Contains(baggage, "lane=test-lane") {
		t.Fatalf("expected baggage to include lane member, got %q", baggage)
	}
}

func TestProxyInjectsLaneWhenBaggageMissing(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Header.Get("Baggage"))
	}))
	defer upstream.Close()

	proxyServer := httptest.NewServer(New("test-lane"))
	defer proxyServer.Close()

	proxyURL, err := url.Parse(proxyServer.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	request, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if string(body) != "lane=test-lane" {
		t.Fatalf("expected only lane baggage, got %q", string(body))
	}
}

func TestProxyReplacesLaneAcrossMultipleBaggageHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Header.Get("Baggage"))
	}))
	defer upstream.Close()

	proxyServer := httptest.NewServer(New("test-lane"))
	defer proxyServer.Close()

	proxyURL, err := url.Parse(proxyServer.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	request, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Add("Baggage", "foo=bar")
	request.Header.Add("Baggage", " Lane = old , trace=value ")

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	baggage := string(body)
	lowerBaggage := strings.ToLower(strings.ReplaceAll(baggage, " ", ""))
	if strings.Contains(lowerBaggage, "lane=old") {
		t.Fatalf("expected old lane member to be replaced across headers, got %q", baggage)
	}
	if strings.Count(lowerBaggage, "lane=") != 1 {
		t.Fatalf("expected exactly one normalized lane member, got %q", baggage)
	}
	if !strings.Contains(baggage, "foo=bar") || !strings.Contains(baggage, "trace=value") || !strings.Contains(baggage, "lane=test-lane") {
		t.Fatalf("expected baggage to preserve other members and inject new lane, got %q", baggage)
	}
}

func TestProxyReplacesExistingLaneInBaggage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Header.Get("Baggage"))
	}))
	defer upstream.Close()

	proxyServer := httptest.NewServer(New("test-lane"))
	defer proxyServer.Close()

	proxyURL, err := url.Parse(proxyServer.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	request, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Baggage", "foo=bar,lane=old,trace=value")

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	baggage := string(body)
	if strings.Contains(baggage, "lane=old") {
		t.Fatalf("expected old lane member to be replaced, got %q", baggage)
	}
	if strings.Count(baggage, "lane=") != 1 {
		t.Fatalf("expected exactly one lane member, got %q", baggage)
	}
	if !strings.Contains(baggage, "lane=test-lane") {
		t.Fatalf("expected baggage to include lane member, got %q", baggage)
	}
	if !strings.Contains(baggage, "foo=bar") || !strings.Contains(baggage, "trace=value") {
		t.Fatalf("expected baggage to preserve other members, got %q", baggage)
	}
}
