package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type SubscriptionSePayPayRequest struct {
	PlanID int `json:"plan_id"`
}

func SubscriptionRequestSePay(c *gin.Context) {
	if !isSePayTopUpEnabled() {
		common.ApiErrorT(c, "Payment information has not been configured by administrator")
		return
	}
	var req SubscriptionSePayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanID <= 0 {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	plan, err := model.GetSubscriptionPlanById(req.PlanID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled || plan.PriceAmount < 0.01 {
		common.ApiErrorT(c, "Subscription plan is not enabled")
		return
	}
	userID := c.GetInt("id")
	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userID, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorT(c, "Purchase limit for this plan has been reached")
			return
		}
	}

	tradeNo := fmt.Sprintf("SUBUSR%dSEPAY%d", userID, time.Now().UnixNano())
	systemCurrency := operation_setting.GetFinancialCurrencyCode()
	providerMoney, exchangeRate, err := service.ConvertToVND(decimal.NewFromFloat(plan.PriceAmount), systemCurrency)
	if err != nil {
		common.ApiErrorT(c, "Exchange rate is not available")
		return
	}
	order := &model.SubscriptionOrder{
		UserId: userID, PlanId: plan.Id, Money: providerMoney.InexactFloat64(), TradeNo: tradeNo,
		PaymentMethod: sePayPaymentMethod, PaymentProvider: model.PaymentProviderSePay,
		Currency: systemCurrency, ProviderCurrency: "VND", ExchangeRate: exchangeRate.String(),
		ExchangeRateSource: operation_setting.GetGeneralSetting().ExchangeRateSource,
		ExchangeRateType:   operation_setting.GetGeneralSetting().ExchangeRateType,
		ExchangeRateDate:   time.Now().Format("2006-01-02"),
		CreateTime:         time.Now().Unix(), Status: common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		common.ApiErrorT(c, "Failed to create order")
		return
	}
	checkout := buildSePayCheckout(tradeNo, providerMoney.InexactFloat64(), "Subscription "+strings.TrimSpace(plan.Title), strconv.Itoa(userID))
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": checkout.Params, "url": checkout.URL})
}
