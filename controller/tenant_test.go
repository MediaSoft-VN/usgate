package controller

import (
	"context"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type tenantControllerDNSResolver struct{}

func (tenantControllerDNSResolver) LookupIP(context.Context, string, string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("203.0.113.10")}, nil
}

func (tenantControllerDNSResolver) LookupCNAME(context.Context, string) (string, error) {
	return "", nil
}

func TestTenantDomainManagementUsesMembershipAcrossApplicationHosts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Tenant{}, &model.TenantMembership{}, &model.TenantDomain{}))
	previousDB := model.DB
	previousResolver := service.TenantDomainDNSResolver
	model.DB = db
	service.TenantDomainDNSResolver = tenantControllerDNSResolver{}
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	common.OptionMap["TenantDomainDNSRecordA"] = "203.0.113.10"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() { model.DB = previousDB })
	t.Cleanup(func() {
		service.TenantDomainDNSResolver = previousResolver
		common.OptionMapRWMutex.Lock()
		common.OptionMap["TenantDomainDNSRecordA"] = ""
		common.OptionMapRWMutex.Unlock()
	})

	owner := model.User{Username: "domain-owner", Password: "owner-secret", AffCode: "domain-owner-code", Status: common.UserStatusEnabled}
	other := model.User{Username: "other-owner", Password: "other-secret", AffCode: "other-owner-code", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&other).Error)
	tenant, err := model.CreateTenantWithOwner(db, "domain-tenant", "Domain Tenant", owner.Id)
	require.NoError(t, err)
	otherTenant, err := model.CreateTenantWithOwner(db, "other-tenant", "Other Tenant", other.Id)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	createContext, _ := gin.CreateTestContext(recorder)
	createContext.Request = httptest.NewRequest("POST", "/api/tenants/1/domains", strings.NewReader(`{"hostname":"app.example.com"}`))
	createContext.Request.Host = "gateway.example.com"
	createContext.Params = gin.Params{{Key: "tenant_id", Value: "1"}}
	createContext.Set("id", owner.Id)

	CreateTenantDomain(createContext)
	require.Equal(t, 200, recorder.Code)
	var createResponse struct {
		Data struct {
			Domain model.TenantDomain `json:"domain"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &createResponse))

	verifyRecorder := httptest.NewRecorder()
	verifyContext, _ := gin.CreateTestContext(verifyRecorder)
	verifyContext.Request = httptest.NewRequest("POST", "/api/tenants/1/domains/1/verify", strings.NewReader(`{}`))
	verifyContext.Request.Host = "gateway.example.com"
	verifyContext.Params = gin.Params{{Key: "tenant_id", Value: "1"}, {Key: "domain_id", Value: "1"}}
	verifyContext.Set("id", owner.Id)
	VerifyTenantDomain(verifyContext)
	require.Equal(t, 200, verifyRecorder.Code)

	deniedDeleteRecorder := httptest.NewRecorder()
	deniedDeleteContext, _ := gin.CreateTestContext(deniedDeleteRecorder)
	deniedDeleteContext.Request = httptest.NewRequest("DELETE", "/api/tenants/1/domains/1", nil)
	deniedDeleteContext.Request.Host = "gateway.example.com"
	deniedDeleteContext.Params = gin.Params{{Key: "tenant_id", Value: "1"}, {Key: "domain_id", Value: "1"}}
	deniedDeleteContext.Set("id", other.Id)
	DeleteTenantDomain(deniedDeleteContext)
	require.Contains(t, deniedDeleteRecorder.Body.String(), "Unauthorized, insufficient tenant privileges")

	deleteRecorder := httptest.NewRecorder()
	deleteContext, _ := gin.CreateTestContext(deleteRecorder)
	deleteContext.Request = httptest.NewRequest("DELETE", "/api/tenants/1/domains/1", nil)
	deleteContext.Request.Host = "gateway.example.com"
	deleteContext.Params = gin.Params{{Key: "tenant_id", Value: "1"}, {Key: "domain_id", Value: "1"}}
	deleteContext.Set("id", owner.Id)
	DeleteTenantDomain(deleteContext)
	require.Equal(t, 200, deleteRecorder.Code)

	forbiddenRecorder := httptest.NewRecorder()
	forbiddenContext, _ := gin.CreateTestContext(forbiddenRecorder)
	forbiddenContext.Request = httptest.NewRequest("POST", "/api/tenants/2/domains", strings.NewReader(`{"hostname":"other.example.com"}`))
	forbiddenContext.Request.Host = "gateway.example.com"
	forbiddenContext.Params = gin.Params{{Key: "tenant_id", Value: "2"}}
	forbiddenContext.Set("id", owner.Id)
	CreateTenantDomain(forbiddenContext)
	require.Contains(t, forbiddenRecorder.Body.String(), "Unauthorized, insufficient tenant privileges")

	var domain model.TenantDomain
	require.ErrorIs(t, db.Where("tenant_id = ?", tenant.Id).First(&domain).Error, gorm.ErrRecordNotFound)

	// Hard deletion releases the unique hostname so it can be added again.
	readdRecorder := httptest.NewRecorder()
	readdContext, _ := gin.CreateTestContext(readdRecorder)
	readdContext.Request = httptest.NewRequest("POST", "/api/tenants/1/domains", strings.NewReader(`{"hostname":"app.example.com"}`))
	readdContext.Request.Host = "gateway.example.com"
	readdContext.Params = gin.Params{{Key: "tenant_id", Value: "1"}}
	readdContext.Set("id", owner.Id)
	CreateTenantDomain(readdContext)
	require.Equal(t, 200, readdRecorder.Code)
	var otherDomains []model.TenantDomain
	require.NoError(t, db.Where("tenant_id = ?", otherTenant.Id).Find(&otherDomains).Error)
	require.Empty(t, otherDomains)
}
