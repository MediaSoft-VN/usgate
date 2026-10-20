package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestListTenantMemberCandidatesRequiresTenantAdminAndReturnsSafeEnabledUsers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Tenant{}, &model.TenantMembership{}, &model.TenantDomain{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	owner := model.User{Username: "partner-owner", Password: "owner-secret", AffCode: "partner-owner-code", Status: common.UserStatusEnabled}
	candidate := model.User{Username: "enabled-user", DisplayName: "Enabled User", Password: "candidate-secret", AffCode: "enabled-user-code", Status: common.UserStatusEnabled}
	disabled := model.User{Username: "disabled-user", Password: "disabled-secret", AffCode: "disabled-user-code", Status: common.UserStatusDisabled}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&candidate).Error)
	require.NoError(t, db.Create(&disabled).Error)
	tenant, err := model.CreateTenantWithOwner(db, "partner-workspace", "Partner Workspace", owner.Id)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.TenantDomain{TenantId: tenant.Id, Hostname: "partner.example.com", Status: model.DomainVerified}).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("GET", "/api/tenants/1/member-candidates", nil)
	context.Request.Host = "partner.example.com"
	context.Params = gin.Params{{Key: "tenant_id", Value: "1"}}
	context.Set("id", owner.Id)

	ListTenantMemberCandidates(context)

	require.Equal(t, 200, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"username":"enabled-user"`)
	require.NotContains(t, recorder.Body.String(), "candidate-secret")
	require.NotContains(t, recorder.Body.String(), "disabled-user")

	member := model.User{Username: "regular-member", Password: "member-secret", AffCode: "regular-member-code", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&member).Error)
	require.NoError(t, db.Create(&model.TenantMembership{TenantId: tenant.Id, UserId: member.Id, RoleKey: model.TenantRoleMember, Status: model.MembershipActive}).Error)
	memberRecorder := httptest.NewRecorder()
	memberContext, _ := gin.CreateTestContext(memberRecorder)
	memberContext.Request = httptest.NewRequest("GET", "/api/tenants/1/member-candidates", nil)
	memberContext.Request.Host = "partner.example.com"
	memberContext.Params = gin.Params{{Key: "tenant_id", Value: "1"}}
	memberContext.Set("id", member.Id)

	ListTenantMemberCandidates(memberContext)

	require.Contains(t, memberRecorder.Body.String(), "Unauthorized, insufficient tenant privileges")
}
