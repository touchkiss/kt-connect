package proxy

import (
	"net/http"
	"net/http/httputil"
	"strings"
)

func New(lane string) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(req *httputil.ProxyRequest) {
			req.SetURL(req.In.URL)
			req.Out.Host = req.In.Host
			if lane == "" {
				return
			}
			req.Out.Header.Set("Baggage", mergeBaggage(req.In.Header.Values("Baggage"), lane))
		},
	}
	return proxy
}

func mergeBaggage(values []string, lane string) string {
	members := make([]string, 0)
	for _, value := range values {
		for _, member := range strings.Split(value, ",") {
			member = strings.TrimSpace(member)
			if member == "" || isLaneBaggageMember(member) {
				continue
			}
			members = append(members, member)
		}
	}
	members = append(members, "lane="+lane)
	return strings.Join(members, ",")
}

func isLaneBaggageMember(member string) bool {
	parts := strings.SplitN(member, "=", 2)
	if len(parts) != 2 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(parts[0]), "lane")
}
