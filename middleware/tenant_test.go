package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestResolveTenantContextUsesVerifiedHostAndActiveMembership(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Tenant{}, &model.TenantMembership{}, &model.TenantDomain{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	owner := model.User{Username: "tenant-owner", Password: "unused", Status: common.UserStatusEnabled}
	member := model.User{Username: "tenant-member", Password: "unused", Status: common.UserStatusEnabled}
	outsider := model.User{Username: "tenant-outsider", Password: "unused", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&member).Error)
	require.NoError(t, db.Create(&outsider).Error)
	tenant, err := model.CreateTenantWithOwner(db, "host-bound", "Host Bound", owner.Id)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.TenantMembership{TenantId: tenant.Id, UserId: member.Id, RoleKey: model.TenantRoleMember, Status: model.MembershipActive}).Error)
	require.NoError(t, db.Create(&model.TenantDomain{TenantId: tenant.Id, Hostname: "app.partner.example.com", Status: model.DomainVerified}).Error)

	request := func(host string, userID int) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/api/user/self", nil)
		c.Request.Host = host
		c.Set("id", userID)
		return c
	}

	memberContext := request("app.partner.example.com:443", member.Id)
	require.NoError(t, ResolveTenantContext(memberContext))
	selectedID, ok := TenantID(memberContext)
	require.True(t, ok)
	require.Equal(t, tenant.Id, selectedID)

	outsiderContext := request("app.partner.example.com", outsider.Id)
	require.ErrorIs(t, ResolveTenantContext(outsiderContext), model.ErrTenantForbidden)

	mainContext := request("platform.example.com", outsider.Id)
	require.NoError(t, ResolveTenantContext(mainContext))
	_, ok = mainContext.Get(string(constant.ContextKeyTenantID))
	require.False(t, ok)
}

func TestResolveTenantContextAllowsLocalhostOnlyWhenExplicitlyEnabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Tenant{}, &model.TenantMembership{}, &model.TenantDomain{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })
	t.Setenv("TENANT_ALLOW_LOCALHOST", "true")
	t.Setenv("TENANT_LOCALHOST_PORT", "5555")

	owner := model.User{Username: "localhost-owner", Password: "unused", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	tenant, err := model.CreateTenantWithOwner(db, "localhost-tenant", "Localhost Tenant", owner.Id)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.TenantDomain{TenantId: tenant.Id, Hostname: "localhost", Status: model.DomainVerified}).Error)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/user/self", nil)
	c.Request.Host = "localhost:5555"
	c.Set("id", owner.Id)
	require.NoError(t, ResolveTenantContext(c))
	selectedID, ok := TenantID(c)
	require.True(t, ok)
	require.Equal(t, tenant.Id, selectedID)

	mainContext := request("localhost:5173", owner.Id)
	require.NoError(t, ResolveTenantContext(mainContext))
	_, ok = mainContext.Get(string(constant.ContextKeyTenantID))
	require.False(t, ok)
}
