package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPaymentReturnPathUsesDefaultDashboardRoutes(t *testing.T) {
	previousAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://dashboard.example.com/"
	t.Cleanup(func() { system_setting.ServerAddress = previousAddress })

	assert.Equal(
		t,
		"https://dashboard.example.com/wallet?pay=success",
		paymentReturnPath("/wallet?pay=success"),
	)
	assert.Equal(
		t,
		"https://dashboard.example.com/usage-logs",
		paymentReturnPath("/usage-logs"),
	)
}

func TestPaymentReturnPathUsesVerifiedTenantHostOnly(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Tenant{}, &model.TenantMembership{}, &model.TenantDomain{}, &model.TopUp{}))
	previousDB := model.DB
	previousAddress := system_setting.ServerAddress
	model.DB = db
	system_setting.ServerAddress = "https://newapi.example.com/"
	t.Cleanup(func() {
		model.DB = previousDB
		system_setting.ServerAddress = previousAddress
	})

	owner := model.User{Username: "return-domain-owner", Password: "unused", AffCode: "return-domain-owner-code", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	tenant, err := model.CreateTenantWithOwner(db, "return-domain", "Return Domain", owner.Id)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.TenantDomain{TenantId: tenant.Id, Hostname: "shop.example.org", Status: model.DomainVerified}).Error)
	topUp := model.TopUp{UserId: owner.Id, TradeNo: "USR1SEPAY123", PaymentProvider: model.PaymentProviderSePay}
	require.NoError(t, db.Create(&topUp).Error)
	otherOwner := model.User{Username: "other-return-domain-owner", Password: "unused", AffCode: "other-return-domain-owner-code", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&otherOwner).Error)
	otherTenant, err := model.CreateTenantWithOwner(db, "other-return-domain", "Other Return Domain", otherOwner.Id)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.TenantDomain{TenantId: otherTenant.Id, Hostname: "other.example.org", Status: model.DomainVerified}).Error)

	gin.SetMode(gin.TestMode)
	requestContext := func(host string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Host = host
		return c
	}

	assert.Equal(t, "https://shop.example.org/wallet", paymentReturnPathForRequest(requestContext("shop.example.org:443"), "/wallet"))
	assert.Equal(t, "https://newapi.example.com/wallet", paymentReturnPathForRequest(requestContext("unverified.example.org"), "/wallet"))
	apiRequest := requestContext("newapi.example.com")
	apiRequest.Request.Header.Set("Origin", "https://shop.example.org")
	assert.Equal(t, "https://shop.example.org", paymentReturnOriginForRequest(apiRequest, topUp.TradeNo))
	returnURL := sePayTopUpReturnURL(
		"https://newapi.example.com/api/sepay/return",
		"pending",
		topUp.TradeNo,
		paymentReturnOriginForRequest(apiRequest, topUp.TradeNo),
	)
	assert.Contains(t, returnURL, "return_origin=https%3A%2F%2Fshop.example.org")
	returnRecorder := httptest.NewRecorder()
	returnContext, _ := gin.CreateTestContext(returnRecorder)
	returnContext.Request = httptest.NewRequest(http.MethodGet, returnURL, nil)
	SePayReturn(returnContext)
	assert.Equal(t, http.StatusFound, returnRecorder.Code)
	assert.Equal(t, "https://shop.example.org/wallet?pay=pending&trade_no="+topUp.TradeNo, returnRecorder.Header().Get("Location"))
	unverifiedOriginRequest := requestContext("newapi.example.com")
	unverifiedOriginRequest.Request.Header.Set("Origin", "https://unverified.example.org")
	assert.Empty(t, paymentReturnOriginForRequest(unverifiedOriginRequest, topUp.TradeNo))
	spoofedHost := requestContext("unverified.example.org")
	spoofedHost.Request.Header.Set("X-Forwarded-Host", "shop.example.org")
	assert.Equal(t, "https://newapi.example.com/wallet", paymentReturnPathForRequest(spoofedHost, "/wallet"))
	assert.Equal(t, "https://shop.example.org", validatedPaymentReturnOrigin("https://shop.example.org", topUp.TradeNo))
	assert.Empty(t, validatedPaymentReturnOrigin("https://unverified.example.org", topUp.TradeNo))
	assert.Empty(t, validatedPaymentReturnOrigin("https://other.example.org", topUp.TradeNo))
}
