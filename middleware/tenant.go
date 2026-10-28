package middleware

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// TenantContext resolves only the trusted request Host. Forwarded host headers
// are deliberately ignored; deployments must terminate TLS and set Request.Host.
func TenantContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := ResolveTenantContext(c); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "tenant access denied"})
			return
		}
		c.Next()
	}
}

// ResolveTenantContext is optional on the platform host and mandatory when a
// verified tenant domain is used. It never reads tenant identity from a
// request header, query parameter, or client state.
func ResolveTenantContext(c *gin.Context) error {
	if model.IsSystemServerHost(c.Request.Host) {
		return nil
	}
	tenant, err := model.ResolveVerifiedTenantForHost(c.Request.Host)
	if errors.Is(err, model.ErrTenantNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	userID := c.GetInt("id")
	if userID <= 0 {
		return model.ErrTenantForbidden
	}
	membership, err := model.ActiveMembership(tenant.Id, userID)
	if err != nil {
		return model.ErrTenantForbidden
	}
	c.Set(string(constant.ContextKeyTenant), tenant)
	c.Set(string(constant.ContextKeyTenantID), tenant.Id)
	c.Set(string(constant.ContextKeyTenantRole), membership.RoleKey)
	return nil
}

func TenantID(c *gin.Context) (int, bool) {
	if value, ok := c.Get(string(constant.ContextKeyTenantID)); ok {
		id, ok := value.(int)
		return id, ok && id > 0
	}
	return 0, false
}

func RequireTenantAccess(c *gin.Context, tenantID int, action string) bool {
	userID := c.GetInt("id")
	membership, err := model.ActiveMembership(tenantID, userID)
	if err != nil || !model.TenantPermission(membership.RoleKey, action) {
		common.ApiErrorT(c, "Unauthorized, insufficient tenant privileges")
		return false
	}
	if selected, ok := TenantID(c); ok && selected != tenantID {
		common.ApiErrorT(c, "Tenant access denied")
		return false
	}
	c.Set(string(constant.ContextKeyTenantID), tenantID)
	c.Set(string(constant.ContextKeyTenantRole), membership.RoleKey)
	return true
}
