package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSePayWebhookTest(t *testing.T) (*gorm.DB, model.User, *model.TopUp) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}, &model.IdempotencyKey{}, &model.Log{}))
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
	t.Cleanup(func() {
		setting.SePayMerchantID = previousMerchantID
		setting.SePaySecretKey = previousSecretKey
		setting.SePayWebhookSecret = previousWebhookSecret
	})
	setting.SePayMerchantID = "merchant-test"
	setting.SePaySecretKey = "checkout-secret"
	setting.SePayWebhookSecret = "webhook-secret"
	user := model.User{Username: "sepay-user", Password: "test-password", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)
	topUp := &model.TopUp{
		UserId: user.Id, Amount: 1, Money: 50000, TradeNo: "USR1SEPAY123",
		PaymentMethod: setting.SePayPaymentMethod, PaymentProvider: model.PaymentProviderSePay,
		Status: common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(topUp).Error)
	return db, user, topUp
}

func sendSePayWebhook(body string, authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sepay/webhook", strings.NewReader(body))
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
	require.True(t, sePayWebhookAuthorized(c, []byte(body)))
	require.False(t, sePayWebhookAuthorized(c, []byte(body+" ")))

	staleTimestamp := strconv.FormatInt(time.Now().Add(-6*time.Minute).Unix(), 10)
	staleMAC := hmac.New(sha256.New, []byte(setting.SePayWebhookSecret))
	_, err = staleMAC.Write([]byte(staleTimestamp + "." + body))
	require.NoError(t, err)
	c.Request.Header.Set("X-SePay-Timestamp", staleTimestamp)
	c.Request.Header.Set("X-SePay-Signature", "sha256="+hex.EncodeToString(staleMAC.Sum(nil)))
	require.False(t, sePayWebhookAuthorized(c, []byte(body)))
}
