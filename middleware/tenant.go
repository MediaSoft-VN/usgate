package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// TenantContext resolves only the trusted request Host. Forwarded host headers
// are deliberately ignored; deployments must terminate TLS and set Request.Host.
func TenantContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt("id")
		tenant, err := model.ResolveVerifiedTenant(c.Request.Host, userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "tenant access denied"})
			return
		}
		membership, err := model.ActiveMembership(tenant.Id, userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "tenant access denied"})
			return
		}
		c.Set(string(constant.ContextKeyTenant), tenant)
		c.Set(string(constant.ContextKeyTenantID), tenant.Id)
		c.Set(string(constant.ContextKeyTenantRole), membership.RoleKey)
		c.Next()
	}
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
	if strings.Contains(c.Request.Host, ".") {
		resolved, err := model.ResolveVerifiedTenant(c.Request.Host, userID)
		if err != nil || resolved.Id != tenantID {
			common.ApiErrorT(c, "Tenant access denied")
			return false
		}
	}
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
