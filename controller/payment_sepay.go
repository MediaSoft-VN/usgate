package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const sePayProvider = model.PaymentProviderSePay
const sePayPaymentMethod = "BANK_TRANSFER"

type sePayCredentials struct {
	merchantID          string
	secretKey           string
	legacyWebhookSecret string
}

func sePayCredentialsForMode(mode string) (sePayCredentials, bool) {
	var credentials sePayCredentials
	var sandbox bool
	switch mode {
	case "":
		sandbox = setting.SePaySandbox
	case "sandbox":
		sandbox = true
	case "live":
		sandbox = false
	default:
		return sePayCredentials{}, false
	}

	if sandbox {
		credentials = sePayCredentials{
			merchantID: setting.SePaySandboxMerchantID,
			secretKey:  setting.SePaySandboxSecretKey,
		}
	} else {
		credentials = sePayCredentials{
			merchantID: setting.SePayLiveMerchantID,
			secretKey:  setting.SePayLiveSecretKey,
		}
	}
	if sandbox == setting.SePaySandbox {
		credentials.legacyWebhookSecret = setting.SePayWebhookSecret
		if credentials.merchantID == "" {
			credentials.merchantID = setting.SePayMerchantID
		}
		if credentials.secretKey == "" {
			credentials.secretKey = setting.SePaySecretKey
		}
	}
	return credentials, true
}

func isSePayModeConfigured(mode string) bool {
	credentials, validMode := sePayCredentialsForMode(mode)
	return validMode && strings.TrimSpace(credentials.merchantID) != "" &&
		strings.TrimSpace(credentials.secretKey) != ""
}

func sePayNotificationCredentials(c *gin.Context, body []byte, mode string) (sePayCredentials, bool) {
	if mode != "" {
		credentials, validMode := sePayCredentialsForMode(mode)
		return credentials, validMode && isSePayModeConfigured(mode) &&
			(sePayNotificationAuthorized(c, body, credentials.secretKey) ||
				sePayNotificationAuthorized(c, body, credentials.legacyWebhookSecret))
	}

	preferredMode := "sandbox"
	otherMode := "live"
	if !setting.SePaySandbox {
		preferredMode, otherMode = otherMode, preferredMode
	}
	for _, candidateMode := range []string{preferredMode, otherMode} {
		credentials, _ := sePayCredentialsForMode(candidateMode)
		if isSePayModeConfigured(candidateMode) &&
			(sePayNotificationAuthorized(c, body, credentials.secretKey) ||
				sePayNotificationAuthorized(c, body, credentials.legacyWebhookSecret)) {
			return credentials, true
		}
	}
	return sePayCredentials{}, false
}

type sePayCheckout struct {
	URL    string
	Params map[string]string
}

func calculateSePayAmount(amount int64, group string) (decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	systemAmount := getTopUpAmountInSystemCurrency(amount, group)
	providerAmount, exchangeRate, err := service.ConvertToVND(
		systemAmount,
		operation_setting.GetFinancialCurrencyCode(),
	)
	return systemAmount, providerAmount.Round(0), exchangeRate, err
}

func sePayCheckoutURL() string {
	if setting.SePaySandbox {
		return "https://pay-sandbox.sepay.vn/v1/checkout/init"
	}
	return "https://pay.sepay.vn/v1/checkout/init"
}

func buildSePayCheckout(tradeNo string, amount decimal.Decimal, description string, customerID string) sePayCheckout {
	callbackAddress := strings.TrimRight(service.GetCallbackAddress(), "/")
	returnURL := callbackAddress + "/api/sepay/return"
	return buildSePayCheckoutWithReturnURLs(tradeNo, amount, description, customerID, returnURL, returnURL, returnURL)
}

func buildSePayTopUpCheckout(tradeNo string, amount decimal.Decimal, description string, customerID string, returnOrigin string) sePayCheckout {
	callbackAddress := strings.TrimRight(service.GetCallbackAddress(), "/")
	callbackURL := callbackAddress + "/api/sepay/return"
	return buildSePayCheckoutWithReturnURLs(
		tradeNo,
		amount,
		description,
		customerID,
		sePayTopUpReturnURL(callbackURL, "pending", tradeNo, returnOrigin),
		sePayTopUpReturnURL(callbackURL, "fail", tradeNo, returnOrigin),
		sePayTopUpReturnURL(callbackURL, "cancel", tradeNo, returnOrigin),
	)
}

func buildSePayCheckoutWithReturnURLs(tradeNo string, amount decimal.Decimal, description string, customerID string, successURL string, errorURL string, cancelURL string) sePayCheckout {
	credentials, _ := sePayCredentialsForMode("")
	params := map[string]string{
		"merchant":             credentials.merchantID,
		"operation":            "PURCHASE",
		"payment_method":       sePayPaymentMethod,
		"order_invoice_number": tradeNo,
		"order_amount":         amount.Round(0).StringFixed(0),
		"currency":             "VND",
		"order_description":    description,
		"success_url":          successURL,
		"error_url":            errorURL,
		"cancel_url":           cancelURL,
	}
	if customerID != "" {
		params["customer_id"] = customerID
	}

	fields := []string{
		"order_amount", "merchant", "currency", "operation", "order_description",
		"order_invoice_number", "customer_id", "payment_method", "success_url",
		"error_url", "cancel_url",
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		value, ok := params[field]
		if ok {
			parts = append(parts, field+"="+value)
		}
	}
	hash := hmac.New(sha256.New, []byte(credentials.secretKey))
	_, _ = hash.Write([]byte(strings.Join(parts, ",")))
	params["signature"] = base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return sePayCheckout{URL: sePayCheckoutURL(), Params: params}
}

func sePayTopUpReturnURL(callbackURL string, status string, tradeNo string, returnOrigin string) string {
	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return callbackURL
	}
	query := parsed.Query()
	query.Set("pay", status)
	query.Set("trade_no", tradeNo)
	if validatedOrigin := validatedPaymentReturnOrigin(returnOrigin); validatedOrigin != "" {
		query.Set("return_origin", validatedOrigin)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func RequestSePayAmount(c *gin.Context) {
	if !isSePayTopUpEnabled() {
		paymentError(c, common.NewMessage("Payment information has not been configured by administrator"))
		return
	}
	var req AmountRequest
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
	systemAmount, providerAmount, _, err := calculateSePayAmount(req.Amount, group)
	if systemAmount.LessThan(decimal.NewFromFloat(0.01)) {
		paymentError(c, common.NewMessage("Payment amount is too low"))
		return
	}
	if err != nil {
		paymentError(c, common.NewMessage("Exchange rate is not available"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": providerAmount.StringFixed(0)})
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
	systemAmount, providerMoney, exchangeRate, err := calculateSePayAmount(req.Amount, group)
	if systemAmount.LessThan(decimal.NewFromFloat(0.01)) {
		paymentError(c, common.NewMessage("Payment amount is too low"))
		return
	}
	if err != nil {
		paymentError(c, common.NewMessage("Exchange rate is not available"))
		return
	}
	providerMoneyFloat := providerMoney.InexactFloat64()
	systemCurrency := operation_setting.GetFinancialCurrencyCode()

	tradeNo := fmt.Sprintf("USR%dSEPAY%d", userID, time.Now().UnixNano())
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = decimal.NewFromInt(amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}
	topUp := &model.TopUp{
		UserId: userID, Amount: amount, Money: providerMoneyFloat, TradeNo: tradeNo,
		PaymentMethod: sePayPaymentMethod, PaymentProvider: sePayProvider,
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
	checkout := buildSePayTopUpCheckout(
		tradeNo,
		providerMoney,
		fmt.Sprintf("Top-up %s", tradeNo),
		strconv.Itoa(userID),
		c.GetHeader("Origin"),
	)
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": checkout.Params, "url": checkout.URL})
}

type sePayIPNPayload struct {
	Timestamp        int64  `json:"timestamp"`
	NotificationType string `json:"notification_type"`
	Order            struct {
		InvoiceNumber string `json:"order_invoice_number"`
		Status        string `json:"order_status"`
		Amount        any    `json:"order_amount"`
		CustomData    any    `json:"custom_data"`
	} `json:"order"`
	Transaction struct {
		Status string `json:"transaction_status"`
	} `json:"transaction"`
}

type sePayTransactionWebhookPayload struct {
	ID             int64  `json:"id"`
	SubAccount     string `json:"subAccount"`
	Code           string `json:"code"`
	Content        string `json:"content"`
	TransferType   string `json:"transferType"`
	TransferAmount int64  `json:"transferAmount"`
}

var sePayOrderReferencePattern = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])((?:SUB)?USR[0-9]+SEPAY[0-9]+)(?:$|[^A-Za-z0-9])`)
var sePayTopUpTradeNoPattern = regexp.MustCompile(`(?i)^USR[0-9]+SEPAY[0-9]+$`)

func sePayNotificationAuthorized(c *gin.Context, body []byte, secretKey string) bool {
	secret := strings.TrimSpace(secretKey)
	if secret == "" {
		return false
	}

	if hmac.Equal([]byte(c.GetHeader("X-Secret-Key")), []byte(secret)) {
		return true
	}

	if scheme, apiKey, ok := strings.Cut(c.GetHeader("Authorization"), " "); ok &&
		strings.EqualFold(scheme, "Apikey") && hmac.Equal([]byte(apiKey), []byte(secret)) {
		return true
	}

	timestampHeader := c.GetHeader("X-SePay-Timestamp")
	timestamp, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	if timestamp < now-300 || timestamp > now+300 {
		return false
	}
	signature := c.GetHeader("X-SePay-Signature")
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestampHeader + "."))
	_, _ = mac.Write(body)
	expected := []byte("sha256=" + hex.EncodeToString(mac.Sum(nil)))
	return hmac.Equal([]byte(signature), expected)
}

func sePayTransactionTradeNo(payload sePayTransactionWebhookPayload) string {
	tradeNos := make(map[string]struct{})
	for _, value := range []string{payload.Code, payload.Content, payload.SubAccount} {
		for _, match := range sePayOrderReferencePattern.FindAllStringSubmatch(value, -1) {
			tradeNos[strings.ToUpper(match[1])] = struct{}{}
		}
	}
	if len(tradeNos) != 1 {
		return ""
	}
	for tradeNo := range tradeNos {
		return tradeNo
	}
	return ""
}

func sePayTransactionRequestHash(payload sePayTransactionWebhookPayload, tradeNo string) (string, error) {
	canonical, err := common.Marshal(struct {
		ID             int64  `json:"id"`
		TradeNo        string `json:"trade_no"`
		TransferType   string `json:"transfer_type"`
		TransferAmount int64  `json:"transfer_amount"`
	}{payload.ID, tradeNo, strings.ToLower(payload.TransferType), payload.TransferAmount})
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(canonical)
	return hex.EncodeToString(hash[:]), nil
}

func processSePayTransactionWebhook(c *gin.Context, payload sePayTransactionWebhookPayload) error {
	if payload.ID <= 0 || payload.TransferAmount <= 0 || !strings.EqualFold(payload.TransferType, "in") {
		return nil
	}
	tradeNo := sePayTransactionTradeNo(payload)
	if tradeNo == "" {
		return nil
	}
	amount := decimal.NewFromInt(payload.TransferAmount)
	matched, exists := sePayWebhookAmountMatches(tradeNo, amount)
	if !exists || !matched {
		logger.LogWarn(c.Request.Context(), common.LogText("SePay transaction did not match a pending order transaction_id=%d", payload.ID))
		return nil
	}
	requestHash, err := sePayTransactionRequestHash(payload, tradeNo)
	if err != nil {
		return err
	}
	key, replay, err := model.ClaimIdempotencyKey("sepay_webhook", strconv.FormatInt(payload.ID, 10), requestHash, 0)
	if err != nil {
		logger.LogWarn(c.Request.Context(), common.LogText("SePay transaction idempotency conflict transaction_id=%d", payload.ID))
		return err
	}
	if replay && key.Status == model.IdempotencyCompleted {
		return nil
	}

	if strings.HasPrefix(tradeNo, "SUBUSR") {
		LockOrder(tradeNo)
		err = model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(payload), sePayProvider, sePayPaymentMethod)
		UnlockOrder(tradeNo)
	} else {
		LockOrder(tradeNo)
		_, err = model.RechargeTopUpWithProvider(tradeNo, sePayProvider, sePayPaymentMethod, c.ClientIP())
		UnlockOrder(tradeNo)
	}
	if err != nil {
		_ = model.DB.Model(key).Update("status", model.IdempotencyFailed).Error
		return err
	}
	if err := model.DB.Model(key).Updates(map[string]any{
		"status": model.IdempotencyCompleted, "resource_type": "payment_order",
		"resource_id": tradeNo, "completed_at": time.Now().Unix(),
	}).Error; err != nil {
		return err
	}
	return nil
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

func SePayIPN(c *gin.Context) {
	mode := c.Param("mode")
	if !isPaymentComplianceConfirmed() {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false})
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false})
		return
	}
	if _, authorized := sePayNotificationCredentials(c, body, mode); !authorized {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false})
		return
	}
	var payload sePayIPNPayload
	if err := common.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false})
		return
	}
	if (payload.NotificationType != "ORDER_PAID" && payload.NotificationType != "PAYMENT_SUCCESS") ||
		payload.Order.Status != "CAPTURED" || payload.Transaction.Status != "APPROVED" {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}
	customData, _ := payload.Order.CustomData.(map[string]any)
	webhookTest, _ := customData["webhook_test"].(bool)
	if webhookTest {
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
		logger.LogWarn(c.Request.Context(), common.LogText("SePay IPN amount mismatch or order missing trade_no=%s", payload.Order.InvoiceNumber))
		c.JSON(http.StatusBadRequest, gin.H{"success": false})
		return
	}

	tradeNo := payload.Order.InvoiceNumber
	providerPayload := common.GetJsonString(payload)
	if strings.HasPrefix(tradeNo, "SUBUSR") {
		LockOrder(tradeNo)
		defer UnlockOrder(tradeNo)
		if err := model.CompleteSubscriptionOrder(tradeNo, providerPayload, sePayProvider, sePayPaymentMethod); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false})
			return
		}
	} else {
		LockOrder(tradeNo)
		defer UnlockOrder(tradeNo)
		if _, err := model.RechargeTopUpWithProvider(tradeNo, sePayProvider, sePayPaymentMethod, c.ClientIP()); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func SePayWebhook(c *gin.Context) {
	mode := c.Param("mode")
	if !isPaymentComplianceConfirmed() {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false})
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false})
		return
	}
	if _, authorized := sePayNotificationCredentials(c, body, mode); !authorized {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false})
		return
	}
	var payload sePayTransactionWebhookPayload
	if err := common.Unmarshal(body, &payload); err != nil || payload.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false})
		return
	}
	if err := processSePayTransactionWebhook(c, payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func SePayReturn(c *gin.Context) {
	status := c.Query("pay")
	if status != "fail" && status != "cancel" {
		status = "pending"
	}
	tradeNo := strings.TrimSpace(c.Query("trade_no"))
	if !sePayTopUpTradeNoPattern.MatchString(tradeNo) {
		tradeNo = ""
	}
	baseAddress := validatedPaymentReturnOrigin(c.Query("return_origin"))
	if baseAddress == "" {
		baseAddress = strings.TrimRight(system_setting.ServerAddress, "/")
	}
	query := url.Values{"pay": {status}}
	if tradeNo != "" {
		query.Set("trade_no", tradeNo)
	}
	c.Redirect(http.StatusFound, baseAddress+"/wallet?"+query.Encode())
}
