package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSePayWebhookTest(t *testing.T) (*gorm.DB, model.User, *model.TopUp) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}, &model.IdempotencyKey{}, &model.Log{}, &model.ExchangeRate{}))
	previousDB, previousLogDB := model.DB, model.LOG_DB
	model.DB, model.LOG_DB = db, db
	previousRDB, previousRedisEnabled := common.RDB, common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RDB, common.RedisEnabled = previousRDB, previousRedisEnabled
	})
	confirmPaymentComplianceForTest(t)
	previousMerchantID := setting.SePayMerchantID
	previousSecretKey := setting.SePaySecretKey
	previousWebhookSecret := setting.SePayWebhookSecret
	previousAllowedIPs := setting.SePayIPNAllowedIPs
	previousSandboxMode := setting.SePaySandbox
	previousSandboxMerchantID, previousSandboxSecretKey := setting.SePaySandboxMerchantID, setting.SePaySandboxSecretKey
	previousLiveMerchantID, previousLiveSecretKey := setting.SePayLiveMerchantID, setting.SePayLiveSecretKey
	t.Cleanup(func() {
		setting.SePayMerchantID = previousMerchantID
		setting.SePaySecretKey = previousSecretKey
		setting.SePayWebhookSecret = previousWebhookSecret
		setting.SePayIPNAllowedIPs = previousAllowedIPs
		setting.SePaySandbox = previousSandboxMode
		setting.SePaySandboxMerchantID, setting.SePaySandboxSecretKey = previousSandboxMerchantID, previousSandboxSecretKey
		setting.SePayLiveMerchantID, setting.SePayLiveSecretKey = previousLiveMerchantID, previousLiveSecretKey
	})
	setting.SePayMerchantID = "merchant-test"
	setting.SePaySecretKey = "checkout-secret"
	setting.SePayWebhookSecret = "webhook-secret"
	setting.SePayIPNAllowedIPs = ""
	setting.SePaySandbox = true
	setting.SePaySandboxMerchantID, setting.SePaySandboxSecretKey = "", ""
	setting.SePayLiveMerchantID, setting.SePayLiveSecretKey = "", ""
	user := model.User{Username: "sepay-user", Password: "test-password", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)
	topUp := &model.TopUp{
		UserId: user.Id, Amount: 1, Money: 50000, TradeNo: "USR1SEPAY123",
		PaymentMethod: sePayPaymentMethod, PaymentProvider: model.PaymentProviderSePay,
		Status: common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(topUp).Error)
	return db, user, topUp
}

func sendSePayWebhook(body string, authorization string) *httptest.ResponseRecorder {
	return sendSePayWebhookForMode(body, authorization, "")
}

func sendSePayIPN(body string, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sepay/ipn", strings.NewReader(body))
	c.Request.Header.Set("Authorization", authorization)
	SePayIPN(c)
	return recorder
}

func sendUnauthenticatedSePayIPN(body string, remoteAddr string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sepay/ipn", strings.NewReader(body))
	c.Request.RemoteAddr = remoteAddr
	SePayIPN(c)
	return recorder
}

func sendSePayWebhookForMode(body string, authorization string, mode string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	path := "/api/sepay/webhook"
	if mode != "" {
		path += "/" + mode
	}
	c.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	c.Params = gin.Params{{Key: "mode", Value: mode}}
	c.Request.Header.Set("Authorization", authorization)
	SePayWebhook(c)
	return recorder
}

func TestSePayTransactionWebhookCreditsMatchingOrderOnce(t *testing.T) {
	db, user, topUp := setupSePayWebhookTest(t)
	body := `{"id":92704,"gateway":"Vietcombank","transactionDate":"2024-07-02 11:08:33","accountNumber":"1017588888","subAccount":"","code":"USR1SEPAY123","content":"USR1SEPAY123 chuyen tien","transferType":"in","transferAmount":50000,"accumulated":105000000,"referenceCode":"FT24012345678"}`
	first := sendSePayWebhook(body, "Apikey webhook-secret")
	require.Equal(t, http.StatusOK, first.Code)
	require.JSONEq(t, `{"success":true}`, first.Body.String())

	var settled model.TopUp
	require.NoError(t, db.First(&settled, topUp.Id).Error)
	require.Equal(t, common.TopUpStatusSuccess, settled.Status)
	var credited model.User
	require.NoError(t, db.First(&credited, user.Id).Error)
	firstQuota := credited.Quota
	require.Greater(t, firstQuota, user.Quota)

	replayed := sendSePayWebhook(body, "Apikey webhook-secret")
	require.Equal(t, http.StatusOK, replayed.Code)
	require.JSONEq(t, `{"success":true}`, replayed.Body.String())
	require.NoError(t, db.First(&credited, user.Id).Error)
	require.Equal(t, firstQuota, credited.Quota)
	var idempotency model.IdempotencyKey
	require.NoError(t, db.Where("scope = ? AND key = ?", "sepay_webhook", "92704").First(&idempotency).Error)
	require.Equal(t, model.IdempotencyCompleted, idempotency.Status)
}

func TestSePayPaymentSuccessIPNCreditsMatchingOrder(t *testing.T) {
	db, user, topUp := setupSePayWebhookTest(t)
	body := `{"timestamp":1793155971,"notification_type":"PAYMENT_SUCCESS","order":{"id":"aa658c15-d27a-11f1-8895-0afffe87053b","order_id":"TEST_ORDER_1793155971","order_status":"CAPTURED","order_currency":"VND","order_amount":50000,"order_invoice_number":"USR1SEPAY123","custom_data":[],"order_description":"SePay IPN test"},"transaction":{"id":"aa659519-d27a-11f1-8895-0afffe87053b","payment_method":"BANK_TRANSFER","transaction_id":"TEST_TXN_1793155971","transaction_type":"CAPTURE","transaction_date":"2026-10-28 09:52:51","transaction_status":"APPROVED","transaction_amount":50000,"transaction_currency":"VND"},"customer":null,"agreement":null}`

	response := sendSePayIPN(body, "Apikey webhook-secret")
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"success":true}`, response.Body.String())

	var updatedTopUp model.TopUp
	require.NoError(t, db.First(&updatedTopUp, topUp.Id).Error)
	require.Equal(t, common.TopUpStatusSuccess, updatedTopUp.Status)
	var updatedUser model.User
	require.NoError(t, db.First(&updatedUser, user.Id).Error)
	require.Greater(t, updatedUser.Quota, user.Quota)
}

func TestSePayIPNAcceptsUnauthenticatedRequestFromConfiguredCIDR(t *testing.T) {
	db, user, topUp := setupSePayWebhookTest(t)
	setting.SePayIPNAllowedIPs = "172.236.138.0/24"
	body := `{"timestamp":1793155971,"notification_type":"ORDER_PAID","order":{"id":"aa658c15-d27a-11f1-8895-0afffe87053b","order_id":"TEST_ORDER_1793155971","order_status":"CAPTURED","order_currency":"VND","order_amount":"50000.00","order_invoice_number":"USR1SEPAY123","custom_data":[],"order_description":"SePay IPN test"},"transaction":{"id":"aa659519-d27a-11f1-8895-0afffe87053b","payment_method":"BANK_TRANSFER","transaction_id":"TEST_TXN_1793155971","transaction_type":"PAYMENT","transaction_date":"2026-10-28 09:52:51","transaction_status":"APPROVED","transaction_amount":"50000","transaction_currency":"VND"},"customer":{"id":"customer-id","customer_id":"1"},"agreement":null}`

	response := sendUnauthenticatedSePayIPN(body, "172.236.138.20:443")
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"success":true}`, response.Body.String())

	var settledTopUp model.TopUp
	require.NoError(t, db.First(&settledTopUp, topUp.Id).Error)
	require.Equal(t, common.TopUpStatusSuccess, settledTopUp.Status)
	var creditedUser model.User
	require.NoError(t, db.First(&creditedUser, user.Id).Error)
	require.Greater(t, creditedUser.Quota, user.Quota)
}

func TestSePayIPNAllowedPrefixesAcceptsIPsAndCIDRsAndRejectsInvalidEntries(t *testing.T) {
	prefixes, err := sePayIPNAllowedPrefixes("192.0.2.5\n2001:db8::/32")
	require.NoError(t, err)
	require.Len(t, prefixes, 2)
	require.True(t, prefixes[0].Contains(netip.MustParseAddr("192.0.2.5")))
	require.False(t, prefixes[0].Contains(netip.MustParseAddr("192.0.2.6")))
	require.True(t, prefixes[1].Contains(netip.MustParseAddr("2001:db8::1")))

	_, err = sePayIPNAllowedPrefixes("192.0.2.5\nnot-an-ip")
	require.Error(t, err)
}

func TestSePayIPNRejectsUnauthenticatedRequestFromUntrustedIP(t *testing.T) {
	setupSePayWebhookTest(t)
	setting.SePayIPNAllowedIPs = "172.236.138.20"
	body := `{"timestamp":1793155971,"notification_type":"ORDER_PAID","order":{"order_status":"CAPTURED","order_currency":"VND","order_amount":"50000.00","order_invoice_number":"USR1SEPAY123"},"transaction":{"transaction_status":"APPROVED"}}`

	response := sendUnauthenticatedSePayIPN(body, "198.51.100.10:443")
	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestSePayIPNAllowsAnySourceWhenAllowlistIsEmpty(t *testing.T) {
	setupSePayWebhookTest(t)
	body := `{"timestamp":1793155971,"notification_type":"ORDER_PAID","order":{"order_status":"CAPTURED","order_currency":"VND","order_amount":"50000.00","order_invoice_number":"USR1SEPAY123"},"transaction":{"transaction_status":"APPROVED"}}`

	response := sendUnauthenticatedSePayIPN(body, "198.51.100.10:443")
	require.Equal(t, http.StatusOK, response.Code)
}

func TestSePayIPNRejectsInvalidSecretEvenFromPublishedSourceIP(t *testing.T) {
	setupSePayWebhookTest(t)
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sepay/ipn", strings.NewReader(`{}`))
	c.Request.RemoteAddr = "172.236.138.20:443"
	c.Request.Header.Set("X-Secret-Key", "wrong-secret")

	SePayIPN(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestSePayWebhookWizardSampleAcknowledgedWithoutCredit(t *testing.T) {
	db, user, topUp := setupSePayWebhookTest(t)
	body := `{"timestamp":1793155971,"notification_type":"PAYMENT_SUCCESS","order":{"id":"aa658c15-d27a-11f1-8895-0afffe87053b","order_id":"TEST_ORDER_1793155971","order_status":"CAPTURED","order_currency":"VND","order_amount":50000,"order_invoice_number":"INV-20261028-075491","custom_data":{"test_mode":true,"webhook_test":true},"user_agent":{},"ip_address":"123.24.143.114","order_description":"Thông báo IPN thử nghiệm từ Trình hướng dẫn tích hợp SePay"},"transaction":{"id":"aa659519-d27a-11f1-8895-0afffe87053b","payment_method":"BANK_TRANSFER","transaction_id":"TEST_TXN_1793155971","transaction_type":"CAPTURE","transaction_date":"2026-10-28 09:52:51","transaction_status":"APPROVED","transaction_amount":50000,"transaction_currency":"VND","authentication_status":null,"card_number":null,"card_holder_name":null,"card_expiry":null,"card_funding_method":null,"card_brand":null},"customer":null,"agreement":null}`

	webhookResponse := sendSePayWebhook(body, "Apikey webhook-secret")
	require.Equal(t, http.StatusBadRequest, webhookResponse.Code)

	response := sendSePayIPN(body, "Apikey webhook-secret")
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"success":true}`, response.Body.String())

	var unchangedTopUp model.TopUp
	require.NoError(t, db.First(&unchangedTopUp, topUp.Id).Error)
	require.Equal(t, common.TopUpStatusPending, unchangedTopUp.Status)
	var unchangedUser model.User
	require.NoError(t, db.First(&unchangedUser, user.Id).Error)
	require.Equal(t, user.Quota, unchangedUser.Quota)
}

func TestSePayTransactionWebhookDoesNotCreditUnmatchedTransfer(t *testing.T) {
	db, user, topUp := setupSePayWebhookTest(t)
	body := `{"id":92705,"code":"USR1SEPAY123","content":"USR1SEPAY123 chuyen tien","transferType":"out","transferAmount":50000}`
	response := sendSePayWebhook(body, "Apikey webhook-secret")
	require.Equal(t, http.StatusOK, response.Code)

	var stored model.TopUp
	require.NoError(t, db.First(&stored, topUp.Id).Error)
	require.Equal(t, common.TopUpStatusPending, stored.Status)
	var storedUser model.User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	require.Equal(t, user.Quota, storedUser.Quota)
}

func TestSePayTransactionWebhookDoesNotCreditAmountMismatch(t *testing.T) {
	db, user, topUp := setupSePayWebhookTest(t)
	body := `{"id":92707,"code":"USR1SEPAY123","content":"USR1SEPAY123 chuyen tien","transferType":"in","transferAmount":49999}`
	response := sendSePayWebhook(body, "Apikey webhook-secret")
	require.Equal(t, http.StatusOK, response.Code)

	var stored model.TopUp
	require.NoError(t, db.First(&stored, topUp.Id).Error)
	require.Equal(t, common.TopUpStatusPending, stored.Status)
	var storedUser model.User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	require.Equal(t, user.Quota, storedUser.Quota)
}

func TestSePayWebhookRejectsInvalidWebhookCredential(t *testing.T) {
	setupSePayWebhookTest(t)
	response := sendSePayWebhook(`{"id":92704}`, "Apikey wrong-secret")
	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestSePayWebhookHMACAuthenticatesFreshRawBody(t *testing.T) {
	setupSePayWebhookTest(t)
	body := `{"id":92706,"code":"USR1SEPAY123","transferType":"in","transferAmount":50000}`
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(setting.SePayWebhookSecret))
	_, err := mac.Write([]byte(timestamp + "." + body))
	require.NoError(t, err)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sepay/webhook", strings.NewReader(body))
	c.Request.Header.Set("X-SePay-Timestamp", timestamp)
	c.Request.Header.Set("X-SePay-Signature", signature)
	require.True(t, sePayNotificationAuthorized(c, []byte(body), setting.SePayWebhookSecret))
	require.False(t, sePayNotificationAuthorized(c, []byte(body+" "), setting.SePayWebhookSecret))

	staleTimestamp := strconv.FormatInt(time.Now().Add(-6*time.Minute).Unix(), 10)
	staleMAC := hmac.New(sha256.New, []byte(setting.SePayWebhookSecret))
	_, err = staleMAC.Write([]byte(staleTimestamp + "." + body))
	require.NoError(t, err)
	c.Request.Header.Set("X-SePay-Timestamp", staleTimestamp)
	c.Request.Header.Set("X-SePay-Signature", "sha256="+hex.EncodeToString(staleMAC.Sum(nil)))
	require.False(t, sePayNotificationAuthorized(c, []byte(body), setting.SePayWebhookSecret))
}

func TestSePayWebhookCredentialsAreIsolatedByMode(t *testing.T) {
	setupSePayWebhookTest(t)
	setting.SePaySandboxMerchantID = "sandbox-merchant"
	setting.SePaySandboxSecretKey = "sandbox-key"
	setting.SePayLiveMerchantID = "live-merchant"
	setting.SePayLiveSecretKey = "live-key"
	body := `{"id":92708,"code":"unmatched","transferType":"in","transferAmount":50000}`

	sandboxAccepted := sendSePayWebhookForMode(body, "Apikey sandbox-key", "sandbox")
	require.Equal(t, http.StatusOK, sandboxAccepted.Code)
	sharedSandboxAccepted := sendSePayWebhook(body, "Apikey sandbox-key")
	require.Equal(t, http.StatusOK, sharedSandboxAccepted.Code)
	sandboxRejectsLiveSecret := sendSePayWebhookForMode(body, "Apikey live-key", "sandbox")
	require.Equal(t, http.StatusUnauthorized, sandboxRejectsLiveSecret.Code)
	liveAccepted := sendSePayWebhookForMode(body, "Apikey live-key", "live")
	require.Equal(t, http.StatusOK, liveAccepted.Code)
	sharedLiveAccepted := sendSePayWebhook(body, "Apikey live-key")
	require.Equal(t, http.StatusOK, sharedLiveAccepted.Code)
	sharedRejectsWrongSecret := sendSePayWebhook(body, "Apikey wrong-webhook")
	require.Equal(t, http.StatusUnauthorized, sharedRejectsWrongSecret.Code)
	liveRejectsSandboxSecret := sendSePayWebhookForMode(body, "Apikey sandbox-key", "live")
	require.Equal(t, http.StatusUnauthorized, liveRejectsSandboxSecret.Code)
	unknownMode := sendSePayWebhookForMode(body, "Apikey sandbox-key", "other")
	require.Equal(t, http.StatusUnauthorized, unknownMode.Code)
}

func TestSePayCheckoutUsesCredentialsForSelectedMode(t *testing.T) {
	setupSePayWebhookTest(t)
	setting.SePaySandboxMerchantID = "sandbox-merchant"
	setting.SePaySandboxSecretKey = "sandbox-key"
	setting.SePayLiveMerchantID = "live-merchant"
	setting.SePayLiveSecretKey = "live-key"

	setting.SePaySandbox = true
	sandboxCheckout := buildSePayCheckout("order-1", decimal.NewFromInt(100), "test", "")
	require.Equal(t, "100", sandboxCheckout.Params["order_amount"])
	require.Equal(t, "sandbox-merchant", sandboxCheckout.Params["merchant"])
	require.Equal(t, "https://pay-sandbox.sepay.vn/v1/checkout/init", sandboxCheckout.URL)
	signedFields := []string{
		"order_amount", "merchant", "currency", "operation", "order_description",
		"order_invoice_number", "customer_id", "payment_method", "success_url",
		"error_url", "cancel_url",
	}
	var signedParts []string
	for _, field := range signedFields {
		if value, ok := sandboxCheckout.Params[field]; ok {
			signedParts = append(signedParts, field+"="+value)
		}
	}
	expectedSignature := hmac.New(sha256.New, []byte("sandbox-key"))
	_, err := expectedSignature.Write([]byte(strings.Join(signedParts, ",")))
	require.NoError(t, err)
	require.Equal(t, base64.StdEncoding.EncodeToString(expectedSignature.Sum(nil)), sandboxCheckout.Params["signature"])

	setting.SePaySandbox = false
	liveCheckout := buildSePayCheckout("order-2", decimal.NewFromInt(100), "test", "")
	require.Equal(t, "live-merchant", liveCheckout.Params["merchant"])
	require.Equal(t, "https://pay.sepay.vn/v1/checkout/init", liveCheckout.URL)
}

func TestSePayTopUpReturnURLsCarryResultAndLocalFrontendOrigin(t *testing.T) {
	previousServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "http://localhost:3000"
	t.Cleanup(func() { system_setting.ServerAddress = previousServerAddress })

	for _, testCase := range []struct {
		name   string
		status string
	}{
		{name: "success waits for IPN confirmation", status: "pending"},
		{name: "provider error", status: "fail"},
		{name: "customer cancellation", status: "cancel"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			returnURL := sePayTopUpReturnURL(
				"http://localhost:3000/api/sepay/return",
				testCase.status,
				"USR7SEPAY123",
				"http://localhost:5173",
			)
			parsedURL, err := url.Parse(returnURL)
			require.NoError(t, err)
			require.Equal(t, "http://localhost:3000/api/sepay/return", parsedURL.Scheme+"://"+parsedURL.Host+parsedURL.Path)
			require.Equal(t, testCase.status, parsedURL.Query().Get("pay"))
			require.Equal(t, "USR7SEPAY123", parsedURL.Query().Get("trade_no"))
			require.Equal(t, "http://localhost:5173", parsedURL.Query().Get("return_origin"))
		})
	}
}

func TestSePayReturnRedirectsToValidatedFrontendWithStatus(t *testing.T) {
	previousServerAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "http://localhost:3000"
	t.Cleanup(func() { system_setting.ServerAddress = previousServerAddress })
	gin.SetMode(gin.TestMode)

	for _, testCase := range []struct {
		name         string
		query        string
		wantLocation string
	}{
		{
			name:         "local frontend return preserves pending order",
			query:        "pay=pending&trade_no=USR7SEPAY123&return_origin=http%3A%2F%2Flocalhost%3A5173",
			wantLocation: "http://localhost:5173/wallet?pay=pending&trade_no=USR7SEPAY123",
		},
		{
			name:         "untrusted return origin falls back to configured server",
			query:        "pay=fail&trade_no=USR7SEPAY123&return_origin=https%3A%2F%2Fevil.example",
			wantLocation: "http://localhost:3000/wallet?pay=fail&trade_no=USR7SEPAY123",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/sepay/return?"+testCase.query, nil)
			SePayReturn(c)

			require.Equal(t, http.StatusFound, recorder.Code)
			require.Equal(t, testCase.wantLocation, recorder.Header().Get("Location"))
		})
	}
}

func TestSePayTopUpAmountIgnoresEpayPriceAndConvertsWithVNDExchangeRate(t *testing.T) {
	db, user, _ := setupSePayWebhookTest(t)
	previousMainDBType, previousLogDBType := common.MainDatabaseType(), common.LogDatabaseType()
	previousMasterNode := common.IsMasterNode
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.IsMasterNode = false
	t.Setenv("LOG_SQL_DSN", "")
	require.NoError(t, model.InitLogDB())
	t.Cleanup(func() {
		common.SetDatabaseTypes(previousMainDBType, previousLogDBType)
		common.IsMasterNode = previousMasterNode
	})

	previousPrice := operation_setting.Price
	previousGeneralSettings := *operation_setting.GetGeneralSetting()
	previousPaymentSettings := *operation_setting.GetPaymentSetting()
	previousCallbackAddress := operation_setting.CustomCallbackAddress
	previousServerAddress := system_setting.ServerAddress
	previousTopupGroupRatio := common.TopupGroupRatio2JSONString()
	t.Cleanup(func() {
		operation_setting.Price = previousPrice
		*operation_setting.GetGeneralSetting() = previousGeneralSettings
		*operation_setting.GetPaymentSetting() = previousPaymentSettings
		operation_setting.CustomCallbackAddress = previousCallbackAddress
		system_setting.ServerAddress = previousServerAddress
		common.UpdateTopupGroupRatioByJSONString(previousTopupGroupRatio)
	})
	operation_setting.CustomCallbackAddress = ""
	system_setting.ServerAddress = "http://localhost:3000"

	operation_setting.Price = 7.3
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetGeneralSetting().ExchangeRateSource = model.ExchangeRateSourceManual
	operation_setting.GetGeneralSetting().ManualExchangeRates = `{"USD":"26000.05"}`
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{10: 0.5}
	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"default":1,"vip":2}`))
	user.Group = "vip"
	require.NoError(t, db.Save(&user).Error)

	gin.SetMode(gin.TestMode)
	amountRecorder := httptest.NewRecorder()
	amountContext, _ := gin.CreateTestContext(amountRecorder)
	amountContext.Set("id", user.Id)
	amountContext.Request = httptest.NewRequest(http.MethodPost, "/api/user/sepay/amount", strings.NewReader(`{"amount":10}`))
	RequestSePayAmount(amountContext)
	var amountResponse struct {
		Data string `json:"data"`
	}
	require.NoError(t, common.Unmarshal(amountRecorder.Body.Bytes(), &amountResponse))
	require.Equal(t, "260001", amountResponse.Data)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", user.Id)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/sepay/pay", strings.NewReader(`{"amount":10}`))
	c.Request.Header.Set("Origin", "http://localhost:5173")
	RequestSePay(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Data map[string]string `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "260001", response.Data["order_amount"])
	for key, status := range map[string]string{
		"success_url": "pending",
		"error_url":   "fail",
		"cancel_url":  "cancel",
	} {
		returnURL, err := url.Parse(response.Data[key])
		require.NoError(t, err)
		require.Equal(t, status, returnURL.Query().Get("pay"))
		require.Equal(t, "http://localhost:5173", returnURL.Query().Get("return_origin"))
		require.NotEmpty(t, returnURL.Query().Get("trade_no"))
	}
	require.Equal(t, float64(73), getPayMoney(10, "vip"), "Epay keeps applying its configured Price multiplier")

	var topUp model.TopUp
	require.NoError(t, db.Where("user_id = ?", user.Id).Order("id DESC").First(&topUp).Error)
	require.Equal(t, float64(260001), topUp.Money)
	require.Equal(t, "26000.05", topUp.ExchangeRate)

	settings := operation_setting.GetGeneralSetting()
	settings.ExchangeRateSource = model.ExchangeRateSourceVietcombank
	settings.ExchangeRateType = model.ExchangeRateTypeSell
	settings.ExchangeRateFeeRatio = 0.001
	require.NoError(t, model.SaveExchangeRate(&model.ExchangeRate{
		BaseCurrency:  "VND",
		QuoteCurrency: "USD",
		Rate:          "26000.05",
		Source:        model.ExchangeRateSourceVietcombank,
		RateType:      model.ExchangeRateTypeSell,
		EffectiveDate: time.Now().Format("2006-01-02"),
	}))

	feeRecorder := httptest.NewRecorder()
	feeContext, _ := gin.CreateTestContext(feeRecorder)
	feeContext.Set("id", user.Id)
	feeContext.Request = httptest.NewRequest(http.MethodPost, "/api/user/sepay/amount", strings.NewReader(`{"amount":10}`))
	RequestSePayAmount(feeContext)
	require.NoError(t, common.Unmarshal(feeRecorder.Body.Bytes(), &amountResponse))
	require.Equal(t, "260261", amountResponse.Data, "a 0.1% Vietcombank fee is added before rounding the VND amount")

	feeCheckoutRecorder := httptest.NewRecorder()
	feeCheckoutContext, _ := gin.CreateTestContext(feeCheckoutRecorder)
	feeCheckoutContext.Set("id", user.Id)
	feeCheckoutContext.Request = httptest.NewRequest(http.MethodPost, "/api/user/sepay/pay", strings.NewReader(`{"amount":10}`))
	RequestSePay(feeCheckoutContext)
	require.Equal(t, http.StatusOK, feeCheckoutRecorder.Code)
	var feeCheckoutResponse struct {
		Data map[string]string `json:"data"`
	}
	require.NoError(t, common.Unmarshal(feeCheckoutRecorder.Body.Bytes(), &feeCheckoutResponse))
	require.Equal(t, "260261", feeCheckoutResponse.Data["order_amount"])

	settings.ExchangeRateFeeRatio = 1
	fullFeeRecorder := httptest.NewRecorder()
	fullFeeContext, _ := gin.CreateTestContext(fullFeeRecorder)
	fullFeeContext.Set("id", user.Id)
	fullFeeContext.Request = httptest.NewRequest(http.MethodPost, "/api/user/sepay/amount", strings.NewReader(`{"amount":10}`))
	RequestSePayAmount(fullFeeContext)
	require.NoError(t, common.Unmarshal(fullFeeRecorder.Body.Bytes(), &amountResponse))
	require.Equal(t, "520001", amountResponse.Data, "a 100% fee doubles the VND conversion amount")

	settings.ExchangeRateFeeRatio = 0
	noFeeRecorder := httptest.NewRecorder()
	noFeeContext, _ := gin.CreateTestContext(noFeeRecorder)
	noFeeContext.Set("id", user.Id)
	noFeeContext.Request = httptest.NewRequest(http.MethodPost, "/api/user/sepay/amount", strings.NewReader(`{"amount":10}`))
	RequestSePayAmount(noFeeContext)
	require.NoError(t, common.Unmarshal(noFeeRecorder.Body.Bytes(), &amountResponse))
	require.Equal(t, "260001", amountResponse.Data, "zero fee preserves the provider conversion amount")
}
