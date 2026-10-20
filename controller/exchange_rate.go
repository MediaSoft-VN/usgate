package controller

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

// GetExchangeRates returns the rates shown in the admin currency table.
func GetExchangeRates(c *gin.Context) {
	settings := operation_setting.GetGeneralSetting()
	common.ApiSuccess(c, gin.H{
		"source":    settings.ExchangeRateSource,
		"rate_type": settings.ExchangeRateType,
		"rates":     service.CurrentRatesForDisplay(),
	})
}

// SyncExchangeRates refreshes the configured bank rates for the current date.
// It is deliberately admin-only; payment creation consumes the stored snapshot
// and never makes an unbounded provider call in the checkout request.
func SyncExchangeRates(c *gin.Context) {
	if err := service.VietcombankExchangeRates(c.Request.Context(), time.Now()); err != nil {
		common.ApiErrorT(c, "Failed to update exchange rates")
		return
	}
	common.ApiSuccess(c, gin.H{"date": time.Now().Format("2006-01-02")})
}
