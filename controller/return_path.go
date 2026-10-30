package controller

import (
	"net"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func paymentReturnPath(suffix string) string {
	base := strings.TrimRight(system_setting.ServerAddress, "/")
	return base + suffix
}

func paymentReturnPathForRequest(c *gin.Context, suffix string) string {
	if origin := verifiedTenantReturnOrigin(c.Request.Host); origin != "" {
		return origin + suffix
	}
	return paymentReturnPath(suffix)
}

func paymentReturnOriginForRequest(c *gin.Context) string {
	if origin := verifiedTenantReturnOrigin(c.Request.Host); origin != "" {
		return origin
	}
	return validatedPaymentReturnOrigin(c.GetHeader("Origin"), "")
}

func verifiedTenantReturnOrigin(requestHost string) string {
	if model.DB == nil {
		return ""
	}
	host := requestHost
	if parsedHost, _, err := net.SplitHostPort(requestHost); err == nil {
		host = parsedHost
	}
	normalized, err := model.NormalizeTenantHostname(host)
	if err != nil {
		return ""
	}
	if _, err = model.ResolveVerifiedTenantForRegistration(normalized); err != nil {
		return ""
	}
	configured, err := url.Parse(system_setting.ServerAddress)
	if err != nil || (configured.Scheme != "http" && configured.Scheme != "https") {
		return ""
	}
	return configured.Scheme + "://" + normalized
}

func validatedPaymentReturnOrigin(candidate string, tradeNo string) string {
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
	if !strings.EqualFold(origin.Scheme, configured.Scheme) {
		return ""
	}
	if sameLoopback {
		return origin.Scheme + "://" + origin.Host
	}
	if sameHost {
		if origin.Port() != configured.Port() {
			return ""
		}
		return origin.Scheme + "://" + origin.Host
	}
	if tradeNo == "" || origin.Port() != "" {
		return ""
	}
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		return ""
	}
	if _, err = model.ResolveVerifiedTenant(origin.Hostname(), topUp.UserId); err != nil {
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
