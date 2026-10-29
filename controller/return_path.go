package controller

import (
	"net"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/setting/system_setting"
)

func paymentReturnPath(suffix string) string {
	base := strings.TrimRight(system_setting.ServerAddress, "/")
	return base + suffix
}

func validatedPaymentReturnOrigin(candidate string) string {
	origin, err := url.Parse(candidate)
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") ||
		origin.Host == "" || origin.User != nil || origin.Path != "" ||
		origin.RawQuery != "" || origin.Fragment != "" {
		return ""
	}

	configured, err := url.Parse(system_setting.ServerAddress)
	if err != nil || configured.Host == "" {
		if origin.Scheme == "http" && isLoopbackHost(origin.Hostname()) {
			return origin.Scheme + "://" + origin.Host
		}
		return ""
	}
	sameHost := strings.EqualFold(origin.Hostname(), configured.Hostname())
	sameLoopback := origin.Scheme == "http" &&
		isLoopbackHost(origin.Hostname()) && isLoopbackHost(configured.Hostname())
	if !strings.EqualFold(origin.Scheme, configured.Scheme) || (!sameHost && !sameLoopback) {
		return ""
	}
	if origin.Port() != configured.Port() &&
		!sameLoopback {
		return ""
	}
	return origin.Scheme + "://" + origin.Host
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
