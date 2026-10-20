package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const sePayProvider = model.PaymentProviderSePay

type sePayCheckout struct {
	URL    string
	Params map[string]string
}

func sePayCheckoutURL() string {
	if setting.SePaySandbox {
		return "https://pay-sandbox.sepay.vn/v1/checkout/init"
	}
	return "https://pay.sepay.vn/v1/checkout/init"
}

func buildSePayCheckout(tradeNo string, amount float64, description string, customerID string) sePayCheckout {
	callbackAddress := strings.TrimRight(service.GetCallbackAddress(), "/")
	returnURL := callbackAddress + "/api/sepay/return"
	params := map[string]string{
		"merchant":             setting.SePayMerchantID,
		"operation":            "PURCHASE",
		"payment_method":       setting.SePayPaymentMethod,
		"order_invoice_number": tradeNo,
		"order_amount":         strconv.FormatFloat(amount, 'f', 2, 64),
		"currency":             "VND",
		"order_description":    description,
		"success_url":          returnURL,
		"error_url":            returnURL,
		"cancel_url":           returnURL,
	}
	if customerID != "" {
		params["customer_id"] = customerID
	}

	fields := []string{
		"merchant", "operation", "payment_method", "order_amount", "currency",
		"order_invoice_number", "order_description", "customer_id", "success_url",
		"error_url", "cancel_url",
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		value, ok := params[field]
		if ok {
			parts = append(parts, field+"="+value)
		}
	}
	hash := hmac.New(sha256.New, []byte(setting.SePaySecretKey))
	_, _ = hash.Write([]byte(strings.Join(parts, ",")))
	params["signature"] = base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return sePayCheckout{URL: sePayCheckoutURL(), Params: params}
}

func RequestSePay(c *gin.Context) {
	if !isSePayTopUpEnabled() {
		paymentError(c, common.NewMessage("Payment information has not been configured by administrator"))
		return
	}
	var req EpayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount < int64(setting.SePayMinTopUp) {
		paymentError(c, common.NewMessage("Invalid parameters"))
		return
	}
	userID := c.GetInt("id")
	if rejectInvalidTopUpQuota(c, userID, req.Amount) {
		return
	}
	group, err := model.GetUserGroup(userID, true)
	if err != nil {
		paymentError(c, common.NewMessage("Failed to get the user group"))
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		paymentError(c, common.NewMessage("Payment amount is too low"))
		return
	}
	systemCurrency := operation_setting.GetFinancialCurrencyCode()
	providerMoney, exchangeRate, err := service.ConvertToVND(decimal.NewFromFloat(payMoney), systemCurrency)
	if err != nil {
		paymentError(c, common.NewMessage("Exchange rate is not available"))
		return
	}
	providerMoneyFloat := providerMoney.InexactFloat64()

	tradeNo := fmt.Sprintf("USR%dSEPAY%d", userID, time.Now().UnixNano())
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = decimal.NewFromInt(amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}
	topUp := &model.TopUp{
		UserId: userID, Amount: amount, Money: providerMoneyFloat, TradeNo: tradeNo,
		PaymentMethod: setting.SePayPaymentMethod, PaymentProvider: sePayProvider,
		Currency: systemCurrency, ProviderCurrency: "VND", ExchangeRate: exchangeRate.String(),
		ExchangeRateSource: operation_setting.GetGeneralSetting().ExchangeRateSource,
		ExchangeRateType:   operation_setting.GetGeneralSetting().ExchangeRateType,
		ExchangeRateDate:   time.Now().Format("2006-01-02"),
		CreateTime:         time.Now().Unix(), Status: common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		paymentError(c, common.NewMessage("Failed to create order"))
		return
	}
	checkout := buildSePayCheckout(tradeNo, providerMoneyFloat, fmt.Sprintf("Top-up %s", tradeNo), strconv.Itoa(userID))
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": checkout.Params, "url": checkout.URL})
}

type sePayWebhookPayload struct {
	Timestamp        int64  `json:"timestamp"`
	NotificationType string `json:"notification_type"`
	Order            struct {
		InvoiceNumber string `json:"order_invoice_number"`
		Status        string `json:"order_status"`
		Amount        any    `json:"order_amount"`
	} `json:"order"`
	Transaction struct {
		Status string `json:"transaction_status"`
	} `json:"transaction"`
}

func sePayAmount(value any) (decimal.Decimal, bool) {
	switch typed := value.(type) {
	case string:
		amount, err := decimal.NewFromString(typed)
		return amount, err == nil
	case float64:
		return decimal.NewFromFloat(typed), true
	case int:
		return decimal.NewFromInt(int64(typed)), true
	default:
		return decimal.Zero, false
	}
}

func sePayWebhookAmountMatches(tradeNo string, amount decimal.Decimal) (bool, bool) {
	if topUp := model.GetTopUpByTradeNo(tradeNo); topUp != nil {
		return decimal.NewFromFloat(topUp.Money).Equal(amount), true
	}
	if order := model.GetSubscriptionOrderByTradeNo(tradeNo); order != nil {
		return decimal.NewFromFloat(order.Money).Equal(amount), true
	}
	return false, false
}

func SePayWebhook(c *gin.Context) {
	if !isSePayTopUpEnabled() || !hmac.Equal([]byte(c.GetHeader("X-Secret-Key")), []byte(setting.SePayWebhookSecret)) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false})
		return
	}
	var payload sePayWebhookPayload
	if err := common.DecodeJson(c.Request.Body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false})
		return
	}
	if payload.NotificationType != "ORDER_PAID" || payload.Order.Status != "CAPTURED" || payload.Transaction.Status != "APPROVED" {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}
	amount, ok := sePayAmount(payload.Order.Amount)
	if !ok || payload.Order.InvoiceNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false})
		return
	}
	matched, exists := sePayWebhookAmountMatches(payload.Order.InvoiceNumber, amount)
	if !exists || !matched {
		logger.LogWarn(c.Request.Context(), common.LogText("SePay webhook amount mismatch or order missing trade_no=%s", payload.Order.InvoiceNumber))
		c.JSON(http.StatusBadRequest, gin.H{"success": false})
		return
	}

	tradeNo := payload.Order.InvoiceNumber
	providerPayload := common.GetJsonString(payload)
	if strings.HasPrefix(tradeNo, "SUBUSR") {
		LockOrder(tradeNo)
		defer UnlockOrder(tradeNo)
		if err := model.CompleteSubscriptionOrder(tradeNo, providerPayload, sePayProvider, setting.SePayPaymentMethod); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false})
			return
		}
	} else {
		LockOrder(tradeNo)
		defer UnlockOrder(tradeNo)
		if _, err := model.RechargeTopUpWithProvider(tradeNo, sePayProvider, setting.SePayPaymentMethod, c.ClientIP()); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func SePayReturn(c *gin.Context) {
	c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=pending"))
}
